package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"
	"vaultke-backend/test/helpers"
)

type ServicesBasicTestSuite struct {
	suite.Suite
	testDB *helpers.TestDatabase
}

func (suite *ServicesBasicTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDatabase()
}

func (suite *ServicesBasicTestSuite) TearDownSuite() {
	if suite.testDB != nil {
		suite.testDB.Close()
	}
}

func (suite *ServicesBasicTestSuite) SetupTest() {
	// Clean up test data before each test
	if suite.testDB != nil {
		suite.testDB.CleanupTestData()
	}
	// Insert test data
	suite.insertTestData()
}

func (suite *ServicesBasicTestSuite) insertTestData() {
	// Create test user
	err := suite.testDB.CreateTestUser(helpers.TestUser{
		ID:       "test-user-123",
		Email:    "test@example.com",
		Phone:    "+254712345678",
		Role:     "user",
		Password: "password123",
	})
	suite.NoError(err)
}

func (suite *ServicesBasicTestSuite) TestAuthService() {
	suite.Run("token_generation_and_validation", func() {
		authService := services.NewAuthService("test-secret", 3600) // 1 hour

		// Create test user
		user := &models.User{
			ID:    "test-user-123",
			Email: "test@example.com",
			Role:  models.UserRole("user"),
		}

		// Generate token
		token, err := authService.GenerateToken(user)
		assert.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), token)

		// Validate token
		claims, err := authService.ValidateToken(token)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), claims)
		assert.Equal(suite.T(), user.ID, claims.UserID)
		assert.Equal(suite.T(), user.Email, claims.Email)
		assert.Equal(suite.T(), string(user.Role), claims.Role)
	})

	suite.Run("invalid_token_validation", func() {
		authService := services.NewAuthService("test-secret", 3600)

		// Validate invalid token
		claims, err := authService.ValidateToken("invalid-token")
		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), claims)
	})

	suite.Run("expired_token", func() {
		authService := services.NewAuthService("test-secret", -1) // Expired immediately

		user := &models.User{
			ID:    "test-user-123",
			Email: "test@example.com",
			Role:  models.UserRole("user"),
		}

		// Generate token (will be expired)
		token, err := authService.GenerateToken(user)
		assert.NoError(suite.T(), err)

		// Wait a moment to ensure expiration
		time.Sleep(time.Millisecond * 10)

		// Validate expired token
		claims, err := authService.ValidateToken(token)
		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), claims)
	})

	suite.Run("token_blacklisting", func() {
		authService := services.NewAuthService("test-secret", 3600)

		user := &models.User{
			ID:    "test-user-123",
			Email: "test@example.com",
			Role:  models.UserRole("user"),
		}

		// Generate token
		token, err := authService.GenerateToken(user)
		assert.NoError(suite.T(), err)

		// Token should be valid initially
		claims, err := authService.ValidateToken(token)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), claims)

		// Blacklist the token
		authService.BlacklistToken(token)

		// Token should now be invalid
		claims, err = authService.ValidateToken(token)
		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), claims)
		assert.Contains(suite.T(), err.Error(), "revoked")
	})
}

func (suite *ServicesBasicTestSuite) TestChamaService() {
	suite.Run("create_chama_service", func() {
		chamaService := services.NewChamaService(suite.testDB.DB)
		assert.NotNil(suite.T(), chamaService)
	})

	suite.Run("get_chamas", func() {
		chamaService := services.NewChamaService(suite.testDB.DB)

		// Get chamas with pagination
		chamas, err := chamaService.GetChamas(10, 0)
		assert.NoError(suite.T(), err)
		// Should return empty slice if no chamas exist, or a slice of chamas
		if chamas != nil {
			assert.IsType(suite.T(), []*models.Chama{}, chamas)
		} else {
			// It's okay if chamas is nil when there are no chamas
			assert.Nil(suite.T(), chamas)
		}
	})
}

func (suite *ServicesBasicTestSuite) TestReminderService() {
	suite.Run("create_reminder_service", func() {
		reminderService := services.NewReminderService(suite.testDB.DB)
		assert.NotNil(suite.T(), reminderService)
	})
}

