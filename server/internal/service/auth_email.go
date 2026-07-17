package service

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"hl6-server/internal/model"
)

var ErrAuthenticationEmailUnavailable = errors.New("authentication email delivery is unavailable")

type authenticationEmailText struct {
	Subject  string
	Title    string
	Copy     string
	Action   string
	Fallback string
}

type authenticationEmailLocale struct {
	HTMLLanguage string
	Footer       string
	Messages     map[string]authenticationEmailText
}

type localizedAuthenticationEmail struct {
	authenticationEmailText
	HTMLLanguage string
	Footer       string
}

var authenticationEmailTranslations = map[string]authenticationEmailLocale{
	"en": {
		HTMLLanguage: "en",
		Footer:       "This email was sent automatically by %s. Please do not reply.",
		Messages: map[string]authenticationEmailText{
			model.AuthTokenPurposeRegistrationVerify: {Subject: "[%s] Verify your email", Title: "Verify your email", Copy: "Continue creating your account by setting a password. This link expires in 30 minutes and can only be used once.", Action: "Verify and set password", Fallback: "If the button does not work, open this link:"},
			model.AuthTokenPurposeAccountActivation:  {Subject: "[%s] Activate your account", Title: "Activate your account", Copy: "Set a new password to activate your existing account. Your profile and managed resources remain unchanged.", Action: "Activate and set password", Fallback: "If the button does not work, open this link:"},
			model.AuthTokenPurposePasswordReset:      {Subject: "[%s] Reset your password", Title: "Reset your password", Copy: "Use this one-time link to choose a new password. If you did not request it, you can safely ignore this email.", Action: "Reset password", Fallback: "If the button does not work, open this link:"},
		},
	},
	"zh": {
		HTMLLanguage: "zh-CN",
		Footer:       "此邮件由 %s 系统自动发送，请勿直接回复。",
		Messages: map[string]authenticationEmailText{
			model.AuthTokenPurposeRegistrationVerify: {Subject: "[%s] 验证邮箱", Title: "验证你的邮箱", Copy: "请点击下方按钮继续创建账户并设置密码。链接将在 30 分钟后过期，且只能使用一次。", Action: "验证并设置密码", Fallback: "如果按钮无法打开，请复制以下链接到浏览器："},
			model.AuthTokenPurposeAccountActivation:  {Subject: "[%s] 激活账户", Title: "激活你的账户", Copy: "请设置新密码以激活现有账户。你的个人资料和已管理资源不会发生变化。", Action: "激活并设置密码", Fallback: "如果按钮无法打开，请复制以下链接到浏览器："},
			model.AuthTokenPurposePasswordReset:      {Subject: "[%s] 重置密码", Title: "重置你的密码", Copy: "请使用此一次性链接设置新密码。如果不是你发起的请求，可以安全忽略此邮件。", Action: "重置密码", Fallback: "如果按钮无法打开，请复制以下链接到浏览器："},
		},
	},
	"zh-Hant": {
		HTMLLanguage: "zh-TW",
		Footer:       "此郵件由 %s 系統自動寄送，請勿直接回覆。",
		Messages: map[string]authenticationEmailText{
			model.AuthTokenPurposeRegistrationVerify: {Subject: "[%s] 驗證電子郵件", Title: "驗證你的電子郵件", Copy: "請點擊下方按鈕繼續建立帳戶並設定密碼。連結將於 30 分鐘後失效，且只能使用一次。", Action: "驗證並設定密碼", Fallback: "如果按鈕無法開啟，請將以下連結複製到瀏覽器："},
			model.AuthTokenPurposeAccountActivation:  {Subject: "[%s] 啟用帳戶", Title: "啟用你的帳戶", Copy: "請設定新密碼以啟用既有帳戶。你的個人資料和已管理資源不會發生變更。", Action: "啟用並設定密碼", Fallback: "如果按鈕無法開啟，請將以下連結複製到瀏覽器："},
			model.AuthTokenPurposePasswordReset:      {Subject: "[%s] 重設密碼", Title: "重設你的密碼", Copy: "請使用此一次性連結設定新密碼。如果不是你提出的請求，可以安全忽略此郵件。", Action: "重設密碼", Fallback: "如果按鈕無法開啟，請將以下連結複製到瀏覽器："},
		},
	},
	"es": {
		HTMLLanguage: "es",
		Footer:       "Este correo fue enviado automáticamente por %s. No respondas a este mensaje.",
		Messages: map[string]authenticationEmailText{
			model.AuthTokenPurposeRegistrationVerify: {Subject: "[%s] Verifica tu correo", Title: "Verifica tu correo", Copy: "Continúa creando tu cuenta y establece una contraseña. Este enlace caduca en 30 minutos y solo puede usarse una vez.", Action: "Verificar y establecer contraseña", Fallback: "Si el botón no funciona, abre este enlace:"},
			model.AuthTokenPurposeAccountActivation:  {Subject: "[%s] Activa tu cuenta", Title: "Activa tu cuenta", Copy: "Establece una nueva contraseña para activar tu cuenta existente. Tu perfil y los recursos administrados no cambiarán.", Action: "Activar y establecer contraseña", Fallback: "Si el botón no funciona, abre este enlace:"},
			model.AuthTokenPurposePasswordReset:      {Subject: "[%s] Restablece tu contraseña", Title: "Restablece tu contraseña", Copy: "Usa este enlace de un solo uso para elegir una nueva contraseña. Si no lo solicitaste, puedes ignorar este mensaje.", Action: "Restablecer contraseña", Fallback: "Si el botón no funciona, abre este enlace:"},
		},
	},
	"ru": {
		HTMLLanguage: "ru",
		Footer:       "Это письмо отправлено системой %s автоматически. Не отвечайте на него.",
		Messages: map[string]authenticationEmailText{
			model.AuthTokenPurposeRegistrationVerify: {Subject: "[%s] Подтвердите адрес электронной почты", Title: "Подтвердите адрес электронной почты", Copy: "Продолжите создание учётной записи и задайте пароль. Ссылка действует 30 минут и может быть использована только один раз.", Action: "Подтвердить и задать пароль", Fallback: "Если кнопка не работает, откройте эту ссылку:"},
			model.AuthTokenPurposeAccountActivation:  {Subject: "[%s] Активируйте учётную запись", Title: "Активируйте учётную запись", Copy: "Задайте новый пароль для активации существующей учётной записи. Профиль и управляемые ресурсы останутся без изменений.", Action: "Активировать и задать пароль", Fallback: "Если кнопка не работает, откройте эту ссылку:"},
			model.AuthTokenPurposePasswordReset:      {Subject: "[%s] Сбросьте пароль", Title: "Сбросьте пароль", Copy: "Используйте эту одноразовую ссылку, чтобы выбрать новый пароль. Если вы не запрашивали сброс, просто проигнорируйте письмо.", Action: "Сбросить пароль", Fallback: "Если кнопка не работает, откройте эту ссылку:"},
		},
	},
	"ja": {
		HTMLLanguage: "ja",
		Footer:       "%s から自動送信されたメールです。返信しないでください。",
		Messages: map[string]authenticationEmailText{
			model.AuthTokenPurposeRegistrationVerify: {Subject: "[%s] メールアドレスを確認してください", Title: "メールアドレスを確認してください", Copy: "アカウント作成を続けてパスワードを設定してください。このリンクは30分後に期限切れとなり、一度だけ使用できます。", Action: "確認してパスワードを設定", Fallback: "ボタンが動作しない場合は、次のリンクを開いてください："},
			model.AuthTokenPurposeAccountActivation:  {Subject: "[%s] アカウントを有効化してください", Title: "アカウントを有効化してください", Copy: "既存のアカウントを有効化するため、新しいパスワードを設定してください。プロフィールと管理中のリソースは変更されません。", Action: "有効化してパスワードを設定", Fallback: "ボタンが動作しない場合は、次のリンクを開いてください："},
			model.AuthTokenPurposePasswordReset:      {Subject: "[%s] パスワードをリセットしてください", Title: "パスワードをリセットしてください", Copy: "この一度だけ使用できるリンクから新しいパスワードを設定してください。心当たりがない場合は、このメールを無視してください。", Action: "パスワードをリセット", Fallback: "ボタンが動作しない場合は、次のリンクを開いてください："},
		},
	},
}

