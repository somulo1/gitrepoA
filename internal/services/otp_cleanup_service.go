package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// OTPCleanupService handles cleanup of expired OTP tokens
type OTPCleanupService struct {
	db *sql.DB
}

// NewOTPCleanupService creates a new OTP cleanup service
func NewOTPCleanupService(db *sql.DB) *OTPCleanupService {
	return &OTPCleanupService{
		db: db,
	}
}

// StartCleanupScheduler starts the background cleanup scheduler
func (s *OTPCleanupService) StartCleanupScheduler() {
	// Run cleanup every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.CleanupExpiredTokens()
			}
		}
	}()

	log.Println("🧹 OTP cleanup scheduler started - running every 30 seconds")
}

// CleanupExpiredTokens removes all expired tokens from both tables
func (s *OTPCleanupService) CleanupExpiredTokens() {
	now := time.Now()

	// Clean up expired email verification tokens
	emailResult, err := s.db.Exec(`
		DELETE FROM email_verification_tokens 
		WHERE expires_at < ? OR used = TRUE
	`, now)

	if err != nil {
		log.Printf("❌ Error cleaning up email verification tokens: %v", err)
	} else {
		emailRowsAffected, _ := emailResult.RowsAffected()
		if emailRowsAffected > 0 {
			log.Printf("🧹 Cleaned up %d expired/used email verification tokens", emailRowsAffected)
		}
	}

	// Clean up expired password reset tokens
	passwordResult, err := s.db.Exec(`
		DELETE FROM password_reset_tokens 
		WHERE expires_at < ? OR used = TRUE
	`, now)

	if err != nil {
		log.Printf("❌ Error cleaning up password reset tokens: %v", err)
	} else {
		passwordRowsAffected, _ := passwordResult.RowsAffected()
		if passwordRowsAffected > 0 {
			log.Printf("🧹 Cleaned up %d expired/used password reset tokens", passwordRowsAffected)
		}
	}
}

// GetTokenStats returns statistics about current tokens
func (s *OTPCleanupService) GetTokenStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Email verification token stats
	var emailTotal, emailExpired, emailUsed int

	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM email_verification_tokens
	`).Scan(&emailTotal)
	if err != nil {
		return nil, fmt.Errorf("failed to get email token total: %w", err)
	}

	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM email_verification_tokens WHERE expires_at < ?
	`, time.Now()).Scan(&emailExpired)
	if err != nil {
		return nil, fmt.Errorf("failed to get email token expired count: %w", err)
	}

	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM email_verification_tokens WHERE used = TRUE
	`).Scan(&emailUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to get email token used count: %w", err)
	}

	// Password reset token stats
	var passwordTotal, passwordExpired, passwordUsed int

	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM password_reset_tokens
	`).Scan(&passwordTotal)
	if err != nil {
		return nil, fmt.Errorf("failed to get password token total: %w", err)
	}

	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM password_reset_tokens WHERE expires_at < ?
	`, time.Now()).Scan(&passwordExpired)
	if err != nil {
		return nil, fmt.Errorf("failed to get password token expired count: %w", err)
	}

	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM password_reset_tokens WHERE used = TRUE
	`).Scan(&passwordUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to get password token used count: %w", err)
	}

	stats["email_verification"] = map[string]int{
		"total":   emailTotal,
		"expired": emailExpired,
		"used":    emailUsed,
		"active":  emailTotal - emailExpired - emailUsed,
	}

	stats["password_reset"] = map[string]int{
		"total":   passwordTotal,
		"expired": passwordExpired,
		"used":    passwordUsed,
		"active":  passwordTotal - passwordExpired - passwordUsed,
	}

	return stats, nil
}

// ForceCleanupUserTokens removes all tokens for a specific user
func (s *OTPCleanupService) ForceCleanupUserTokens(userID string) error {
	// Clean up email verification tokens
	_, err := s.db.Exec(`DELETE FROM email_verification_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to cleanup user email tokens: %w", err)
	}

	// Clean up password reset tokens
	_, err = s.db.Exec(`DELETE FROM password_reset_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to cleanup user password tokens: %w", err)
	}

	log.Printf("🧹 Force cleaned up all tokens for user: %s", userID)
	return nil
}

// ValidateTokenNotExpired checks if a token is not expired (generic function)
func (s *OTPCleanupService) ValidateTokenNotExpired(tableName, token string) (bool, error) {
	var expiresAt time.Time
	var used bool

	query := fmt.Sprintf(`
		SELECT expires_at, used FROM %s WHERE token = ?
	`, tableName)

	err := s.db.QueryRow(query, token).Scan(&expiresAt, &used)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("token not found")
		}
		return false, fmt.Errorf("failed to validate token: %w", err)
	}

	if used {
		return false, fmt.Errorf("token has already been used")
	}

	if time.Now().After(expiresAt) {
		return false, fmt.Errorf("token has expired")
	}

	return true, nil
}

// GetUserActiveTokens returns all active tokens for a user
func (s *OTPCleanupService) GetUserActiveTokens(userID string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get email verification tokens
	emailRows, err := s.db.Query(`
		SELECT token, expires_at, created_at, trial_count 
		FROM email_verification_tokens 
		WHERE user_id = ? AND expires_at > ? AND used = FALSE
	`, userID, time.Now())

	if err != nil {
		return nil, fmt.Errorf("failed to get email tokens: %w", err)
	}
	defer emailRows.Close()

	var emailTokens []map[string]interface{}
	for emailRows.Next() {
		var token string
		var expiresAt, createdAt time.Time
		var trialCount int

		err := emailRows.Scan(&token, &expiresAt, &createdAt, &trialCount)
		if err != nil {
			continue
		}

		emailTokens = append(emailTokens, map[string]interface{}{
			"token":             token,
			"expires_at":        expiresAt,
			"created_at":        createdAt,
			"trial_count":       trialCount,
			"remaining_seconds": int(expiresAt.Sub(time.Now()).Seconds()),
		})
	}

	// Get password reset tokens
	passwordRows, err := s.db.Query(`
		SELECT token, expires_at, created_at, trial_count 
		FROM password_reset_tokens 
		WHERE user_id = ? AND expires_at > ? AND used = FALSE
	`, userID, time.Now())

	if err != nil {
		return nil, fmt.Errorf("failed to get password tokens: %w", err)
	}
	defer passwordRows.Close()

	var passwordTokens []map[string]interface{}
	for passwordRows.Next() {
		var token string
		var expiresAt, createdAt time.Time
		var trialCount int

		err := passwordRows.Scan(&token, &expiresAt, &createdAt, &trialCount)
		if err != nil {
			continue
		}

		passwordTokens = append(passwordTokens, map[string]interface{}{
			"token":             token,
			"expires_at":        expiresAt,
			"created_at":        createdAt,
			"trial_count":       trialCount,
			"remaining_seconds": int(expiresAt.Sub(time.Now()).Seconds()),
		})
	}

	result["email_verification"] = emailTokens
	result["password_reset"] = passwordTokens

	return result, nil
}
