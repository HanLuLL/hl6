package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"hl6-server/internal/auth"
	"hl6-server/internal/clientauth"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gorm.io/gorm"
)

const (
	localAuthEnabledConfigKey    = "auth.local.enabled"
	registrationEnabledConfigKey = "auth.registration.enabled"
	emailDomainModeConfigKey     = "auth.email_domain.mode"
	emailDomainDomainsConfigKey  = "auth.email_domain.domains"
	sessionVersionClaim          = "hl6_session_version"
	browserSessionTTL            = 7 * 24 * time.Hour
	nativeSessionTTL             = 24 * time.Hour
	authTokenTTL                 = 30 * time.Minute
	authRequestRateLimit         = 10
	authRequestRateLimitWindow   = 15 * time.Minute
	maximumAuthTokenLength       = 256
)

var referralCodePattern = regexp.MustCompile(`^(?:[a-z]{5}|[0-9a-f]{16})$`)

type EmailAuthHandler struct {
	repo        *repository.Repository
	emailSvc    *service.EmailService
	cfg         *config.Config
	urlResolver *URLResolver
}

type emailRequest struct {
	Email        string `json:"email"`
	ReferralCode string `json:"referral_code"`
	Locale       string `json:"locale"`
}

type passwordCompleteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registrationTokenPayload struct {
	ReferralCode string `json:"referral_code,omitempty"`
}

type registrationSettings struct {
	Enabled bool
	Policy  auth.DomainPolicy
}

func NewEmailAuthHandler(repo *repository.Repository, emailSvc *service.EmailService, cfg *config.Config) *EmailAuthHandler {
	return &EmailAuthHandler{
		repo:        repo,
		emailSvc:    emailSvc,
		cfg:         cfg,
		urlResolver: NewURLResolver(repo, cfg),
	}
}

func (h *EmailAuthHandler) RegistrationRequest(c *gin.Context) {
	h.setPublicAuthHeaders(c)
	if !h.requireLocalAuthEnabled(c) {
		return
	}

	var body emailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	emailNormalized, err := auth.NormalizeEmail(body.Email)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid email address", "error.invalidRequestBody")
		return
	}
	if !h.allowAuthRequest(c, "registration_request", emailNormalized) {
		return
	}

	settings, err := h.loadRegistrationSettings()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load registration settings", "error.databaseError")
		return
	}
	if !settings.Enabled {
		h.recordSecurityEvent(c, nil, "registration_request", model.AuthSecurityOutcomeFailure, emailNormalized)
		response.ErrorWithKey(c, http.StatusForbidden, "registration is currently disabled", "error.registrationDisabled")
		return
	}
	if err := auth.ValidateRegistrationDomain(emailNormalized, settings.Policy); err != nil {
		h.recordSecurityEvent(c, nil, "registration_request", model.AuthSecurityOutcomeFailure, emailNormalized)
		response.ErrorWithKey(c, http.StatusForbidden, "this email domain is not accepted for registration", "error.registrationDomainDenied")
		return
	}

	if _, err := h.repo.FindCredentialByEmail(emailNormalized); err == nil {
		h.recordSecurityEvent(c, nil, "registration_request", model.AuthSecurityOutcomeSuccess, emailNormalized)
		h.accepted(c)
		return
	} else if !errors.Is(err, repository.ErrCredentialNotFound) {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to check account", "error.databaseError")
		return
	}
	if h.emailSvc == nil || !h.emailSvc.IsEnabled() {
		response.ErrorWithKey(c, http.StatusServiceUnavailable, "email verification is unavailable", "error.emailUnavailable")
		return
	}

	referralCode := strings.ToLower(strings.TrimSpace(body.ReferralCode))
	if referralCode != "" && !referralCodePattern.MatchString(referralCode) {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid referral code", "error.invalidRequestBody")
		return
	}
	payload, err := json.Marshal(registrationTokenPayload{ReferralCode: referralCode})
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to prepare registration", "error.databaseError")
		return
	}
	if err := h.createAndSendAuthToken(c, model.AuthTokenPurposeRegistrationVerify, nil, emailNormalized, requestedAuthEmailLocale(c, body.Locale), payload); err != nil {
		log.Printf("[auth] registration email failed for %s: %v", emailNormalized, err)
		response.ErrorWithKey(c, http.StatusBadGateway, "failed to send verification email", "error.emailUnavailable")
		return
	}
	h.recordSecurityEvent(c, nil, "registration_request", model.AuthSecurityOutcomeSuccess, emailNormalized)
	h.accepted(c)
}

