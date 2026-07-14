package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"hl6-server/internal/clientauth"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/oidc"
	"hl6-server/internal/referral"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"

	"gorm.io/gorm"
)

type OIDCHandler struct {
	repo         *repository.Repository
	cfg          *config.Config
	oidcResolver *OIDCRuntimeResolver
	urlResolver  *URLResolver
}

const (
	firstUserAdminLockKey         int64 = 19490331
	maxReferralCodeCreateAttempts       = 20
	nativeRedirectCookieName             = "hl6_native_redirect"
	nativeAuthRequestTTL                 = 90 * time.Second
	nativeAuthCodeTTL                    = 90 * time.Second
)

var (
	referralCodePattern         = regexp.MustCompile(`^[a-z]{5}$`)
	legacyReferralCodePattern   = regexp.MustCompile(`^[0-9a-f]{16}$`)
	nativeRedirectSchemePattern = regexp.MustCompile(`^[a-z][a-z0-9+.-]{0,63}$`)
)

func NewOIDCHandler(repo *repository.Repository, cfg *config.Config) *OIDCHandler {
	return &OIDCHandler{
		repo:         repo,
		cfg:          cfg,
		oidcResolver: NewOIDCRuntimeResolver(repo, cfg),
		urlResolver:  NewURLResolver(repo, cfg),
	}
}

// setSessionCookie sets a cookie with full attributes including SameSite=Lax.
func (h *OIDCHandler) setSessionCookie(c *gin.Context, name, value string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *OIDCHandler) Login(c *gin.Context) {
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		log.Printf("failed to resolve runtime URLs: %v", err)
		c.String(http.StatusInternalServerError, "failed to resolve runtime URL")
		return
	}
	runtimeState, provider, err := h.oidcResolver.ResolveProvider(c.Request.Context())
	if err != nil {
		if errors.Is(err, errOIDCNotConfigured) {
			response.ErrorWithKey(c, http.StatusServiceUnavailable, "oidc not configured", "error.oidcNotConfigured")
			return
		}
		log.Printf("failed to resolve oidc provider: %v", err)
		response.ErrorWithKey(c, http.StatusBadGateway, "failed to resolve oidc provider", "error.oidcDiscoveryFailed")
		return
	}

	callbackURL := strings.TrimRight(urlState.BackendURL, "/") + "/api/v1/auth/callback"
	secureCookie := strings.HasPrefix(urlState.FrontendURL, "https://")
	if strings.TrimSpace(c.Query("native_redirect_uri")) != "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "native redirect uri must be created through the client API", "error.invalidRequestBody")
		return
	}
	nativeRedirectURI, err := h.consumeNativeLoginRequest(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "invalid native login request", "error.invalidToken")
		return
	}
	if nativeRedirectURI == "" {
		h.setSessionCookie(c, nativeRedirectCookieName, "", -1, secureCookie)
	} else {
		h.setSessionCookie(c, nativeRedirectCookieName, base64.RawURLEncoding.EncodeToString([]byte(nativeRedirectURI)), 900, secureCookie)
	}

	state, err := generateRandomState()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate oauth state")
		return
	}

	// Store state in httpOnly cookie
	h.setSessionCookie(c, "hl6_state", state, 900, secureCookie) // 15 min TTL

	// Store referral code if present
	if ref := strings.ToLower(strings.TrimSpace(c.Query("ref"))); ref != "" && isValidReferralCode(ref) {
		h.setSessionCookie(c, "hl6_ref", ref, 900, secureCookie) // 15 min TTL
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		provider.AuthorizationEndpoint,
		url.QueryEscape(runtimeState.ClientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape("openid email profile"),
		url.QueryEscape(state),
	)

	c.Redirect(http.StatusFound, authURL)
}

