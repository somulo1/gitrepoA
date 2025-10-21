package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Privacy Settings structures
type PrivacySettings struct {
	ProfileVisibility  string `json:"profile_visibility" db:"profile_visibility"`
	TransactionPrivacy bool   `json:"transaction_privacy" db:"transaction_privacy"`
	LocationSharing    bool   `json:"location_sharing" db:"location_sharing"`
}

// Security Settings structures
type SecuritySettings struct {
	BiometricLogin           bool `json:"biometric_login" db:"biometric_login"`
	TwoFactorAuth            bool `json:"two_factor_auth" db:"two_factor_auth"`
	AutoLogout               bool `json:"auto_logout" db:"auto_logout"`
	LoginNotifications       bool `json:"login_notifications" db:"login_notifications"`
	SuspiciousActivityAlerts bool `json:"suspicious_activity_alerts" db:"suspicious_activity_alerts"`
	DeviceManagement         bool `json:"device_management" db:"device_management"`
}

// User Preferences structures
type UserPreferences struct {
	Language   string `json:"language" db:"language"`
	Currency   string `json:"currency" db:"currency"`
	DateFormat string `json:"date_format" db:"date_format"`
}

// GetPrivacySettings retrieves user privacy settings
func GetPrivacySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Ensure table exists
	if err := ensurePrivacySettingsTable(db.(*sql.DB)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize privacy settings",
		})
		return
	}

	// Get privacy settings
	var settings PrivacySettings
	query := `
		SELECT profile_visibility, transaction_privacy, location_sharing
		FROM user_privacy_settings
		WHERE user_id = ?
	`

	err := db.(*sql.DB).QueryRow(query, userID).Scan(
		&settings.ProfileVisibility,
		&settings.TransactionPrivacy,
		&settings.LocationSharing,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings if none exist
			settings = PrivacySettings{
				ProfileVisibility:  "chama_members",
				TransactionPrivacy: true,
				LocationSharing:    false,
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to retrieve privacy settings",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// UpdatePrivacySettings updates user privacy settings
func UpdatePrivacySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var settings PrivacySettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Ensure table exists
	if err := ensurePrivacySettingsTable(db.(*sql.DB)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize privacy settings",
		})
		return
	}

	// Update or insert privacy settings
	query := `
		INSERT INTO user_privacy_settings (user_id, profile_visibility, transaction_privacy, location_sharing, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET
		profile_visibility = excluded.profile_visibility,
		transaction_privacy = excluded.transaction_privacy,
		location_sharing = excluded.location_sharing,
		updated_at = datetime('now')
	`

	_, err := db.(*sql.DB).Exec(query, userID, settings.ProfileVisibility, settings.TransactionPrivacy, settings.LocationSharing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update privacy settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Privacy settings updated successfully",
	})
}

// GetSecuritySettings retrieves user security settings
func GetSecuritySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Ensure table exists
	if err := ensureSecuritySettingsTable(db.(*sql.DB)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize security settings",
		})
		return
	}

	// Get security settings
	var settings SecuritySettings
	query := `
		SELECT biometric_login, two_factor_auth, auto_logout, login_notifications, 
		       suspicious_activity_alerts, device_management
		FROM user_security_settings
		WHERE user_id = ?
	`

	err := db.(*sql.DB).QueryRow(query, userID).Scan(
		&settings.BiometricLogin,
		&settings.TwoFactorAuth,
		&settings.AutoLogout,
		&settings.LoginNotifications,
		&settings.SuspiciousActivityAlerts,
		&settings.DeviceManagement,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings if none exist
			settings = SecuritySettings{
				BiometricLogin:           false,
				TwoFactorAuth:            false,
				AutoLogout:               true,
				LoginNotifications:       true,
				SuspiciousActivityAlerts: true,
				DeviceManagement:         true,
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to retrieve security settings",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// UpdateSecuritySettings updates user security settings
func UpdateSecuritySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var settings SecuritySettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Ensure table exists
	if err := ensureSecuritySettingsTable(db.(*sql.DB)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize security settings",
		})
		return
	}

	// Update or insert security settings
	query := `
		INSERT INTO user_security_settings (user_id, biometric_login, two_factor_auth, auto_logout, 
		                                   login_notifications, suspicious_activity_alerts, device_management, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET
		biometric_login = excluded.biometric_login,
		two_factor_auth = excluded.two_factor_auth,
		auto_logout = excluded.auto_logout,
		login_notifications = excluded.login_notifications,
		suspicious_activity_alerts = excluded.suspicious_activity_alerts,
		device_management = excluded.device_management,
		updated_at = datetime('now')
	`

	_, err := db.(*sql.DB).Exec(query, userID, settings.BiometricLogin, settings.TwoFactorAuth,
		settings.AutoLogout, settings.LoginNotifications, settings.SuspiciousActivityAlerts, settings.DeviceManagement)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update security settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Security settings updated successfully",
	})
}