func (h *EmailAuthHandler) ActivationRequest(c *gin.Context) {
	h.setPublicAuthHeaders(c)
	if !h.requireLocalAuthEnabled(c) {
		return
	}

	emailNormalized, emailLocale, ok := h.parseEmailRequest(c)
	if !ok {
		return
	}
	if !h.allowAuthRequest(c, "activation_request", emailNormalized) {
		return
	}

	credential, err := h.repo.FindCredentialByEmail(emailNormalized)
	if err != nil {
		h.recordSecurityEvent(c, nil, "activation_request", model.AuthSecurityOutcomeFailure, emailNormalized)
		h.accepted(c)
		return
	}
	// 账号已激活：返回明确错误提示
	if credential.ActivationRequiredAt == nil && credential.PasswordSetAt != nil {
		h.recordSecurityEvent(c, &credential.UserID, "activation_request", model.AuthSecurityOutcomeFailure, emailNormalized)
		response.ErrorWithKey(c, http.StatusBadRequest, "account already activated", "error.accountAlreadyActivated")
		return
	}
	user, err := h.repo.FindUserByID(credential.UserID)
	if err == nil && h.emailSvc != nil && h.emailSvc.IsEnabled() {
		if err := h.createAndSendAuthToken(c, model.AuthTokenPurposeAccountActivation, &user.ID, credential.EmailNormalized, emailLocale, nil); err != nil {
			log.Printf("[auth] activation email failed for %s: %v", emailNormalized, err)
			h.recordSecurityEvent(c, &user.ID, "activation_request", model.AuthSecurityOutcomeFailure, emailNormalized)
			h.accepted(c)
			return
		}
		h.recordSecurityEvent(c, &user.ID, "activation_request", model.AuthSecurityOutcomeSuccess, emailNormalized)
		h.accepted(c)
		return
	}
	h.recordSecurityEvent(c, &credential.UserID, "activation_request", model.AuthSecurityOutcomeFailure, emailNormalized)
	h.accepted(c)
}

func (h *EmailAuthHandler) ForgotPassword(c *gin.Context) {
	h.setPublicAuthHeaders(c)
	if !h.requireLocalAuthEnabled(c) {
		return
	}

	emailNormalized, emailLocale, ok := h.parseEmailRequest(c)
	if !ok {
		return
	}
	if !h.allowAuthRequest(c, "password_forgot", emailNormalized) {
		return
	}

	credential, err := h.repo.FindCredentialByEmail(emailNormalized)
	if err != nil || credential.PasswordSetAt == nil || credential.ActivationRequiredAt != nil || h.emailSvc == nil || !h.emailSvc.IsEnabled() {
		h.recordSecurityEvent(c, nil, "password_forgot", model.AuthSecurityOutcomeFailure, emailNormalized)
		h.accepted(c)
		return
	}
	user, err := h.repo.FindUserByID(credential.UserID)
	if err != nil {
		h.recordSecurityEvent(c, &credential.UserID, "password_forgot", model.AuthSecurityOutcomeFailure, emailNormalized)
		h.accepted(c)
		return
	}
	if err := h.createAndSendAuthToken(c, model.AuthTokenPurposePasswordReset, &user.ID, credential.EmailNormalized, emailLocale, nil); err != nil {
		log.Printf("[auth] password reset email failed for %s: %v", emailNormalized, err)
		h.recordSecurityEvent(c, &user.ID, "password_forgot", model.AuthSecurityOutcomeFailure, emailNormalized)
		h.accepted(c)
		return
	}
	h.recordSecurityEvent(c, &user.ID, "password_forgot", model.AuthSecurityOutcomeSuccess, emailNormalized)
	h.accepted(c)
}

