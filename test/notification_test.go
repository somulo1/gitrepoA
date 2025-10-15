package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

// setupNotificationTestDB creates an in-memory SQLite database for testing
func setupNotificationTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return db
}

type NotificationTestSuite struct {
	suite.Suite
	db     *sql.DB
	router *gin.Engine
}

func (suite *NotificationTestSuite) SetupSuite() {
	suite.db = setupNotificationTestDB()

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup notification routes with placeholder handlers
	apiGroup := suite.router.Group("/api/v1")
	{
		// Use placeholder handlers since notification API handlers don't exist yet
		apiGroup.GET("/notifications", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "data": []interface{}{}})
		})
		apiGroup.POST("/notifications", func(c *gin.Context) {
			c.JSON(201, gin.H{"success": true, "data": gin.H{"id": "test-notification"}})
		})
		apiGroup.GET("/notifications/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "data": gin.H{"id": c.Param("id")}})
		})
		apiGroup.PUT("/notifications/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "data": gin.H{"id": c.Param("id")}})
		})
		apiGroup.DELETE("/notifications/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "message": "Notification deleted"})
		})
	}
}

func (suite *NotificationTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
	// Insert test data
	suite.insertTestData()
}

func (suite *NotificationTestSuite) TearDownSuite() {
	suite.db.Close()
}

func (suite *NotificationTestSuite) cleanupTestData() {
	tables := []string{"notifications", "notification_settings", "users"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id LIKE 'test-%%'", table))
		suite.NoError(err)
	}
}