// SendAuthenticationLink sends a one-time authentication link without
// persisting its raw URL or token in EmailLog. A new request creates a new
// token instead of retrying a stored secret-bearing message.
func (s *EmailService) SendAuthenticationLink(recipient, purpose, link, locale string, userID *uint) error {
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
	content, err := authenticationEmailContent(purpose, siteName, locale)
	if err != nil {
		return err
	}
	body := buildAuthenticationLinkEmailHTML(siteName, content, link)
	log := &model.EmailLog{
		Recipient: recipient,
		Subject:   content.Subject,
		Body:      "One-time authentication link intentionally omitted from logs.",
		Status:    model.EmailStatusPending,
		UserID:    userID,
		EmailType: "auth_" + purpose,
	}
	if err := s.repo.CreateEmailLog(log); err != nil {
		return fmt.Errorf("create authentication email log: %w", err)
	}

	if err := s.sendSMTP(cfg, recipient, content.Subject, body); err != nil {
		log.Status = model.EmailStatusFailed
		log.Error = err.Error()
		log.RetryCount = 1
		_ = s.repo.UpdateEmailLog(log)
		return err
	}

	log.Status = model.EmailStatusSent
	return s.repo.UpdateEmailLog(log)
}

func authenticationEmailContent(purpose, siteName, locale string) (localizedAuthenticationEmail, error) {
	translations := authenticationEmailTranslations[normalizeAuthenticationEmailLocale(locale)]
	content, ok := translations.Messages[purpose]
	if !ok {
		return localizedAuthenticationEmail{}, errors.New("unsupported authentication email purpose")
	}
	siteName = strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(siteName))
	content.Subject = fmt.Sprintf(content.Subject, siteName)
	return localizedAuthenticationEmail{
		authenticationEmailText: content,
		HTMLLanguage:            translations.HTMLLanguage,
		Footer:                  fmt.Sprintf(translations.Footer, siteName),
	}, nil
}

