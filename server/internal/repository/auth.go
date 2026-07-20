package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"hl6-server/internal/model"
	"hl6-server/internal/referral"
)

const firstLocalUserAdminLockKey int64 = 19490331

var (
	ErrAuthTokenUnavailable = errors.New("authentication token is unavailable")
	ErrCredentialNotFound   = errors.New("credential not found")
	ErrInvalidNewUserInput  = errors.New("invalid new user input")
)

// NewUserInput carries server-validated registration data.
type NewUserInput struct {
	Email                  string
	EmailNormalized        string
	Name                   string
	PasswordHash           string
	PasswordHashVersion    string
	ReferralCode           string
	ReferralEnabled        bool
	ReferralInviterCredits model.Credit
	ReferralInviteeCredits model.Credit
	RegistrationBonus      model.Credit
}

func (r *Repository) FindCredentialByEmail(email string) (*model.UserCredential, error) {
	var credential model.UserCredential
	err := r.DB.Where("email_normalized = ?", email).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *Repository) FindCredentialByUserID(userID uint) (*model.UserCredential, error) {
	var credential model.UserCredential
	err := r.DB.Where("user_id = ?", userID).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *Repository) FindCredentialsByUserIDs(userIDs []uint) (map[uint]*model.UserCredential, error) {
	if len(userIDs) == 0 {
		return make(map[uint]*model.UserCredential), nil
	}
	var credentials []model.UserCredential
	if err := r.DB.Where("user_id IN ?", userIDs).Find(&credentials).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]*model.UserCredential, len(credentials))
	for i := range credentials {
		result[credentials[i].UserID] = &credentials[i]
	}
	return result, nil
}

func (r *Repository) CreateAuthToken(token *model.AuthToken) error {
	if token == nil || strings.TrimSpace(token.Purpose) == "" || strings.TrimSpace(token.EmailNormalized) == "" || len(token.TokenHash) != 64 || token.ExpiresAt.IsZero() {
		return ErrInvalidNewUserInput
	}
	return r.DB.Create(token).Error
}

// ConsumeAuthToken atomically marks a non-expired token as used. A second
// consumer observes the same neutral unavailable error rather than token state.
func (r *Repository) ConsumeAuthToken(ctx context.Context, hash string, purpose string) (*model.AuthToken, error) {
	return r.consumeAuthToken(ctx, hash, []string{purpose})
}

// ConsumeAnyAuthToken atomically consumes one valid token from the supplied
// purpose set. The token itself, not a client-provided purpose field, remains
// authoritative for password completion.
func (r *Repository) ConsumeAnyAuthToken(ctx context.Context, hash string, purposes []string) (*model.AuthToken, error) {
	if len(purposes) == 0 {
		return nil, ErrAuthTokenUnavailable
	}
	return r.consumeAuthToken(ctx, hash, purposes)
}

func (r *Repository) consumeAuthToken(ctx context.Context, hash string, purposes []string) (*model.AuthToken, error) {
	if len(hash) != 64 {
		return nil, ErrAuthTokenUnavailable
	}
	for _, purpose := range purposes {
		if strings.TrimSpace(purpose) == "" {
			return nil, ErrAuthTokenUnavailable
		}
	}

	now := time.Now().UTC()
	var token model.AuthToken
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AuthToken{}).
			Where("token_hash = ? AND purpose IN ? AND consumed_at IS NULL AND expires_at > ?", hash, purposes, now).
			Update("consumed_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrAuthTokenUnavailable
		}
		return tx.Where("token_hash = ?", hash).First(&token).Error
	})
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// CountRecentAuthSecurityEventsForEmail counts one action's attempts for an
// account across all clients. It intentionally does not combine the email
// predicate with the IP predicate: both dimensions need independent limits.
func (r *Repository) CountRecentAuthSecurityEventsForEmail(action, emailNormalized string, since time.Time) (int64, error) {
	if strings.TrimSpace(emailNormalized) == "" {
		return 0, nil
	}
	return r.countRecentAuthSecurityEvents(action, since, "email_normalized = ?", emailNormalized)
}

