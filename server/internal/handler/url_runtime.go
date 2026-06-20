package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/config"
	"hl6-server/internal/repository"
)

const (
	configKeyFrontendURL           = "frontend_url"
	configKeyFrontendURLs          = "frontend_urls"
	configKeyBackendURL            = "backend_url"
	configKeyBackendURLs           = "backend_urls"
	configKeyURLConfirmedSignature = "url_confirmed_signature"

	urlSourceEnv      = "env"
	urlSourceDB       = "db"
	urlSourceAuto     = "auto"
	urlSourceFallback = "fallback"
)

type URLConfigState struct {
	FrontendURLs      []string `json:"frontend_urls"`
	BackendURLs       []string `json:"backend_urls"`
	FrontendURL       string   `json:"frontend_url"`
	BackendURL        string   `json:"backend_url"`
	FrontendSource    string   `json:"frontend_source"`
	BackendSource     string   `json:"backend_source"`
	FrontendEnvLocked bool     `json:"frontend_env_locked"`
	BackendEnvLocked  bool     `json:"backend_env_locked"`
	Confirmed         bool     `json:"confirmed"`
	Signature         string   `json:"-"`
}

type URLResolver struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewURLResolver(repo *repository.Repository, cfg *config.Config) *URLResolver {
	return &URLResolver{repo: repo, cfg: cfg}
}

func (r *URLResolver) Resolve(c *gin.Context) (*URLConfigState, error) {
	configs, err := r.repo.GetSystemConfigsByKeys([]string{
		configKeyFrontendURLs,
		configKeyFrontendURL,
		configKeyBackendURLs,
		configKeyBackendURL,
		configKeyURLConfirmedSignature,
	})
	if err != nil {
		return nil, err
	}

	state := &URLConfigState{
		FrontendEnvLocked: r.cfg.FrontendURLEnvSet,
		BackendEnvLocked:  r.cfg.BackendURLEnvSet,
	}

	if state.FrontendEnvLocked {
		state.FrontendURLs = cloneStringSlice(r.cfg.FrontendURLs)
		state.FrontendSource = urlSourceEnv
	} else if dbURLs, ok := normalizeStoredURLList(configs[configKeyFrontendURLs]); ok {
		state.FrontendURLs = dbURLs
		state.FrontendSource = urlSourceDB
	} else if dbURL, ok := normalizeStoredURL(configs[configKeyFrontendURL]); ok {
		state.FrontendURLs = []string{dbURL}
		state.FrontendSource = urlSourceDB
	}

	if state.BackendEnvLocked {
		state.BackendURLs = cloneStringSlice(r.cfg.BackendURLs)
		state.BackendSource = urlSourceEnv
	} else if dbURLs, ok := normalizeStoredURLList(configs[configKeyBackendURLs]); ok {
		state.BackendURLs = dbURLs
		state.BackendSource = urlSourceDB
	} else if dbURL, ok := normalizeStoredURL(configs[configKeyBackendURL]); ok {
		state.BackendURLs = []string{dbURL}
		state.BackendSource = urlSourceDB
	}

	detected := ""
	if len(state.FrontendURLs) == 0 || len(state.BackendURLs) == 0 {
		detected = detectPublicBaseURL(c.Request)
	}

	if len(state.FrontendURLs) == 0 && detected != "" {
		state.FrontendURLs = []string{detected}
		state.FrontendSource = urlSourceAuto
		if err := persistURLList(r.repo, configKeyFrontendURLs, configKeyFrontendURL, state.FrontendURLs); err != nil {
			return nil, err
		}
	}

	if len(state.BackendURLs) == 0 && detected != "" {
		state.BackendURLs = []string{detected}
		state.BackendSource = urlSourceAuto
		if err := persistURLList(r.repo, configKeyBackendURLs, configKeyBackendURL, state.BackendURLs); err != nil {
			return nil, err
		}
	}

	if len(state.FrontendURLs) == 0 {
		state.FrontendURLs = []string{"http://localhost:5174"}
		state.FrontendSource = urlSourceFallback
	}
	if len(state.BackendURLs) == 0 {
		state.BackendURLs = cloneStringSlice(state.FrontendURLs)
		state.BackendSource = state.FrontendSource
		if state.BackendSource == "" {
			state.BackendSource = urlSourceFallback
		}
	}

	state.FrontendURL = chooseActiveURL(state.FrontendURLs, c.Request)
	state.BackendURL = chooseActiveURL(state.BackendURLs, c.Request)

	state.Signature = buildURLSignature(state)
	savedSignature := strings.TrimSpace(configs[configKeyURLConfirmedSignature])
	state.Confirmed = savedSignature != "" && savedSignature == state.Signature
	return state, nil
}

