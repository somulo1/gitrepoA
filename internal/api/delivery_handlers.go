package api

// import (
// 	"database/sql"
// 	"log"
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// // GetDeliveries returns delivery assignments for delivery personnel
// func GetDeliveries(c *gin.Context) {
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

// 	status := c.Query("status")

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

// 	// Get real delivery data from database
// 	deliveries, err := getDeliveriesFromDB(db, userID, status)
// 	if err != nil {
// 		log.Printf("Error getting deliveries: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get deliveries"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    deliveries,
// 	})
// }

// // AcceptDelivery allows delivery personnel to accept a delivery
// func AcceptDelivery(c *gin.Context) {
// 	deliveryID := c.Param("id")
// 	if deliveryID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Delivery ID is required",
// 		})
// 		return
// 	}

// 	// Get user ID from context
// 	_, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get database connection
// 	_, exists = c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Mock acceptance for now
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delivery accepted successfully",
// 	})
// }

// // UpdateDeliveryStatus updates the status of a product delivery
// func UpdateDeliveryStatus(c *gin.Context) {
// 	orderItemID := c.Param("id") // This is the order_item_id
// 	if orderItemID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Order item ID is required",
// 		})
// 		return
// 	}

// 	var req struct {
// 		Status string `json:"status" binding:"required"`
// 		Notes  string `json:"notes"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Invalid request data: " + err.Error(),
// 		})
// 		return
// 	}

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

// 	// Verify this delivery person is assigned to this product
// 	var assignedDeliveryPersonID string
// 	checkQuery := `SELECT delivery_person_id FROM order_items WHERE id = ?`
// 	err := db.QueryRow(checkQuery, orderItemID).Scan(&assignedDeliveryPersonID)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Order item not found",
// 			})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to verify delivery assignment",
// 		})
// 		return
// 	}

// 	// Check if this delivery person is assigned to this product
// 	if assignedDeliveryPersonID != userID {
// 		c.JSON(http.StatusForbidden, gin.H{
// 			"success": false,
// 			"error":   "You are not assigned to deliver this product",
// 		})
// 		return
// 	}

// 	// Update the order delivery status (this affects the whole order)
// 	// Get the order_id first
// 	var orderID string
// 	orderQuery := `SELECT order_id FROM order_items WHERE id = ?`
// 	err = db.QueryRow(orderQuery, orderItemID).Scan(&orderID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get order information",
// 		})
// 		return
// 	}

// 	// Update order delivery status
// 	updateQuery := `
// 		UPDATE orders
// 		SET delivery_status = ?, updated_at = CURRENT_TIMESTAMP
// 		WHERE id = ?
// 	`
// 	_, err = db.Exec(updateQuery, req.Status, orderID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to update delivery status",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delivery status updated successfully",
// 		"data": map[string]interface{}{
// 			"orderItemId": orderItemID,
// 			"orderId":     orderID,
// 			"status":      req.Status,
// 			"notes":       req.Notes,
// 		},
// 	})
// }

// // getDeliveriesFromDB retrieves products assigned to the delivery person
// func getDeliveriesFromDB(db *sql.DB, userID, status string) ([]map[string]interface{}, error) {
// 	var deliveries []map[string]interface{}

// 	// Query to get products assigned to this delivery person
// 	query := `
// 		SELECT
// 			oi.id as order_item_id,
// 			oi.order_id,
// 			oi.product_id,
// 			oi.quantity,
// 			oi.price,
// 			oi.name as product_name,
// 			oi.delivery_person_id,
// 			o.delivery_status,
// 			o.delivery_address,
// 			o.delivery_phone,
// 			o.delivery_fee,
// 			o.delivery_town,
// 			o.delivery_county,
// 			o.notes as order_notes,
// 			u.first_name as buyer_first_name,
// 			u.last_name as buyer_last_name,
// 			u.phone as buyer_phone,
// 			u.email as buyer_email,
// 			p.name as product_full_name,
// 			p.description as product_description,
// 			p.images as product_images,
// 			s.first_name as seller_first_name,
// 			s.last_name as seller_last_name
// 		FROM order_items oi
// 		JOIN orders o ON oi.order_id = o.id
// 		JOIN users u ON o.buyer_id = u.id
// 		JOIN products p ON oi.product_id = p.id
// 		JOIN users s ON p.seller_id = s.id
// 		WHERE oi.delivery_person_id = ?
// 	`

