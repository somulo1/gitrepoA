package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vaultke-backend/internal/api"
	"vaultke-backend/test/helpers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type WalletTestSuite struct {
	suite.Suite
	testDB *helpers.TestDatabase
	db     *sql.DB
	router *gin.Engine
}

func (suite *WalletTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDatabase()
	suite.db = suite.testDB.DB

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup wallet routes
	apiGroup := suite.router.Group("/api/v1")
	{
		apiGroup.GET("/wallets", api.GetWallets)
		apiGroup.GET("/wallets/:id", api.GetWallet)
		apiGroup.GET("/wallets/:id/transactions", api.GetWalletTransactions)
		apiGroup.GET("/wallets/balance", api.GetWalletBalance)
		apiGroup.POST("/wallets/deposit", api.DepositMoney)
		apiGroup.POST("/wallets/transfer", api.TransferMoney)
		apiGroup.POST("/wallets/withdraw", api.WithdrawMoney)
	}
}

func (suite *WalletTestSuite) TearDownSuite() {
	if suite.testDB != nil {
		suite.testDB.Close()
	}
}

func (suite *WalletTestSuite) SetupTest() {
	// Clean up test data before each test
	if suite.testDB != nil {
		suite.testDB.CleanupTestData()
	}
	// Insert test data
	suite.insertTestData()
}

func (suite *WalletTestSuite) insertTestData() {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	// Insert test users
	users := []struct {
		id, email, phone, firstName, lastName string
	}{
		{"test-user-123", "test@example.com", "+254712345678", "Test", "User"},
		{"test-user-456", "jane@example.com", "+254712345679", "Jane", "Smith"},
		{"test-user-789", "bob@example.com", "+254712345680", "Bob", "Johnson"},
	}

	for _, user := range users {
		_, err := suite.db.Exec(`
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, user.email, user.phone, user.firstName, user.lastName, string(hashedPassword), "user", "active", time.Now())
		suite.NoError(err)
	}

	// Insert test chama
	_, err := suite.db.Exec(`
		INSERT INTO chamas (id, name, description, type, county, town, contribution_amount, contribution_frequency, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-chama-123", "Test Chama", "Test Description", "savings", "Nairobi", "Nairobi", 5000.0, "monthly", "test-user-123", time.Now())
	suite.NoError(err)

	// Insert test wallets
	wallets := []struct {
		id, ownerId, walletType  string
		balance                  float64
		isLocked                 bool
		dailyLimit, monthlyLimit float64
	}{
		{"test-wallet-personal-123", "test-user-123", "personal", 50000.0, false, 10000.0, 100000.0},
		{"test-wallet-personal-456", "test-user-456", "personal", 25000.0, false, 5000.0, 50000.0},
		{"test-wallet-chama-123", "test-chama-123", "chama", 100000.0, false, 20000.0, 200000.0},
		{"test-wallet-locked", "test-user-123", "personal", 15000.0, true, 10000.0, 100000.0},
	}

	for _, wallet := range wallets {
		_, err := suite.db.Exec(`
			INSERT INTO wallets (id, owner_id, type, balance, is_locked, daily_limit, monthly_limit, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, wallet.id, wallet.ownerId, wallet.walletType, wallet.balance, wallet.isLocked, wallet.dailyLimit, wallet.monthlyLimit, time.Now())
		suite.NoError(err)
	}

	// Insert test transactions
	transactions := []struct {
		id, fromWalletId, toWalletId, transactionType, status, initiatedBy string
		amount                                                             float64
		description                                                        string
		paymentMethod                                                      string
	}{
		{"test-transaction-1", "test-wallet-personal-123", "test-wallet-personal-456", "transfer", "completed", "test-user-123", 5000.0, "Test transfer", "wallet"},
		{"test-transaction-2", "", "test-wallet-personal-123", "deposit", "completed", "test-user-123", 10000.0, "Test deposit", "mpesa"},
		{"test-transaction-3", "test-wallet-personal-456", "", "withdrawal", "pending", "test-user-456", 2000.0, "Test withdrawal", "bank"},
		{"test-transaction-4", "test-wallet-personal-123", "test-wallet-chama-123", "contribution", "completed", "test-user-123", 5000.0, "Monthly contribution", "wallet"},
	}

	for _, transaction := range transactions {
		_, err := suite.db.Exec(`
			INSERT INTO transactions (id, from_wallet_id, to_wallet_id, type, amount, description, payment_method, status, initiated_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, transaction.id,
			func() interface{} {
				if transaction.fromWalletId == "" {
					return nil
				} else {
					return transaction.fromWalletId
				}
			}(),
			func() interface{} {
				if transaction.toWalletId == "" {
					return nil
				} else {
					return transaction.toWalletId
				}
			}(),
			transaction.transactionType, transaction.amount, transaction.description, transaction.paymentMethod,
			transaction.status, transaction.initiatedBy, time.Now())
		suite.NoError(err)
	}
}

func (suite *WalletTestSuite) TestGetWallets() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful wallets retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wallets with type filter",
			queryParams:    "?type=personal",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wallets with active filter",
			queryParams:    "?isActive=true",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wallets with locked filter",
			queryParams:    "?isLocked=false",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/wallets"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				assert.NotNil(suite.T(), response["data"])

				data := response["data"].(map[string]interface{})
				wallets := data["wallets"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(wallets), 1)
			}
		})
	}
}

