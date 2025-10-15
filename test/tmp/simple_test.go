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

func TestSimpleAPIHandlers(t *testing.T) {
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

	// Setup basic routes that we know exist
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
		apiGroup.GET("/marketplace/products", api.GetProducts)
		apiGroup.POST("/marketplace/products", api.CreateProduct)
		apiGroup.GET("/notifications", api.GetNotifications)
		apiGroup.GET("/loan-applications", api.GetLoanApplications)
		apiGroup.POST("/loan-applications", api.CreateLoanApplication)
	}

	// Test basic user endpoints
	t.Run("GetUsers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("GetProfile", func(t *testing.T) {
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
	t.Run("GetChamas", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("GetUserChamas", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas/my", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("CreateChama", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":                   "Test Chama",
			"description":            "A test chama",
			"type":                   "savings",
			"county":                 "Nairobi",
			"town":                   "Nairobi",
			"contributionAmount":     5000.0,
			"contributionFrequency":  "monthly",
			"maxMembers":             50,
			"isPublic":               true,
			"requiresApproval":       false,
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
	t.Run("GetWallets", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/wallets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("DepositMoney", func(t *testing.T) {
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

	// Test marketplace endpoints
	t.Run("GetProducts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/marketplace/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("CreateProduct", func(t *testing.T) {
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
	t.Run("GetNotifications", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// Test loan endpoints
	t.Run("GetLoanApplications", func(t *testing.T) {
		// Create a test chama first
		_, err = db.Exec(`
			INSERT INTO chamas (id, name, description, type, county, town, contribution_amount, contribution_frequency, created_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "test-chama-123", "Test Chama", "Test Description", "savings", "Nairobi", "Nairobi", 5000.0, "monthly", "test-user-123", time.Now())
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", "/api/v1/loan-applications?chamaId=test-chama-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("CreateLoanApplication", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"chamaId":    "test-chama-123",
			"amount":     50000.0,
			"purpose":    "Business expansion",
			"duration":   12,
			"loanType":   "business",
			"guarantors": []string{"test-user-456"},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/loan-applications", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// This might return various status codes depending on validation
		assert.Contains(t, []int{http.StatusCreated, http.StatusBadRequest}, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		// Don't assert on success as this depends on the validation logic
	})
}

func TestErrorHandling(t *testing.T) {
	db := setupTestDB()
	defer db.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", db)
		c.Next()
	})

	apiGroup := router.Group("/api/v1")
	{
		apiGroup.POST("/chamas", api.CreateChama)
		apiGroup.POST("/wallets/deposit", api.DepositMoney)
	}

	t.Run("CreateChama with invalid data", func(t *testing.T) {
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

	t.Run("DepositMoney with invalid data", func(t *testing.T) {
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

func TestAuthenticationRequired(t *testing.T) {
	db := setupTestDB()
	defer db.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup routes without authentication middleware
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/profile", api.GetProfile)
	}

	t.Run("GetUsers without authentication", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
	})

	t.Run("GetProfile without authentication", func(t *testing.T) {
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

func TestDatabaseConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware without database connection
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		// Don't set db to simulate connection failure
		c.Next()
	})

	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/chamas", api.GetChamas)
	}

	t.Run("GetUsers without database connection", func(t *testing.T) {
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

	t.Run("GetChamas without database connection", func(t *testing.T) {
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
