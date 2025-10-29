package middleware

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// OTPValidationMiddleware provides middleware for OTP validation
type OTPValidationMiddleware struct {
	db *sql.DB
}

// NewOTPValidationMiddleware creates a new OTP validation middleware
func NewOTPValidationMiddleware(db *sql.DB) *OTPValidationMiddleware {
	return &OTPValidationMiddleware{
		db: db,
	}
}

// ValidateEmailVerificationToken validates email verification token before processing
func (m *OTPValidationMiddleware) ValidateEmailVerificationToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token string `json:"token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Token is required",
			})
			c.Abort()
			return
		}

		// Check if token exists and is valid
		var userID string
		var expiresAt time.Time
		var used bool
		var trialCount int

		err := m.db.QueryRow(`
			SELECT user_id, expires_at, used, trial_count 
			FROM email_verification_tokens 
			WHERE token = ?
		`, req.Token).Scan(&userID, &expiresAt, &used, &trialCount)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid verification token",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Failed to validate token",
				})
			}
			c.Abort()
			return
		}

		// Check if token is already used
		if used {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Token has already been used",
			})
			c.Abort()
			return
		}

		// Check if token has expired
		if time.Now().After(expiresAt) {
			// Clean up expired token
			m.db.Exec("DELETE FROM email_verification_tokens WHERE token = ?", req.Token)
			
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Token has expired. Please request a new verification email.",
			})
			c.Abort()
			return
		}

		// Check trial count
		if trialCount >= 3 {
			// Clean up token with too many attempts
			m.db.Exec("DELETE FROM email_verification_tokens WHERE token = ?", req.Token)
			
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Maximum verification attempts exceeded. Please request a new verification email.",
			})
			c.Abort()
			return
		}

		// Store token info in context for use by handlers
		c.Set("token", req.Token)
		c.Set("user_id", userID)
		c.Set("trial_count", trialCount)
		c.Set("expires_at", expiresAt)

		c.Next()
	}
}

// ValidatePasswordResetToken validates password reset token before processing
func (m *OTPValidationMiddleware) ValidatePasswordResetToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token       string `json:"token" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Token and new password are required",
			})
			c.Abort()
			return
		}

		// Check if token exists and is valid
		var userID string
		var expiresAt time.Time
		var used bool
		var trialCount int

		err := m.db.QueryRow(`
			SELECT user_id, expires_at, used, trial_count 
			FROM password_reset_tokens 
			WHERE token = ?
		`, req.Token).Scan(&userID, &expiresAt, &used, &trialCount)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid reset token",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Failed to validate token",
				})
			}
			c.Abort()
			return
		}

		// Check if token is already used
		if used {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Token has already been used",
			})
			c.Abort()
			return
		}

		// Check if token has expired
		if time.Now().After(expiresAt) {
			// Clean up expired token
			m.db.Exec("DELETE FROM password_reset_tokens WHERE token = ?", req.Token)
			
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Token has expired. Please request a new password reset.",
			})
			c.Abort()
			return
		}

		// Check trial count
		if trialCount >= 3 {
			// Clean up token with too many attempts
			m.db.Exec("DELETE FROM password_reset_tokens WHERE token = ?", req.Token)
			
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Maximum reset attempts exceeded. Please request a new password reset.",
			})
			c.Abort()
			return
		}

		// Store token info in context for use by handlers
		c.Set("token", req.Token)
		c.Set("user_id", userID)
		c.Set("new_password", req.NewPassword)
		c.Set("trial_count", trialCount)
		c.Set("expires_at", expiresAt)

		c.Next()
	}
}

// IncrementTrialCount increments the trial count for a token
func (m *OTPValidationMiddleware) IncrementTrialCount(tableName, token string) error {
	query := fmt.Sprintf(`
		UPDATE %s 
		SET trial_count = trial_count + 1 
		WHERE token = ?
	`, tableName)
	
	_, err := m.db.Exec(query, token)
	return err
}

// GetTokenStatus returns the current status of a token
func (m *OTPValidationMiddleware) GetTokenStatus(tableName, token string) (map[string]interface{}, error) {
	var userID string
	var expiresAt, createdAt time.Time
	var used bool
	var trialCount int

	query := fmt.Sprintf(`
		SELECT user_id, expires_at, created_at, used, trial_count 
		FROM %s 
		WHERE token = ?
	`, tableName)

	err := m.db.QueryRow(query, token).Scan(&userID, &expiresAt, &createdAt, &used, &trialCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return map[string]interface{}{
				"valid": false,
				"error": "Token not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get token status: %w", err)
	}

	now := time.Now()
	
	// Check if expired
	if now.After(expiresAt) {
		return map[string]interface{}{
			"valid": false,
			"error": "Token has expired",
		}, nil
	}

	// Check if used
	if used {
		return map[string]interface{}{
			"valid": false,
			"error": "Token has already been used",
		}, nil
	}

	// Check trial count
	if trialCount >= 3 {
		return map[string]interface{}{
			"valid": false,
			"error": "Maximum attempts exceeded",
		}, nil
	}

	// Calculate remaining time
	remainingSeconds := int(expiresAt.Sub(now).Seconds())
	
	return map[string]interface{}{
		"valid":            true,
		"user_id":          userID,
		"remaining_seconds": remainingSeconds,
		"trial_count":      trialCount,
		"max_trials":       3,
		"expires_at":       expiresAt.Unix(),
		"created_at":       createdAt.Unix(),
	}, nil
}

// CleanupExpiredTokens removes expired tokens from a specific table
func (m *OTPValidationMiddleware) CleanupExpiredTokens(tableName string) (int64, error) {
	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE expires_at < ? OR used = TRUE
	`, tableName)
	
	result, err := m.db.Exec(query, time.Now())
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}