func (suite *WalletTestSuite) TestGetWallet() {
	tests := []struct {
		name           string
		walletID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful wallet retrieval",
			walletID:       "test-wallet-personal-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wallet not found",
			walletID:       "nonexistent-wallet",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Wallet not found",
		},
		{
			name:           "access other user's wallet",
			walletID:       "test-wallet-personal-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/wallets/"+tt.walletID, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				wallet := data["wallet"].(map[string]interface{})
				assert.Equal(suite.T(), tt.walletID, wallet["id"])
			}
		})
	}
}

func (suite *WalletTestSuite) TestGetWalletTransactions() {
	tests := []struct {
		name           string
		walletID       string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful transactions retrieval",
			walletID:       "test-wallet-personal-123",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions with pagination",
			walletID:       "test-wallet-personal-123",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions with type filter",
			walletID:       "test-wallet-personal-123",
			queryParams:    "?type=transfer",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions with status filter",
			walletID:       "test-wallet-personal-123",
			queryParams:    "?status=completed",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions with date range",
			walletID:       "test-wallet-personal-123",
			queryParams:    "?startDate=2024-01-01&endDate=2024-12-31",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "transactions of non-existent wallet",
			walletID:       "nonexistent-wallet",
			queryParams:    "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Wallet not found",
		},
		{
			name:           "transactions of other user's wallet",
			walletID:       "test-wallet-personal-456",
			queryParams:    "",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/wallets/"+tt.walletID+"/transactions"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				assert.NotNil(suite.T(), response["data"])
			}
		})
	}
}

