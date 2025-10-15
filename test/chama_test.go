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

	"vaultke-backend/internal/api"
)

type ChamaTestSuite struct {
	suite.Suite
	db     *sql.DB
	router *gin.Engine
}

// setupChamaTestDB creates an in-memory SQLite database for testing
func setupChamaTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Create test tables
	createTables := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			phone TEXT UNIQUE NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			avatar TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			status TEXT NOT NULL DEFAULT 'pending',
			is_email_verified BOOLEAN DEFAULT FALSE,
			is_phone_verified BOOLEAN DEFAULT FALSE,
			language TEXT DEFAULT 'en',
			theme TEXT DEFAULT 'dark',
			county TEXT,
			town TEXT,
			latitude REAL,
			longitude REAL,
			business_type TEXT,
			business_description TEXT,
			rating REAL DEFAULT 0,
			total_ratings INTEGER DEFAULT 0,
			bio TEXT,
			occupation TEXT,
			date_of_birth DATE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE chamas (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			county TEXT NOT NULL,
			town TEXT NOT NULL,
			contribution_amount REAL NOT NULL,
			contribution_frequency TEXT NOT NULL,
			max_members INTEGER NOT NULL,
			current_members INTEGER DEFAULT 1,
			is_public BOOLEAN DEFAULT TRUE,
			requires_approval BOOLEAN DEFAULT FALSE,
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES users(id)
		)`,
		`CREATE TABLE chama_members (
			id TEXT PRIMARY KEY,
			chama_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			total_contributions REAL DEFAULT 0,
			last_contribution DATETIME,
			rating REAL DEFAULT 0,
			total_ratings INTEGER DEFAULT 0,
			FOREIGN KEY (chama_id) REFERENCES chamas(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(chama_id, user_id)
		)`,
	}

	for _, table := range createTables {
		_, err := db.Exec(table)
		if err != nil {
			panic(err)
		}
	}

	return db
}

func (suite *ChamaTestSuite) SetupSuite() {
	suite.db = setupChamaTestDB()

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup chama routes
	apiGroup := suite.router.Group("/api/v1")
	{
		apiGroup.GET("/chamas", api.GetChamas)
		apiGroup.GET("/chamas/my", api.GetUserChamas)
		apiGroup.POST("/chamas", api.CreateChama)
		apiGroup.GET("/chamas/:id", api.GetChama)
		apiGroup.PUT("/chamas/:id", api.UpdateChama)
		apiGroup.DELETE("/chamas/:id", api.DeleteChama)
		apiGroup.GET("/chamas/:id/members", api.GetChamaMembers)
		apiGroup.POST("/chamas/:id/join", api.JoinChama)
		apiGroup.POST("/chamas/:id/leave", api.LeaveChama)
		apiGroup.GET("/chamas/:id/transactions", api.GetChamaTransactions)
		apiGroup.POST("/chamas/:id/invite", api.InviteToChama)
		apiGroup.PUT("/chamas/:id/members/:memberId", api.UpdateChamaMember)
		apiGroup.DELETE("/chamas/:id/members/:memberId", api.RemoveChamaMember)
		apiGroup.GET("/chamas/:id/statistics", api.GetChamaStatistics)
	}
}

func (suite *ChamaTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
	// Insert test data
	suite.insertTestData()
}

func (suite *ChamaTestSuite) TearDownSuite() {
	suite.db.Close()
}

func (suite *ChamaTestSuite) cleanupTestData() {
	tables := []string{"chama_members", "chamas", "users", "wallets", "transactions"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id LIKE 'test-%%'", table))
		suite.NoError(err)
	}
}

func (suite *ChamaTestSuite) insertTestData() {
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
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, county, town, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, user.email, user.phone, user.firstName, user.lastName, string(hashedPassword), "user", "active", "Nairobi", "Nairobi", time.Now())
		suite.NoError(err)
	}

	// Insert test chamas
	chamas := []struct {
		id, name, description, chamaType, county, town, createdBy string
		contributionAmount                                        float64
		contributionFrequency                                     string
		maxMembers                                                int
		isPublic, requiresApproval                                bool
	}{
		{"test-chama-123", "Test Chama 1", "A test chama for testing", "savings", "Nairobi", "Nairobi", "test-user-123", 5000.0, "monthly", 20, true, false},
		{"test-chama-456", "Test Chama 2", "Another test chama", "investment", "Mombasa", "Mombasa", "test-user-456", 10000.0, "weekly", 50, true, true},
		{"test-chama-789", "Private Chama", "A private test chama", "credit", "Kisumu", "Kisumu", "test-user-789", 2000.0, "monthly", 10, false, true},
	}

	for _, chama := range chamas {
		_, err := suite.db.Exec(`
			INSERT INTO chamas (id, name, description, type, county, town, contribution_amount, contribution_frequency, 
							   max_members, is_public, requires_approval, created_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, chama.id, chama.name, chama.description, chama.chamaType, chama.county, chama.town, chama.contributionAmount,
			chama.contributionFrequency, chama.maxMembers, chama.isPublic, chama.requiresApproval, chama.createdBy, time.Now())
		suite.NoError(err)
	}

	// Insert test chama members
	members := []struct {
		id, chamaId, userId, role string
		totalContributions        float64
		isActive                  bool
	}{
		{"test-member-123", "test-chama-123", "test-user-123", "chairperson", 25000.0, true},
		{"test-member-456", "test-chama-123", "test-user-456", "member", 15000.0, true},
		{"test-member-789", "test-chama-456", "test-user-456", "chairperson", 40000.0, true},
		{"test-member-012", "test-chama-456", "test-user-789", "treasurer", 30000.0, true},
	}

	for _, member := range members {
		_, err := suite.db.Exec(`
			INSERT INTO chama_members (id, chama_id, user_id, role, total_contributions, is_active, joined_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, member.id, member.chamaId, member.userId, member.role, member.totalContributions, member.isActive, time.Now())
		suite.NoError(err)
	}

	// Insert test wallets
	wallets := []struct {
		id, ownerId, walletType string
		balance                 float64
	}{
		{"test-wallet-chama-123", "test-chama-123", "chama", 100000.0},
		{"test-wallet-chama-456", "test-chama-456", "chama", 200000.0},
		{"test-wallet-personal-123", "test-user-123", "personal", 50000.0},
	}

	for _, wallet := range wallets {
		_, err := suite.db.Exec(`
			INSERT INTO wallets (id, owner_id, type, balance, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, wallet.id, wallet.ownerId, wallet.walletType, wallet.balance, time.Now())
		suite.NoError(err)
	}
}

