package utils

import (
	"fmt"
)

// GetVaultKeIconSVG returns a clean, email-optimized VaultKe logo
func GetVaultKeIconSVG() string {
	return `<svg width="80" height="80" viewBox="0 0 80 80" xmlns="http://www.w3.org/2000/svg">
		<!-- Clean VaultKe Logo for Email Compatibility -->

		<!-- Outer Circle Background -->
		<circle cx="40" cy="40" r="38" fill="#2563eb" stroke="#1d4ed8" stroke-width="2"/>

		<!-- Inner Vault Ring -->
		<circle cx="40" cy="40" r="30" fill="none" stroke="#ffffff" stroke-width="2" opacity="0.8"/>

		<!-- Security Bolts -->
		<circle cx="20" cy="20" r="2.5" fill="#ffffff"/>
		<circle cx="60" cy="20" r="2.5" fill="#ffffff"/>
		<circle cx="20" cy="60" r="2.5" fill="#ffffff"/>
		<circle cx="60" cy="60" r="2.5" fill="#ffffff"/>

		<!-- Central Lock -->
		<circle cx="40" cy="40" r="15" fill="#fbbf24" stroke="#f59e0b" stroke-width="2"/>

		<!-- Lock Dial Markers -->
		<g stroke="#ffffff" stroke-width="2" stroke-linecap="round">
			<line x1="40" y1="28" x2="40" y2="32"/>
			<line x1="52" y1="40" x2="48" y2="40"/>
			<line x1="40" y1="52" x2="40" y2="48"/>
			<line x1="28" y1="40" x2="32" y2="40"/>
		</g>

		<!-- Central Handle -->
		<circle cx="40" cy="40" r="5" fill="#ffffff"/>
		<rect x="38" y="35" width="4" height="10" fill="#2563eb" rx="2"/>

		<!-- VK Text -->
		<text x="40" y="72" text-anchor="middle" fill="#ffffff" font-family="Arial, sans-serif" font-size="10" font-weight="bold">VK</text>
	</svg>`
}

// GetVaultKeIconDataURL returns the VaultKe PNG icon URL for embedding in emails
func GetVaultKeIconDataURL() string {
	// Use the Imgur-hosted VaultKe icon directly - try different quality options
	possibleURLs := []string{
		"https://i.imgur.com/XdVwni7.png",  // Original
		"https://imgur.com/XdVwni7.png",    // Alternative format
		"https://i.imgur.com/XdVwni7h.png", // High quality version
	}

	// Return the first URL (can be enhanced with URL validation later)
	return possibleURLs[0]
}

// GetVaultKeLogoHTML returns HTML for displaying the VaultKe logo in emails
func GetVaultKeLogoHTML() string {
	return fmt.Sprintf(`
		<div style="text-align: center; padding: 40px 20px; background: linear-gradient(135deg, #ffffff 0%%, #f8fafc 100%%);">
			<img src="%s" alt="VaultKe" style="width: 80px; height: 80px; display: block; margin: 0 auto 20px auto;" />
			<h1 style="font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Arial, sans-serif; font-size: 32px; font-weight: 700; color: #00D4AA; margin: 0 0 8px 0; letter-spacing: -0.5px;">VaultKe</h1>
			<p style="font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Arial, sans-serif; font-size: 16px; color: #64748b; margin: 0; font-weight: 500;">Your Trusted Chama Finance Companion</p>
		</div>
	`, GetVaultKeIconDataURL())
}

