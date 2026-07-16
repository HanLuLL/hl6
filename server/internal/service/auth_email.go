package service

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"hl6-server/internal/model"
)

var ErrAuthenticationEmailUnavailable = errors.New("authentication email delivery is unavailable")

// SendAuthenticationLink sends a one-time authentication link without
// persisting its raw URL or token in EmailLog. A new request creates a new
// token instead of retrying a stored secret-bearing message.
func (s *EmailService) SendAuthenticationLink(recipient, purpose, link string, userID *uint) error {
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(link) == "" {
		return errors.New("authentication email recipient or link is empty")
	}
	cfg, err := s.getSMTPConfig()
	if err != nil {
		return fmt.Errorf("load SMTP configuration: %w", err)
	}
	if !cfg.Enabled || strings.TrimSpace(cfg.Host) == "" {
		return ErrAuthenticationEmailUnavailable
	}

	siteName := s.getSiteName()
	subject, title, copy, err := authenticationEmailContent(purpose, siteName)
	if err != nil {
		return err
	}
	body := buildAuthenticationLinkEmailHTML(siteName, title, copy, link)
	log := &model.EmailLog{
		Recipient: recipient,
		Subject:   subject,
		Body:      "One-time authentication link intentionally omitted from logs.",
		Status:    model.EmailStatusPending,
		UserID:    userID,
		EmailType: "auth_" + purpose,
	}
	if err := s.repo.CreateEmailLog(log); err != nil {
		return fmt.Errorf("create authentication email log: %w", err)
	}

	if err := s.sendSMTP(cfg, recipient, subject, body); err != nil {
		log.Status = model.EmailStatusFailed
		log.Error = err.Error()
		log.RetryCount = 1
		_ = s.repo.UpdateEmailLog(log)
		return err
	}

	log.Status = model.EmailStatusSent
	return s.repo.UpdateEmailLog(log)
}

func authenticationEmailContent(purpose, siteName string) (subject, title, copy string, err error) {
	siteName = html.EscapeString(strings.TrimSpace(siteName))
	switch purpose {
	case model.AuthTokenPurposeRegistrationVerify:
		return fmt.Sprintf("[%s] Verify your email", siteName), "Verify your email", "Continue creating your account by setting a password. This link expires soon and can only be used once.", nil
	case model.AuthTokenPurposeAccountActivation:
		return fmt.Sprintf("[%s] Activate your account", siteName), "Activate your account", "Set a new password to activate your existing account. Your profile and managed resources remain unchanged.", nil
	case model.AuthTokenPurposePasswordReset:
		return fmt.Sprintf("[%s] Reset your password", siteName), "Reset your password", "Use this one-time link to choose a new password. If you did not request it, you can safely ignore this email.", nil
	default:
		return "", "", "", errors.New("unsupported authentication email purpose")
	}
}

func buildAuthenticationLinkEmailHTML(siteName, title, copy, link string) string {
	safeSiteName := html.EscapeString(siteName)
	safeTitle := html.EscapeString(title)
	safeCopy := html.EscapeString(copy)
	safeLink := html.EscapeString(link)
	content := fmt.Sprintf(`<p style="margin:0 0 20px;color:#374151;font-size:15px;line-height:1.7;">%s</p>
<p style="margin:0 0 24px;text-align:center;"><a href="%s" style="display:inline-block;background:#2563eb;color:#fff;text-decoration:none;padding:12px 20px;border-radius:6px;font-size:15px;font-weight:600;">Continue securely</a></p>
<p style="margin:0;color:#6b7280;font-size:13px;line-height:1.6;word-break:break-all;">If the button does not work, open this link: %s</p>`, safeCopy, safeLink, safeLink)
	return emailBaseHTML(safeSiteName, safeTitle, content)
}
