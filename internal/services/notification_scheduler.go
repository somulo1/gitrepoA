package services

import (
	"database/sql"
	"log"
	"time"

	"vaultke-backend/internal/models"
)

// NotificationScheduler handles scheduling and sending reminder notifications
type NotificationScheduler struct {
	db              *sql.DB
	reminderService *ReminderService
	ticker          *time.Ticker
	stopChan        chan bool
}

// NewNotificationScheduler creates a new notification scheduler
func NewNotificationScheduler(db *sql.DB) *NotificationScheduler {
	return &NotificationScheduler{
		db:              db,
		reminderService: NewReminderService(db),
		stopChan:        make(chan bool),
	}
}

// Start begins the notification scheduling process
func (ns *NotificationScheduler) Start() {
	log.Println("Starting notification scheduler...")

	// Check for pending notifications every minute
	ns.ticker = time.NewTicker(1 * time.Minute)
	defer ns.ticker.Stop()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Notification scheduler panic recovered: %v", r)
			}
		}()

		for {
			select {
			case <-ns.ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Notification processing panic recovered: %v", r)
						}
					}()
					ns.processPendingNotifications()
				}()
			case <-ns.stopChan:
				log.Println("Stopping notification scheduler...")
				return
			}
		}
	}()
}

// Stop stops the notification scheduler
func (ns *NotificationScheduler) Stop() {
	if ns.ticker != nil {
		ns.ticker.Stop()
	}
	ns.stopChan <- true
}

// processPendingNotifications checks for and processes pending reminder notifications
func (ns *NotificationScheduler) processPendingNotifications() {
	reminders, err := ns.reminderService.GetPendingReminders()
	if err != nil {
		log.Printf("Error getting pending reminders: %v", err)
		return
	}

	if len(reminders) == 0 {
		return
	}

	log.Printf("Processing %d pending reminder notifications", len(reminders))

	for _, reminder := range reminders {
		if ns.shouldSendNotification(&reminder) {
			ns.sendNotification(&reminder)
		}
	}
}

// shouldSendNotification determines if a notification should be sent for a reminder
func (ns *NotificationScheduler) shouldSendNotification(reminder *models.Reminder) bool {
	if !reminder.IsEnabled || reminder.IsCompleted {
		return false
	}

	now := time.Now()

	// For one-time reminders, check if the scheduled time has passed and notification hasn't been sent
	if reminder.ReminderType == models.ReminderTypeOnce {
		return now.After(reminder.ScheduledAt) && !reminder.NotificationSent
	}

	// For recurring reminders, check if it's time for the next occurrence
	nextTime := reminder.GetNextScheduledTime()
	if nextTime == nil {
		return false
	}

	// Send notification if we're within 1 minute of the scheduled time
	timeDiff := nextTime.Sub(now)
	return timeDiff <= time.Minute && timeDiff >= -time.Minute
}

// sendNotification sends a notification for a reminder
func (ns *NotificationScheduler) sendNotification(reminder *models.Reminder) {
	log.Printf("Sending notification for reminder: %s (User: %s)", reminder.Title, reminder.UserID)

	// Create notification record (this would integrate with your notification system)
	notification := &models.ReminderNotification{
		ReminderID:  reminder.ID,
		UserID:      reminder.UserID,
		Title:       reminder.Title,
		Description: getNotificationDescription(reminder),
		ScheduledAt: reminder.ScheduledAt,
		Type:        "reminder",
	}

	// Here you would integrate with your notification service
	// For example: push notifications, email, SMS, etc.
	ns.processNotification(notification)

	// Mark notification as sent for one-time reminders
	if reminder.ReminderType == models.ReminderTypeOnce {
		err := ns.reminderService.MarkNotificationSent(reminder.ID)
		if err != nil {
			log.Printf("Error marking notification as sent for reminder %s: %v", reminder.ID, err)
		}
	}
}

// processNotification processes the actual notification sending
func (ns *NotificationScheduler) processNotification(notification *models.ReminderNotification) {
	// This is where you would integrate with your notification service
	// For now, we'll just log the notification
	log.Printf("📱 NOTIFICATION: %s - %s (User: %s)",
		notification.Title,
		notification.Description,
		notification.UserID)

	// TODO: Integrate with actual notification services:
	// - Push notifications (FCM, APNs)
	// - Email notifications
	// - SMS notifications
	// - In-app notifications

	// Example integration points:
	// - Send push notification via FCM/APNs
	// - Send email via SMTP or email service
	// - Send SMS via Twilio or similar service
	// - Store in-app notification in database
}

// getNotificationDescription creates a description for the notification
func getNotificationDescription(reminder *models.Reminder) string {
	if reminder.Description != nil && *reminder.Description != "" {
		return *reminder.Description
	}

	switch reminder.ReminderType {
	case models.ReminderTypeDaily:
		return "Daily reminder"
	case models.ReminderTypeWeekly:
		return "Weekly reminder"
	case models.ReminderTypeMonthly:
		return "Monthly reminder"
	default:
		return "Reminder notification"
	}
}

// GetNotificationStats returns statistics about notification processing
func (ns *NotificationScheduler) GetNotificationStats() map[string]interface{} {
	// Get pending reminders count
	pendingReminders, err := ns.reminderService.GetPendingReminders()
	pendingCount := 0
	if err == nil {
		pendingCount = len(pendingReminders)
	}

	return map[string]interface{}{
		"pending_notifications": pendingCount,
		"scheduler_running":     ns.ticker != nil,
		"last_check":            time.Now().Format(time.RFC3339),
	}
}

// ProcessImmediateNotification processes a notification immediately (for testing)
func (ns *NotificationScheduler) ProcessImmediateNotification(reminderID string) error {
	// This method can be used for testing or manual notification triggers
	reminder, err := ns.reminderService.GetReminderByID(reminderID, "")
	if err != nil {
		return err
	}

	ns.sendNotification(reminder)
	return nil
}
