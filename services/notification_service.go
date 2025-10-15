package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"vaultke-backend/models"
)

// NotificationService handles all notification operations
type NotificationService struct {
	db *sql.DB
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{db: db}
}

// CreateNotification creates and sends a notification
func (ns *NotificationService) CreateNotification(req models.CreateNotificationRequest) (*models.Notification, error) {
	// Validate request
	if err := ns.validateNotificationRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get user preferences
	preferences, err := ns.GetUserPreferences(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	// Check if user wants this type of notification
	if !ns.shouldSendNotification(req.Type, preferences) {
		log.Printf("Notification skipped due to user preferences: user_id=%d, type=%s", req.UserID, req.Type)
		return nil, nil
	}

	// Set defaults
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if req.ScheduledFor == nil {
		now := time.Now()
		req.ScheduledFor = &now
	}

	// Convert data to JSON
	var dataJSON []byte
	if req.Data != nil {
		dataJSON, _ = json.Marshal(req.Data)
	}

	// Create notification
	query := `
		INSERT INTO notifications 
		(user_id, title, message, type, priority, category, reference_type, reference_id, 
		 scheduled_for, data, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
	`

	now := time.Now()
	result, err := ns.db.Exec(query,
		req.UserID, req.Title, req.Message, req.Type, req.Priority,
		req.Category, req.ReferenceType, req.ReferenceID,
		req.ScheduledFor, string(dataJSON), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification ID: %w", err)
	}

	// Get the created notification
	notification, err := ns.GetNotificationByID(int(id))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created notification: %w", err)
	}

	// Schedule immediate delivery if not scheduled for later
	if req.ScheduledFor.Before(time.Now()) || req.ScheduledFor.Equal(time.Now()) {
		go ns.DeliverNotification(notification, preferences)
	}

	log.Printf("Notification created successfully: id=%d, user_id=%d, type=%s", id, req.UserID, req.Type)
	return notification, nil
}

// CreateFromTemplate creates a notification from a template
func (ns *NotificationService) CreateFromTemplate(templateName string, userID string, variables map[string]interface{}) (*models.Notification, error) {
	// Get template
	template, err := ns.getTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Replace template variables
	title := ns.replaceTemplateVariables(template.TitleTemplate, variables)
	message := ns.replaceTemplateVariables(template.MessageTemplate, variables)

	// Create notification request
	req := models.CreateNotificationRequest{
		UserID:   userID,
		Title:    title,
		Message:  message,
		Type:     template.Type,
		Priority: template.DefaultPriority,
		Category: template.Category,
		Data:     variables,
	}

	return ns.CreateNotification(req)
}

// DeliverNotification delivers a notification to the user
func (ns *NotificationService) DeliverNotification(notification *models.Notification, preferences *models.UserNotificationPreferences) error {
	if preferences == nil {
		var err error
		preferences, err = ns.GetUserPreferences(notification.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user preferences: %w", err)
		}
	}

	// Check quiet hours
	if ns.isQuietHours(preferences) && notification.Priority != "urgent" {
		log.Printf("Notification delayed due to quiet hours: id=%d", notification.ID)
		return ns.scheduleForLater(notification, preferences)
	}

	// Deliver via different channels
	delivered := false

	// In-app notification (always delivered)
	if err := ns.deliverInApp(notification); err == nil {
		delivered = true
	}

	// Push notification with sound
	if preferences.SoundEnabled || preferences.VibrationEnabled {
		if err := ns.deliverPush(notification, preferences); err == nil {
			delivered = true
		}
	}

	// Update notification status
	if delivered {
		now := time.Now()
		_, err := ns.db.Exec(`
			UPDATE notifications 
			SET status = 'delivered', sent_at = ?, delivered_at = ?, updated_at = ?
			WHERE id = ?
		`, now, now, now, notification.ID)

		if err != nil {
			log.Printf("Failed to update notification status: %v", err)
		}
	}

	return nil
}