func normalizeStoredURL(raw string) (string, bool) {
	normalized, err := config.NormalizePublicURL(raw)
	if err != nil || normalized == "" {
		return "", false
	}
	return normalized, true
}

func normalizeStoredURLList(raw string) ([]string, bool) {
	values, err := config.ParsePublicURLList(raw)
	if err != nil || len(values) == 0 {
		return nil, false
	}
	return values, true
}

func detectPublicBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}

	forwardedProto, forwardedHost := parseForwardedHeader(r.Header.Get("Forwarded"))
	xfProto := normalizeForwardedScheme(firstCSVToken(r.Header.Get("X-Forwarded-Proto")))
	xfHost := firstCSVToken(r.Header.Get("X-Forwarded-Host"))
	reqHost := strings.TrimSpace(r.Host)

	candidates := []struct {
		scheme string
		host   string
	}{
		{scheme: firstNonEmpty(forwardedProto, xfProto, requestScheme(r)), host: forwardedHost},
		{scheme: firstNonEmpty(xfProto, forwardedProto, requestScheme(r)), host: xfHost},
		{scheme: firstNonEmpty(xfProto, forwardedProto, requestScheme(r)), host: reqHost},
		{scheme: requestScheme(r), host: reqHost},
	}

	for _, candidate := range candidates {
		if candidate.host == "" || candidate.scheme == "" {
			continue
		}
		u := fmt.Sprintf("%s://%s", candidate.scheme, candidate.host)
		normalized, err := config.NormalizePublicURL(u)
		if err == nil && normalized != "" {
			return normalized
		}
	}

	return ""
}

func parseForwardedHeader(raw string) (proto string, host string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ""
	}

	parts := strings.Split(value, ",")
	first := strings.TrimSpace(parts[0])
	for _, seg := range strings.Split(first, ";") {
		kv := strings.SplitN(strings.TrimSpace(seg), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		switch key {
		case "proto":
			proto = normalizeForwardedScheme(val)
		case "host":
			host = val
		}
	}

	return proto, host
}

func normalizeForwardedScheme(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "http" || v == "https" {
		return v
	}
	return ""
}

func requestScheme(r *http.Request) string {
	if r == nil {
		return "http"
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func firstCSVToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, ",")
	return strings.Trim(strings.TrimSpace(parts[0]), "\"")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func buildURLSignature(state *URLConfigState) string {
	payload := fmt.Sprintf(
		"frontend_urls=%s|backend_urls=%s|frontend=%s|backend=%s|frontend_source=%s|backend_source=%s",
		strings.Join(state.FrontendURLs, ","),
		strings.Join(state.BackendURLs, ","),
		state.FrontendURL,
		state.BackendURL,
		state.FrontendSource,
		state.BackendSource,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func chooseActiveURL(urls []string, req *http.Request) string {
	if len(urls) == 0 {
		return ""
	}
	if req == nil {
		return urls[0]
	}

	detected := detectPublicBaseURL(req)
	if detected == "" {
		return urls[0]
	}
	detectedNorm, err := config.NormalizePublicURL(detected)
	if err != nil || detectedNorm == "" {
		return urls[0]
	}
	for _, candidate := range urls {
		if candidate == detectedNorm {
			return candidate
		}
	}

	parsedDetected, err := urlFromString(detectedNorm)
	if err != nil {
		return urls[0]
	}
	for _, candidate := range urls {
		parsedCandidate, parseErr := urlFromString(candidate)
		if parseErr == nil && strings.EqualFold(parsedCandidate.Host, parsedDetected.Host) {
			return candidate
		}
	}

	return urls[0]
}

func urlFromString(raw string) (*neturl.URL, error) {
	return neturl.Parse(raw)
}

func persistURLList(repo *repository.Repository, listKey, legacyKey string, values []string) error {
	listValue := strings.Join(values, ",")
	if err := repo.SetSystemConfig(listKey, listValue); err != nil {
		return err
	}
	first := ""
	if len(values) > 0 {
		first = values[0]
	}
	return repo.SetSystemConfig(legacyKey, first)
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
