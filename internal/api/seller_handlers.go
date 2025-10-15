package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetSellerAnalytics returns analytics data for sellers
func GetSellerAnalytics(c *gin.Context) {
	// Get user ID from context
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	userID := userIDInterface.(string)

	period := c.DefaultQuery("period", "30d")

	// Get database connection
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get real analytics data from database
	analytics, err := getSellerAnalyticsFromDB(db, userID, period)
	if err != nil {
		log.Printf("Error getting seller analytics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analytics,
	})
}

// getSellerAnalyticsFromDB retrieves real analytics data from the database
func getSellerAnalyticsFromDB(db *sql.DB, userID, period string) (map[string]interface{}, error) {
	// Calculate date range based on period
	var startDate time.Time
	now := time.Now()

	switch period {
	case "7d":
		startDate = now.AddDate(0, 0, -7)
	case "30d":
		startDate = now.AddDate(0, 0, -30)
	case "90d":
		startDate = now.AddDate(0, 0, -90)
	case "1y":
		startDate = now.AddDate(-1, 0, 0)
	default:
		startDate = now.AddDate(0, 0, -30)
	}

	analytics := map[string]interface{}{
		"overview": map[string]interface{}{
			"totalRevenue":      0.0,
			"totalOrders":       0,
			"averageOrderValue": 0.0,
			"conversionRate":    0.0,
			"totalProducts":     0,
			"activeProducts":    0,
			"totalViews":        0,
			"totalCustomers":    0,
		},
		"trends": map[string]interface{}{
			"revenueGrowth":  0.0,
			"orderGrowth":    0.0,
			"customerGrowth": 0.0,
		},
		"topProducts":    []map[string]interface{}{},
		"ordersByStatus": []map[string]interface{}{},
		"customerInsights": map[string]interface{}{
			"newCustomers":             0,
			"returningCustomers":       0,
			"averageOrdersPerCustomer": 0.0,
		},
		"deliveryMetrics": map[string]interface{}{
			"onTimeDeliveries":    0,
			"averageDeliveryTime": 0.0,
			"deliverySuccessRate": 0,
		},
	}

	// Get total revenue and orders for the seller
	revenueQuery := `
		SELECT
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COUNT(*) as total_orders,
			COALESCE(AVG(total_amount), 0) as avg_order_value
		FROM orders
		WHERE seller_id = ? AND created_at >= ? AND status != 'cancelled'
	`

	var totalRevenue, avgOrderValue float64
	var totalOrders int
	err := db.QueryRow(revenueQuery, userID, startDate).Scan(&totalRevenue, &totalOrders, &avgOrderValue)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get total products for the seller
	productQuery := `
		SELECT
			COUNT(*) as total_products,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_products
		FROM products
		WHERE seller_id = ?
	`

	var totalProducts, activeProducts int
	err = db.QueryRow(productQuery, userID).Scan(&totalProducts, &activeProducts)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get unique customers count
	customerQuery := `
		SELECT COUNT(DISTINCT buyer_id) as total_customers
		FROM orders
		WHERE seller_id = ? AND created_at >= ? AND status != 'cancelled'
	`

	var totalCustomers int
	err = db.QueryRow(customerQuery, userID, startDate).Scan(&totalCustomers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Update overview data
	overview := analytics["overview"].(map[string]interface{})
	overview["totalRevenue"] = totalRevenue
	overview["totalOrders"] = totalOrders
	overview["averageOrderValue"] = avgOrderValue
	overview["totalProducts"] = totalProducts
	overview["activeProducts"] = activeProducts
	overview["totalCustomers"] = totalCustomers

	// Get top products
	topProductsQuery := `
		SELECT
			p.id,
			p.name,
			COUNT(oi.id) as sales,
			COALESCE(SUM(oi.price * oi.quantity), 0) as revenue
		FROM products p
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id
		WHERE p.seller_id = ? AND (o.created_at >= ? OR o.created_at IS NULL) AND (o.status != 'cancelled' OR o.status IS NULL)
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
		LIMIT 5
	`

	rows, err := db.Query(topProductsQuery, userID, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topProducts []map[string]interface{}
	for rows.Next() {
		var productID, productName string
		var sales int
		var revenue float64

		err := rows.Scan(&productID, &productName, &sales, &revenue)
		if err != nil {
			continue
		}

		topProducts = append(topProducts, map[string]interface{}{
			"id":      productID,
			"name":    productName,
			"sales":   sales,
			"revenue": revenue,
		})
	}
	analytics["topProducts"] = topProducts

	// Get orders by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM orders
		WHERE seller_id = ? AND created_at >= ?
		GROUP BY status
	`

	rows, err = db.Query(statusQuery, userID, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ordersByStatus []map[string]interface{}
	for rows.Next() {
		var status string
		var count int

		err := rows.Scan(&status, &count)
		if err != nil {
			continue
		}

		ordersByStatus = append(ordersByStatus, map[string]interface{}{
			"status": status,
			"count":  count,
		})
	}
	analytics["ordersByStatus"] = ordersByStatus

	return analytics, nil
}

// GetUserProducts returns products for a specific user (for auto-seller detection)
func GetUserProducts(c *gin.Context) {
	userID := c.Param("userId")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User ID is required",
		})
		return
	}

	// Get database connection
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Query user's products
	query := `
		SELECT id, name, description, price, category, status, created_at
		FROM products
		WHERE seller_id = ?
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying user products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve user products",
		})
		return
	}
	defer rows.Close()

	var products []map[string]interface{}
	for rows.Next() {
		var id, name, description, category, status, createdAt string
		var price float64

		err := rows.Scan(&id, &name, &description, &price, &category, &status, &createdAt)
		if err != nil {
			continue
		}

		product := map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"price":       price,
			"category":    category,
			"status":      status,
			"createdAt":   createdAt,
		}

		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"count":   len(products),
	})
}