// GetUserPreferences retrieves user preferences
func GetUserPreferences(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Ensure table exists
	if err := ensureUserPreferencesTable(db.(*sql.DB)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize user preferences",
		})
		return
	}

	// Get user preferences
	var preferences UserPreferences
	query := `
		SELECT language, currency, date_format
		FROM user_preferences
		WHERE user_id = ?
	`

	err := db.(*sql.DB).QueryRow(query, userID).Scan(
		&preferences.Language,
		&preferences.Currency,
		&preferences.DateFormat,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences if none exist
			preferences = UserPreferences{
				Language:   "en",
				Currency:   "KES",
				DateFormat: "dd/mm/yyyy",
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to retrieve user preferences",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    preferences,
	})
}

// UpdateUserPreferences updates user preferences
func UpdateUserPreferences(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var preferences UserPreferences
	if err := c.ShouldBindJSON(&preferences); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Ensure table exists
	if err := ensureUserPreferencesTable(db.(*sql.DB)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize user preferences",
		})
		return
	}

	// Update or insert user preferences
	query := `
		INSERT INTO user_preferences (user_id, language, currency, date_format, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET
		language = excluded.language,
		currency = excluded.currency,
		date_format = excluded.date_format,
		updated_at = datetime('now')
	`

	_, err := db.(*sql.DB).Exec(query, userID, preferences.Language, preferences.Currency, preferences.DateFormat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update user preferences",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User preferences updated successfully",
	})
}

// Table creation helper functions

// ensurePrivacySettingsTable creates the user_privacy_settings table if it doesn't exist
func ensurePrivacySettingsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS user_privacy_settings (
			user_id TEXT PRIMARY KEY,
			profile_visibility TEXT NOT NULL DEFAULT 'chama_members',
			transaction_privacy BOOLEAN NOT NULL DEFAULT true,
			location_sharing BOOLEAN NOT NULL DEFAULT false,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		)
	`
	_, err := db.Exec(query)
	return err
}

// ensureSecuritySettingsTable creates the user_security_settings table if it doesn't exist
func ensureSecuritySettingsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS user_security_settings (
			user_id TEXT PRIMARY KEY,
			biometric_login BOOLEAN NOT NULL DEFAULT false,
			two_factor_auth BOOLEAN NOT NULL DEFAULT false,
			auto_logout BOOLEAN NOT NULL DEFAULT true,
			login_notifications BOOLEAN NOT NULL DEFAULT true,
			suspicious_activity_alerts BOOLEAN NOT NULL DEFAULT true,
			device_management BOOLEAN NOT NULL DEFAULT true,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		)
	`
	_, err := db.Exec(query)
	return err
}

// ensureUserPreferencesTable creates the user_preferences table if it doesn't exist
func ensureUserPreferencesTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS user_preferences (
			user_id TEXT PRIMARY KEY,
			language TEXT NOT NULL DEFAULT 'en',
			currency TEXT NOT NULL DEFAULT 'KES',
			date_format TEXT NOT NULL DEFAULT 'dd/mm/yyyy',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		)
	`
	_, err := db.Exec(query)
	return err
}
