package api

// import (
// 	"database/sql"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"
// 	"vaultke-backend/internal/config"
// 	"vaultke-backend/internal/models"
// 	"vaultke-backend/internal/services"

// 	"github.com/gin-gonic/gin"
// )

// func GetOrder(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get order ID from URL parameter
// 	orderID := c.Param("id")
// 	if orderID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Order ID is required",
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

// 	// Get order by ID
// 	order, err := marketplaceService.GetOrderByID(orderID, userID.(string))
// 	if err != nil {
// 		if err.Error() == "order not found" {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Order not found",
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to get order: " + err.Error(),
// 			})
// 		}
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    order,
// 		"message": "Order retrieved successfully",
// 	})
// }

// func UpdateOrder(c *gin.Context) {
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Update order endpoint - coming soon",
// 	})
// }

// func CreateOrder(c *gin.Context) {
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
// 		Items           []string `json:"items" binding:"required"`         // Cart item IDs
// 		PaymentMethod   string   `json:"paymentMethod" binding:"required"` // wallet, mpesa, bank
// 		DeliveryCounty  string   `json:"deliveryCounty" binding:"required"`
// 		DeliveryTown    string   `json:"deliveryTown" binding:"required"`
// 		DeliveryAddress string   `json:"deliveryAddress" binding:"required"`
// 		DeliveryPhone   string   `json:"deliveryPhone" binding:"required"`
// 		Notes           string   `json:"notes"`
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

// 	// Create order from cart items
// 	var notes *string
// 	if req.Notes != "" {
// 		notes = &req.Notes
// 	}

// 	order, err := marketplaceService.CreateOrderFromCart(userID.(string), req.Items, &models.OrderCreation{
// 		PaymentMethod:   req.PaymentMethod,
// 		DeliveryCounty:  req.DeliveryCounty,
// 		DeliveryTown:    req.DeliveryTown,
// 		DeliveryAddress: req.DeliveryAddress,
// 		DeliveryPhone:   req.DeliveryPhone,
// 		Notes:           notes,
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Failed to create order: " + err.Error(),
// 		})
// 		return
// 	}

// 	// If payment method is M-Pesa, initiate STK push
// 	if req.PaymentMethod == "mpesa" {
// 		// Get configuration
// 		cfg, exists := c.Get("config")
// 		if !exists {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Configuration not available",
// 			})
// 			return
// 		}

// 		// Create M-Pesa service
// 		mpesaService := services.NewMpesaService(db.(*sql.DB), cfg.(*config.Config))

// 		// Prepare M-Pesa transaction request
// 		mpesaReq := &models.MpesaTransaction{
// 			PhoneNumber:      req.DeliveryPhone,
// 			Amount:           order.TotalAmount,
// 			AccountReference: fmt.Sprintf("ORDER_%s", order.ID[:8]),
// 			TransactionDesc:  fmt.Sprintf("Payment for Order %s", order.ID[:8]),
// 		}

// 		// Validate and format phone number
// 		phoneNumber := strings.TrimSpace(mpesaReq.PhoneNumber)
// 		if strings.HasPrefix(phoneNumber, "0") {
// 			phoneNumber = "254" + phoneNumber[1:]
// 		} else if strings.HasPrefix(phoneNumber, "+254") {
// 			phoneNumber = phoneNumber[1:]
// 		} else if !strings.HasPrefix(phoneNumber, "254") {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"success": false,
// 				"error":   "Invalid phone number format for M-Pesa payment",
// 			})
// 			return
// 		}
// 		mpesaReq.PhoneNumber = phoneNumber

// 		// Generate reference for the payment
// 		reference := fmt.Sprintf("ORDER_%s_%d", order.ID[:8], time.Now().UnixNano())

// 		// Create pending M-Pesa transaction record
// 		transactionID, err := createPendingMpesaTransaction(db.(*sql.DB), mpesaReq, userID.(string), reference)
// 		if err != nil {
// 			log.Printf("Failed to create pending M-Pesa transaction: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to create payment transaction",
// 			})
// 			return
// 		}

// 		// Initiate STK push
// 		stkResponse, err := mpesaService.InitiateSTKPush(mpesaReq)
// 		if err != nil {
// 			log.Printf("STK Push failed for order %s: %v", order.ID, err)

// 			// Mark transaction as failed
// 			_, updateErr := db.(*sql.DB).Exec(`
// 				UPDATE transactions
// 				SET status = 'failed',
// 				    description = CONCAT(description, ' - STK Push failed'),
// 				    updated_at = CURRENT_TIMESTAMP
// 				WHERE id = ?
// 			`, transactionID)

// 			if updateErr != nil {
// 				log.Printf("Failed to update transaction status: %v", updateErr)
// 			}

// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Order created but M-Pesa payment failed to initiate",
// 				"data": gin.H{
// 					"orderId":       order.ID,
// 					"transactionId": transactionID,
// 					"status":        "payment_failed",
// 				},
// 			})
// 			return
// 		}

// 		// Update transaction with checkout request ID
// 		updateTransactionReference(db.(*sql.DB), transactionID, stkResponse.CheckoutRequestID)

// 		// Return success response with payment details
// 		c.JSON(http.StatusCreated, gin.H{
// 			"success": true,
// 			"message": "Order created and M-Pesa payment initiated successfully",
// 			"data": gin.H{
// 				"order":             order,
// 				"transactionId":     transactionID,
// 				"checkoutRequestId": stkResponse.CheckoutRequestID,
// 				"paymentStatus":     "pending",
// 				"message":           "Please check your phone for M-Pesa payment prompt",
// 			},
// 		})
// 	} else {
// 		// Return success response for non-M-Pesa payments
// 		c.JSON(http.StatusCreated, gin.H{
// 			"success": true,
// 			"message": "Order created successfully",
// 			"data":    order,
// 		})
// 	}
// }

// func GetOrders(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get query parameters
// 	limitStr := c.DefaultQuery("limit", "20")
// 	offsetStr := c.DefaultQuery("offset", "0")
// 	status := c.Query("status")
// 	role := c.DefaultQuery("role", "buyer") // buyer or seller

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
// 	if status != "" {
// 		filters["status"] = status
// 	}
// 	filters["role"] = role

// 	// Get orders
// 	orders, err := marketplaceService.GetOrders(userID.(string), filters, limit, offset)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get orders: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    orders,
// 		"meta": map[string]interface{}{
// 			"limit":  limit,
// 			"offset": offset,
// 			"count":  len(orders),
// 			"role":   role,
// 		},
// 	})
// }

// // UpdateOrderStatus updates the status of an order
// func UpdateOrderStatus(c *gin.Context) {
// 	orderID := c.Param("id")
// 	if orderID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Order ID is required",
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

// 	// Update order status
// 	query := `
// 		UPDATE orders
// 		SET status = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
// 		WHERE id = ? AND seller_id = ?
// 	`
// 	result, err := db.(*sql.DB).Exec(query, req.Status, req.Notes, orderID, userID.(string))
// 	if err != nil {
// 		log.Printf("Failed to update order status: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to update order status",
// 		})
// 		return
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil || rowsAffected == 0 {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"success": false,
// 			"error":   "Order not found or you don't have permission to update it",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Order status updated successfully",
// 	})
// }
