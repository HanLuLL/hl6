package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type OIDCHandler struct {
	repo     *repository.Repository
	cfg      *config.Config
	provider *oidc.ProviderConfig
}

func NewOIDCHandler(repo *repository.Repository, cfg *config.Config, provider *oidc.ProviderConfig) *OIDCHandler {
	return &OIDCHandler{repo: repo, cfg: cfg, provider: provider}
}

func (h *OIDCHandler) callbackURL() string {
	base := strings.TrimRight(h.cfg.BackendURL, "/")
	return base + "/api/v1/auth/callback"
}

// setSessionCookie sets a cookie with full attributes including SameSite=Lax.
func (h *OIDCHandler) setSessionCookie(c *gin.Context, name, value string, maxAge int) {
	secure := strings.HasPrefix(h.cfg.FrontendURL, "https")
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
	state := generateRandomState()

	// Store state in httpOnly cookie
	h.setSessionCookie(c, "hl6_state", state, 900) // 15 min TTL

	redirectURI := h.callbackURL()
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		h.provider.AuthorizationEndpoint,
		url.QueryEscape(h.cfg.OIDCClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape("openid email profile"),
		url.QueryEscape(state),
	)

	c.Redirect(http.StatusFound, authURL)
}

func (h *OIDCHandler) Callback(c *gin.Context) {
	// 1. Verify state
	code := c.Query("code")
	state := c.Query("state")
	cookieState, err := c.Cookie("hl6_state")
	if err != nil || cookieState != state || state == "" {
		c.String(http.StatusBadRequest, "invalid state")
		return
	}

	// Clear state cookie
	h.setSessionCookie(c, "hl6_state", "", -1)

	// 2. Exchange code for tokens
	redirectURI := h.callbackURL()
	tokenResp, err := h.exchangeCode(code, redirectURI)
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
	keySet, err := jwk.Fetch(c.Request.Context(), h.provider.JwksURI)
	if err != nil {
		log.Printf("failed to fetch JWKS: %v", err)
		c.String(http.StatusBadGateway, "authentication failed")
		return
	}

	idToken, err := jwt.Parse([]byte(idTokenStr),
		jwt.WithKeySet(keySet),
		jwt.WithIssuer(h.provider.Issuer),
		jwt.WithAudience(h.cfg.OIDCClientID),
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
		if azp == "" || azp != h.cfg.OIDCClientID {
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
		// New user — create in a single transaction
		user = &model.User{
			ExternalID: sub,
			Email:      email,
			Name:       name,
			AvatarURL:  picture,
			Role:       "user",
		}
		if h.cfg.IsAdminEmail(email) {
			user.Role = "admin"
		}

		// Assign default group
		if defaultGroup, err := h.repo.GetDefaultUserGroup(); err == nil {
			user.GroupID = &defaultGroup.ID
		}

		tx := h.repo.GetDB().Begin()

		if err := tx.Create(user).Error; err != nil {
			tx.Rollback()
			c.String(http.StatusInternalServerError, "failed to create user")
			return
		}

		// Create credit balance
		tx.Create(&model.CreditBalance{UserID: user.ID, Balance: 0})

		// Audit log for registration
		regDetails, _ := json.Marshal(map[string]string{"email": email})
		tx.Create(&model.AuditLog{
			UserID:     user.ID,
			Action:     "user_register",
			Resource:   "user",
			ResourceID: user.ID,
			Details:    regDetails,
		})

		// Grant registration bonus
		if bonusStr, err := h.repo.GetSystemConfig("registration_bonus_credits"); err == nil {
			if bonus, err := strconv.ParseFloat(bonusStr, 64); err == nil && bonus > 0 {
				amount := model.CreditFromFloat(bonus)
				if err := h.repo.GrantCredits(tx, user.ID, amount, "txn.registrationBonus", nil); err != nil {
					log.Printf("failed to grant registration bonus for user %d: %v", user.ID, err)
				} else {
					bonusDetails, _ := json.Marshal(map[string]interface{}{"amount": amount})
					tx.Create(&model.AuditLog{
						UserID:     user.ID,
						Action:     "user_registration_bonus",
						Resource:   "credit",
						ResourceID: user.ID,
						Details:    bonusDetails,
					})
				}
			}
		}

		tx.Commit()

		// Reload user with group
		user, _ = h.repo.FindUserByExternalID(sub)
	} else {
		// Existing user — update info
		user.Email = email
		user.Name = name
		user.AvatarURL = picture
		// Bidirectional admin sync
		if h.cfg.IsAdminEmail(email) {
			user.Role = "admin"
		} else {
			user.Role = "user"
		}
		h.repo.UpdateUser(user)
	}

	// 5. Issue session JWT
	sessionToken, err := h.issueSessionJWT(user.ExternalID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to issue session")
		return
	}

	// 6. Set session cookie
	maxAge := 7 * 24 * 60 * 60 // 7 days
	h.setSessionCookie(c, "hl6_session", sessionToken, maxAge)

	// 7. Redirect to dashboard
	c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/dashboard")
}

func (h *OIDCHandler) Logout(c *gin.Context) {
	h.setSessionCookie(c, "hl6_session", "", -1)

	if h.provider.EndSessionEndpoint != "" {
		logoutURL := fmt.Sprintf("%s?post_logout_redirect_uri=%s",
			h.provider.EndSessionEndpoint,
			url.QueryEscape(h.cfg.FrontendURL),
		)
		response.OK(c, gin.H{"logout_url": logoutURL})
	} else {
		response.OK(c, gin.H{"logout_url": h.cfg.FrontendURL})
	}
}

func (h *OIDCHandler) exchangeCode(code, redirectURI string) (map[string]interface{}, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {h.cfg.OIDCClientID},
		"client_secret": {h.cfg.OIDCClientSecret},
	}

	resp, err := http.Post(h.provider.TokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
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

func generateRandomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
