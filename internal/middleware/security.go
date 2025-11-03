package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	MaxRequestSize    int64
	RateLimitRequests int
	RateLimitWindow   time.Duration
	RequireHTTPS      bool
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MaxRequestSize:    10 * 1024 * 1024, // 10MB
		RateLimitRequests: 10000,             // Very high for development
		RateLimitWindow:   time.Minute,
		RequireHTTPS:      false,         // Set to true in production
	}
}

// SecurityMiddleware provides comprehensive security protection
func SecurityMiddleware(config *SecurityConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	// Rate limiter per IP
	limiters := make(map[string]*rate.Limiter)

	return func(c *gin.Context) {

		// 1. Request size validation
		if c.Request.ContentLength > config.MaxRequestSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"success": false,
				"error":   "Request body too large",
			})
			c.Abort()
			return
		}

		// 2. Rate limiting per IP (skip if disabled for development)
		if os.Getenv("DISABLE_RATE_LIMITING") != "true" {
			clientIP := c.ClientIP()
			limiter, exists := limiters[clientIP]
			if !exists {
				limiter = rate.NewLimiter(rate.Every(config.RateLimitWindow/time.Duration(config.RateLimitRequests)), config.RateLimitRequests)
				limiters[clientIP] = limiter
			}

			if !limiter.Allow() {
				fmt.Printf("🚨 Rate limit exceeded for IP: %s, Path: %s %s\n", clientIP, c.Request.Method, c.Request.URL.Path)

				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error":   "Rate limit exceeded",
				})
				c.Abort()
				return
			}
		}

		// 3. Content-Type validation for POST/PUT requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			// fmt.Printf("🔍 Security Middleware - Content-Type: '%s' for %s %s\n", contentType, c.Request.Method, c.Request.URL.Path)

			// Skip Content-Type validation for upload endpoints to debug
			if strings.Contains(c.Request.URL.Path, "/upload/") {
				// fmt.Printf("🔧 Skipping Content-Type validation for upload endpoint\n")
				c.Next()
				return
			}

			if contentType == "" {
				// fmt.Printf("❌ Missing Content-Type header for %s %s\n", c.Request.Method, c.Request.URL.Path)
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Content-Type header required",
				})
				c.Abort()
				return
			}

			validContentTypes := []string{
				"application/json",
				"multipart/form-data",
				"application/x-www-form-urlencoded",
			}

			isValid := false
			for _, validType := range validContentTypes {
				if strings.Contains(contentType, validType) {
					isValid = true
					break
				}
			}

			if !isValid {
				fmt.Printf("❌ Invalid Content-Type: '%s' for %s %s\n", contentType, c.Request.Method, c.Request.URL.Path)
				fmt.Printf("❌ Valid types: %v\n", validContentTypes)
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"success": false,
					"error":   "Unsupported content type: " + contentType,
				})
				c.Abort()
				return
			}

			fmt.Printf("✅ Valid Content-Type: '%s'\n", contentType)
		}

		// 4. Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Dynamic CSP based on route - more permissive for OAuth callback
		if strings.Contains(c.Request.URL.Path, "/auth/google/callback") {
			// Allow inline styles and scripts for OAuth success page
			c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:")
		} else {
			// Strict CSP for other routes
			c.Header("Content-Security-Policy", "default-src 'self'")
		}

		// 5. HTTPS enforcement (if enabled)
		if config.RequireHTTPS && c.Request.Header.Get("X-Forwarded-Proto") != "https" {
			c.JSON(http.StatusUpgradeRequired, gin.H{
				"success": false,
				"error":   "HTTPS required",
			})
			c.Abort()
			return
		}

		// 6. Validate User-Agent (block empty or suspicious agents)
		userAgent := c.GetHeader("User-Agent")
		if userAgent == "" || len(userAgent) < 10 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid User-Agent",
			})
			c.Abort()
			return
		}

		// 7. Block suspicious patterns in URL
		suspiciousPatterns := []string{
			"../", "..\\", "<script", "javascript:", "vbscript:",
			"onload=", "onerror=", "eval(", "expression(",
		}

		requestURI := strings.ToLower(c.Request.RequestURI)
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(requestURI, pattern) {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Suspicious request pattern detected",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// InputValidationMiddleware validates common input patterns
func InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if len(value) > 1000 { // Max query param length
					c.JSON(http.StatusBadRequest, gin.H{
						"success": false,
						"error":   "Query parameter too long: " + key,
					})
					c.Abort()
					return
				}

				// Check for injection patterns
				dangerous := []string{
					"'", "\"", "<script", "javascript:", "onload=", "onerror=",
					"onclick=", "onmouseover=", "onfocus=", "onblur=", "onchange=",
					"onsubmit=", "<iframe", "<object", "<embed", "<link", "<meta",
					"data:text/html", "eval(", "expression(", "url(javascript:",
					"&#", "&#x", "<svg", "<img", "union", "select", "insert",
					"update", "delete", "drop", "create", "alter", "truncate",
					"exec", "execute", "declare", "cast", "convert", "grant", "revoke",
				}
				lowerValue := strings.ToLower(value)
				for _, pattern := range dangerous {
					if strings.Contains(lowerValue, pattern) {
						c.JSON(http.StatusBadRequest, gin.H{
							"success": false,
							"error":   "Invalid characters in query parameter: " + key,
						})
						c.Abort()
						return
					}
				}
			}
		}

		c.Next()
	}
}

