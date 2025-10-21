package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"vaultke-backend/config"
	// "vaultke-backend/internal/models"
)

// NotificationService handles notifications
type NotificationService struct {
	db     *sql.DB
	config *config.Config
	client *http.Client
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *sql.DB, cfg *config.Config) *NotificationService {
	return &NotificationService{
		db:     db,
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// NotificationType represents notification types
type NotificationType string

const (
	NotificationTypeInfo        NotificationType = "info"
	NotificationTypeSuccess     NotificationType = "success"
	NotificationTypeWarning     NotificationType = "warning"
	NotificationTypeError       NotificationType = "error"
	NotificationTypeTransaction NotificationType = "transaction"
	NotificationTypeChama       NotificationType = "chama"
	NotificationTypeMarketplace NotificationType = "marketplace"
	NotificationTypeMeeting     NotificationType = "meeting"
	NotificationTypeVote        NotificationType = "vote"
	NotificationTypeLoan        NotificationType = "loan"
)

// Notification represents a notification
type Notification struct {
	ID        string           `json:"id" db:"id"`
	UserID    string           `json:"userId" db:"user_id"`
	Type      NotificationType `json:"type" db:"type"`
	Title     string           `json:"title" db:"title"`
	Message   string           `json:"message" db:"message"`
	Data      string           `json:"data" db:"data"`
	IsRead    bool             `json:"isRead" db:"is_read"`
	IsPush    bool             `json:"isPush" db:"is_push"`
	IsEmail   bool             `json:"isEmail" db:"is_email"`
	IsSMS     bool             `json:"isSMS" db:"is_sms"`
	CreatedAt time.Time        `json:"createdAt" db:"created_at"`
	ReadAt    *time.Time       `json:"readAt,omitempty" db:"read_at"`
}

// FCMMessage represents Firebase Cloud Messaging message
type FCMMessage struct {
	To           string                 `json:"to,omitempty"`
	Registration []string               `json:"registration_ids,omitempty"`
	Notification FCMNotification        `json:"notification"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Priority     string                 `json:"priority"`
}

// FCMNotification represents FCM notification payload
type FCMNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	Sound string `json:"sound,omitempty"`
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(userID string, notifType NotificationType, title, message string, data map[string]interface{}, sendPush, sendEmail, sendSMS bool) (*Notification, error) {
	// Serialize data
	dataJSON := "{}"
	if data != nil {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize notification data: %w", err)
		}
		dataJSON = string(dataBytes)
	}

	notification := &Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		Data:      dataJSON,
		IsRead:    false,
		IsPush:    sendPush,
		IsEmail:   sendEmail,
		IsSMS:     sendSMS,
		CreatedAt: time.Now(),
	}

	// Insert notification
	query := `
		INSERT INTO notifications (
			id, user_id, type, title, message, data, is_read, is_push, is_email, is_sms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		notification.ID, notification.UserID, notification.Type, notification.Title,
		notification.Message, notification.Data, notification.IsRead, notification.IsPush,
		notification.IsEmail, notification.IsSMS, notification.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Send push notification if requested
	if sendPush {
		go s.sendPushNotification(userID, title, message, data)
	}

	// Send email if requested
	if sendEmail {
		go s.sendEmailNotification(userID, title, message)
	}

	// Send SMS if requested
	if sendSMS {
		go s.sendSMSNotification(userID, message)
	}

	return notification, nil
}

// GetUserNotifications retrieves notifications for a user
func (s *NotificationService) GetUserNotifications(userID string, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, is_read, is_push, is_email, is_sms, created_at, read_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Type,
			&notification.Title, &notification.Message, &notification.Data,
			&notification.IsRead, &notification.IsPush, &notification.IsEmail,
			&notification.IsSMS, &notification.CreatedAt, &notification.ReadAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(userID, notificationID string) error {
	now := time.Now()
	query := "UPDATE notifications SET is_read = true, read_at = ? WHERE id = ? AND user_id = ?"

	result, err := s.db.Exec(query, now, notificationID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(userID string) error {
	now := time.Now()
	query := "UPDATE notifications SET is_read = true, read_at = ? WHERE user_id = ? AND is_read = false"

	_, err := s.db.Exec(query, now, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}

// GetUnreadCount gets the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(userID string) (int, error) {
	query := "SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = false"

	var count int
	err := s.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(userID, notificationID string) error {
	query := "DELETE FROM notifications WHERE id = ? AND user_id = ?"

	result, err := s.db.Exec(query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// sendPushNotification sends a push notification via FCM
func (s *NotificationService) sendPushNotification(userID, title, message string, data map[string]interface{}) {
	// Get user's FCM token
	var fcmToken string
	query := "SELECT fcm_token FROM users WHERE id = ? AND fcm_token IS NOT NULL"
	err := s.db.QueryRow(query, userID).Scan(&fcmToken)
	if err != nil {
		// User doesn't have FCM token, skip push notification
		return
	}

	// Create FCM message
	fcmMessage := FCMMessage{
		To: fcmToken,
		Notification: FCMNotification{
			Title: title,
			Body:  message,
			Icon:  "ic_notification",
			Sound: "default",
		},
		Data:     data,
		Priority: "high",
	}

	// Convert to JSON
	jsonData, err := json.Marshal(fcmMessage)
	if err != nil {
		fmt.Printf("Failed to marshal FCM message: %v\n", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to create FCM request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "key="+s.config.FirebaseServerKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send FCM notification: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FCM request failed with status: %d\n", resp.StatusCode)
	}
}

// sendEmailNotification sends an email notification
func (s *NotificationService) sendEmailNotification(userID, title, message string) {
	// Get user's email
	var email string
	query := "SELECT email FROM users WHERE id = ?"
	err := s.db.QueryRow(query, userID).Scan(&email)
	if err != nil {
		fmt.Printf("Failed to get user email: %v\n", err)
		return
	}

	// TODO: Implement email sending logic
	// This could use services like SendGrid, AWS SES, etc.
	fmt.Printf("Email notification to %s: %s - %s\n", email, title, message)
}

// sendSMSNotification sends an SMS notification
func (s *NotificationService) sendSMSNotification(userID, message string) {
	// Get user's phone
	var phone string
	query := "SELECT phone FROM users WHERE id = ?"
	err := s.db.QueryRow(query, userID).Scan(&phone)
	if err != nil {
		fmt.Printf("Failed to get user phone: %v\n", err)
		return
	}

	// TODO: Implement SMS sending logic using Africa's Talking
	fmt.Printf("SMS notification to %s: %s\n", phone, message)
}

// NotifyTransactionComplete sends transaction completion notification
func (s *NotificationService) NotifyTransactionComplete(userID string, amount float64, transactionType string) error {
	title := "Transaction Complete"
	message := fmt.Sprintf("Your %s of KSh %.2f has been completed successfully", transactionType, amount)

	data := map[string]interface{}{
		"type":            "transaction",
		"amount":          amount,
		"transactionType": transactionType,
	}

	_, err := s.CreateNotification(userID, NotificationTypeTransaction, title, message, data, true, false, false)
	return err
}

// NotifyChamaInvitation sends chama invitation notification
func (s *NotificationService) NotifyChamaInvitation(userID, chamaName, inviterName string) error {
	title := "Chama Invitation"
	message := fmt.Sprintf("%s has invited you to join %s chama", inviterName, chamaName)

	data := map[string]interface{}{
		"type":        "chama_invitation",
		"chamaName":   chamaName,
		"inviterName": inviterName,
	}

	_, err := s.CreateNotification(userID, NotificationTypeChama, title, message, data, true, true, false)
	return err
}

// NotifyMeetingReminder sends meeting reminder notification
func (s *NotificationService) NotifyMeetingReminder(userID, chamaName string, meetingTime time.Time) error {
	title := "Meeting Reminder"
	message := fmt.Sprintf("You have a %s chama meeting in 1 hour", chamaName)

	data := map[string]interface{}{
		"type":        "meeting_reminder",
		"chamaName":   chamaName,
		"meetingTime": meetingTime.Format(time.RFC3339),
	}

	_, err := s.CreateNotification(userID, NotificationTypeMeeting, title, message, data, true, false, true)
	return err
}

// NotifyOrderStatusUpdate sends order status update notification
// func (s *NotificationService) NotifyOrderStatusUpdate(userID, orderID string, status models.OrderStatus) error {
// 	title := "Order Update"
// 	message := fmt.Sprintf("Your order #%s is now %s", orderID[:8], status)

// 	data := map[string]interface{}{
// 		"type": "order_update",
// 		"orderId": orderID,
// 		"status": status,
// 	}

// 	_, err := s.CreateNotification(userID, NotificationTypeMarketplace, title, message, data, true, false, false)
// 	return err
// }

// NotifyLoanApproval sends loan approval notification
func (s *NotificationService) NotifyLoanApproval(userID string, amount float64, approved bool) error {
	var title, message string

	if approved {
		title = "Loan Approved"
		message = fmt.Sprintf("Your loan application for KSh %.2f has been approved", amount)
	} else {
		title = "Loan Declined"
		message = fmt.Sprintf("Your loan application for KSh %.2f has been declined", amount)
	}

	data := map[string]interface{}{
		"type":     "loan_decision",
		"amount":   amount,
		"approved": approved,
	}

	_, err := s.CreateNotification(userID, NotificationTypeLoan, title, message, data, true, true, false)
	return err
}
