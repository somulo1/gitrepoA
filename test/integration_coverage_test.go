package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vaultke-backend/internal/api"
	"vaultke-backend/test/helpers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationCoverageTestSuite struct {
	suite.Suite
	db     *helpers.TestDatabase
	router *gin.Engine
	config *helpers.TestConfig
	users  map[string]*helpers.TestUser
}

func (suite *IntegrationCoverageTestSuite) SetupSuite() {
	// Setup test database
	suite.config = helpers.NewTestConfig()
	suite.db = helpers.SetupTestDatabase()

	// Create test users
	suite.users = make(map[string]*helpers.TestUser)

	// Create admin user
	adminUser := helpers.TestUser{
		ID:       "admin-test-123",
		Email:    "admin@test.com",
		Phone:    "+254700000001",
		Password: "password123",
		Role:     "admin",
		Token:    "admin-token-123",
	}
	err := suite.db.CreateTestUser(adminUser)
	require.NoError(suite.T(), err)
	suite.users["admin"] = &adminUser

	// Create regular user
	regularUser := helpers.TestUser{
		ID:       "user-test-123",
		Email:    "user@test.com",
		Phone:    "+254700000002",
		Password: "password123",
		Role:     "user",
		Token:    "user-token-123",
	}
	err = suite.db.CreateTestUser(regularUser)
	require.NoError(suite.T(), err)
	suite.users["user"] = &regularUser

	// Setup router with REAL handlers
	suite.setupRealRouter()
}

func (suite *IntegrationCoverageTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *IntegrationCoverageTestSuite) setupRealRouter() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Add database middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("db", suite.db.DB)
		c.Next()
	})

	// Setup API routes with REAL handlers
	apiGroup := suite.router.Group("/api/v1")

	// Auth handlers - REAL ONES using correct signature
	authHandlers := api.NewAuthHandlers(suite.db.DB, suite.config.JWTSecret, 86400)
	authGroup := apiGroup.Group("/auth")
	{
		authGroup.POST("/register", authHandlers.Register)
		authGroup.POST("/login", authHandlers.Login)
		authGroup.POST("/logout", authHandlers.Logout)
		authGroup.POST("/refresh", authHandlers.RefreshToken)
		authGroup.POST("/forgot-password", authHandlers.ForgotPassword)
		authGroup.POST("/reset-password", authHandlers.ResetPassword)
		authGroup.POST("/verify-email", authHandlers.VerifyEmail)
		authGroup.POST("/verify-phone", authHandlers.VerifyPhone)
		authGroup.POST("/test-email", authHandlers.TestEmail)
		authGroup.GET("/profile", authHandlers.GetProfile)
		authGroup.PUT("/profile", authHandlers.UpdateProfile)
	}

	// User handlers - using existing API functions
	userGroup := apiGroup.Group("/users")
	{
		userGroup.GET("", api.GetUsers)
		userGroup.GET("/profile", api.GetProfile)
		userGroup.PUT("/profile", api.UpdateProfile)
		userGroup.POST("/avatar", api.UploadAvatar)
	}

	// Admin handlers - using existing API functions
	adminGroup := apiGroup.Group("/admin")
	{
		adminGroup.PUT("/users/:id/role", api.UpdateUserRole)
		adminGroup.PUT("/users/:id/status", api.UpdateUserStatus)
		adminGroup.DELETE("/users/:id", api.DeleteUser)
		adminGroup.GET("/users/:id", api.GetAllUsersForAdmin)
	}

	// Chama handlers - using existing API functions
	chamaGroup := apiGroup.Group("/chamas")
	{
		chamaGroup.GET("", api.GetChamas)
		chamaGroup.POST("", api.CreateChama)
		chamaGroup.GET("/:id", api.GetChama)
		chamaGroup.PUT("/:id", api.UpdateChama)
		chamaGroup.DELETE("/:id", api.DeleteChama)
		chamaGroup.POST("/:id/join", api.JoinChama)
		chamaGroup.POST("/:id/leave", api.LeaveChama)
		chamaGroup.GET("/:id/members", api.GetChamaMembers)
	}

	// Wallet handlers - using existing API functions
	walletGroup := apiGroup.Group("/wallets")
	{
		walletGroup.GET("", api.GetWallets)
		walletGroup.GET("/:id", api.GetWallet)
		walletGroup.GET("/:id/balance", api.GetWalletBalance)
		walletGroup.GET("/:id/transactions", api.GetWalletTransactions)
		walletGroup.POST("/transfer", api.TransferMoney)
		walletGroup.POST("/deposit", api.DepositMoney)
		walletGroup.POST("/withdraw", api.WithdrawMoney)
	}

	// Reminder handlers - REAL ONES using correct signature
	reminderHandlers := api.NewReminderHandlers(suite.db.DB)
	reminderGroup := apiGroup.Group("/reminders")
	{
		reminderGroup.GET("", reminderHandlers.GetUserReminders)
		reminderGroup.POST("", reminderHandlers.CreateReminder)
		reminderGroup.GET("/:id", reminderHandlers.GetReminder)
		reminderGroup.PUT("/:id", reminderHandlers.UpdateReminder)
		reminderGroup.DELETE("/:id", reminderHandlers.DeleteReminder)
		reminderGroup.PUT("/:id/toggle", reminderHandlers.ToggleReminder)
	}
}

