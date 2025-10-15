package services

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"vaultke-backend/internal/utils"
)

// EmailService handles email sending functionality
type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	fmt.Printf("🔧 [EMAIL SERVICE] Initializing email service...\n")

	// Get all SMTP configuration from environment
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")

	// Debug: Print SMTP configuration (without showing full password)
	fmt.Printf("📧 [EMAIL SERVICE] SMTP Configuration:\n")
	fmt.Printf("  - SMTP_HOST: '%s'\n", smtpHost)
	fmt.Printf("  - SMTP_PORT: '%s'\n", smtpPort)
	fmt.Printf("  - SMTP_USERNAME: '%s'\n", smtpUsername)
	fmt.Printf("  - SMTP_PASSWORD: '%s' (length: %d)\n", maskPassword(password), len(password))

	// Trim quotes from password if present
	if len(password) >= 2 && password[0] == '"' && password[len(password)-1] == '"' {
		password = password[1 : len(password)-1]
		fmt.Printf("  - Trimmed quotes from password (new length: %d)\n", len(password))
	}

	emailService := &EmailService{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: password,
		fromEmail:    smtpUsername, // Use SMTP username as from email
	}

	// Check if configuration is complete
	if smtpHost == "" || smtpPort == "" || smtpUsername == "" || password == "" {
		fmt.Printf("⚠️  [EMAIL SERVICE] Incomplete SMTP configuration detected!\n")
		fmt.Printf("  - Missing SMTP_HOST: %t\n", smtpHost == "")
		fmt.Printf("  - Missing SMTP_PORT: %t\n", smtpPort == "")
		fmt.Printf("  - Missing SMTP_USERNAME: %t\n", smtpUsername == "")
		fmt.Printf("  - Missing SMTP_PASSWORD: %t\n", password == "")
		fmt.Printf("❌ [EMAIL SERVICE] Email service will not be functional!\n")
	} else {
		fmt.Printf("✅ [EMAIL SERVICE] Email service initialized successfully!\n")
	}

	return emailService
}

// maskPassword masks a password for logging purposes
func maskPassword(password string) string {
	if len(password) == 0 {
		return "<empty>"
	}
	if len(password) <= 4 {
		return "****"
	}
	if len(password) <= 8 {
		return password[:1] + "****" + password[len(password)-1:]
	}
	return password[:2] + "****" + password[len(password)-2:]
}

// SendPasswordResetEmail sends a password reset email to the user
func (s *EmailService) SendPasswordResetEmail(toEmail, resetToken, userName string) error {
	fmt.Printf("📧 SendPasswordResetEmail called for: %s\n", toEmail)

	// Validate configuration
	if s.smtpHost == "" || s.smtpPort == "" || s.smtpUsername == "" || s.smtpPassword == "" {
		fmt.Printf("❌ Email service configuration missing:\n")
		fmt.Printf("  Host: '%s' (empty: %t)\n", s.smtpHost, s.smtpHost == "")
		fmt.Printf("  Port: '%s' (empty: %t)\n", s.smtpPort, s.smtpPort == "")
		fmt.Printf("  Username: '%s' (empty: %t)\n", s.smtpUsername, s.smtpUsername == "")
		fmt.Printf("  Password set: %t\n", s.smtpPassword != "")

		// For development, let's simulate sending the email
		fmt.Printf("🔧 DEVELOPMENT MODE: Simulating email send\n")
		fmt.Printf("📧 Would send password reset email to: %s\n", toEmail)
		fmt.Printf("🔑 Reset token: %s\n", resetToken)
		fmt.Printf("👤 User name: %s\n", userName)

		// Return success in development mode when SMTP is not configured
		return nil
	}

	fmt.Printf("Email service configuration loaded:\n")
	fmt.Printf("  Host: %s\n", s.smtpHost)
	fmt.Printf("  Port: %s\n", s.smtpPort)
	fmt.Printf("  Username: %s\n", s.smtpUsername)
	fmt.Printf("  Password: '%s' (length: %d)\n", s.smtpPassword, len(s.smtpPassword))
	fmt.Printf("Sending password reset email to: %s with token: %s\n", toEmail, resetToken)

	// Create reset URL (you can customize this based on your frontend)
	resetURL := fmt.Sprintf("https://vaultke.com/reset-password?token=%s", resetToken)

	// Email subject and body
	subject := "VaultKe - Password Reset Request"
	body := s.generatePasswordResetEmailBody(userName, resetURL, resetToken)

	// Create email message in the exact format as your working app
	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		toEmail, subject, body)

	// Send email
	err := s.sendEmail(toEmail, message)
	if err != nil {
		fmt.Printf("Failed to send email: %v\n", err)
		return err
	}

	fmt.Printf("Email sent successfully to: %s\n", toEmail)
	return nil
}

