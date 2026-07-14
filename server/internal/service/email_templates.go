package service

import (
	"fmt"
	"html"
	"strings"
)

// emailBaseHTML 构建邮件 HTML 外壳，所有邮件模板共用此基础结构。
// 支持 Gmail、Outlook、Apple Mail 等主流客户端，内联样式确保兼容性。
func emailBaseHTML(siteName, title, contentHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s</title>
</head>
<body style="margin:0;padding:0;background-color:#f0f2f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f0f2f5;padding:24px 0;">
		<tr>
			<td align="center">
				<table width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background-color:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
					<!-- 顶部品牌色条 -->
					<tr>
						<td style="background:linear-gradient(135deg,#6366f1 0%%,#8b5cf6 100%%);padding:32px 40px;text-align:center;">
							<h1 style="margin:0;color:#ffffff;font-size:24px;font-weight:700;letter-spacing:0.5px;">%s</h1>
						</td>
					</tr>
					<!-- 标题 -->
					<tr>
						<td style="padding:32px 40px 0 40px;">
							<h2 style="margin:0;color:#1a1a2e;font-size:20px;font-weight:600;">%s</h2>
						</td>
					</tr>
					<!-- 正文内容 -->
					<tr>
						<td style="padding:24px 40px 32px 40px;">
							%s
						</td>
					</tr>
					<!-- 底部版权 -->
					<tr>
						<td style="padding:24px 40px;background-color:#f8f9fa;border-top:1px solid #e9ecef;">
							<p style="margin:0;color:#6c757d;font-size:13px;line-height:1.6;text-align:center;">
								此邮件由 %s 系统自动发送，请勿直接回复。<br/>
								&copy; 2026 %s. All rights reserved.
							</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, title, siteName, title, contentHTML, siteName, siteName)
}

// buildBanEmailHTML 构建封禁通知邮件的 HTML 内容。
// 包含封禁原因、封禁时间和申诉途径，不包含 SubDomain 相关信息。
func buildBanEmailHTML(userName, reason, bannedAt, bannedUntil, siteName string) string {
	var sb strings.Builder
	userName = html.EscapeString(userName)
	reason = html.EscapeString(reason)
	siteName = html.EscapeString(siteName)

	sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 16px 0;color:#374151;font-size:15px;line-height:1.8;">尊敬的 <strong>%s</strong>：</p>`, userName))
	sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 24px 0;color:#374151;font-size:15px;line-height:1.8;">您的 <strong>%s</strong> 账号已被封禁。请仔细阅读以下封禁详情：</p>`, siteName))

	// 封禁详情卡片
	sb.WriteString(`<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#fef2f2;border:1px solid #fecaca;border-radius:8px;margin-bottom:24px;">`)
	sb.WriteString(fmt.Sprintf(`<tr><td style="padding:16px 20px;color:#991b1b;font-size:14px;font-weight:600;padding-bottom:8px;">封禁详情</td></tr>`))
	sb.WriteString(fmt.Sprintf(`<tr><td style="padding:0 20px 8px 20px;color:#374151;font-size:14px;line-height:1.8;">封禁原因：<strong>%s</strong></td></tr>`, reason))
	if bannedAt != "" {
		sb.WriteString(fmt.Sprintf(`<tr><td style="padding:0 20px 16px 20px;color:#374151;font-size:14px;line-height:1.8;">封禁时间：%s</td></tr>`, bannedAt))
	} else {
		sb.WriteString(`<tr><td style="padding:0 20px 16px 20px;color:#374151;font-size:14px;line-height:1.8;">封禁时间：—</td></tr>`)
	}
	sb.WriteString(`</table>`)
	if bannedUntil != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 24px 0;color:#374151;font-size:14px;line-height:1.8;">&#39044;&#35745;&#35299;&#23553;&#26102;&#38388;: <strong>%s</strong></p>`, html.EscapeString(bannedUntil)))
	} else {
		sb.WriteString(`<p style="margin:0 0 24px 0;color:#374151;font-size:14px;line-height:1.8;">&#39044;&#35745;&#35299;&#23553;&#26102;&#38388;: &#38656;&#31649;&#29702;&#21592;&#23457;&#26680;&#21518;&#35299;&#38500;</p>`)
	}

	// 申诉途径
	sb.WriteString(`<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#eff6ff;border:1px solid #bfdbfe;border-radius:8px;margin-bottom:24px;">`)
	sb.WriteString(`<tr><td style="padding:16px 20px;color:#1e40af;font-size:14px;font-weight:600;padding-bottom:8px;">申诉途径</td></tr>`)
	sb.WriteString(`<tr><td style="padding:0 20px 16px 20px;color:#374151;font-size:14px;line-height:1.8;">如果您认为此封禁有误，可以登录平台后在封禁页面提交申诉，管理员将会审核您的申诉请求。</td></tr>`)
	sb.WriteString(`</table>`)

	return emailBaseHTML(siteName, "账号封禁通知", sb.String())
}

