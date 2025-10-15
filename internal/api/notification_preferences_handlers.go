package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"vaultke-backend/models"
	"vaultke-backend/services"

	"github.com/gin-gonic/gin"
)

// Helper function to get user ID from context (returns UUID string)
func getUserIDFromContext(c *gin.Context) (string, error) {
	userID := c.GetString("userID")
	if userID == "" {
		return "", fmt.Errorf("user not authenticated")
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		return "", fmt.Errorf("database connection not available")
	}

	// Verify user exists in database
	var count int
	err := db.(*sql.DB).QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	if count == 0 {
		return "", fmt.Errorf("user not found")
	}

	return userID, nil
}

// GetNotificationPreferences gets user notification preferences
func GetNotificationPreferences(c *gin.Context) {
	// Get user ID (returns UUID string)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
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

	// Create notification service
	notificationService := services.NewNotificationService(db.(*sql.DB))

	// Get user preferences
	preferences, err := notificationService.GetUserPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get notification preferences",
		})
		return
	}

	// Get available sounds
	sounds, err := notificationService.GetAvailableSounds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get available sounds",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"preferences":      preferences,
			"available_sounds": sounds,
		},
	})
}

// UpdateNotificationPreferences updates user notification preferences
func UpdateNotificationPreferences(c *gin.Context) {
	// Get user ID (returns UUID string)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Parse request body
	var req models.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
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

	// Create notification service
	notificationService := services.NewNotificationService(db.(*sql.DB))

	// Update preferences
	updatedPreferences, err := notificationService.UpdateUserPreferences(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update notification preferences",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification preferences updated successfully",
		"data": gin.H{
			"preferences": updatedPreferences,
		},
	})
}

// GetAvailableNotificationSounds gets all available notification sounds
func GetAvailableNotificationSounds(c *gin.Context) {
	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create notification service
	notificationService := services.NewNotificationService(db.(*sql.DB))

	// Get available sounds
	sounds, err := notificationService.GetAvailableSounds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get available sounds",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"sounds": sounds,
		},
	})
}

// TestNotificationSound tests a notification sound
func TestNotificationSound(c *gin.Context) {
	// Get user ID (returns UUID string)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Parse request body
	var req struct {
		SoundID int `json:"sound_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
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

	// Create notification service
	notificationService := services.NewNotificationService(db.(*sql.DB))

	// Create a test notification
	testNotification := models.CreateNotificationRequest{
		UserID:   userID,
		Title:    "Test Notification",
		Message:  "This is a test notification to preview your selected sound.",
		Type:     "system",
		Priority: "normal",
		Data: map[string]interface{}{
			"test":     true,
			"sound_id": req.SoundID,
		},
	}

	// Send test notification
	notification, err := notificationService.CreateNotification(testNotification)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send test notification",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test notification sent successfully",
		"data": gin.H{
			"notification_id": notification.ID,
		},
	})
}

// GetNotificationSettings gets comprehensive notification settings (legacy compatibility)
func GetNotificationSettings(c *gin.Context) {
	// Get user ID (returns UUID string)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
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

	// Create notification service
	notificationService := services.NewNotificationService(db.(*sql.DB))

	// Get user preferences
	preferences, err := notificationService.GetUserPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get notification preferences",
		})
		return
	}

	// Convert to legacy format for backward compatibility
	settings := gin.H{
		"pushNotifications":        preferences.SoundEnabled,
		"emailNotifications":       preferences.SystemNotifications, // Map to system notifications
		"smsNotifications":         false,                           // Not implemented yet
		"inAppNotifications":       true,                            // Always enabled
		"chamaNotifications":       preferences.ChamaNotifications,
		"transactionNotifications": preferences.TransactionNotifications,
		"reminderNotifications":    preferences.ReminderNotifications,
		"systemNotifications":      preferences.SystemNotifications,
		"marketingNotifications":   preferences.MarketingNotifications,
		"soundEnabled":             preferences.SoundEnabled,
		"vibrationEnabled":         preferences.VibrationEnabled,
		"volumeLevel":              preferences.VolumeLevel,
		"notificationSoundId":      preferences.NotificationSoundID,
		"quietHoursEnabled":        preferences.QuietHoursEnabled,
		"quietHoursStart":          preferences.QuietHoursStart,
		"quietHoursEnd":            preferences.QuietHoursEnd,
		"timezone":                 preferences.Timezone,
		"notificationFrequency":    preferences.NotificationFrequency,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"settings": settings,
		},
	})
}

// UpdateNotificationSettings updates notification settings (legacy compatibility)
func UpdateNotificationSettings(c *gin.Context) {
	// Get user ID (returns UUID string)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Parse request body
	var legacyReq map[string]interface{}
	if err := c.ShouldBindJSON(&legacyReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Convert legacy format to new format
	req := models.UpdatePreferencesRequest{}

	if val, ok := legacyReq["pushNotifications"].(bool); ok {
		req.SoundEnabled = &val
	}
	if val, ok := legacyReq["chamaNotifications"].(bool); ok {
		req.ChamaNotifications = &val
	}
	if val, ok := legacyReq["transactionNotifications"].(bool); ok {
		req.TransactionNotifications = &val
	}
	if val, ok := legacyReq["reminderNotifications"].(bool); ok {
		req.ReminderNotifications = &val
	}
	if val, ok := legacyReq["systemNotifications"].(bool); ok {
		req.SystemNotifications = &val
	}
	if val, ok := legacyReq["marketingNotifications"].(bool); ok {
		req.MarketingNotifications = &val
	}
	if val, ok := legacyReq["soundEnabled"].(bool); ok {
		req.SoundEnabled = &val
	}
	if val, ok := legacyReq["vibrationEnabled"].(bool); ok {
		req.VibrationEnabled = &val
	}
	if val, ok := legacyReq["volumeLevel"].(float64); ok {
		intVal := int(val)
		req.VolumeLevel = &intVal
	}
	if val, ok := legacyReq["notificationSoundId"].(float64); ok {
		intVal := int(val)
		req.NotificationSoundID = &intVal
	}
	if val, ok := legacyReq["quietHoursEnabled"].(bool); ok {
		req.QuietHoursEnabled = &val
	}
	if val, ok := legacyReq["quietHoursStart"].(string); ok {
		req.QuietHoursStart = val
	}
	if val, ok := legacyReq["quietHoursEnd"].(string); ok {
		req.QuietHoursEnd = val
	}
	if val, ok := legacyReq["timezone"].(string); ok {
		req.Timezone = val
	}
	if val, ok := legacyReq["notificationFrequency"].(string); ok {
		req.NotificationFrequency = val
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

	// Create notification service
	notificationService := services.NewNotificationService(db.(*sql.DB))

	// Update preferences
	updatedPreferences, err := notificationService.UpdateUserPreferences(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update notification settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification settings updated successfully",
		"data": gin.H{
			"preferences": updatedPreferences,
		},
	})
}
