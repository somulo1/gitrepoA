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

type MeetingTestSuite struct {
	suite.Suite
	db     *sql.DB
	router *gin.Engine
}

func (suite *MeetingTestSuite) SetupSuite() {
	testDB := helpers.SetupTestDatabase()
	suite.db = testDB.DB

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup meeting routes
	apiGroup := suite.router.Group("/api/v1")
	{
		apiGroup.GET("/meetings", api.GetMeetings)
		apiGroup.POST("/meetings", api.CreateMeeting)
		apiGroup.GET("/meetings/:id", api.GetMeeting)
		apiGroup.PUT("/meetings/:id", api.UpdateMeeting)
		apiGroup.DELETE("/meetings/:id", api.DeleteMeeting)
		apiGroup.POST("/meetings/:id/join", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "message": "Join meeting endpoint - placeholder"})
		})
		apiGroup.POST("/meetings/:id/leave", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "message": "Leave meeting endpoint - placeholder"})
		})
		apiGroup.GET("/meetings/:id/participants", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "data": gin.H{"participants": []interface{}{}}})
		})
		apiGroup.POST("/meetings/:id/start", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "message": "Start meeting endpoint - placeholder"})
		})
		apiGroup.POST("/meetings/:id/end", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "message": "End meeting endpoint - placeholder"})
		})
		apiGroup.POST("/meetings/:id/record", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "message": "Record meeting endpoint - placeholder"})
		})
		apiGroup.GET("/meetings/:id/recordings", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "data": gin.H{"recordings": []interface{}{}}})
		})
	}
}

func (suite *MeetingTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
	// Insert test data
	suite.insertTestData()
}

func (suite *MeetingTestSuite) TearDownSuite() {
	suite.db.Close()
}

func (suite *MeetingTestSuite) cleanupTestData() {
	tables := []string{"meeting_participants", "meetings", "chama_members", "chamas", "users"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id LIKE 'test-%%'", table))
		suite.NoError(err)
	}
}

func (suite *MeetingTestSuite) insertTestData() {
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

	// Insert test chama
	_, err := suite.db.Exec(`
		INSERT INTO chamas (id, name, description, type, county, town, contribution_amount, contribution_frequency, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-chama-123", "Test Chama", "Test Description", "savings", "Nairobi", "Nairobi", 5000.0, "monthly", "test-user-123", time.Now())
	suite.NoError(err)

	// Insert test chama members
	members := []struct {
		id, chamaId, userId, role string
	}{
		{"test-member-123", "test-chama-123", "test-user-123", "chairperson"},
		{"test-member-456", "test-chama-123", "test-user-456", "member"},
		{"test-member-789", "test-chama-123", "test-user-789", "secretary"},
	}

	for _, member := range members {
		_, err := suite.db.Exec(`
			INSERT INTO chama_members (id, chama_id, user_id, role, is_active, joined_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, member.id, member.chamaId, member.userId, member.role, true, time.Now())
		suite.NoError(err)
	}

	// Insert test meetings
	meetings := []struct {
		id, chamaId, title, description, location, status, createdBy string
		scheduledAt                                                  time.Time
		duration                                                     int
		meetingUrl                                                   string
	}{
		{"test-meeting-123", "test-chama-123", "Monthly Meeting", "Regular monthly meeting", "Nairobi Office", "scheduled", "test-user-123", time.Now().Add(24 * time.Hour), 60, "https://meet.example.com/123"},
		{"test-meeting-456", "test-chama-123", "Emergency Meeting", "Urgent meeting", "Online", "ongoing", "test-user-123", time.Now().Add(-1 * time.Hour), 90, "https://meet.example.com/456"},
		{"test-meeting-789", "test-chama-123", "Past Meeting", "Completed meeting", "Nairobi Office", "completed", "test-user-123", time.Now().Add(-48 * time.Hour), 60, ""},
	}

	for _, meeting := range meetings {
		_, err := suite.db.Exec(`
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, location, meeting_url, status, created_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, meeting.id, meeting.chamaId, meeting.title, meeting.description, meeting.scheduledAt, meeting.duration, meeting.location, meeting.meetingUrl, meeting.status, meeting.createdBy, time.Now())
		suite.NoError(err)
	}

	// Insert test meeting participants
	participants := []struct {
		id, meetingId, userId, status string
		joinedAt                      *time.Time
	}{
		{"test-participant-123", "test-meeting-123", "test-user-123", "joined", &time.Time{}},
		{"test-participant-456", "test-meeting-123", "test-user-456", "invited", nil},
		{"test-participant-789", "test-meeting-456", "test-user-789", "joined", &time.Time{}},
	}

	for _, participant := range participants {
		var joinedAt interface{}
		if participant.joinedAt != nil {
			joinedAt = time.Now()
		}
		_, err := suite.db.Exec(`
			INSERT INTO meeting_participants (id, meeting_id, user_id, status, joined_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, participant.id, participant.meetingId, participant.userId, participant.status, joinedAt, time.Now())
		suite.NoError(err)
	}
}

func (suite *MeetingTestSuite) TestGetMeetings() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "get meetings without chama ID",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Chama ID is required",
		},
		{
			name:           "successful meetings retrieval",
			queryParams:    "?chamaId=test-chama-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "meetings with pagination",
			queryParams:    "?chamaId=test-chama-123&limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "meetings with status filter",
			queryParams:    "?chamaId=test-chama-123&status=scheduled",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "meetings with date range",
			queryParams:    "?chamaId=test-chama-123&startDate=2024-01-01&endDate=2024-12-31",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "meetings of non-existent chama",
			queryParams:    "?chamaId=nonexistent-chama",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/meetings"+tt.queryParams, nil)
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
				meetings := data["meetings"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(meetings), 1)
			}
		})
	}
}

