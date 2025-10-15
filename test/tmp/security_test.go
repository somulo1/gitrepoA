package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"vaultke-backend/internal/middleware"
	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

func TestSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply security middleware
	router.Use(middleware.SecurityMiddleware(middleware.DefaultSecurityConfig()))
	router.Use(middleware.InputValidationMiddleware())

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		body           string
		expectedStatus int
		shouldFail     bool
	}{
		{
			name:           "Valid request",
			method:         "POST",
			path:           "/test",
			headers:        map[string]string{"Content-Type": "application/json", "User-Agent": "TestAgent/1.0"},
			body:           `{"name": "test"}`,
			expectedStatus: http.StatusOK,
			shouldFail:     false,
		},
		{
			name:           "Request too large",
			method:         "POST",
			path:           "/test",
			headers:        map[string]string{"Content-Type": "application/json", "User-Agent": "TestAgent/1.0"},
			body:           string(make([]byte, 11*1024*1024)), // 11MB
			expectedStatus: http.StatusRequestEntityTooLarge,
			shouldFail:     true,
		},
		{
			name:           "Invalid User-Agent",
			method:         "POST",
			path:           "/test",
			headers:        map[string]string{"Content-Type": "application/json", "User-Agent": ""},
			body:           `{"name": "test"}`,
			expectedStatus: http.StatusBadRequest,
			shouldFail:     true,
		},
		{
			name:           "Suspicious path",
			method:         "GET",
			path:           "/test?q=%3Cscript%3Ealert%28%27xss%27%29%3C%2Fscript%3E", // URL encoded: <script>alert('xss')</script>
			headers:        map[string]string{"User-Agent": "TestAgent/1.0"},
			expectedStatus: http.StatusBadRequest,
			shouldFail:     true,
		},
		{
			name:           "SQL injection in query",
			method:         "GET",
			path:           "/test?id=1%27%20OR%20%271%27%3D%271", // URL encoded: 1' OR '1'='1
			headers:        map[string]string{"User-Agent": "TestAgent/1.0"},
			expectedStatus: http.StatusBadRequest,
			shouldFail:     true,
		},
		{
			name:           "Invalid content type",
			method:         "POST",
			path:           "/test",
			headers:        map[string]string{"Content-Type": "text/plain", "User-Agent": "TestAgent/1.0"},
			body:           `{"name": "test"}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			shouldFail:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		shouldFail  bool
		description string
	}{
		{
			name: "Valid user registration",
			input: models.UserRegistration{
				Email:     "test@example.com",
				Phone:     "+254712345678",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "SecurePass123!",
				Language:  "en",
			},
			shouldFail:  false,
			description: "Should pass with valid data",
		},
		{
			name: "XSS attempt in name",
			input: models.UserRegistration{
				Email:     "test@example.com",
				Phone:     "+254712345678",
				FirstName: "<script>alert('xss')</script>",
				LastName:  "Doe",
				Password:  "SecurePass123!",
				Language:  "en",
			},
			shouldFail:  true,
			description: "Should fail with XSS attempt",
		},
		{
			name: "SQL injection attempt",
			input: models.UserRegistration{
				Email:     "test'; DROP TABLE users; --",
				Phone:     "+254712345678",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "SecurePass123!",
				Language:  "en",
			},
			shouldFail:  true,
			description: "Should fail with SQL injection attempt",
		},
		{
			name: "Weak password",
			input: models.UserRegistration{
				Email:     "test@example.com",
				Phone:     "+254712345678",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "weak",
				Language:  "en",
			},
			shouldFail:  true,
			description: "Should fail with weak password",
		},
		{
			name: "Invalid email",
			input: models.UserRegistration{
				Email:     "invalid-email",
				Phone:     "+254712345678",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "SecurePass123!",
				Language:  "en",
			},
			shouldFail:  true,
			description: "Should fail with invalid email",
		},
		{
			name: "Invalid phone number",
			input: models.UserRegistration{
				Email:     "test@example.com",
				Phone:     "invalid-phone",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "SecurePass123!",
				Language:  "en",
			},
			shouldFail:  true,
			description: "Should fail with invalid phone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateStruct(tt.input)

			if tt.shouldFail {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"Strong password", "SecurePass123!", false},
		{"Missing uppercase", "securepass123!", true},
		{"Missing lowercase", "SECUREPASS123!", true},
		{"Missing number", "SecurePass!", true},
		{"Missing special char", "SecurePass123", true},
		{"Too short", "Sec1!", true},
		{"Too long", string(make([]byte, 130)), true},
		{"Empty password", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := utils.ValidatePassword(tt.password)
			hasError := len(errors) > 0

			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean input",
			input:    "John Doe",
			expected: "John Doe",
		},
		{
			name:     "XSS attempt",
			input:    "<script>alert('xss')</script>",
			expected: "alert(xss)", // Single quotes are removed by sanitization
		},
		{
			name:     "SQL injection",
			input:    "'; DROP TABLE users; --",
			expected: "DROP TABLE users", // Semicolons, quotes, and -- are removed
		},
		{
			name:     "Control characters",
			input:    "John\x00\x01Doe",
			expected: "JohnDoe",
		},
		{
			name:     "Mixed dangerous content",
			input:    "John<script>alert('test')</script>'; DROP TABLE users; --",
			expected: "Johnalert(test) DROP TABLE users", // Quotes, semicolons, and -- are removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply auth rate limiting
	router.Use(middleware.AuthRateLimitMiddleware())

	router.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Make multiple requests to test rate limiting
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "TestAgent/1.0")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if i < 5 {
			// First 5 requests should succeed
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			// Subsequent requests should be rate limited
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}
