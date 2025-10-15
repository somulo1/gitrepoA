package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vaultke-backend/internal/api"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

// setupAPITestDB creates an in-memory SQLite database for testing
func setupAPITestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Create test tables
	createTables := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			phone TEXT UNIQUE NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			avatar TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			status TEXT NOT NULL DEFAULT 'pending',
			is_email_verified BOOLEAN DEFAULT FALSE,
			is_phone_verified BOOLEAN DEFAULT FALSE,
			language TEXT DEFAULT 'en',
			theme TEXT DEFAULT 'dark',
			county TEXT,
			town TEXT,
			latitude REAL,
			longitude REAL,
			business_type TEXT,
			business_description TEXT,
			rating REAL DEFAULT 0,
			total_ratings INTEGER DEFAULT 0,
			bio TEXT,
			occupation TEXT,
			date_of_birth DATE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE chamas (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			avatar TEXT,
			county TEXT NOT NULL,
			town TEXT NOT NULL,
			latitude REAL,
			longitude REAL,
			contribution_amount REAL NOT NULL,
			contribution_frequency TEXT NOT NULL,
			max_members INTEGER,
			current_members INTEGER DEFAULT 1,
			total_funds REAL DEFAULT 0,
			is_public BOOLEAN DEFAULT TRUE,
			requires_approval BOOLEAN DEFAULT FALSE,
			rules TEXT DEFAULT '[]',
			meeting_frequency TEXT,
			meeting_day_of_week INTEGER,
			meeting_day_of_month INTEGER,
			meeting_time TEXT,
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE chama_members (
			id TEXT PRIMARY KEY,
			chama_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			total_contributions REAL DEFAULT 0,
			last_contribution DATETIME,
			rating REAL DEFAULT 0,
			total_ratings INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chama_id) REFERENCES chamas(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(chama_id, user_id)
		)`,
		`CREATE TABLE wallets (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			balance REAL DEFAULT 0,
			currency TEXT DEFAULT 'KES',
			is_active BOOLEAN DEFAULT TRUE,
			is_locked BOOLEAN DEFAULT FALSE,
			daily_limit REAL,
			monthly_limit REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE transactions (
			id TEXT PRIMARY KEY,
			from_wallet_id TEXT,
			to_wallet_id TEXT,
			type TEXT NOT NULL,
			amount REAL NOT NULL,
			currency TEXT DEFAULT 'KES',
			description TEXT,
			reference TEXT,
			payment_method TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			initiated_by TEXT NOT NULL,
			recipient_id TEXT,
			metadata TEXT,
			fees REAL DEFAULT 0,
			approved_by TEXT,
			requires_approval BOOLEAN DEFAULT FALSE,
			approval_deadline DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_wallet_id) REFERENCES wallets(id),
			FOREIGN KEY (to_wallet_id) REFERENCES wallets(id),
			FOREIGN KEY (initiated_by) REFERENCES users(id),
			FOREIGN KEY (approved_by) REFERENCES users(id)
		)`,

		`CREATE TABLE meetings (
			id TEXT PRIMARY KEY,
			chama_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			scheduled_at DATETIME NOT NULL,
			duration INTEGER DEFAULT 60,
			location TEXT,
			meeting_url TEXT,
			status TEXT DEFAULT 'scheduled',
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE products (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			category TEXT NOT NULL,
			price REAL NOT NULL,
			currency TEXT DEFAULT 'KES',
			images TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			stock INTEGER DEFAULT 0,
			min_order INTEGER DEFAULT 1,
			max_order INTEGER,
			seller_id TEXT NOT NULL,
			chama_id TEXT,
			county TEXT NOT NULL,
			town TEXT NOT NULL,
			address TEXT,
			tags TEXT,
			rating REAL DEFAULT 0,
			total_ratings INTEGER DEFAULT 0,
			total_sales INTEGER DEFAULT 0,
			is_promoted BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (seller_id) REFERENCES users(id),
			FOREIGN KEY (chama_id) REFERENCES chamas(id)
		)`,
		`CREATE TABLE cart_items (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			product_id TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			price REAL NOT NULL,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE orders (
			id TEXT PRIMARY KEY,
			buyer_id TEXT NOT NULL,
			seller_id TEXT NOT NULL,
			chama_id TEXT,
			total_amount REAL NOT NULL,
			currency TEXT DEFAULT 'KES',
			status TEXT DEFAULT 'pending',
			payment_method TEXT,
			payment_status TEXT DEFAULT 'pending',
			delivery_county TEXT,
			delivery_town TEXT,
			delivery_address TEXT,
			delivery_phone TEXT,
			delivery_fee REAL DEFAULT 0,
			delivery_status TEXT DEFAULT 'pending',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE order_items (
			id TEXT PRIMARY KEY,
			order_id TEXT NOT NULL,
			product_id TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			price REAL NOT NULL,
			name TEXT NOT NULL
		)`,

		`CREATE TABLE loans (
			id TEXT PRIMARY KEY,
			borrower_id TEXT NOT NULL,
			chama_id TEXT NOT NULL,
			type TEXT NOT NULL,
			amount REAL NOT NULL,
			interest_rate REAL DEFAULT 0,
			duration INTEGER NOT NULL,
			purpose TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			approved_by TEXT,
			approved_at DATETIME,
			disbursed_at DATETIME,
			due_date DATETIME,
			total_amount REAL DEFAULT 0,
			paid_amount REAL DEFAULT 0,
			remaining_amount REAL DEFAULT 0,
			required_guarantors INTEGER NOT NULL,
			approved_guarantors INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (borrower_id) REFERENCES users(id),
			FOREIGN KEY (chama_id) REFERENCES chamas(id),
			FOREIGN KEY (approved_by) REFERENCES users(id)
		)`,
		`CREATE TABLE reminders (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			reminder_type TEXT NOT NULL DEFAULT 'once',
			scheduled_at DATETIME NOT NULL,
			is_enabled BOOLEAN DEFAULT TRUE,
			is_completed BOOLEAN DEFAULT FALSE,
			notification_sent BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, query := range createTables {
		if _, err := db.Exec(query); err != nil {
			panic(err)
		}
	}

	// Insert test data
	testData := []string{
		`INSERT OR IGNORE INTO users (id, email, first_name, last_name, phone, password_hash, role, status) VALUES
			('test-user-123', 'test@example.com', 'Test', 'User', '+1234567890', 'hashed_password', 'user', 'active'),
			('user-2', 'jane@example.com', 'Jane', 'Smith', '+1234567891', 'hashed_password', 'user', 'active'),
			('user-3', 'bob@example.com', 'Bob', 'Johnson', '+1234567892', 'hashed_password', 'user', 'active')`,
		`INSERT OR IGNORE INTO chamas (id, name, description, type, county, town, contribution_amount, contribution_frequency, created_by, rules) VALUES
			('test-chama', 'Test Chama', 'A test chama', 'investment', 'Nairobi', 'Nairobi', 5000.0, 'monthly', 'user-2', '[]')`,
		`INSERT OR IGNORE INTO chama_members (id, chama_id, user_id, role, is_active) VALUES
			('member-1', 'test-chama', 'test-user-123', 'member', TRUE),
			('member-2', 'test-chama', 'user-2', 'chairperson', TRUE),
			('member-3', 'test-chama', 'user-3', 'secretary', TRUE)`,
		`INSERT OR IGNORE INTO wallets (id, owner_id, type, balance) VALUES
			('wallet-personal-test-user-123', 'test-user-123', 'personal', 10000.0),
			('wallet-personal-user-2', 'user-2', 'personal', 15000.0),
			('wallet-chama-test-chama', 'test-chama', 'chama', 50000.0)`,
		`INSERT OR IGNORE INTO products (id, name, description, category, price, seller_id, county, town, stock, images, tags) VALUES
			('product-1', 'Test Product 1', 'A test product for testing', 'electronics', 1000.0, 'user-2', 'Nairobi', 'Nairobi', 10, '[]', '[]'),
			('product-2', 'Test Product 2', 'Another test product', 'clothing', 500.0, 'user-3', 'Mombasa', 'Mombasa', 5, '[]', '[]')`,
	}

	for _, query := range testData {
		if _, err := db.Exec(query); err != nil {
			panic(err)
		}
	}

	return db
}

// Global test database instance
var testDB *sql.DB

// resetTestDB resets the global test database
func resetTestDB() {
	if testDB != nil {
		testDB.Close()
		testDB = nil
	}
}

// setupTestRouter creates a test router with all API handlers
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create test database only once
	if testDB == nil {
		testDB = setupAPITestDB()
	}

	// Mock middleware to set userID and db
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", testDB)
		c.Next()
	})

	// Setup all API routes
	apiGroup := router.Group("/api/v1")
	{
		// User routes
		apiGroup.GET("/users", api.GetUsers)
		apiGroup.GET("/profile", api.GetProfile)
		apiGroup.PUT("/profile", api.UpdateProfile)
		apiGroup.POST("/upload-avatar", api.UploadAvatar)

		// Chama routes
		apiGroup.GET("/chamas", api.GetChamas)
		apiGroup.GET("/chamas/my", api.GetUserChamas)
		apiGroup.POST("/chamas", api.CreateChama)
		apiGroup.GET("/chamas/:id", api.GetChama)
		apiGroup.PUT("/chamas/:id", api.UpdateChama)
		apiGroup.DELETE("/chamas/:id", api.DeleteChama)
		apiGroup.GET("/chamas/:id/members", api.GetChamaMembers)
		apiGroup.POST("/chamas/:id/join", api.JoinChama)
		apiGroup.POST("/chamas/:id/leave", api.LeaveChama)
		apiGroup.GET("/chamas/:id/transactions", api.GetChamaTransactions)

		// Wallet routes
		apiGroup.GET("/wallets", api.GetWallets)
		apiGroup.GET("/wallets/:id", api.GetWallet)
		apiGroup.GET("/wallets/:id/transactions", api.GetWalletTransactions)
		apiGroup.POST("/wallets/deposit", api.DepositMoney)
		apiGroup.POST("/wallets/transfer", api.TransferMoney)
		apiGroup.POST("/wallets/withdraw", api.WithdrawMoney)

		// Contribution routes
		apiGroup.GET("/contributions", api.GetContributions)
		apiGroup.POST("/contributions", api.MakeContribution)
		apiGroup.GET("/contributions/:id", api.GetContribution)

		// Meeting routes
		apiGroup.GET("/meetings", api.GetMeetings)
		apiGroup.POST("/meetings", api.CreateMeeting)
		apiGroup.GET("/meetings/:id", api.GetMeeting)
		apiGroup.PUT("/meetings/:id", api.UpdateMeeting)
		apiGroup.DELETE("/meetings/:id", api.DeleteMeeting)
		apiGroup.POST("/meetings/:id/join", api.JoinMeeting)

		// Marketplace routes
		// apiGroup.GET("/marketplace/products", api.GetProducts)
		// apiGroup.POST("/marketplace/products", api.CreateProduct)
		// apiGroup.GET("/marketplace/products/:id", api.GetProduct)
		// apiGroup.PUT("/marketplace/products/:id", api.UpdateProduct)
		// apiGroup.DELETE("/marketplace/products/:id", api.DeleteProduct)
		// apiGroup.GET("/marketplace/cart", api.GetCart)
		// apiGroup.POST("/marketplace/cart", api.AddToCart)
		// apiGroup.DELETE("/marketplace/cart/:id", api.RemoveFromCart)
		// apiGroup.GET("/marketplace/orders", api.GetOrders)
		// apiGroup.POST("/marketplace/orders", api.CreateOrder)
		// apiGroup.GET("/marketplace/orders/:id", api.GetOrder)
		// apiGroup.PUT("/marketplace/orders/:id", api.UpdateOrder)
		// apiGroup.GET("/marketplace/reviews", api.GetReviews)
		// apiGroup.POST("/marketplace/reviews", api.CreateReview)

		// Payment routes
		apiGroup.POST("/payments/mpesa/stk", api.InitiateMpesaSTK)
		apiGroup.POST("/payments/mpesa/callback", api.HandleMpesaCallback)
		apiGroup.POST("/payments/bank-transfer", api.InitiateBankTransfer)
	}

	return router
}

func TestGetProfile(t *testing.T) {
	resetTestDB() // Reset database to pick up schema changes
	router := setupTestRouter()

	t.Run("get profile endpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.Contains(t, response["message"], "coming soon")
	})
}