// GetUserPreferences gets or creates user notification preferences
func (ns *NotificationService) GetUserPreferences(userID string) (*models.UserNotificationPreferences, error) {
	preferences := &models.UserNotificationPreferences{}

	query := `
		SELECT id, user_id, notification_sound_id, sound_enabled, vibration_enabled, volume_level,
			   chama_notifications, transaction_notifications, reminder_notifications, 
			   system_notifications, marketing_notifications, quiet_hours_enabled,
			   quiet_hours_start, quiet_hours_end, timezone, notification_frequency,
			   priority_only_during_quiet, created_at, updated_at
		FROM user_notification_preferences 
		WHERE user_id = ?
	`

	err := ns.db.QueryRow(query, userID).Scan(
		&preferences.ID, &preferences.UserID, &preferences.NotificationSoundID,
		&preferences.SoundEnabled, &preferences.VibrationEnabled, &preferences.VolumeLevel,
		&preferences.ChamaNotifications, &preferences.TransactionNotifications,
		&preferences.ReminderNotifications, &preferences.SystemNotifications,
		&preferences.MarketingNotifications, &preferences.QuietHoursEnabled,
		&preferences.QuietHoursStart, &preferences.QuietHoursEnd, &preferences.Timezone,
		&preferences.NotificationFrequency, &preferences.PriorityOnlyDuringQuiet,
		&preferences.CreatedAt, &preferences.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create default preferences
		return ns.createDefaultPreferences(userID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	return preferences, nil
}

// UpdateUserPreferences updates user notification preferences
func (ns *NotificationService) UpdateUserPreferences(userID string, req models.UpdatePreferencesRequest) (*models.UserNotificationPreferences, error) {
	// Get existing preferences
	preferences, err := ns.GetUserPreferences(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing preferences: %w", err)
	}

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}

	if req.NotificationSoundID != nil {
		setParts = append(setParts, "notification_sound_id = ?")
		args = append(args, *req.NotificationSoundID)
	}
	if req.SoundEnabled != nil {
		setParts = append(setParts, "sound_enabled = ?")
		args = append(args, *req.SoundEnabled)
	}
	if req.VibrationEnabled != nil {
		setParts = append(setParts, "vibration_enabled = ?")
		args = append(args, *req.VibrationEnabled)
	}
	if req.VolumeLevel != nil {
		setParts = append(setParts, "volume_level = ?")
		args = append(args, *req.VolumeLevel)
	}
	if req.ChamaNotifications != nil {
		setParts = append(setParts, "chama_notifications = ?")
		args = append(args, *req.ChamaNotifications)
	}
	if req.TransactionNotifications != nil {
		setParts = append(setParts, "transaction_notifications = ?")
		args = append(args, *req.TransactionNotifications)
	}
	if req.ReminderNotifications != nil {
		setParts = append(setParts, "reminder_notifications = ?")
		args = append(args, *req.ReminderNotifications)
	}
	if req.SystemNotifications != nil {
		setParts = append(setParts, "system_notifications = ?")
		args = append(args, *req.SystemNotifications)
	}
	if req.MarketingNotifications != nil {
		setParts = append(setParts, "marketing_notifications = ?")
		args = append(args, *req.MarketingNotifications)
	}
	if req.QuietHoursEnabled != nil {
		setParts = append(setParts, "quiet_hours_enabled = ?")
		args = append(args, *req.QuietHoursEnabled)
	}
	if req.QuietHoursStart != "" {
		setParts = append(setParts, "quiet_hours_start = ?")
		args = append(args, req.QuietHoursStart)
	}
	if req.QuietHoursEnd != "" {
		setParts = append(setParts, "quiet_hours_end = ?")
		args = append(args, req.QuietHoursEnd)
	}
	if req.Timezone != "" {
		setParts = append(setParts, "timezone = ?")
		args = append(args, req.Timezone)
	}
	if req.NotificationFrequency != "" {
		setParts = append(setParts, "notification_frequency = ?")
		args = append(args, req.NotificationFrequency)
	}
	if req.PriorityOnlyDuringQuiet != nil {
		setParts = append(setParts, "priority_only_during_quiet = ?")
		args = append(args, *req.PriorityOnlyDuringQuiet)
	}

	if len(setParts) == 0 {
		return preferences, nil // No updates
	}

	// Add updated_at and user_id
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE user_notification_preferences 
		SET %s 
		WHERE user_id = ?
	`, strings.Join(setParts, ", "))

	_, err = ns.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	log.Printf("User notification preferences updated: user_id=%d", userID)

	// Return updated preferences
	return ns.GetUserPreferences(userID)
}

// GetAvailableSounds returns all available notification sounds
func (ns *NotificationService) GetAvailableSounds() ([]models.NotificationSound, error) {
	query := `
		SELECT id, name, file_path, file_size, duration_seconds, is_default, is_active, created_at, updated_at
		FROM notification_sounds 
		WHERE is_active = 1 
		ORDER BY is_default DESC, name ASC
	`

	rows, err := ns.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get available sounds: %w", err)
	}
	defer rows.Close()

	var sounds []models.NotificationSound
	for rows.Next() {
		var sound models.NotificationSound
		err := rows.Scan(
			&sound.ID, &sound.Name, &sound.FilePath, &sound.FileSize,
			&sound.DurationSeconds, &sound.IsDefault, &sound.IsActive,
			&sound.CreatedAt, &sound.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sound: %w", err)
		}
		sounds = append(sounds, sound)
	}

	return sounds, nil
}

// MarkAsRead marks a notification as read
func (ns *NotificationService) MarkAsRead(notificationID int, userID string) error {
	now := time.Now()
	result, err := ns.db.Exec(`
		UPDATE notifications 
		SET is_read = 1, read_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, now, now, notificationID, userID)

	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found or not owned by user")
	}

	return nil
}