func (h *EmailAuthHandler) CompletePassword(c *gin.Context) {
	h.setPublicAuthHeaders(c)
	if !h.requireLocalAuthEnabled(c) {
		return
	}

	var body passwordCompleteRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	rawToken := strings.TrimSpace(body.Token)
	if rawToken == "" || len(rawToken) > maximumAuthTokenLength {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid or expired link", "error.invalidToken")
		return
	}
	if err := auth.ValidatePassword(body.Password); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "password must contain 8 to 128 characters", "error.invalidPassword")
		return
	}

	token, err := h.repo.ConsumeAnyAuthToken(c.Request.Context(), auth.HashToken(rawToken), []string{
		model.AuthTokenPurposeRegistrationVerify,
		model.AuthTokenPurposeAccountActivation,
		model.AuthTokenPurposePasswordReset,
	})
	if err != nil {
		h.recordSecurityEvent(c, nil, "password_complete", model.AuthSecurityOutcomeFailure, "")
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid or expired link", "error.invalidToken")
		return
	}
	// A raw token is single-use. Consume it before the expensive Argon2id
	// derivation so arbitrary invalid inputs cannot exhaust server resources.
	passwordHash, err := auth.HashPassword(body.Password, h.peppers())
	if err != nil {
		h.recordSecurityEvent(c, token.UserID, "password_complete", model.AuthSecurityOutcomeFailure, token.EmailNormalized)
		response.ErrorWithKey(c, http.StatusServiceUnavailable, "password authentication is unavailable", "error.authUnavailable")
		return
	}

	user, credential, err := h.completePassword(c, token, passwordHash)
	if err != nil {
		h.recordSecurityEvent(c, token.UserID, "password_complete", model.AuthSecurityOutcomeFailure, token.EmailNormalized)
		response.ErrorWithKey(c, http.StatusBadRequest, "unable to set password", "error.invalidRequestBody")
		return
	}
	h.recordSecurityEvent(c, &user.ID, "password_complete", model.AuthSecurityOutcomeSuccess, credential.EmailNormalized)
	h.writeSession(c, user, credential)
}

func (h *EmailAuthHandler) Login(c *gin.Context) {
	h.setPublicAuthHeaders(c)
	if !h.requireLocalAuthEnabled(c) {
		return
	}

	var body loginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	emailNormalized, err := auth.NormalizeEmail(body.Email)
	if err != nil {
		h.recordSecurityEvent(c, nil, "login", model.AuthSecurityOutcomeFailure, "")
		h.invalidLogin(c)
		return
	}
	if !h.allowAuthRequest(c, "login", emailNormalized) {
		return
	}

	credential, err := h.repo.FindCredentialByEmail(emailNormalized)
	if err != nil || credential.PasswordSetAt == nil {
		h.recordSecurityEvent(c, nil, "login", model.AuthSecurityOutcomeFailure, emailNormalized)
		h.invalidLogin(c)
		return
	}
	if credential.ActivationRequiredAt != nil {
		h.recordSecurityEvent(c, nil, "login", model.AuthSecurityOutcomeFailure, emailNormalized)
		response.ErrorWithKey(c, http.StatusForbidden, "account requires activation", "error.accountActivationRequired")
		return
	}
	valid, needsRehash, err := auth.VerifyPassword(body.Password, credential.PasswordHash, h.peppers())
	if err != nil || !valid {
		h.recordSecurityEvent(c, &credential.UserID, "login", model.AuthSecurityOutcomeFailure, emailNormalized)
		h.invalidLogin(c)
		return
	}
	if needsRehash {
		newHash, err := auth.HashPassword(body.Password, h.peppers())
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to refresh password hash", "error.databaseError")
			return
		}
		credential, err = h.repo.UpdateCredentialPassword(c.Request.Context(), credential.UserID, newHash, "argon2id")
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to refresh password hash", "error.databaseError")
			return
		}
	}
	user, err := h.repo.FindUserByID(credential.UserID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "invalid credentials", "error.invalidToken")
		return
	}
	// 单设备登录：新登录踢掉其他设备
	if err := h.repo.IncrementSessionVersion(credential.UserID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create session", "error.databaseError")
		return
	}
	// 重新获取 credential 以获取新的 session_version
	credential, err = h.repo.FindCredentialByEmail(emailNormalized)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create session", "error.databaseError")
		return
	}
	h.recordSecurityEvent(c, &user.ID, "login", model.AuthSecurityOutcomeSuccess, credential.EmailNormalized)
	h.writeSession(c, user, credential)
}