// CountRecentAuthSecurityEventsForIP counts one action's attempts from a
// client network across all accounts. IP addresses are already keyed hashes.
func (r *Repository) CountRecentAuthSecurityEventsForIP(action, ipHash string, since time.Time) (int64, error) {
	if strings.TrimSpace(ipHash) == "" {
		return 0, nil
	}
	return r.countRecentAuthSecurityEvents(action, since, "ip_hash = ?", ipHash)
}

func (r *Repository) countRecentAuthSecurityEvents(action string, since time.Time, predicate string, value interface{}) (int64, error) {
	query := r.DB.Model(&model.AuthSecurityEvent{}).Where("action = ? AND created_at >= ?", action, since).Where(predicate, value)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) IncrementSessionVersion(userID uint) error {
	result := r.DB.Model(&model.UserCredential{}).
		Where("user_id = ?", userID).
		UpdateColumn("session_version", gorm.Expr("session_version + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCredentialNotFound
	}
	return nil
}

func (r *Repository) UpdateCredentialPassword(ctx context.Context, userID uint, passwordHash, version string) (*model.UserCredential, error) {
	return r.updateCredentialPassword(ctx, userID, passwordHash, version, nil)
}

// UpdateCredentialPasswordFromAuthToken changes a password only when the
// consumed activation or reset token was issued after the last password set.
// This keeps separately consumed sibling links from racing to overwrite each
// other after their Argon2id derivations complete.
func (r *Repository) UpdateCredentialPasswordFromAuthToken(ctx context.Context, token *model.AuthToken, passwordHash, version string) (*model.UserCredential, error) {
	if token == nil || token.ID == 0 || token.UserID == nil || (token.Purpose != model.AuthTokenPurposeAccountActivation && token.Purpose != model.AuthTokenPurposePasswordReset) {
		return nil, ErrAuthTokenUnavailable
	}
	return r.updateCredentialPassword(ctx, *token.UserID, passwordHash, version, token)
}

func (r *Repository) updateCredentialPassword(ctx context.Context, userID uint, passwordHash, version string, authToken *model.AuthToken) (*model.UserCredential, error) {
	if userID == 0 || strings.TrimSpace(passwordHash) == "" {
		return nil, ErrInvalidNewUserInput
	}

	now := time.Now().UTC()
	var credential model.UserCredential
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&credential).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCredentialNotFound
			}
			return err
		}
		if authToken != nil {
			var storedToken model.AuthToken
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND user_id = ? AND email_normalized = ? AND purpose IN ? AND consumed_at IS NOT NULL", authToken.ID, userID, authToken.EmailNormalized, []string{
					model.AuthTokenPurposeAccountActivation,
					model.AuthTokenPurposePasswordReset,
				}).
				First(&storedToken).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAuthTokenUnavailable
			}
			if err != nil {
				return err
			}
			if credential.EmailNormalized != storedToken.EmailNormalized || (credential.PasswordSetAt != nil && !credential.PasswordSetAt.Before(storedToken.CreatedAt)) {
				return ErrAuthTokenUnavailable
			}
		}
		updates := map[string]interface{}{
			"password_hash":          passwordHash,
			"password_hash_version":  version,
			"email_verified_at":      now,
			"password_set_at":        now,
			"activation_required_at": nil,
		}
		// 只有通过 auth token 设置密码时才递增 session_version（激活/重置密码需要使旧 session 失效）
		// rehash 场景不需要递增，避免其他设备 session 失效
		if authToken != nil {
			updates["session_version"] = gorm.Expr("session_version + 1")
		}
		if err := tx.Model(&model.UserCredential{}).Where("id = ?", credential.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AuthToken{}).
			Where("user_id = ? AND purpose IN ? AND consumed_at IS NULL", userID, []string{
				model.AuthTokenPurposeAccountActivation,
				model.AuthTokenPurposePasswordReset,
				model.AuthTokenPurposeRestoreChallenge,
			}).
			Update("consumed_at", now).Error; err != nil {
			return fmt.Errorf("invalidate outstanding authentication tokens: %w", err)
		}
		return tx.Where("id = ?", credential.ID).First(&credential).Error
	})
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *Repository) CreateAuthSecurityEvent(event *model.AuthSecurityEvent) error {
	if event == nil || strings.TrimSpace(event.Action) == "" || strings.TrimSpace(event.Outcome) == "" {
		return ErrInvalidNewUserInput
	}
	return r.DB.Create(event).Error
}