func (h *OIDCHandler) Callback(c *gin.Context) {
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		log.Printf("failed to resolve runtime URLs: %v", err)
		c.String(http.StatusInternalServerError, "failed to resolve runtime URL")
		return
	}
	runtimeState, provider, err := h.oidcResolver.ResolveProvider(c.Request.Context())
	if err != nil {
		if errors.Is(err, errOIDCNotConfigured) {
			c.String(http.StatusServiceUnavailable, "oidc not configured")
			return
		}
		log.Printf("failed to resolve oidc provider: %v", err)
		c.String(http.StatusBadGateway, "authentication failed")
		return
	}
	loginURL := strings.TrimRight(urlState.BackendURL, "/") + "/api/v1/auth/login"
	callbackURL := strings.TrimRight(urlState.BackendURL, "/") + "/api/v1/auth/callback"
	frontendBaseURL := strings.TrimRight(urlState.FrontendURL, "/")
	frontendDashboardURL := frontendBaseURL + "/dashboard"
	secureCookie := strings.HasPrefix(urlState.FrontendURL, "https://")

	// 1. Verify state
	code := c.Query("code")
	state := c.Query("state")
	cookieState, err := c.Cookie("hl6_state")
	if err != nil || cookieState != state || state == "" {
		// Common after refresh/retry of a consumed callback URL; restart login flow.
		if code != "" {
			c.Redirect(http.StatusFound, loginURL)
			return
		}
		c.String(http.StatusBadRequest, "invalid state")
		return
	}

	// Clear state cookie
	h.setSessionCookie(c, "hl6_state", "", -1, secureCookie)
	nativeRedirectURI := h.consumeNativeRedirectURI(c, secureCookie)

	// 2. Exchange code for tokens
	tokenResp, err := h.exchangeCode(c.Request.Context(), provider, runtimeState, code, callbackURL)
	if err != nil {
		log.Printf("token exchange failed: %v", err)
		c.String(http.StatusBadGateway, "authentication failed")
		return
	}

	// 3. Parse ID token to get user info
	idTokenStr, ok := tokenResp["id_token"].(string)
	if !ok {
		log.Printf("no id_token in OIDC response")
		c.String(http.StatusBadGateway, "authentication failed")
		return
	}

	// Fetch JWKS for verification
	keySet, err := jwk.Fetch(c.Request.Context(), provider.JwksURI)
	if err != nil {
		log.Printf("failed to fetch JWKS: %v", err)
		c.String(http.StatusBadGateway, "authentication failed")
		return
	}

	// Try standard validation first. Some OIDC providers omit kid, so retry each
	// published JWK, but never accept an ID token without signature validation.
	idToken, err := jwt.Parse([]byte(idTokenStr),
		jwt.WithKeySet(keySet),
		jwt.WithIssuer(provider.Issuer),
		jwt.WithAudience(runtimeState.ClientID),
		jwt.WithAcceptableSkew(2*time.Minute),
	)
	if err != nil {
		log.Printf("standard id_token validation failed: %v", err)
		idToken, err = validateTokenWithAllKeys([]byte(idTokenStr), keySet, provider.Issuer, runtimeState.ClientID)
		if err != nil {
			log.Printf("id_token validation failed with all keys: %v", err)
			c.String(http.StatusBadGateway, "authentication failed")
			return
		}
		log.Printf("parsed id_token using all available keys for Authing compatibility")
	}

	sub := idToken.Subject()
	claims := idToken.PrivateClaims()
	audiences := idToken.Audience()
	if len(audiences) > 1 {
		azp, _ := claims["azp"].(string)
		if azp == "" || azp != runtimeState.ClientID {
			log.Printf("invalid id_token azp: %v", claims["azp"])
			c.String(http.StatusBadGateway, "authentication failed")
			return
		}
	}
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	
	// Authing compatibility: try alternative field names
	if name == "" {
		name, _ = claims["username"].(string)
	}
	if name == "" {
		name, _ = claims["nickname"].(string)
	}
	if name == "" {
		name, _ = claims["given_name"].(string)
	}
	if name == "" {
		name, _ = claims["family_name"].(string)
	}
	if picture == "" {
		picture, _ = claims["avatar"].(string)
	}
	if picture == "" {
		picture, _ = claims["avatar_url"].(string)
	}
	if email == "" {
		email, _ = claims["email_address"].(string)
	}

	// 4. Find or create user
	user, err := h.repo.FindUserByExternalID(sub)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("failed to find user by external_id %q: %v", sub, err)
			c.String(http.StatusInternalServerError, "failed to load user")
			return
		}

		// New user — create in a single transaction
		if strings.TrimSpace(name) == "" {
			name = defaultProfileName(email)
		}
		user = &model.User{
			ExternalID: sub,
			Email:      email,
			Name:       name,
			AvatarURL:  picture,
			Role:       "user",
		}

		// Assign default group
		if defaultGroup, err := h.repo.GetDefaultUserGroup(); err == nil {
			user.GroupID = &defaultGroup.ID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("failed to load default user group: %v", err)
			c.String(http.StatusInternalServerError, "failed to initialize user")
			return
		}

		tx := h.repo.GetDB().Begin()
		if tx.Error != nil {
			c.String(http.StatusInternalServerError, "failed to start transaction")
			return
		}

		// Serialize first-user role assignment and only grant admin to the first created user.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", firstUserAdminLockKey).Error; err != nil {
			tx.Rollback()
			c.String(http.StatusInternalServerError, "failed to initialize user role")
			return
		}

		var userCount int64
		if err := tx.Model(&model.User{}).Count(&userCount).Error; err != nil {
			tx.Rollback()
			c.String(http.StatusInternalServerError, "failed to initialize user role")
			return
		}
		if userCount == 0 {
			user.Role = "admin"
		}

		if err := h.createUserWithUniqueReferralCode(tx, user); err != nil {
			log.Printf("failed to create user %q with unique referral code: %v", sub, err)
			tx.Rollback()
			c.String(http.StatusInternalServerError, "failed to create user")
			return
		}

		// Create credit balance
		if err := tx.Create(&model.CreditBalance{UserID: user.ID, Balance: 0}).Error; err != nil {
			tx.Rollback()
			c.String(http.StatusInternalServerError, "failed to initialize user credits")
			return
		}

		// Audit log for registration
		regDetails, _ := json.Marshal(map[string]string{"email": email})
		if err := tx.Create(&model.AuditLog{
			UserID:     user.ID,
			Action:     "user_register",
			Resource:   "user",
			ResourceID: user.ID,
			Details:    regDetails,
		}).Error; err != nil {
			tx.Rollback()
			c.String(http.StatusInternalServerError, "failed to initialize user audit")
			return
		}

		// Grant registration bonus
		if bonusStr, err := h.repo.GetSystemConfig("registration_bonus_credits"); err == nil {
			if amount, ok := parseNonNegativeCreditConfigForRuntime(bonusStr); ok && amount > 0 {
				if err := h.repo.GrantCredits(tx, user.ID, amount, "txn.registrationBonus", nil); err != nil {
					tx.Rollback()
					c.String(http.StatusInternalServerError, "failed to grant registration bonus")
					return
				}
				bonusDetails, _ := json.Marshal(map[string]interface{}{"amount": amount})
				if err := tx.Create(&model.AuditLog{
					UserID:     user.ID,
					Action:     "user_registration_bonus",
					Resource:   "credit",
					ResourceID: user.ID,
					Details:    bonusDetails,
				}).Error; err != nil {
					tx.Rollback()
					c.String(http.StatusInternalServerError, "failed to write bonus audit log")
					return
				}
			}
		}

		// Process referral
		if refCode, err := c.Cookie("hl6_ref"); err == nil && refCode != "" {
			h.setSessionCookie(c, "hl6_ref", "", -1, secureCookie) // Clear cookie
			if err := h.processReferral(tx, user, refCode); err != nil {
				log.Printf("failed to process referral for user %d: %v", user.ID, err)
				tx.Rollback()
				c.String(http.StatusInternalServerError, "failed to process referral")
				return
			}
		}

		if err := tx.Commit().Error; err != nil {
			c.String(http.StatusInternalServerError, "failed to finalize user creation")
			return
		}

		// Reload user with group
		loaded, loadErr := h.repo.FindUserByExternalID(sub)
		if loadErr != nil {
			c.String(http.StatusInternalServerError, "failed to reload user")
			return
		}
		user = loaded
	} else {
		// OIDC claims are an identity source, not profile ownership. Only fill
		// fields which have never been set by the user.
		profileChanged := false
		if email = strings.TrimSpace(email); email != "" && email != user.Email {
			user.Email = email
			profileChanged = true
		}
		if user.Name == "" && strings.TrimSpace(name) != "" {
			user.Name = strings.TrimSpace(name)
			profileChanged = true
		}
		if user.Name == "" {
			user.Name = defaultProfileName(user.Email)
			profileChanged = true
		}
		if profileChanged && h.repo.UpdateUser(user) != nil {
			c.String(http.StatusInternalServerError, "failed to update user profile")
			return
		}
	}

	// Banned users cannot create new sessions.
	if user.IsBanned && user.BannedUntil != nil && !user.BannedUntil.After(time.Now()) {
		if err := h.repo.UnbanUser(user.ID); err != nil {
			c.String(http.StatusInternalServerError, "failed to restore expired ban")
			return
		}
		user.IsBanned = false
		user.BannedReason = ""
		user.BannedAt = nil
		user.BannedUntil = nil
	}
	if nativeRedirectURI != "" {
		h.redirectNativeAuth(c, nativeRedirectURI, user.ExternalID)
		return
	}
	if user.IsBanned {
		// Issue a restricted session so the user can read ban details and submit an
		// appeal. Auth middleware blocks every other protected endpoint.
		sessionToken, err := h.issueSessionJWT(user.ExternalID)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to issue banned session")
			return
		}
		h.setSessionCookie(c, "hl6_session", sessionToken, 7*24*60*60, secureCookie)
		c.Redirect(http.StatusFound, buildBannedRedirectURL(frontendBaseURL, user.BannedReason))
		return
	}

	// 5. Issue session JWT
	sessionToken, err := h.issueSessionJWT(user.ExternalID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to issue session")
		return
	}

	// 6. Set session cookie
	maxAge := 7 * 24 * 60 * 60 // 7 days
	h.setSessionCookie(c, "hl6_session", sessionToken, maxAge, secureCookie)

	// 7. Redirect to dashboard
	c.Redirect(http.StatusFound, frontendDashboardURL)
}

