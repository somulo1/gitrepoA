package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// NotificationSound represents available notification sounds
type NotificationSound struct {
	ID              int       `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	FilePath        string    `json:"file_path" db:"file_path"`
	FileSize        int       `json:"file_size" db:"file_size"`
	DurationSeconds float64   `json:"duration_seconds" db:"duration_seconds"`
	IsDefault       bool      `json:"is_default" db:"is_default"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// UserNotificationPreferences represents user notification settings
type UserNotificationPreferences struct {
	ID                       int       `json:"id" db:"id"`
	UserID                   string    `json:"user_id" db:"user_id"`
	NotificationSoundID      *int      `json:"notification_sound_id" db:"notification_sound_id"`
	SoundEnabled             bool      `json:"sound_enabled" db:"sound_enabled"`
	VibrationEnabled         bool      `json:"vibration_enabled" db:"vibration_enabled"`
	VolumeLevel              int       `json:"volume_level" db:"volume_level"`
	ChamaNotifications       bool      `json:"chama_notifications" db:"chama_notifications"`
	TransactionNotifications bool      `json:"transaction_notifications" db:"transaction_notifications"`
	ReminderNotifications    bool      `json:"reminder_notifications" db:"reminder_notifications"`
	SystemNotifications      bool      `json:"system_notifications" db:"system_notifications"`
	MarketingNotifications   bool      `json:"marketing_notifications" db:"marketing_notifications"`
	QuietHoursEnabled        bool      `json:"quiet_hours_enabled" db:"quiet_hours_enabled"`
	QuietHoursStart          string    `json:"quiet_hours_start" db:"quiet_hours_start"`
	QuietHoursEnd            string    `json:"quiet_hours_end" db:"quiet_hours_end"`
	Timezone                 string    `json:"timezone" db:"timezone"`
	NotificationFrequency    string    `json:"notification_frequency" db:"notification_frequency"`
	PriorityOnlyDuringQuiet  bool      `json:"priority_only_during_quiet" db:"priority_only_during_quiet"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// Notification represents a notification
type Notification struct {
	ID            int        `json:"id" db:"id"`
	UserID        string     `json:"user_id" db:"user_id"`
	Title         string     `json:"title" db:"title"`
	Message       string     `json:"message" db:"message"`
	Type          string     `json:"type" db:"type"`
	Priority      string     `json:"priority" db:"priority"`
	Category      *string    `json:"category" db:"category"`
	ReferenceType *string    `json:"reference_type" db:"reference_type"`
	ReferenceID   *int       `json:"reference_id" db:"reference_id"`
	Status        string     `json:"status" db:"status"`
	IsRead        bool       `json:"is_read" db:"is_read"`
	ReadAt        *time.Time `json:"read_at" db:"read_at"`
	ScheduledFor  time.Time  `json:"scheduled_for" db:"scheduled_for"`
	SentAt        *time.Time `json:"sent_at" db:"sent_at"`
	DeliveredAt   *time.Time `json:"delivered_at" db:"delivered_at"`
	Data          JSONMap    `json:"data" db:"data"`
	SoundPlayed   bool       `json:"sound_played" db:"sound_played"`
	RetryCount    int        `json:"retry_count" db:"retry_count"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID                int       `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Type              string    `json:"type" db:"type"`
	Category          string    `json:"category" db:"category"`
	TitleTemplate     string    `json:"title_template" db:"title_template"`
	MessageTemplate   string    `json:"message_template" db:"message_template"`
	DefaultPriority   string    `json:"default_priority" db:"default_priority"`
	RequiresSound     bool      `json:"requires_sound" db:"requires_sound"`
	RequiresVibration bool      `json:"requires_vibration" db:"requires_vibration"`
	Variables         JSONArray `json:"variables" db:"variables"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// UserReminder represents a user reminder
type UserReminder struct {
	ID                 int        `json:"id" db:"id"`
	UserID             string     `json:"user_id" db:"user_id"`
	Title              string     `json:"title" db:"title"`
	Description        *string    `json:"description" db:"description"`
	ReminderDatetime   time.Time  `json:"reminder_datetime" db:"reminder_datetime"`
	Timezone           string     `json:"timezone" db:"timezone"`
	IsRecurring        bool       `json:"is_recurring" db:"is_recurring"`
	RecurrencePattern  *string    `json:"recurrence_pattern" db:"recurrence_pattern"`
	RecurrenceInterval int        `json:"recurrence_interval" db:"recurrence_interval"`
	RecurrenceEndDate  *time.Time `json:"recurrence_end_date" db:"recurrence_end_date"`
	SoundEnabled       bool       `json:"sound_enabled" db:"sound_enabled"`
	VibrationEnabled   bool       `json:"vibration_enabled" db:"vibration_enabled"`
	CustomSoundID      *int       `json:"custom_sound_id" db:"custom_sound_id"`
	Status             string     `json:"status" db:"status"`
	SnoozeUntil        *time.Time `json:"snooze_until" db:"snooze_until"`
	CompletedAt        *time.Time `json:"completed_at" db:"completed_at"`
	Category           string     `json:"category" db:"category"`
	Priority           string     `json:"priority" db:"priority"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// NotificationDeliveryLog represents delivery tracking
type NotificationDeliveryLog struct {
	ID             int        `json:"id" db:"id"`
	NotificationID int        `json:"notification_id" db:"notification_id"`
	UserID         string     `json:"user_id" db:"user_id"`
	DeliveryMethod string     `json:"delivery_method" db:"delivery_method"`
	Status         string     `json:"status" db:"status"`
	AttemptedAt    time.Time  `json:"attempted_at" db:"attempted_at"`
	DeliveredAt    *time.Time `json:"delivered_at" db:"delivered_at"`
	ErrorMessage   *string    `json:"error_message" db:"error_message"`
	RetryCount     int        `json:"retry_count" db:"retry_count"`
	DeviceInfo     JSONMap    `json:"device_info" db:"device_info"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// JSONMap is a custom type for JSON data
type JSONMap map[string]interface{}

// JSONArray is a custom type for JSON arrays
type JSONArray []string

// Value implements the driver.Valuer interface for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface for JSONArray
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONArray
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	TotalNotifications  int            `json:"total_notifications"`
	UnreadNotifications int            `json:"unread_notifications"`
	NotificationsByType map[string]int `json:"notifications_by_type"`
	RecentActivity      int            `json:"recent_activity"`
}

// CreateNotificationRequest represents a request to create a notification
type CreateNotificationRequest struct {
	UserID        string                 `json:"user_id" validate:"required"`
	Title         string                 `json:"title" validate:"required"`
	Message       string                 `json:"message" validate:"required"`
	Type          string                 `json:"type" validate:"required,oneof=chama transaction reminder system marketing alert"`
	Priority      string                 `json:"priority" validate:"oneof=low normal high urgent"`
	Category      string                 `json:"category"`
	ReferenceType string                 `json:"reference_type"`
	ReferenceID   int                    `json:"reference_id"`
	ScheduledFor  *time.Time             `json:"scheduled_for"`
	Data          map[string]interface{} `json:"data"`
}

// UpdatePreferencesRequest represents a request to update notification preferences
type UpdatePreferencesRequest struct {
	NotificationSoundID      *int   `json:"notification_sound_id"`
	SoundEnabled             *bool  `json:"sound_enabled"`
	VibrationEnabled         *bool  `json:"vibration_enabled"`
	VolumeLevel              *int   `json:"volume_level" validate:"omitempty,min=0,max=100"`
	ChamaNotifications       *bool  `json:"chama_notifications"`
	TransactionNotifications *bool  `json:"transaction_notifications"`
	ReminderNotifications    *bool  `json:"reminder_notifications"`
	SystemNotifications      *bool  `json:"system_notifications"`
	MarketingNotifications   *bool  `json:"marketing_notifications"`
	QuietHoursEnabled        *bool  `json:"quiet_hours_enabled"`
	QuietHoursStart          string `json:"quiet_hours_start" validate:"omitempty"`
	QuietHoursEnd            string `json:"quiet_hours_end" validate:"omitempty"`
	Timezone                 string `json:"timezone"`
	NotificationFrequency    string `json:"notification_frequency" validate:"omitempty,oneof=immediate batched_15min batched_1hour daily_digest"`
	PriorityOnlyDuringQuiet  *bool  `json:"priority_only_during_quiet"`
}

// CreateReminderRequest represents a request to create a reminder
type CreateReminderRequest struct {
	Title              string     `json:"title" validate:"required"`
	Description        string     `json:"description"`
	ReminderDatetime   time.Time  `json:"reminder_datetime" validate:"required"`
	Timezone           string     `json:"timezone"`
	IsRecurring        bool       `json:"is_recurring"`
	RecurrencePattern  string     `json:"recurrence_pattern" validate:"omitempty,oneof=daily weekly monthly yearly"`
	RecurrenceInterval int        `json:"recurrence_interval" validate:"omitempty,min=1"`
	RecurrenceEndDate  *time.Time `json:"recurrence_end_date"`
	SoundEnabled       bool       `json:"sound_enabled"`
	VibrationEnabled   bool       `json:"vibration_enabled"`
	CustomSoundID      *int       `json:"custom_sound_id"`
	Category           string     `json:"category"`
	Priority           string     `json:"priority" validate:"omitempty,oneof=low normal high"`
}

// NotificationResponse represents the API response structure
type NotificationResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalCount  int `json:"total_count"`
	PerPage     int `json:"per_page"`
}

// NotificationListResponse represents paginated notification response
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Pagination    PaginationInfo `json:"pagination"`
	UnreadCount   int            `json:"unread_count"`
}