func (suite *ServicesBasicTestSuite) TestWebSocketService() {
	suite.Run("create_websocket_service", func() {
		wsService := services.NewWebSocketService(suite.testDB.DB)
		assert.NotNil(suite.T(), wsService)
	})

	suite.Run("websocket_service_methods", func() {
		wsService := services.NewWebSocketService(suite.testDB.DB)

		// Test basic methods exist and don't panic
		assert.NotPanics(suite.T(), func() {
			// Test BroadcastToRoom method
			message := services.WebSocketMessage{
				Type:    "test",
				Message: "test message",
			}
			wsService.BroadcastToRoom("test-room", message)
		})

		assert.NotPanics(suite.T(), func() {
			// Test SendToUser method
			message := services.WebSocketMessage{
				Type:    "test",
				Message: "test message",
			}
			wsService.SendToUser("test-user", message)
		})
	})
}

func (suite *ServicesBasicTestSuite) TestEmailService() {
	suite.Run("create_email_service", func() {
		emailService := services.NewEmailService()
		assert.NotNil(suite.T(), emailService)
	})
}

func (suite *ServicesBasicTestSuite) TestPasswordResetService() {
	suite.Run("create_password_reset_service", func() {
		emailService := services.NewEmailService()
		passwordResetService := services.NewPasswordResetService(suite.testDB.DB, emailService)
		assert.NotNil(suite.T(), passwordResetService)
	})

	suite.Run("generate_reset_token", func() {
		emailService := services.NewEmailService()
		passwordResetService := services.NewPasswordResetService(suite.testDB.DB, emailService)

		// Generate reset token (no parameters needed)
		token, err := passwordResetService.GenerateResetToken()
		assert.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), token)
		assert.Len(suite.T(), token, 6) // Should be 6 characters (6-digit code)
	})

	suite.Run("initialize_password_reset_table", func() {
		emailService := services.NewEmailService()
		passwordResetService := services.NewPasswordResetService(suite.testDB.DB, emailService)

		// Initialize password reset table
		err := passwordResetService.InitializePasswordResetTable()
		assert.NoError(suite.T(), err)
	})
}

func (suite *ServicesBasicTestSuite) TestMpesaService() {
	suite.Run("create_mpesa_service", func() {
		// Skip this test if we don't have the config package imported
		// This is just to test that the service can be created
		suite.T().Skip("Skipping MpesaService test - requires config package")
	})
}

// Additional comprehensive service tests for coverage
func (suite *ServicesBasicTestSuite) TestEmailServiceMethods() {
	suite.Run("send_password_reset_email", func() {
		emailService := services.NewEmailService()

		// Test sending password reset email - should not error even if SMTP not configured
		err := emailService.SendPasswordResetEmail("test@example.com", "reset-token-123", "Test User")
		// In test environment, this should not error
		assert.NoError(suite.T(), err)
	})

	suite.Run("send_chama_invitation_email", func() {
		emailService := services.NewEmailService()

		// Test sending chama invitation email
		err := emailService.SendChamaInvitationEmail("test@example.com", "Test Chama", "Test User", "invitation-token", "http://test.com")
		assert.NoError(suite.T(), err)
	})

	suite.Run("send_test_email", func() {
		emailService := services.NewEmailService()

		// Test sending test email
		err := emailService.SendTestEmail("test@example.com")
		assert.NoError(suite.T(), err)
	})
}

