package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strconv"
	"strings"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"
)

// EmailService 邮件发送服务，支持 SMTP 协议。
type EmailService struct {
	repo          *repository.Repository
	encryptionKey []byte
}

// NewEmailService 创建邮件服务实例。
func NewEmailService(repo *repository.Repository, encryptionKey []byte) *EmailService {
	return &EmailService{repo: repo, encryptionKey: encryptionKey}
}

// smtpConfig 从 SystemConfig 读取 SMTP 配置。
type smtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	FromName string
	FromAddr string
	UseTLS   bool
	Enabled  bool
}

// getSMTPConfig 从数据库读取 SMTP 配置。
func (s *EmailService) getSMTPConfig() (*smtpConfig, error) {
	keys := []string{
		"smtp_host", "smtp_port", "smtp_username", "smtp_password",
		"smtp_from_name", "smtp_from_addr", "smtp_use_tls", "smtp_enabled",
	}
	configs, err := s.repo.GetSystemConfigsByKeys(keys)
	if err != nil {
		return nil, err
	}

	cfg := &smtpConfig{
		Host:     configs["smtp_host"],
		Port:     587,
		Username: configs["smtp_username"],
		Password: crypto.DecryptOrPlaintext(configs["smtp_password"], s.encryptionKey),
		FromName: configs["smtp_from_name"],
		FromAddr: configs["smtp_from_addr"],
		UseTLS:   configs["smtp_use_tls"] == "true",
		Enabled:  configs["smtp_enabled"] == "true",
	}

	if port, err := strconv.Atoi(configs["smtp_port"]); err == nil && port > 0 {
		cfg.Port = port
	}

	return cfg, nil
}

// IsEnabled 检查邮件服务是否已启用。
func (s *EmailService) IsEnabled() bool {
	cfg, err := s.getSMTPConfig()
	if err != nil {
		return false
	}
	return cfg.Enabled && cfg.Host != ""
}

// SendBanNotification 发送封禁通知邮件。
func (s *EmailService) SendBanNotification(user *model.User, reason string) error {
	if !s.IsEnabled() {
		log.Printf("[email] SMTP not enabled, skip ban notification for user %d", user.ID)
		return nil
	}

	if user.Email == "" {
		log.Printf("[email] user %d has no email, skip ban notification", user.ID)
		return nil
	}

	// 获取站点名称
	siteName := "SubDomain"
	if name, err := s.repo.GetSystemConfig("site_name"); err == nil && name != "" {
		siteName = name
	}

	subject := fmt.Sprintf("[%s] 账号封禁通知", siteName)
	body := s.buildBanEmailBody(user, reason, siteName)

	// 创建邮件发送记录
	emailLog := &model.EmailLog{
		Recipient: user.Email,
		Subject:   subject,
		Body:      body,
		Status:    model.EmailStatusPending,
		UserID:    &user.ID,
		EmailType: "ban_notify",
	}
	if err := s.repo.CreateEmailLog(emailLog); err != nil {
		log.Printf("[email] failed to create email log: %v", err)
		return err
	}

	// 发送邮件
	cfg, _ := s.getSMTPConfig()
	err := s.sendSMTP(cfg, user.Email, subject, body)
	if err != nil {
		emailLog.Status = model.EmailStatusFailed
		emailLog.Error = err.Error()
		emailLog.RetryCount++
		_ = s.repo.UpdateEmailLog(emailLog)
		log.Printf("[email] failed to send ban notification to %s: %v", user.Email, err)
		return err
	}

	emailLog.Status = model.EmailStatusSent
	_ = s.repo.UpdateEmailLog(emailLog)
	log.Printf("[email] ban notification sent to %s", user.Email)
	return nil
}

// buildBanEmailBody 构建封禁通知邮件正文。
func (s *EmailService) buildBanEmailBody(user *model.User, reason, siteName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("尊敬的 %s：\n\n", user.Name))
	sb.WriteString(fmt.Sprintf("您的 %s 账号已被封禁。\n\n", siteName))
	sb.WriteString("封禁详情：\n")
	sb.WriteString(fmt.Sprintf("  - 封禁原因：%s\n", reason))
	if user.BannedAt != nil {
		sb.WriteString(fmt.Sprintf("  - 封禁时间：%s\n", user.BannedAt.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("\n")
	sb.WriteString("申诉途径：\n")
	sb.WriteString("  如果您认为此封禁有误，可以登录平台后在封禁页面提交申诉，管理员将会审核您的申诉请求。\n\n")
	sb.WriteString(fmt.Sprintf("此邮件由 %s 系统自动发送，请勿直接回复。\n", siteName))

	return sb.String()
}

// sendSMTP 通过 SMTP 发送邮件。
func (s *EmailService) sendSMTP(cfg *smtpConfig, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	from := cfg.FromAddr
	if from == "" {
		from = cfg.Username
	}

	// 构建邮件内容
	msg := strings.Builder{}
	msg.WriteString("From: ")
	if cfg.FromName != "" {
		msg.WriteString(fmt.Sprintf("%s <%s>", cfg.FromName, from))
	} else {
		msg.WriteString(from)
	}
	msg.WriteString("\r\n")
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	if cfg.UseTLS {
		return s.sendWithTLS(addr, auth, from, to, []byte(msg.String()))
	}

	// 非 TLS 模式（如本地开发或 25 端口）
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg.String()))
}

// sendWithTLS 使用 TLS 连接发送邮件（STARTTLS）。
func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	host := strings.Split(addr, ":")[0]

	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write body failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close body failed: %w", err)
	}
	return client.Quit()
}

// RetrySingleEmail 重试发送单封邮件。
func (s *EmailService) RetrySingleEmail(emailLog *model.EmailLog) error {
	cfg, err := s.getSMTPConfig()
	if err != nil {
		return err
	}

	err = s.sendSMTP(cfg, emailLog.Recipient, emailLog.Subject, emailLog.Body)
	if err != nil {
		emailLog.Status = model.EmailStatusFailed
		emailLog.Error = err.Error()
		emailLog.RetryCount++
	} else {
		emailLog.Status = model.EmailStatusSent
		emailLog.Error = ""
	}
	_ = s.repo.UpdateEmailLog(emailLog)
	return err
}

// RetryFailedEmails 重试发送失败的邮件。
func (s *EmailService) RetryFailedEmails(maxRetries int) error {
	logs, err := s.repo.ListFailedEmailLogs(maxRetries)
	if err != nil {
		return err
	}

	cfg, err := s.getSMTPConfig()
	if err != nil {
		return err
	}

	for _, emailLog := range logs {
		err := s.sendSMTP(cfg, emailLog.Recipient, emailLog.Subject, emailLog.Body)
		if err != nil {
			emailLog.Error = err.Error()
			emailLog.RetryCount++
		} else {
			emailLog.Status = model.EmailStatusSent
			emailLog.Error = ""
		}
		_ = s.repo.UpdateEmailLog(&emailLog)
	}
	return nil
}

// SendTestEmail 发送测试邮件。
func (s *EmailService) SendTestEmail(recipientEmail, siteName string) error {
	cfg, err := s.getSMTPConfig()
	if err != nil {
		return fmt.Errorf("SMTP config error: %w", err)
	}
	if !cfg.Enabled || cfg.Host == "" {
		return fmt.Errorf("SMTP is not enabled or not configured")
	}

	subject := fmt.Sprintf("[%s] SMTP 配置测试", siteName)
	body := "这是一封测试邮件，用于验证 SMTP 配置是否正确。如果您收到此邮件，说明配置成功。"

	return s.sendSMTP(cfg, recipientEmail, subject, body)
}