func normalizeAuthenticationEmailLocale(raw string) string {
	for _, candidate := range strings.Split(raw, ",") {
		candidate = strings.TrimSpace(strings.SplitN(candidate, ";", 2)[0])
		candidate = strings.ToLower(strings.ReplaceAll(candidate, "_", "-"))
		switch {
		case strings.HasPrefix(candidate, "zh-hant") || strings.HasPrefix(candidate, "zh-tw") || strings.HasPrefix(candidate, "zh-hk") || strings.HasPrefix(candidate, "zh-mo"):
			return "zh-Hant"
		case strings.HasPrefix(candidate, "zh"):
			return "zh"
		case strings.HasPrefix(candidate, "es"):
			return "es"
		case strings.HasPrefix(candidate, "ru"):
			return "ru"
		case strings.HasPrefix(candidate, "ja"):
			return "ja"
		case strings.HasPrefix(candidate, "en"):
			return "en"
		}
	}
	return "en"
}

func buildAuthenticationLinkEmailHTML(siteName string, text localizedAuthenticationEmail, link string) string {
	safeSiteName := html.EscapeString(siteName)
	safeTitle := html.EscapeString(text.Title)
	safeCopy := html.EscapeString(text.Copy)
	safeAction := html.EscapeString(text.Action)
	safeFallback := html.EscapeString(text.Fallback)
	safeLink := html.EscapeString(link)
	content := fmt.Sprintf(`<p style="margin:0 0 20px;color:#374151;font-size:15px;line-height:1.7;">%s</p>
<p style="margin:0 0 24px;text-align:center;"><a href="%s" style="display:inline-block;background:#2563eb;color:#fff;text-decoration:none;padding:12px 20px;border-radius:6px;font-size:15px;font-weight:600;">%s</a></p>
<p style="margin:0;color:#6b7280;font-size:13px;line-height:1.6;word-break:break-all;">%s %s</p>`, safeCopy, safeLink, safeAction, safeFallback, safeLink)
	footer := html.EscapeString(text.Footer)
	return emailBaseHTMLLocalized(safeSiteName, safeTitle, content, text.HTMLLanguage, footer)
}