// sendEmail sends an email using SMTP - exact same approach as your working code
func (s *EmailService) sendEmail(toEmail, message string) error {
	// Set up authentication - exact same as your working code
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	// SMTP server address
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)

	fmt.Printf("Attempting to send email via SMTP server: %s\n", addr)
	fmt.Printf("From: %s, To: %s\n", s.fromEmail, toEmail)
	fmt.Printf("Auth details - Username: %s, Password: '%s' (length: %d)\n", s.smtpUsername, s.smtpPassword, len(s.smtpPassword))

	// Send email - exact same method as your working code
	err := smtp.SendMail(addr, auth, s.fromEmail, []string{toEmail}, []byte(message))
	if err != nil {
		fmt.Printf("SMTP error: %v\n", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("✅ Email sent successfully!\n")
	return nil
}

// SendChamaInvitationEmail sends a chama invitation email to the invitee
func (s *EmailService) SendChamaInvitationEmail(toEmail, chamaName, inviterName, message, invitationToken string) error {
	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ [CHAMA INVITATION] Panic during email send: %v\n", r)
		}
	}()

	fmt.Printf("🔍 [CHAMA INVITATION] Starting email send process...\n")
	fmt.Printf("  - To: %s\n", toEmail)
	fmt.Printf("  - Chama: %s\n", chamaName)
	fmt.Printf("  - Inviter: %s\n", inviterName)
	fmt.Printf("  - Token: %s\n", invitationToken)

	// Validate input parameters
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}
	if chamaName == "" {
		return fmt.Errorf("chama name is required")
	}
	if inviterName == "" {
		return fmt.Errorf("inviter name is required")
	}
	if invitationToken == "" {
		return fmt.Errorf("invitation token is required")
	}

	// Validate configuration
	if s.smtpHost == "" || s.smtpPort == "" || s.smtpUsername == "" || s.smtpPassword == "" {
		fmt.Printf("❌ [CHAMA INVITATION] Email service configuration missing\n")
		fmt.Printf("  - SMTP Host: '%s'\n", s.smtpHost)
		fmt.Printf("  - SMTP Port: '%s'\n", s.smtpPort)
		fmt.Printf("  - SMTP Username: '%s'\n", s.smtpUsername)
		fmt.Printf("  - SMTP Password: '%s' (length: %d)\n", maskPassword(s.smtpPassword), len(s.smtpPassword))
		return fmt.Errorf("email service not configured properly")
	}

	fmt.Printf("📧 [CHAMA INVITATION] Sending chama invitation email to: %s for chama: %s\n", toEmail, chamaName)

	// Create invitation URL
	invitationURL := fmt.Sprintf("https://vaultke.com/chama-invitation?token=%s", invitationToken)
	fmt.Printf("🔗 [CHAMA INVITATION] Invitation URL: %s\n", invitationURL)

	// Email subject and body
	subject := fmt.Sprintf("VaultKe - Invitation to Join %s Chama", chamaName)
	fmt.Printf("📝 [CHAMA INVITATION] Email subject: %s\n", subject)

	body := s.generateChamaInvitationEmailBody(chamaName, inviterName, message, invitationURL, invitationToken)
	fmt.Printf("📄 [CHAMA INVITATION] Email body generated (length: %d chars)\n", len(body))

	// Debug: Print image info for chama invitation email
	imageURL := utils.GetVaultKeIconDataURL()
	if strings.HasPrefix(imageURL, "https://") {
		fmt.Printf("🌐 [CHAMA INVITATION] Using hosted image URL: %s\n", imageURL)
	} else if strings.HasPrefix(imageURL, "data:image/png;base64,") {
		fmt.Printf("🖼️  [CHAMA INVITATION] Using PNG image (length: %d chars)\n", len(imageURL))
		previewLen := 100
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("📊 PNG preview: %s...\n", imageURL[:previewLen])
	} else if strings.HasPrefix(imageURL, "data:image/svg+xml;base64,") {
		fmt.Printf("⚠️  [CHAMA INVITATION] Using SVG fallback (length: %d chars)\n", len(imageURL))
		previewLen := 100
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("📊 SVG preview: %s...\n", imageURL[:previewLen])
	} else {
		fmt.Printf("❌ [CHAMA INVITATION] Unknown image format (length: %d): ", len(imageURL))
		previewLen := 50
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("%s...\n", imageURL[:previewLen])
	}

	// Create email message in the exact format as your working app
	emailMessage := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		toEmail, subject, body)
	fmt.Printf("📧 [CHAMA INVITATION] Email message created (total length: %d chars)\n", len(emailMessage))

	// Send email
	fmt.Printf("🚀 [CHAMA INVITATION] Attempting to send email via SMTP...\n")
	err := s.sendEmail(toEmail, emailMessage)
	if err != nil {
		fmt.Printf("❌ [CHAMA INVITATION] Failed to send chama invitation email: %v\n", err)
		return err
	}

	fmt.Printf("✅ [CHAMA INVITATION] Email sent successfully to: %s\n", toEmail)
	fmt.Printf("🎉 [CHAMA INVITATION] Chama invitation email sent successfully!\n")
	return nil
}

