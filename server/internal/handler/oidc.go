package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
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
)

var (
	referralCodePattern       = regexp.MustCompile(`^[a-z]{5}$`)
	legacyReferralCodePattern = regexp.MustCompile(`^[0-9a-f]{16}$`)
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

	// 2. Exchange code for tokens
	tokenResp, err := h.exchangeCode(provider, runtimeState, code, callbackURL)
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

	idToken, err := jwt.Parse([]byte(idTokenStr),
		jwt.WithKeySet(keySet),
		jwt.WithIssuer(provider.Issuer),
		jwt.WithAudience(runtimeState.ClientID),
		jwt.WithAcceptableSkew(2*time.Minute),
	)
	if err != nil {
		log.Printf("invalid id_token: %v", err)
		c.String(http.StatusBadGateway, "authentication failed")
		return
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
	if name == "" {
		name, _ = claims["username"].(string)
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
			if bonus, err := strconv.ParseFloat(bonusStr, 64); err == nil && bonus > 0 {
				amount := model.CreditFromFloat(bonus)
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
		// Existing user — update info
		user.Email = email
		user.Name = name
		user.AvatarURL = picture
		if err := h.repo.UpdateUser(user); err != nil {
			c.String(http.StatusInternalServerError, "failed to update user profile")
			return
		}
	}

	// Banned users cannot create new sessions.
	if user.IsBanned {
		h.setSessionCookie(c, "hl6_session", "", -1, secureCookie)
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

func buildBannedRedirectURL(frontendBaseURL, reason string) string {
	targetURL, err := url.Parse(frontendBaseURL + "/")
	if err != nil {
		values := url.Values{"error": []string{"user_banned"}}
		trimmedReason := strings.TrimSpace(reason)
		if trimmedReason != "" {
			values.Set("reason", trimmedReason)
		}
		return frontendBaseURL + "/?" + values.Encode()
	}

	query := targetURL.Query()
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

func (h *OIDCHandler) exchangeCode(provider *oidc.ProviderConfig, runtimeState *OIDCRuntimeState, code, redirectURI string) (map[string]interface{}, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {runtimeState.ClientID},
		"client_secret": {runtimeState.ClientSecret},
	}

	resp, err := http.Post(provider.TokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return result, nil
}

func (h *OIDCHandler) issueSessionJWT(externalID string) (string, error) {
	key, err := jwk.FromRaw([]byte(h.cfg.SessionSecret))
	if err != nil {
		return "", err
	}

	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(externalID).
		Issuer("hl6").
		Audience([]string{"hl6"}).
		IssuedAt(now).
		Expiration(now.Add(7 * 24 * time.Hour)).
		Build()
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
	inviterCreditsFloat, err := strconv.ParseFloat(inviterCreditsStr, 64)
	if err != nil {
		inviterCreditsFloat = 0
	}
	inviteeCreditsFloat, err := strconv.ParseFloat(inviteeCreditsStr, 64)
	if err != nil {
		inviteeCreditsFloat = 0
	}
	inviterCredits := model.CreditFromFloat(inviterCreditsFloat)
	inviteeCredits := model.CreditFromFloat(inviteeCreditsFloat)

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