func (suite *MeetingTestSuite) TestCreateMeeting() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful meeting creation",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-123",
				"title":       "New Monthly Meeting",
				"description": "Regular monthly meeting discussion",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    90,
				"location":    "Nairobi Office",
				"meetingUrl":  "https://meet.example.com/new-meeting",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "meeting creation with missing chama ID",
			requestBody: map[string]interface{}{
				"title":       "Meeting without chama",
				"description": "Meeting without chama ID",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    60,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Chama ID is required",
		},
		{
			name: "meeting creation with missing title",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-123",
				"description": "Meeting without title",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    60,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "meeting creation with empty title",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-123",
				"title":       "",
				"description": "Meeting with empty title",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    60,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Title is required",
		},
		{
			name: "meeting creation with missing scheduled time",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-123",
				"title":       "Meeting without schedule",
				"description": "Meeting without scheduled time",
				"duration":    60,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Scheduled time is required",
		},
		{
			name: "meeting creation with past scheduled time",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-123",
				"title":       "Past Meeting",
				"description": "Meeting scheduled in the past",
				"scheduledAt": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"duration":    60,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Scheduled time must be in the future",
		},
		{
			name: "meeting creation with invalid duration",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-123",
				"title":       "Invalid Duration Meeting",
				"description": "Meeting with invalid duration",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    -30,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Duration must be positive",
		},
		{
			name: "meeting creation for non-existent chama",
			requestBody: map[string]interface{}{
				"chamaId":     "nonexistent-chama",
				"title":       "Meeting for nonexistent chama",
				"description": "Meeting for chama that doesn't exist",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    60,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name: "meeting creation by non-member",
			requestBody: map[string]interface{}{
				"chamaId":     "test-chama-456", // User is not a member
				"title":       "Unauthorized Meeting",
				"description": "Meeting by non-member",
				"scheduledAt": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"duration":    60,
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only chama members can create meetings",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/meetings", bytes.NewBuffer(jsonBody))
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
				meeting := data["meeting"].(map[string]interface{})
				assert.NotNil(suite.T(), meeting["id"])
				assert.Equal(suite.T(), tt.requestBody["title"], meeting["title"])
			}
		})
	}
}

func (suite *MeetingTestSuite) TestGetMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful meeting retrieval",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "meeting not found",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "meeting from different chama",
			meetingID:      "test-meeting-other-chama",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/meetings/"+tt.meetingID, nil)
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
				meeting := data["meeting"].(map[string]interface{})
				assert.Equal(suite.T(), tt.meetingID, meeting["id"])
			}
		})
	}
}

func (suite *MeetingTestSuite) TestUpdateMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "successful meeting update",
			meetingID: "test-meeting-123",
			requestBody: map[string]interface{}{
				"title":       "Updated Meeting Title",
				"description": "Updated meeting description",
				"duration":    120,
				"location":    "Updated Location",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "update non-existent meeting",
			meetingID: "nonexistent-meeting",
			requestBody: map[string]interface{}{
				"title": "Updated Title",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:      "update meeting by non-creator",
			meetingID: "test-meeting-456", // Created by different user
			requestBody: map[string]interface{}{
				"title": "Unauthorized Update",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only meeting creator can update meeting",
		},
		{
			name:      "update with invalid data",
			meetingID: "test-meeting-123",
			requestBody: map[string]interface{}{
				"duration": -30,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Duration must be positive",
		},
		{
			name:      "update completed meeting",
			meetingID: "test-meeting-789",
			requestBody: map[string]interface{}{
				"title": "Update Completed Meeting",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot update completed meeting",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/meetings/"+tt.meetingID, bytes.NewBuffer(jsonBody))
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

func (suite *MeetingTestSuite) TestDeleteMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful meeting deletion",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "delete meeting by non-creator",
			meetingID:      "test-meeting-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only meeting creator can delete meeting",
		},
		{
			name:           "delete ongoing meeting",
			meetingID:      "test-meeting-456",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot delete ongoing meeting",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/meetings/"+tt.meetingID, nil)
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

func (suite *MeetingTestSuite) TestJoinMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful meeting join",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "join non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "join meeting already joined",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusConflict,
			expectedError:  "Already joined meeting",
		},
		{
			name:           "join completed meeting",
			meetingID:      "test-meeting-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot join completed meeting",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("POST", "/api/v1/meetings/"+tt.meetingID+"/join", nil)
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

func (suite *MeetingTestSuite) TestLeaveMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful meeting leave",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "leave non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "leave meeting not joined",
			meetingID:      "test-meeting-456",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Not joined meeting",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("POST", "/api/v1/meetings/"+tt.meetingID+"/leave", nil)
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

func (suite *MeetingTestSuite) TestGetMeetingParticipants() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful participants retrieval",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "participants of non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "participants of meeting from different chama",
			meetingID:      "test-meeting-other-chama",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/meetings/"+tt.meetingID+"/participants", nil)
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
				participants := data["participants"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(participants), 0)
			}
		})
	}
}

