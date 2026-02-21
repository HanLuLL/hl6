package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type OIDCHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewOIDCHandler(repo *repository.Repository, cfg *config.Config) *OIDCHandler {
	return &OIDCHandler{repo: repo, cfg: cfg}
}

func (h *OIDCHandler) Login(c *gin.Context) {
	state := generateRandomState()

	// Store state in httpOnly cookie
	secure := strings.HasPrefix(h.cfg.FrontendURL, "https")
	c.SetCookie("hl6_state", state, 900, "/", "", secure, true) // 15 min TTL

	redirectURI := fmt.Sprintf("%s/api/v1/auth/callback", h.cfg.FrontendURL)
	authURL := fmt.Sprintf("%s/oidc/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		h.cfg.LogtoEndpoint,
		url.QueryEscape(h.cfg.LogtoAppID),
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
	secure := strings.HasPrefix(h.cfg.FrontendURL, "https")
	c.SetCookie("hl6_state", "", -1, "/", "", secure, true)

	// 2. Exchange code for tokens
	redirectURI := fmt.Sprintf("%s/api/v1/auth/callback", h.cfg.FrontendURL)
	tokenResp, err := h.exchangeCode(code, redirectURI)
	if err != nil {
		c.String(http.StatusBadGateway, "token exchange failed: %v", err)
		return
	}

	// 3. Parse ID token to get user info
	idTokenStr, ok := tokenResp["id_token"].(string)
	if !ok {
		c.String(http.StatusBadGateway, "no id_token in response")
		return
	}

	// Fetch JWKS for verification
	jwksURL := h.cfg.LogtoEndpoint + "/oidc/jwks"
	keySet, err := jwk.Fetch(c.Request.Context(), jwksURL)
	if err != nil {
		c.String(http.StatusBadGateway, "failed to fetch JWKS: %v", err)
		return
	}

	idToken, err := jwt.Parse([]byte(idTokenStr),
		jwt.WithKeySet(keySet),
		jwt.WithIssuer(h.cfg.LogtoEndpoint+"/oidc"),
	)
	if err != nil {
		c.String(http.StatusBadGateway, "invalid id_token: %v", err)
		return
	}

	sub := idToken.Subject()
	claims := idToken.PrivateClaims()
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	if name == "" {
		name, _ = claims["username"].(string)
	}

	// 4. Find or create user
	user, err := h.repo.FindUserByLogtoID(sub)
	if err != nil {
		// New user — create in a single transaction
		user = &model.User{
			LogtoID:   sub,
			Email:     email,
			Name:      name,
			AvatarURL: picture,
			Role:      "user",
		}
		if h.cfg.IsAdminEmail(email) {
			user.Role = "admin"
		}

		// Assign default group
		if defaultGroup, err := h.repo.GetDefaultUserGroup(); err == nil {
			user.GroupID = &defaultGroup.ID
		}

		tx := h.repo.DB.Begin()

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
				if err := h.repo.GrantCredits(tx, user.ID, amount, "txn.registrationBonus", nil); err == nil {
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
		user, _ = h.repo.FindUserByLogtoID(sub)
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
	sessionToken, err := h.issueSessionJWT(user.LogtoID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to issue session")
		return
	}

	// 6. Set session cookie
	maxAge := 7 * 24 * 60 * 60 // 7 days
	c.SetCookie("hl6_session", sessionToken, maxAge, "/", "", secure, true)

	// 7. Redirect to dashboard
	c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/dashboard")
}

func (h *OIDCHandler) Logout(c *gin.Context) {
	secure := strings.HasPrefix(h.cfg.FrontendURL, "https")
	c.SetCookie("hl6_session", "", -1, "/", "", secure, true)

	logoutURL := fmt.Sprintf("%s/oidc/session/end?post_logout_redirect_uri=%s",
		h.cfg.LogtoEndpoint,
		url.QueryEscape(h.cfg.FrontendURL),
	)
	response.OK(c, gin.H{"logout_url": logoutURL})
}

func (h *OIDCHandler) exchangeCode(code, redirectURI string) (map[string]interface{}, error) {
	tokenURL := h.cfg.LogtoEndpoint + "/oidc/token"
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {h.cfg.LogtoAppID},
		"client_secret": {h.cfg.LogtoAppSecret},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
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

func (h *OIDCHandler) issueSessionJWT(logtoID string) (string, error) {
	key, err := jwk.FromRaw([]byte(h.cfg.SessionSecret))
	if err != nil {
		return "", err
	}

	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(logtoID).
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
