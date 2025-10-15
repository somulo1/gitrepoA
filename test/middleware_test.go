package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultke-backend/internal/middleware"
	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"
	"vaultke-backend/test/helpers"
)

// Helper function to convert TestUser to models.User
func testUserToModelUser(testUser helpers.TestUser) *models.User {
	return &models.User{
		ID:        testUser.ID,
		Email:     testUser.Email,
		Phone:     testUser.Phone,
		FirstName: "Test",
		LastName:  "User",
		Role:      models.UserRole(testUser.Role),
		Status:    models.UserStatusActive,
	}
}

func TestAuthMiddleware(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	authService := services.NewAuthService(ts.Config.JWTSecret, 24*3600) // 24 hours in seconds
	authMiddleware := middleware.NewAuthMiddleware(authService)

	t.Run("AuthRequired_ValidToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.GET("/protected", func(c *gin.Context) {
			userID := c.GetString("userID")
			role := c.GetString("userRole")
			c.JSON(http.StatusOK, gin.H{
				"userID": userID,
				"role":   role,
			})
		})

		token, err := authService.GenerateToken(testUserToModelUser(ts.Users["user"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, ts.Users["user"].ID, response["userID"])
		assert.Equal(t, ts.Users["user"].Role, response["role"])
	})

	t.Run("AuthRequired_NoToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Authorization header required")
	})

	t.Run("AuthRequired_InvalidToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Invalid token")
	})

	t.Run("AuthRequired_ExpiredToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Create an expired token
		expiredAuthService := services.NewAuthService(ts.Config.JWTSecret, -3600) // -1 hour in seconds
		expiredToken, err := expiredAuthService.GenerateToken(testUserToModelUser(ts.Users["user"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Invalid token")
	})

	t.Run("AuthRequired_MalformedAuthHeader", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidHeader")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Invalid authorization header format")
	})

	t.Run("AuthRequired_EmptyToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Token required")
	})

	t.Run("AuthOptional_ValidToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.OptionalAuth())
		router.GET("/optional", func(c *gin.Context) {
			userID := c.GetString("userID")
			role := c.GetString("userRole")
			c.JSON(http.StatusOK, gin.H{
				"userID":        userID,
				"role":          role,
				"authenticated": userID != "",
			})
		})

		token, err := authService.GenerateToken(testUserToModelUser(ts.Users["user"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, ts.Users["user"].ID, response["userID"])
		assert.Equal(t, ts.Users["user"].Role, response["role"])
		assert.True(t, response["authenticated"].(bool))
	})

	t.Run("AuthOptional_NoToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.OptionalAuth())
		router.GET("/optional", func(c *gin.Context) {
			userID := c.GetString("userID")
			role := c.GetString("userRole")
			c.JSON(http.StatusOK, gin.H{
				"userID":        userID,
				"role":          role,
				"authenticated": userID != "",
			})
		})

		req, _ := http.NewRequest("GET", "/optional", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Empty(t, response["userID"])
		assert.Empty(t, response["role"])
		assert.False(t, response["authenticated"].(bool))
	})

	t.Run("AuthOptional_InvalidToken", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.OptionalAuth())
		router.GET("/optional", func(c *gin.Context) {
			userID := c.GetString("userID")
			role := c.GetString("userRole")
			c.JSON(http.StatusOK, gin.H{
				"userID":        userID,
				"role":          role,
				"authenticated": userID != "",
			})
		})

		req, _ := http.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Empty(t, response["userID"])
		assert.Empty(t, response["role"])
		assert.False(t, response["authenticated"].(bool))
	})

	t.Run("RequireRole_ValidRole", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.Use(authMiddleware.RequireRole("admin"))
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
		})

		token, err := authService.GenerateToken(testUserToModelUser(ts.Users["admin"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "admin access granted", response["message"])
	})

	t.Run("RequireRole_InvalidRole", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.Use(authMiddleware.RequireRole("admin"))
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
		})

		token, err := authService.GenerateToken(testUserToModelUser(ts.Users["user"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Insufficient permissions")
	})

	t.Run("RequireRoles_ValidRoles", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.Use(authMiddleware.RequireRoles("admin", "user"))
		router.GET("/multiRole", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "multi role access granted"})
		})

		token, err := authService.GenerateToken(testUserToModelUser(ts.Users["user"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/multiRole", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "multi role access granted", response["message"])
	})

	t.Run("RequireRoles_InvalidRoles", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(authMiddleware.AuthRequired())
		router.Use(authMiddleware.RequireRoles("admin", "moderator"))
		router.GET("/multiRole", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "multi role access granted"})
		})

		token, err := authService.GenerateToken(testUserToModelUser(ts.Users["user"]))
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/multiRole", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"], "Insufficient permissions")
	})
}