func (h *EmailAuthHandler) Logout(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}
	// 单设备登录：登出只清除当前设备的 session，不影响其他设备
	// 如果其他设备已登录，它们的 session 仍然有效
	h.clearSessionCookie(c)
	response.OK(c, gin.H{"logout_url": h.frontendURL(c)})
}

func (h *EmailAuthHandler) completePassword(c *gin.Context, token *model.AuthToken, passwordHash string) (*model.User, *model.UserCredential, error) {
	if token == nil {
		return nil, nil, repository.ErrAuthTokenUnavailable
	}
	switch token.Purpose {
	case model.AuthTokenPurposeRegistrationVerify:
		payload, err := parseRegistrationTokenPayload(token.Payload)
		if err != nil {
			return nil, nil, err
		}
		rewards, err := h.registrationRewards()
		if err != nil {
			return nil, nil, err
		}
		return h.repo.CreateUserWithCredential(c.Request.Context(), repository.NewUserInput{
			Email:                  token.EmailNormalized,
			EmailNormalized:        token.EmailNormalized,
			Name:                   defaultEmailProfileName(token.EmailNormalized),
			PasswordHash:           passwordHash,
			PasswordHashVersion:    "argon2id",
			ReferralCode:           payload.ReferralCode,
			ReferralEnabled:        rewards.enabled,
			ReferralInviterCredits: rewards.inviterCredits,
			ReferralInviteeCredits: rewards.inviteeCredits,
			RegistrationBonus:      rewards.registrationBonus,
		})
	case model.AuthTokenPurposeAccountActivation, model.AuthTokenPurposePasswordReset:
		if token.UserID == nil {
			return nil, nil, repository.ErrAuthTokenUnavailable
		}
		credential, err := h.repo.FindCredentialByUserID(*token.UserID)
		if err != nil || credential.EmailNormalized != token.EmailNormalized {
			return nil, nil, repository.ErrAuthTokenUnavailable
		}
		credential, err = h.repo.UpdateCredentialPasswordFromAuthToken(c.Request.Context(), token, passwordHash, "argon2id")
		if err != nil {
			return nil, nil, err
		}
		user, err := h.repo.FindUserByID(*token.UserID)
		if err != nil {
			return nil, nil, err
		}
		return user, credential, nil
	default:
		return nil, nil, repository.ErrAuthTokenUnavailable
	}
}

func (h *EmailAuthHandler) createAndSendAuthToken(c *gin.Context, purpose string, userID *uint, emailNormalized, locale string, payload json.RawMessage) error {
	rawToken, err := auth.NewRawToken()
	if err != nil {
		return err
	}
	link, err := h.authenticationLink(c, purpose, rawToken)
	if err != nil {
		return err
	}
	// Generate app deep link (uses custom scheme hl6://)
	appLink := h.appAuthenticationLink(purpose, rawToken)
	token := &model.AuthToken{
		Purpose:         purpose,
		UserID:          userID,
		EmailNormalized: emailNormalized,
		TokenHash:       auth.HashToken(rawToken),
		Payload:         payload,
		ExpiresAt:       time.Now().UTC().Add(authTokenTTL),
	}
	if err := h.repo.CreateAuthToken(token); err != nil {
		return err
	}
	return h.emailSvc.SendAuthenticationLink(emailNormalized, purpose, link, appLink, locale, userID)
}

func (h *EmailAuthHandler) authenticationLink(c *gin.Context, purpose, rawToken string) (string, error) {
	state, err := h.urlResolver.Resolve(c)
	if err != nil {
		return "", err
	}
	if !state.FrontendEnvLocked && !state.Confirmed {
		log.Printf("[auth] frontend URL not confirmed (source=%s, url=%s), cannot send authentication links. Admin must confirm URL in Administration -> Site and Appearance.", state.FrontendSource, state.FrontendURL)
		return "", errors.New("frontend URL must be explicitly confirmed before sending authentication links")
	}
	path := "/set-password"
	if purpose == model.AuthTokenPurposePasswordReset {
		path = "/reset-password"
	}
	target, err := url.Parse(state.FrontendURL)
	if err != nil {
		return "", err
	}
	target.Path = strings.TrimRight(target.Path, "/") + path
	query := target.Query()
	query.Set("token", rawToken)
	target.RawQuery = query.Encode()
	return target.String(), nil
}

