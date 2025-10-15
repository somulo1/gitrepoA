package api

// import (
// 	"database/sql"
// 	"net/http"
// 	"vaultke-backend/internal/services"

// 	"github.com/gin-gonic/gin"
// )

// func GetCart(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
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

// 	// Get user's cart
// 	cartItems, err := marketplaceService.GetCart(userID.(string))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get cart: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Calculate totals
// 	var totalAmount float64
// 	var totalItems int
// 	for _, item := range cartItems {
// 		totalAmount += item.Price * float64(item.Quantity)
// 		totalItems += item.Quantity
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data": map[string]interface{}{
// 			"items":       cartItems,
// 			"totalItems":  totalItems,
// 			"totalAmount": totalAmount,
// 			"currency":    "KES",
// 		},
// 	})
// }

// func AddToCart(c *gin.Context) {
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
// 	var req struct {
// 		ProductID string `json:"productId" binding:"required"`
// 		Quantity  int    `json:"quantity" binding:"required,min=1"`
// 		Notes     string `json:"notes"`
// 	}

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

// 	// Add to cart
// 	err := marketplaceService.AddToCart(userID.(string), req.ProductID, req.Quantity)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Failed to add to cart: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Product added to cart successfully",
// 	})
// }

// func RemoveFromCart(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get cart item ID from URL parameter
// 	cartItemID := c.Param("id")
// 	if cartItemID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Cart item ID is required",
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

// 	// Remove from cart
// 	err := marketplaceService.RemoveFromCart(userID.(string), cartItemID)
// 	if err != nil {
// 		if err.Error() == "cart item not found" {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Cart item not found",
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to remove from cart: " + err.Error(),
// 			})
// 		}
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Item removed from cart successfully",
// 	})
// }