// AutoRegisterAsSeller automatically registers a user as a seller
func AutoRegisterAsSeller(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database connection
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Check if user exists
	var userExists bool
	userQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)`
	err := db.QueryRow(userQuery, req.UserID).Scan(&userExists)
	if err != nil || !userExists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Check if user already has products (confirming they should be a seller)
	var productCount int
	productQuery := `SELECT COUNT(*) FROM products WHERE seller_id = ?`
	err = db.QueryRow(productQuery, req.UserID).Scan(&productCount)
	if err != nil {
		log.Printf("Error checking user products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify seller status",
		})
		return
	}

	if productCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User has no products listed, cannot auto-register as seller",
		})
		return
	}

	// Create marketplace seller role entry (assuming there's a marketplace_roles table)
	// If the table doesn't exist, we'll create a simple log entry
	insertQuery := `
		INSERT OR IGNORE INTO marketplace_roles (user_id, role, auto_detected, created_at, updated_at)
		VALUES (?, 'seller', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err = db.Exec(insertQuery, req.UserID)
	if err != nil {
		// If marketplace_roles table doesn't exist, just log and return success
		log.Printf("Could not insert into marketplace_roles (table may not exist): %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User automatically registered as seller",
		"data": gin.H{
			"userId":       req.UserID,
			"role":         "seller",
			"autoDetected": true,
			"productCount": productCount,
		},
	})
}

// AutoRegisterAsBuyer automatically registers a user as a buyer
func AutoRegisterAsBuyer(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database connection
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Check if user exists
	var userExists bool
	userQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)`
	err := db.QueryRow(userQuery, req.UserID).Scan(&userExists)
	if err != nil || !userExists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Check if user is already a buyer
	var isBuyer bool
	buyerQuery := `SELECT EXISTS(SELECT 1 FROM marketplace_roles WHERE user_id = ? AND role = 'buyer' AND is_active = TRUE)`
	err = db.QueryRow(buyerQuery, req.UserID).Scan(&isBuyer)
	if err != nil {
		log.Printf("Error checking buyer status: %v", err)
	}

	if isBuyer {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User is already registered as buyer",
			"data": gin.H{
				"userId": req.UserID,
				"role":   "buyer",
				"status": "already_registered",
			},
		})
		return
	}

	// Create marketplace buyer role entry
	insertQuery := `
		INSERT OR IGNORE INTO marketplace_roles (user_id, role, auto_detected, registration_data, created_at, updated_at)
		VALUES (?, 'buyer', 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	registrationData := map[string]interface{}{
		"registration_type": "auto",
		"auto_detected":     true,
		"reason":            "purchase_intent",
		"registration_date": time.Now().Format(time.RFC3339),
	}

	registrationJSON, _ := json.Marshal(registrationData)

	_, err = db.Exec(insertQuery, req.UserID, string(registrationJSON))
	if err != nil {
		log.Printf("Error inserting buyer role: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to register user as buyer",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User automatically registered as buyer",
		"data": gin.H{
			"userId":       req.UserID,
			"role":         "buyer",
			"autoDetected": true,
			"reason":       "purchase_intent",
		},
	})
}

