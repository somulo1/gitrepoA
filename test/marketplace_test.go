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

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

	"vaultke-backend/internal/api"
)

type MarketplaceTestSuite struct {
	suite.Suite
	db     *sql.DB
	router *gin.Engine
}

// setupMarketplaceTestDB creates an in-memory SQLite database for testing
func setupMarketplaceTestDB() *sql.DB {
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
		`CREATE TABLE products (
			id TEXT PRIMARY KEY,
			seller_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			price REAL NOT NULL,
			category TEXT NOT NULL,
			stock_quantity INTEGER NOT NULL DEFAULT 0,
			images TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			county TEXT NOT NULL,
			town TEXT NOT NULL,
			rating REAL DEFAULT 0,
			total_ratings INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (seller_id) REFERENCES users(id)
		)`,
		`CREATE TABLE cart_items (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			product_id TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (product_id) REFERENCES products(id),
			UNIQUE(user_id, product_id)
		)`,
		`CREATE TABLE orders (
			id TEXT PRIMARY KEY,
			buyer_id TEXT NOT NULL,
			seller_id TEXT NOT NULL,
			total_amount REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			delivery_address TEXT NOT NULL,
			delivery_phone TEXT NOT NULL,
			payment_method TEXT NOT NULL,
			payment_status TEXT NOT NULL DEFAULT 'pending',
			delivery_status TEXT NOT NULL DEFAULT 'pending',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (buyer_id) REFERENCES users(id),
			FOREIGN KEY (seller_id) REFERENCES users(id)
		)`,
	}

	for _, table := range createTables {
		_, err := db.Exec(table)
		if err != nil {
			panic(err)
		}
	}

	return db
}

func (suite *MarketplaceTestSuite) SetupSuite() {
	suite.db = setupMarketplaceTestDB()

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Mock authentication middleware
	suite.router.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Set("db", suite.db)
		c.Next()
	})

	// Setup marketplace routes
	apiGroup := suite.router.Group("/api/v1/marketplace")
	{
		apiGroup.GET("/products", api.GetProducts)
		apiGroup.POST("/products", api.CreateProduct)
		apiGroup.GET("/products/:id", api.GetProduct)
		apiGroup.PUT("/products/:id", api.UpdateProduct)
		apiGroup.DELETE("/products/:id", api.DeleteProduct)
		apiGroup.GET("/cart", api.GetCart)
		apiGroup.POST("/cart", api.AddToCart)
		apiGroup.PUT("/cart/:id", api.UpdateCartItem)
		apiGroup.DELETE("/cart/:id", api.RemoveFromCart)
		apiGroup.POST("/cart/clear", api.ClearCart)
		apiGroup.GET("/orders", api.GetOrders)
		apiGroup.POST("/orders", api.CreateOrder)
		apiGroup.GET("/orders/:id", api.GetOrder)
		apiGroup.PUT("/orders/:id", api.UpdateOrder)
		apiGroup.DELETE("/orders/:id", api.CancelOrder)
		apiGroup.GET("/reviews", api.GetReviews)
		apiGroup.POST("/reviews", api.CreateReview)
		apiGroup.GET("/categories", api.GetCategories)
		apiGroup.GET("/search", api.SearchProducts)
	}
}

func (suite *MarketplaceTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
	// Insert test data
	suite.insertTestData()
}

func (suite *MarketplaceTestSuite) TearDownSuite() {
	suite.db.Close()
}

func (suite *MarketplaceTestSuite) cleanupTestData() {
	tables := []string{"order_items", "orders", "cart_items", "product_reviews", "products", "users"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id LIKE 'test-%%'", table))
		suite.NoError(err)
	}
}

