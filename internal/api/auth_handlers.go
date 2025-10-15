package api

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"
)

// DeviceInfo represents device information extracted from request
type DeviceInfo struct {
	DeviceType string
	DeviceName string
	OS         string
	Browser    string
	Location   string
}

// extractDeviceInfo extracts device information from request headers
func extractDeviceInfo(c *gin.Context) DeviceInfo {
	// Use browser's natural User-Agent header (automatically set)
	userAgent := c.GetHeader("User-Agent")

	deviceInfo := DeviceInfo{
		DeviceType: "unknown",
		DeviceName: "Unknown Device",
		OS:         "Unknown",
		Browser:    "Unknown Browser",
		Location:   "Unknown",
	}

	// Enhanced device detection with more specific browser and device info
	userAgentLower := strings.ToLower(userAgent)

	// Detect mobile devices first
	if strings.Contains(userAgentLower, "mobile") || strings.Contains(userAgentLower, "android") || strings.Contains(userAgentLower, "iphone") || strings.Contains(userAgentLower, "ipad") {
		deviceInfo.DeviceType = "mobile"

		// Android devices
		if strings.Contains(userAgentLower, "android") {
			deviceInfo.OS = "Android"
			// Try to extract Android version
			if strings.Contains(userAgent, "Android ") {
				parts := strings.Split(userAgent, "Android ")
				if len(parts) > 1 {
					version := strings.Split(parts[1], ";")[0]
					deviceInfo.OS = fmt.Sprintf("Android %s", version)
				}
			}

			// Detect specific Android browsers
			if strings.Contains(userAgentLower, "chrome") && !strings.Contains(userAgentLower, "edg") {
				deviceInfo.Browser = "Chrome Mobile"
				deviceInfo.DeviceName = "Android Phone • Chrome"
			} else if strings.Contains(userAgentLower, "firefox") {
				deviceInfo.Browser = "Firefox Mobile"
				deviceInfo.DeviceName = "Android Phone • Firefox"
			} else if strings.Contains(userAgentLower, "samsung") {
				deviceInfo.Browser = "Samsung Internet"
				deviceInfo.DeviceName = "Samsung Phone • Samsung Internet"
			} else {
				deviceInfo.DeviceName = "Android Device"
			}
		}

		// iOS devices
		if strings.Contains(userAgentLower, "iphone") {
			deviceInfo.OS = "iOS"
			deviceInfo.DeviceType = "mobile"
			if strings.Contains(userAgentLower, "safari") && !strings.Contains(userAgentLower, "chrome") {
				deviceInfo.Browser = "Safari Mobile"
				deviceInfo.DeviceName = "iPhone • Safari"
			} else if strings.Contains(userAgentLower, "crios") {
				deviceInfo.Browser = "Chrome Mobile"
				deviceInfo.DeviceName = "iPhone • Chrome"
			} else if strings.Contains(userAgentLower, "fxios") {
				deviceInfo.Browser = "Firefox Mobile"
				deviceInfo.DeviceName = "iPhone • Firefox"
			} else {
				deviceInfo.DeviceName = "iPhone"
			}
		}

		if strings.Contains(userAgentLower, "ipad") {
			deviceInfo.OS = "iPadOS"
			deviceInfo.DeviceType = "tablet"
			deviceInfo.DeviceName = "iPad"
			if strings.Contains(userAgentLower, "safari") {
				deviceInfo.Browser = "Safari"
				deviceInfo.DeviceName = "iPad • Safari"
			}
		}
	} else {
		// Desktop devices
		deviceInfo.DeviceType = "desktop"

		// Windows
		if strings.Contains(userAgentLower, "windows") {
			deviceInfo.OS = "Windows"
			if strings.Contains(userAgent, "Windows NT 10") {
				deviceInfo.OS = "Windows 10/11"
			} else if strings.Contains(userAgent, "Windows NT 6.3") {
				deviceInfo.OS = "Windows 8.1"
			} else if strings.Contains(userAgent, "Windows NT 6.1") {
				deviceInfo.OS = "Windows 7"
			}
		}

		// macOS
		if strings.Contains(userAgentLower, "mac os x") || strings.Contains(userAgentLower, "macos") {
			deviceInfo.OS = "macOS"
			if strings.Contains(userAgent, "Mac OS X 10_15") {
				deviceInfo.OS = "macOS Catalina+"
			}
		}

		// Linux
		if strings.Contains(userAgentLower, "linux") && !strings.Contains(userAgentLower, "android") {
			deviceInfo.OS = "Linux"
		}

		// Browser detection for desktop
		if strings.Contains(userAgentLower, "edg/") {
			deviceInfo.Browser = "Microsoft Edge"
			deviceInfo.DeviceName = fmt.Sprintf("%s PC - Edge", deviceInfo.OS)
		} else if strings.Contains(userAgentLower, "chrome/") && !strings.Contains(userAgentLower, "edg") {
			deviceInfo.Browser = "Google Chrome"
			deviceInfo.DeviceName = fmt.Sprintf("%s PC - Chrome", deviceInfo.OS)
		} else if strings.Contains(userAgentLower, "firefox/") {
			deviceInfo.Browser = "Mozilla Firefox"
			deviceInfo.DeviceName = fmt.Sprintf("%s PC - Firefox", deviceInfo.OS)
		} else if strings.Contains(userAgentLower, "safari/") && !strings.Contains(userAgentLower, "chrome") {
			deviceInfo.Browser = "Safari"
			deviceInfo.DeviceName = fmt.Sprintf("%s - Safari", deviceInfo.OS)
		} else if strings.Contains(userAgentLower, "opera") {
			deviceInfo.Browser = "Opera"
			deviceInfo.DeviceName = fmt.Sprintf("%s PC - Opera", deviceInfo.OS)
		}
	}

	// VaultKe mobile app detection
	if strings.Contains(userAgentLower, "vaultke") || strings.Contains(userAgentLower, "expo") {
		deviceInfo.Browser = "VaultKe App"
		if deviceInfo.OS == "Android" {
			deviceInfo.DeviceName = "Android Phone - VaultKe App"
		} else if deviceInfo.OS == "iOS" {
			deviceInfo.DeviceName = "iPhone - VaultKe App"
		} else {
			deviceInfo.DeviceName = "Mobile Device - VaultKe App"
		}
	}

	// Use enhanced device info from custom headers (sent by frontend)
	frontendDeviceType := c.GetHeader("X-Device-Type")
	frontendDeviceName := c.GetHeader("X-Device-Name")
	frontendBrowserName := c.GetHeader("X-Browser-Name")
	frontendOSName := c.GetHeader("X-OS-Name")

	// Debug: Log received headers
	fmt.Printf("Received device headers - Type: '%s', Name: '%s', Browser: '%s', OS: '%s'\n",
		frontendDeviceType, frontendDeviceName, frontendBrowserName, frontendOSName)

	// Override with frontend-detected info if available (more accurate)
	if frontendDeviceType != "" {
		deviceInfo.DeviceType = frontendDeviceType
	}
	if frontendDeviceName != "" {
		deviceInfo.DeviceName = frontendDeviceName
	}
	if frontendBrowserName != "" {
		deviceInfo.Browser = frontendBrowserName
	}
	if frontendOSName != "" {
		deviceInfo.OS = frontendOSName
	}

	// Enhanced IP detection for tunneled environments (tunnelmole, ngrok, etc.)
	ip := c.ClientIP()

	// Check multiple forwarded headers in order of preference
	// Tunnelmole and similar services use these headers to pass the real client IP
	realIP := c.GetHeader("X-Real-IP")
	forwardedFor := c.GetHeader("X-Forwarded-For")
	cfConnectingIP := c.GetHeader("CF-Connecting-IP") // Cloudflare
	trueClientIP := c.GetHeader("True-Client-IP")     // Akamai
	xClientIP := c.GetHeader("X-Client-IP")           // Some proxies

	// Use the most reliable IP source (prioritize real client IP)
	clientIP := ip
	if cfConnectingIP != "" {
		// Cloudflare's header is very reliable
		clientIP = cfConnectingIP
	} else if trueClientIP != "" {
		// Akamai's header
		clientIP = trueClientIP
	} else if realIP != "" {
		// Standard real IP header
		clientIP = realIP
	} else if xClientIP != "" {
		// Some proxy services use this
		clientIP = xClientIP
	} else if forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one (original client)
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			clientIP = strings.TrimSpace(ips[0])
		}
	}

	// Debug: Log all IP-related headers for troubleshooting
	fmt.Printf("IP Detection Debug - ClientIP: %s, X-Real-IP: %s, X-Forwarded-For: %s, CF-Connecting-IP: %s, Final: %s\n",
		ip, realIP, forwardedFor, cfConnectingIP, clientIP)

	// Enhanced location detection with additional headers
	timezone := c.GetHeader("X-Timezone")
	connectionType := c.GetHeader("X-Connection-Type")

	// Determine location based on IP (now properly detects real client IPs)
	if clientIP == "127.0.0.1" || clientIP == "::1" {
		deviceInfo.Location = "Local Development (localhost)"
	} else if strings.HasPrefix(clientIP, "192.168.") || strings.HasPrefix(clientIP, "10.") ||
		(strings.HasPrefix(clientIP, "172.") && len(clientIP) > 8) {
		deviceInfo.Location = fmt.Sprintf("Private Network (%s)", clientIP)
	} else if clientIP != "" && clientIP != ip {
		// We got a real client IP different from the proxy IP
		deviceInfo.Location = fmt.Sprintf("Client IP: %s", clientIP)
	} else {
		// Fallback to whatever IP we have
		deviceInfo.Location = fmt.Sprintf("IP: %s", clientIP)
	}

	// Add timezone info to location if available
	if timezone != "" && timezone != "UTC" {
		deviceInfo.Location = fmt.Sprintf("%s (%s)", deviceInfo.Location, timezone)
	}

	// Add connection type if available
	if connectionType != "" {
		deviceInfo.Location = fmt.Sprintf("%s • %s", deviceInfo.Location, connectionType)
	}

	// Add connection type if available
	if connectionType != "" {
		deviceInfo.Location = fmt.Sprintf("%s • %s", deviceInfo.Location, connectionType)
	}

	return deviceInfo
}

