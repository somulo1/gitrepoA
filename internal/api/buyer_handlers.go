package api

// import (
// 	"database/sql"
// 	"log"
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// // GetBuyerStats returns statistics for buyers
// func GetBuyerStats(c *gin.Context) {
// 	// Get user ID from context
// 	userIDInterface, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}
// 	userID := userIDInterface.(string)

// 	// Get database connection
// 	dbInterface, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}
// 	db := dbInterface.(*sql.DB)

// 	// Get real buyer stats from database
// 	stats, err := getBuyerStatsFromDB(db, userID)
// 	if err != nil {
// 		log.Printf("Error getting buyer stats: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get buyer stats"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    stats,
// 	})
// }

// // getBuyerStatsFromDB retrieves real buyer statistics from the database
// func getBuyerStatsFromDB(db *sql.DB, userID string) (map[string]interface{}, error) {
// 	stats := map[string]interface{}{
// 		"totalOrders":        0,
// 		"totalSpent":         0.0,
// 		"pendingOrders":      0,
// 		"deliveredOrders":    0,
// 		"cancelledOrders":    0,
// 		"favoriteCategories": []string{},
// 	}

// 	// Get order statistics
// 	orderQuery := `
// 		SELECT
// 			COUNT(*) as total_orders,
// 			COALESCE(SUM(total_amount), 0) as total_spent,
// 			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_orders,
// 			COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered_orders,
// 			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_orders
// 		FROM orders
// 		WHERE buyer_id = ?
// 	`

// 	var totalOrders, pendingOrders, deliveredOrders, cancelledOrders int
// 	var totalSpent float64
// 	err := db.QueryRow(orderQuery, userID).Scan(&totalOrders, &totalSpent, &pendingOrders, &deliveredOrders, &cancelledOrders)
// 	if err != nil && err != sql.ErrNoRows {
// 		return nil, err
// 	}

// 	stats["totalOrders"] = totalOrders
// 	stats["totalSpent"] = totalSpent
// 	stats["pendingOrders"] = pendingOrders
// 	stats["deliveredOrders"] = deliveredOrders
// 	stats["cancelledOrders"] = cancelledOrders

// 	// Get favorite categories (top 3 categories by order count)
// 	categoryQuery := `
// 		SELECT p.category, COUNT(*) as order_count
// 		FROM orders o
// 		JOIN order_items oi ON o.id = oi.order_id
// 		JOIN products p ON oi.product_id = p.id
// 		WHERE o.buyer_id = ? AND o.status != 'cancelled'
// 		GROUP BY p.category
// 		ORDER BY order_count DESC
// 		LIMIT 3
// 	`

// 	rows, err := db.Query(categoryQuery, userID)
// 	if err != nil {
// 		return stats, nil // Return stats without categories if query fails
// 	}
// 	defer rows.Close()

// 	var favoriteCategories []string
// 	for rows.Next() {
// 		var category string
// 		var count int

// 		err := rows.Scan(&category, &count)
// 		if err != nil {
// 			continue
// 		}

// 		favoriteCategories = append(favoriteCategories, category)
// 	}
// 	stats["favoriteCategories"] = favoriteCategories

// 	return stats, nil
// }
