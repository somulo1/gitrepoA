package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginSession represents a user login session
type LoginSession struct {
	ID              string    `json:"id" db:"id"`
	UserID          string    `json:"userId" db:"user_id"`
	DeviceType      string    `json:"deviceType" db:"device_type"`
	DeviceName      string    `json:"deviceName" db:"device_name"`
	OperatingSystem string    `json:"operatingSystem" db:"operating_system"`
	Browser         string    `json:"browser" db:"browser"`
	IPAddress       string    `json:"ipAddress" db:"ip_address"`
	Location        string    `json:"location" db:"location"`
	LoginTime       time.Time `json:"loginTime" db:"login_time"`
	LastActivity    time.Time `json:"lastActivity" db:"last_activity"`
	Status          string    `json:"status" db:"status"`
	IsCurrent       bool      `json:"isCurrent" db:"is_current"`
}

// ChangePassword handles password change requests
func ChangePassword(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var requestData struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
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

	// Get current password hash from database
	var currentPasswordHash string
	err := db.(*sql.DB).QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentPasswordHash)
	if err != nil {
		fmt.Printf("Failed to get user password: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify current password",
		})
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(requestData.CurrentPassword))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Current password is incorrect",
		})
		return
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(requestData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Failed to hash new password: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to process new password",
		})
		return
	}

	// Update password in database
	updateQuery := `
		UPDATE users
		SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.(*sql.DB).Exec(updateQuery, string(newPasswordHash), userID)
	if err != nil {
		fmt.Printf("Failed to update password: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update password",
		})
		return
	}

	fmt.Printf("✅ Password changed successfully for user: %s\n", userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// GetLoginHistory retrieves user login history
func GetLoginHistory(c *gin.Context) {
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

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Create login_sessions table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS login_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			device_type TEXT,
			device_name TEXT,
			operating_system TEXT,
			browser TEXT,
			ip_address TEXT,
			location TEXT,
			login_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'active',
			is_current BOOLEAN DEFAULT FALSE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	_, err := db.(*sql.DB).Exec(createTableQuery)
	if err != nil {
		fmt.Printf("Failed to create login_sessions table: %v\n", err)
	}

	// Get login sessions
	query := `
		SELECT 
			id, user_id, device_type, device_name, operating_system, 
			browser, ip_address, location, login_time, last_activity, 
			status, is_current
		FROM login_sessions 
		WHERE user_id = ? 
		ORDER BY login_time DESC 
		LIMIT ? OFFSET ?
	`

	rows, err := db.(*sql.DB).Query(query, userID, limit, offset)
	if err != nil {
		fmt.Printf("Failed to get login history: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve login history",
		})
		return
	}
	defer rows.Close()

	var sessions []LoginSession
	for rows.Next() {
		var session LoginSession
		err := rows.Scan(
			&session.ID, &session.UserID, &session.DeviceType, &session.DeviceName,
			&session.OperatingSystem, &session.Browser, &session.IPAddress,
			&session.Location, &session.LoginTime, &session.LastActivity,
			&session.Status, &session.IsCurrent,
		)
		if err != nil {
			fmt.Printf("Failed to scan login session: %v\n", err)
			continue
		}
		sessions = append(sessions, session)
	}

	// If no sessions exist, try to create a current session with real device info from headers
	if len(sessions) == 0 {
		// Extract device info from request headers (sent by frontend)
		deviceType := c.GetHeader("X-Device-Type")
		deviceName := c.GetHeader("X-Device-Name")
		browserName := c.GetHeader("X-Browser-Name")
		osName := c.GetHeader("X-OS-Name")
		timezone := c.GetHeader("X-Timezone")

		// Fallback values if headers are not present
		if deviceType == "" {
			deviceType = "unknown"
		}
		if deviceName == "" {
			deviceName = "Current Device"
		}
		if browserName == "" {
			browserName = "VaultKe App"
		}
		if osName == "" {
			osName = "Unknown OS"
		}

		// Determine location based on IP
		ip := c.ClientIP()
		location := "Unknown"
		if ip == "127.0.0.1" || ip == "::1" {
			location = "Local Development"
		} else if ip != "" {
			location = fmt.Sprintf("IP: %s", ip)
		}

		// Add timezone info if available
		if timezone != "" && timezone != "UTC" {
			location = fmt.Sprintf("%s (%s)", location, timezone)
		}

		currentSession := LoginSession{
			ID:              "current-session",
			UserID:          userID,
			DeviceType:      deviceType,
			DeviceName:      deviceName,
			OperatingSystem: osName,
			Browser:         browserName,
			IPAddress:       ip,
			Location:        location,
			LoginTime:       time.Now(),
			LastActivity:    time.Now(),
			Status:          "active",
			IsCurrent:       true,
		}

		// Try to insert this session into the database for future reference
		insertQuery := `
			INSERT OR REPLACE INTO login_sessions
			(id, user_id, device_type, device_name, operating_system, browser, ip_address, location, is_current)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE)
		`
		_, err := db.(*sql.DB).Exec(insertQuery,
			currentSession.ID, currentSession.UserID, currentSession.DeviceType,
			currentSession.DeviceName, currentSession.OperatingSystem, currentSession.Browser,
			currentSession.IPAddress, currentSession.Location)
		if err != nil {
			fmt.Printf("Failed to insert current session: %v\n", err)
		}

		sessions = append(sessions, currentSession)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sessions,
		"meta": gin.H{
			"total":  len(sessions),
			"limit":  limit,
			"offset": offset,
		},
	})
}

// LogoutAllDevices logs out user from all other devices
func LogoutAllDevices(c *gin.Context) {
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

	// Update all sessions except current to revoked status
	updateQuery := `
		UPDATE login_sessions 
		SET status = 'revoked', last_activity = CURRENT_TIMESTAMP 
		WHERE user_id = ? AND is_current = FALSE
	`

	result, err := db.(*sql.DB).Exec(updateQuery, userID)
	if err != nil {
		fmt.Printf("Failed to logout all devices: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to logout from all devices",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Logged out from %d devices for user: %s\n", rowsAffected, userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Logged out from %d other devices", rowsAffected),
	})
}

// LogoutSpecificDevice logs out user from a specific device
func LogoutSpecificDevice(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
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

	// Update specific session to revoked status (only if it belongs to the user)
	updateQuery := `
		UPDATE login_sessions
		SET status = 'revoked', last_activity = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND is_current = FALSE
	`

	result, err := db.(*sql.DB).Exec(updateQuery, sessionID, userID)
	if err != nil {
		fmt.Printf("Failed to logout specific device: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to logout from device",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Session not found or cannot logout from current device",
		})
		return
	}

	fmt.Printf("✅ Logged out from specific device (session: %s) for user: %s\n", sessionID, userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out from device successfully",
	})
}

// RecordLoginSession records a new login session
func RecordLoginSession(db *sql.DB, userID, deviceType, deviceName, os, browser, ipAddress, location string) error {
	// Ensure the table exists first
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS login_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			device_type TEXT,
			device_name TEXT,
			operating_system TEXT,
			browser TEXT,
			ip_address TEXT,
			location TEXT,
			login_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'active',
			is_current BOOLEAN DEFAULT FALSE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	_, err := db.Exec(createTableQuery)
	if err != nil {
		fmt.Printf("Failed to create login_sessions table in RecordLoginSession: %v\n", err)
		return err
	}

	// Mark all previous sessions as not current
	_, err = db.Exec("UPDATE login_sessions SET is_current = FALSE WHERE user_id = ?", userID)
	if err != nil {
		fmt.Printf("Failed to update previous sessions: %v\n", err)
	}

	// Insert new session with debug logging
	insertQuery := `
		INSERT INTO login_sessions
		(user_id, device_type, device_name, operating_system, browser, ip_address, location, is_current)
		VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)
	`

	fmt.Printf("Recording login session for user %s: device=%s, name=%s, os=%s, browser=%s, ip=%s, location=%s\n",
		userID, deviceType, deviceName, os, browser, ipAddress, location)

	_, err = db.Exec(insertQuery, userID, deviceType, deviceName, os, browser, ipAddress, location)
	if err != nil {
		fmt.Printf("Failed to record login session: %v\n", err)
		return err
	}

	fmt.Printf("Successfully recorded login session for user %s\n", userID)
	return nil
}