func (suite *WalletTestSuite) TestDepositMoney() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful deposit",
			requestBody: map[string]interface{}{
				"amount":        10000.0,
				"paymentMethod": "mpesa",
				"reference":     "MP12345678",
				"description":   "Test deposit via M-Pesa",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "deposit with bank transfer",
			requestBody: map[string]interface{}{
				"amount":        5000.0,
				"paymentMethod": "bank",
				"reference":     "BT87654321",
				"description":   "Test deposit via bank transfer",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "deposit with missing amount",
			requestBody: map[string]interface{}{
				"paymentMethod": "mpesa",
				"reference":     "MP12345678",
				"description":   "Test deposit",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Amount is required",
		},
		{
			name: "deposit with negative amount",
			requestBody: map[string]interface{}{
				"amount":        -1000.0,
				"paymentMethod": "mpesa",
				"reference":     "MP12345678",
				"description":   "Test deposit",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Amount must be positive",
		},
		{
			name: "deposit with zero amount",
			requestBody: map[string]interface{}{
				"amount":        0.0,
				"paymentMethod": "mpesa",
				"reference":     "MP12345678",
				"description":   "Test deposit",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Amount must be positive",
		},
		{
			name: "deposit with missing payment method",
			requestBody: map[string]interface{}{
				"amount":      10000.0,
				"reference":   "MP12345678",
				"description": "Test deposit",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Payment method is required",
		},
		{
			name: "deposit with invalid payment method",
			requestBody: map[string]interface{}{
				"amount":        10000.0,
				"paymentMethod": "invalid-method",
				"reference":     "MP12345678",
				"description":   "Test deposit",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid payment method",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/wallets/deposit", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				transaction := data["transaction"].(map[string]interface{})
				assert.NotNil(suite.T(), transaction["id"])
				assert.Equal(suite.T(), "deposit", transaction["type"])
			}
		})
	}
}

func (suite *WalletTestSuite) TestTransferMoney() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful transfer",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-personal-123",
				"toWalletId":   "test-wallet-personal-456",
				"amount":       5000.0,
				"description":  "Test transfer",
				"reference":    "TXN12345",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "transfer to chama wallet",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-personal-123",
				"toWalletId":   "test-wallet-chama-123",
				"amount":       5000.0,
				"description":  "Monthly contribution",
				"reference":    "CONTR12345",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "transfer with missing from wallet",
			requestBody: map[string]interface{}{
				"toWalletId":  "test-wallet-personal-456",
				"amount":      5000.0,
				"description": "Test transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "From wallet is required",
		},
		{
			name: "transfer with missing to wallet",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-personal-123",
				"amount":       5000.0,
				"description":  "Test transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "To wallet is required",
		},
		{
			name: "transfer with insufficient funds",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-personal-123",
				"toWalletId":   "test-wallet-personal-456",
				"amount":       100000.0, // More than available balance
				"description":  "Large transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Insufficient funds",
		},
		{
			name: "transfer from locked wallet",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-locked",
				"toWalletId":   "test-wallet-personal-456",
				"amount":       1000.0,
				"description":  "Transfer from locked wallet",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Wallet is locked",
		},
		{
			name: "transfer to same wallet",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-personal-123",
				"toWalletId":   "test-wallet-personal-123",
				"amount":       1000.0,
				"description":  "Self transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot transfer to same wallet",
		},
		{
			name: "transfer with nonexistent from wallet",
			requestBody: map[string]interface{}{
				"fromWalletId": "nonexistent-wallet",
				"toWalletId":   "test-wallet-personal-456",
				"amount":       1000.0,
				"description":  "Transfer from nonexistent wallet",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Source wallet not found",
		},
		{
			name: "transfer with nonexistent to wallet",
			requestBody: map[string]interface{}{
				"fromWalletId": "test-wallet-personal-123",
				"toWalletId":   "nonexistent-wallet",
				"amount":       1000.0,
				"description":  "Transfer to nonexistent wallet",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Destination wallet not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/wallets/transfer", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				transaction := data["transaction"].(map[string]interface{})
				assert.NotNil(suite.T(), transaction["id"])
				assert.Equal(suite.T(), "transfer", transaction["type"])
			}
		})
	}
}