// generatePasswordResetEmailBody generates the HTML email body for password reset
func (s *EmailService) generatePasswordResetEmailBody(userName, resetURL, resetToken string) string {
	// If userName is empty, use a generic greeting
	if userName == "" {
		userName = "VaultKe User"
	}

	// Debug: Print image info for password reset email
	imageURL := utils.GetVaultKeIconDataURL()
	if strings.HasPrefix(imageURL, "https://") {
		fmt.Printf("🌐 [PASSWORD RESET] Using hosted image URL: %s\n", imageURL)
	} else if strings.HasPrefix(imageURL, "data:image/png;base64,") {
		fmt.Printf("🖼️  [PASSWORD RESET] Using PNG image (length: %d chars)\n", len(imageURL))
		previewLen := 100
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("📊 PNG preview: %s...\n", imageURL[:previewLen])
	} else if strings.HasPrefix(imageURL, "data:image/svg+xml;base64,") {
		fmt.Printf("⚠️  [PASSWORD RESET] Using SVG fallback (length: %d chars)\n", len(imageURL))
		previewLen := 100
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("📊 SVG preview: %s...\n", imageURL[:previewLen])
	} else {
		fmt.Printf("❌ [PASSWORD RESET] Unknown image format (length: %d): ", len(imageURL))
		previewLen := 50
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("%s...\n", imageURL[:previewLen])
	}

	content := fmt.Sprintf(`
		<p style="font-size: 18px; color: #1e293b; margin-bottom: 24px;">Hello <span class="highlight">%s</span>,</p>

		<p>We received a request to reset your password for your VaultKe account. Use the verification code below to complete the process:</p>

		<div style="text-align: center; margin: 30px 0;">
			<div style="display: inline-block; background: linear-gradient(135deg, #f0fdf4 0%%, #dcfce7 100%%); border: 2px solid #00D4AA; border-radius: 12px; padding: 20px 30px;">
				<p style="font-size: 14px; color: #64748b; margin: 0 0 8px 0; text-transform: uppercase; letter-spacing: 1px; font-weight: 600;">Verification Code</p>
				<div style="font-size: 32px; font-weight: 800; color: #00D4AA; letter-spacing: 3px; font-family: 'Courier New', monospace;">%s</div>
			</div>
		</div>

		<div class="security-notice">
			<p style="margin: 0; font-size: 14px; color: #92400e;"><strong>⚠️ Security Notice:</strong> This code expires in <strong>2 minutes</strong>. If you didn't request this reset, please ignore this email - your account remains secure.</p>
		</div>

		<p>For your security, never share this code with anyone. Our team will never ask for your verification code.</p>

		<p style="margin-top: 30px;">Best regards,<br><span class="highlight">The VaultKe Team</span></p>
	`, userName, resetToken)

	return utils.GetEmailTemplate("Password Reset Request", content, "", "")
}