// GetUserNotifications gets paginated user notifications
func (ns *NotificationService) GetUserNotifications(userID string, filters map[string]interface{}) (*models.NotificationListResponse, error) {
	// Build WHERE clause
	whereParts := []string{"user_id = ?"}
	args := []interface{}{userID}

	if notifType, ok := filters["type"].(string); ok && notifType != "" {
		whereParts = append(whereParts, "type = ?")
		args = append(args, notifType)
	}

	if isRead, ok := filters["is_read"].(bool); ok {
		whereParts = append(whereParts, "is_read = ?")
		args = append(args, isRead)
	}

	if priority, ok := filters["priority"].(string); ok && priority != "" {
		whereParts = append(whereParts, "priority = ?")
		args = append(args, priority)
	}

	whereClause := strings.Join(whereParts, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", whereClause)
	var totalCount int
	err := ns.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get unread count
	var unreadCount int
	err = ns.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = 0", userID).Scan(&unreadCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}

	// Pagination
	perPage := 20
	if pp, ok := filters["per_page"].(int); ok && pp > 0 && pp <= 100 {
		perPage = pp
	}

	page := 1
	if p, ok := filters["page"].(int); ok && p > 0 {
		page = p
	}

	offset := (page - 1) * perPage
	totalPages := (totalCount + perPage - 1) / perPage

	// Get notifications
	query := fmt.Sprintf(`
		SELECT id, user_id, title, message, type, priority, category, reference_type, reference_id,
			   status, is_read, read_at, scheduled_for, sent_at, delivered_at, data, sound_played,
			   retry_count, created_at, updated_at
		FROM notifications 
		WHERE %s 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, perPage, offset)
	rows, err := ns.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var dataStr sql.NullString

		err := rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.Priority,
			&n.Category, &n.ReferenceType, &n.ReferenceID, &n.Status,
			&n.IsRead, &n.ReadAt, &n.ScheduledFor, &n.SentAt, &n.DeliveredAt,
			&dataStr, &n.SoundPlayed, &n.RetryCount, &n.CreatedAt, &n.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		// Parse JSON data
		if dataStr.Valid && dataStr.String != "" {
			json.Unmarshal([]byte(dataStr.String), &n.Data)
		}

		notifications = append(notifications, n)
	}

	return &models.NotificationListResponse{
		Notifications: notifications,
		Pagination: models.PaginationInfo{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalCount:  totalCount,
			PerPage:     perPage,
		},
		UnreadCount: unreadCount,
	}, nil
}

// GetNotificationByID gets a notification by ID
func (ns *NotificationService) GetNotificationByID(id int) (*models.Notification, error) {
	query := `
		SELECT id, user_id, title, message, type, priority, category, reference_type, reference_id,
			   status, is_read, read_at, scheduled_for, sent_at, delivered_at, data, sound_played,
			   retry_count, created_at, updated_at
		FROM notifications 
		WHERE id = ?
	`

	var n models.Notification
	var dataStr sql.NullString

	err := ns.db.QueryRow(query, id).Scan(
		&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.Priority,
		&n.Category, &n.ReferenceType, &n.ReferenceID, &n.Status,
		&n.IsRead, &n.ReadAt, &n.ScheduledFor, &n.SentAt, &n.DeliveredAt,
		&dataStr, &n.SoundPlayed, &n.RetryCount, &n.CreatedAt, &n.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Parse JSON data
	if dataStr.Valid && dataStr.String != "" {
		json.Unmarshal([]byte(dataStr.String), &n.Data)
	}

	return &n, nil
}

// Helper methods

// validateNotificationRequest validates the notification request
func (ns *NotificationService) validateNotificationRequest(req models.CreateNotificationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}

	validTypes := []string{"chama", "transaction", "reminder", "system", "marketing", "alert"}
	validType := false
	for _, vt := range validTypes {
		if req.Type == vt {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("invalid notification type: %s", req.Type)
	}

	return nil
}

// shouldSendNotification checks if user wants this type of notification
func (ns *NotificationService) shouldSendNotification(notifType string, preferences *models.UserNotificationPreferences) bool {
	switch notifType {
	case "chama":
		return preferences.ChamaNotifications
	case "transaction":
		return preferences.TransactionNotifications
	case "reminder":
		return preferences.ReminderNotifications
	case "system":
		return preferences.SystemNotifications
	case "marketing":
		return preferences.MarketingNotifications
	default:
		return true // Always send alerts and unknown types
	}
}

// isQuietHours checks if current time is within quiet hours
func (ns *NotificationService) isQuietHours(preferences *models.UserNotificationPreferences) bool {
	if !preferences.QuietHoursEnabled {
		return false
	}

	// Parse time strings
	startTime, err := time.Parse("15:04:05", preferences.QuietHoursStart)
	if err != nil {
		return false
	}

	endTime, err := time.Parse("15:04:05", preferences.QuietHoursEnd)
	if err != nil {
		return false
	}

	// Get current time in user's timezone
	loc, err := time.LoadLocation(preferences.Timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	// Handle overnight quiet hours (e.g., 22:00 to 07:00)
	if startTime.After(endTime) {
		return currentTime.After(startTime) || currentTime.Before(endTime)
	}

	return currentTime.After(startTime) && currentTime.Before(endTime)
}

// scheduleForLater schedules notification for after quiet hours
func (ns *NotificationService) scheduleForLater(notification *models.Notification, preferences *models.UserNotificationPreferences) error {
	// Parse end time
	endTime, err := time.Parse("15:04:05", preferences.QuietHoursEnd)
	if err != nil {
		return err
	}

	// Get user's timezone
	loc, err := time.LoadLocation(preferences.Timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	scheduleTime := time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), 0, loc)

	// If end time is earlier than current time, schedule for tomorrow
	if scheduleTime.Before(now) {
		scheduleTime = scheduleTime.AddDate(0, 0, 1)
	}

	// Update notification
	_, err = ns.db.Exec(`
		UPDATE notifications
		SET scheduled_for = ?, status = 'pending', updated_at = ?
		WHERE id = ?
	`, scheduleTime, time.Now(), notification.ID)

	return err
}

// deliverInApp delivers in-app notification
func (ns *NotificationService) deliverInApp(notification *models.Notification) error {
	// In-app notifications are stored in database and delivered via API
	ns.logDelivery(notification.ID, notification.UserID, "in_app", "delivered", "")
	return nil
}

// deliverPush delivers push notification with sound
func (ns *NotificationService) deliverPush(notification *models.Notification, preferences *models.UserNotificationPreferences) error {
	// Get sound file path
	soundPath := "/notification_sound/ring.mp3" // Default
	if preferences.NotificationSoundID != nil {
		sound, err := ns.getSoundByID(*preferences.NotificationSoundID)
		if err == nil && sound.FilePath != "" {
			soundPath = sound.FilePath
		}
	}

	// Prepare push notification payload
	payload := map[string]interface{}{
		"title":    notification.Title,
		"body":     notification.Message,
		"sound":    soundPath,
		"vibrate":  preferences.VibrationEnabled,
		"priority": ns.mapPriorityToAndroid(notification.Priority),
		"data": map[string]interface{}{
			"notification_id": notification.ID,
			"type":            notification.Type,
			"category":        notification.Category,
			"reference_type":  notification.ReferenceType,
			"reference_id":    notification.ReferenceID,
			"custom_data":     notification.Data,
		},
	}

	// Here you would integrate with your push notification service
	// (Firebase, OneSignal, etc.)
	log.Printf("Push notification sent: user_id=%d, payload=%+v", notification.UserID, payload)

	ns.logDelivery(notification.ID, notification.UserID, "push", "sent", "")
	return nil
}

// logDelivery logs notification delivery attempt
func (ns *NotificationService) logDelivery(notificationID int, userID string, method, status, errorMsg string) {
	now := time.Now()
	var deliveredAt *time.Time
	if status == "delivered" {
		deliveredAt = &now
	}

	_, err := ns.db.Exec(`
		INSERT INTO notification_delivery_log
		(notification_id, user_id, delivery_method, status, attempted_at, delivered_at, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, notificationID, userID, method, status, now, deliveredAt, errorMsg, now)

	if err != nil {
		log.Printf("Failed to log delivery: %v", err)
	}
}

// replaceTemplateVariables replaces variables in template strings
func (ns *NotificationService) replaceTemplateVariables(template string, variables map[string]interface{}) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// mapPriorityToAndroid maps notification priority to Android priority
func (ns *NotificationService) mapPriorityToAndroid(priority string) string {
	switch priority {
	case "urgent":
		return "max"
	case "high":
		return "high"
	case "normal":
		return "default"
	case "low":
		return "low"
	default:
		return "default"
	}
}

// createDefaultPreferences creates default notification preferences for a user
func (ns *NotificationService) createDefaultPreferences(userID string) (*models.UserNotificationPreferences, error) {
	// Get default sound ID
	defaultSoundID, err := ns.getDefaultSoundID()
	if err != nil {
		log.Printf("Warning: Could not get default sound ID: %v", err)
	}

	now := time.Now()
	query := `
		INSERT INTO user_notification_preferences
		(user_id, notification_sound_id, sound_enabled, vibration_enabled, volume_level,
		 chama_notifications, transaction_notifications, reminder_notifications,
		 system_notifications, marketing_notifications, quiet_hours_enabled,
		 quiet_hours_start, quiet_hours_end, timezone, notification_frequency,
		 priority_only_during_quiet, created_at, updated_at)
		VALUES (?, ?, 1, 1, 80, 1, 1, 1, 1, 0, 0, '22:00:00', '07:00:00',
		        'Africa/Nairobi', 'immediate', 1, ?, ?)
	`

	_, err = ns.db.Exec(query, userID, defaultSoundID, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create default preferences: %w", err)
	}

	return ns.GetUserPreferences(userID)
}

// getDefaultSoundID gets the default notification sound ID
func (ns *NotificationService) getDefaultSoundID() (*int, error) {
	var id int
	err := ns.db.QueryRow("SELECT id FROM notification_sounds WHERE is_default = 1 AND is_active = 1 LIMIT 1").Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &id, nil
}

// getSoundByID gets a notification sound by ID
func (ns *NotificationService) getSoundByID(id int) (*models.NotificationSound, error) {
	sound := &models.NotificationSound{}
	query := `
		SELECT id, name, file_path, file_size, duration_seconds, is_default, is_active, created_at, updated_at
		FROM notification_sounds
		WHERE id = ? AND is_active = 1
	`

	err := ns.db.QueryRow(query, id).Scan(
		&sound.ID, &sound.Name, &sound.FilePath, &sound.FileSize,
		&sound.DurationSeconds, &sound.IsDefault, &sound.IsActive,
		&sound.CreatedAt, &sound.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sound not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get sound: %w", err)
	}

	return sound, nil
}

// getTemplate gets a notification template by name
func (ns *NotificationService) getTemplate(name string) (*models.NotificationTemplate, error) {
	template := &models.NotificationTemplate{}
	var variablesStr sql.NullString

	query := `
		SELECT id, name, type, category, title_template, message_template,
			   default_priority, requires_sound, requires_vibration, variables,
			   is_active, created_at, updated_at
		FROM notification_templates
		WHERE name = ? AND is_active = 1
	`

	err := ns.db.QueryRow(query, name).Scan(
		&template.ID, &template.Name, &template.Type, &template.Category,
		&template.TitleTemplate, &template.MessageTemplate, &template.DefaultPriority,
		&template.RequiresSound, &template.RequiresVibration, &variablesStr,
		&template.IsActive, &template.CreatedAt, &template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Parse variables JSON
	if variablesStr.Valid && variablesStr.String != "" {
		json.Unmarshal([]byte(variablesStr.String), &template.Variables)
	}

	return template, nil
}