// appAuthenticationLink generates a deep link URL for the mobile app.
// Format: hl6://activate?token={token} (for registration verification and account activation)
// or hl6://reset-password?token={token} (for password reset)
func (h *EmailAuthHandler) appAuthenticationLink(purpose, rawToken string) string {
	path := "activate"
	if purpose == model.AuthTokenPurposePasswordReset {
		path = "reset-password"
	}
	return fmt.Sprintf("hl6://%s?token=%s", path, rawToken)
}

func (h *EmailAuthHandler) writeSession(c *gin.Context, user *model.User, credential *model.UserCredential) {
	if user == nil || credential == nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to establish session", "error.databaseError")
		return
	}
	if user.IsBanned && user.BannedUntil != nil && !user.BannedUntil.After(time.Now()) {
		if err := h.repo.UnbanUser(user.ID); err == nil {
			user.IsBanned = false
			user.BannedReason = ""
			user.BannedAt = nil
			user.BannedUntil = nil
		}
	}

	native, clientKeyHash, err := h.nativeRequest(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "invalid client key", "error.invalidToken")
		return
	}
	token, err := h.issueSessionJWT(user.ID, credential.SessionVersion, native, clientKeyHash)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to establish session", "error.databaseError")
		return
	}

	data := gin.H{
		"user":          user,
		"banned":        user.IsBanned,
		"banned_reason": user.BannedReason,
		"banned_at":     user.BannedAt,
		"banned_until":  user.BannedUntil,
	}
	if native {
		data["access_token"] = token
		data["expires_in"] = int(nativeSessionTTL.Seconds())
	} else {
		h.setSessionCookie(c, token, browserSessionTTL)
		data["access_token"] = token
		data["expires_in"] = int(browserSessionTTL.Seconds())
	}
	c.Header("Cache-Control", "no-store")
	response.OK(c, data)
}

func (h *EmailAuthHandler) issueSessionJWT(userID uint, sessionVersion uint, native bool, clientKeyHash string) (string, error) {
	if userID == 0 || sessionVersion == 0 || h.cfg == nil || strings.TrimSpace(h.cfg.SessionSecret) == "" {
		return "", errors.New("session signing configuration is invalid")
	}
	key, err := jwk.FromRaw([]byte(h.cfg.SessionSecret))
	if err != nil {
		return "", err
	}
	ttl := browserSessionTTL
	if native {
		if clientKeyHash == "" {
			return "", errors.New("native session requires a client key hash")
		}
		ttl = nativeSessionTTL
	}
	now := time.Now().UTC()
	builder := jwt.NewBuilder().
		Subject(strconv.FormatUint(uint64(userID), 10)).
		Issuer("hl6").
		Audience([]string{"hl6"}).
		IssuedAt(now).
		Expiration(now.Add(ttl)).
		Claim(sessionVersionClaim, sessionVersion)
	if native {
		builder = builder.Claim(clientauth.NativeSessionClaim, true).Claim(clientauth.NativeSessionKeyHashClaim, clientKeyHash)
	}
	signed, err := builder.Build()
	if err != nil {
		return "", err
	}
	encoded, err := jwt.Sign(signed, jwt.WithKey(jwa.HS256, key))
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (h *EmailAuthHandler) nativeRequest(c *gin.Context) (bool, string, error) {
	presentedKey := strings.TrimSpace(c.GetHeader("X-HL6-Client-Key"))
	if presentedKey == "" {
		return false, "", nil
	}
	storedHash, err := h.repo.GetSystemConfig(clientauth.CommunicationKeyHashConfigKey)
	if err != nil || !clientauth.IsAuthorized(presentedKey, storedHash) {
		return false, "", errors.New("client key is invalid")
	}
	return true, storedHash, nil
}

func (h *EmailAuthHandler) parseEmailRequest(c *gin.Context) (string, string, bool) {
	var body emailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return "", "", false
	}
	emailNormalized, err := auth.NormalizeEmail(body.Email)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid email address", "error.invalidRequestBody")
		return "", "", false
	}
	return emailNormalized, requestedAuthEmailLocale(c, body.Locale), true
}