// FileUploadSecurityMiddleware validates file uploads
func FileUploadSecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			fmt.Printf("🔍 FileUpload Middleware - Content-Type: %s\n", contentType)
			fmt.Printf("🔍 FileUpload Middleware - Method: %s, Path: %s\n", c.Request.Method, c.Request.URL.Path)

			if strings.Contains(contentType, "multipart/form-data") {
				fmt.Printf("🔍 Parsing multipart form data...\n")
				// Parse multipart form with size limit
				err := c.Request.ParseMultipartForm(5 * 1024 * 1024) // 5MB limit
				if err != nil {
					fmt.Printf("❌ Failed to parse multipart form: %v\n", err)
					c.JSON(http.StatusBadRequest, gin.H{
						"success": false,
						"error":   "Failed to parse multipart form: " + err.Error(),
					})
					c.Abort()
					return
				}
				fmt.Printf("✅ Multipart form parsed successfully\n")

				// Validate uploaded files
				if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
					// Different allowed types based on endpoint
					var allowedTypes map[string]bool

					// Check if this is a meeting document upload
					if strings.Contains(c.Request.URL.Path, "/meetings/") && strings.Contains(c.Request.URL.Path, "/documents") {
						// Allow more file types for meeting documents
						allowedTypes = map[string]bool{
							"image/jpeg":         true,
							"image/jpg":          true,
							"image/png":          true,
							"image/webp":         true,
							"image/gif":          true,
							"application/pdf":    true,
							"application/msword": true,
							"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // .docx
							"application/vnd.ms-excel": true,
							"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true, // .xlsx
							"application/vnd.ms-powerpoint":                                             true,
							"application/vnd.openxmlformats-officedocument.presentationml.presentation": true, // .pptx
							"text/plain": true,
							"text/csv":   true,
						}
					} else {
						// Default to images only for other endpoints (like avatar uploads)
						allowedTypes = map[string]bool{
							"image/jpeg": true,
							"image/jpg":  true,
							"image/png":  true,
							"image/webp": true,
							"image/gif":  true,
						}
					}

					for _, files := range c.Request.MultipartForm.File {
						for _, file := range files {
							// Check file size
							if file.Size > 5*1024*1024 { // 5MB per file
								c.JSON(http.StatusBadRequest, gin.H{
									"success": false,
									"error":   "File too large: " + file.Filename,
								})
								c.Abort()
								return
							}

							// Check file type
							contentType := file.Header.Get("Content-Type")
							if !allowedTypes[contentType] {
								c.JSON(http.StatusBadRequest, gin.H{
									"success": false,
									"error":   "Invalid file type: " + file.Filename,
								})
								c.Abort()
								return
							}

							// Check filename for dangerous patterns
							filename := strings.ToLower(file.Filename)
							dangerousExtensions := []string{".exe", ".bat", ".cmd", ".scr", ".pif", ".js", ".vbs", ".php", ".asp"}
							for _, ext := range dangerousExtensions {
								if strings.HasSuffix(filename, ext) {
									c.JSON(http.StatusBadRequest, gin.H{
										"success": false,
										"error":   "Dangerous file type: " + file.Filename,
									})
									c.Abort()
									return
								}
							}
						}
					}
				}
			}
		}

		c.Next()
	}
}

// AuthRateLimitMiddleware provides stricter rate limiting for auth endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	// Very high rate limiting for development: 500 requests per minute per IP
	authLimiters := make(map[string]*rate.Limiter)

	return func(c *gin.Context) {
		// Skip rate limiting if disabled for development
		if os.Getenv("DISABLE_RATE_LIMITING") == "true" {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		limiter, exists := authLimiters[clientIP]
		if !exists {
			// 500 requests per minute for auth endpoints (very high for development)
			limiter = rate.NewLimiter(rate.Every(time.Minute/500), 500)
			authLimiters[clientIP] = limiter
		}

		if !limiter.Allow() {
			fmt.Printf("🚨 Auth rate limit exceeded for IP: %s, Path: %s %s\n", clientIP, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Too many authentication attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