// GetEmailTemplate returns a professional email template
func GetEmailTemplate(title, content, buttonText, buttonURL string) string {
	logoHTML := GetVaultKeLogoHTML()

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - VaultKe</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Arial, sans-serif; line-height: 1.6; color: #334155; background-color: #f1f5f9; }
        .email-container { max-width: 600px; margin: 0 auto; background-color: #ffffff; box-shadow: 0 10px 25px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #00D4AA 0%%, #00B894 100%%); padding: 0; }
        .content { padding: 40px 30px; }
        .footer { background-color: #f8fafc; padding: 30px; text-align: center; border-top: 1px solid #e2e8f0; }
        .btn { display: inline-block; padding: 16px 32px; background: linear-gradient(135deg, #00D4AA 0%%, #00B894 100%%); color: #ffffff; text-decoration: none; border-radius: 8px; font-weight: 600; font-size: 16px; margin: 20px 0; box-shadow: 0 4px 12px rgba(0, 212, 170, 0.3); transition: all 0.3s ease; }
        .btn:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0, 212, 170, 0.4); }
        .security-notice { background-color: #fef3c7; border-left: 4px solid #f59e0b; padding: 16px; margin: 20px 0; border-radius: 4px; }
        .divider { height: 1px; background: linear-gradient(90deg, transparent, #e2e8f0, transparent); margin: 30px 0; }
        h2 { color: #1e293b; font-size: 24px; font-weight: 700; margin-bottom: 16px; }
        p { margin-bottom: 16px; font-size: 16px; line-height: 1.7; }
        .highlight { color: #00D4AA; font-weight: 600; }
        .social-links { margin-top: 20px; }
        .social-links a { display: inline-block; margin: 0 10px; color: #64748b; text-decoration: none; }
        @media (max-width: 600px) {
            .email-container { margin: 0; box-shadow: none; }
            .content { padding: 30px 20px; }
            .btn { display: block; text-align: center; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            %s
        </div>
        <div class="content">
            <h2>%s</h2>
            %s
            %s
        </div>
        <div class="footer">
            <div class="divider"></div>
            <p style="color: #64748b; font-size: 14px; margin-bottom: 16px;">
                This email was sent by VaultKe. If you have any questions, please contact our support team.
            </p>
            <div class="social-links">
                <a href="#" style="color: #64748b;">Privacy Policy</a>
                <a href="#" style="color: #64748b;">Terms of Service</a>
                <a href="#" style="color: #64748b;">Support</a>
            </div>
            <p style="color: #94a3b8; font-size: 12px; margin-top: 20px;">
                © 2024 VaultKe. All rights reserved.<br>
                Empowering Chama communities across Kenya.
            </p>
        </div>
    </div>
</body>
</html>`, title, logoHTML, title, content,
		func() string {
			if buttonText != "" && buttonURL != "" {
				return fmt.Sprintf(`<div style="text-align: center; margin: 30px 0;"><a href="%s" class="btn">%s</a></div>`, buttonURL, buttonText)
			}
			return ""
		}())
}

// GetVaultKeLogoHTMLSimple returns a clean HTML-only logo for maximum email compatibility
func GetVaultKeLogoHTMLSimple() string {
	return `
		<div style="text-align: center; margin: 20px 0; padding: 20px; background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%); border-radius: 12px; border: 2px solid #e5e7eb;">
			<!-- Clean HTML Logo Circle -->
			<div style="display: inline-block; width: 80px; height: 80px; background: #2563eb; border-radius: 50%; position: relative; margin-bottom: 15px; box-shadow: 0 4px 12px rgba(37, 99, 235, 0.2);">
				<!-- Vault Door Effect -->
				<div style="position: absolute; top: 10px; left: 10px; width: 60px; height: 60px; border: 2px solid #ffffff; border-radius: 50%; opacity: 0.8;"></div>
				<!-- Security Bolts -->
				<div style="position: absolute; top: 8px; left: 8px; width: 6px; height: 6px; background: #ffffff; border-radius: 50%;"></div>
				<div style="position: absolute; top: 8px; right: 8px; width: 6px; height: 6px; background: #ffffff; border-radius: 50%;"></div>
				<div style="position: absolute; bottom: 8px; left: 8px; width: 6px; height: 6px; background: #ffffff; border-radius: 50%;"></div>
				<div style="position: absolute; bottom: 8px; right: 8px; width: 6px; height: 6px; background: #ffffff; border-radius: 50%;"></div>
				<!-- Central Lock -->
				<div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 24px; height: 24px; background: #fbbf24; border-radius: 50%; border: 2px solid #f59e0b;">
					<div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 8px; height: 8px; background: #ffffff; border-radius: 50%;"></div>
				</div>
				<!-- VK Text -->
				<div style="position: absolute; bottom: -25px; left: 50%; transform: translateX(-50%); color: #ffffff; font-family: Arial, sans-serif; font-size: 10px; font-weight: bold;">VK</div>
			</div>
			<!-- Brand Text -->
			<div style="font-family: 'Segoe UI', Arial, sans-serif; font-size: 28px; font-weight: bold; color: #2563eb; margin-bottom: 6px; letter-spacing: 1px;">VaultKe</div>
			<div style="font-family: Arial, sans-serif; font-size: 14px; color: #64748b; font-weight: 500;">Your Trusted Chama Finance Companion</div>
		</div>
	`
}

// GetVaultKeLogoHTMLWithPNG returns HTML with PNG icon for emails (requires hosting the PNG)
func GetVaultKeLogoHTMLWithPNG(iconURL string) string {
	return fmt.Sprintf(`
		<div style="text-align: center; margin: 20px 0; padding: 15px;">
			<img src="%s" alt="VaultKe" style="width: 80px; height: 80px; display: block; margin: 0 auto 10px auto; border-radius: 50%%; border: 3px solid #2563eb;" />
			<div style="font-family: Arial, sans-serif; font-size: 24px; font-weight: bold; color: #2563eb; margin-bottom: 4px;">VaultKe</div>
			<div style="font-family: Arial, sans-serif; font-size: 12px; color: #64748b;">Your Trusted Chama Finance Companion</div>
		</div>
	`, iconURL)
}
