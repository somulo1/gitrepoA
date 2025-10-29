package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"
)

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"userId" db:"user_id"`
	Token        string    `json:"token" db:"token"`
	ExpiresAt    time.Time `json:"expiresAt" db:"expires_at"`
	Used         bool      `json:"used" db:"used"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	TrialCount   int       `json:"trialCount" db:"trial_count"`
	SessionID    string    `json:"sessionId" db:"session_id"`
}

// EmailVerificationService handles email verification operations
type EmailVerificationService struct {
	db           *sql.DB
	emailService *EmailService
}

// NewEmailVerificationService creates a new email verification service
func NewEmailVerificationService(db *sql.DB, emailService *EmailService) *EmailVerificationService {
	return &EmailVerificationService{
		db:           db,
		emailService: emailService,
	}
}

// InitializeEmailVerificationTable creates the email verification tokens table
func (s *EmailVerificationService) InitializeEmailVerificationTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS email_verification_tokens (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		user_id TEXT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		used BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		trial_count INTEGER DEFAULT 0,
		session_id TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	)`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create email verification tokens table: %w", err)
	}

	// Create index for faster lookups
	indexQuery := `CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_token ON email_verification_tokens(token)`
	_, err = s.db.Exec(indexQuery)
	if err != nil {
		return fmt.Errorf("failed to create email verification tokens index: %w", err)
	}

	return nil
}

// CreateEmailVerificationToken creates a new email verification token for a user
func (s *EmailVerificationService) CreateEmailVerificationToken(userID string) (*EmailVerificationToken, error) {
	// Generate token
	token, err := s.generateVerificationToken()
	if err != nil {
		return nil, err
	}

	// Set expiry to 2 minutes from now
	expiresAt := time.Now().Add(2 * time.Minute)

	// Generate session ID for this verification attempt
	sessionID := s.generateSessionID()

	// Delete any existing tokens for this user (immediate cleanup)
	_, err = s.db.Exec("DELETE FROM email_verification_tokens WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup existing tokens: %w", err)
	}

	// Insert new token
	query := `
	INSERT INTO email_verification_tokens (user_id, token, expires_at, session_id, trial_count)
	VALUES (?, ?, ?, ?, 0)
	`

	result, err := s.db.Exec(query, userID, token, expiresAt, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create email verification token: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted token ID: %w", err)
	}

	verificationToken := &EmailVerificationToken{
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
	go s.scheduleTokenCleanup(verificationToken.ID, expiresAt)

	return verificationToken, nil
}

// generateVerificationToken generates a 6-digit verification code
func (s *EmailVerificationService) generateVerificationToken() (string, error) {
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

// generateSessionID generates a unique session ID for tracking verification attempts
func (s *EmailVerificationService) generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// scheduleTokenCleanup automatically deletes token after expiry
func (s *EmailVerificationService) scheduleTokenCleanup(tokenID string, expiresAt time.Time) {
	// Wait until expiry time
	time.Sleep(time.Until(expiresAt))
	
	// Delete the expired token
	query := `DELETE FROM email_verification_tokens WHERE id = ?`
	_, err := s.db.Exec(query, tokenID)
	if err != nil {
		fmt.Printf("Failed to auto-cleanup expired verification token %s: %v\n", tokenID, err)
	} else {
		fmt.Printf("Auto-cleaned expired verification token: %s\n", tokenID)
	}
}

// ValidateVerificationToken validates an email verification token and handles trial counting
func (s *EmailVerificationService) ValidateVerificationToken(token string) (*EmailVerificationToken, error) {
	query := `
	SELECT id, user_id, token, expires_at, used, created_at, trial_count, session_id
	FROM email_verification_tokens
	WHERE token = ? AND used = FALSE
	`

	var verificationToken EmailVerificationToken
	err := s.db.QueryRow(query, token).Scan(
		&verificationToken.ID,
		&verificationToken.UserID,
		&verificationToken.Token,
		&verificationToken.ExpiresAt,
		&verificationToken.Used,
		&verificationToken.CreatedAt,
		&verificationToken.TrialCount,
		&verificationToken.SessionID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired verification token")
		}
		return nil, fmt.Errorf("failed to validate verification token: %w", err)
	}

	// Check if token has expired - if so, delete it immediately
	if time.Now().After(verificationToken.ExpiresAt) {
		s.deleteToken(verificationToken.ID)
		return nil, fmt.Errorf("verification token has expired")
	}

	// Check trial count - if 3 or more attempts, delete token and block
	if verificationToken.TrialCount >= 3 {
		s.deleteToken(verificationToken.ID)
		return nil, fmt.Errorf("maximum attempts exceeded - please request a new verification code")
	}

	return &verificationToken, nil
}

// IncrementTrialCount increments the trial count for a token
func (s *EmailVerificationService) IncrementTrialCount(tokenID string) error {
	query := `UPDATE email_verification_tokens SET trial_count = trial_count + 1 WHERE id = ?`
	_, err := s.db.Exec(query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to increment trial count: %w", err)
	}
	return nil
}

// deleteToken immediately deletes a token from database
func (s *EmailVerificationService) deleteToken(tokenID string) error {
	query := `DELETE FROM email_verification_tokens WHERE id = ?`
	_, err := s.db.Exec(query, tokenID)
	if err != nil {
		fmt.Printf("Failed to delete verification token %s: %v\n", tokenID, err)
		return err
	}
	fmt.Printf("Deleted verification token: %s\n", tokenID)
	return nil
}

// UseVerificationToken deletes the token after successful use
func (s *EmailVerificationService) UseVerificationToken(token string) error {
	// Get token ID first
	var tokenID string
	query := `SELECT id FROM email_verification_tokens WHERE token = ?`
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

// SendVerificationEmail sends an email verification email to the user
func (s *EmailVerificationService) SendVerificationEmail(userEmail, userName, verificationToken string) error {
	if s.emailService == nil {
		fmt.Printf("⚠️ Email service is nil - cannot send verification email\n")
		return fmt.Errorf("email service not configured")
	}

	fmt.Printf("📧 Attempting to send verification email to: %s\n", userEmail)
	err := s.emailService.SendEmailVerificationEmail(userEmail, verificationToken, userName)
	if err != nil {
		fmt.Printf("❌ Failed to send verification email: %v\n", err)
		return err
	}

	fmt.Printf("✅ Verification email sent successfully to: %s\n", userEmail)
	return nil
}

// VerifyEmail verifies a user's email using a valid token
func (s *EmailVerificationService) VerifyEmail(token string) (string, error) {
	// Validate token (this also checks expiry and trial count)
	verificationToken, err := s.ValidateVerificationToken(token)
	if err != nil {
		// If validation fails, increment trial count if token exists
		if verificationToken != nil {
			s.IncrementTrialCount(verificationToken.ID)
		}
		return "", err
	}

	// Mark user's email as verified
	query := `UPDATE users SET is_email_verified = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = s.db.Exec(query, verificationToken.UserID)
	if err != nil {
		// Increment trial count on database update failure
		s.IncrementTrialCount(verificationToken.ID)
		return "", fmt.Errorf("failed to verify email: %w", err)
	}

	// Delete token after successful email verification
	err = s.UseVerificationToken(token)
	if err != nil {
		return "", fmt.Errorf("failed to cleanup token: %w", err)
	}

	return verificationToken.UserID, nil
}

// GetTokenStatus returns the current status of a verification token for countdown display
func (s *EmailVerificationService) GetTokenStatus(token string) (map[string]interface{}, error) {
	query := `
	SELECT id, expires_at, trial_count, created_at
	FROM email_verification_tokens
	WHERE token = ? AND used = FALSE
	`

	var tokenID string
	var expiresAt, createdAt time.Time
	var trialCount int

	err := s.db.QueryRow(query, token).Scan(&tokenID, &expiresAt, &trialCount, &createdAt)
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