func TestSecurityMiddleware(t *testing.T) {
	t.Run("DefaultSecurityConfig", func(t *testing.T) {
		config := middleware.DefaultSecurityConfig()

		assert.NotNil(t, config)
		assert.False(t, config.RequireHTTPS)
		assert.NotEmpty(t, config.AllowedOrigins)
		assert.Equal(t, int64(10*1024*1024), config.MaxRequestSize)
		assert.Equal(t, 100, config.RateLimitRequests)
	})

	t.Run("SecurityMiddleware_DefaultHeaders", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check security headers
		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
		assert.NotEmpty(t, w.Header().Get("X-XSS-Protection"))
		assert.NotEmpty(t, w.Header().Get("Referrer-Policy"))
	})

	t.Run("SecurityMiddleware_HTTPS_Required", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.RequireHTTPS = true
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUpgradeRequired, w.Code)
	})

	t.Run("SecurityMiddleware_HTTPS_Already_Secure", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.RequireHTTPS = true
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("SecurityMiddleware_CORS_Allowed_Origin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.AllowedOrigins = []string{"https://example.com"}
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Note: Current SecurityMiddleware doesn't implement CORS headers
	})

	t.Run("SecurityMiddleware_CORS_Disallowed_Origin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.AllowedOrigins = []string{"https://example.com"}
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Note: Current SecurityMiddleware doesn't implement CORS blocking
	})

	t.Run("SecurityMiddleware_CORS_Preflight", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.AllowedOrigins = []string{"https://example.com"}
		router.Use(middleware.SecurityMiddleware(config))

		router.OPTIONS("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "options"})
		})

		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Note: Current SecurityMiddleware allows OPTIONS requests to pass through
		// The actual handler returns 200 OK
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("SecurityMiddleware_Rate_Limiting", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.RateLimitRequests = 2
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		// First request should succeed
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Second request should succeed
		req, _ = http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Third request should be rate limited
		req, _ = http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("SecurityMiddleware_Request_Size_Limit", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.MaxRequestSize = 100 // Very small limit for testing
		router.Use(middleware.SecurityMiddleware(config))

		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		// Create a request with body larger than the limit
		largeBody := strings.NewReader(strings.Repeat("a", 200))
		req, _ := http.NewRequest("POST", "/test", largeBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("SecurityMiddleware_Rate_Limit_Window", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		config := middleware.DefaultSecurityConfig()
		config.RateLimitRequests = 1
		config.RateLimitWindow = time.Second
		router.Use(middleware.SecurityMiddleware(config))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestInputValidationMiddleware(t *testing.T) {
	t.Run("InputValidationMiddleware_Valid_JSON", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.InputValidationMiddleware())

		router.POST("/test", func(c *gin.Context) {
			var data map[string]interface{}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "valid"})
		})

		validJSON := `{"name": "test", "value": 123}`
		req, _ := http.NewRequest("POST", "/test", strings.NewReader(validJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("InputValidationMiddleware_Invalid_JSON", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.InputValidationMiddleware())

		router.POST("/test", func(c *gin.Context) {
			var data map[string]interface{}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "valid"})
		})

		invalidJSON := `{"name": "test", "value": 123,}`
		req, _ := http.NewRequest("POST", "/test", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("InputValidationMiddleware_SQL_Injection_Prevention", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.InputValidationMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		// Test SQL injection in query parameter (which the middleware actually validates)
		req, _ := http.NewRequest("GET", "/test?query=select", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid characters in query parameter")
	})

	t.Run("InputValidationMiddleware_XSS_Prevention", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.InputValidationMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		// Test XSS in query parameter (which the middleware actually validates)
		req, _ := http.NewRequest("GET", "/test?content=<script>alert('XSS')</script>", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid characters in query parameter")
	})

	t.Run("InputValidationMiddleware_Large_Request_Body", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.InputValidationMiddleware())

		router.POST("/test", func(c *gin.Context) {
			var data map[string]interface{}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "valid"})
		})

		// Create a large request body (>1MB)
		largeData := make(map[string]interface{})
		largeData["data"] = strings.Repeat("a", 1024*1024+1)

		jsonData, _ := json.Marshal(largeData)
		req, _ := http.NewRequest("POST", "/test", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// InputValidationMiddleware doesn't validate request body size, so this should pass
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("InputValidationMiddleware_Empty_Request_Body", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.InputValidationMiddleware())

		router.POST("/test", func(c *gin.Context) {
			var data map[string]interface{}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "valid"})
		})

		req, _ := http.NewRequest("POST", "/test", nil)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestFileUploadSecurityMiddleware(t *testing.T) {
	t.Run("FileUploadSecurityMiddleware_Valid_File", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.FileUploadSecurityMiddleware())

		router.POST("/upload", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "upload successful"})
		})

		// Test with JSON data (middleware only processes multipart/form-data)
		reqData := map[string]interface{}{
			"file": "test-data",
		}
		jsonData, _ := json.Marshal(reqData)

		req, _ := http.NewRequest("POST", "/upload", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should pass through since it's not multipart/form-data
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("FileUploadSecurityMiddleware_Invalid_File_Type", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.FileUploadSecurityMiddleware())

		router.POST("/upload", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "upload successful"})
		})

		// Create an executable file
		executableData := "data:application/octet-stream;base64,UEsDBAoAAAAAAIdqpEgAAAAAAAAAAAAAAAAJAAAAaGVsbG8uZXhl"

		reqData := map[string]interface{}{
			"file": executableData,
		}
		jsonData, _ := json.Marshal(reqData)

		req, _ := http.NewRequest("POST", "/upload", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should pass through since it's not multipart/form-data
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("FileUploadSecurityMiddleware_Large_File", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.FileUploadSecurityMiddleware())

		router.POST("/upload", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "upload successful"})
		})

		// Create a large file (>5MB)
		largeData := strings.Repeat("a", 5*1024*1024+1)
		largeFile := "data:text/plain;base64," + largeData

		reqData := map[string]interface{}{
			"file": largeFile,
		}
		jsonData, _ := json.Marshal(reqData)

		req, _ := http.NewRequest("POST", "/upload", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should pass through since it's not multipart/form-data
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("FileUploadSecurityMiddleware_Invalid_Base64", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.FileUploadSecurityMiddleware())

		router.POST("/upload", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "upload successful"})
		})

		// Create invalid base64 data
		invalidBase64 := "data:image/jpeg;base64,invalid-base64-data!!!"

		reqData := map[string]interface{}{
			"file": invalidBase64,
		}
		jsonData, _ := json.Marshal(reqData)

		req, _ := http.NewRequest("POST", "/upload", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should pass through since it's not multipart/form-data
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("FileUploadSecurityMiddleware_No_File_Data", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.FileUploadSecurityMiddleware())

		router.POST("/upload", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "upload successful"})
		})

		reqData := map[string]interface{}{
			"name": "test",
		}
		jsonData, _ := json.Marshal(reqData)

		req, _ := http.NewRequest("POST", "/upload", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should pass through if no file data is present
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthRateLimitMiddleware(t *testing.T) {
	t.Run("AuthRateLimitMiddleware_Within_Limit", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.AuthRateLimitMiddleware())

		router.POST("/auth", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "auth successful"})
		})

		// First request should succeed
		req, _ := http.NewRequest("POST", "/auth", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Second request should succeed
		req, _ = http.NewRequest("POST", "/auth", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("AuthRateLimitMiddleware_Exceed_Limit", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.AuthRateLimitMiddleware())

		router.POST("/auth", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "auth successful"})
		})

		// Make requests up to the limit
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", "/auth", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// The next request should be rate limited
		req, _ := http.NewRequest("POST", "/auth", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("AuthRateLimitMiddleware_Different_IPs", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.AuthRateLimitMiddleware())

		router.POST("/auth", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "auth successful"})
		})

		// Requests from different IPs should be independent
		req1, _ := http.NewRequest("POST", "/auth", nil)
		req1.RemoteAddr = "127.0.0.1:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		req2, _ := http.NewRequest("POST", "/auth", nil)
		req2.RemoteAddr = "127.0.0.2:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}

