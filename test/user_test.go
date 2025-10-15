package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"vaultke-backend/internal/api"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

// setupUserTestDB creates an in-memory SQLite database for testing
func setupUserTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return db
}

type UserTestSuite struct {
	suite.Suite
	db     *sql.DB
	router *gin.Engine
}

func (suite *UserTestSuite) SetupSuite() {
	suite.db = setupUserTestDB()

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup user routes - using actual handlers that exist
	apiGroup := suite.router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/profile", api.GetProfile)
		apiGroup.PUT("/profile", api.UpdateProfile)
		apiGroup.POST("/avatar", api.UploadAvatar)
		apiGroup.PUT("/users/:id/role", api.AdminUpdateUserRole)
		apiGroup.PUT("/users/:id/status", api.UpdateUserStatus)
		apiGroup.DELETE("/users/:id", api.DeleteUser)
		apiGroup.GET("/users/admin/all", api.GetAllUsersForAdmin)
	}
}

func (suite *UserTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
	// Insert test users
	suite.insertTestUsers()
}

func (suite *UserTestSuite) TearDownSuite() {
	suite.db.Close()
}

func (suite *UserTestSuite) cleanupTestData() {
	_, err := suite.db.Exec("DELETE FROM users WHERE id LIKE 'test-%'")
	suite.NoError(err)
}

func (suite *UserTestSuite) insertTestUsers() {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	users := []struct {
		id, email, phone, firstName, lastName, role, status string
	}{
		{"test-user-123", "test@example.com", "+254712345678", "Test", "User", "user", "active"},
		{"test-user-456", "jane@example.com", "+254712345679", "Jane", "Smith", "user", "active"},
		{"test-user-789", "bob@example.com", "+254712345680", "Bob", "Johnson", "admin", "active"},
		{"test-user-suspended", "suspended@example.com", "+254712345681", "Suspended", "User", "user", "suspended"},
	}

	for _, user := range users {
		_, err := suite.db.Exec(`
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, county, town, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, user.email, user.phone, user.firstName, user.lastName, string(hashedPassword), user.role, user.status, "Nairobi", "Nairobi", time.Now())
		suite.NoError(err)
	}
}

func (suite *UserTestSuite) TestGetUsers() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful users retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "users with pagination",
			queryParams:    "?limit=2&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "users with search query",
			queryParams:    "?q=Jane",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "users with invalid limit",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusOK, // Should default to 50
		},
		{
			name:           "users with large limit",
			queryParams:    "?limit=1000",
			expectedStatus: http.StatusOK, // Should be capped at 100
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/users"+tt.queryParams, nil)
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
				users := data["users"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(users), 1)

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

func (suite *UserTestSuite) TestGetProfile() {
	req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	assert.Equal(suite.T(), "test-user-123", user["id"])
	assert.Equal(suite.T(), "test@example.com", user["email"])
}

func (suite *UserTestSuite) TestUpdateProfile() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful profile update",
			requestBody: map[string]interface{}{
				"firstName": "Updated",
				"lastName":  "Name",
				"bio":       "Updated bio",
				"county":    "Mombasa",
				"town":      "Mombasa",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update with invalid email",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid email format",
		},
		{
			name: "update with duplicate email",
			requestBody: map[string]interface{}{
				"email": "jane@example.com", // Already exists
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Email already exists",
		},
		{
			name: "update with invalid phone",
			requestBody: map[string]interface{}{
				"phone": "invalid-phone",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid phone format",
		},
		{
			name: "update with empty first name",
			requestBody: map[string]interface{}{
				"firstName": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "First name is required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/profile", bytes.NewBuffer(jsonBody))
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

func (suite *UserTestSuite) TestGetUser() {
	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful user retrieval",
			userID:         "test-user-456",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "nonexistent-user",
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name:           "get suspended user",
			userID:         "test-user-suspended",
			expectedStatus: http.StatusOK, // Should return user but with status info
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/users/"+tt.userID, nil)
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
				user := data["user"].(map[string]interface{})
				assert.Equal(suite.T(), tt.userID, user["id"])
			}
		})
	}
}

func (suite *UserTestSuite) TestUpdateUser() {
	tests := []struct {
		name           string
		userID         string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful user update by admin",
			userID: "test-user-456",
			requestBody: map[string]interface{}{
				"firstName": "Updated",
				"lastName":  "User",
				"status":    "suspended",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "update non-existent user",
			userID: "nonexistent-user",
			requestBody: map[string]interface{}{
				"firstName": "Updated",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name:   "update with invalid data",
			userID: "test-user-456",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid email format",
		},
	}

	// Set current user as admin for these tests
	suite.db.Exec("UPDATE users SET role = 'admin' WHERE id = 'test-user-123'")

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/users/"+tt.userID, bytes.NewBuffer(jsonBody))
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

func (suite *UserTestSuite) TestDeleteUser() {
	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful user deletion",
			userID:         "test-user-456",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete non-existent user",
			userID:         "nonexistent-user",
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name:           "delete self",
			userID:         "test-user-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot delete your own account",
		},
	}

	// Set current user as admin for these tests
	suite.db.Exec("UPDATE users SET role = 'admin' WHERE id = 'test-user-123'")

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/users/"+tt.userID, nil)
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

func (suite *UserTestSuite) TestSearchUsers() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "search by name",
			queryParams:    "?q=Jane",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "search by email",
			queryParams:    "?q=test@example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "search with no results",
			queryParams:    "?q=nonexistent",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "search with empty query",
			queryParams:    "?q=",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Search query is required",
		},
		{
			name:           "search with location filter",
			queryParams:    "?q=Test&county=Nairobi",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/users/search"+tt.queryParams, nil)
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
			}
		})
	}
}

func (suite *UserTestSuite) TestUploadAvatar() {
	tests := []struct {
		name           string
		setupFile      bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful avatar upload",
			setupFile:      true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "upload without file",
			setupFile:      false,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "No file uploaded",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var req *http.Request
			if tt.setupFile {
				// Create a multipart form with a fake image file
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)

				// Create a fake image file part
				part, err := writer.CreateFormFile("avatar", "test.jpg")
				suite.NoError(err)

				// Write fake image data
				_, err = part.Write([]byte("fake-image-data"))
				suite.NoError(err)

				writer.Close()

				req, _ = http.NewRequest("POST", "/api/v1/upload-avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			} else {
				req, _ = http.NewRequest("POST", "/api/v1/upload-avatar", nil)
			}

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

func (suite *UserTestSuite) TestUnauthorizedAccess() {
	// Test without authentication middleware
	router := gin.New()
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/profile", api.GetProfile)
	}

	tests := []struct {
		name     string
		endpoint string
		method   string
	}{
		{"get users without auth", "/api/v1/users", "GET"},
		{"get profile without auth", "/api/v1/profile", "GET"},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest(tt.method, tt.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)
			assert.False(suite.T(), response["success"].(bool))
		})
	}
}

func (suite *UserTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func (suite *UserTestSuite) TestConcurrentUserUpdates() {
	// Test concurrent updates to same user
	done := make(chan bool)
	results := make(chan int, 5)

	userID := "test-user-456"

	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()

			requestBody := map[string]interface{}{
				"firstName": fmt.Sprintf("Updated%d", index),
				"lastName":  "Concurrent",
			}

			jsonBody, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/users/"+userID, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)
			results <- w.Code
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
	close(results)

	// Check that all requests were handled properly
	var successCount int
	for code := range results {
		if code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(suite.T(), 5, successCount)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
