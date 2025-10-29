package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"vaultke-backend/config"
	"vaultke-backend/database"
	"vaultke-backend/internal/api"
	"vaultke-backend/internal/middleware"
	"vaultke-backend/internal/services"
)

// Custom response writer to ensure CORS headers on all responses including redirects
type corsResponseWriter struct {
	gin.ResponseWriter
	origin string
}

func (w *corsResponseWriter) WriteHeader(code int) {
	// Ensure CORS headers are set for ALL responses, including redirects
	w.Header().Set("Access-Control-Allow-Origin", w.origin)
	w.Header().Set("Access-Control-Allow-Credentials", "false")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin, Cache-Control, X-CSRF-Token, X-File-Name, X-File-Size, X-Timezone, X-Language, X-Screen-Resolution, X-Device-Type, X-Device-Name, X-Browser-Name, X-OS-Name, X-Connection-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Authorization, Content-Disposition")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.ResponseWriter.WriteHeader(code)
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Run database migrations
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Run notification system migrations
	migrationManager := database.NewMigrationManager(db)
	if err := migrationManager.RunMigrations(); err != nil {
		log.Fatal("Failed to run notification system migrations:", err)
	}

	// Initialize Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add memory monitoring middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Log memory-intensive requests
		if duration > 5*time.Second {
			log.Printf("🚨 SLOW REQUEST: %s %s took %v", c.Request.Method, c.Request.URL.Path, duration)
		}
	})

	// Load HTML templates for OAuth pages

	router.LoadHTMLGlob("templates/*")

	// Middleware
	// router.Use(gin.Logger()) // Commented out to reduce log noise
	// router.Use(gin.Recovery())

	// HSTS middleware for production
	if os.Getenv("ENVIRONMENT") == "production" {
		router.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			c.Next()
		})
	}

	// Smart CORS middleware - handles localhost and production URLs
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method
		path := c.Request.URL.Path

		// Define allowed origins for different environments
		allowedOrigins := []string{
			"https://gitrepoa-1.onrender.com", // your backend domain
			"http://localhost:8081",           // Metro / Expo
			"http://127.0.0.1:8081",
			"http://localhost:8080", 
			"http://localhost:19006", // Expo web preview
			"http://127.0.0.1:19006",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}

		// Check if origin is allowed - be restrictive in production
		allowedOrigin := ""
		if origin != "" {
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					allowedOrigin = origin
					break
				}
			}
		}

		// In production, only allow specific origins - no fallback to "*"
		if allowedOrigin == "" && os.Getenv("ENVIRONMENT") == "production" {
			log.Printf("🚫 CORS: Origin '%s' not allowed in production", origin)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Origin not allowed",
			})
			return
		}

		// For development, allow "*" if no specific match (for flexibility during development)
		if allowedOrigin == "" && os.Getenv("ENVIRONMENT") != "production" {
			allowedOrigin = "*"
		}

		// Log CORS processing for debugging
		log.Printf("🔒 CORS: Origin=%s, Method=%s, Path=%s, AllowedOrigin=%s", origin, method, path, allowedOrigin)
		log.Printf("🔒 CORS: Headers - Origin:%s, Host:%s, User-Agent:%s", c.GetHeader("Origin"), c.GetHeader("Host"), c.GetHeader("User-Agent"))

		// Replace response writer with CORS-enabled one
		c.Writer = &corsResponseWriter{
			ResponseWriter: c.Writer,
			origin:         allowedOrigin,
		}

		// Handle preflight OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			log.Printf("🔒 CORS: Handling OPTIONS preflight for %s", path)
			// Set CORS headers for OPTIONS requests
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "false")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin, Cache-Control, X-CSRF-Token, X-File-Name, X-File-Size, X-Timezone, X-Language, X-Screen-Resolution, X-Device-Type, X-Device-Name, X-Browser-Name, X-OS-Name, X-Connection-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Authorization, Content-Disposition")
			c.Header("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Disable rate limiting for development
	if os.Getenv("DISABLE_RATE_LIMITING") != "true" {
		securityConfig := middleware.DefaultSecurityConfig()
		router.Use(middleware.SecurityMiddleware(securityConfig))
	}

	router.Use(middleware.InputValidationMiddleware())
	router.Use(middleware.FileUploadSecurityMiddleware())

	// IP debugging middleware for tunneled environments
	router.Use(func(c *gin.Context) {
		// Only log for non-OPTIONS requests to avoid spam
		if c.Request.Method != "OPTIONS" {
			log.Printf("🌐 REQUEST: %s %s - Proto:%s, Host:%s, RemoteAddr:%s", c.Request.Method, c.Request.URL.String(), c.Request.Proto, c.GetHeader("Host"), c.Request.RemoteAddr)
		}
		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "VaultKe API is running",
			"version": "1.0.0",
		})
	})

	// Legal pages for Google OAuth
	router.GET("/privacy-policy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy-policy.html", nil)
	})
	router.GET("/terms-of-service", func(c *gin.Context) {
		c.HTML(http.StatusOK, "terms-of-service.html", nil)
	})

	// Serve static files (uploaded images)
	router.Static("/uploads", "./uploads")

	// Serve notification sound files
	router.Static("/notification_sound", "./notification_sound")

	// Initialize services
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpiration)
	authMiddleware := middleware.NewAuthMiddleware(authService)
	wsService := services.NewWebSocketService(db)

	// Initialize email service
	emailService := services.NewEmailService()

	// Initialize password reset service
	passwordResetService := services.NewPasswordResetService(db, emailService)

	// Initialize password reset table
	if err := passwordResetService.InitializePasswordResetTable(); err != nil {
		log.Fatalf("Failed to initialize password reset table: %v", err)
	}

	// Initialize email verification service
	emailVerificationService := services.NewEmailVerificationService(db, emailService)

	// Initialize email verification table
	if err := emailVerificationService.InitializeEmailVerificationTable(); err != nil {
		log.Fatalf("Failed to initialize email verification table: %v", err)
	}

	// Initialize notification service
	notificationService := services.NewNotificationService(db, cfg)

	// Initialize LiveKit meeting service
	api.InitializeMeetingService(db, notificationService)

	// Initialize notification scheduler for reminders
	notificationScheduler := services.NewNotificationScheduler(db)
	notificationScheduler.Start()

	// Initialize scheduler service for meeting auto-unlock
	// Note: You'll need to get the meeting service instance to pass here
	// For now, we'll initialize it separately in the API package

	// Initialize handlers
	authHandlers := api.NewAuthHandlers(db, cfg.JWTSecret, cfg.JWTExpiration)
	reminderHandlers := api.NewReminderHandlers(db)
	sharesHandlers := api.NewSharesHandlers(db)
	dividendsHandlers := api.NewDividendsHandlers(db)
	pollsHandlers := api.NewPollsHandlers(db)
	disbursementHandlers := api.NewDisbursementHandlers(db)
	reportsHandlers := api.NewFinancialReportsHandlers(db)
	// deliveryContactsHandlers := api.NewDeliveryContactsHandlers(db)
	userSearchHandlers := api.NewUserSearchHandlers(db)
	receiptHandlers := api.NewReceiptHandlers(db)
	moneyRequestHandlers := api.NewMoneyRequestHandlers(db)
	accountHandlers := api.NewAccountHandlers(db)

	// Initialize E2EE service
	e2eeService := services.NewMilitaryGradeE2EEService(db)

	// Database middleware to inject db into context
	dbMiddleware := func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}

	// Configuration middleware to inject config into context
	configMiddleware := func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	}

	// WebSocket middleware to inject wsService into context
	wsMiddleware := func(c *gin.Context) {
		c.Set("wsService", wsService)
		c.Next()
	}

	// Password reset middleware to inject passwordResetService into context
	passwordResetMiddleware := func(c *gin.Context) {
		c.Set("passwordResetService", passwordResetService)
		c.Next()
	}

	// Email verification middleware to inject emailVerificationService into context
	emailVerificationMiddleware := func(c *gin.Context) {
		c.Set("emailVerificationService", emailVerificationService)
		c.Next()
	}

	// E2EE middleware to inject e2eeService into context
	e2eeMiddleware := func(c *gin.Context) {
		c.Set("e2eeService", e2eeService)
		c.Next()
	}

	// API routes
	apiGroup := router.Group("/api/v1")
	{
		// Health check endpoint (public, no authentication required)
		apiGroup.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"success":   true,
				"status":    "healthy",
				"message":   "VaultKe API is running",
				"timestamp": time.Now().Unix(),
			})
		})

		// Authentication routes with stricter rate limiting
		auth := apiGroup.Group("/auth")
		auth.Use(middleware.AuthRateLimitMiddleware()) // Stricter rate limiting for auth endpoints
		auth.Use(dbMiddleware)                         // Add database context for login session recording
		auth.Use(passwordResetMiddleware)              // Add password reset service to context
		auth.Use(emailVerificationMiddleware)          // Add email verification service to context
		{
			auth.POST("/register", authHandlers.Register)
			auth.POST("/login", authHandlers.Login)
			auth.POST("/logout", authMiddleware.AuthRequired(), authHandlers.Logout)
			auth.POST("/refresh", authHandlers.RefreshToken)
			auth.POST("/verify-email", authMiddleware.AuthRequired(), authHandlers.VerifyEmail)
			auth.POST("/verify-phone", authMiddleware.AuthRequired(), authHandlers.VerifyPhone)
			auth.POST("/forgot-password", authHandlers.ForgotPassword)
			auth.POST("/reset-password", authHandlers.ResetPassword)
			auth.POST("/check-token-status", authHandlers.CheckTokenStatus)

			// Email verification routes
			auth.POST("/send-email-verification", authHandlers.SendEmailVerification)
			auth.POST("/verify-email-code", authHandlers.VerifyEmailCode)
			auth.POST("/check-email-verification-status", authHandlers.CheckEmailVerificationStatus)

			auth.POST("/test-email", authHandlers.TestEmail) // For development testing
		}

		// WebSocket route (handles auth internally)
		apiGroup.GET("/ws", wsService.HandleWebSocket)

		// Public marketplace routes (no authentication required for browsing)
		// publicMarketplace := apiGroup.Group("/marketplace")
		// publicMarketplace.Use(dbMiddleware)
		// publicMarketplace.Use(wsMiddleware)
		// {
		// 	publicProducts := publicMarketplace.Group("/products")
		// 	{
		// 		publicProducts.GET("/", api.GetProducts)     // ✅ Public - browse products
		// 		publicProducts.GET("/all", api.GetProducts)  // ✅ Public - get all products (same as above but with higher default limit)
		// 		publicProducts.GET("/:id", api.GetProduct)   // ✅ Public - view product details
		// 	}

		// 	// Public categories endpoint
		// 	publicMarketplace.GET("/categories", api.GetMarketplaceCategories) // ✅ Public - get categories with counts
		// }

		// Public payment routes (no authentication required for callbacks)
		publicPayments := apiGroup.Group("/payments")
		publicPayments.Use(dbMiddleware)
		publicPayments.Use(configMiddleware)
		{
			publicPayments.POST("/mpesa/callback", api.HandleMpesaCallback)
		}

		// Public Google Drive OAuth routes (no authentication required)
		publicAuth := apiGroup.Group("/auth")
		{
			publicAuth.GET("/google/drive", api.InitiateGoogleDriveAuth)
			publicAuth.GET("/google/callback", api.HandleGoogleDriveCallback)
		}

		// Protected routes
		protected := apiGroup.Group("/")
		protected.Use(authMiddleware.AuthRequired())
		protected.Use(dbMiddleware)
		protected.Use(configMiddleware)
		protected.Use(wsMiddleware)
		protected.Use(e2eeMiddleware)
		{
			// User routes
			users := protected.Group("/users")
			{
				users.GET("/", api.GetUsers)
				users.GET("/:id", authHandlers.GetUserByID)            // Get user by ID
				users.GET("/admin/all", api.GetAllUsersForAdmin)       // Admin endpoint to get all users
				users.GET("/admin/statistics", api.GetAdminStatistics) // Admin statistics endpoint
				users.GET("/admin/analytics", api.GetSystemAnalytics)  // System analytics endpoint
				users.GET("/profile", authHandlers.GetProfile)
				users.PUT("/profile", authHandlers.UpdateProfile)
				users.GET("/statistics", api.GetUserStatistics) // User statistics endpoint

				// Google Drive backup routes
				users.GET("/google-drive/auth-url", api.GetGoogleDriveAuthURL)
				users.POST("/google-drive/store-tokens", api.StoreGoogleDriveTokens)
				users.POST("/google-drive/disconnect", api.DisconnectGoogleDrive)
				users.POST("/google-drive/backup", api.CreateGoogleDriveBackup)
				users.POST("/google-drive/restore", api.RestoreGoogleDriveBackup)
				users.GET("/google-drive/backup-info", api.GetGoogleDriveBackupInfo)
				users.GET("/google-drive/status", api.GetGoogleDriveStatus)
				users.GET("/google-drive/debug-tokens", api.DebugGoogleDriveTokens)
				users.POST("/google-drive/generate-test-tokens", api.GenerateTestTokens)

				// Backup maintenance endpoints
				users.GET("/backup/history", api.GetBackupHistory)
				users.GET("/backup/system-status", api.GetSystemStatus)
				users.GET("/backup/settings", api.GetBackupSettings)
				users.PUT("/backup/settings", api.UpdateBackupSettings)
				users.POST("/backup/start", api.StartBackup)
				users.POST("/backup/maintenance", api.PerformSystemMaintenance)

				// User settings routes
				users.GET("/privacy-settings", api.GetPrivacySettings)
				users.PUT("/privacy-settings", api.UpdatePrivacySettings)
				users.GET("/security-settings", api.GetSecuritySettings)
				users.PUT("/security-settings", api.UpdateSecuritySettings)
				users.GET("/preferences", api.GetUserPreferences)
				users.PUT("/preferences", api.UpdateUserPreferences)

				users.POST("/avatar", api.UploadAvatar)
				users.PUT("/:id/role", api.AdminUpdateUserRole) // Admin endpoint to update user role
				users.PUT("/:id/status", api.UpdateUserStatus)  // Admin endpoint to update user status
				users.DELETE("/:id", api.DeleteUser)            // Admin endpoint to delete user
			}

			// E2EE routes
			e2ee := protected.Group("/e2ee")
			{
				e2ee.POST("/devices/register", api.RegisterDevice)
				e2ee.GET("/devices", api.GetDevices)
				e2ee.POST("/pre-keys/upload", api.UploadPreKeys)
				e2ee.GET("/pre-key-bundle/:userId", api.GetPreKeyBundle)
				e2ee.POST("/messages/send", api.SendE2EEMessage)
				e2ee.GET("/messages", api.GetMessages)
				e2ee.GET("/safety-number/:userId", api.GetSafetyNumber)
				e2ee.POST("/keys/rotate", api.RotateKeys)
				e2ee.POST("/session/reset/:userId", api.ResetSession)
				e2ee.POST("/initialize", api.InitializeE2EEKeys)
				e2ee.GET("/key-bundle/:userId", api.GetE2EEKeyBundle)
				e2ee.POST("/encrypt", api.EncryptMessage)
				e2ee.POST("/decrypt", api.DecryptMessage)
				e2ee.GET("/security-status", api.GetE2EESecurityStatus)
			}

			// Chama routes
			chamas := protected.Group("/chamas")
			{
				chamas.GET("/", api.GetChamas)
				chamas.GET("/admin/all", api.GetAllChamasForAdmin) // Admin endpoint to get all chamas
				chamas.POST("/", api.CreateChama)
				chamas.GET("/my", api.GetUserChamas)
				chamas.GET("/:id", api.GetChama)
				chamas.PUT("/:id", api.UpdateChama)
				chamas.DELETE("/:id", api.DeleteChama)
				chamas.GET("/:id/members", api.GetChamaMembers)
				chamas.GET("/:id/members/:userId/role", api.GetMemberRole)
				chamas.POST("/:id/join", api.JoinChama)
				chamas.POST("/:id/leave", api.LeaveChama)
				chamas.GET("/:id/transactions", api.GetChamaTransactions)
				chamas.GET("/:id/statistics", api.GetChamaStatistics)

				// Invitation routes
				chamas.POST("/:id/invite", api.SendChamaInvitation)
				chamas.GET("/:id/invitations/sent", api.GetChamaSentInvitations)
				chamas.GET("/invitations", api.GetUserInvitations)
				// Match frontend URL pattern: /chamas/{chamaId}/invitations/{invitationId}/respond
				// Use :id for chamaId to avoid parameter name conflicts with existing routes
				chamas.POST("/:id/invitations/:invitationId/respond", api.RespondToInvitation)
				chamas.POST("/:id/invitations/:invitationId/cancel", api.CancelInvitation)
				chamas.POST("/:id/invitations/:invitationId/resend", api.ResendInvitation)

				// Disbursement and Creation routes
				chamas.GET("/:id/eligible-loan-members", api.GetEligibleLoanMembers)
				chamas.GET("/:id/eligible-welfare-members", api.GetEligibleWelfareMembers)
				chamas.GET("/:id/eligible-dividend-members", api.GetEligibleDividendMembers)
				chamas.GET("/:id/eligible-shares-members", api.GetEligibleSharesMembers)
				chamas.GET("/:id/eligible-savings-members", api.GetEligibleSavingsMembers)
				chamas.GET("/:id/eligible-other-members", api.GetEligibleOtherMembers)
				chamas.POST("/:id/disbursements/individual", api.CreateIndividualDisbursement)
				chamas.POST("/:id/disbursements/bulk", api.CreateBulkDisbursement)
				chamas.POST("/:id/shares", api.CreateChamaShares)
			}

			// Wallet routes
			wallets := protected.Group("/wallets")
			{
				wallets.GET("/", api.GetWallets)
				wallets.GET("/balance", api.GetWalletBalance)
				wallets.GET("/transactions", api.GetUserTransactions)
				wallets.GET("/:id", api.GetWallet)
				wallets.GET("/:id/transactions", api.GetWalletTransactions)
				wallets.POST("/transfer", api.TransferMoney)
				wallets.POST("/deposit", api.DepositMoney)
				wallets.POST("/withdraw", api.WithdrawMoney)
			}

			// Money request routes (part of wallet functionality)
			wallet := protected.Group("/wallet")
			{
				wallet.POST("/create-money-request", moneyRequestHandlers.CreateMoneyRequest)
				wallet.POST("/send-money-request", moneyRequestHandlers.SendMoneyRequest)
				wallet.GET("/recent-contacts", moneyRequestHandlers.GetRecentContacts)
			}

			// Receipt routes
			receipts := protected.Group("/receipts")
			{
				receipts.GET("/transactions/:transactionId", receiptHandlers.GetTransactionReceipt)
				receipts.GET("/transactions/:transactionId/download", receiptHandlers.DownloadTransactionReceipt)
			}

			// Protected marketplace routes (authentication required)
			// marketplace := protected.Group("/marketplace")
			// {
			// 	products := marketplace.Group("/products")
			// 	{
			// 		products.GET("/manage/all", api.GetAllProducts)           // ✅ Auth required - get all products for management (including out of stock)
			// 		products.GET("/:id/details", api.GetProductWithOwnership) // ✅ Enhanced product details with ownership
			// 		products.POST("/", api.CreateProduct)                     // ✅ Auth required - create product
			// 		products.PUT("/:id", api.UpdateProduct)                   // ✅ Auth required - update product
			// 		products.DELETE("/:id", api.DeleteProduct)                // ✅ Auth required - delete product
			// 	}

			// 	marketplace.GET("/cart", api.GetCart)               // ✅ Auth required - view cart
			// 	marketplace.POST("/cart", api.AddToCart)            // ✅ Auth required - add to cart
			// 	marketplace.DELETE("/cart/:id", api.RemoveFromCart) // ✅ Auth required - remove from cart

			// 	marketplace.GET("/wishlist", api.GetWishlist)                      // ✅ Auth required - view wishlist
			// 	marketplace.POST("/wishlist", api.AddToWishlist)                   // ✅ Auth required - add to wishlist
			// 	marketplace.DELETE("/wishlist/:productId", api.RemoveFromWishlist) // ✅ Auth required - remove from wishlist

			// 	orders := marketplace.Group("/orders")
			// 	{
			// 		orders.GET("/", api.GetOrders)                                                 // ✅ Auth required - view orders
			// 		orders.POST("/", api.CreateOrder)                                              // ✅ Auth required - create order
			// 		orders.GET("/:id", api.GetOrder)                                               // ✅ Auth required - view order details
			// 		orders.PUT("/:id", api.UpdateOrder)                                            // ✅ Auth required - update order
			// 		orders.PUT("/:id/status", api.UpdateOrderStatus)                               // ✅ Auth required - update order status
			// 		orders.POST("/:id/assign-delivery", api.AssignDeliveryPerson)                  // ✅ Auth required - assign delivery (legacy)
			// 		orders.POST("/assign-delivery-to-products", api.AssignDeliveryPersonToProduct) // ✅ Auth required - assign delivery to specific products
			// 	}

			// 	// Analytics routes
			// 	analytics := marketplace.Group("/analytics")
			// 	{
			// 		analytics.GET("/seller", api.GetSellerAnalytics) // ✅ Auth required - seller analytics
			// 		analytics.GET("/buyer", api.GetBuyerStats)       // ✅ Auth required - buyer stats
			// 	}

			// 	// Delivery routes
			// 	deliveries := marketplace.Group("/deliveries")
			// 	{
			// 		deliveries.GET("/", api.GetDeliveries)                  // ✅ Auth required - view deliveries
			// 		deliveries.POST("/:id/accept", api.AcceptDelivery)      // ✅ Auth required - accept delivery
			// 		deliveries.PUT("/:id/status", api.UpdateDeliveryStatus) // ✅ Auth required - update delivery status
			// 	}

			// 	marketplace.GET("/reviews/product/:productId", api.GetReviews)                  // ✅ Auth required - view product reviews
			// 	marketplace.GET("/reviews/product/:productId/stats", api.GetProductReviewStats) // ✅ Auth required - get review stats
			// 	marketplace.POST("/reviews", api.CreateReview)                                  // ✅ Auth required - create review
			// 	marketplace.GET("/reviews/my", api.GetMyReviews)                                // ✅ Auth required - get user's reviews
			// }

			// Payment routes
			payments := protected.Group("/payments")
			{
				payments.POST("/mpesa/stk", api.InitiateMpesaSTK)
				payments.GET("/mpesa/status/:checkoutRequestId", api.GetMpesaTransactionStatus)
				payments.POST("/bank-transfer", api.InitiateBankTransfer)
			}

			// Chat routes
			chat := protected.Group("/chat")
			{
				chat.GET("/rooms", api.GetChatRooms)
				chat.POST("/rooms", api.CreateChatRoom)
				chat.GET("/rooms/:id", api.GetChatRoom)
				chat.POST("/rooms/:id/join", api.JoinChatRoom) // Added missing join endpoint
				chat.GET("/rooms/:id/members", api.GetChatRoomMembers)
				chat.DELETE("/rooms/:id", api.DeleteChatRoom)
				chat.POST("/rooms/:id/clear", api.ClearChatRoom)
				chat.GET("/rooms/:id/messages", api.GetChatMessages)
				chat.POST("/rooms/:id/messages", api.SendMessage)
				chat.PUT("/rooms/:id/read", api.MarkMessagesAsRead)
			}

			// Notifications routes
			notifications := protected.Group("/notifications")
			{
				notifications.GET("/", api.GetNotifications)
				notifications.GET("/unread-count", api.GetUnreadNotificationCount)
				notifications.PUT("/:id/read", api.MarkNotificationAsRead)
				notifications.POST("/read-all", api.MarkAllNotificationsAsRead)
				notifications.DELETE("/:id", api.DeleteNotification)
				notifications.POST("/system", api.SendSystemNotification)

				// Notification preferences routes
				notifications.GET("/preferences", api.GetNotificationPreferences)
				notifications.PUT("/preferences", api.UpdateNotificationPreferences)
				notifications.GET("/sounds", api.GetAvailableNotificationSounds)
				notifications.POST("/test-sound", api.TestNotificationSound)

				// Legacy notification settings routes (for backward compatibility)
				notifications.GET("/settings", api.GetNotificationSettings)
				notifications.PUT("/settings", api.UpdateNotificationSettings)

				// Chama invitation response routes
				notifications.POST("/invitations/:id/accept", api.AcceptChamaInvitation)
				notifications.POST("/invitations/:id/reject", api.RejectChamaInvitation)
			}

			// Support routes
			support := protected.Group("/support")
			{
				support.POST("/requests", api.CreateSupportRequest)
				support.GET("/requests", api.GetSupportRequests)
				support.PUT("/requests/:id", api.UpdateSupportRequest)
				support.POST("/test-request", api.CreateTestSupportRequest) // Test endpoint
			}

			// Security routes
			auth := protected.Group("/auth")
			{
				auth.POST("/change-password", api.ChangePassword)
				auth.GET("/login-history", api.GetLoginHistory)
				auth.POST("/logout-all-devices", api.LogoutAllDevices)
				auth.POST("/logout-device/:sessionId", api.LogoutSpecificDevice)
			}

			// Learning routes
			learning := protected.Group("/learning")
			{
				// Public learning routes (require authentication but not admin)
				learning.GET("/categories", api.GetLearningCategories)
				learning.GET("/categories/:id", api.GetLearningCategory)
				learning.GET("/courses", api.GetLearningCourses)
				learning.GET("/courses/:id", api.GetLearningCourse)
				learning.POST("/courses/:id/start", api.StartCourse)
				learning.POST("/courses/:id/submit-quiz", api.SubmitQuizResults)

				// Learning content upload routes (require authentication)
				learning.POST("/upload/image", api.UploadLearningImage)
				learning.POST("/upload/video", api.UploadLearningVideo)
				learning.POST("/upload/document", api.UploadLearningDocument)
				learning.POST("/validate-video-url", api.ValidateVideoURL)

				// Admin learning routes
				admin := learning.Group("/admin")
				admin.Use(func(c *gin.Context) {
					userRole := c.GetString("userRole")
					if userRole != "admin" {
						c.JSON(http.StatusForbidden, gin.H{
							"success": false,
							"error":   "Admin access required",
						})
						c.Abort()
						return
					}
					c.Next()
				})
				{
					// Category management
					admin.POST("/categories", api.CreateLearningCategory)
					admin.PUT("/categories/:id", api.UpdateLearningCategory)
					admin.DELETE("/categories/:id", api.DeleteLearningCategory)

					// Course management
					admin.POST("/courses", api.CreateLearningCourse)
					admin.PUT("/courses/:id", api.UpdateLearningCourse)
					admin.DELETE("/courses/:id", api.DeleteLearningCourse)
				}
			}

			// Reminder routes
			reminders := protected.Group("/reminders")
			{
				reminders.POST("/", reminderHandlers.CreateReminder)
				reminders.GET("/", reminderHandlers.GetUserReminders)
				reminders.GET("/:id", reminderHandlers.GetReminder)
				reminders.PUT("/:id", reminderHandlers.UpdateReminder)
				reminders.DELETE("/:id", reminderHandlers.DeleteReminder)
				reminders.POST("/:id/toggle", reminderHandlers.ToggleReminder)
			}

			// Shares routes
			shares := protected.Group("/chamas/:id/shares")
			{
				shares.POST("/offering", sharesHandlers.CreateShareOffering)
				shares.POST("/", sharesHandlers.CreateShares) // Legacy endpoint
				shares.POST("/buy", sharesHandlers.BuyShares)
				shares.POST("/buy-dividends", sharesHandlers.BuyDividends)
				shares.POST("/transfer", sharesHandlers.TransferShares)
				shares.GET("/", sharesHandlers.GetChamaShares)
				shares.GET("/summary", sharesHandlers.GetChamaSharesSummary)
				shares.GET("/transactions", sharesHandlers.GetShareTransactions)
				shares.GET("/members/:memberId", sharesHandlers.GetMemberShares)
				shares.PUT("/:shareId", sharesHandlers.UpdateShares)
			}

			// Dividends routes
			dividends := protected.Group("/chamas/:id/dividends")
			{
				dividends.POST("/", dividendsHandlers.DeclareDividend)
				dividends.GET("/", dividendsHandlers.GetChamaDividendDeclarations)
				dividends.GET("/:declarationId", dividendsHandlers.GetDividendDeclarationDetails)
				dividends.POST("/:declarationId/approve", dividendsHandlers.ApproveDividend)
				dividends.POST("/:declarationId/process", dividendsHandlers.ProcessDividendPayments)
				dividends.GET("/members/:memberId/history", dividendsHandlers.GetMemberDividendHistory)
				dividends.GET("/my-history", dividendsHandlers.GetMyDividendHistory)
			}

			// Polls and Voting routes (new system - currently broken due to table conflicts)
			polls := protected.Group("/chamas/:id/polls")
			{
				polls.POST("/", pollsHandlers.CreatePoll)
				polls.GET("/", pollsHandlers.GetChamaPolls)
				polls.GET("/active", pollsHandlers.GetActivePolls)
				polls.GET("/results", pollsHandlers.GetPollResults)
				polls.GET("/:pollId", pollsHandlers.GetPollDetails)
				polls.POST("/:pollId/vote", pollsHandlers.CastVote)
				polls.POST("/role-escalation", pollsHandlers.CreateRoleEscalationPoll)
				polls.GET("/members", pollsHandlers.GetChamaMembers)
			}

			// Vote routes (using old vote system - working)
			votes := protected.Group("/chamas/:id/votes")
			{
				votes.POST("", api.CreateVote)
				votes.GET("", api.GetChamaVotes)
				votes.GET("/active", api.GetActiveVotes)
				votes.GET("/results", api.GetVoteResults)
				votes.GET("/:voteId", api.GetVoteDetails)
				votes.POST("/:voteId/vote", api.CastVoteOnItem)
				votes.POST("/role-escalation", api.CreateRoleEscalationVote)
			}

			// Disbursement and Account Management routes
			disbursements := protected.Group("/chamas/:id")
			{
				disbursements.GET("/disbursements", disbursementHandlers.GetDisbursementBatches)
				disbursements.POST("/disbursements/:batchId/process", disbursementHandlers.ProcessDisbursementBatch)
				disbursements.POST("/disbursements/:batchId/approve", disbursementHandlers.ApproveDisbursementBatch)
				disbursements.GET("/transparency", disbursementHandlers.GetTransparencyLog)
			}

			// Financial Reports routes
			reports := protected.Group("/chamas/:id")
			{
				reports.GET("/reports", reportsHandlers.GetFinancialReports)
				reports.POST("/reports", reportsHandlers.GenerateFinancialReport)
				reports.GET("/reports/:reportId/download", reportsHandlers.DownloadFinancialReport)
			}

			// Account Management routes
			account := protected.Group("/chamas/:id")
			{
				account.GET("/welfare/eligible-members", accountHandlers.GetEligibleWelfareMembers)
				account.GET("/transparency-feed", accountHandlers.GetTransparencyFeed)
				account.GET("/account-notifications", accountHandlers.GetAccountNotifications)
				account.GET("/validate-security", accountHandlers.ValidateSystemSecurity)
			}

			// Global account routes
			globalAccount := protected.Group("/account")
			{
				globalAccount.POST("/security-events", accountHandlers.LogSecurityEvent)
			}

			// User Search routes
			userSearch := protected.Group("/user-search")
			{
				userSearch.GET("/search", userSearchHandlers.SearchUsers)
				userSearch.GET("/search/advanced", userSearchHandlers.SearchUsersAdvanced)
				userSearch.GET("/:userId/profile", userSearchHandlers.GetUserProfile)
			}

			// Marketplace roles routes
			marketplaceRoles := protected.Group("/marketplace")
			{
				marketplaceRoles.GET("/user-roles/:userId", userSearchHandlers.CheckMarketplaceRoles)
			}

			// Marketplace Delivery Contacts routes
			// deliveryContacts := protected.Group("/marketplace")
			// {
			// 	deliveryContacts.GET("/delivery-contacts", deliveryContactsHandlers.GetDeliveryContacts)
			// 	deliveryContacts.POST("/delivery-contacts", deliveryContactsHandlers.CreateDeliveryContact)
			// 	deliveryContacts.PUT("/delivery-contacts/:contactId", deliveryContactsHandlers.UpdateDeliveryContact)
			// 	deliveryContacts.DELETE("/delivery-contacts/:contactId", deliveryContactsHandlers.DeleteDeliveryContact)

			// 	// Auto-seller detection routes
			// 	deliveryContacts.GET("/products/user/:userId", api.GetUserProducts)
			// 	deliveryContacts.POST("/auto-register-seller", api.AutoRegisterAsSeller)
			// 	deliveryContacts.POST("/auto-register-buyer", api.AutoRegisterAsBuyer)
			// }

			// Contributions routes
			contributions := protected.Group("/contributions")
			{
				contributions.GET("", api.GetContributions)
				contributions.POST("", api.MakeContribution)
				contributions.GET("/:id", api.GetContribution)
				contributions.GET("/chamas/:chamaId/members", api.GetChamaMembersForContributions)                 // For cash contributions
				contributions.GET("/chamas/:chamaId/merry-go-round-amount", api.GetMerryGoRoundContributionAmount) // Get expected merry-go-round amount
			}

			// Meetings routes
			meetings := protected.Group("/meetings")
			{
				meetings.GET("/", api.GetMeetings)
				meetings.GET("/user", api.GetUserMeetings) // Get all meetings for the current user
				meetings.POST("/", api.CreateMeeting)
				meetings.POST("/calendar", api.CreateMeetingWithCalendar) // New meeting with calendar integration
				meetings.GET("/:id", api.GetMeeting)
				meetings.PUT("/:id", api.UpdateMeeting)
				meetings.PATCH("/:id", api.UpdateMeeting) // Support PATCH requests for meeting updates
				meetings.DELETE("/:id", api.DeleteMeeting)
				meetings.POST("/:id/join", api.JoinMeeting)
				meetings.POST("/:id/join-jitsi", api.JoinMeetingWithJitsi)              // New Jitsi Meet join endpoint
				meetings.GET("/:id/preview", api.PreviewMeeting)                        // New meeting preview for chairperson/secretary
				meetings.POST("/:id/start", api.StartMeeting)                           // New start meeting endpoint
				meetings.POST("/:id/end", api.EndMeeting)                               // New end meeting endpoint
				meetings.POST("/:id/attendance", api.MarkAttendance)                    // New attendance marking
				meetings.GET("/:id/attendance", api.GetMeetingAttendance)               // New attendance retrieval
				meetings.POST("/:id/documents", api.UploadMeetingDocument)              // Upload meeting documents
				meetings.GET("/:id/documents", api.GetMeetingDocuments)                 // Get meeting documents
				meetings.DELETE("/:id/documents/:docId", api.DeleteMeetingDocument)     // Delete meeting document
				meetings.POST("/:id/minutes", api.SaveMeetingMinutes)                   // Save meeting minutes/notes
				meetings.PUT("/:id/minutes", api.UpdateMeetingMinutes)                  // Update meeting minutes/notes
				meetings.GET("/:id/minutes", api.GetMeetingMinutes)                     // Get meeting minutes/notes
				meetings.GET("/:id/calendar/add-url", api.GetGoogleCalendarAddEventURL) // Get Google Calendar add-event URL
				meetings.POST("/:id/calendar/create", api.CreateGoogleCalendarEvent)    // Create calendar event with reminders
			}

			// Merry-Go-Round routes
			merryGoRounds := protected.Group("/merry-go-rounds")
			{
				merryGoRounds.GET("/", api.GetMerryGoRounds)
				merryGoRounds.POST("/", api.CreateMerryGoRound)
				merryGoRounds.GET("/:id", api.GetMerryGoRound)
				merryGoRounds.PUT("/:id", api.UpdateMerryGoRound)
				merryGoRounds.DELETE("/:id", api.DeleteMerryGoRound)
				merryGoRounds.POST("/:id/join", api.JoinMerryGoRound)
				merryGoRounds.POST("/:id/check-advance/:chamaId", api.CheckAndAdvanceRound)
				merryGoRounds.GET("/contribution-status/:chamaId", api.CheckUserContributionStatus)
				merryGoRounds.GET("/:id/calendar/add-url", api.GetMerryGoRoundCalendarAddEventURL)
				merryGoRounds.POST("/:id/calendar/create", api.CreateMerryGoRoundCalendarEvent)
			}

			// Welfare routes
			welfare := protected.Group("/welfare")
			{
				welfare.GET("/", api.GetWelfareRequests)
				welfare.POST("/", api.CreateWelfareRequest)
				welfare.GET("/:id", api.GetWelfareRequest)
				welfare.PUT("/:id", api.UpdateWelfareRequest)
				welfare.DELETE("/:id", api.DeleteWelfareRequest)
				welfare.POST("/:id/vote", api.VoteOnWelfareRequest)
				welfare.POST("/contribute", api.ContributeToWelfare)
				welfare.GET("/:id/contributions", api.GetWelfareContributions)
			}

			// Loan routes
			loans := protected.Group("/loans")
			{
				loans.GET("/", api.GetLoanApplications)
				loans.POST("/apply", api.CreateLoanApplication)
				loans.GET("/:id", api.GetLoanApplication)
				loans.PUT("/:id", api.UpdateLoanApplication)
				loans.DELETE("/:id", api.DeleteLoanApplication)
				loans.POST("/:id/approve", api.ApproveLoan)
				loans.POST("/:id/reject", api.RejectLoan)
				loans.POST("/:id/disburse", api.DisburseLoan)
				loans.POST("/:id/guarantor-response", api.RespondToGuarantorRequest)
				loans.GET("/guarantor-requests", api.GetGuarantorRequests)
				loans.POST("/guarantors/:guarantorId/respond", api.RespondToGuarantorRequest)
			}
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure TLS 1.3
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	// Configure server to handle both IPv4 and IPv6
	server := &http.Server{
		Addr:      ":" + port,
		Handler:   router,
		TLSConfig: tlsConfig,
		// Add timeouts for better stability
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("VaultKe API server starting on port %s", port)

	// Check if TLS certificates are available
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	log.Printf("🔍 DEBUG: TLS_CERT_FILE='%s', TLS_KEY_FILE='%s'", certFile, keyFile)
	log.Printf("🔍 DEBUG: ENVIRONMENT='%s'", os.Getenv("ENVIRONMENT"))

	// Graceful shutdown
	go func() {
		var err error
		if certFile != "" && keyFile != "" {
			log.Printf("🔒 Starting server with TLS 1.3")
			log.Printf("Server accessible via:")
			log.Printf("  - https://localhost:%s", port)
			log.Printf("  - https://127.0.0.1:%s", port)
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("🔓 Starting server without TLS (development mode)")
			log.Printf("Server accessible via:")
			log.Printf("  - http://localhost:%s", port)
			log.Printf("  - http://127.0.0.1:%s", port)
			log.Printf("  - http://[::1]:%s", port)
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Stop notification scheduler
	notificationScheduler.Stop()

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server shutdown complete")
}