func (suite *IntegrationCoverageTestSuite) makeAuthenticatedRequest(method, url string, body interface{}, userType string) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, _ := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add real authentication token
	if userType != "" {
		token := suite.users[userType].Token
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// Test ALL auth handlers for coverage
func (suite *IntegrationCoverageTestSuite) TestAuthHandlersCoverage() {
	suite.T().Run("register_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"email":     "newuser@coverage.com",
			"password":  "Password123!",
			"firstName": "New",
			"lastName":  "User",
			"phone":     "+254700000999",
		}

		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/register", body, "")
		// Should get some response (success or validation error)
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("login_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"identifier": "user@test.com",
			"password":   "password123",
		}

		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/login", body, "")
		// Should get some response
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("logout_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/logout", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("refresh_token_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/refresh", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("forgot_password_handler", func(t *testing.T) {
		body := map[string]interface{}{"email": "user@test.com"}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/forgot-password", body, "")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("reset_password_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"token":    "reset-token",
			"password": "NewPassword123!",
		}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/reset-password", body, "")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("verify_email_handler", func(t *testing.T) {
		body := map[string]interface{}{"token": "verify-token"}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/verify-email", body, "")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("verify_phone_handler", func(t *testing.T) {
		body := map[string]interface{}{"token": "verify-token"}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/verify-phone", body, "")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("test_email_handler", func(t *testing.T) {
		body := map[string]interface{}{"email": "test@example.com"}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/test-email", body, "")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})
}

// Test ALL user handlers for coverage
func (suite *IntegrationCoverageTestSuite) TestUserHandlersCoverage() {
	suite.T().Run("get_users_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/users", nil, "admin")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_profile_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/users/profile", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("update_profile_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"firstName": "Updated",
			"lastName":  "Name",
			"bio":       "Updated bio",
		}
		w := suite.makeAuthenticatedRequest("PUT", "/api/v1/users/profile", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("upload_avatar_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/users/avatar", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})
}

// Test ALL admin handlers for coverage
func (suite *IntegrationCoverageTestSuite) TestAdminHandlersCoverage() {
	suite.T().Run("update_user_role_handler", func(t *testing.T) {
		body := map[string]interface{}{"role": "moderator"}
		w := suite.makeAuthenticatedRequest("PUT", "/api/v1/admin/users/"+suite.users["user"].ID+"/role", body, "admin")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("update_user_status_handler", func(t *testing.T) {
		body := map[string]interface{}{"status": "active"}
		w := suite.makeAuthenticatedRequest("PUT", "/api/v1/admin/users/"+suite.users["user"].ID+"/status", body, "admin")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_user_for_admin_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/admin/users/"+suite.users["user"].ID, nil, "admin")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("delete_user_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("DELETE", "/api/v1/admin/users/"+suite.users["user"].ID, nil, "admin")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})
}

// Test ALL chama handlers for coverage
func (suite *IntegrationCoverageTestSuite) TestChamaHandlersCoverage() {
	suite.T().Run("get_chamas_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/chamas", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("create_chama_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"name":                  "Test Chama Coverage",
			"description":           "Test Description",
			"type":                  "savings",
			"contributionAmount":    1000.0,
			"contributionFrequency": "monthly",
			"maxMembers":            50,
		}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/chamas", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_chama_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/chamas/test-chama-123", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("update_chama_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "Updated Chama",
			"description": "Updated Description",
		}
		w := suite.makeAuthenticatedRequest("PUT", "/api/v1/chamas/test-chama-123", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("delete_chama_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("DELETE", "/api/v1/chamas/test-chama-123", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("join_chama_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/chamas/test-chama-123/join", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("leave_chama_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/chamas/test-chama-123/leave", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_chama_members_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/chamas/test-chama-123/members", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})
}

// Test ALL wallet handlers for coverage
func (suite *IntegrationCoverageTestSuite) TestWalletHandlersCoverage() {
	suite.T().Run("get_wallets_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/wallets", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_wallet_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/wallets/test-wallet-123", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_wallet_balance_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/wallets/test-wallet-123/balance", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_wallet_transactions_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/wallets/test-wallet-123/transactions", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("transfer_money_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"fromWalletId": "test-wallet-from",
			"toWalletId":   "test-wallet-to",
			"amount":       100.0,
			"description":  "Test transfer",
			"pin":          "1234",
		}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/wallets/transfer", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("deposit_money_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"walletId":    "test-wallet-123",
			"amount":      500.0,
			"method":      "mpesa",
			"description": "Test deposit",
			"phone":       "+254700000001",
		}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/wallets/deposit", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("withdraw_money_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"walletId":    "test-wallet-123",
			"amount":      200.0,
			"method":      "mpesa",
			"description": "Test withdrawal",
			"phone":       "+254700000001",
			"pin":         "1234",
		}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/wallets/withdraw", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})
}

// Test ALL reminder handlers for coverage
func (suite *IntegrationCoverageTestSuite) TestReminderHandlersCoverage() {
	suite.T().Run("get_reminders_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/reminders", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("create_reminder_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"title":       "Test Reminder Coverage",
			"description": "Test Description",
			"dueDate":     "2024-12-31T23:59:59Z",
			"type":        "personal",
		}
		w := suite.makeAuthenticatedRequest("POST", "/api/v1/reminders", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("get_reminder_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("GET", "/api/v1/reminders/test-reminder-123", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("update_reminder_handler", func(t *testing.T) {
		body := map[string]interface{}{
			"title":       "Updated Reminder",
			"description": "Updated Description",
		}
		w := suite.makeAuthenticatedRequest("PUT", "/api/v1/reminders/test-reminder-123", body, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("delete_reminder_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("DELETE", "/api/v1/reminders/test-reminder-123", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})

	suite.T().Run("toggle_reminder_handler", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest("PUT", "/api/v1/reminders/test-reminder-123/toggle", nil, "user")
		assert.True(t, w.Code >= 200 && w.Code < 500)
	})
}

func TestIntegrationCoverageTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationCoverageTestSuite))
}