// generateChamaInvitationEmailBody generates the HTML email body for chama invitation
func (s *EmailService) generateChamaInvitationEmailBody(chamaName, inviterName, message, invitationURL, invitationToken string) string {
	// If message is empty, use a default message
	if message == "" {
		message = fmt.Sprintf("You have been invited to join %s chama. Join us to start saving and investing together!", chamaName)
	}

	// Debug: Print image info for chama invitation email
	dataURL := utils.GetVaultKeIconDataURL()
	if strings.HasPrefix(dataURL, "data:image/png;base64,") {
		fmt.Printf("🖼️  [CHAMA INVITATION] Using PNG image (length: %d chars)\n", len(dataURL))
		previewLen := 100
		if len(dataURL) < previewLen {
			previewLen = len(dataURL)
		}
		fmt.Printf("📊 PNG preview: %s...\n", dataURL[:previewLen])
	} else if strings.HasPrefix(dataURL, "data:image/svg+xml;base64,") {
		fmt.Printf("⚠️  [CHAMA INVITATION] Using SVG fallback (length: %d chars)\n", len(dataURL))
		previewLen := 100
		if len(dataURL) < previewLen {
			previewLen = len(dataURL)
		}
		fmt.Printf("📊 SVG preview: %s...\n", dataURL[:previewLen])
	} else {
		fmt.Printf("❌ [CHAMA INVITATION] Unknown image format (length: %d): ", len(dataURL))
		previewLen := 50
		if len(dataURL) < previewLen {
			previewLen = len(dataURL)
		}
		fmt.Printf("%s...\n", dataURL[:previewLen])
	}

	content := fmt.Sprintf(`
		<p style="font-size: 18px; color: #1e293b; margin-bottom: 24px;">Hello there,</p>

		<p><span class="highlight">%s</span> has invited you to join <strong>%s</strong> chama on VaultKe. Use the invitation code below to join this chama:</p>

		<div style="text-align: center; margin: 30px 0;">
			<div style="display: inline-block; background: linear-gradient(135deg, #f0fdf4 0%%, #dcfce7 100%%); border: 2px solid #00D4AA; border-radius: 12px; padding: 20px 30px;">
				<p style="font-size: 14px; color: #64748b; margin: 0 0 8px 0; text-transform: uppercase; letter-spacing: 1px; font-weight: 600;">Invitation Code</p>
				<div style="font-size: 32px; font-weight: 800; color: #00D4AA; letter-spacing: 3px; font-family: 'Courier New', monospace;">%s</div>
			</div>
		</div>

		<div class="security-notice">
			<p style="margin: 0; font-size: 14px; color: #92400e;"><strong>⚠️ Security Notice:</strong> This invitation code expires in <strong>7 days</strong>. If you don't know %s, please ignore this email - your account remains secure.</p>
		</div>

		<p><strong>Personal Message from %s:</strong><br>"%s"</p>

		<p>For your security, never share this invitation code with anyone. Our team will never ask for your invitation code.</p>

		<p style="margin-top: 30px;">Ready to join the chama?<br><span class="highlight">The VaultKe Team</span></p>
	`, inviterName, chamaName, invitationToken, inviterName, inviterName, message)

	return utils.GetEmailTemplate(fmt.Sprintf("Join %s Chama", chamaName), content, "Accept Invitation", invitationURL)
}