func (suite *WalletTestSuite) TestWithdrawMoney() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful withdrawal",
			requestBody: map[string]interface{}{
				"walletId":      "test-wallet-personal-123",
				"amount":        5000.0,
				"paymentMethod": "mpesa",
				"reference":     "WD12345678",
				"description":   "Test withdrawal",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "withdrawal via bank transfer",
			requestBody: map[string]interface{}{
				"walletId":      "test-wallet-personal-123",
				"amount":        10000.0,
				"paymentMethod": "bank",
				"reference":     "BW87654321",
				"description":   "Bank withdrawal",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "withdrawal with missing wallet ID",
			requestBody: map[string]interface{}{
				"amount":        5000.0,
				"paymentMethod": "mpesa",
				"reference":     "WD12345678",
				"description":   "Test withdrawal",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Wallet ID is required",
		},
		{
			name: "withdrawal with insufficient funds",
			requestBody: map[string]interface{}{
				"walletId":      "test-wallet-personal-123",
				"amount":        100000.0,
				"paymentMethod": "mpesa",
				"reference":     "WD12345678",
				"description":   "Large withdrawal",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Insufficient funds",
		},
		{
			name: "withdrawal from locked wallet",
			requestBody: map[string]interface{}{
				"walletId":      "test-wallet-locked",
				"amount":        1000.0,
				"paymentMethod": "mpesa",
				"reference":     "WD12345678",
				"description":   "Withdrawal from locked wallet",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Wallet is locked",
		},
		{
			name: "withdrawal exceeding daily limit",
			requestBody: map[string]interface{}{
				"walletId":      "test-wallet-personal-123",
				"amount":        15000.0, // Exceeds daily limit of 10000
				"paymentMethod": "mpesa",
				"reference":     "WD12345678",
				"description":   "Large withdrawal",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Daily limit exceeded",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/wallets/withdraw", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				transaction := data["transaction"].(map[string]interface{})
				assert.NotNil(suite.T(), transaction["id"])
				assert.Equal(suite.T(), "withdrawal", transaction["type"])
			}
		})
	}
}

func (suite *WalletTestSuite) TestUpdateWallet() {
	tests := []struct {
		name           string
		walletID       string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful wallet update",
			walletID: "test-wallet-personal-123",
			requestBody: map[string]interface{}{
				"dailyLimit":   15000.0,
				"monthlyLimit": 150000.0,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "update nonexistent wallet",
			walletID: "nonexistent-wallet",
			requestBody: map[string]interface{}{
				"dailyLimit": 15000.0,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Wallet not found",
		},
		{
			name:     "update other user's wallet",
			walletID: "test-wallet-personal-456",
			requestBody: map[string]interface{}{
				"dailyLimit": 15000.0,
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:     "update with negative daily limit",
			walletID: "test-wallet-personal-123",
			requestBody: map[string]interface{}{
				"dailyLimit": -1000.0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Daily limit must be positive",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/wallets/"+tt.walletID, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *WalletTestSuite) TestLockWallet() {
	tests := []struct {
		name           string
		walletID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful wallet lock",
			walletID:       "test-wallet-personal-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "lock nonexistent wallet",
			walletID:       "nonexistent-wallet",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Wallet not found",
		},
		{
			name:           "lock other user's wallet",
			walletID:       "test-wallet-personal-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "lock already locked wallet",
			walletID:       "test-wallet-locked",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Wallet is already locked",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("PUT", "/api/v1/wallets/"+tt.walletID+"/lock", nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *WalletTestSuite) TestUnlockWallet() {
	tests := []struct {
		name           string
		walletID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful wallet unlock",
			walletID:       "test-wallet-locked",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unlock nonexistent wallet",
			walletID:       "nonexistent-wallet",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Wallet not found",
		},
		{
			name:           "unlock other user's wallet",
			walletID:       "test-wallet-personal-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name:           "unlock already unlocked wallet",
			walletID:       "test-wallet-personal-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Wallet is already unlocked",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("PUT", "/api/v1/wallets/"+tt.walletID+"/unlock", nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			suite.NoError(err)

			if tt.expectedStatus >= 400 {
				assert.False(suite.T(), response["success"].(bool))
				if tt.expectedError != "" {
					assert.Contains(suite.T(), response["error"].(string), tt.expectedError)
				}
			} else {
				assert.True(suite.T(), response["success"].(bool))
			}
		})
	}
}

func (suite *WalletTestSuite) TestGetWalletBalance() {
	req, _ := http.NewRequest("GET", "/api/v1/wallets/test-wallet-personal-123/balance", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(suite.T(), data, "balance")
	assert.Contains(suite.T(), data, "currency")
	assert.Equal(suite.T(), 50000.0, data["balance"])
}

func (suite *WalletTestSuite) TestGetWalletStatement() {
	req, _ := http.NewRequest("GET", "/api/v1/wallets/test-wallet-personal-123/statement?startDate=2024-01-01&endDate=2024-12-31", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(suite.T(), data, "transactions")
	assert.Contains(suite.T(), data, "summary")
}

func (suite *WalletTestSuite) TestConcurrentTransactions() {
	// Test concurrent transfers from same wallet
	done := make(chan bool)
	results := make(chan int, 5)

	fromWalletID := "test-wallet-personal-123"
	toWalletID := "test-wallet-personal-456"

	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()

			requestBody := map[string]interface{}{
				"fromWalletId": fromWalletID,
				"toWalletId":   toWalletID,
				"amount":       1000.0,
				"description":  fmt.Sprintf("Concurrent transfer %d", index),
			}

			jsonBody, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/wallets/transfer", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)
			results <- w.Code
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
	close(results)

	// Count successes and failures
	var successCount, failureCount int
	for code := range results {
		if code == http.StatusCreated {
			successCount++
		} else {
			failureCount++
		}
	}

	// Some should succeed, some might fail due to insufficient funds
	assert.Greater(suite.T(), successCount, 0)
	assert.GreaterOrEqual(suite.T(), failureCount, 0)
	assert.Equal(suite.T(), 5, successCount+failureCount)
}

func (suite *WalletTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/wallets", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func TestWalletSuite(t *testing.T) {
	suite.Run(t, new(WalletTestSuite))
}