func (suite *NotificationTestSuite) insertTestData() {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	// Insert test users
	users := []struct {
		id, email, phone, firstName, lastName string
	}{
		{"test-user-123", "test@example.com", "+254712345678", "Test", "User"},
		{"test-user-456", "jane@example.com", "+254712345679", "Jane", "Smith"},
		{"test-user-789", "bob@example.com", "+254712345680", "Bob", "Johnson"},
	}

	for _, user := range users {
		_, err := suite.db.Exec(`
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, user.email, user.phone, user.firstName, user.lastName, string(hashedPassword), "user", "active", time.Now())
		suite.NoError(err)
	}

	// Insert test notifications
	notifications := []struct {
		id, userId, title, message, notificationType, status, relatedId string
		priority                                                        int
		isRead                                                          bool
		scheduledAt                                                     *time.Time
		expiresAt                                                       *time.Time
		metadata                                                        string
	}{
		{"test-notification-123", "test-user-123", "Welcome!", "Welcome to VaultKe", "system", "sent", "", 1, false, nil, nil, "{}"},
		{"test-notification-456", "test-user-123", "Payment Received", "You received a payment of KES 5000", "payment", "sent", "test-transaction-123", 2, false, nil, nil, "{}"},
		{"test-notification-789", "test-user-123", "Meeting Reminder", "Monthly meeting starts in 1 hour", "meeting", "sent", "test-meeting-123", 3, true, nil, nil, "{}"},
		{"test-notification-012", "test-user-456", "Chama Invitation", "You have been invited to join Test Chama", "chama", "sent", "test-chama-123", 2, false, nil, nil, "{}"},
		{"test-notification-345", "test-user-123", "Scheduled Notification", "This is a scheduled notification", "system", "scheduled", "", 1, false, func() *time.Time { t := time.Now().Add(24 * time.Hour); return &t }(), nil, "{}"},
	}

	for _, notification := range notifications {
		_, err := suite.db.Exec(`
			INSERT INTO notifications (id, user_id, title, message, type, status, related_id, priority, is_read, scheduled_at, expires_at, metadata, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, notification.id, notification.userId, notification.title, notification.message, notification.notificationType, notification.status, notification.relatedId, notification.priority, notification.isRead, notification.scheduledAt, notification.expiresAt, notification.metadata, time.Now())
		suite.NoError(err)
	}

	// Insert test notification settings
	settings := []struct {
		id, userId                                                                  string
		emailNotifications, smsNotifications, pushNotifications, inAppNotifications bool
		meetingReminders, paymentAlerts, chamaUpdates, systemAlerts                 bool
		quietHoursStart, quietHoursEnd                                              string
	}{
		{"test-settings-123", "test-user-123", true, true, true, true, true, true, true, true, "22:00", "07:00"},
		{"test-settings-456", "test-user-456", true, false, true, true, true, true, false, true, "23:00", "06:00"},
	}

	for _, setting := range settings {
		_, err := suite.db.Exec(`
			INSERT INTO notification_settings (id, user_id, email_notifications, sms_notifications, push_notifications, in_app_notifications, meeting_reminders, payment_alerts, chama_updates, system_alerts, quiet_hours_start, quiet_hours_end, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, setting.id, setting.userId, setting.emailNotifications, setting.smsNotifications, setting.pushNotifications, setting.inAppNotifications, setting.meetingReminders, setting.paymentAlerts, setting.chamaUpdates, setting.systemAlerts, setting.quietHoursStart, setting.quietHoursEnd, time.Now())
		suite.NoError(err)
	}
}

func (suite *NotificationTestSuite) TestGetNotifications() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful notifications retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with pagination",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with type filter",
			queryParams:    "?type=payment",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with status filter",
			queryParams:    "?status=sent",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with read filter",
			queryParams:    "?isRead=false",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with priority filter",
			queryParams:    "?priority=high",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with date range",
			queryParams:    "?startDate=2024-01-01&endDate=2024-12-31",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notifications with sorting",
			queryParams:    "?sortBy=createdAt&sortOrder=desc",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/notifications"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				assert.NotNil(suite.T(), response["data"])

				data := response["data"].(map[string]interface{})
				notifications := data["notifications"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(notifications), 1)

				// Check pagination info
				assert.Contains(suite.T(), data, "pagination")
				pagination := data["pagination"].(map[string]interface{})
				assert.Contains(suite.T(), pagination, "total")
				assert.Contains(suite.T(), pagination, "limit")
				assert.Contains(suite.T(), pagination, "offset")
			}
		})
	}
}

func (suite *NotificationTestSuite) TestCreateNotification() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful notification creation",
			requestBody: map[string]interface{}{
				"title":       "New Test Notification",
				"message":     "This is a test notification",
				"type":        "system",
				"priority":    2,
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "notification with scheduling",
			requestBody: map[string]interface{}{
				"title":       "Scheduled Notification",
				"message":     "This notification will be sent later",
				"type":        "system",
				"priority":    1,
				"recipientId": "test-user-456",
				"scheduledAt": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "notification with expiration",
			requestBody: map[string]interface{}{
				"title":       "Expiring Notification",
				"message":     "This notification will expire",
				"type":        "system",
				"priority":    2,
				"recipientId": "test-user-456",
				"expiresAt":   time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "notification creation with missing title",
			requestBody: map[string]interface{}{
				"message":     "Notification without title",
				"type":        "system",
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "notification creation with empty title",
			requestBody: map[string]interface{}{
				"title":       "",
				"message":     "Notification with empty title",
				"type":        "system",
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "notification creation with missing message",
			requestBody: map[string]interface{}{
				"title":       "Test Notification",
				"type":        "system",
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Message is required",
		},
		{
			name: "notification creation with invalid type",
			requestBody: map[string]interface{}{
				"title":       "Test Notification",
				"message":     "Test message",
				"type":        "invalid-type",
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid notification type",
		},
		{
			name: "notification creation with missing recipient",
			requestBody: map[string]interface{}{
				"title":   "Test Notification",
				"message": "Test message",
				"type":    "system",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Recipient ID is required",
		},
		{
			name: "notification creation with invalid recipient",
			requestBody: map[string]interface{}{
				"title":       "Test Notification",
				"message":     "Test message",
				"type":        "system",
				"recipientId": "nonexistent-user",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Recipient not found",
		},
		{
			name: "notification creation with invalid priority",
			requestBody: map[string]interface{}{
				"title":       "Test Notification",
				"message":     "Test message",
				"type":        "system",
				"priority":    10, // Invalid priority
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Priority must be between 1 and 5",
		},
		{
			name: "notification creation with past scheduled time",
			requestBody: map[string]interface{}{
				"title":       "Past Scheduled Notification",
				"message":     "This notification is scheduled in the past",
				"type":        "system",
				"recipientId": "test-user-456",
				"scheduledAt": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Scheduled time must be in the future",
		},
		{
			name: "notification creation with past expiration",
			requestBody: map[string]interface{}{
				"title":       "Past Expiring Notification",
				"message":     "This notification expires in the past",
				"type":        "system",
				"recipientId": "test-user-456",
				"expiresAt":   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Expiration time must be in the future",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				notification := data["notification"].(map[string]interface{})
				assert.NotNil(suite.T(), notification["id"])
				assert.Equal(suite.T(), tt.requestBody["title"], notification["title"])
			}
		})
	}
}

func (suite *NotificationTestSuite) TestGetNotification() {
	tests := []struct {
		name           string
		notificationID string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful notification retrieval",
			notificationID: "test-notification-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "notification not found",
			notificationID: "nonexistent-notification",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Notification not found",
		},
		{
			name:           "access other user's notification",
			notificationID: "test-notification-012", // Belongs to test-user-456
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/notifications/"+tt.notificationID, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				notification := data["notification"].(map[string]interface{})
				assert.Equal(suite.T(), tt.notificationID, notification["id"])
			}
		})
	}
}

func (suite *NotificationTestSuite) TestUpdateNotification() {
	tests := []struct {
		name           string
		notificationID string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful notification update",
			notificationID: "test-notification-123",
			requestBody: map[string]interface{}{
				"title":    "Updated Notification Title",
				"message":  "Updated notification message",
				"priority": 3,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "update non-existent notification",
			notificationID: "nonexistent-notification",
			requestBody: map[string]interface{}{
				"title": "Updated Title",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Notification not found",
		},
		{
			name:           "update other user's notification",
			notificationID: "test-notification-012",
			requestBody: map[string]interface{}{
				"title": "Unauthorized Update",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "update with invalid data",
			notificationID: "test-notification-123",
			requestBody: map[string]interface{}{
				"priority": 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Priority must be between 1 and 5",
		},
		{
			name:           "update sent notification",
			notificationID: "test-notification-456",
			requestBody: map[string]interface{}{
				"title": "Update Sent Notification",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot update sent notification",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/notifications/"+tt.notificationID, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *NotificationTestSuite) TestDeleteNotification() {
	tests := []struct {
		name           string
		notificationID string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful notification deletion",
			notificationID: "test-notification-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete non-existent notification",
			notificationID: "nonexistent-notification",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Notification not found",
		},
		{
			name:           "delete other user's notification",
			notificationID: "test-notification-012",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/notifications/"+tt.notificationID, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *NotificationTestSuite) TestMarkAsRead() {
	tests := []struct {
		name           string
		notificationID string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful mark as read",
			notificationID: "test-notification-456", // Unread notification
			expectedStatus: http.StatusOK,
		},
		{
			name:           "mark non-existent notification as read",
			notificationID: "nonexistent-notification",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Notification not found",
		},
		{
			name:           "mark other user's notification as read",
			notificationID: "test-notification-012",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "mark already read notification as read",
			notificationID: "test-notification-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Notification is already read",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("PUT", "/api/v1/notifications/"+tt.notificationID+"/read", nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *NotificationTestSuite) TestMarkAsUnread() {
	tests := []struct {
		name           string
		notificationID string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful mark as unread",
			notificationID: "test-notification-789", // Read notification
			expectedStatus: http.StatusOK,
		},
		{
			name:           "mark non-existent notification as unread",
			notificationID: "nonexistent-notification",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Notification not found",
		},
		{
			name:           "mark other user's notification as unread",
			notificationID: "test-notification-012",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "mark already unread notification as unread",
			notificationID: "test-notification-456",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Notification is already unread",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("PUT", "/api/v1/notifications/"+tt.notificationID+"/unread", nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *NotificationTestSuite) TestMarkAllAsRead() {
	req, _ := http.NewRequest("POST", "/api/v1/notifications/mark-all-read", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(suite.T(), data, "updated")
	updated := data["updated"].(float64)
	assert.Greater(suite.T(), updated, 0.0)
}

func (suite *NotificationTestSuite) TestGetUnreadCount() {
	req, _ := http.NewRequest("GET", "/api/v1/notifications/unread-count", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(suite.T(), data, "count")
	count := data["count"].(float64)
	assert.GreaterOrEqual(suite.T(), count, 0.0)
}

func (suite *NotificationTestSuite) TestSendNotification() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful notification sending",
			requestBody: map[string]interface{}{
				"title":       "Urgent Message",
				"message":     "This is an urgent notification",
				"type":        "system",
				"priority":    3,
				"recipientId": "test-user-456",
				"channels":    []string{"email", "push"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "send to multiple recipients",
			requestBody: map[string]interface{}{
				"title":        "Broadcast Message",
				"message":      "This is a broadcast notification",
				"type":         "system",
				"priority":     2,
				"recipientIds": []string{"test-user-456", "test-user-789"},
				"channels":     []string{"push"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "send with missing title",
			requestBody: map[string]interface{}{
				"message":     "Message without title",
				"type":        "system",
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "send with missing message",
			requestBody: map[string]interface{}{
				"title":       "Title without message",
				"type":        "system",
				"recipientId": "test-user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Message is required",
		},
		{
			name: "send with missing recipient",
			requestBody: map[string]interface{}{
				"title":   "Message without recipient",
				"message": "This message has no recipient",
				"type":    "system",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Recipient is required",
		},
		{
			name: "send with invalid channel",
			requestBody: map[string]interface{}{
				"title":       "Message with invalid channel",
				"message":     "This message has invalid channel",
				"type":        "system",
				"recipientId": "test-user-456",
				"channels":    []string{"invalid-channel"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid channel",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/notifications/send", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *NotificationTestSuite) TestGetNotificationSettings() {
	req, _ := http.NewRequest("GET", "/api/v1/notifications/settings", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	settings := data["settings"].(map[string]interface{})
	assert.Contains(suite.T(), settings, "emailNotifications")
	assert.Contains(suite.T(), settings, "smsNotifications")
	assert.Contains(suite.T(), settings, "pushNotifications")
	assert.Contains(suite.T(), settings, "inAppNotifications")
}

func (suite *NotificationTestSuite) TestUpdateNotificationSettings() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful settings update",
			requestBody: map[string]interface{}{
				"emailNotifications": false,
				"smsNotifications":   true,
				"pushNotifications":  true,
				"meetingReminders":   false,
				"paymentAlerts":      true,
				"quietHoursStart":    "23:30",
				"quietHoursEnd":      "06:30",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update with invalid quiet hours",
			requestBody: map[string]interface{}{
				"quietHoursStart": "25:00",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid quiet hours format",
		},
		{
			name: "update with invalid boolean value",
			requestBody: map[string]interface{}{
				"emailNotifications": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid boolean value",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/notifications/settings", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *NotificationTestSuite) TestConcurrentNotificationCreation() {
	// Test concurrent notification creation
	done := make(chan bool)
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			requestBody := map[string]interface{}{
				"title":       fmt.Sprintf("Concurrent Notification %d", index),
				"message":     fmt.Sprintf("This is concurrent notification %d", index),
				"type":        "system",
				"priority":    2,
				"recipientId": "test-user-456",
			}

			jsonBody, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)
			results <- w.Code
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	close(results)

	// All should succeed
	var successCount int
	for code := range results {
		if code == http.StatusCreated {
			successCount++
		}
	}

	assert.Equal(suite.T(), 10, successCount)
}

func (suite *NotificationTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func TestNotificationSuite(t *testing.T) {
	suite.Run(t, new(NotificationTestSuite))
}