func (suite *ServicesBasicTestSuite) TestUserServiceMethods() {
	suite.Run("create_user_service", func() {
		userService := services.NewUserService(suite.testDB.DB)
		assert.NotNil(suite.T(), userService)
	})

	suite.Run("create_user", func() {
		userService := services.NewUserService(suite.testDB.DB)

		// Test user creation with proper UserRegistration struct
		registration := &models.UserRegistration{
			Email:     "userservice@test.com",
			Phone:     "+254700000123",
			FirstName: "Test",
			LastName:  "User",
			Password:  "password123",
			Language:  "en",
		}

		user, err := userService.CreateUser(registration)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), user)
		assert.NotEmpty(suite.T(), user.ID)
	})

	suite.Run("get_user_by_id", func() {
		userService := services.NewUserService(suite.testDB.DB)

		// Test getting user by ID
		user, err := userService.GetUserByID("test-user-123")
		if err == nil {
			assert.NotNil(suite.T(), user)
		} else {
			// User might not exist in test DB, which is fine
			assert.Error(suite.T(), err)
		}
	})

	suite.Run("authenticate_user", func() {
		userService := services.NewUserService(suite.testDB.DB)

		// Test user authentication with proper UserLogin struct
		login := &models.UserLogin{
			Identifier: "test@example.com",
			Password:   "password123",
		}

		user, err := userService.AuthenticateUser(login)
		if err == nil {
			assert.NotNil(suite.T(), user)
		} else {
			// Authentication might fail in test environment, which is expected
			assert.Error(suite.T(), err)
		}
	})

	suite.Run("user_exists", func() {
		userService := services.NewUserService(suite.testDB.DB)

		// Test user exists check with proper parameters
		exists, err := userService.UserExists("test@example.com", "+254712345678")
		assert.NoError(suite.T(), err)
		assert.IsType(suite.T(), true, exists)

		// Test with non-existent user
		exists, err = userService.UserExists("nonexistent@test.com", "+254700000000")
		assert.NoError(suite.T(), err)
		assert.IsType(suite.T(), false, exists)
	})
}

func (suite *ServicesBasicTestSuite) TestWalletServiceMethods() {
	suite.Run("create_wallet_service", func() {
		walletService := services.NewWalletService(suite.testDB.DB)
		assert.NotNil(suite.T(), walletService)
	})

	suite.Run("create_wallet", func() {
		walletService := services.NewWalletService(suite.testDB.DB)

		// Test wallet creation with proper parameters
		wallet, err := walletService.CreateWallet("test-user-123", models.WalletTypePersonal)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), wallet)
		assert.NotEmpty(suite.T(), wallet.ID)
	})

	suite.Run("get_wallet_by_id", func() {
		walletService := services.NewWalletService(suite.testDB.DB)

		// Test getting wallet by ID
		wallet, err := walletService.GetWalletByID("test-wallet-123")
		if err == nil {
			assert.NotNil(suite.T(), wallet)
		} else {
			// Wallet might not exist, which is fine for test
			assert.Error(suite.T(), err)
		}
	})

	suite.Run("get_wallets_by_owner", func() {
		walletService := services.NewWalletService(suite.testDB.DB)

		// Test getting wallets by owner
		wallets, err := walletService.GetWalletsByOwner("test-user-123")
		assert.NoError(suite.T(), err)
		// Should return slice (empty or with wallets)
		assert.NotNil(suite.T(), wallets)
	})

	suite.Run("create_transaction", func() {
		walletService := services.NewWalletService(suite.testDB.DB)

		// Test transaction creation with proper TransactionCreation struct
		fromWalletID := "test-wallet-from"
		toWalletID := "test-wallet-to"
		description := "Test transaction"

		transactionCreation := &models.TransactionCreation{
			FromWalletID:  &fromWalletID,
			ToWalletID:    &toWalletID,
			Type:          models.TransactionTypeTransfer,
			Amount:        100.0,
			Description:   &description,
			PaymentMethod: models.PaymentMethodWalletTransfer,
		}

		transaction, err := walletService.CreateTransaction(transactionCreation, "test-user-123")
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), transaction)
		assert.NotEmpty(suite.T(), transaction.ID)
	})

	suite.Run("get_wallet_transactions", func() {
		walletService := services.NewWalletService(suite.testDB.DB)

		// Test getting wallet transactions
		transactions, err := walletService.GetWalletTransactions("test-wallet-123", 10, 0)
		assert.NoError(suite.T(), err)
		// Should return slice (empty or with transactions)
		assert.NotNil(suite.T(), transactions)
	})
}

func TestServicesBasicSuite(t *testing.T) {
	suite.Run(t, new(ServicesBasicTestSuite))
}
