package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// PasswordResetService handles password reset functionality
type PasswordResetService struct {
	db           *sql.DB
	emailService *EmailService
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"userId" db:"user_id"`
	Token        string    `json:"token" db:"token"`
	ExpiresAt    time.Time `json:"expiresAt" db:"expires_at"`
	Used         bool      `json:"used" db:"used"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	TrialCount   int       `json:"trialCount" db:"trial_count"`
	SessionID    string    `json:"sessionId" db:"session_id"`
}

// NewPasswordResetService creates a new password reset service
func NewPasswordResetService(db *sql.DB, emailService *EmailService) *PasswordResetService {
	return &PasswordResetService{
		db:           db,
		emailService: emailService,
	}
}

// InitializePasswordResetTable creates the password reset tokens table if it doesn't exist
func (s *PasswordResetService) InitializePasswordResetTable() error {
	// First, create the table with basic structure if it doesn't exist
	query := `
	CREATE TABLE IF NOT EXISTS password_reset_tokens (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		user_id TEXT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		used BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	)`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create password_reset_tokens table: %w", err)
	}

	// Add new columns if they don't exist (for existing databases)
	alterQueries := []string{
		`ALTER TABLE password_reset_tokens ADD COLUMN trial_count INTEGER DEFAULT 0`,
		`ALTER TABLE password_reset_tokens ADD COLUMN session_id TEXT DEFAULT ''`,
	}

	for _, alterQuery := range alterQueries {
		_, err := s.db.Exec(alterQuery)
		if err != nil {
			// Ignore "duplicate column name" errors - column already exists
			if !strings.Contains(err.Error(), "duplicate column name") &&
			   !strings.Contains(err.Error(), "already exists") {
				fmt.Printf("Warning: Failed to add column to password_reset_tokens: %v\n", err)
			}
		}
	}

	// Create index for faster lookups
	indexQuery := `CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token)`
	_, err = s.db.Exec(indexQuery)
	if err != nil {
		return fmt.Errorf("failed to create password reset tokens index: %w", err)
	}

	return nil
}

// GenerateResetToken generates a secure random token
func (s *PasswordResetService) GenerateResetToken() (string, error) {
	// Generate 6-digit numeric code for user-friendly experience
	bytes := make([]byte, 3)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to 6-digit number
	token := fmt.Sprintf("%06d", int(bytes[0])<<16|int(bytes[1])<<8|int(bytes[2]))
	return token[:6], nil
}

// generateSessionID generates a unique session ID for tracking reset attempts
func (s *PasswordResetService) generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// scheduleTokenCleanup automatically deletes token after expiry
func (s *PasswordResetService) scheduleTokenCleanup(tokenID string, expiresAt time.Time) {
	// Wait until expiry time
	time.Sleep(time.Until(expiresAt))

	// Delete the expired token
	query := `DELETE FROM password_reset_tokens WHERE id = ?`
	_, err := s.db.Exec(query, tokenID)
	if err != nil {
		fmt.Printf("Failed to auto-cleanup expired token %s: %v\n", tokenID, err)
	} else {
		fmt.Printf("Auto-cleaned expired token: %s\n", tokenID)
	}
}

// CreatePasswordResetToken creates a new password reset token for a user
func (s *PasswordResetService) CreatePasswordResetToken(userID string) (*PasswordResetToken, error) {
	// Generate token
	token, err := s.GenerateResetToken()
	if err != nil {
		return nil, err
	}

	// Set expiry to 2 minutes from now
	expiresAt := time.Now().Add(2 * time.Minute)

	// Generate session ID for this reset attempt
	sessionID := s.generateSessionID()

	// Delete any existing tokens for this user (immediate cleanup)
	_, err = s.db.Exec("DELETE FROM password_reset_tokens WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup existing tokens: %w", err)
	}

	// Insert new token - try with new columns first, fallback to basic columns
	query := `
	INSERT INTO password_reset_tokens (user_id, token, expires_at, session_id, trial_count)
	VALUES (?, ?, ?, ?, 0)
	`

	result, err := s.db.Exec(query, userID, token, expiresAt, sessionID)
	if err != nil && strings.Contains(err.Error(), "no column named") {
		// Fallback to basic columns for existing databases
		fmt.Printf("⚠️ Using fallback query for existing database schema\n")
		query = `
		INSERT INTO password_reset_tokens (user_id, token, expires_at)
		VALUES (?, ?, ?)
		`
		result, err = s.db.Exec(query, userID, token, expiresAt)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create password reset token: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted token ID: %w", err)
	}

	resetToken := &PasswordResetToken{
		ID:         fmt.Sprintf("%d", id),
		UserID:     userID,
		Token:      token,
		ExpiresAt:  expiresAt,
		Used:       false,
		CreatedAt:  time.Now(),
		TrialCount: 0,
		SessionID:  sessionID,
	}

	// Start automatic cleanup timer for this token
	go s.scheduleTokenCleanup(resetToken.ID, expiresAt)

	return resetToken, nil
}