// buildUnbanEmailHTML 构建解封通知邮件的 HTML 内容。
// 包含解封时间、账号状态说明及后续使用建议。
func buildUnbanEmailHTML(userName, siteName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 16px 0;color:#374151;font-size:15px;line-height:1.8;">尊敬的 <strong>%s</strong>：</p>`, userName))
	sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 24px 0;color:#374151;font-size:15px;line-height:1.8;">好消息！您的 <strong>%s</strong> 账号已被解封，您可以正常登录并使用平台服务了。</p>`, siteName))

	// 账号状态卡片
	sb.WriteString(`<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f0fdf4;border:1px solid #bbf7d0;border-radius:8px;margin-bottom:24px;">`)
	sb.WriteString(`<tr><td style="padding:16px 20px;color:#15803d;font-size:14px;font-weight:600;padding-bottom:8px;">账号状态</td></tr>`)
	sb.WriteString(`<tr><td style="padding:0 20px 16px 20px;color:#374151;font-size:14px;line-height:1.8;">当前状态：<strong style="color:#15803d;">已解封</strong></td></tr>`)
	sb.WriteString(`</table>`)

	// 后续使用建议
	sb.WriteString(`<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#eff6ff;border:1px solid #bfdbfe;border-radius:8px;margin-bottom:24px;">`)
	sb.WriteString(`<tr><td style="padding:16px 20px;color:#1e40af;font-size:14px;font-weight:600;padding-bottom:8px;">后续使用建议</td></tr>`)
	sb.WriteString(`<tr><td style="padding:0 20px 16px 20px;color:#374151;font-size:14px;line-height:1.8;">`)
	sb.WriteString(`<ul style="margin:0;padding-left:20px;">`)
	sb.WriteString(`<li style="margin-bottom:6px;">请遵守平台使用条款，避免再次违规</li>`)
	sb.WriteString(`<li style="margin-bottom:6px;">及时检查您的子域名和 DNS 记录状态</li>`)
	sb.WriteString(`<li style="margin-bottom:6px;">如有任何疑问，可通过平台反馈渠道联系管理员</li>`)
	sb.WriteString(`</ul>`)
	sb.WriteString(`</td></tr>`)
	sb.WriteString(`</table>`)

	return emailBaseHTML(siteName, "账号解封通知", sb.String())
}

// buildTestEmailHTML 构建测试邮件的 HTML 内容。
func buildTestEmailHTML(siteName string) string {
	content := fmt.Sprintf(`<p style="margin:0 0 16px 0;color:#374151;font-size:15px;line-height:1.8;">您好！</p>
<p style="margin:0 0 24px 0;color:#374151;font-size:15px;line-height:1.8;">这是一封测试邮件，用于验证 <strong>%s</strong> 的 SMTP 配置是否正确。如果您收到此邮件，说明配置成功。</p>
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f0fdf4;border:1px solid #bbf7d0;border-radius:8px;margin-bottom:24px;">
	<tr><td style="padding:16px 20px;color:#15803d;font-size:14px;font-weight:600;">SMTP 配置验证结果</td></tr>
	<tr><td style="padding:0 20px 16px 20px;color:#374151;font-size:14px;line-height:1.8;">状态：<strong style="color:#15803d;">配置成功</strong></td></tr>
</table>`, siteName)

	return emailBaseHTML(siteName, "SMTP 配置测试", content)
}
