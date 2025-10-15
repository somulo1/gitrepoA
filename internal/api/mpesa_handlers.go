package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"vaultke-backend/config"
	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// Payment handlers
func InitiateMpesaSTK(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Parse request body
	var req models.MpesaTransaction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Phone number is required",
		})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Amount must be greater than 0",
		})
		return
	}

	// Validate phone number format (Kenyan format)
	phoneNumber := strings.TrimSpace(req.PhoneNumber)
	if strings.HasPrefix(phoneNumber, "0") {
		phoneNumber = "254" + phoneNumber[1:]
	} else if strings.HasPrefix(phoneNumber, "+254") {
		phoneNumber = phoneNumber[1:]
	} else if !strings.HasPrefix(phoneNumber, "254") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid phone number format. Use format: 254XXXXXXXXX",
		})
		return
	}

	// Update phone number in request
	req.PhoneNumber = phoneNumber

	// Set default values if not provided
	if req.AccountReference == "" {
		req.AccountReference = "VaultKe"
	}
	if req.TransactionDesc == "" {
		req.TransactionDesc = "VaultKe Deposit"
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get configuration
	cfg, exists := c.Get("config")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Configuration not available",
		})
		return
	}

	// Create M-Pesa service
	mpesaService := services.NewMpesaService(db.(*sql.DB), cfg.(*config.Config))

	// Generate reference for direct STK push
	reference := fmt.Sprintf("STK_%d_%s", time.Now().UnixNano(), userID.(string)[:8])

	// Create pending transaction record first
	transactionID, err := createPendingMpesaTransaction(db.(*sql.DB), &req, userID.(string), reference)
	if err != nil {
		log.Printf("Failed to create pending transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create transaction record",
		})
		return
	}

	// Initiate STK push
	stkResponse, err := mpesaService.InitiateSTKPush(&req)
	if err != nil {
		log.Printf("STK Push failed: %v", err)

		// Mark transaction as failed with proper error details
		failureReason := fmt.Sprintf("M-Pesa STK Push failed: %v", err)

		// Update transaction status to failed
		updateErr := updateTransactionStatus(db.(*sql.DB), transactionID, models.TransactionStatusFailed)
		if updateErr != nil {
			log.Printf("Failed to update transaction status to failed: %v", updateErr)
		}

		// Update additional failure details
		_, updateErr = db.(*sql.DB).Exec(`
			UPDATE transactions
			SET reference = ?,
			    description = CONCAT(description, ' - ', ?),
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, fmt.Sprintf("FAILED_STK_%d", time.Now().UnixNano()), failureReason, transactionID)

		if updateErr != nil {
			log.Printf("Failed to update transaction failure details: %v", updateErr)
		}

		// Return proper error response
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initiate M-Pesa payment: STK push failed",
			"details": err.Error(),
			"data": gin.H{
				"transactionId": transactionID,
				"status":        "failed",
				"reason":        failureReason,
			},
		})
		return
	}

	// Update transaction with checkout request ID in both fields for better lookup
	updateTransactionReference(db.(*sql.DB), transactionID, stkResponse.CheckoutRequestID)
	updateTransactionCheckoutRequestID(db.(*sql.DB), transactionID, stkResponse.CheckoutRequestID)

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "STK push initiated successfully",
		"data": gin.H{
			"transactionId":       transactionID,
			"checkoutRequestId":   stkResponse.CheckoutRequestID,
			"merchantRequestId":   stkResponse.MerchantRequestID,
			"customerMessage":     stkResponse.CustomerMessage,
			"responseDescription": stkResponse.ResponseDescription,
		},
	})
}

func HandleMpesaCallback(c *gin.Context) {
	log.Println("📱 M-Pesa callback received")

	// Parse callback data
	var callback models.MpesaCallback
	if err := c.ShouldBindJSON(&callback); err != nil {
		log.Printf("Failed to parse callback: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid callback format",
		})
		return
	}

	log.Printf("📱 Callback data: %+v", callback)

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		log.Println("Database connection not available")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get configuration
	cfg, exists := c.Get("config")
	if !exists {
		log.Println("Configuration not available")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Configuration not available",
		})
		return
	}

	// Create M-Pesa service
	mpesaService := services.NewMpesaService(db.(*sql.DB), cfg.(*config.Config))

	// Process the callback
	err := mpesaService.ProcessMpesaCallback(&callback)
	if err != nil {
		log.Printf("Failed to process M-Pesa callback: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to process callback",
		})
		return
	}

	log.Println("✅ M-Pesa callback processed successfully")

	// Return success response (required by Safaricom)
	c.JSON(http.StatusOK, gin.H{
		"ResultCode": 0,
		"ResultDesc": "Success",
	})
}

// GetMpesaTransactionStatus checks the status of an M-Pesa transaction
func GetMpesaTransactionStatus(c *gin.Context) {
	checkoutRequestID := c.Param("checkoutRequestId")
	if checkoutRequestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Checkout request ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get configuration
	cfg, exists := c.Get("config")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Configuration not available",
		})
		return
	}

	// Create M-Pesa service
	mpesaService := services.NewMpesaService(db.(*sql.DB), cfg.(*config.Config))

	// Check transaction status
	status, err := mpesaService.GetTransactionStatus(checkoutRequestID)
	if err != nil {
		log.Printf("Failed to get transaction status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check transaction status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"checkoutRequestId": checkoutRequestID,
			"status":            status,
		},
	})
}

// createPendingMpesaTransaction creates a pending transaction record
func createPendingMpesaTransaction(db *sql.DB, req *models.MpesaTransaction, userID string, reference string) (string, error) {
	// Find user's personal wallet
	var walletID string
	walletQuery := "SELECT id FROM wallets WHERE owner_id = ? AND type = ?"
	err := db.QueryRow(walletQuery, userID, models.WalletTypePersonal).Scan(&walletID)
	if err != nil {
		return "", fmt.Errorf("failed to find user wallet: %w", err)
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("TXN_%d", time.Now().UnixNano())

	// Create pending transaction with proper reference and auto-approval
	insertQuery := `
		INSERT INTO transactions (
			id, to_wallet_id, type, status, amount, currency, description,
			reference, payment_method, initiated_by, approved_by, requires_approval, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err = db.Exec(insertQuery,
		transactionID, walletID, models.TransactionTypeDeposit, models.TransactionStatusPending,
		req.Amount, "KES", req.TransactionDesc, reference, models.PaymentMethodMpesa,
		userID, userID, false, // approved_by = userID (self-approved deposit)
	)
	if err != nil {
		return "", fmt.Errorf("failed to create pending transaction: %w", err)
	}

	return transactionID, nil
}