func (r *Repository) ListAuthSecurityEvents(page, perPage int) ([]model.AuthSecurityEvent, int64, error) {
	var events []model.AuthSecurityEvent
	var total int64
	query := r.DB.Model(&model.AuthSecurityEvent{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *Repository) CreateUserWithCredential(ctx context.Context, input NewUserInput) (*model.User, *model.UserCredential, error) {
	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.EmailNormalized) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.PasswordHash) == "" {
		return nil, nil, ErrInvalidNewUserInput
	}

	var user model.User
	var credential model.UserCredential
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var group model.UserGroup
		if err := tx.Where("is_default = ?", true).First(&group).Error; err != nil {
			return fmt.Errorf("load default user group: %w", err)
		}

		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", firstLocalUserAdminLockKey).Error; err != nil {
			return fmt.Errorf("lock first local user role: %w", err)
		}
		var userCount int64
		if err := tx.Model(&model.User{}).Count(&userCount).Error; err != nil {
			return fmt.Errorf("count users: %w", err)
		}

		user = model.User{
			Email:   strings.TrimSpace(input.Email),
			Name:    strings.TrimSpace(input.Name),
			Role:    "user",
			GroupID: &group.ID,
		}
		if userCount == 0 {
			user.Role = "admin"
		}
		if err := createUserWithUniqueReferralCode(tx, &user); err != nil {
			return err
		}
		if err := tx.Create(&model.CreditBalance{UserID: user.ID, Balance: 0}).Error; err != nil {
			return fmt.Errorf("create credit balance: %w", err)
		}

		now := time.Now().UTC()
		credential = model.UserCredential{
			UserID:              user.ID,
			EmailNormalized:     strings.TrimSpace(input.EmailNormalized),
			PasswordHash:        input.PasswordHash,
			PasswordHashVersion: strings.TrimSpace(input.PasswordHashVersion),
			EmailVerifiedAt:     &now,
			PasswordSetAt:       &now,
			SessionVersion:      1,
		}
		if err := tx.Create(&credential).Error; err != nil {
			return fmt.Errorf("create credential: %w", err)
		}

		if err := tx.Create(&model.AuditLog{
			UserID:     user.ID,
			Action:     "user_register",
			Resource:   "user",
			ResourceID: user.ID,
			Details:    registrationAuditDetails(user.Email),
		}).Error; err != nil {
			return fmt.Errorf("create registration audit: %w", err)
		}

		if input.RegistrationBonus > 0 {
			if err := r.GrantCredits(tx, user.ID, input.RegistrationBonus, "txn.registrationBonus", nil); err != nil {
				return fmt.Errorf("grant registration bonus: %w", err)
			}
		}
		if err := r.createReferralForNewUser(tx, &user, input); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &user, &credential, nil
}

func createUserWithUniqueReferralCode(tx *gorm.DB, user *model.User) error {
	for attempts := 0; attempts < 20; attempts++ {
		code, err := referral.GenerateCode(5)
		if err != nil {
			return fmt.Errorf("generate referral code: %w", err)
		}
		user.ReferralCode = code
		if err := tx.Create(user).Error; err != nil {
			if referral.IsCodeUniqueViolation(err) {
				continue
			}
			return err
		}
		return nil
	}
	return errors.New("create user: referral code collision limit reached")
}