// SendTestEmail sends a test email to verify configuration
func (s *EmailService) SendTestEmail(toEmail string) error {
	subject := "VaultKe - Email Service Test"

	// Debug: Print image info for test email
	dataURL := utils.GetVaultKeIconDataURL()
	if strings.HasPrefix(dataURL, "data:image/png;base64,") {
		fmt.Printf("🖼️  [TEST EMAIL] Using PNG image (length: %d chars)\n", len(dataURL))
		previewLen := 100
		if len(dataURL) < previewLen {
			previewLen = len(dataURL)
		}
		fmt.Printf("📊 PNG preview: %s...\n", dataURL[:previewLen])
	} else if strings.HasPrefix(dataURL, "data:image/svg+xml;base64,") {
		fmt.Printf("⚠️  [TEST EMAIL] Using SVG fallback (length: %d chars)\n", len(dataURL))
		previewLen := 100
		if len(dataURL) < previewLen {
			previewLen = len(dataURL)
		}
		fmt.Printf("📊 SVG preview: %s...\n", dataURL[:previewLen])
	} else {
		fmt.Printf("❌ [TEST EMAIL] Unknown image format (length: %d): ", len(dataURL))
		previewLen := 50
		if len(dataURL) < previewLen {
			previewLen = len(dataURL)
		}
		fmt.Printf("%s...\n", dataURL[:previewLen])
	}

	content := `
		<p style="font-size: 18px; color: #1e293b; margin-bottom: 24px;">Hello <span class="highlight">VaultKe Team</span>,</p>

		<p>This is a test email from VaultKe to verify that our email service is configured correctly and working properly.</p>

		<div style="text-align: center; margin: 30px 0;">
			<div style="display: inline-block; background: linear-gradient(135deg, #f0fdf4 0%%, #dcfce7 100%%); border: 2px solid #00D4AA; border-radius: 12px; padding: 20px 30px;">
				<p style="font-size: 14px; color: #64748b; margin: 0 0 8px 0; text-transform: uppercase; letter-spacing: 1px; font-weight: 600;">System Status</p>
				<div style="font-size: 32px; font-weight: 800; color: #00D4AA; letter-spacing: 3px; font-family: 'Courier New', monospace;">SUCCESS</div>
			</div>
		</div>

		<div class="security-notice">
			<p style="margin: 0; font-size: 14px; color: #92400e;"><strong>⚠️ System Notice:</strong> All email services are functioning correctly. SMTP configuration, template rendering, and image loading are operational.</p>
		</div>

		<p>Your customers will receive professional, well-formatted emails for password resets, email verification, and chama invitations.</p>

		<p>For your security, this test email confirms that the email system is ready for production use.</p>

		<p style="margin-top: 30px;">Email system operational!<br><span class="highlight">The VaultKe Development Team</span></p>
	`

	body := utils.GetEmailTemplate("Email Service Test", content, "", "")

	message := fmt.Sprintf("From: %s\r\n", s.fromEmail)
	message += fmt.Sprintf("To: %s\r\n", toEmail)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	return s.sendEmail(toEmail, message)
}