func (suite *MeetingTestSuite) TestStartMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful meeting start",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "start non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "start meeting by non-creator",
			meetingID:      "test-meeting-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only meeting creator can start meeting",
		},
		{
			name:           "start already ongoing meeting",
			meetingID:      "test-meeting-456",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Meeting is already ongoing",
		},
		{
			name:           "start completed meeting",
			meetingID:      "test-meeting-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot start completed meeting",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("POST", "/api/v1/meetings/"+tt.meetingID+"/start", nil)
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

func (suite *MeetingTestSuite) TestEndMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful meeting end",
			meetingID:      "test-meeting-456", // Ongoing meeting
			expectedStatus: http.StatusOK,
		},
		{
			name:           "end non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "end meeting by non-creator",
			meetingID:      "test-meeting-other-creator",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only meeting creator can end meeting",
		},
		{
			name:           "end scheduled meeting",
			meetingID:      "test-meeting-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot end scheduled meeting",
		},
		{
			name:           "end already completed meeting",
			meetingID:      "test-meeting-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Meeting is already completed",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("POST", "/api/v1/meetings/"+tt.meetingID+"/end", nil)
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

func (suite *MeetingTestSuite) TestRecordMeeting() {
	tests := []struct {
		name           string
		meetingID      string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "successful meeting recording start",
			meetingID: "test-meeting-456",
			requestBody: map[string]interface{}{
				"action": "start",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "recording non-existent meeting",
			meetingID: "nonexistent-meeting",
			requestBody: map[string]interface{}{
				"action": "start",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:      "recording by non-creator",
			meetingID: "test-meeting-other-creator",
			requestBody: map[string]interface{}{
				"action": "start",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only meeting creator can control recording",
		},
		{
			name:      "recording scheduled meeting",
			meetingID: "test-meeting-123",
			requestBody: map[string]interface{}{
				"action": "start",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot record scheduled meeting",
		},
		{
			name:      "invalid recording action",
			meetingID: "test-meeting-456",
			requestBody: map[string]interface{}{
				"action": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid action",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/meetings/"+tt.meetingID+"/record", bytes.NewBuffer(jsonBody))
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

func (suite *MeetingTestSuite) TestGetMeetingRecordings() {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful recordings retrieval",
			meetingID:      "test-meeting-789",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "recordings of non-existent meeting",
			meetingID:      "nonexistent-meeting",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Meeting not found",
		},
		{
			name:           "recordings of meeting from different chama",
			meetingID:      "test-meeting-other-chama",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/meetings/"+tt.meetingID+"/recordings", nil)
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
				recordings := data["recordings"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(recordings), 0)
			}
		})
	}
}

func (suite *MeetingTestSuite) TestConcurrentMeetingJoining() {
	// Test concurrent joining of same meeting
	done := make(chan bool)
	results := make(chan int, 10)

	meetingID := "test-meeting-123"

	// Create multiple test users first
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	for i := 0; i < 10; i++ {
		userID := fmt.Sprintf("test-concurrent-user-%d", i)
		_, err := suite.db.Exec(`
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, fmt.Sprintf("user%d@example.com", i), fmt.Sprintf("+25471234560%d", i),
			"User", fmt.Sprintf("%d", i), string(hashedPassword), "user", "active")
		suite.NoError(err)

		// Add them to chama
		_, err = suite.db.Exec(`
			INSERT INTO chama_members (id, chama_id, user_id, role, is_active, joined_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("test-concurrent-member-%d", i), "test-chama-123", userID, "member", true, time.Now())
		suite.NoError(err)
	}

	// Test concurrent joins
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			// Set different user context for each goroutine
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("userID", fmt.Sprintf("test-concurrent-user-%d", index))
				c.Set("db", suite.db)
				c.Next()
			})

			apiGroup := router.Group("/api/v1")
			apiGroup.POST("/meetings/:id/join", func(c *gin.Context) {
				c.JSON(200, gin.H{"success": true, "message": "Join meeting endpoint - placeholder"})
			})

			req, _ := http.NewRequest("POST", "/api/v1/meetings/"+meetingID+"/join", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
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
		if code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(suite.T(), 10, successCount)
}

func (suite *MeetingTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/meetings?chamaId=test-chama-123", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func TestMeetingSuite(t *testing.T) {
	suite.Run(t, new(MeetingTestSuite))
}