func (suite *MarketplaceTestSuite) insertTestData() {
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
			INSERT INTO users (id, email, phone, first_name, last_name, password_hash, role, status, county, town, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, user.email, user.phone, user.firstName, user.lastName, string(hashedPassword), "user", "active", "Nairobi", "Nairobi", time.Now())
		suite.NoError(err)
	}

	// Insert test products
	products := []struct {
		id, name, description, category, sellerId, county, town, status string
		price                                                           float64
		stock                                                           int
		images, tags                                                    string
		rating                                                          float64
		totalRatings                                                    int
	}{
		{"test-product-123", "Test Product 1", "A test product for testing", "electronics", "test-user-456", "Nairobi", "Nairobi", "active", 15000.0, 10, "[]", "[]", 4.5, 20},
		{"test-product-456", "Test Product 2", "Another test product", "clothing", "test-user-789", "Mombasa", "Mombasa", "active", 2500.0, 5, "[]", "[]", 3.8, 15},
		{"test-product-789", "Test Product 3", "Inactive product", "electronics", "test-user-456", "Kisumu", "Kisumu", "inactive", 8000.0, 0, "[]", "[]", 4.0, 10},
		{"test-product-out-of-stock", "Out of Stock Product", "Product with no stock", "electronics", "test-user-456", "Nairobi", "Nairobi", "active", 5000.0, 0, "[]", "[]", 4.2, 8},
	}

	for _, product := range products {
		_, err := suite.db.Exec(`
			INSERT INTO products (id, name, description, category, price, stock, seller_id, county, town, status, images, tags, rating, total_ratings, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, product.id, product.name, product.description, product.category, product.price, product.stock, product.sellerId, product.county, product.town, product.status, product.images, product.tags, product.rating, product.totalRatings, time.Now())
		suite.NoError(err)
	}

	// Insert test cart items
	cartItems := []struct {
		id, userId, productId string
		quantity              int
		price                 float64
	}{
		{"test-cart-item-123", "test-user-123", "test-product-123", 2, 15000.0},
		{"test-cart-item-456", "test-user-123", "test-product-456", 1, 2500.0},
	}

	for _, item := range cartItems {
		_, err := suite.db.Exec(`
			INSERT INTO cart_items (id, user_id, product_id, quantity, price, added_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, item.id, item.userId, item.productId, item.quantity, item.price, time.Now())
		suite.NoError(err)
	}

	// Insert test orders
	orders := []struct {
		id, buyerId, sellerId, status, paymentStatus string
		totalAmount                                  float64
	}{
		{"test-order-123", "test-user-123", "test-user-456", "pending", "pending", 15000.0},
		{"test-order-456", "test-user-123", "test-user-789", "completed", "paid", 2500.0},
	}

	for _, order := range orders {
		_, err := suite.db.Exec(`
			INSERT INTO orders (id, buyer_id, seller_id, total_amount, status, payment_status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, order.id, order.buyerId, order.sellerId, order.totalAmount, order.status, order.paymentStatus, time.Now())
		suite.NoError(err)
	}

	// Insert test order items
	orderItems := []struct {
		id, orderId, productId, name string
		quantity                     int
		price                        float64
	}{
		{"test-order-item-123", "test-order-123", "test-product-123", "Test Product 1", 1, 15000.0},
		{"test-order-item-456", "test-order-456", "test-product-456", "Test Product 2", 1, 2500.0},
	}

	for _, item := range orderItems {
		_, err := suite.db.Exec(`
			INSERT INTO order_items (id, order_id, product_id, name, quantity, price)
			VALUES (?, ?, ?, ?, ?, ?)
		`, item.id, item.orderId, item.productId, item.name, item.quantity, item.price)
		suite.NoError(err)
	}

	// Insert test reviews
	reviews := []struct {
		id, userId, productId, comment string
		rating                         float64
	}{
		{"test-review-123", "test-user-123", "test-product-123", "Great product!", 5.0},
		{"test-review-456", "test-user-789", "test-product-456", "Good quality", 4.0},
	}

	for _, review := range reviews {
		_, err := suite.db.Exec(`
			INSERT INTO product_reviews (id, user_id, product_id, rating, comment, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, review.id, review.userId, review.productId, review.rating, review.comment, time.Now())
		suite.NoError(err)
	}
}

func (suite *MarketplaceTestSuite) TestGetProducts() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful products retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with pagination",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with category filter",
			queryParams:    "?category=electronics",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with price range filter",
			queryParams:    "?minPrice=1000&maxPrice=20000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with location filter",
			queryParams:    "?county=Nairobi&town=Nairobi",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with search query",
			queryParams:    "?q=Test%20Product",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with sorting",
			queryParams:    "?sortBy=price&sortOrder=asc",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "products with rating filter",
			queryParams:    "?minRating=4.0",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/marketplace/products"+tt.queryParams, nil)
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
				products := data["products"].([]interface{})
				assert.GreaterOrEqual(suite.T(), len(products), 0)
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestCreateProduct() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful product creation",
			requestBody: map[string]interface{}{
				"name":        "New Test Product",
				"description": "A new test product for testing",
				"category":    "electronics",
				"price":       12000.0,
				"stock":       15,
				"images":      []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
				"tags":        []string{"electronics", "gadget", "test"},
				"county":      "Nairobi",
				"town":        "Nairobi",
				"minOrder":    1,
				"maxOrder":    10,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "product creation with missing name",
			requestBody: map[string]interface{}{
				"description": "A product without name",
				"category":    "electronics",
				"price":       12000.0,
				"stock":       15,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
		},
		{
			name: "product creation with empty name",
			requestBody: map[string]interface{}{
				"name":        "",
				"description": "A product with empty name",
				"category":    "electronics",
				"price":       12000.0,
				"stock":       15,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
		},
		{
			name: "product creation with missing description",
			requestBody: map[string]interface{}{
				"name":     "Test Product",
				"category": "electronics",
				"price":    12000.0,
				"stock":    15,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Description is required",
		},
		{
			name: "product creation with invalid category",
			requestBody: map[string]interface{}{
				"name":        "Test Product",
				"description": "A test product",
				"category":    "invalid-category",
				"price":       12000.0,
				"stock":       15,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid category",
		},
		{
			name: "product creation with negative price",
			requestBody: map[string]interface{}{
				"name":        "Test Product",
				"description": "A test product",
				"category":    "electronics",
				"price":       -1000.0,
				"stock":       15,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Price must be positive",
		},
		{
			name: "product creation with negative stock",
			requestBody: map[string]interface{}{
				"name":        "Test Product",
				"description": "A test product",
				"category":    "electronics",
				"price":       12000.0,
				"stock":       -5,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Stock must be non-negative",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/marketplace/products", bytes.NewBuffer(jsonBody))
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
				product := data["product"].(map[string]interface{})
				assert.NotNil(suite.T(), product["id"])
				assert.Equal(suite.T(), tt.requestBody["name"], product["name"])
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestGetProduct() {
	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful product retrieval",
			productID:      "test-product-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "product not found",
			productID:      "nonexistent-product",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Product not found",
		},
		{
			name:           "inactive product retrieval",
			productID:      "test-product-789",
			expectedStatus: http.StatusOK, // Should still be viewable
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/marketplace/products/"+tt.productID, nil)
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
				product := data["product"].(map[string]interface{})
				assert.Equal(suite.T(), tt.productID, product["id"])
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestUpdateProduct() {
	tests := []struct {
		name           string
		productID      string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "successful product update by seller",
			productID: "test-product-123",
			requestBody: map[string]interface{}{
				"name":  "Updated Test Product",
				"price": 18000.0,
				"stock": 20,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "update non-existent product",
			productID: "nonexistent-product",
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Product not found",
		},
		{
			name:      "update product by non-seller",
			productID: "test-product-456", // Sold by test-user-789, current user is test-user-123
			requestBody: map[string]interface{}{
				"name": "Unauthorized Update",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only the seller can update this product",
		},
		{
			name:      "update with invalid data",
			productID: "test-product-123",
			requestBody: map[string]interface{}{
				"price": -500.0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Price must be positive",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/marketplace/products/"+tt.productID, bytes.NewBuffer(jsonBody))
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

func (suite *MarketplaceTestSuite) TestDeleteProduct() {
	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful product deletion by seller",
			productID:      "test-product-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete non-existent product",
			productID:      "nonexistent-product",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Product not found",
		},
		{
			name:           "delete product by non-seller",
			productID:      "test-product-456",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Only the seller can delete this product",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/marketplace/products/"+tt.productID, nil)
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

func (suite *MarketplaceTestSuite) TestGetCart() {
	req, _ := http.NewRequest("GET", "/api/v1/marketplace/cart", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(items), 2)

	assert.Contains(suite.T(), data, "totalAmount")
	assert.Contains(suite.T(), data, "totalItems")
}

func (suite *MarketplaceTestSuite) TestAddToCart() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful add to cart",
			requestBody: map[string]interface{}{
				"productId": "test-product-123",
				"quantity":  2,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "add non-existent product to cart",
			requestBody: map[string]interface{}{
				"productId": "nonexistent-product",
				"quantity":  1,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Product not found",
		},
		{
			name: "add with invalid quantity",
			requestBody: map[string]interface{}{
				"productId": "test-product-123",
				"quantity":  0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Quantity must be positive",
		},
		{
			name: "add out of stock product",
			requestBody: map[string]interface{}{
				"productId": "test-product-out-of-stock",
				"quantity":  1,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Product is out of stock",
		},
		{
			name: "add quantity exceeding stock",
			requestBody: map[string]interface{}{
				"productId": "test-product-456",
				"quantity":  10, // Stock is 5
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Insufficient stock",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/marketplace/cart", bytes.NewBuffer(jsonBody))
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
				item := data["item"].(map[string]interface{})
				assert.NotNil(suite.T(), item["id"])
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestUpdateCartItem() {
	tests := []struct {
		name           string
		itemID         string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful cart item update",
			itemID: "test-cart-item-123",
			requestBody: map[string]interface{}{
				"quantity": 3,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "update non-existent cart item",
			itemID: "nonexistent-item",
			requestBody: map[string]interface{}{
				"quantity": 2,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Cart item not found",
		},
		{
			name:   "update with invalid quantity",
			itemID: "test-cart-item-123",
			requestBody: map[string]interface{}{
				"quantity": -1,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Quantity must be positive",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/marketplace/cart/"+tt.itemID, bytes.NewBuffer(jsonBody))
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

func (suite *MarketplaceTestSuite) TestRemoveFromCart() {
	tests := []struct {
		name           string
		itemID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful cart item removal",
			itemID:         "test-cart-item-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "remove non-existent cart item",
			itemID:         "nonexistent-item",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Cart item not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("DELETE", "/api/v1/marketplace/cart/"+tt.itemID, nil)
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

func (suite *MarketplaceTestSuite) TestClearCart() {
	req, _ := http.NewRequest("POST", "/api/v1/marketplace/cart/clear", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))
}

func (suite *MarketplaceTestSuite) TestGetOrders() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful orders retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "orders with pagination",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "orders with status filter",
			queryParams:    "?status=pending",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "orders with date range",
			queryParams:    "?startDate=2024-01-01&endDate=2024-12-31",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/marketplace/orders"+tt.queryParams, nil)
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

func (suite *MarketplaceTestSuite) TestCreateOrder() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful order creation",
			requestBody: map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"productId": "test-product-123",
						"quantity":  1,
					},
				},
				"deliveryAddress": "123 Test Street",
				"deliveryPhone":   "+254712345678",
				"paymentMethod":   "mpesa",
				"notes":           "Test order",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "order creation with empty items",
			requestBody: map[string]interface{}{
				"items":           []map[string]interface{}{},
				"deliveryAddress": "123 Test Street",
				"paymentMethod":   "mpesa",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Order items are required",
		},
		{
			name: "order creation with invalid product",
			requestBody: map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"productId": "nonexistent-product",
						"quantity":  1,
					},
				},
				"deliveryAddress": "123 Test Street",
				"paymentMethod":   "mpesa",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Product not found",
		},
		{
			name: "order creation with missing delivery address",
			requestBody: map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"productId": "test-product-123",
						"quantity":  1,
					},
				},
				"paymentMethod": "mpesa",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Delivery address is required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/marketplace/orders", bytes.NewBuffer(jsonBody))
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
				order := data["order"].(map[string]interface{})
				assert.NotNil(suite.T(), order["id"])
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestGetOrder() {
	tests := []struct {
		name           string
		orderID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful order retrieval",
			orderID:        "test-order-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "order not found",
			orderID:        "nonexistent-order",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Order not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/marketplace/orders/"+tt.orderID, nil)
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
				order := data["order"].(map[string]interface{})
				assert.Equal(suite.T(), tt.orderID, order["id"])
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestCreateReview() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful review creation",
			requestBody: map[string]interface{}{
				"productId": "test-product-456",
				"rating":    4.5,
				"comment":   "Great product, highly recommended!",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "review creation with missing product ID",
			requestBody: map[string]interface{}{
				"rating":  4.5,
				"comment": "Great product!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Product ID is required",
		},
		{
			name: "review creation with invalid rating",
			requestBody: map[string]interface{}{
				"productId": "test-product-456",
				"rating":    6.0, // Rating should be 1-5
				"comment":   "Great product!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Rating must be between 1 and 5",
		},
		{
			name: "review creation for non-existent product",
			requestBody: map[string]interface{}{
				"productId": "nonexistent-product",
				"rating":    4.5,
				"comment":   "Great product!",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Product not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/marketplace/reviews", bytes.NewBuffer(jsonBody))
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
				review := data["review"].(map[string]interface{})
				assert.NotNil(suite.T(), review["id"])
			}
		})
	}
}

func (suite *MarketplaceTestSuite) TestGetReviews() {
	req, _ := http.NewRequest("GET", "/api/v1/marketplace/reviews?productId=test-product-123", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	reviews := data["reviews"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(reviews), 0)
}

func (suite *MarketplaceTestSuite) TestGetCategories() {
	req, _ := http.NewRequest("GET", "/api/v1/marketplace/categories", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	categories := data["categories"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(categories), 1)
}

func (suite *MarketplaceTestSuite) TestSearchProducts() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "successful product search",
			queryParams:    "?q=Test%20Product",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "search with empty query",
			queryParams:    "?q=",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Search query is required",
		},
		{
			name:           "search with filters",
			queryParams:    "?q=Test&category=electronics&minPrice=1000&maxPrice=20000",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", "/api/v1/marketplace/search"+tt.queryParams, nil)
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

func (suite *MarketplaceTestSuite) TestConcurrentProductCreation() {
	// Test concurrent product creation
	done := make(chan bool)
	results := make(chan int, 5)

	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()

			requestBody := map[string]interface{}{
				"name":        fmt.Sprintf("Concurrent Product %d", index),
				"description": fmt.Sprintf("Product created concurrently %d", index),
				"category":    "electronics",
				"price":       float64(10000 + index*1000),
				"stock":       10,
				"county":      "Nairobi",
				"town":        "Nairobi",
			}

			jsonBody, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/marketplace/products", bytes.NewBuffer(jsonBody))
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

	// All should succeed
	var successCount int
	for code := range results {
		if code == http.StatusCreated {
			successCount++
		}
	}

	assert.Equal(suite.T(), 5, successCount)
}

func (suite *MarketplaceTestSuite) TestDatabaseConnectionFailure() {
	// Test with closed database connection
	suite.db.Close()

	req, _ := http.NewRequest("GET", "/api/v1/marketplace/products", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "database")
}

func TestMarketplaceSuite(t *testing.T) {
	suite.Run(t, new(MarketplaceTestSuite))
}