func TestUpdateProfile(t *testing.T) {
	router := setupTestRouter()

	t.Run("update profile endpoint", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/profile", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.Contains(t, response["message"], "coming soon")
	})
}

func TestGetWallets(t *testing.T) {
	router := setupTestRouter()

	t.Run("get wallets endpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/wallets", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.Contains(t, response["message"], "coming soon")
	})
}

func TestGetWallet(t *testing.T) {
	router := setupTestRouter()

	t.Run("get wallet endpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/wallets/test-wallet", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.Contains(t, response["message"], "coming soon")
	})
}

func TestDepositMoney(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful money deposit", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"amount":        1000.0,
			"paymentMethod": "mpesa",
			"reference":     "TEST123",
			"description":   "Test deposit",
		}
		jsonBody, _ := json.Marshal(reqBody)

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

	t.Run("deposit with invalid amount", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"amount": -100.0, // Invalid negative amount
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/wallets/deposit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetContributions(t *testing.T) {
	router := setupTestRouter()

	t.Run("contributions retrieval without chama ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/contributions", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("contributions retrieval with chama ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/contributions?chamaId=test-chama", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMakeContribution(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful contribution", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"chamaId":       "test-chama",
			"amount":        5000.0,
			"description":   "Monthly contribution",
			"type":          "regular",
			"paymentMethod": "wallet",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/contributions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Expected 201, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestGetMeetings(t *testing.T) {
	router := setupTestRouter()

	t.Run("meetings retrieval without chama ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/meetings", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("meetings retrieval with chama ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/meetings?chamaId=test-chama", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetProducts(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful products retrieval", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/marketplace/products", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Expected 200, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("products retrieval with filters", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/marketplace/products?category=electronics&county=Nairobi", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestGetUsers(t *testing.T) {
	resetTestDB() // Reset database to pick up schema changes
	router := setupTestRouter()

	t.Run("successful users retrieval", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("users retrieval with pagination", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users?limit=10&offset=0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetChamas(t *testing.T) {
	resetTestDB() // Reset database to pick up schema changes
	router := setupTestRouter()

	t.Run("successful chamas retrieval", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Expected 200, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("chamas retrieval with filters", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas?county=Nairobi&type=investment", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetUserChamas(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful user chamas retrieval", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas/my", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestCreateChama(t *testing.T) {
	resetTestDB() // Reset database to pick up schema changes
	router := setupTestRouter()

	t.Run("successful chama creation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":                   "Test Chama New",
			"description":            "A new test chama",
			"type":                   "savings",
			"county":                 "Nairobi",
			"town":                   "Westlands",
			"contribution_amount":    5000.0,
			"contribution_frequency": "monthly",
			"max_members":            50,
			"is_public":              true,
			"requires_approval":      false,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/chamas", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Expected 201, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("chama creation with invalid data", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Te", // Too short
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/chamas", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetChama(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful chama retrieval", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas/test-chama", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("chama retrieval with invalid ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas/nonexistent-chama", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// This should return 404 since the chama doesn't exist
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestGetChamaMembers(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful chama members retrieval", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/chamas/test-chama/members", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Expected 200, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestCreateProduct(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful product creation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":        "Test Product New",
			"description": "A new test product",
			"category":    "electronics",
			"price":       1500.0,
			"images":      []string{"https://example.com/image1.jpg"},
			"stock":       20,
			"minOrder":    1,
			"county":      "Nairobi",
			"town":        "Nairobi",
			"tags":        []string{"electronics", "test"},
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/marketplace/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Expected 201, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("product creation with invalid data", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "", // Empty name
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/marketplace/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400, got %d. Response body: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCreateMeeting(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful meeting creation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"chamaId":     "test-chama",
			"title":       "Monthly Meeting",
			"description": "Regular monthly meeting",
			"scheduledAt": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"duration":    60,
			"location":    "Nairobi Office",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/meetings", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("meeting creation with invalid data", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"title": "", // Empty title
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/meetings", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
