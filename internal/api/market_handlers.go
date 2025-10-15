package api

// import (
// 	"database/sql"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"vaultke-backend/internal/models"
// 	"vaultke-backend/internal/services"

// 	"github.com/gin-gonic/gin"
// )

// // GetMarketplaceCategories returns available product categories with product counts
// func GetMarketplaceCategories(c *gin.Context) {
// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Create marketplace service
// 	marketplaceService := services.NewMarketplaceService(db.(*sql.DB))

// 	// Get categories with product counts
// 	categories, err := marketplaceService.GetCategoriesWithCounts()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to fetch categories: " + err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    categories,
// 		"count":   len(categories),
// 	})
// }

// // Marketplace handlers
// func GetProducts(c *gin.Context) {
// 	// Get query parameters
// 	limitStr := c.DefaultQuery("limit", "20")
// 	offsetStr := c.DefaultQuery("offset", "0")
// 	category := c.Query("category")
// 	county := c.Query("county")
// 	chamaID := c.Query("chamaId")
// 	search := c.Query("search")

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil || limit <= 0 {
// 		limit = 20
// 	}

// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil || offset < 0 {
// 		offset = 0
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Create marketplace service
// 	marketplaceService := services.NewMarketplaceService(db.(*sql.DB))

// 	// Build filters
// 	filters := make(map[string]interface{})
// 	if category != "" {
// 		filters["category"] = category
// 	}
// 	if county != "" {
// 		filters["county"] = county
// 	}
// 	if chamaID != "" {
// 		filters["chamaId"] = chamaID
// 	}
// 	if search != "" {
// 		filters["search"] = search
// 	}

// 	// Get products
// 	products, err := marketplaceService.GetProducts(filters, limit, offset)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get products: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    products,
// 		"count":   len(products),
// 	})
// }

// func GetAllProducts(c *gin.Context) {
// 	// Get query parameters
// 	limitStr := c.DefaultQuery("limit", "20")
// 	offsetStr := c.DefaultQuery("offset", "0")
// 	category := c.Query("category")
// 	county := c.Query("county")
// 	chamaID := c.Query("chamaId")
// 	sellerID := c.Query("sellerId")
// 	search := c.Query("search")

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil || limit <= 0 {
// 		limit = 20
// 	}

// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil || offset < 0 {
// 		offset = 0
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Create marketplace service
// 	marketplaceService := services.NewMarketplaceService(db.(*sql.DB))

// 	// Build filters
// 	filters := make(map[string]interface{})
// 	if category != "" {
// 		filters["category"] = category
// 	}
// 	if county != "" {
// 		filters["county"] = county
// 	}
// 	if chamaID != "" {
// 		filters["chamaId"] = chamaID
// 	}
// 	if sellerID != "" {
// 		filters["sellerId"] = sellerID
// 	}
// 	if search != "" {
// 		filters["search"] = search
// 	}

// 	// Get all products (including out of stock)
// 	products, err := marketplaceService.GetAllProducts(filters, limit, offset)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get products: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    products,
// 		"count":   len(products),
// 	})
// }

// func CreateProduct(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Parse request body
// 	var req models.ProductCreation
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Invalid request data: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Create marketplace service
// 	marketplaceService := services.NewMarketplaceService(db.(*sql.DB))

// 	// Get chama ID from query parameter (optional)
// 	chamaID := c.Query("chamaId")
// 	var chamaIDPtr *string
// 	if chamaID != "" {
// 		chamaIDPtr = &chamaID
// 	}

// 	// Create the product
// 	product, err := marketplaceService.CreateProduct(&req, userID.(string), chamaIDPtr)
// 	if err != nil {
// 		// Check if it's a validation error
// 		if strings.Contains(err.Error(), "validation error") {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"success": false,
// 				"error":   "Failed to create product: " + err.Error(),
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to create product: " + err.Error(),
// 			})
// 		}
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusCreated, gin.H{
// 		"success": true,
// 		"message": "Product created successfully",
// 		"data":    product,
// 	})
// }

// func GetProduct(c *gin.Context) {
// 	productID := c.Param("id")
// 	if productID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Product ID is required",
// 		})
// 		return
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Create marketplace service
// 	marketplaceService := services.NewMarketplaceService(db.(*sql.DB))

// 	// Get product by ID
// 	product, err := marketplaceService.GetProductByID(productID)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Product not found",
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to get product: " + err.Error(),
// 			})
// 		}
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    product,
// 	})
// }

// func UpdateProduct(c *gin.Context) {
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Update product endpoint - coming soon",
// 	})
// }

// func DeleteProduct(c *gin.Context) {
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delete product endpoint - coming soon",
// 	})
// }
