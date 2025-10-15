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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

	"vaultke-backend/internal/api"
	"vaultke-backend/test/helpers"
)

type ReminderTestSuite struct {
	suite.Suite
	testDB *helpers.TestDatabase
	db     *sql.DB
	router *gin.Engine
}

func (suite *ReminderTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDatabase()
	suite.db = suite.testDB.DB

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup reminder routes
	reminderHandlers := api.NewReminderHandlers(suite.db)
	apiGroup := suite.router.Group("/api/v1")
	{
		apiGroup.GET("/reminders", reminderHandlers.GetUserReminders)
		apiGroup.POST("/reminders", reminderHandlers.CreateReminder)
		apiGroup.GET("/reminders/:id", reminderHandlers.GetReminder)
		apiGroup.PUT("/reminders/:id", reminderHandlers.UpdateReminder)
		apiGroup.DELETE("/reminders/:id", reminderHandlers.DeleteReminder)
		apiGroup.POST("/reminders/:id/toggle", reminderHandlers.ToggleReminder)
	}
}

func (suite *ReminderTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
	// Insert test data
	suite.insertTestData()
}

func (suite *ReminderTestSuite) TearDownSuite() {
	suite.db.Close()
}

func (suite *ReminderTestSuite) cleanupTestData() {
	_, err := suite.db.Exec("DELETE FROM reminders WHERE id LIKE 'test-%'")
	suite.NoError(err)
	_, err = suite.db.Exec("DELETE FROM users WHERE id LIKE 'test-%'")
	suite.NoError(err)
}

