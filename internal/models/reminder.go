package models

import (
	"time"
)

// ReminderType represents the type of reminder
type ReminderType string

const (
	ReminderTypeOnce    ReminderType = "once"
	ReminderTypeDaily   ReminderType = "daily"
	ReminderTypeWeekly  ReminderType = "weekly"
	ReminderTypeMonthly ReminderType = "monthly"
)

// Reminder represents a user reminder in the system
type Reminder struct {
	ID               string       `json:"id" db:"id"`
	UserID           string       `json:"userId" db:"user_id"`
	Title            string       `json:"title" db:"title"`
	Description      *string      `json:"description,omitempty" db:"description"`
	ReminderType     ReminderType `json:"reminderType" db:"reminder_type"`
	ScheduledAt      time.Time    `json:"scheduledAt" db:"scheduled_at"`
	IsEnabled        bool         `json:"isEnabled" db:"is_enabled"`
	IsCompleted      bool         `json:"isCompleted" db:"is_completed"`
	NotificationSent bool         `json:"notificationSent" db:"notification_sent"`
	CreatedAt        time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time    `json:"updatedAt" db:"updated_at"`
}

// CreateReminderRequest represents the request to create a new reminder
type CreateReminderRequest struct {
	Title        string       `json:"title" binding:"required,min=1,max=100"`
	Description  *string      `json:"description,omitempty" binding:"omitempty,max=500"`
	ReminderType ReminderType `json:"reminderType" binding:"required,oneof=once daily weekly monthly"`
	ScheduledAt  time.Time    `json:"scheduledAt" binding:"required"`
	IsEnabled    *bool        `json:"isEnabled,omitempty"`
}

// UpdateReminderRequest represents the request to update a reminder
type UpdateReminderRequest struct {
	Title        *string      `json:"title,omitempty" binding:"omitempty,min=1,max=100"`
	Description  *string      `json:"description,omitempty" binding:"omitempty,max=500"`
	ReminderType *ReminderType `json:"reminderType,omitempty" binding:"omitempty,oneof=once daily weekly monthly"`
	ScheduledAt  *time.Time   `json:"scheduledAt,omitempty"`
	IsEnabled    *bool        `json:"isEnabled,omitempty"`
	IsCompleted  *bool        `json:"isCompleted,omitempty"`
}

// ReminderResponse represents the response structure for reminder operations
type ReminderResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RemindersListResponse represents the response for listing reminders
type RemindersListResponse struct {
	Success bool       `json:"success"`
	Data    []Reminder `json:"data"`
	Count   int        `json:"count"`
	Error   string     `json:"error,omitempty"`
}

// ReminderNotification represents a notification to be sent for a reminder
type ReminderNotification struct {
	ReminderID  string    `json:"reminderId"`
	UserID      string    `json:"userId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Type        string    `json:"type"`
}

// IsValidReminderType checks if the reminder type is valid
func IsValidReminderType(reminderType string) bool {
	switch ReminderType(reminderType) {
	case ReminderTypeOnce, ReminderTypeDaily, ReminderTypeWeekly, ReminderTypeMonthly:
		return true
	default:
		return false
	}
}

// GetNextScheduledTime calculates the next scheduled time for recurring reminders
func (r *Reminder) GetNextScheduledTime() *time.Time {
	if r.ReminderType == ReminderTypeOnce {
		return nil // One-time reminders don't have a next time
	}

	now := time.Now()
	if r.ScheduledAt.After(now) {
		return &r.ScheduledAt // If the original time hasn't passed yet
	}

	var nextTime time.Time
	switch r.ReminderType {
	case ReminderTypeDaily:
		nextTime = r.ScheduledAt.AddDate(0, 0, 1)
		for nextTime.Before(now) {
			nextTime = nextTime.AddDate(0, 0, 1)
		}
	case ReminderTypeWeekly:
		nextTime = r.ScheduledAt.AddDate(0, 0, 7)
		for nextTime.Before(now) {
			nextTime = nextTime.AddDate(0, 0, 7)
		}
	case ReminderTypeMonthly:
		nextTime = r.ScheduledAt.AddDate(0, 1, 0)
		for nextTime.Before(now) {
			nextTime = nextTime.AddDate(0, 1, 0)
		}
	default:
		return nil
	}

	return &nextTime
}

// ShouldSendNotification checks if a notification should be sent for this reminder
func (r *Reminder) ShouldSendNotification() bool {
	if !r.IsEnabled || r.IsCompleted {
		return false
	}

	now := time.Now()
	
	// For one-time reminders, check if the scheduled time has passed and notification hasn't been sent
	if r.ReminderType == ReminderTypeOnce {
		return now.After(r.ScheduledAt) && !r.NotificationSent
	}

	// For recurring reminders, check if it's time for the next occurrence
	nextTime := r.GetNextScheduledTime()
	if nextTime == nil {
		return false
	}

	// Send notification if we're within 1 minute of the scheduled time
	timeDiff := nextTime.Sub(now)
	return timeDiff <= time.Minute && timeDiff >= -time.Minute
}