func (h *OIDCHandler) NativeStart(c *gin.Context) {
	var body struct {
		RedirectURI string `json:"redirect_uri"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	redirectURI := strings.TrimSpace(body.RedirectURI)
	if !isValidNativeRedirectURI(redirectURI) {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid native redirect uri", "error.invalidRequestBody")
		return
	}
	requestToken, err := generateRandomState()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create native login request", "error.databaseError")
		return
	}
	requestHash := sha256.Sum256([]byte(requestToken))
	if err := h.repo.CreateNativeAuthRequest(hex.EncodeToString(requestHash[:]), redirectURI, time.Now().Add(nativeAuthRequestTTL)); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save native login request", "error.databaseError")
		return
	}
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		log.Printf("failed to resolve runtime URLs: %v", err)
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime URL", "error.databaseError")
		return
	}
	loginURL, err := url.Parse(strings.TrimRight(urlState.BackendURL, "/") + "/api/v1/auth/login")
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to build native login url", "error.databaseError")
		return
	}
	query := loginURL.Query()
	query.Set("native_request", requestToken)
	loginURL.RawQuery = query.Encode()
	response.OK(c, gin.H{"login_url": loginURL.String()})
}

func (h *OIDCHandler) NativeExchange(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	code := strings.TrimSpace(body.Code)
	if code == "" || len(code) > 128 {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid native auth code", "error.invalidRequestBody")
		return
	}
	codeHash := sha256.Sum256([]byte(code))
	externalID, err := h.repo.ConsumeNativeAuthCode(hex.EncodeToString(codeHash[:]))
	if err != nil {
		if errors.Is(err, repository.ErrNativeAuthCodeInvalid) {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid native auth code", "error.invalidToken")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to exchange native auth code", "error.databaseError")
		return
	}
	clientKeyHash, err := h.repo.GetSystemConfig(clientauth.CommunicationKeyHashConfigKey)
	if err != nil || clientKeyHash == "" {
		response.ErrorWithKey(c, http.StatusUnauthorized, "invalid client key", "error.invalidToken")
		return
	}
	token, err := h.issueNativeSessionJWT(externalID, clientKeyHash)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to issue native session", "error.databaseError")
		return
	}
	response.OK(c, gin.H{
		"access_token": token,
		"expires_in":   int((7 * 24 * time.Hour).Seconds()),
	})
}

func (h *OIDCHandler) consumeNativeLoginRequest(c *gin.Context) (string, error) {
	requestToken := strings.TrimSpace(c.Query("native_request"))
	if requestToken == "" {
		return "", nil
	}
	if len(requestToken) > 128 {
		return "", repository.ErrNativeAuthRequestInvalid
	}
	requestHash := sha256.Sum256([]byte(requestToken))
	redirectURI, err := h.repo.ConsumeNativeAuthRequest(hex.EncodeToString(requestHash[:]))
	if err != nil {
		return "", err
	}
	if !isValidNativeRedirectURI(redirectURI) {
		return "", repository.ErrNativeAuthRequestInvalid
	}
	return redirectURI, nil
}

func (h *OIDCHandler) consumeNativeRedirectURI(c *gin.Context, secureCookie bool) string {
	encoded, err := c.Cookie(nativeRedirectCookieName)
	h.setSessionCookie(c, nativeRedirectCookieName, "", -1, secureCookie)
	if err != nil || encoded == "" {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	redirectURI := string(raw)
	if !isValidNativeRedirectURI(redirectURI) {
		return ""
	}
	return redirectURI
}

func (h *OIDCHandler) redirectNativeAuth(c *gin.Context, redirectURI, externalID string) {
	code, err := generateRandomState()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create native auth code")
		return
	}
	codeHash := sha256.Sum256([]byte(code))
	if err := h.repo.CreateNativeAuthCode(hex.EncodeToString(codeHash[:]), externalID, time.Now().Add(nativeAuthCodeTTL)); err != nil {
		c.String(http.StatusInternalServerError, "failed to save native auth code")
		return
	}
	target, err := url.Parse(redirectURI)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid native redirect uri")
		return
	}
	query := target.Query()
	query.Set("code", code)
	target.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, target.String())
}

func isValidNativeRedirectURI(rawURI string) bool {
	parsed, err := url.ParseRequestURI(rawURI)
	if err != nil || !nativeRedirectSchemePattern.MatchString(parsed.Scheme) {
		return false
	}
	return parsed.Host == "auth" && parsed.Path == "/callback" && parsed.RawQuery == "" && parsed.Fragment == ""
}

func defaultProfileName(email string) string {
	email = strings.TrimSpace(email)
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at]
	}
	if email != "" {
		return email
	}
	return "User"
}

func buildBannedRedirectURL(frontendBaseURL, reason string) string {
	targetURL, err := url.Parse(frontendBaseURL + "/")
	if err != nil {
		values := url.Values{"error": []string{"user_banned"}}
		trimmedReason := strings.TrimSpace(reason)
		if trimmedReason != "" {
			values.Set("reason", trimmedReason)
		}
		return strings.TrimRight(frontendBaseURL, "/") + "/banned?" + values.Encode()
	}

	query := targetURL.Query()
	targetURL.Path = strings.TrimRight(targetURL.Path, "/") + "/banned"
	query.Set("error", "user_banned")
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason != "" {
		query.Set("reason", trimmedReason)
	} else {
		query.Del("reason")
	}
	targetURL.RawQuery = query.Encode()

	return targetURL.String()
}

func (h *OIDCHandler) Logout(c *gin.Context) {
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		log.Printf("failed to resolve runtime URLs: %v", err)
		c.String(http.StatusInternalServerError, "failed to resolve runtime URL")
		return
	}
	secureCookie := strings.HasPrefix(urlState.FrontendURL, "https://")
	h.setSessionCookie(c, "hl6_session", "", -1, secureCookie)

	_, provider, err := h.oidcResolver.ResolveProvider(c.Request.Context())
	if err != nil {
		response.OK(c, gin.H{"logout_url": urlState.FrontendURL})
		return
	}

	if provider.EndSessionEndpoint != "" {
		logoutURL := fmt.Sprintf("%s?post_logout_redirect_uri=%s",
			provider.EndSessionEndpoint,
			url.QueryEscape(urlState.FrontendURL),
		)
		response.OK(c, gin.H{"logout_url": logoutURL})
	} else {
		response.OK(c, gin.H{"logout_url": urlState.FrontendURL})
	}
}

func (h *OIDCHandler) exchangeCode(ctx context.Context, provider *oidc.ProviderConfig, runtimeState *OIDCRuntimeState, code, redirectURI string) (map[string]interface{}, error) {
	// Some OIDC providers (like Authing) may require different authentication methods
	// Try client_secret_post first (standard), then fall back to other methods if needed
	
	// Standard method: client credentials in request body
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {runtimeState.ClientID},
		"client_secret": {runtimeState.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}
	
	// Handle different response codes that might be returned by different OIDC providers
	if resp.StatusCode != http.StatusOK {
		// Some providers might return 2xx codes other than 200
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Accept other 2xx codes as successful
		} else {
			return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return result, nil
}

func (h *OIDCHandler) issueSessionJWT(externalID string) (string, error) {
	return h.issueSessionJWTWithClientType(externalID, false, "")
}

func (h *OIDCHandler) issueNativeSessionJWT(externalID, clientKeyHash string) (string, error) {
	return h.issueSessionJWTWithClientType(externalID, true, clientKeyHash)
}

func (h *OIDCHandler) issueSessionJWTWithClientType(externalID string, nativeClient bool, clientKeyHash string) (string, error) {
	key, err := jwk.FromRaw([]byte(h.cfg.SessionSecret))
	if err != nil {
		return "", err
	}

	now := time.Now()
	builder := jwt.NewBuilder().
		Subject(externalID).
		Issuer("hl6").
		Audience([]string{"hl6"}).
		IssuedAt(now).
		Expiration(now.Add(7 * 24 * time.Hour))
	if nativeClient {
		if clientKeyHash == "" {
			return "", errors.New("native session requires a client key hash")
		}
		builder = builder.Claim(clientauth.NativeSessionClaim, true)
		builder = builder.Claim(clientauth.NativeSessionKeyHashClaim, clientKeyHash)
	}
	token, err := builder.Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, key))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

func (h *OIDCHandler) createUserWithUniqueReferralCode(tx *gorm.DB, user *model.User) error {
	for attempts := 0; attempts < maxReferralCodeCreateAttempts; attempts++ {
		code, err := referral.GenerateCode(5)
		if err != nil {
			return err
		}
		user.ID = 0
		user.ReferralCode = code
		if err := tx.Create(user).Error; err != nil {
			if referral.IsCodeUniqueViolation(err) {
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("unable to generate unique referral code after %d attempts", maxReferralCodeCreateAttempts)
}

func isValidReferralCode(code string) bool {
	return referralCodePattern.MatchString(code) || legacyReferralCodePattern.MatchString(code)
}

func (h *OIDCHandler) processReferral(tx *gorm.DB, newUser *model.User, refCode string) error {
	// Check if referral feature is enabled
	enabledStr, err := h.repo.GetSystemConfig("referral_enabled")
	if err != nil || enabledStr != "true" {
		return nil
	}

	// Find inviter by referral code
	inviter, err := h.repo.FindUserByReferralCode(refCode)
	if err != nil {
		return nil
	}

	// Prevent self-referral
	if inviter.ID == newUser.ID {
		return nil
	}

	inviterCreditsStr, err := h.repo.GetSystemConfig("referral_inviter_credits")
	if err != nil {
		inviterCreditsStr = "0"
	}
	inviteeCreditsStr, err := h.repo.GetSystemConfig("referral_invitee_credits")
	if err != nil {
		inviteeCreditsStr = "0"
	}
	inviterCredits, inviterOK := parseNonNegativeCreditConfigForRuntime(inviterCreditsStr)
	if !inviterOK {
		inviterCredits = 0
	}
	inviteeCredits, inviteeOK := parseNonNegativeCreditConfigForRuntime(inviteeCreditsStr)
	if !inviteeOK {
		inviteeCredits = 0
	}

	// Grant credits to inviter
	if inviterCredits > 0 {
		inviterParams, _ := json.Marshal(map[string]string{"name": newUser.Name})
		if err := h.repo.GrantCredits(tx, inviter.ID, inviterCredits, "txn.referralInviter", inviterParams); err != nil {
			return err
		}
	}

	// Grant credits to invitee
	if inviteeCredits > 0 {
		inviteeParams, _ := json.Marshal(map[string]string{"name": inviter.Name})
		if err := h.repo.GrantCredits(tx, newUser.ID, inviteeCredits, "txn.referralInvitee", inviteeParams); err != nil {
			return err
		}
	}

	// Create referral record
	if err := tx.Create(&model.UserReferral{
		InviterID:      inviter.ID,
		InviteeID:      newUser.ID,
		InviterCredits: inviterCredits,
		InviteeCredits: inviteeCredits,
	}).Error; err != nil {
		return err
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"inviter_id":      inviter.ID,
		"invitee_id":      newUser.ID,
		"inviter_credits": inviterCredits,
		"invitee_credits": inviteeCredits,
	})
	if err := tx.Create(&model.AuditLog{
		UserID:     newUser.ID,
		Action:     "user_referral",
		Resource:   "user",
		ResourceID: newUser.ID,
		Details:    details,
	}).Error; err != nil {
		return err
	}
	return nil
}

func generateRandomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// validateTokenWithAllKeys tries to validate the token with all available keys in the key set
// This is useful for providers like Authing that may not include kid (Key ID) in the JWT header
func validateTokenWithAllKeys(tokenBytes []byte, keySet jwk.Set, issuer, audience string) (jwt.Token, error) {
	// Iterate through all keys in the key set and try each one
	iter := keySet.Keys(context.Background())
	for iter.Next(context.Background()) {
		pair := iter.Pair()
		key, ok := pair.Value.(jwk.Key)
		if !ok {
			continue
		}

			// Get the algorithm from the key
		algIntf := key.Algorithm()
		if algIntf == nil {
			// If algorithm is not specified in the key, try common ones
			// Most OIDC providers use RS256
			algIntf = jwa.RS256
		}
		
		alg, ok := algIntf.(jwa.SignatureAlgorithm)
		if !ok {
			// If we can't determine the algorithm properly, skip this key
			continue
		}

		// Try to validate the token with this specific key
		idToken, err := jwt.Parse(tokenBytes,
			jwt.WithKey(alg, key),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(audience),
			jwt.WithAcceptableSkew(2*time.Minute),
		)
		
		if err == nil {
			// Validation succeeded with this key
			return idToken, nil
		}
	}

	// If we get here, none of the keys worked
	return nil, fmt.Errorf("failed to validate token with any of the available keys")
}