func (suite *ChamaTestSuite) TestGetChamas() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful chamas retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chamas with pagination",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chamas with type filter",
			queryParams:    "?type=savings",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chamas with location filter",
			queryParams:    "?county=Nairobi&town=Nairobi",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chamas with search query",
			queryParams:    "?q=Test%20Chama",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chamas with contribution amount filter",
			queryParams:    "?minContribution=1000&maxContribution=10000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "public chamas only",
			queryParams:    "?isPublic=true",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/chamas"+tt.queryParams, nil)
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
				chamas := data["chamas"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(chamas), 1)
			}
		})
	}
}

func (suite *ChamaTestSuite) TestGetUserChamas() {
	req, _ := http.NewRequest("GET", "/api/v1/chamas/my", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	chamas := data["chamas"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(chamas), 1)
}

func (suite *ChamaTestSuite) TestCreateChama() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful chama creation",
			requestBody: map[string]interface{}{
				"name":                  "New Test Chama",
				"description":           "A new test chama for testing",
				"type":                  "savings",
				"county":                "Nakuru",
				"town":                  "Nakuru",
				"contributionAmount":    7500.0,
				"contributionFrequency": "monthly",
				"maxMembers":            30,
				"isPublic":              true,
				"requiresApproval":      false,
				"meetingFrequency":      "monthly",
				"meetingDayOfMonth":     15,
				"meetingTime":           "14:00",
				"rules":                 []string{"No late payments", "Attend meetings"},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "chama creation with missing name",
			requestBody: map[string]interface{}{
				"description":           "A test chama without name",
				"type":                  "savings",
				"county":                "Nairobi",
				"town":                  "Nairobi",
				"contributionAmount":    5000.0,
				"contributionFrequency": "monthly",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
		},
		{
			name: "chama creation with invalid type",
			requestBody: map[string]interface{}{
				"name":                  "Invalid Type Chama",
				"description":           "A test chama with invalid type",
				"type":                  "invalid-type",
				"county":                "Nairobi",
				"town":                  "Nairobi",
				"contributionAmount":    5000.0,
				"contributionFrequency": "monthly",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid chama type",
		},
		{
			name: "chama creation with negative contribution amount",
			requestBody: map[string]interface{}{
				"name":                  "Negative Contribution Chama",
				"description":           "A test chama with negative contribution",
				"type":                  "savings",
				"county":                "Nairobi",
				"town":                  "Nairobi",
				"contributionAmount":    -1000.0,
				"contributionFrequency": "monthly",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Contribution amount must be positive",
		},
		{
			name: "chama creation with invalid contribution frequency",
			requestBody: map[string]interface{}{
				"name":                  "Invalid Frequency Chama",
				"description":           "A test chama with invalid frequency",
				"type":                  "savings",
				"county":                "Nairobi",
				"town":                  "Nairobi",
				"contributionAmount":    5000.0,
				"contributionFrequency": "invalid-frequency",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid contribution frequency",
		},
		{
			name: "chama creation with duplicate name",
			requestBody: map[string]interface{}{
				"name":                  "Test Chama 1", // Already exists
				"description":           "A duplicate test chama",
				"type":                  "savings",
				"county":                "Nairobi",
				"town":                  "Nairobi",
				"contributionAmount":    5000.0,
				"contributionFrequency": "monthly",
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Chama name already exists",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/chamas", bytes.NewBuffer(jsonBody))
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
				chama := data["chama"].(map[string]interface{})
				assert.NotNil(suite.T(), chama["id"])
				assert.Equal(suite.T(), tt.requestBody["name"], chama["name"])
			}
		})
	}
}

func (suite *ChamaTestSuite) TestGetChama() {
	tests := []struct {
		name           string
		chamaID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful chama retrieval",
			chamaID:        "test-chama-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chama not found",
			chamaID:        "nonexistent-chama",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:           "private chama access by non-member",
			chamaID:        "test-chama-789",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/chamas/"+tt.chamaID, nil)
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
				chama := data["chama"].(map[string]interface{})
				assert.Equal(suite.T(), tt.chamaID, chama["id"])
			}
		})
	}
}

