package service

import (
	"fmt"
	"html"
)

func buildVerifyCodeEmailBody(code, siteName string) string {
	safeCode := html.EscapeString(code)

	primary := fmt.Sprintf(`
<div style="margin-top: 24px; padding: 24px; border: 1px solid #e7e9ee; border-radius: 18px; background-color: #f7f8fc; text-align: center;">
    <div style="font-size: 12px; line-height: 18px; letter-spacing: 0.14em; text-transform: uppercase; color: #7c3aed; font-weight: 700;">Verification Code</div>
    <div style="margin-top: 14px; font-size: 34px; line-height: 1; letter-spacing: 0.22em; font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace; font-weight: 700; color: #111827;">%s</div>
</div>
`, safeCode)

	details := `
<div style="margin-top: 20px; padding: 18px 20px; border-left: 4px solid #7c3aed; background-color: #faf7ff; border-radius: 14px;">
    <div style="font-size: 14px; line-height: 24px; color: #4b5563;">
        验证码将在 <strong style="color: #111827;">15 分钟</strong>后失效。<br>
        如果这不是你的操作，忽略此邮件即可，账号不会受到影响。
    </div>
</div>
`

	return renderBusinessEmail(
		siteName,
		"ACCOUNT SECURITY",
		"邮箱验证码",
		"你正在进行邮箱验证。请在页面中输入下方验证码以继续操作。",
		primary,
		details,
	)
}

func buildPasswordResetEmailBody(resetURL, siteName string) string {
	safeResetURL := html.EscapeString(resetURL)

	primary := fmt.Sprintf(`
<div style="margin-top: 24px; text-align: center;">
    <a href="%s" style="display: inline-block; padding: 14px 30px; border-radius: 999px; background: linear-gradient(135deg, #6d28d9 0%%, #8b5cf6 100%%); color: #ffffff; text-decoration: none; font-size: 15px; line-height: 22px; font-weight: 700;">重置密码</a>
</div>
`, safeResetURL)

	details := fmt.Sprintf(`
<div style="margin-top: 20px; padding: 18px 20px; border-left: 4px solid #7c3aed; background-color: #faf7ff; border-radius: 14px;">
    <div style="font-size: 14px; line-height: 24px; color: #4b5563;">
        此链接将在 <strong style="color: #111827;">30 分钟</strong>后失效。<br>
        如果按钮无法点击，可复制以下链接到浏览器打开：
    </div>
    <div style="margin-top: 12px; padding: 14px 16px; border: 1px solid #e7e9ee; border-radius: 12px; background-color: #ffffff; font-size: 12px; line-height: 20px; color: #6b7280; word-break: break-all;">%s</div>
</div>
`, safeResetURL)

	return renderBusinessEmail(
		siteName,
		"ACCOUNT SECURITY",
		"重置登录密码",
		"我们收到了你的密码重置请求。点击下方按钮即可继续设置新密码。",
		primary,
		details,
	)
}

func BuildTestEmailBody(siteName string) string {
	primary := `
<div style="margin-top: 24px; padding: 24px; border: 1px solid #e7e9ee; border-radius: 18px; background-color: #f7f8fc; text-align: center;">
    <div style="display: inline-block; padding: 6px 12px; border-radius: 999px; background-color: #ecfdf3; color: #047857; font-size: 12px; line-height: 18px; font-weight: 700;">SMTP Ready</div>
    <div style="margin-top: 14px; font-size: 22px; line-height: 30px; font-weight: 700; color: #111827;">邮件投递链路工作正常</div>
</div>
`

	details := `
<div style="margin-top: 20px; padding: 18px 20px; border-left: 4px solid #7c3aed; background-color: #faf7ff; border-radius: 14px;">
    <div style="font-size: 14px; line-height: 24px; color: #4b5563;">
        这是一封系统测试邮件，用于确认当前 SMTP 主机、鉴权信息与发件人配置已可正常发送。
    </div>
</div>
`

	return renderBusinessEmail(
		siteName,
		"SYSTEM CHECK",
		"SMTP 测试成功",
		"如果你收到了这封邮件，说明当前邮件配置已经通过基础发送验证。",
		primary,
		details,
	)
}

func renderBusinessEmail(siteName, eyebrow, title, intro, primaryHTML, detailsHTML string) string {
	safeSiteName := html.EscapeString(siteName)
	if safeSiteName == "" {
		safeSiteName = "Sub2API"
	}

	safeEyebrow := html.EscapeString(eyebrow)
	safeTitle := html.EscapeString(title)
	safeIntro := html.EscapeString(intro)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="color-scheme" content="light only">
    <title>%s</title>
</head>
<body style="margin: 0; padding: 0; background-color: #f3f4f8; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif; color: #111827;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width: 100%%; background-color: #f3f4f8;">
        <tr>
            <td align="center" style="padding: 32px 16px;">
                <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width: 640px; width: 100%%; border-collapse: separate; background-color: #ffffff; border: 1px solid #e5e7eb; border-radius: 24px; overflow: hidden;">
                    <tr>
                        <td style="height: 6px; background: linear-gradient(90deg, #6d28d9 0%%, #8b5cf6 100%%); font-size: 0; line-height: 0;">&nbsp;</td>
                    </tr>
                    <tr>
                        <td style="padding: 30px 32px 12px 32px;">
                            <div style="font-size: 12px; line-height: 18px; letter-spacing: 0.14em; text-transform: uppercase; color: #7c3aed; font-weight: 700;">%s</div>
                            <div style="margin-top: 10px; font-size: 30px; line-height: 38px; font-weight: 700; color: #111827;">%s</div>
                            <div style="margin-top: 12px; font-size: 15px; line-height: 26px; color: #4b5563;">%s</div>
                            %s
                            %s
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 0 32px 30px 32px;">
                            <div style="padding-top: 18px; border-top: 1px solid #e5e7eb; font-size: 13px; line-height: 22px; color: #6b7280;">
                                <strong style="display: block; margin-bottom: 4px; font-size: 14px; color: #111827;">%s</strong>
                                这是一封系统自动发送的邮件，请勿直接回复。
                            </div>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`, safeTitle, safeEyebrow, safeTitle, safeIntro, primaryHTML, detailsHTML, safeSiteName)
}