// SendEmailVerificationEmail sends an email verification email
func (s *EmailService) SendEmailVerificationEmail(to, verificationCode, userName string) error {
	fmt.Printf("📧 SendEmailVerificationEmail called for: %s\n", to)

	// Validate configuration
	if s.smtpHost == "" || s.smtpPort == "" || s.smtpUsername == "" || s.smtpPassword == "" {
		fmt.Printf("❌ Email service configuration missing for verification email\n")

		// For development, let's simulate sending the email
		fmt.Printf("🔧 DEVELOPMENT MODE: Simulating verification email send\n")
		fmt.Printf("📧 Would send verification email to: %s\n", to)
		fmt.Printf("🔑 Verification code: %s\n", verificationCode)
		fmt.Printf("👤 User name: %s\n", userName)

		// Return success in development mode when SMTP is not configured
		return nil
	}

	subject := "Verify Your Email - VaultKe"

	// Debug: Print image info for email verification
	imageURL := utils.GetVaultKeIconDataURL()
	if strings.HasPrefix(imageURL, "https://") {
		fmt.Printf("🌐 [EMAIL VERIFICATION] Using hosted image URL: %s\n", imageURL)
	} else if strings.HasPrefix(imageURL, "data:image/png;base64,") {
		fmt.Printf("🖼️  [EMAIL VERIFICATION] Using PNG image (length: %d chars)\n", len(imageURL))
		previewLen := 100
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("📊 PNG preview: %s...\n", imageURL[:previewLen])
	} else if strings.HasPrefix(imageURL, "data:image/svg+xml;base64,") {
		fmt.Printf("⚠️  [EMAIL VERIFICATION] Using SVG fallback (length: %d chars)\n", len(imageURL))
		previewLen := 100
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("📊 SVG preview: %s...\n", imageURL[:previewLen])
	} else {
		fmt.Printf("❌ [EMAIL VERIFICATION] Unknown image format (length: %d): ", len(imageURL))
		previewLen := 50
		if len(imageURL) < previewLen {
			previewLen = len(imageURL)
		}
		fmt.Printf("%s...\n", imageURL[:previewLen])
	}

	content := fmt.Sprintf(`
		<p style="font-size: 18px; color: #1e293b; margin-bottom: 24px;">Hello <span class="highlight">%s</span>,</p>

		<p>Welcome to <strong>VaultKe</strong>! 🎉 To complete your registration and start managing your contribution groups and chama finances, please verify your email address using the code below:</p>

		<div style="text-align: center; margin: 30px 0;">
			<div style="display: inline-block; background: linear-gradient(135deg, #f0fdf4 0%%, #dcfce7 100%%); border: 2px solid #00D4AA; border-radius: 12px; padding: 20px 30px;">
				<p style="font-size: 14px; color: #64748b; margin: 0 0 8px 0; text-transform: uppercase; letter-spacing: 1px; font-weight: 600;">Email Verification Code</p>
				<div style="font-size: 32px; font-weight: 800; color: #00D4AA; letter-spacing: 3px; font-family: 'Courier New', monospace;">%s</div>
			</div>
		</div>

		<div class="security-notice">
			<p style="margin: 0; font-size: 14px; color: #92400e;"><strong>⚠️ Security Notice:</strong> This verification code expires in <strong>2 minutes</strong>. If you didn't create a VaultKe account, please ignore this email - your account remains secure.</p>
		</div>

		<p>Once verified, you'll have full access to VaultKe's chama management, financial tracking, and secure transaction features.</p>

		<p>For your security, never share this verification code with anyone. Our team will never ask for your verification code.</p>

		<p style="margin-top: 30px;">Welcome to the VaultKe family!<br><span class="highlight">The VaultKe Team</span></p>
	`, userName, verificationCode)

	htmlBody := utils.GetEmailTemplate("Verify Your Email Address", content, "", "")

	// Create email message
	message := fmt.Sprintf("From: %s\r\n", s.fromEmail)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += htmlBody

	return s.sendEmail(to, message)
}