// ValidateResetToken validates a password reset token and handles trial counting
func (s *PasswordResetService) ValidateResetToken(token string) (*PasswordResetToken, error) {
	fmt.Printf("🔍 ValidateResetToken called with token: %s\n", token)

	// Try with new columns first
	query := `
	SELECT id, user_id, token, expires_at, used, created_at, trial_count, session_id
	FROM password_reset_tokens
	WHERE token = ? AND used = FALSE
	`

	var resetToken PasswordResetToken
	err := s.db.QueryRow(query, token).Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.Token,
		&resetToken.ExpiresAt,
		&resetToken.Used,
		&resetToken.CreatedAt,
		&resetToken.TrialCount,
		&resetToken.SessionID,
	)

	if err != nil && strings.Contains(err.Error(), "no column named") {
		// Fallback to basic columns for existing databases
		fmt.Printf("⚠️ Using fallback query for existing database schema\n")
		query = `
		SELECT id, user_id, token, expires_at, used, created_at
		FROM password_reset_tokens
		WHERE token = ? AND used = FALSE
		`

		err = s.db.QueryRow(query, token).Scan(
			&resetToken.ID,
			&resetToken.UserID,
			&resetToken.Token,
			&resetToken.ExpiresAt,
			&resetToken.Used,
			&resetToken.CreatedAt,
		)

		// Set default values for missing fields
		resetToken.TrialCount = 0
		resetToken.SessionID = ""
	}

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("❌ Token not found in database: %s\n", token)
			return nil, fmt.Errorf("invalid or expired reset token")
		}
		fmt.Printf("❌ Database error during token validation: %v\n", err)
		return nil, fmt.Errorf("failed to validate reset token: %w", err)
	}

	fmt.Printf("✅ Token found in database: ID=%s, ExpiresAt=%v, TrialCount=%d\n",
		resetToken.ID, resetToken.ExpiresAt, resetToken.TrialCount)

	// Check if token has expired - if so, delete it immediately
	now := time.Now()
	if now.After(resetToken.ExpiresAt) {
		fmt.Printf("❌ Token has expired: now=%v, expiresAt=%v\n", now, resetToken.ExpiresAt)
		s.deleteToken(resetToken.ID)
		return nil, fmt.Errorf("reset token has expired")
	}

	// Check trial count - if 3 or more attempts, delete token and block
	if resetToken.TrialCount >= 3 {
		fmt.Printf("❌ Maximum attempts exceeded: trialCount=%d\n", resetToken.TrialCount)
		s.deleteToken(resetToken.ID)
		return nil, fmt.Errorf("maximum attempts exceeded - please request a new reset code")
	}

	fmt.Printf("✅ Token validation successful: remaining time=%v\n", resetToken.ExpiresAt.Sub(now))

	return &resetToken, nil
}

// IncrementTrialCount increments the trial count for a token
func (s *PasswordResetService) IncrementTrialCount(tokenID string) error {
	query := `UPDATE password_reset_tokens SET trial_count = trial_count + 1 WHERE id = ?`
	_, err := s.db.Exec(query, tokenID)
	if err != nil {
		// If trial_count column doesn't exist, just log and continue
		if strings.Contains(err.Error(), "no column named trial_count") {
			fmt.Printf("⚠️ trial_count column not available, skipping increment\n")
			return nil
		}
		return fmt.Errorf("failed to increment trial count: %w", err)
	}
	return nil
}

