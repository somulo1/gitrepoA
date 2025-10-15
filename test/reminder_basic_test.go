package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"vaultke-backend/internal/api"
	"vaultke-backend/test/helpers"
)

type ReminderBasicTestSuite struct {
	suite.Suite
	testDB *helpers.TestDatabase
	router *gin.Engine
}

func (suite *ReminderBasicTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDatabase()

	// Manually create reminders table if it doesn't exist
	suite.createRemindersTable()

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Add middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.testDB.DB)
		c.Next()
	})

	// Setup reminder routes
	reminderHandlers := api.NewReminderHandlers(suite.testDB.DB)
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

func (suite *ReminderBasicTestSuite) TearDownSuite() {
	if suite.testDB != nil {
		suite.testDB.Close()
	}
}

func (suite *ReminderBasicTestSuite) SetupTest() {
	// Clean up test data before each test
	if suite.testDB != nil {
		suite.testDB.CleanupTestData()
	}
	// Insert test data
	suite.insertTestData()
}

func (suite *ReminderBasicTestSuite) createRemindersTable() {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS reminders (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		reminder_type TEXT NOT NULL DEFAULT 'once',
		scheduled_at DATETIME NOT NULL,
		is_enabled BOOLEAN DEFAULT TRUE,
		is_completed BOOLEAN DEFAULT FALSE,
		notification_sent BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	_, err := suite.testDB.DB.Exec(createTableSQL)
	suite.NoError(err)

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_reminders_user_id ON reminders(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_at ON reminders(scheduled_at);",
		"CREATE INDEX IF NOT EXISTS idx_reminders_type ON reminders(reminder_type);",
		"CREATE INDEX IF NOT EXISTS idx_reminders_enabled ON reminders(is_enabled);",
		"CREATE INDEX IF NOT EXISTS idx_reminders_completed ON reminders(is_completed);",
	}

	for _, indexSQL := range indexes {
		_, err := suite.testDB.DB.Exec(indexSQL)
		suite.NoError(err)
	}
}

func (suite *ReminderBasicTestSuite) insertTestData() {
	// Create test user
	err := suite.testDB.CreateTestUser(helpers.TestUser{
		ID:       "test-user-123",
		Email:    "test@example.com",
		Phone:    "+254712345678",
		Role:     "user",
		Password: "password123",
	})
	suite.NoError(err)
}

func (suite *ReminderBasicTestSuite) TestCreateReminder() {
	suite.Run("successful_reminder_creation", func() {
		reminderData := map[string]interface{}{
			"title":        "Test Reminder",
			"description":  "This is a test reminder",
			"reminderType": "once",
			"scheduledAt":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}
		jsonData, _ := json.Marshal(reminderData)

		req, _ := http.NewRequest("POST", "/api/v1/reminders", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), response["success"].(bool))
		assert.NotNil(suite.T(), response["data"])
	})

	suite.Run("reminder_creation_with_missing_title", func() {
		reminderData := map[string]interface{}{
			"description":  "This is a test reminder",
			"reminderType": "once",
			"scheduledAt":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}
		jsonData, _ := json.Marshal(reminderData)

		req, _ := http.NewRequest("POST", "/api/v1/reminders", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))
		assert.Contains(suite.T(), response["error"], "Invalid request data")
	})

	suite.Run("reminder_creation_with_past_scheduled_time", func() {
		reminderData := map[string]interface{}{
			"title":        "Test Reminder",
			"description":  "This is a test reminder",
			"reminderType": "once",
			"scheduledAt":  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		}
		jsonData, _ := json.Marshal(reminderData)

		req, _ := http.NewRequest("POST", "/api/v1/reminders", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))
		assert.Contains(suite.T(), response["error"], "scheduled time must be in the future")
	})
}

func (suite *ReminderBasicTestSuite) TestGetReminders() {
	req, _ := http.NewRequest("GET", "/api/v1/reminders", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	// Data field should exist, even if it's an empty array or nil
	_, exists := response["data"]
	assert.True(suite.T(), exists)
}

func (suite *ReminderBasicTestSuite) TestGetRemindersWithPagination() {
	req, _ := http.NewRequest("GET", "/api/v1/reminders?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	// Data field should exist, even if it's an empty array or nil
	_, exists := response["data"]
	assert.True(suite.T(), exists)
}

func (suite *ReminderBasicTestSuite) TestGetReminder() {
	suite.Run("reminder_not_found", func() {
		req, _ := http.NewRequest("GET", "/api/v1/reminders/nonexistent-reminder", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))
		assert.Contains(suite.T(), response["error"], "Reminder not found")
	})
}

func (suite *ReminderBasicTestSuite) TestUpdateReminder() {
	suite.Run("update_non-existent_reminder", func() {
		updateData := map[string]interface{}{
			"title": "Updated Reminder",
		}
		jsonData, _ := json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", "/api/v1/reminders/nonexistent-reminder", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))
		assert.Contains(suite.T(), response["error"], "Reminder not found")
	})
}

func (suite *ReminderBasicTestSuite) TestDeleteReminder() {
	suite.Run("delete_non-existent_reminder", func() {
		req, _ := http.NewRequest("DELETE", "/api/v1/reminders/nonexistent-reminder", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))
		assert.Contains(suite.T(), response["error"], "Reminder not found")
	})
}

func (suite *ReminderBasicTestSuite) TestToggleReminder() {
	suite.Run("toggle_non-existent_reminder", func() {
		req, _ := http.NewRequest("POST", "/api/v1/reminders/nonexistent-reminder/toggle", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))
		assert.Contains(suite.T(), response["error"], "Reminder not found")
	})
}

func TestReminderBasicSuite(t *testing.T) {
	suite.Run(t, new(ReminderBasicTestSuite))
}