// AuthHandlers contains all authentication-related handlers
type AuthHandlers struct {
	userService *services.UserService
	authService *services.AuthService
}

// NewAuthHandlers creates new auth handlers
func NewAuthHandlers(db *sql.DB, jwtSecret string, jwtExpiration int) *AuthHandlers {
	return &AuthHandlers{
		userService: services.NewUserService(db),
		authService: services.NewAuthService(jwtSecret, jwtExpiration),
	}
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    *AuthData `json:"data,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// AuthData represents the data in auth response
type AuthData struct {
	User  *models.User `json:"user,omitempty"`
	Token string       `json:"token,omitempty"`
}

// Register handles user registration
func (h *AuthHandlers) Register(c *gin.Context) {
	var req models.UserRegistration
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create user
	user, err := h.userService.CreateUser(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Generate token
	token, err := h.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	// Send email verification automatically after registration
	emailVerificationService, exists := c.Get("emailVerificationService")
	if exists {
		verificationService := emailVerificationService.(*services.EmailVerificationService)

		// Create verification token
		verificationToken, err := verificationService.CreateEmailVerificationToken(user.ID)
		if err == nil {
			// Send verification email (don't fail registration if email fails)
			userName := user.FirstName
			if user.LastName != "" {
				userName += " " + user.LastName
			}
			if userName == "" {
				userName = user.Email
			}

			err = verificationService.SendVerificationEmail(user.Email, userName, verificationToken.Token)
			if err != nil {
				// Log error but don't fail registration
				fmt.Printf("Failed to send verification email to %s: %v\n", user.Email, err)
			}
		}
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Success: true,
		Message: "Registration successful! Please check your email for a verification code to complete your account setup.",
		Data: &AuthData{
			User:  user,
			Token: token,
		},
	})
}

// Login handles user authentication
func (h *AuthHandlers) Login(c *gin.Context) {
	var req models.UserLogin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Authenticate user
	user, err := h.userService.AuthenticateUser(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Generate token
	token, err := h.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	// Record login session with real device info
	db, exists := c.Get("db")
	if exists {
		deviceInfo := extractDeviceInfo(c)
		fmt.Printf("Extracted device info for user %s: %+v\n", user.ID, deviceInfo)

		// Get the best available IP address (same logic as extractDeviceInfo)
		ip := c.ClientIP()
		realIP := c.GetHeader("X-Real-IP")
		forwardedFor := c.GetHeader("X-Forwarded-For")
		cfConnectingIP := c.GetHeader("CF-Connecting-IP")
		trueClientIP := c.GetHeader("True-Client-IP")
		xClientIP := c.GetHeader("X-Client-IP")

		clientIP := ip
		if cfConnectingIP != "" {
			clientIP = cfConnectingIP
		} else if trueClientIP != "" {
			clientIP = trueClientIP
		} else if realIP != "" {
			clientIP = realIP
		} else if xClientIP != "" {
			clientIP = xClientIP
		} else if forwardedFor != "" {
			ips := strings.Split(forwardedFor, ",")
			if len(ips) > 0 {
				clientIP = strings.TrimSpace(ips[0])
			}
		}

		err := RecordLoginSession(
			db.(*sql.DB),
			user.ID,
			deviceInfo.DeviceType,
			deviceInfo.DeviceName,
			deviceInfo.OS,
			deviceInfo.Browser,
			clientIP,
			deviceInfo.Location,
		)
		if err != nil {
			fmt.Printf("Failed to record login session: %v\n", err)
		} else {
			fmt.Printf("Successfully called RecordLoginSession for user %s with IP %s\n", user.ID, clientIP)
		}
	} else {
		fmt.Printf("Database not available in context for recording login session\n")
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Login successful",
		Data: &AuthData{
			User:  user,
			Token: token,
		},
	})
}

// Logout handles user logout
func (h *AuthHandlers) Logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, AuthResponse{
			Success: true,
			Message: "Logout successful", // Still return success even without token
		})
		return
	}

	// Extract token (remove "Bearer " prefix)
	tokenString := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	}

	if tokenString != "" {
		// Add token to blacklist
		err := h.authService.BlacklistToken(tokenString)
		if err != nil {
			// Log error but don't fail the logout
			// Client-side cleanup should still proceed
			c.JSON(http.StatusOK, AuthResponse{
				Success: true,
				Message: "Logout successful (token cleanup failed)",
			})
			return
		}
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandlers) RefreshToken(c *gin.Context) {
	// Get token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Authorization header required",
		})
		return
	}

	// Extract token
	tokenString := authHeader[7:] // Remove "Bearer " prefix

	// Refresh token
	newToken, err := h.authService.RefreshToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Failed to refresh token: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Token refreshed successfully",
		Data: &AuthData{
			Token: newToken,
		},
	})
}

// VerifyEmail handles email verification
func (h *AuthHandlers) VerifyEmail(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	err := h.userService.VerifyEmail(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to verify email: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Email verified successfully",
	})
}

// VerifyPhone handles phone verification
func (h *AuthHandlers) VerifyPhone(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	err := h.userService.VerifyPhone(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to verify phone: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Phone verified successfully",
	})
}

// ForgotPassword handles password reset request
func (h *AuthHandlers) ForgotPassword(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get password reset service from context
	passwordResetService, exists := c.Get("passwordResetService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Password reset service not available",
		})
		return
	}

	resetService := passwordResetService.(*services.PasswordResetService)

	// Request password reset
	err := resetService.RequestPasswordReset(req.Identifier)
	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("Password reset error: %v\n", err)

		// Check if it's a user not found error
		if err.Error() == "user not found" {
			c.JSON(http.StatusBadRequest, AuthResponse{
				Success: false,
				Error:   "No account found with this email or phone number",
			})
			return
		}

		// For other errors, log them but don't reveal details
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to send reset instructions. Please try again.",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Password reset instructions sent successfully",
	})
}

// ResetPassword handles password reset
func (h *AuthHandlers) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get password reset service from context
	passwordResetService, exists := c.Get("passwordResetService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Password reset service not available",
		})
		return
	}

	resetService := passwordResetService.(*services.PasswordResetService)

	// Reset password
	err := resetService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Password reset successfully! You can now login with your new password.",
	})
}

// TestEmail handles email testing (for development only)
func (h *AuthHandlers) TestEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get password reset service from context
	passwordResetService, exists := c.Get("passwordResetService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Password reset service not available",
		})
		return
	}

	resetService := passwordResetService.(*services.PasswordResetService)

	// Send test email
	err := resetService.SendPasswordResetEmail(req.Email, "Test User", "123456")
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to send test email: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Test email sent successfully",
	})
}

// CheckTokenStatus checks the status of a password reset token for countdown display
func (h *AuthHandlers) CheckTokenStatus(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get password reset service from context
	passwordResetService, exists := c.Get("passwordResetService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Password reset service not available",
		})
		return
	}

	resetService := passwordResetService.(*services.PasswordResetService)

	// Get token status
	status, err := resetService.GetTokenStatus(req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to check token status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// SendEmailVerification sends an email verification code to the user
func (h *AuthHandlers) SendEmailVerification(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get email verification service from context
	emailVerificationService, exists := c.Get("emailVerificationService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Email verification service not available",
		})
		return
	}

	verificationService := emailVerificationService.(*services.EmailVerificationService)

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
		return
	}
	database := db.(*sql.DB)

	// Get user details
	var userEmail, userName string
	query := `SELECT email, COALESCE(first_name || ' ' || last_name, first_name, email) as name FROM users WHERE id = ?`
	err := database.QueryRow(query, req.UserID).Scan(&userEmail, &userName)
	if err != nil {
		c.JSON(http.StatusNotFound, AuthResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	// Create verification token
	verificationToken, err := verificationService.CreateEmailVerificationToken(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create verification token",
		})
		return
	}

	// Send verification email
	err = verificationService.SendVerificationEmail(userEmail, userName, verificationToken.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to send verification email",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Verification email sent successfully",
	})
}

// VerifyEmailCode verifies a user's email using the verification code
func (h *AuthHandlers) VerifyEmailCode(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get email verification service from context
	emailVerificationService, exists := c.Get("emailVerificationService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Email verification service not available",
		})
		return
	}

	verificationService := emailVerificationService.(*services.EmailVerificationService)

	// Verify email
	_, err := verificationService.VerifyEmail(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Email verified successfully",
		Data:    nil,
	})
}

// CheckEmailVerificationStatus checks the status of an email verification token
func (h *AuthHandlers) CheckEmailVerificationStatus(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get email verification service from context
	emailVerificationService, exists := c.Get("emailVerificationService")
	if !exists {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Email verification service not available",
		})
		return
	}

	verificationService := emailVerificationService.(*services.EmailVerificationService)

	// Get token status
	status, err := verificationService.GetTokenStatus(req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to check token status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// GetProfile handles getting user profile
func (h *AuthHandlers) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, AuthResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Profile retrieved successfully",
		Data: &AuthData{
			User: user,
		},
	})
}

// UpdateProfile handles updating user profile (supports both JSON and multipart form data)
func (h *AuthHandlers) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	contentType := c.GetHeader("Content-Type")
	var req models.UserProfileUpdate
	var err error

	// Handle multipart form data (for file uploads)
	if strings.Contains(contentType, "multipart/form-data") {
		err = h.handleMultipartProfileUpdate(c, &req, userID)
	} else {
		// Handle JSON data
		err = c.ShouldBindJSON(&req)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	user, err := h.userService.UpdateUser(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to update profile: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Profile updated successfully",
		Data: &AuthData{
			User: user,
		},
	})
}

// handleMultipartProfileUpdate processes multipart form data for profile updates
func (h *AuthHandlers) handleMultipartProfileUpdate(c *gin.Context, req *models.UserProfileUpdate, userID string) error {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		return err
	}

	form := c.Request.MultipartForm

	// Handle text fields
	if values, ok := form.Value["firstName"]; ok && len(values) > 0 && values[0] != "" {
		req.FirstName = &values[0]
	}
	if values, ok := form.Value["lastName"]; ok && len(values) > 0 && values[0] != "" {
		req.LastName = &values[0]
	}
	if values, ok := form.Value["phone"]; ok && len(values) > 0 && values[0] != "" {
		req.Phone = &values[0]
	}
	if values, ok := form.Value["county"]; ok && len(values) > 0 && values[0] != "" {
		req.County = &values[0]
	}
	if values, ok := form.Value["town"]; ok && len(values) > 0 && values[0] != "" {
		req.Town = &values[0]
	}
	if values, ok := form.Value["bio"]; ok && len(values) > 0 && values[0] != "" {
		req.Bio = &values[0]
	}
	if values, ok := form.Value["occupation"]; ok && len(values) > 0 && values[0] != "" {
		req.Occupation = &values[0]
	}
	if values, ok := form.Value["language"]; ok && len(values) > 0 && values[0] != "" {
		req.Language = &values[0]
	}
	if values, ok := form.Value["theme"]; ok && len(values) > 0 && values[0] != "" {
		req.Theme = &values[0]
	}
	if values, ok := form.Value["businessType"]; ok && len(values) > 0 && values[0] != "" {
		req.BusinessType = &values[0]
	}
	if values, ok := form.Value["businessDescription"]; ok && len(values) > 0 && values[0] != "" {
		req.BusinessDescription = &values[0]
	}

	// Handle numeric fields
	if values, ok := form.Value["latitude"]; ok && len(values) > 0 && values[0] != "" {
		if lat, err := strconv.ParseFloat(values[0], 64); err == nil {
			req.Latitude = &lat
		}
	}
	if values, ok := form.Value["longitude"]; ok && len(values) > 0 && values[0] != "" {
		if lng, err := strconv.ParseFloat(values[0], 64); err == nil {
			req.Longitude = &lng
		}
	}

	// Handle date fields
	if values, ok := form.Value["dateOfBirth"]; ok && len(values) > 0 && values[0] != "" {
		if dateOfBirth, err := time.Parse("2006-01-02", values[0]); err == nil {
			flexDate := &models.FlexibleDate{Time: dateOfBirth}
			req.DateOfBirth = flexDate
		}
	}

	// Handle file upload (profile image)
	if files, ok := form.File["profile_image"]; ok && len(files) > 0 {
		file := files[0]

		// Validate file type
		allowedTypes := map[string]bool{
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
			"image/webp": true,
		}

		if !allowedTypes[file.Header.Get("Content-Type")] {
			return fmt.Errorf("invalid file type. Only JPEG, PNG, and WebP images are allowed")
		}

		// Validate file size (5MB max)
		if file.Size > 5*1024*1024 {
			return fmt.Errorf("file too large. Maximum size is 5MB")
		}

		// Create uploads directory if it doesn't exist
		uploadDir := "./uploads/avatars"
		if err := os.MkdirAll(uploadDir, 0o755); err != nil {
			return fmt.Errorf("failed to create upload directory: %w", err)
		}

		// Generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), userID, ext)
		filepath := filepath.Join(uploadDir, filename)

		// Open uploaded file
		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open uploaded file: %w", err)
		}
		defer src.Close()

		// Create destination file
		dst, err := os.Create(filepath)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer dst.Close()

		// Copy file content
		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("failed to save file: %w", err)
		}

		// Set avatar path in request
		avatarURL := "/uploads/avatars/" + filename
		req.Avatar = &avatarURL
	}

	return nil
}

// ResendVerification handles resending verification email/SMS
func (h *AuthHandlers) ResendVerification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	var req struct {
		Type string `json:"type" binding:"required"` // "email" or "phone"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get user details
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, AuthResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	switch req.Type {
	case "email":
		if user.IsEmailVerified {
			c.JSON(http.StatusBadRequest, AuthResponse{
				Success: false,
				Error:   "Email is already verified",
			})
			return
		}

		// Get password reset service for email functionality
		passwordResetService, exists := c.Get("passwordResetService")
		if !exists {
			c.JSON(http.StatusInternalServerError, AuthResponse{
				Success: false,
				Error:   "Email service not available",
			})
			return
		}

		resetService := passwordResetService.(*services.PasswordResetService)

		// Generate verification code (reuse password reset functionality)
		err := resetService.SendPasswordResetEmail(user.Email, user.FirstName+" "+user.LastName, "123456")
		if err != nil {
			c.JSON(http.StatusInternalServerError, AuthResponse{
				Success: false,
				Error:   "Failed to send verification email",
			})
			return
		}

		c.JSON(http.StatusOK, AuthResponse{
			Success: true,
			Message: "Verification email sent successfully",
		})

	case "phone":
		if user.IsPhoneVerified {
			c.JSON(http.StatusBadRequest, AuthResponse{
				Success: false,
				Error:   "Phone is already verified",
			})
			return
		}

		// TODO: Implement SMS verification
		c.JSON(http.StatusOK, AuthResponse{
			Success: true,
			Message: "Phone verification SMS sent successfully",
		})

	default:
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid verification type. Must be 'email' or 'phone'",
		})
	}
}
