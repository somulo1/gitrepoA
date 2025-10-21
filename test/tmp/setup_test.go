package test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"

	"vaultke-backend/internal/api"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() *sql.DB {
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
		`CREATE TABLE meeting_participants (
			id TEXT PRIMARY KEY,
			meeting_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			status TEXT DEFAULT 'invited',
			joined_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (meeting_id) REFERENCES meetings(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
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
		`CREATE TABLE product_reviews (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			product_id TEXT NOT NULL,
			rating REAL NOT NULL,
			comment TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (product_id) REFERENCES products(id)
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
		`CREATE TABLE loan_guarantors (
			id TEXT PRIMARY KEY,
			loan_id TEXT NOT NULL,
			guarantor_id TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			approved_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (loan_id) REFERENCES loans(id),
			FOREIGN KEY (guarantor_id) REFERENCES users(id)
		)`,
		`CREATE TABLE loan_repayments (
			id TEXT PRIMARY KEY,
			loan_id TEXT NOT NULL,
			amount REAL NOT NULL,
			payment_method TEXT NOT NULL,
			reference TEXT,
			status TEXT DEFAULT 'pending',
			paid_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (loan_id) REFERENCES loans(id)
		)`,
		`CREATE TABLE notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'sent',
			related_id TEXT,
			priority INTEGER DEFAULT 1,
			is_read BOOLEAN DEFAULT FALSE,
			scheduled_at DATETIME,
			expires_at DATETIME,
			metadata TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE notification_settings (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			email_notifications BOOLEAN DEFAULT TRUE,
			sms_notifications BOOLEAN DEFAULT TRUE,
			push_notifications BOOLEAN DEFAULT TRUE,
			in_app_notifications BOOLEAN DEFAULT TRUE,
			meeting_reminders BOOLEAN DEFAULT TRUE,
			payment_alerts BOOLEAN DEFAULT TRUE,
			chama_updates BOOLEAN DEFAULT TRUE,
			system_alerts BOOLEAN DEFAULT TRUE,
			quiet_hours_start TEXT DEFAULT '22:00',
			quiet_hours_end TEXT DEFAULT '07:00',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
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
		`CREATE TABLE password_reset_tokens (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE email_verifications (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE phone_verifications (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
	}

	for _, query := range createTables {
		if _, err := db.Exec(query); err != nil {
			panic(fmt.Sprintf("Failed to create table: %s, error: %v", query, err))
		}
	}

	return db
}

// setupTestRouter creates a test router with all API handlers
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware to set userID and db
	router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", setupTestDB())
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
		apiGroup.GET("/marketplace/products", api.GetProducts)
		apiGroup.POST("/marketplace/products", api.CreateProduct)
		apiGroup.GET("/marketplace/products/:id", api.GetProduct)
		apiGroup.PUT("/marketplace/products/:id", api.UpdateProduct)
		apiGroup.DELETE("/marketplace/products/:id", api.DeleteProduct)
		apiGroup.GET("/marketplace/cart", api.GetCart)
		apiGroup.POST("/marketplace/cart", api.AddToCart)
		apiGroup.DELETE("/marketplace/cart/:id", api.RemoveFromCart)
		apiGroup.GET("/marketplace/orders", api.GetOrders)
		apiGroup.POST("/marketplace/orders", api.CreateOrder)
		apiGroup.GET("/marketplace/orders/:id", api.GetOrder)
		apiGroup.PUT("/marketplace/orders/:id", api.UpdateOrder)
		apiGroup.GET("/marketplace/reviews", api.GetReviews)
		apiGroup.POST("/marketplace/reviews", api.CreateReview)

		// Notification routes
		apiGroup.GET("/notifications", api.GetNotifications)

		// Loan routes
		apiGroup.GET("/loan-applications", api.GetLoanApplications)
		apiGroup.POST("/loan-applications", api.CreateLoanApplication)
	}

	return router
}

// Common test assertion helper
func assertJSONResponse(t *testing.T, expectedStatus int, response *testing.T, body []byte) {
	var jsonResponse map[string]interface{}
	err := json.Unmarshal(body, &jsonResponse)
	assert.NoError(t, err)

	if expectedStatus >= 400 {
		assert.False(t, jsonResponse["success"].(bool))
	} else {
		assert.True(t, jsonResponse["success"].(bool))
	}
}
