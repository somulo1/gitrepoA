package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vaultke-backend/internal/api"
	"vaultke-backend/test/helpers"
)

// BenchmarkBasicUserRegistration benchmarks user registration performance
func BenchmarkBasicUserRegistration(b *testing.B) {
	// Create a minimal test environment for benchmarks
	testDB := helpers.SetupTestDatabase()
	defer testDB.Close()

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(func(c *gin.Context) {
		c.Set("db", testDB.DB)
		c.Next()
	})

	// Setup auth routes
	authHandlers := api.NewAuthHandlers(testDB.DB, "test-secret", 3600)
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandlers.Register)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			registerData := map[string]interface{}{
				"email":     "benchmark" + string(rune(i)) + "@example.com",
				"phone":     "+25470000" + string(rune(1000+i)),
				"firstName": "Benchmark",
				"lastName":  "User",
				"password":  "password123",
			}
			jsonData, _ := json.Marshal(registerData)

			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// We don't check the result in benchmarks, just measure performance
			i++
		}
	})
}

// BenchmarkBasicUserLogin benchmarks user login performance
func BenchmarkBasicUserLogin(b *testing.B) {
	// Create a minimal test environment for benchmarks
	testDB := helpers.SetupTestDatabase()
	defer testDB.Close()

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(func(c *gin.Context) {
		c.Set("db", testDB.DB)
		c.Next()
	})

	// Setup auth routes
	authHandlers := api.NewAuthHandlers(testDB.DB, "test-secret", 3600)
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandlers.Register)
		auth.POST("/login", authHandlers.Login)
	}

	// Create a test user first
	registerData := map[string]interface{}{
		"email":     "benchmark@example.com",
		"phone":     "+254700000500",
		"firstName": "Benchmark",
		"lastName":  "User",
		"password":  "password123",
	}
	jsonData, _ := json.Marshal(registerData)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			loginData := map[string]interface{}{
				"email":    "benchmark@example.com",
				"password": "password123",
			}
			jsonData, _ := json.Marshal(loginData)

			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// We don't check the result in benchmarks, just measure performance
		}
	})
}

// BenchmarkGetUsers benchmarks getting users performance
func BenchmarkGetUsers(b *testing.B) {
	// Create a minimal test environment for benchmarks
	testDB := helpers.SetupTestDatabase()
	defer testDB.Close()

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", testDB.DB)
		c.Next()
	})

	// Setup user routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/users", api.GetUsers)
	}

	// Create some test users
	for i := 0; i < 10; i++ {
		err := testDB.CreateTestUser(helpers.TestUser{
			ID:       "test-user-" + string(rune(i)),
			Email:    "test" + string(rune(i)) + "@example.com",
			Phone:    "+25471234567" + string(rune(i)),
			Role:     "user",
			Password: "password123",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/v1/users", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// We don't check the result in benchmarks, just measure performance
		}
	})
}

// BenchmarkGetWallets benchmarks getting wallets performance
func BenchmarkGetWallets(b *testing.B) {
	// Create a minimal test environment for benchmarks
	testDB := helpers.SetupTestDatabase()
	defer testDB.Close()

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", testDB.DB)
		c.Next()
	})

	// Setup wallet routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/wallets", api.GetWallets)
	}

	// Create test user and wallet
	err := testDB.CreateTestUser(helpers.TestUser{
		ID:       "test-user-123",
		Email:    "test@example.com",
		Phone:    "+254712345678",
		Role:     "user",
		Password: "password123",
	})
	if err != nil {
		b.Fatal(err)
	}

	err = testDB.CreateTestWallet("test-wallet-123", "test-user-123", "personal", 10000.0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/v1/wallets", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// We don't check the result in benchmarks, just measure performance
		}
	})
}

// BenchmarkDepositMoney benchmarks deposit money performance
func BenchmarkDepositMoney(b *testing.B) {
	// Create a minimal test environment for benchmarks
	testDB := helpers.SetupTestDatabase()
	defer testDB.Close()

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", testDB.DB)
		c.Next()
	})

	// Setup wallet routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.POST("/wallets/deposit", api.DepositMoney)
	}

	// Create test user and wallet
	err := testDB.CreateTestUser(helpers.TestUser{
		ID:       "test-user-123",
		Email:    "test@example.com",
		Phone:    "+254712345678",
		Role:     "user",
		Password: "password123",
	})
	if err != nil {
		b.Fatal(err)
	}

	err = testDB.CreateTestWallet("test-wallet-123", "test-user-123", "personal", 10000.0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			depositData := map[string]interface{}{
				"amount":        100.0,
				"paymentMethod": "simulation",
				"description":   "Benchmark deposit",
			}
			jsonData, _ := json.Marshal(depositData)

			req, _ := http.NewRequest("POST", "/api/v1/wallets/deposit", bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// We don't check the result in benchmarks, just measure performance
		}
	})
}