func (suite *ChamaTestSuite) TestUpdateChama() {
	tests := []struct {
		name           string
		chamaID        string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "successful chama update",
			chamaID: "test-chama-123",
			requestBody: map[string]interface{}{
				"name":        "Updated Test Chama",
				"description": "Updated description",
				"maxMembers":  25,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "update non-existent chama",
			chamaID: "nonexistent-chama",
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:    "update chama without permission",
			chamaID: "test-chama-456", // User is not chairperson
			requestBody: map[string]interface{}{
				"name": "Unauthorized Update",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only chairperson can update chama",
		},
		{
			name:    "update with invalid data",
			chamaID: "test-chama-123",
			requestBody: map[string]interface{}{
				"contributionAmount": -500.0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Contribution amount must be positive",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/chamas/"+tt.chamaID, bytes.NewBuffer(jsonBody))
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

func (suite *ChamaTestSuite) TestDeleteChama() {
	tests := []struct {
		name           string
		chamaID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful chama deletion",
			chamaID:        "test-chama-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete non-existent chama",
			chamaID:        "nonexistent-chama",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:           "delete chama without permission",
			chamaID:        "test-chama-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only chairperson can delete chama",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/chamas/"+tt.chamaID, nil)
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

func (suite *ChamaTestSuite) TestGetChamaMembers() {
	tests := []struct {
		name           string
		chamaID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful members retrieval",
			chamaID:        "test-chama-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "members of non-existent chama",
			chamaID:        "nonexistent-chama",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:           "private chama members access by non-member",
			chamaID:        "test-chama-789",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/chamas/"+tt.chamaID+"/members", nil)
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
				members := data["members"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(members), 1)
			}
		})
	}
}

func (suite *ChamaTestSuite) TestJoinChama() {
	tests := []struct {
		name           string
		chamaID        string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "successful chama joining",
			chamaID: "test-chama-456",
			requestBody: map[string]interface{}{
				"message": "I'd like to join this chama",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "join non-existent chama",
			chamaID: "nonexistent-chama",
			requestBody: map[string]interface{}{
				"message": "Join request",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:    "join chama already member of",
			chamaID: "test-chama-123",
			requestBody: map[string]interface{}{
				"message": "Join request",
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Already a member",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/chamas/"+tt.chamaID+"/join", bytes.NewBuffer(jsonBody))
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

func (suite *ChamaTestSuite) TestLeaveChama() {
	tests := []struct {
		name           string
		chamaID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful chama leaving",
			chamaID:        "test-chama-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "leave non-existent chama",
			chamaID:        "nonexistent-chama",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:           "leave chama not member of",
			chamaID:        "test-chama-789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Not a member",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("POST", "/api/v1/chamas/"+tt.chamaID+"/leave", nil)
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

func (suite *ChamaTestSuite) TestGetChamaTransactions() {
	tests := []struct {
		name           string
		chamaID        string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful transactions retrieval",
			chamaID:        "test-chama-123",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions with pagination",
			chamaID:        "test-chama-123",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions of non-existent chama",
			chamaID:        "nonexistent-chama",
			queryParams:    "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Chama not found",
		},
		{
			name:           "private chama transactions access by non-member",
			chamaID:        "test-chama-789",
			queryParams:    "",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/chamas/"+tt.chamaID+"/transactions"+tt.queryParams, nil)
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

func (suite *ChamaTestSuite) TestGetChamaStatistics() {
	req, _ := http.NewRequest("GET", "/api/v1/chamas/test-chama-123/statistics", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(suite.T(), data, "totalMembers")
	assert.Contains(suite.T(), data, "totalFunds")
	assert.Contains(suite.T(), data, "totalContributions")
}

func (suite *ChamaTestSuite) TestConcurrentChamaJoining() {
	// Test concurrent joining of same chama
	done := make(chan bool)
	results := make(chan int, 10)

	chamaID := "test-chama-456"

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
			apiGroup.POST("/chamas/:id/join", func(c *gin.Context) {
				c.JSON(200, gin.H{"success": true, "message": "Join request submitted"})
			})

			requestBody := map[string]interface{}{
				"message": fmt.Sprintf("Join request from user %d", index),
			}

			jsonBody, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/chamas/"+chamaID+"/join", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
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

	// Check results - all should succeed if chama has enough space
	var successCount int
	for code := range results {
		if code == http.StatusCreated {
			successCount++
		}
	}

	assert.GreaterOrEqual(suite.T(), successCount, 1)
}

func (suite *ChamaTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/chamas", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func TestChamaSuite(t *testing.T) {
	suite.Run(t, new(ChamaTestSuite))
}