// 	args := []interface{}{userID}

// 	// Add status filter if provided
// 	if status != "" && status != "all" {
// 		if status == "pending" {
// 			query += " AND o.delivery_status = 'pending'"
// 		} else if status == "assigned" {
// 			query += " AND o.delivery_status = 'assigned'"
// 		} else if status == "in_transit" {
// 			query += " AND o.delivery_status = 'in_transit'"
// 		} else if status == "delivered" {
// 			query += " AND o.delivery_status = 'delivered'"
// 		} else {
// 			query += " AND o.delivery_status = ?"
// 			args = append(args, status)
// 		}
// 	}

// 	query += " ORDER BY o.created_at DESC"

// 	rows, err := db.Query(query, args...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var orderItemID, orderID, productID, productName, deliveryPersonID string
// 		var deliveryStatus, deliveryAddress, deliveryPhone, deliveryTown, deliveryCounty string
// 		var orderNotes, buyerFirstName, buyerLastName, buyerPhone, buyerEmail string
// 		var productFullName, productDescription, productImages string
// 		var sellerFirstName, sellerLastName string
// 		var quantity int
// 		var price, deliveryFee float64

// 		err := rows.Scan(
// 			&orderItemID, &orderID, &productID, &quantity, &price, &productName,
// 			&deliveryPersonID, &deliveryStatus, &deliveryAddress, &deliveryPhone,
// 			&deliveryFee, &deliveryTown, &deliveryCounty, &orderNotes,
// 			&buyerFirstName, &buyerLastName, &buyerPhone, &buyerEmail,
// 			&productFullName, &productDescription, &productImages,
// 			&sellerFirstName, &sellerLastName,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		delivery := map[string]interface{}{
// 			"id":              orderItemID,
// 			"orderItemId":     orderItemID,
// 			"orderId":         orderID,
// 			"productId":       productID,
// 			"quantity":        quantity,
// 			"price":           price,
// 			"deliveryStatus":  deliveryStatus,
// 			"deliveryAddress": deliveryAddress,
// 			"deliveryPhone":   deliveryPhone,
// 			"deliveryFee":     deliveryFee,
// 			"deliveryTown":    deliveryTown,
// 			"deliveryCounty":  deliveryCounty,
// 			"orderNotes":      orderNotes,
// 			"product": map[string]interface{}{
// 				"id":          productID,
// 				"name":        productFullName,
// 				"description": productDescription,
// 				"images":      productImages,
// 			},
// 			"buyer": map[string]interface{}{
// 				"firstName": buyerFirstName,
// 				"lastName":  buyerLastName,
// 				"phone":     buyerPhone,
// 				"email":     buyerEmail,
// 			},
// 			"seller": map[string]interface{}{
// 				"firstName": sellerFirstName,
// 				"lastName":  sellerLastName,
// 			},
// 		}

// 		deliveries = append(deliveries, delivery)
// 	}

// 	return deliveries, nil
// }

// // AssignDeliveryPersonToProduct assigns a delivery person to specific products in an order
// func AssignDeliveryPersonToProduct(c *gin.Context) {
// 	var req struct {
// 		OrderItemIDs      []string `json:"orderItemIds" binding:"required"`
// 		DeliveryPersonID  string   `json:"deliveryPersonId" binding:"required"`
// 		DeliveryFee       float64  `json:"deliveryFee"`
// 		EstimatedDelivery string   `json:"estimatedDelivery"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Invalid request data: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Get user ID from context (seller)
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