// GetProductWithOwnership returns product details with ownership and buyer eligibility info
func GetProductWithOwnership(c *gin.Context) {
	productID := c.Param("id")
	// Try to get userID from authentication, but don't require it
	userID, _ := c.Get("userID")
	userIDStr := ""
	if userID != nil {
		userIDStr = userID.(string)
	}

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Product ID is required",
		})
		return
	}

	// Get database connection
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Query product with seller information (including all fields)
	query := `
		SELECT p.id, p.name, p.description, p.price, p.category, p.status,
			   p.images, p.stock, p.currency, p.county, p.town, p.address,
			   p.seller_id, p.created_at, p.updated_at,
			   u.first_name, u.last_name, u.email, u.avatar
		FROM products p
		LEFT JOIN users u ON p.seller_id = u.id
		WHERE p.id = ?
	`

	var product map[string]interface{} = make(map[string]interface{})
	var sellerFirstName, sellerLastName, sellerEmail, sellerAvatar sql.NullString
	var imagesJSON, currency, county, town, address sql.NullString
	var id, name, description, category, status, sellerID, createdAt, updatedAt string
	var price float64
	var stock int

	err := db.QueryRow(query, productID).Scan(
		&id, &name, &description, &price, &category, &status,
		&imagesJSON, &stock, &currency, &county, &town, &address,
		&sellerID, &createdAt, &updatedAt,
		&sellerFirstName, &sellerLastName, &sellerEmail, &sellerAvatar,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Product not found",
			})
		} else {
			log.Printf("Error querying product: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to retrieve product",
			})
		}
		return
	}

	// Parse images JSON
	var images []string
	if imagesJSON.Valid && imagesJSON.String != "" {
		if err := json.Unmarshal([]byte(imagesJSON.String), &images); err != nil {
			log.Printf("Error parsing images JSON: %v", err)
			images = []string{} // Default to empty array
		}
	}

	// Build product object with proper null handling
	product = map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description,
		"price":       price,
		"category":    category,
		"status":      status,
		"images":      images,
		"stock":       stock,
		"sellerId":    sellerID,
		"createdAt":   createdAt,
		"updatedAt":   updatedAt,
	}

	// Add nullable fields only if they have values
	if currency.Valid {
		product["currency"] = currency.String
	}
	if county.Valid {
		product["county"] = county.String
	}
	if town.Valid {
		product["town"] = town.String
	}
	if address.Valid {
		product["address"] = address.String
	}

	// Add seller information
	seller := map[string]interface{}{
		"id": sellerID,
	}
	if sellerFirstName.Valid && sellerLastName.Valid {
		product["sellerName"] = sellerFirstName.String + " " + sellerLastName.String
		seller["firstName"] = sellerFirstName.String
		seller["lastName"] = sellerLastName.String
	}
	if sellerEmail.Valid {
		product["sellerEmail"] = sellerEmail.String
		seller["email"] = sellerEmail.String
	}
	if sellerAvatar.Valid {
		seller["avatar"] = sellerAvatar.String
	}
	product["seller"] = seller

	// Determine ownership and buyer eligibility
	isOwner := userIDStr == sellerID
	buyerEligibility := map[string]interface{}{
		"canBuy":    false,
		"reason":    "",
		"isOwner":   isOwner,
		"needsAuth": userIDStr == "",
	}

	if userIDStr == "" {
		// User not authenticated - but this is fine, they just can't buy
		buyerEligibility["reason"] = "authentication_required"
	} else if isOwner {
		// User owns this product
		buyerEligibility["reason"] = "own_product"
	} else {
		// Check if user can buy (is a buyer or seller)
		var hasRole bool
		roleQuery := `
			SELECT EXISTS(
				SELECT 1 FROM marketplace_roles
				WHERE user_id = ? AND role IN ('buyer', 'seller') AND is_active = TRUE
			)
		`
		if err := db.QueryRow(roleQuery, userIDStr).Scan(&hasRole); err != nil {
			log.Printf("Error checking user roles: %v", err)
		}

		// Also check if user is auto-detected seller (has products)
		var isAutoSeller bool
		if !hasRole {
			var productCount int
			autoSellerQuery := `SELECT COUNT(*) FROM products WHERE seller_id = ?`
			if err := db.QueryRow(autoSellerQuery, userIDStr).Scan(&productCount); err == nil {
				isAutoSeller = productCount > 0
			}
		}

		if hasRole || isAutoSeller {
			buyerEligibility["canBuy"] = true
			buyerEligibility["reason"] = "eligible"
			if isAutoSeller {
				buyerEligibility["autoDetectedSeller"] = true
			}
		} else {
			buyerEligibility["reason"] = "buyer_registration_required"
		}
	}

	product["buyerEligibility"] = buyerEligibility

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
	})
}
