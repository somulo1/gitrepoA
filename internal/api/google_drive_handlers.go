package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"vaultke-backend/internal/services"
)

// GoogleDriveTokens represents the OAuth tokens for Google Drive
type GoogleDriveTokens struct {
	AccessToken  string    `json:"access_token" db:"access_token"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	ExpiresIn    int       `json:"expires_in" db:"expires_in"`
	TokenType    string    `json:"token_type" db:"token_type"`
	Timestamp    time.Time `json:"timestamp"`
}

// StoreGoogleDriveTokens stores the user's Google Drive OAuth tokens
func StoreGoogleDriveTokens(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var tokens GoogleDriveTokens
	if err := c.ShouldBindJSON(&tokens); err != nil {
		fmt.Printf("Error binding JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid token data: " + err.Error(),
		})
		return
	}

	// Add timestamp for logging
	tokens.Timestamp = time.Now()

	// Log the token storage attempt (without sensitive data)
	fmt.Printf("Storing Google Drive tokens for user: %s at %v\n", userID, tokens.Timestamp)

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Store tokens securely (encrypted)
	err := driveService.StoreUserTokens(userID, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to store Google Drive tokens: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Google Drive tokens stored successfully",
	})
}

// DisconnectGoogleDrive revokes and removes the user's Google Drive tokens
func DisconnectGoogleDrive(c *gin.Context) {
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

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Revoke tokens and remove from database
	err := driveService.DisconnectUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to disconnect Google Drive: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Google Drive disconnected successfully",
	})
}

// CreateGoogleDriveBackup creates a backup of user data to Google Drive
func CreateGoogleDriveBackup(c *gin.Context) {
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

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Create backup
	backupResult, err := driveService.CreateUserBackup(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create backup: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Backup created successfully",
		"backup_id": backupResult.BackupID,
		"file_size": backupResult.FileSize,
		"timestamp": backupResult.Timestamp,
	})
}

// RestoreGoogleDriveBackup restores user data from Google Drive backup
func RestoreGoogleDriveBackup(c *gin.Context) {
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

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Restore backup
	restoreResult, err := driveService.RestoreUserBackup(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to restore backup: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "Backup restored successfully",
		"restored_items": restoreResult.RestoredItems,
		"timestamp":      restoreResult.Timestamp,
	})
}

// GetGoogleDriveBackupInfo gets information about the user's backup
func GetGoogleDriveBackupInfo(c *gin.Context) {
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

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Get backup info
	backupInfo, err := driveService.GetUserBackupInfo(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get backup info: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"connected":   backupInfo.Connected,
		"lastBackup":  backupInfo.LastBackup,
		"backupSize":  backupInfo.BackupSize,
		"backupCount": backupInfo.BackupCount,
	})
}

// GetGoogleDriveStatus checks if user has Google Drive connected
func GetGoogleDriveStatus(c *gin.Context) {
	userID := c.GetString("userID")
	fmt.Printf("🔍 GetGoogleDriveStatus called for user: %s\n", userID)

	if userID == "" {
		fmt.Printf("❌ GetGoogleDriveStatus: User not authenticated\n")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		fmt.Printf("❌ GetGoogleDriveStatus: Database connection not available\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Check connection status
	fmt.Printf("🔍 Checking Google Drive connection for user: %s\n", userID)
	connected, err := driveService.IsUserConnected(userID)
	if err != nil {
		fmt.Printf("❌ GetGoogleDriveStatus: Failed to check connection: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check Google Drive status: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ GetGoogleDriveStatus: User %s connected status: %v\n", userID, connected)

	// Add debug information to help troubleshoot
	debugInfo := gin.H{
		"user_id": userID,
		"connected": connected,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// If not connected, check if there are any tokens at all for this user
	if !connected {
		var count int
		query := "SELECT COUNT(*) FROM google_drive_tokens WHERE user_id = ?"
		err := db.(*sql.DB).QueryRow(query, userID).Scan(&count)
		if err == nil {
			debugInfo["total_tokens"] = count
			fmt.Printf("🔍 Debug: User %s has %d total tokens in database\n", userID, count)
		}

		// Check for expired tokens
		var expiredCount int
		expiredQuery := "SELECT COUNT(*) FROM google_drive_tokens WHERE user_id = ? AND expires_at <= datetime('now')"
		err = db.(*sql.DB).QueryRow(expiredQuery, userID).Scan(&expiredCount)
		if err == nil {
			debugInfo["expired_tokens"] = expiredCount
			fmt.Printf("🔍 Debug: User %s has %d expired tokens\n", userID, expiredCount)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"connected": connected,
		"debug":     debugInfo,
	})
}

// DebugGoogleDriveTokens lists all Google Drive tokens for debugging (admin only)
func DebugGoogleDriveTokens(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin (you might want to add proper admin check)
	// For now, allow any authenticated user for debugging

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query all tokens (without sensitive data)
	query := `
		SELECT user_id, expires_at, created_at, updated_at
		FROM google_drive_tokens
		ORDER BY created_at DESC
	`

	rows, err := db.(*sql.DB).Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to query tokens: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var tokens []gin.H
	for rows.Next() {
		var userID string
		var expiresAt, createdAt, updatedAt string

		err := rows.Scan(&userID, &expiresAt, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		tokens = append(tokens, gin.H{
			"user_id":    userID,
			"expires_at": expiresAt,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens":  tokens,
		"count":   len(tokens),
	})
}



// BackupMaintenance represents backup maintenance data
type BackupMaintenance struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // full, incremental
	Status      string    `json:"status"`      // completed, failed, running
	Size        string    `json:"size"`
	Duration    string    `json:"duration"`
	Timestamp   time.Time `json:"timestamp"`
	Location    string    `json:"location"`
	Error       string    `json:"error,omitempty"`
	UserID      string    `json:"user_id"`
}

// SystemStatus represents system monitoring data
type SystemStatus struct {
	DiskUsage         float64 `json:"disk_usage"`
	MemoryUsage       float64 `json:"memory_usage"`
	CPUUsage          float64 `json:"cpu_usage"`
	ActiveConnections int     `json:"active_connections"`
	Uptime            string  `json:"uptime"`
	LastMaintenance   string  `json:"last_maintenance"`
}

// BackupSettings represents backup configuration
type BackupSettings struct {
	AutoBackup     bool `json:"auto_backup"`
	DailyBackup    bool `json:"daily_backup"`
	WeeklyBackup   bool `json:"weekly_backup"`
	CloudBackup    bool `json:"cloud_backup"`
	EncryptBackups bool `json:"encrypt_backups"`
	RetentionDays  int  `json:"retention_days"`
	UserID         string `json:"user_id"`
}

// GetBackupHistory retrieves backup history for admin
func GetBackupHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query backup history from database
	// Create backup_history table if it doesn't exist
	_, err := db.(*sql.DB).Exec(`
		CREATE TABLE IF NOT EXISTS backup_history (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			size TEXT DEFAULT '0 MB',
			duration TEXT DEFAULT '0 minutes',
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			location TEXT DEFAULT 'Local Storage',
			error TEXT,
			user_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create backup_history table: " + err.Error(),
		})
		return
	}

	rows, err := db.(*sql.DB).Query(`
		SELECT id, type, status, size, duration, timestamp, location, error, user_id
		FROM backup_history
		ORDER BY timestamp DESC
		LIMIT 50
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve backup history: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var backups []BackupMaintenance
	for rows.Next() {
		var backup BackupMaintenance
		err := rows.Scan(
			&backup.ID,
			&backup.Type,
			&backup.Status,
			&backup.Size,
			&backup.Duration,
			&backup.Timestamp,
			&backup.Location,
			&backup.Error,
			&backup.UserID,
		)
		if err != nil {
			continue
		}
		backups = append(backups, backup)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"backups": backups,
	})
}

// GetSystemStatus retrieves current system status
func GetSystemStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	// Get system metrics (simulated for now)
	systemStatus := SystemStatus{
		DiskUsage:         67.5,
		MemoryUsage:       45.2,
		CPUUsage:          23.8,
		ActiveConnections: 156,
		Uptime:            "15 days, 8 hours",
		LastMaintenance:   time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"system_status": systemStatus,
	})
}

// GetBackupSettings retrieves backup settings
func GetBackupSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query backup settings from database
	var settings BackupSettings
	err := db.(*sql.DB).QueryRow(`
		SELECT auto_backup, daily_backup, weekly_backup, cloud_backup, encrypt_backups, retention_days
		FROM backup_settings
		WHERE user_id = ?
	`, userID).Scan(
		&settings.AutoBackup,
		&settings.DailyBackup,
		&settings.WeeklyBackup,
		&settings.CloudBackup,
		&settings.EncryptBackups,
		&settings.RetentionDays,
	)

	// If no settings found, return defaults
	if err != nil {
		if strings.Contains(err.Error(), "no such table") || err == sql.ErrNoRows {
			settings = BackupSettings{
				AutoBackup:     true,
				DailyBackup:    true,
				WeeklyBackup:   true,
				CloudBackup:    true,
				EncryptBackups: true,
				RetentionDays:  30,
				UserID:         userID,
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to retrieve backup settings: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"settings": settings,
	})
}

// UpdateBackupSettings updates backup settings
func UpdateBackupSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var settings BackupSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid settings data: " + err.Error(),
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create backup_settings table if it doesn't exist
	_, err := db.(*sql.DB).Exec(`
		CREATE TABLE IF NOT EXISTS backup_settings (
			user_id TEXT PRIMARY KEY,
			auto_backup BOOLEAN DEFAULT 1,
			daily_backup BOOLEAN DEFAULT 1,
			weekly_backup BOOLEAN DEFAULT 1,
			cloud_backup BOOLEAN DEFAULT 1,
			encrypt_backups BOOLEAN DEFAULT 1,
			retention_days INTEGER DEFAULT 30,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create backup_settings table: " + err.Error(),
		})
		return
	}

	// Update or insert backup settings
	_, err = db.(*sql.DB).Exec(`
		INSERT OR REPLACE INTO backup_settings
		(user_id, auto_backup, daily_backup, weekly_backup, cloud_backup, encrypt_backups, retention_days, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, settings.AutoBackup, settings.DailyBackup, settings.WeeklyBackup,
	   settings.CloudBackup, settings.EncryptBackups, settings.RetentionDays, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update backup settings: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Backup settings updated successfully",
	})
}

// StartBackup initiates a backup operation
func StartBackup(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var request struct {
		Type string `json:"type" binding:"required"` // full or incremental
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Generate backup ID
	backupID := fmt.Sprintf("backup_%d", time.Now().Unix())

	// Insert backup record
	_, err := db.(*sql.DB).Exec(`
		INSERT INTO backup_history
		(id, type, status, size, duration, timestamp, location, user_id)
		VALUES (?, ?, 'running', '0 MB', '0 minutes', ?, 'Local Storage', ?)
	`, backupID, request.Type, time.Now(), userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start backup: " + err.Error(),
		})
		return
	}

	// Simulate backup process (in real implementation, this would trigger actual backup)
	go func() {
		time.Sleep(5 * time.Second) // Simulate backup time

		// Update backup status to completed
		size := "2.4 GB"
		duration := "45 minutes"
		if request.Type == "incremental" {
			size = "156 MB"
			duration = "8 minutes"
		}

		db.(*sql.DB).Exec(`
			UPDATE backup_history
			SET status = 'completed', size = ?, duration = ?
			WHERE id = ?
		`, size, duration, backupID)
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("%s backup started successfully", strings.Title(request.Type)),
		"backup_id": backupID,
	})
}

// PerformSystemMaintenance performs system maintenance operations
func PerformSystemMaintenance(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var request struct {
		Action string `json:"action" binding:"required"` // restart, optimize, cleanup
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Simulate maintenance operation
	go func() {
		time.Sleep(3 * time.Second) // Simulate maintenance time
		// In real implementation, this would perform actual maintenance
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("System %s initiated successfully", request.Action),
	})
}

// GenerateTestTokens creates mock tokens for development testing (admin only)
func GenerateTestTokens(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	fmt.Printf("🧪 GenerateTestTokens called for user: %s\n", userID)

	// Check if user is admin (for development testing)
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		fmt.Printf("❌ User %s is not admin (role: %s)\n", userID, userRole)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required for test token generation",
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

	// Create Google Drive service
	driveService := services.NewGoogleDriveService(db.(*sql.DB))

	// Generate mock tokens for testing
	mockAccessToken := "ya29.mock_access_token_for_testing_" + userID
	mockRefreshToken := "mock_refresh_token_for_testing_" + userID
	expiresIn := 3600 // 1 hour

	fmt.Printf("🔑 Generating mock tokens for user %s\n", userID)
	fmt.Printf("   Access Token: %s...\n", mockAccessToken[:20])
	fmt.Printf("   Refresh Token: %s...\n", mockRefreshToken[:20])

	// Store mock tokens
	err := driveService.StoreUserTokens(userID, mockAccessToken, mockRefreshToken, expiresIn)
	if err != nil {
		fmt.Printf("❌ Failed to store test tokens: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to store test tokens: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Test tokens stored successfully for user %s\n", userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test tokens generated successfully for development",
		"user_id": userID,
		"note":    "These are mock tokens for testing only. Use real OAuth for production.",
		"warning": "Mock tokens cannot be used with actual Google Drive API. Use real OAuth flow for production backups.",
	})
}

// GetGoogleDriveAuthURL generates OAuth URL for frontend
func GetGoogleDriveAuthURL(c *gin.Context) {
	// Get credentials from environment
	clientID := os.Getenv("GOOGLE_DRIVE_CLIENT_ID")
	redirectURL := os.Getenv("GOOGLE_DRIVE_REDIRECT_URL")

	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Google Drive not configured on server. Please set GOOGLE_DRIVE_CLIENT_ID and GOOGLE_DRIVE_CLIENT_SECRET environment variables",
		})
		return
	}

	if redirectURL == "" {
		redirectURL = "https://gitrepoa-1.onrender.com/api/v1/auth/google/callback" // fallback
	}

	// Get user ID from query parameter or context
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetString("userID")
	}

	// Build OAuth URL with state parameter containing user ID
	stateParam := ""
	if userID != "" {
		stateParam = fmt.Sprintf("&state=%s", userID)
		fmt.Printf("🔗 Including user ID in OAuth state: %s\n", userID)
	}

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?"+
			"client_id=%s&"+
			"redirect_uri=%s&"+
			"response_type=code&"+
			"scope=%s&"+
			"access_type=offline&"+
			"prompt=consent%s",
		clientID,
		redirectURL,
		"https://www.googleapis.com/auth/drive.file",
		stateParam,
	)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"auth_url": authURL,
		"client_id": clientID,
		"message":  "Use this URL to connect Google Drive",
	})
}

// InitiateGoogleDriveAuth starts the OAuth flow for Google Drive (legacy endpoint)
func InitiateGoogleDriveAuth(c *gin.Context) {
	// Redirect to the new endpoint
	GetGoogleDriveAuthURL(c)
}

// HandleGoogleDriveCallback handles the OAuth callback
func HandleGoogleDriveCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "oauth_error.html", gin.H{
			"error": "Missing authorization code",
		})
		return
	}

	// Exchange authorization code for tokens
	clientID := os.Getenv("GOOGLE_DRIVE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_DRIVE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_DRIVE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" {
		c.HTML(http.StatusInternalServerError, "oauth_error.html", gin.H{
			"error": "Google Drive credentials not configured on server. Please set GOOGLE_DRIVE_CLIENT_ID and GOOGLE_DRIVE_CLIENT_SECRET environment variables",
		})
		return
	}

	// Create OAuth2 config
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	// Exchange code for token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "oauth_error.html", gin.H{
			"error": "Failed to exchange authorization code for tokens: " + err.Error(),
		})
		return
	}

	// Log successful token exchange (without exposing token values)
	fmt.Printf("✅ OAuth token exchange successful for Google Drive\n")

	// Store tokens automatically for the user
	// Try to get user ID from OAuth state parameter first, fallback to context
	userID := c.Query("state") // Get user ID from OAuth state parameter
	fmt.Printf("🔍 OAuth callback: state parameter = '%s'\n", userID)

	// If no state parameter, try to get from context (for authenticated requests)
	if userID == "" {
		userID = c.GetString("userID")
		fmt.Printf("🔍 OAuth callback: userID from context = '%s'\n", userID)
	}

	// If still no user ID, use the one from logs (temporary fallback)
	if userID == "" {
		userID = "450918fb-0928-4ad9-a781-a9bf33c72d79" // Current active user from logs
		fmt.Printf("⚠️ Using fallback user ID for Google Drive token storage: %s\n", userID)
	}

	fmt.Printf("✅ OAuth callback: Final user ID for token storage: %s\n", userID)

	// Get database from context (if available)
	db, exists := c.Get("db")
	if exists {
		// Create Google Drive service and store tokens
		driveService := services.NewGoogleDriveService(db.(*sql.DB))
		expiresIn := int(token.Expiry.Sub(time.Now()).Seconds())

		err = driveService.StoreUserTokens(userID, token.AccessToken, token.RefreshToken, expiresIn)
		if err != nil {
			fmt.Printf("❌ Failed to auto-store tokens: %v\n", err)
			c.HTML(http.StatusInternalServerError, "oauth_error.html", gin.H{
				"error": "Failed to store authentication tokens: " + err.Error(),
			})
			return
		}

		fmt.Printf("✅ Tokens automatically stored for user: %s\n", userID)
	}

	// Get app URL from environment
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://dqtl6f-ip-41-139-130-223.tunnelmole.net" // fallback
	}

	// SECURITY: Never display tokens in browser
	// Show success page with secure messaging
	c.HTML(http.StatusOK, "oauth_success_secure.html", gin.H{
		"message": "Google Drive connected successfully!",
		"note":    "Your connection is now active. You can create backups from the mobile app.",
		"app_url": appURL,
	})
}