// deleteToken immediately deletes a token from database
func (s *PasswordResetService) deleteToken(tokenID string) error {
	query := `DELETE FROM password_reset_tokens WHERE id = ?`
	_, err := s.db.Exec(query, tokenID)
	if err != nil {
		fmt.Printf("Failed to delete token %s: %v\n", tokenID, err)
		return err
	}
	fmt.Printf("Deleted token: %s\n", tokenID)
	return nil
}

// UseResetToken deletes the token after successful use
func (s *PasswordResetService) UseResetToken(token string) error {
	// Get token ID first
	var tokenID string
	query := `SELECT id FROM password_reset_tokens WHERE token = ?`
	err := s.db.QueryRow(query, token).Scan(&tokenID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("token not found")
		}
		return fmt.Errorf("failed to find token: %w", err)
	}

	// Delete the token immediately after successful use
	return s.deleteToken(tokenID)
}

// SendPasswordResetEmail sends a password reset email to the user
func (s *PasswordResetService) SendPasswordResetEmail(userEmail, userName, resetToken string) error {
	if s.emailService == nil {
		fmt.Printf("⚠️ Email service is nil - cannot send password reset email\n")
		return fmt.Errorf("email service not configured")
	}

	fmt.Printf("📧 Attempting to send password reset email to: %s\n", userEmail)
	err := s.emailService.SendPasswordResetEmail(userEmail, resetToken, userName)
	if err != nil {
		fmt.Printf("❌ Failed to send password reset email: %v\n", err)
		return err
	}

	fmt.Printf("✅ Password reset email sent successfully to: %s\n", userEmail)
	return nil
}