func requestedAuthEmailLocale(c *gin.Context, requested string) string {
	locale := strings.TrimSpace(requested)
	if locale == "" && c != nil {
		locale = strings.TrimSpace(c.GetHeader("Accept-Language"))
	}
	if len(locale) > 128 {
		return ""
	}
	return locale
}

func (h *EmailAuthHandler) requireLocalAuthEnabled(c *gin.Context) bool {
	enabled, err := h.localAuthEnabled()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load authentication settings", "error.databaseError")
		return false
	}
	if !enabled {
		response.ErrorWithKey(c, http.StatusServiceUnavailable, "local authentication is not enabled", "error.authUnavailable")
		return false
	}
	peppers := h.peppers()
	if err := auth.ValidateCurrentPepper(peppers); err != nil {
		response.ErrorWithKey(c, http.StatusServiceUnavailable, "password authentication is unavailable", "error.authUnavailable")
		return false
	}
	return true
}

func (h *EmailAuthHandler) localAuthEnabled() (bool, error) {
	raw, err := h.repo.GetSystemConfig(localAuthEnabledConfigKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return raw == "true", nil
}

func (h *EmailAuthHandler) loadRegistrationSettings() (registrationSettings, error) {
	settings := registrationSettings{Enabled: true, Policy: auth.DomainPolicy{Mode: auth.DomainPolicyUnrestricted}}
	configs, err := h.repo.GetSystemConfigsByKeys([]string{
		registrationEnabledConfigKey,
		emailDomainModeConfigKey,
		emailDomainDomainsConfigKey,
		"registration_enabled",
	})
	if err != nil {
		return registrationSettings{}, err
	}
	if raw, ok := configs[registrationEnabledConfigKey]; ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return registrationSettings{}, errors.New("invalid registration enabled setting")
		}
		settings.Enabled = parsed
	} else if raw, ok := configs["registration_enabled"]; ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return registrationSettings{}, errors.New("invalid legacy registration enabled setting")
		}
		settings.Enabled = parsed
	}
	if raw, ok := configs[emailDomainModeConfigKey]; ok && strings.TrimSpace(raw) != "" {
		settings.Policy.Mode = strings.TrimSpace(raw)
	}
	if raw, ok := configs[emailDomainDomainsConfigKey]; ok && strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings.Policy.Domains); err != nil {
			return registrationSettings{}, errors.New("invalid email domain settings")
		}
	}
	return settings, nil
}

type configuredRewards struct {
	enabled           bool
	inviterCredits    model.Credit
	inviteeCredits    model.Credit
	registrationBonus model.Credit
}

func (h *EmailAuthHandler) registrationRewards() (configuredRewards, error) {
	configs, err := h.repo.GetSystemConfigsByKeys([]string{
		"referral_enabled",
		"referral_inviter_credits",
		"referral_invitee_credits",
		"registration_bonus_credits",
	})
	if err != nil {
		return configuredRewards{}, err
	}
	rewards := configuredRewards{enabled: configs["referral_enabled"] == "true"}
	if value, ok := parseNonNegativeCreditConfigForRuntime(configs["referral_inviter_credits"]); ok {
		rewards.inviterCredits = value
	}
	if value, ok := parseNonNegativeCreditConfigForRuntime(configs["referral_invitee_credits"]); ok {
		rewards.inviteeCredits = value
	}
	if value, ok := parseNonNegativeCreditConfigForRuntime(configs["registration_bonus_credits"]); ok {
		rewards.registrationBonus = value
	}
	return rewards, nil
}

func parseRegistrationTokenPayload(raw json.RawMessage) (registrationTokenPayload, error) {
	if len(raw) == 0 {
		return registrationTokenPayload{}, nil
	}
	var payload registrationTokenPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return registrationTokenPayload{}, err
	}
	payload.ReferralCode = strings.ToLower(strings.TrimSpace(payload.ReferralCode))
	if payload.ReferralCode != "" && !referralCodePattern.MatchString(payload.ReferralCode) {
		return registrationTokenPayload{}, errors.New("invalid registration token payload")
	}
	return payload, nil
}