func (suite *ReminderTestSuite) insertTestData() {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	// Insert test users
	users := []struct {
		id, email, phone, firstName, lastName string
	}{
		{"test-user-123", "test@example.com", "+254712345678", "Test", "User"},
		{"test-user-456", "jane@example.com", "+254712345679", "Jane", "Smith"},
	}

	for _, user := range users {
		_, err := suite.db.Exec(`
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, user.email, user.phone, user.firstName, user.lastName, string(hashedPassword), "user", "active", time.Now())
		suite.NoError(err)
	}

	// Insert test reminders
	reminders := []struct {
		id, userId, title, description, reminderType string
		scheduledAt                                  time.Time
		isEnabled, isCompleted, notificationSent     bool
	}{
		{"test-reminder-123", "test-user-123", "Monthly Meeting", "Attend the monthly chama meeting", "once", time.Now().Add(24 * time.Hour), true, false, false},
		{"test-reminder-456", "test-user-123", "Pay Bills", "Pay electricity and water bills", "monthly", time.Now().Add(7 * 24 * time.Hour), true, false, false},
		{"test-reminder-789", "test-user-123", "Completed Task", "This task is completed", "once", time.Now().Add(-24 * time.Hour), true, true, true},
		{"test-reminder-012", "test-user-123", "Disabled Reminder", "This reminder is disabled", "weekly", time.Now().Add(48 * time.Hour), false, false, false},
		{"test-reminder-345", "test-user-123", "Overdue Reminder", "This reminder is overdue", "once", time.Now().Add(-48 * time.Hour), true, false, false},
		{"test-reminder-678", "test-user-456", "Other User Reminder", "Reminder for another user", "once", time.Now().Add(12 * time.Hour), true, false, false},
	}

	for _, reminder := range reminders {
		_, err := suite.db.Exec(`
			INSERT INTO reminders (id, user_id, title, description, reminder_type, scheduled_at, is_enabled, is_completed, notification_sent, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, reminder.id, reminder.userId, reminder.title, reminder.description, reminder.reminderType, reminder.scheduledAt, reminder.isEnabled, reminder.isCompleted, reminder.notificationSent, time.Now())
		suite.NoError(err)
	}
}

func (suite *ReminderTestSuite) TestGetReminders() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful reminders retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with pagination",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with type filter",
			queryParams:    "?type=once",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with status filter",
			queryParams:    "?status=pending",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with completed filter",
			queryParams:    "?completed=false",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with enabled filter",
			queryParams:    "?enabled=true",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with date range",
			queryParams:    "?startDate=2024-01-01&endDate=2024-12-31",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with sorting",
			queryParams:    "?sortBy=scheduledAt&sortOrder=asc",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminders with search query",
			queryParams:    "?q=meeting",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/reminders"+tt.queryParams, nil)
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
				reminders := data["reminders"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(reminders), 1)

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

func (suite *ReminderTestSuite) TestCreateReminder() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful reminder creation",
			requestBody: map[string]interface{}{
				"title":        "New Test Reminder",
				"description":  "This is a test reminder",
				"reminderType": "once",
				"scheduledAt":  time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "recurring reminder creation",
			requestBody: map[string]interface{}{
				"title":        "Weekly Reminder",
				"description":  "This is a weekly reminder",
				"reminderType": "weekly",
				"scheduledAt":  time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "monthly reminder creation",
			requestBody: map[string]interface{}{
				"title":        "Monthly Reminder",
				"description":  "This is a monthly reminder",
				"reminderType": "monthly",
				"scheduledAt":  time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "reminder creation with missing title",
			requestBody: map[string]interface{}{
				"description":  "Reminder without title",
				"reminderType": "once",
				"scheduledAt":  time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "reminder creation with empty title",
			requestBody: map[string]interface{}{
				"title":        "",
				"description":  "Reminder with empty title",
				"reminderType": "once",
				"scheduledAt":  time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "reminder creation with missing scheduled time",
			requestBody: map[string]interface{}{
				"title":        "Test Reminder",
				"description":  "Reminder without scheduled time",
				"reminderType": "once",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Scheduled time is required",
		},
		{
			name: "reminder creation with invalid reminder type",
			requestBody: map[string]interface{}{
				"title":        "Test Reminder",
				"description":  "Reminder with invalid type",
				"reminderType": "invalid-type",
				"scheduledAt":  time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid reminder type",
		},
		{
			name: "reminder creation with past scheduled time",
			requestBody: map[string]interface{}{
				"title":        "Past Reminder",
				"description":  "Reminder scheduled in the past",
				"reminderType": "once",
				"scheduledAt":  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Scheduled time must be in the future",
		},
		{
			name: "reminder creation with invalid scheduled time format",
			requestBody: map[string]interface{}{
				"title":        "Test Reminder",
				"description":  "Reminder with invalid time format",
				"reminderType": "once",
				"scheduledAt":  "invalid-time-format",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid scheduled time format",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/reminders", bytes.NewBuffer(jsonBody))
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
				reminder := data["reminder"].(map[string]interface{})
				assert.NotNil(suite.T(), reminder["id"])
				assert.Equal(suite.T(), tt.requestBody["title"], reminder["title"])
			}
		})
	}
}

func (suite *ReminderTestSuite) TestGetReminder() {
	tests := []struct {
		name           string
		reminderID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful reminder retrieval",
			reminderID:     "test-reminder-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "reminder not found",
			reminderID:     "nonexistent-reminder",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:           "access other user's reminder",
			reminderID:     "test-reminder-678", // Belongs to test-user-456
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/reminders/"+tt.reminderID, nil)
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
				reminder := data["reminder"].(map[string]interface{})
				assert.Equal(suite.T(), tt.reminderID, reminder["id"])
			}
		})
	}
}

func (suite *ReminderTestSuite) TestUpdateReminder() {
	tests := []struct {
		name           string
		reminderID     string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "successful reminder update",
			reminderID: "test-reminder-123",
			requestBody: map[string]interface{}{
				"title":       "Updated Reminder Title",
				"description": "Updated reminder description",
				"scheduledAt": time.Now().Add(72 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "update non-existent reminder",
			reminderID: "nonexistent-reminder",
			requestBody: map[string]interface{}{
				"title": "Updated Title",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:       "update other user's reminder",
			reminderID: "test-reminder-678",
			requestBody: map[string]interface{}{
				"title": "Unauthorized Update",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:       "update with invalid data",
			reminderID: "test-reminder-123",
			requestBody: map[string]interface{}{
				"reminderType": "invalid-type",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid reminder type",
		},
		{
			name:       "update with past scheduled time",
			reminderID: "test-reminder-123",
			requestBody: map[string]interface{}{
				"scheduledAt": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Scheduled time must be in the future",
		},
		{
			name:       "update completed reminder",
			reminderID: "test-reminder-789",
			requestBody: map[string]interface{}{
				"title": "Update Completed Reminder",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot update completed reminder",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/reminders/"+tt.reminderID, bytes.NewBuffer(jsonBody))
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

func (suite *ReminderTestSuite) TestDeleteReminder() {
	tests := []struct {
		name           string
		reminderID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful reminder deletion",
			reminderID:     "test-reminder-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete non-existent reminder",
			reminderID:     "nonexistent-reminder",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:           "delete other user's reminder",
			reminderID:     "test-reminder-678",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/reminders/"+tt.reminderID, nil)
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

func (suite *ReminderTestSuite) TestCompleteReminder() {
	tests := []struct {
		name           string
		reminderID     string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "successful reminder completion",
			reminderID: "test-reminder-123",
			requestBody: map[string]interface{}{
				"notes": "Task completed successfully",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "complete non-existent reminder",
			reminderID: "nonexistent-reminder",
			requestBody: map[string]interface{}{
				"notes": "Completed",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:       "complete other user's reminder",
			reminderID: "test-reminder-678",
			requestBody: map[string]interface{}{
				"notes": "Unauthorized completion",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:       "complete already completed reminder",
			reminderID: "test-reminder-789",
			requestBody: map[string]interface{}{
				"notes": "Already completed",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder is already completed",
		},
		{
			name:       "complete disabled reminder",
			reminderID: "test-reminder-012",
			requestBody: map[string]interface{}{
				"notes": "Complete disabled reminder",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot complete disabled reminder",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/reminders/"+tt.reminderID+"/complete", bytes.NewBuffer(jsonBody))
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

func (suite *ReminderTestSuite) TestSnoozeReminder() {
	tests := []struct {
		name           string
		reminderID     string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "successful reminder snoozing",
			reminderID: "test-reminder-123",
			requestBody: map[string]interface{}{
				"snoozeUntil": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "snooze non-existent reminder",
			reminderID: "nonexistent-reminder",
			requestBody: map[string]interface{}{
				"snoozeUntil": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:       "snooze other user's reminder",
			reminderID: "test-reminder-678",
			requestBody: map[string]interface{}{
				"snoozeUntil": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "snooze with missing snooze time",
			reminderID:     "test-reminder-123",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Snooze time is required",
		},
		{
			name:       "snooze with past snooze time",
			reminderID: "test-reminder-123",
			requestBody: map[string]interface{}{
				"snoozeUntil": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Snooze time must be in the future",
		},
		{
			name:       "snooze completed reminder",
			reminderID: "test-reminder-789",
			requestBody: map[string]interface{}{
				"snoozeUntil": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot snooze completed reminder",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/reminders/"+tt.reminderID+"/snooze", bytes.NewBuffer(jsonBody))
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

func (suite *ReminderTestSuite) TestEnableReminder() {
	tests := []struct {
		name           string
		reminderID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful reminder enabling",
			reminderID:     "test-reminder-012", // Disabled reminder
			expectedStatus: http.StatusOK,
		},
		{
			name:           "enable non-existent reminder",
			reminderID:     "nonexistent-reminder",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:           "enable other user's reminder",
			reminderID:     "test-reminder-678",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "enable already enabled reminder",
			reminderID:     "test-reminder-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder is already enabled",
		},
		{
			name:           "enable completed reminder",
			reminderID:     "test-reminder-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot enable completed reminder",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("PUT", "/api/v1/reminders/"+tt.reminderID+"/enable", nil)
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

func (suite *ReminderTestSuite) TestDisableReminder() {
	tests := []struct {
		name           string
		reminderID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful reminder disabling",
			reminderID:     "test-reminder-123", // Enabled reminder
			expectedStatus: http.StatusOK,
		},
		{
			name:           "disable non-existent reminder",
			reminderID:     "nonexistent-reminder",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Reminder not found",
		},
		{
			name:           "disable other user's reminder",
			reminderID:     "test-reminder-678",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "disable already disabled reminder",
			reminderID:     "test-reminder-012",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder is already disabled",
		},
		{
			name:           "disable completed reminder",
			reminderID:     "test-reminder-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot disable completed reminder",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("PUT", "/api/v1/reminders/"+tt.reminderID+"/disable", nil)
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

func (suite *ReminderTestSuite) TestGetUpcomingReminders() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful upcoming reminders retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "upcoming reminders with limit",
			queryParams:    "?limit=5",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "upcoming reminders with time range",
			queryParams:    "?hours=24",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/reminders/upcoming"+tt.queryParams, nil)
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
				reminders := data["reminders"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(reminders), 0)
			}
		})
	}
}

func (suite *ReminderTestSuite) TestGetOverdueReminders() {
	req, _ := http.NewRequest("GET", "/api/v1/reminders/overdue", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	reminders := data["reminders"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(reminders), 0)
}

func (suite *ReminderTestSuite) TestBulkUpdateReminders() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful bulk update",
			requestBody: map[string]interface{}{
				"reminderIds": []string{"test-reminder-123", "test-reminder-456"},
				"updates": map[string]interface{}{
					"isEnabled": false,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "bulk update with missing reminder IDs",
			requestBody: map[string]interface{}{
				"updates": map[string]interface{}{
					"isEnabled": false,
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder IDs are required",
		},
		{
			name: "bulk update with empty reminder IDs",
			requestBody: map[string]interface{}{
				"reminderIds": []string{},
				"updates": map[string]interface{}{
					"isEnabled": false,
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder IDs are required",
		},
		{
			name: "bulk update with missing updates",
			requestBody: map[string]interface{}{
				"reminderIds": []string{"test-reminder-123"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Updates are required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/reminders/bulk-update", bytes.NewBuffer(jsonBody))
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
				assert.Contains(suite.T(), data, "updated")
			}
		})
	}
}

func (suite *ReminderTestSuite) TestBulkDeleteReminders() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful bulk delete",
			requestBody: map[string]interface{}{
				"reminderIds": []string{"test-reminder-123", "test-reminder-456"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "bulk delete with missing reminder IDs",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder IDs are required",
		},
		{
			name: "bulk delete with empty reminder IDs",
			requestBody: map[string]interface{}{
				"reminderIds": []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Reminder IDs are required",
		},
		{
			name: "bulk delete with non-existent reminders",
			requestBody: map[string]interface{}{
				"reminderIds": []string{"nonexistent-reminder-1", "nonexistent-reminder-2"},
			},
			expectedStatus: http.StatusOK, // Should not fail, just return 0 deleted
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("DELETE", "/api/v1/reminders/bulk-delete", bytes.NewBuffer(jsonBody))
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
				assert.Contains(suite.T(), data, "deleted")
			}
		})
	}
}

func (suite *ReminderTestSuite) TestConcurrentReminderCreation() {
	// Test concurrent reminder creation
	done := make(chan bool)
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			requestBody := map[string]interface{}{
				"title":        fmt.Sprintf("Concurrent Reminder %d", index),
				"description":  fmt.Sprintf("This is concurrent reminder %d", index),
				"reminderType": "once",
				"scheduledAt":  time.Now().Add(time.Duration(index+1) * time.Hour).Format(time.RFC3339),
			}

			jsonBody, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/reminders", bytes.NewBuffer(jsonBody))
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

func (suite *ReminderTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/reminders", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func TestReminderSuite(t *testing.T) {
	suite.Run(t, new(ReminderTestSuite))
}
