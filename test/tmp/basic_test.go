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
	"golang.org/x/crypto/bcrypt"

	"vaultke-backend/internal/api"
)

// TestBasicAPIEndpoints tests basic API functionality
func TestBasicAPIEndpoints(t *testing.T) {
	// Setup test database
	db := setupTestDB()
	defer db.Close()

	// Insert test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	_, err := db.Exec(`
		INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-user-123", "test@example.com", "+254712345678", "Test", "User", string(hashedPassword), "user", "active", time.Now())
	assert.NoError(t, err)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", db)
		c.Next()
	})

	// Setup routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/profile", api.GetProfile)
		apiGroup.PUT("/profile", api.UpdateProfile)
		apiGroup.GET("/chamas", api.GetChamas)
		apiGroup.GET("/chamas/my", api.GetUserChamas)
		apiGroup.POST("/chamas", api.CreateChama)
		apiGroup.GET("/wallets", api.GetWallets)
		apiGroup.POST("/wallets/deposit", api.DepositMoney)
		apiGroup.GET("/meetings", api.GetMeetings)
		apiGroup.POST("/meetings", api.CreateMeeting)
		apiGroup.GET("/marketplace/products", api.GetProducts)
		apiGroup.POST("/marketplace/products", api.CreateProduct)
		apiGroup.GET("/notifications", api.GetNotifications)
		apiGroup.POST("/notifications", api.CreateNotification)
		apiGroup.GET("/reminders", api.GetReminders)
		apiGroup.POST("/reminders", api.CreateReminder)
		apiGroup.GET("/loans", api.GetLoans)
		apiGroup.POST("/loans", api.CreateLoan)
	}

	// Test user endpoints
	t.Run("GET /api/v1/users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("GET /api/v1/profile", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test chama endpoints
	t.Run("GET /api/v1/chamas", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("GET /api/v1/chamas/my", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas/my", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/chamas", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":                  "Test Chama",
			"description":           "A test chama",
			"type":                  "savings",
			"county":                "Nairobi",
			"town":                  "Nairobi",
			"contributionAmount":    5000.0,
			"contributionFrequency": "monthly",
			"maxMembers":            50,
			"isPublic":              true,
			"requiresApproval":      false,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/chamas", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test wallet endpoints
	t.Run("GET /api/v1/wallets", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/wallets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/wallets/deposit", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"amount":        10000.0,
			"paymentMethod": "mpesa",
			"reference":     "MP12345678",
			"description":   "Test deposit",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/wallets/deposit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test meeting endpoints
	t.Run("GET /api/v1/meetings", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/meetings?chamaId=test-chama", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// This should return 400 because chama doesn't exist
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test marketplace endpoints
	t.Run("GET /api/v1/marketplace/products", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/marketplace/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/marketplace/products", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "Test Product",
			"description": "A test product",
			"category":    "electronics",
			"price":       15000.0,
			"stock":       10,
			"county":      "Nairobi",
			"town":        "Nairobi",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/marketplace/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test notification endpoints
	t.Run("GET /api/v1/notifications", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/notifications", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"title":       "Test Notification",
			"message":     "This is a test notification",
			"type":        "system",
			"recipientId": "test-user-123",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test reminder endpoints
	t.Run("GET /api/v1/reminders", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/reminders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/reminders", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"title":        "Test Reminder",
			"description":  "This is a test reminder",
			"reminderType": "once",
			"scheduledAt":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/reminders", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test loan endpoints
	t.Run("GET /api/v1/loans", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/loans?chamaId=test-chama", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// This should return 400 because chama doesn't exist
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestBasicErrorHandling tests error handling scenarios
func TestBasicErrorHandling(t *testing.T) {
	// Setup test database
	db := setupTestDB()
	defer db.Close()

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", db)
		c.Next()
	})

	// Setup routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.POST("/chamas", api.CreateChama)
		apiGroup.POST("/marketplace/products", api.CreateProduct)
		apiGroup.POST("/wallets/deposit", api.DepositMoney)
	}

	// Test validation errors
	t.Run("POST /api/v1/chamas with invalid data", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name": "Te", // Too short
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/chamas", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/marketplace/products with invalid data", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name": "", // Empty name
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/marketplace/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
	})

	t.Run("POST /api/v1/wallets/deposit with invalid data", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"amount": -100.0, // Negative amount
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/wallets/deposit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
	})
}

// TestUnauthorizedAccess tests unauthorized access scenarios
func TestUnauthorizedAccess(t *testing.T) {
	// Setup test database
	db := setupTestDB()
	defer db.Close()

	// Setup router without authentication middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/profile", api.GetProfile)
	}

	// Test unauthorized access
	t.Run("GET /api/v1/users without auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
	})

	t.Run("GET /api/v1/profile without auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
	})
}

// TestDatabaseConnectionFailure tests database connection failures
func TestDatabaseConnectionFailure(t *testing.T) {
	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware without database connection
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		// Don't set db to simulate connection failure
		c.Next()
	})

	// Setup routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/chamas", api.GetChamas)
	}

	// Test database connection failures
	t.Run("GET /api/v1/users without db connection", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(string), "Database connection not available")
	})

	t.Run("GET /api/v1/chamas without db connection", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(string), "Database connection not available")
	})
}
