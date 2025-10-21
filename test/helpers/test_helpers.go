package helpers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"vaultke-backend/internal/api"

	"vaultke-backend/internal/database"
	"vaultke-backend/internal/middleware"
	"vaultke-backend/internal/services"
)

// TestConfig holds test configuration
type TestConfig struct {
	JWTSecret           string
	DatabaseURL         string
	Environment         string
	ServerPort          string
	LiveKitKey          string
	LiveKitSecret       string
	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaPasskey        string
	MpesaShortcode      string
}

// TestUser represents a test user
type TestUser struct {
	ID       string
	Email    string
	Phone    string
	Role     string
	Password string
	Token    string
}

// TestDatabase manages test database setup and teardown
type TestDatabase struct {
	DB *sql.DB
}

// NewTestConfig creates a new test configuration
func NewTestConfig() *TestConfig {
	return &TestConfig{
		JWTSecret:           "test-jwt-secret-key-12345678901234567890",
		DatabaseURL:         ":memory:",
		Environment:         "test",
		ServerPort:          "8080",
		LiveKitKey:          "test-livekit-key",
		LiveKitSecret:       "test-livekit-secret",
		MpesaConsumerKey:    "test-mpesa-consumer-key",
		MpesaConsumerSecret: "test-mpesa-consumer-secret",
		MpesaPasskey:        "test-mpesa-passkey",
		MpesaShortcode:      "123456",
	}
}

// SetupTestDatabase creates an in-memory SQLite database for testing
func SetupTestDatabase() *TestDatabase {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("Failed to open test database: %v", err))
	}

	if err := database.Migrate(db); err != nil {
		panic(fmt.Sprintf("Failed to migrate test database: %v", err))
	}

	return &TestDatabase{DB: db}
}

// Close closes the test database
func (td *TestDatabase) Close() {
	if td.DB != nil {
		td.DB.Close()
	}
}

// CreateTestUser creates a test user in the database
func (td *TestDatabase) CreateTestUser(user TestUser) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, is_email_verified, is_phone_verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'active', true, true)
	`
	_, err = td.DB.Exec(query, user.ID, user.Email, user.Phone, "Test", "User", string(hashedPassword), user.Role)
	return err
}

// CreateTestChama creates a test chama in the database
func (td *TestDatabase) CreateTestChama(chamaID, userID string) error {
	// Create chama
	query := `
		INSERT INTO chamas (id, name, description, type, county, town, contribution_amount, contribution_frequency, created_by, max_members, current_members)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := td.DB.Exec(query, chamaID, "Test Chama", "Test Description", "savings", "Nairobi", "Nairobi", 1000.0, "monthly", userID, 50, 1)
	if err != nil {
		return err
	}

	// Create chama member
	memberQuery := `
		INSERT INTO chama_members (id, chama_id, user_id, role, joined_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	memberID := uuid.New().String()
	_, err = td.DB.Exec(memberQuery, memberID, chamaID, userID, "chairperson", time.Now(), true)
	return err
}

// CreateTestWallet creates a test wallet in the database
func (td *TestDatabase) CreateTestWallet(walletID, ownerID, walletType string, balance float64) error {
	query := `
		INSERT INTO wallets (id, owner_id, type, balance, currency, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := td.DB.Exec(query, walletID, ownerID, walletType, balance, "KES", true)
	return err
}