// CleanupExpiredTokens removes expired password reset tokens
func (s *PasswordResetService) CleanupExpiredTokens() error {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < ? OR used = TRUE`
	_, err := s.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return nil
}

// GetUserByIdentifier finds a user by email or phone number
func (s *PasswordResetService) GetUserByIdentifier(identifier string) (string, string, string, error) {
	// Clean and normalize the identifier
	identifier = strings.TrimSpace(identifier)

	var userID, email, name string
	var query string

	// Check if identifier looks like an email
	if strings.Contains(identifier, "@") {
		// Normalize email for case-insensitive comparison
		normalizedEmail := normalizeEmail(identifier)
		fmt.Printf("🔍 Password reset: original='%s' -> normalized='%s'\n", identifier, normalizedEmail)

		// Use both direct comparison (for new normalized emails) and LOWER() for legacy data
		query = `SELECT id, email, COALESCE(first_name || ' ' || last_name, first_name, email) as name FROM users WHERE (email = ? OR LOWER(TRIM(email)) = ?) AND status = 'active'`

		// We'll pass the normalized email twice for both comparisons
		err := s.db.QueryRow(query, normalizedEmail, normalizedEmail).Scan(&userID, &email, &name)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", "", "", fmt.Errorf("user not found")
			}
			return "", "", "", fmt.Errorf("failed to find user: %w", err)
		}
		return userID, email, name, nil
	} else {
		// Assume it's a phone number - format it properly
		identifier = formatPhoneNumber(identifier)
		query = `SELECT id, email, COALESCE(first_name || ' ' || last_name, first_name, phone) as name FROM users WHERE phone = ? AND status = 'active'`
	}

	err := s.db.QueryRow(query, identifier).Scan(&userID, &email, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", fmt.Errorf("user not found")
		}
		return "", "", "", fmt.Errorf("failed to find user: %w", err)
	}

	return userID, email, name, nil
}

// RequestPasswordReset handles the complete password reset request process
func (s *PasswordResetService) RequestPasswordReset(identifier string) error {
	fmt.Printf("🔍 RequestPasswordReset called with identifier: %s\n", identifier)

	// Find user by email or phone
	userID, email, name, err := s.GetUserByIdentifier(identifier)
	if err != nil {
		fmt.Printf("❌ GetUserByIdentifier failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ Found user: ID=%s, Email=%s, Name=%s\n", userID, email, name)

	// Create reset token
	resetToken, err := s.CreatePasswordResetToken(userID)
	if err != nil {
		fmt.Printf("❌ CreatePasswordResetToken failed: %v\n", err)
		return fmt.Errorf("failed to create reset token: %w", err)
	}
	fmt.Printf("✅ Created reset token: %s\n", resetToken.Token)

	// Send email
	err = s.SendPasswordResetEmail(email, name, resetToken.Token)
	if err != nil {
		fmt.Printf("❌ SendPasswordResetEmail failed: %v\n", err)
		return fmt.Errorf("failed to send reset email: %w", err)
	}
	fmt.Printf("✅ Password reset email sent successfully\n")

	return nil
}

// ResetPassword resets a user's password using a valid token
func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	// Validate token (this also checks expiry and trial count)
	resetToken, err := s.ValidateResetToken(token)
	if err != nil {
		// If validation fails, increment trial count if token exists
		if resetToken != nil {
			s.IncrementTrialCount(resetToken.ID)
		}
		return err
	}

	// Hash the new password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		// Increment trial count on password hashing failure
		s.IncrementTrialCount(resetToken.ID)
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update the password_hash column (not password)
	query := `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = s.db.Exec(query, string(hashedPassword), resetToken.UserID)
	if err != nil {
		// Increment trial count on database update failure
		s.IncrementTrialCount(resetToken.ID)
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Delete token after successful password reset
	err = s.UseResetToken(token)
	if err != nil {
		return fmt.Errorf("failed to cleanup token: %w", err)
	}

	return nil
}

// GetTokenStatus returns the current status of a token for countdown display
func (s *PasswordResetService) GetTokenStatus(token string) (map[string]interface{}, error) {
	// Try with new columns first
	query := `
	SELECT id, expires_at, trial_count, created_at
	FROM password_reset_tokens
	WHERE token = ? AND used = FALSE
	`

	var tokenID string
	var expiresAt, createdAt time.Time
	var trialCount int

	err := s.db.QueryRow(query, token).Scan(&tokenID, &expiresAt, &trialCount, &createdAt)
	if err != nil && strings.Contains(err.Error(), "no column named") {
		// Fallback to basic columns for existing databases
		fmt.Printf("⚠️ Using fallback query for token status\n")
		query = `
		SELECT id, expires_at, created_at
		FROM password_reset_tokens
		WHERE token = ? AND used = FALSE
		`

		err = s.db.QueryRow(query, token).Scan(&tokenID, &expiresAt, &createdAt)
		trialCount = 0 // Default value for missing column
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return map[string]interface{}{
				"valid": false,
				"error": "OTP not found or expired",
			}, nil
		}
		return nil, fmt.Errorf("failed to get token status: %w", err)
	}

	now := time.Now()

	// Check if expired
	if now.After(expiresAt) {
		// Delete expired token
		s.deleteToken(tokenID)
		return map[string]interface{}{
			"valid": false,
			"error": "Token has expired",
		}, nil
	}

	// Check trial count
	if trialCount >= 3 {
		// Delete token with too many attempts
		s.deleteToken(tokenID)
		return map[string]interface{}{
			"valid": false,
			"error": "Maximum attempts exceeded",
		}, nil
	}

	// Calculate remaining time
	remainingSeconds := int(expiresAt.Sub(now).Seconds())

	return map[string]interface{}{
		"valid":            true,
		"remainingSeconds": remainingSeconds,
		"trialCount":       trialCount,
		"maxTrials":        3,
		"expiresAt":        expiresAt.Unix(),
		"createdAt":        createdAt.Unix(),
	}, nil
}

// normalizeEmail normalizes an email address for consistent comparison
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// formatPhoneNumber formats a phone number for consistent storage and comparison
func formatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")

	// Add Kenya country code if it's a local number
	if len(cleaned) == 10 && strings.HasPrefix(cleaned, "0") {
		cleaned = "254" + cleaned[1:]
	} else if len(cleaned) == 9 {
		cleaned = "254" + cleaned
	}

	return cleaned
}
