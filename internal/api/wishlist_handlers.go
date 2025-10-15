package api

// import (
// 	"database/sql"
// 	"net/http"
// 	"vaultke-backend/internal/services"

// 	"github.com/gin-gonic/gin"
// )

// // GetWishlist retrieves user's wishlist
// func GetWishlist(c *gin.Context) {
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

// 	// Get wishlist
// 	wishlistItems, err := marketplaceService.GetWishlist(userID.(string))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get wishlist: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    wishlistItems,
// 	})
// }

// // AddToWishlist adds a product to user's wishlist
// func AddToWishlist(c *gin.Context) {
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

// 	// Add to wishlist
// 	err := marketplaceService.AddToWishlist(userID.(string), req.ProductID)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Failed to add to wishlist: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Product added to wishlist successfully",
// 	})
// }

// // RemoveFromWishlist removes a product from user's wishlist
// func RemoveFromWishlist(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get product ID from URL parameter
// 	productID := c.Param("productId")
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

// 	// Remove from wishlist
// 	err := marketplaceService.RemoveFromWishlist(userID.(string), productID)
// 	if err != nil {
// 		if err.Error() == "item not found in wishlist" {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Item not found in wishlist",
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to remove from wishlist: " + err.Error(),
// 			})
// 		}
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Item removed from wishlist successfully",
// 	})
// }