func (r *Repository) createReferralForNewUser(tx *gorm.DB, user *model.User, input NewUserInput) error {
	if !input.ReferralEnabled || strings.TrimSpace(input.ReferralCode) == "" {
		return nil
	}

	var inviter model.User
	if err := tx.Where("referral_code = ?", strings.ToLower(strings.TrimSpace(input.ReferralCode))).First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("find referral inviter: %w", err)
	}
	if inviter.ID == user.ID {
		return nil
	}
	if err := tx.Create(&model.UserReferral{
		InviterID:      inviter.ID,
		InviteeID:      user.ID,
		InviterCredits: input.ReferralInviterCredits,
		InviteeCredits: input.ReferralInviteeCredits,
	}).Error; err != nil {
		return fmt.Errorf("create referral: %w", err)
	}
	if input.ReferralInviterCredits > 0 {
		if err := r.GrantCredits(tx, inviter.ID, input.ReferralInviterCredits, "txn.referralInviter", nil); err != nil {
			return fmt.Errorf("grant inviter referral credits: %w", err)
		}
	}
	if input.ReferralInviteeCredits > 0 {
		if err := r.GrantCredits(tx, user.ID, input.ReferralInviteeCredits, "txn.referralInvitee", nil); err != nil {
			return fmt.Errorf("grant invitee referral credits: %w", err)
		}
	}
	return nil
}

func registrationAuditDetails(email string) json.RawMessage {
	details, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		return nil
	}
	return details
}

// CreateUserSession 创建新的用户会话记录
func (r *Repository) CreateUserSession(session *model.UserSession) error {
	if session == nil || session.UserID == 0 || strings.TrimSpace(session.SessionJTI) == "" {
		return ErrInvalidNewUserInput
	}
	return r.DB.Create(session).Error
}

// FindUserSessionByJTI 通过 JTI 哈希查找会话
func (r *Repository) FindUserSessionByJTI(jtiHash string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.DB.Where("session_jti = ?", jtiHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListUserSessions 列出用户的所有活跃会话
func (r *Repository) ListUserSessions(userID uint) ([]model.UserSession, error) {
	var sessions []model.UserSession
	err := r.DB.Where("user_id = ? AND expires_at > ?", userID, time.Now().UTC()).
		Order("last_active_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// DeleteUserSession 删除指定会话（踢出设备）
func (r *Repository) DeleteUserSession(userID, sessionID uint) error {
	result := r.DB.Where("id = ? AND user_id = ?", sessionID, userID).Delete(&model.UserSession{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteUserSessionByJTI 通过 JTI 删除会话
func (r *Repository) DeleteUserSessionByJTI(userID uint, jtiHash string) error {
	result := r.DB.Where("user_id = ? AND session_jti = ?", userID, jtiHash).Delete(&model.UserSession{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteAllUserSessions 删除用户的所有会话（登出所有设备）
func (r *Repository) DeleteAllUserSessions(userID uint) error {
	return r.DB.Where("user_id = ?", userID).Delete(&model.UserSession{}).Error
}

// UpdateUserSessionLastActive 更新会话最后活跃时间
func (r *Repository) UpdateUserSessionLastActive(jtiHash string) error {
	return r.DB.Model(&model.UserSession{}).
		Where("session_jti = ? AND expires_at > ?", jtiHash, time.Now().UTC()).
		Update("last_active_at", time.Now().UTC()).Error
}

// CleanupExpiredSessions 清理过期的会话记录
func (r *Repository) CleanupExpiredSessions() (int64, error) {
	result := r.DB.Where("expires_at < ?", time.Now().UTC()).Delete(&model.UserSession{})
	return result.RowsAffected, result.Error
}