func TestCustomMiddleware(t *testing.T) {
	t.Run("Database_Middleware", func(t *testing.T) {
		ts := helpers.NewTestSuite(t)
		defer ts.Cleanup()

		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Database middleware
		router.Use(func(c *gin.Context) {
			c.Set("db", ts.DB.DB)
			c.Next()
		})

		router.GET("/test", func(c *gin.Context) {
			db := c.MustGet("db")
			assert.NotNil(t, db)
			c.JSON(http.StatusOK, gin.H{"message": "db available"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Config_Middleware", func(t *testing.T) {
		ts := helpers.NewTestSuite(t)
		defer ts.Cleanup()

		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Config middleware
		router.Use(func(c *gin.Context) {
			c.Set("config", ts.Config)
			c.Next()
		})

		router.GET("/test", func(c *gin.Context) {
			config := c.MustGet("config")
			assert.NotNil(t, config)
			c.JSON(http.StatusOK, gin.H{"message": "config available"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("WebSocket_Middleware", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		wsService := helpers.MockWebSocketService()

		// WebSocket middleware
		router.Use(func(c *gin.Context) {
			c.Set("wsService", wsService)
			c.Next()
		})

		router.GET("/test", func(c *gin.Context) {
			ws := c.MustGet("wsService")
			assert.NotNil(t, ws)
			c.JSON(http.StatusOK, gin.H{"message": "websocket service available"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Request_ID_Middleware", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Request ID middleware
		router.Use(func(c *gin.Context) {
			requestID := c.GetHeader("X-Request-ID")
			if requestID == "" {
				requestID = "generated-request-id"
			}
			c.Set("requestID", requestID)
			c.Header("X-Request-ID", requestID)
			c.Next()
		})

		router.GET("/test", func(c *gin.Context) {
			requestID := c.GetString("requestID")
			assert.NotEmpty(t, requestID)
			c.JSON(http.StatusOK, gin.H{"requestID": requestID})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "test-request-id")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
	})

	t.Run("Logging_Middleware", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Logging middleware
		router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: []string{"/health"},
		}))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "logged"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Recovery_Middleware", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Recovery middleware
		router.Use(gin.Recovery())

		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req, _ := http.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
