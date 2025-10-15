package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"vaultke-backend/internal/models"

	"github.com/google/uuid"
)

// ReminderService handles reminder-related business logic
type ReminderService struct {
	db *sql.DB
}

// NewReminderService creates a new reminder service
func NewReminderService(db *sql.DB) *ReminderService {
	return &ReminderService{db: db}
}

// CreateReminder creates a new reminder for a user
func (s *ReminderService) CreateReminder(userID string, req *models.CreateReminderRequest) (*models.Reminder, error) {
	// Validate that the scheduled time is in the future
	if req.ScheduledAt.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled time must be in the future")
	}

	// Generate unique ID
	reminderID := uuid.New().String()

	// Set default values
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	now := time.Now()
	reminder := &models.Reminder{
		ID:           reminderID,
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		ReminderType: req.ReminderType,
		ScheduledAt:  req.ScheduledAt,
		IsEnabled:    isEnabled,
		IsCompleted:  false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Insert into database
	query := `
		INSERT INTO reminders (
			id, user_id, title, description, reminder_type, 
			scheduled_at, is_enabled, is_completed, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		reminder.ID,
		reminder.UserID,
		reminder.Title,
		reminder.Description,
		reminder.ReminderType,
		reminder.ScheduledAt,
		reminder.IsEnabled,
		reminder.IsCompleted,
		reminder.CreatedAt,
		reminder.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	log.Printf("Created reminder %s for user %s", reminderID, userID)
	return reminder, nil
}

// GetUserReminders retrieves all reminders for a user
func (s *ReminderService) GetUserReminders(userID string, limit, offset int) ([]models.Reminder, error) {
	query := `
		SELECT id, user_id, title, description, reminder_type, scheduled_at, 
			   is_enabled, is_completed, notification_sent, created_at, updated_at
		FROM reminders 
		WHERE user_id = ? 
		ORDER BY scheduled_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reminders: %w", err)
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var reminder models.Reminder
		err := rows.Scan(
			&reminder.ID,
			&reminder.UserID,
			&reminder.Title,
			&reminder.Description,
			&reminder.ReminderType,
			&reminder.ScheduledAt,
			&reminder.IsEnabled,
			&reminder.IsCompleted,
			&reminder.NotificationSent,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

// GetReminderByID retrieves a specific reminder by ID
func (s *ReminderService) GetReminderByID(reminderID, userID string) (*models.Reminder, error) {
	query := `
		SELECT id, user_id, title, description, reminder_type, scheduled_at, 
			   is_enabled, is_completed, notification_sent, created_at, updated_at
		FROM reminders 
		WHERE id = ? AND user_id = ?
	`

	var reminder models.Reminder
	err := s.db.QueryRow(query, reminderID, userID).Scan(
		&reminder.ID,
		&reminder.UserID,
		&reminder.Title,
		&reminder.Description,
		&reminder.ReminderType,
		&reminder.ScheduledAt,
		&reminder.IsEnabled,
		&reminder.IsCompleted,
		&reminder.NotificationSent,
		&reminder.CreatedAt,
		&reminder.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reminder not found")
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return &reminder, nil
}

// UpdateReminder updates an existing reminder
func (s *ReminderService) UpdateReminder(reminderID, userID string, req *models.UpdateReminderRequest) (*models.Reminder, error) {
	// First, get the existing reminder
	existing, err := s.GetReminderByID(reminderID, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.ReminderType != nil {
		existing.ReminderType = *req.ReminderType
	}
	if req.ScheduledAt != nil {
		// Validate that the new scheduled time is in the future
		if req.ScheduledAt.Before(time.Now()) {
			return nil, fmt.Errorf("scheduled time must be in the future")
		}
		existing.ScheduledAt = *req.ScheduledAt
	}
	if req.IsEnabled != nil {
		existing.IsEnabled = *req.IsEnabled
	}
	if req.IsCompleted != nil {
		existing.IsCompleted = *req.IsCompleted
	}

	existing.UpdatedAt = time.Now()

	// Update in database
	query := `
		UPDATE reminders 
		SET title = ?, description = ?, reminder_type = ?, scheduled_at = ?, 
			is_enabled = ?, is_completed = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`

	_, err = s.db.Exec(
		query,
		existing.Title,
		existing.Description,
		existing.ReminderType,
		existing.ScheduledAt,
		existing.IsEnabled,
		existing.IsCompleted,
		existing.UpdatedAt,
		reminderID,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update reminder: %w", err)
	}

	log.Printf("Updated reminder %s for user %s", reminderID, userID)
	return existing, nil
}

// DeleteReminder deletes a reminder
func (s *ReminderService) DeleteReminder(reminderID, userID string) error {
	query := `DELETE FROM reminders WHERE id = ? AND user_id = ?`

	result, err := s.db.Exec(query, reminderID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reminder not found")
	}

	log.Printf("Deleted reminder %s for user %s", reminderID, userID)
	return nil
}

// GetPendingReminders retrieves reminders that need notifications sent
func (s *ReminderService) GetPendingReminders() ([]models.Reminder, error) {
	now := time.Now()

	query := `
		SELECT id, user_id, title, description, reminder_type, scheduled_at, 
			   is_enabled, is_completed, notification_sent, created_at, updated_at
		FROM reminders 
		WHERE is_enabled = TRUE 
		  AND is_completed = FALSE 
		  AND (
		    (reminder_type = 'once' AND scheduled_at <= ? AND notification_sent = FALSE)
		    OR (reminder_type != 'once' AND scheduled_at <= ?)
		  )
		ORDER BY scheduled_at ASC
	`

	rows, err := s.db.Query(query, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending reminders: %w", err)
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var reminder models.Reminder
		err := rows.Scan(
			&reminder.ID,
			&reminder.UserID,
			&reminder.Title,
			&reminder.Description,
			&reminder.ReminderType,
			&reminder.ScheduledAt,
			&reminder.IsEnabled,
			&reminder.IsCompleted,
			&reminder.NotificationSent,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

// MarkNotificationSent marks a reminder as having its notification sent
func (s *ReminderService) MarkNotificationSent(reminderID string) error {
	query := `UPDATE reminders SET notification_sent = TRUE WHERE id = ?`

	_, err := s.db.Exec(query, reminderID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	return nil
}