// 	// Verify the delivery person exists and is a delivery person
// 	var deliveryPersonExists bool
// 	checkDeliveryPersonQuery := `
// 		SELECT EXISTS(
// 			SELECT 1 FROM marketplace_roles
// 			WHERE user_id = ? AND role = 'delivery_person'
// 		)
// 	`
// 	err := db.QueryRow(checkDeliveryPersonQuery, req.DeliveryPersonID).Scan(&deliveryPersonExists)
// 	if err != nil || !deliveryPersonExists {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Invalid delivery person",
// 		})
// 		return
// 	}

// 	// Start transaction
// 	tx, err := db.Begin()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to start transaction",
// 		})
// 		return
// 	}
// 	defer tx.Rollback()

// 	// Assign delivery person to each product
// 	for _, orderItemID := range req.OrderItemIDs {
// 		// Verify the seller owns this product
// 		var sellerID string
// 		checkOwnershipQuery := `
// 			SELECT p.seller_id
// 			FROM order_items oi
// 			JOIN products p ON oi.product_id = p.id
// 			WHERE oi.id = ?
// 		`
// 		err := tx.QueryRow(checkOwnershipQuery, orderItemID).Scan(&sellerID)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to verify product ownership",
// 			})
// 			return
// 		}

// 		if sellerID != userID {
// 			c.JSON(http.StatusForbidden, gin.H{
// 				"success": false,
// 				"error":   "You don't have permission to assign delivery for this product",
// 			})
// 			return
// 		}

// 		// Update the order item with delivery person
// 		updateQuery := `
// 			UPDATE order_items
// 			SET delivery_person_id = ?, assigned_at = CURRENT_TIMESTAMP
// 			WHERE id = ?
// 		`
// 		_, err = tx.Exec(updateQuery, req.DeliveryPersonID, orderItemID)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to assign delivery person to product",
// 			})
// 			return
// 		}
// 	}

// 	// Update the order status to 'assigned'
// 	if len(req.OrderItemIDs) > 0 {
// 		// Get the order ID from the first item
// 		var orderID string
// 		getOrderQuery := `SELECT order_id FROM order_items WHERE id = ?`
// 		err := tx.QueryRow(getOrderQuery, req.OrderItemIDs[0]).Scan(&orderID)
// 		if err == nil {
// 			// Update order delivery status
// 			updateOrderQuery := `
// 				UPDATE orders
// 				SET delivery_status = 'assigned', delivery_person_id = ?, updated_at = CURRENT_TIMESTAMP
// 				WHERE id = ?
// 			`
// 			tx.Exec(updateOrderQuery, req.DeliveryPersonID, orderID)
// 		}
// 	}

// 	// Commit transaction
// 	if err := tx.Commit(); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to commit assignment",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delivery person assigned to products successfully",
// 		"data": map[string]interface{}{
// 			"orderItemIds":     req.OrderItemIDs,
// 			"deliveryPersonId": req.DeliveryPersonID,
// 			"assignedAt":       "now",
// 		},
// 	})
// }

// // Legacy function - kept for backward compatibility
// func AssignDeliveryPerson(c *gin.Context) {
// 	orderID := c.Param("id")
// 	if orderID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Order ID is required",
// 		})
// 		return
// 	}

// 	// Get user ID from context
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

// 	// For now, we'll simulate assigning a delivery person
// 	// In a real implementation, this would integrate with a delivery service
// 	query := `
// 		UPDATE orders
// 		SET delivery_status = 'assigned', updated_at = CURRENT_TIMESTAMP
// 		WHERE id = ? AND seller_id = ?
// 	`
// 	result, err := db.(*sql.DB).Exec(query, orderID, userID.(string))
// 	if err != nil {
// 		log.Printf("Failed to assign delivery person: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to assign delivery person",
// 		})
// 		return
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil || rowsAffected == 0 {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"success": false,
// 			"error":   "Order not found or you don't have permission to assign delivery",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delivery person assigned successfully",
// 	})
// }