// CreateTestProduct creates a test product in the database
func (td *TestDatabase) CreateTestProduct(productID, sellerID string) error {
	query := `
		INSERT INTO products (id, name, description, category, price, seller_id, county, town, stock, status, images, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := td.DB.Exec(query, productID, "Test Product", "Test Description", "electronics", 1000.0, sellerID, "Nairobi", "Nairobi", 10, "active", "[]", "[]")
	return err
}

// GenerateJWTToken generates a JWT token for testing
func GenerateJWTToken(userID, role, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"role":   role,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// SetupTestRouter creates a test router with middleware
func SetupTestRouter(db *sql.DB, cfg *TestConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("config", cfg)
		c.Next()
	})

	// Initialize services
	authService := services.NewAuthService(cfg.JWTSecret, 86400) // 24 hours in seconds
	authMiddleware := middleware.NewAuthMiddleware(authService)
	wsService := services.NewWebSocketService(db)

	// Initialize handlers
	authHandlers := api.NewAuthHandlers(db, cfg.JWTSecret, 86400) // 24 hours in seconds
	reminderHandlers := api.NewReminderHandlers(db)
	sharesHandlers := api.NewSharesHandlers(db)
	dividendsHandlers := api.NewDividendsHandlers(db)
	pollsHandlers := api.NewPollsHandlers(db)

	// Add WebSocket service to context
	router.Use(func(c *gin.Context) {
		c.Set("wsService", wsService)
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// API routes
	apiGroup := router.Group("/api/v1")
	{
		// Auth routes
		auth := apiGroup.Group("/auth")
		{
			auth.POST("/register", authHandlers.Register)
			auth.POST("/login", authHandlers.Login)
			auth.POST("/logout", authMiddleware.AuthRequired(), authHandlers.Logout)
			auth.POST("/refresh", authHandlers.RefreshToken)
		}

		// Public routes
		public := apiGroup.Group("/")
		{
			public.GET("/marketplace/products", api.GetProducts)
			public.GET("/marketplace/products/:id", api.GetProduct)
			public.POST("/payments/mpesa/callback", api.HandleMpesaCallback)
		}

		// Protected routes
		protected := apiGroup.Group("/")
		protected.Use(authMiddleware.AuthRequired())
		{
			// Users
			protected.GET("/users", api.GetUsers)
			protected.GET("/users/profile", authHandlers.GetProfile)
			protected.PUT("/users/profile", authHandlers.UpdateProfile)
			protected.POST("/users/avatar", api.UploadAvatar)

			// Chamas
			protected.GET("/chamas", api.GetChamas)
			protected.POST("/chamas", api.CreateChama)
			protected.GET("/chamas/my", api.GetUserChamas)
			protected.GET("/chamas/:id", api.GetChama)
			protected.PUT("/chamas/:id", api.UpdateChama)
			protected.DELETE("/chamas/:id", api.DeleteChama)
			protected.GET("/chamas/:id/members", api.GetChamaMembers)
			protected.POST("/chamas/:id/join", api.JoinChama)
			protected.POST("/chamas/:id/leave", api.LeaveChama)

			// Wallets
			protected.GET("/wallets", api.GetWallets)
			protected.GET("/wallets/:id", api.GetWallet)
			protected.GET("/wallets/balance", api.GetWalletBalance)
			protected.POST("/wallets/deposit", api.DepositMoney)
			protected.POST("/wallets/transfer", api.TransferMoney)
			protected.POST("/wallets/withdraw", api.WithdrawMoney)

			// Marketplace
			protected.POST("/marketplace/products", api.CreateProduct)
			protected.PUT("/marketplace/products/:id", api.UpdateProduct)
			protected.DELETE("/marketplace/products/:id", api.DeleteProduct)
			protected.GET("/marketplace/cart", api.GetCart)
			protected.POST("/marketplace/cart", api.AddToCart)
			protected.DELETE("/marketplace/cart/:id", api.RemoveFromCart)
			protected.GET("/marketplace/orders", api.GetOrders)
			protected.POST("/marketplace/orders", api.CreateOrder)
			protected.GET("/marketplace/orders/:id", api.GetOrder)
			protected.PUT("/marketplace/orders/:id", api.UpdateOrder)

			// Payments
			protected.POST("/payments/mpesa/stk", api.InitiateMpesaSTK)
			protected.GET("/payments/mpesa/status/:checkoutRequestId", api.GetMpesaTransactionStatus)

			// Meetings
			protected.GET("/meetings", api.GetMeetings)
			protected.POST("/meetings", api.CreateMeeting)
			protected.GET("/meetings/:id", api.GetMeeting)
			protected.PUT("/meetings/:id", api.UpdateMeeting)
			protected.DELETE("/meetings/:id", api.DeleteMeeting)
			protected.POST("/meetings/:id/join", api.JoinMeeting)

			// Loans
			protected.GET("/loans", api.GetLoanApplications)
			protected.POST("/loans/apply", api.CreateLoanApplication)
			protected.GET("/loans/:id", api.GetLoanApplication)
			protected.PUT("/loans/:id", api.UpdateLoanApplication)
			protected.DELETE("/loans/:id", api.DeleteLoanApplication)
			protected.POST("/loans/:id/approve", api.ApproveLoan)
			protected.POST("/loans/:id/reject", api.RejectLoan)

			// Reminders
			protected.GET("/reminders", reminderHandlers.GetUserReminders)
			protected.POST("/reminders", reminderHandlers.CreateReminder)
			protected.GET("/reminders/:id", reminderHandlers.GetReminder)
			protected.PUT("/reminders/:id", reminderHandlers.UpdateReminder)
			protected.DELETE("/reminders/:id", reminderHandlers.DeleteReminder)

			// Shares
			protected.GET("/chamas/:id/shares", sharesHandlers.GetChamaShares)
			protected.POST("/chamas/:id/shares", sharesHandlers.CreateShares)
			protected.GET("/chamas/:id/shares/summary", sharesHandlers.GetChamaSharesSummary)

			// Dividends
			protected.GET("/chamas/:id/dividends", dividendsHandlers.GetChamaDividendDeclarations)
			protected.POST("/chamas/:id/dividends", dividendsHandlers.DeclareDividend)

			// Polls
			protected.GET("/chamas/:id/polls", pollsHandlers.GetChamaPolls)
			protected.POST("/chamas/:id/polls", pollsHandlers.CreatePoll)
			protected.POST("/chamas/:id/polls/:pollId/vote", pollsHandlers.CastVote)

			// Notifications
			protected.GET("/notifications", api.GetNotifications)
			protected.PUT("/notifications/:id/read", api.MarkNotificationAsRead)
			protected.POST("/notifications/read-all", api.MarkAllNotificationsAsRead)

			// Chat
			protected.GET("/chat/rooms", api.GetChatRooms)
			protected.POST("/chat/rooms", api.CreateChatRoom)
			protected.GET("/chat/rooms/:id", api.GetChatRoom)
			protected.GET("/chat/rooms/:id/messages", api.GetChatMessages)
			protected.POST("/chat/rooms/:id/messages", api.SendMessage)

			// Welfare
			protected.GET("/welfare", api.GetWelfareRequests)
			protected.POST("/welfare", api.CreateWelfareRequest)
			protected.GET("/welfare/:id", api.GetWelfareRequest)
			protected.PUT("/welfare/:id", api.UpdateWelfareRequest)
			protected.POST("/welfare/:id/vote", api.VoteOnWelfareRequest)

			// Contributions
			protected.GET("/contributions", api.GetContributions)
			protected.POST("/contributions", api.MakeContribution)
			protected.GET("/contributions/:id", api.GetContribution)
		}
	}

	return router
}

// MakeRequest makes an HTTP request to the test server
func MakeRequest(router *gin.Engine, method, url string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, _ := http.NewRequest(method, url, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// AssertSuccessResponse asserts that the response is successful
func AssertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	assert.Equal(t, expectedStatus, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	return response
}

// AssertErrorResponse asserts that the response is an error
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	assert.Equal(t, expectedStatus, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	return response
}

// GetTestDataPath returns the path to test data files
func GetTestDataPath(filename string) string {
	return filepath.Join("testdata", filename)
}

// CreateTestFile creates a test file with given content
func CreateTestFile(t *testing.T, filename, content string) string {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
	return filePath
}

// MockWebSocketService creates a mock WebSocket service for testing
func MockWebSocketService(db *sql.DB) *services.WebSocketService {
	return services.NewWebSocketService(db)
}

// MockAuthService creates a mock auth service for testing
func MockAuthService() *services.AuthService {
	return services.NewAuthService("test-secret", 86400) // 24 hours in seconds
}

// CleanupTestData removes test data from database
func (td *TestDatabase) CleanupTestData() {
	tables := []string{
		"transactions", "cart_items", "order_items", "orders", "products",
		"chama_members", "chamas", "wallets", "users", "reminders", "meetings",
		"loans", "notifications", "chat_rooms", "chat_messages", "welfare_requests",
		"contributions", "shares", "dividends", "polls", "votes",
	}

	for _, table := range tables {
		td.DB.Exec("DELETE FROM " + table)
	}
}

// SetupTestEnvironment sets up the complete test environment
func SetupTestEnvironment(t *testing.T) (*TestDatabase, *gin.Engine, *TestConfig, func()) {
	cfg := NewTestConfig()
	db := SetupTestDatabase()
	router := SetupTestRouter(db.DB, cfg)

	cleanup := func() {
		db.Close()
	}

	return db, router, cfg, cleanup
}

// MockRequest represents a mock HTTP request for testing
type MockRequest struct {
	Method  string
	URL     string
	Body    interface{}
	Headers map[string]string
}

// ExecuteRequest executes a mock request and returns the response
func (mr MockRequest) Execute(router *gin.Engine) *httptest.ResponseRecorder {
	return MakeRequest(router, mr.Method, mr.URL, mr.Body, mr.Headers)
}

// TestSuite represents a test suite with common setup
type TestSuite struct {
	DB     *TestDatabase
	Router *gin.Engine
	Config *TestConfig
	Users  map[string]TestUser
}

// NewTestSuite creates a new test suite
func NewTestSuite(t *testing.T) *TestSuite {
	db, router, cfg, _ := SetupTestEnvironment(t)

	// Create test users
	users := map[string]TestUser{
		"admin": {
			ID:       "admin-user",
			Email:    "admin@example.com",
			Phone:    "+254700000001",
			Role:     "admin",
			Password: "password123",
		},
		"user": {
			ID:       "regular-user",
			Email:    "user@example.com",
			Phone:    "+254700000002",
			Role:     "user",
			Password: "password123",
		},
		"chairperson": {
			ID:       "chairperson-user",
			Email:    "chairperson@example.com",
			Phone:    "+254700000003",
			Role:     "user",
			Password: "password123",
		},
	}

	// Create users in database and generate tokens
	for key, user := range users {
		err := db.CreateTestUser(user)
		assert.NoError(t, err)

		token, err := GenerateJWTToken(user.ID, user.Role, cfg.JWTSecret)
		assert.NoError(t, err)
		user.Token = token
		users[key] = user
	}

	return &TestSuite{
		DB:     db,
		Router: router,
		Config: cfg,
		Users:  users,
	}
}

// Cleanup cleans up the test suite
func (ts *TestSuite) Cleanup() {
	ts.DB.CleanupTestData()
	ts.DB.Close()
}

// GetAuthHeaders returns authorization headers for a user
func (ts *TestSuite) GetAuthHeaders(userType string) map[string]string {
	user, exists := ts.Users[userType]
	if !exists {
		return nil
	}

	return map[string]string{
		"Authorization": "Bearer " + user.Token,
	}
}

// CreateTestChama creates a test chama for the test suite
func (ts *TestSuite) CreateTestChama(chamaID string) error {
	return ts.DB.CreateTestChama(chamaID, ts.Users["chairperson"].ID)
}

// CreateTestWallet creates a test wallet for the test suite
func (ts *TestSuite) CreateTestWallet(walletID, ownerID, walletType string, balance float64) error {
	return ts.DB.CreateTestWallet(walletID, ownerID, walletType, balance)
}

// CreateTestProduct creates a test product for the test suite
func (ts *TestSuite) CreateTestProduct(productID string) error {
	return ts.DB.CreateTestProduct(productID, ts.Users["user"].ID)
}
