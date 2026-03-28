package handler

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"hl6-server/internal/config"
	"hl6-server/internal/oidc"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"
)

const (
	configKeyOIDCIssuer       = "oidc_issuer"
	configKeyOIDCClientID     = "oidc_client_id"
	configKeyOIDCClientSecret = "oidc_client_secret"

	oidcSourceUnset = "none"
)

var errOIDCNotConfigured = errors.New("oidc is not configured")

type OIDCRuntimeState struct {
	Issuer                 string   `json:"issuer"`
	ClientID               string   `json:"client_id"`
	ClientSecret           string   `json:"-"`
	IssuerSource           string   `json:"issuer_source"`
	ClientIDSource         string   `json:"client_id_source"`
	ClientSecretSource     string   `json:"client_secret_source"`
	IssuerEnvLocked        bool     `json:"issuer_env_locked"`
	ClientIDEnvLocked      bool     `json:"client_id_env_locked"`
	ClientSecretEnvLocked  bool     `json:"client_secret_env_locked"`
	ClientSecretConfigured bool     `json:"client_secret_configured"`
	Configured             bool     `json:"configured"`
	MissingFields          []string `json:"missing_fields"`
}

type OIDCRuntimeResolver struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewOIDCRuntimeResolver(repo *repository.Repository, cfg *config.Config) *OIDCRuntimeResolver {
	return &OIDCRuntimeResolver{repo: repo, cfg: cfg}
}

func (r *OIDCRuntimeResolver) Resolve() (*OIDCRuntimeState, error) {
	configs, err := r.repo.GetSystemConfigsByKeys([]string{
		configKeyOIDCIssuer,
		configKeyOIDCClientID,
		configKeyOIDCClientSecret,
	})
	if err != nil {
		return nil, err
	}

	state := &OIDCRuntimeState{
		IssuerEnvLocked:       strings.TrimSpace(r.cfg.OIDCIssuer) != "",
		ClientIDEnvLocked:     strings.TrimSpace(r.cfg.OIDCClientID) != "",
		ClientSecretEnvLocked: strings.TrimSpace(r.cfg.OIDCClientSecret) != "",
	}

	if state.IssuerEnvLocked {
		state.Issuer = strings.TrimSpace(r.cfg.OIDCIssuer)
		state.IssuerSource = urlSourceEnv
	} else if v := strings.TrimSpace(configs[configKeyOIDCIssuer]); v != "" {
		state.Issuer = v
		state.IssuerSource = urlSourceDB
	} else {
		state.IssuerSource = oidcSourceUnset
	}

	if state.ClientIDEnvLocked {
		state.ClientID = strings.TrimSpace(r.cfg.OIDCClientID)
		state.ClientIDSource = urlSourceEnv
	} else if v := strings.TrimSpace(configs[configKeyOIDCClientID]); v != "" {
		state.ClientID = v
		state.ClientIDSource = urlSourceDB
	} else {
		state.ClientIDSource = oidcSourceUnset
	}

	if state.ClientSecretEnvLocked {
		state.ClientSecret = strings.TrimSpace(r.cfg.OIDCClientSecret)
		state.ClientSecretSource = urlSourceEnv
	} else if v := strings.TrimSpace(configs[configKeyOIDCClientSecret]); v != "" {
		state.ClientSecret = strings.TrimSpace(crypto.DecryptOrPlaintext(v, r.cfg.EncryptionKey))
		state.ClientSecretSource = urlSourceDB
	} else {
		state.ClientSecretSource = oidcSourceUnset
	}

	state.ClientSecretConfigured = state.ClientSecret != ""
	state.MissingFields = collectOIDCMissingFields(state)
	state.Configured = len(state.MissingFields) == 0
	return state, nil
}

func (r *OIDCRuntimeResolver) ResolveProvider(ctx context.Context) (*OIDCRuntimeState, *oidc.ProviderConfig, error) {
	state, err := r.Resolve()
	if err != nil {
		return nil, nil, err
	}
	if !state.Configured {
		return state, nil, errOIDCNotConfigured
	}

	issuer, err := validateOIDCIssuer(state.Issuer)
	if err != nil {
		return state, nil, err
	}
	provider, err := oidc.Discover(ctx, issuer)
	if err != nil {
		return state, nil, err
	}
	state.Issuer = issuer
	return state, provider, nil
}

func validateOIDCIssuer(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("oidc issuer is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("oidc issuer must be an absolute URL")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported oidc issuer scheme %q", parsed.Scheme)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("oidc issuer cannot include query or fragment")
	}
	if parsed.User != nil {
		return "", errors.New("oidc issuer cannot include user info")
	}

	return trimmed, nil
}

func collectOIDCMissingFields(state *OIDCRuntimeState) []string {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(state.Issuer) == "" {
		missing = append(missing, configKeyOIDCIssuer)
	}
	if strings.TrimSpace(state.ClientID) == "" {
		missing = append(missing, configKeyOIDCClientID)
	}
	if strings.TrimSpace(state.ClientSecret) == "" {
		missing = append(missing, configKeyOIDCClientSecret)
	}
	return missing
}