func defaultEmailProfileName(email string) string {
	email = strings.TrimSpace(email)
	if localPart, _, ok := strings.Cut(email, "@"); ok && localPart != "" {
		return localPart
	}
	if email != "" {
		return email
	}
	return "User"
}

func (h *EmailAuthHandler) peppers() auth.PepperSet {
	if h.cfg == nil {
		return auth.PepperSet{}
	}
	return auth.PepperSet{
		CurrentID:  strings.TrimSpace(h.cfg.AuthPasswordPepperID),
		Current:    []byte(h.cfg.AuthPasswordPepper),
		PreviousID: strings.TrimSpace(h.cfg.AuthPreviousPepperID),
		Previous:   []byte(h.cfg.AuthPreviousPepper),
	}
}

func (h *EmailAuthHandler) allowAuthRequest(c *gin.Context, action, emailNormalized string) bool {
	since := time.Now().Add(-authRequestRateLimitWindow)
	emailCount, err := h.repo.CountRecentAuthSecurityEventsForEmail(action, emailNormalized, since)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to validate request rate", "error.databaseError")
		return false
	}
	if emailCount >= authRequestRateLimit {
		response.ErrorWithKey(c, http.StatusTooManyRequests, "too many authentication requests", "error.rateLimited")
		return false
	}
	ipHash := h.ipHash(c)
	if ipHash == "" {
		return true
	}
	ipCount, err := h.repo.CountRecentAuthSecurityEventsForIP(action, ipHash, since)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to validate request rate", "error.databaseError")
		return false
	}
	if ipCount >= authRequestRateLimit {
		response.ErrorWithKey(c, http.StatusTooManyRequests, "too many authentication requests", "error.rateLimited")
		return false
	}
	return true
}

func (h *EmailAuthHandler) recordSecurityEvent(c *gin.Context, userID *uint, action, outcome, emailNormalized string) {
	_ = h.repo.CreateAuthSecurityEvent(&model.AuthSecurityEvent{
		UserID:          userID,
		Action:          action,
		Outcome:         outcome,
		EmailNormalized: emailNormalized,
		IPHash:          h.ipHash(c),
		UserAgent:       truncateAuthUserAgent(c.GetHeader("User-Agent")),
	})
}

func (h *EmailAuthHandler) ipHash(c *gin.Context) string {
	if h.cfg == nil || strings.TrimSpace(h.cfg.SessionSecret) == "" {
		return ""
	}
	remoteAddress := strings.TrimSpace(c.Request.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddress); err == nil {
		remoteAddress = host
	}
	if remoteAddress == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(h.cfg.SessionSecret))
	_, _ = mac.Write([]byte(remoteAddress))
	return hex.EncodeToString(mac.Sum(nil))
}

func truncateAuthUserAgent(raw string) string {
	const maxRunes = 512
	if len([]rune(raw)) <= maxRunes {
		return raw
	}
	return string([]rune(raw)[:maxRunes])
}

func (h *EmailAuthHandler) setSessionCookie(c *gin.Context, token string, ttl time.Duration) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "hl6_session",
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		Secure:   h.isSecureCookie(c),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *EmailAuthHandler) clearSessionCookie(c *gin.Context) {
	for _, secure := range []bool{false, true} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "hl6_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Secure:   secure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func (h *EmailAuthHandler) isSecureCookie(c *gin.Context) bool {
	if h.cfg != nil && strings.HasPrefix(h.cfg.FrontendURL, "https://") {
		return true
	}
	return c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

func (h *EmailAuthHandler) frontendURL(c *gin.Context) string {
	state, err := h.urlResolver.Resolve(c)
	if err == nil && state.FrontendURL != "" {
		return state.FrontendURL
	}
	if h.cfg != nil && h.cfg.FrontendURL != "" {
		return h.cfg.FrontendURL
	}
	return "/"
}

func (h *EmailAuthHandler) setPublicAuthHeaders(c *gin.Context) {
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Cache-Control", "no-store")
}

func (h *EmailAuthHandler) accepted(c *gin.Context) {
	c.JSON(http.StatusAccepted, response.Response{Code: 0, Message: "accepted"})
}

func (h *EmailAuthHandler) invalidLogin(c *gin.Context) {
	response.ErrorWithKey(c, http.StatusUnauthorized, "invalid email or password", "error.invalidCredentials")
}
