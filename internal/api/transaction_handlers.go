package api

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"net/http"
	"strconv"
	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"
	"vaultke-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func GetUserTransactions(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
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

	// Parse pagination parameters
	limit := 50
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Create wallet service
	walletService := services.NewWalletService(db.(*sql.DB))

	// Get user's personal wallet
	wallet, err := walletService.GetWalletByOwnerAndType(userID.(string), models.WalletTypePersonal)
	if err != nil {
		log.Printf("Failed to get user wallet: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Wallet not found",
		})
		return
	}

	// Get wallet transactions
	transactions, err := walletService.GetWalletTransactions(wallet.ID, limit, offset)
	if err != nil {
		log.Printf("Failed to get wallet transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve transactions",
		})
		return
	}

	// Format transactions for frontend compatibility
	var formattedTransactions []map[string]interface{}
	for _, tx := range transactions {
		// Get user information for the transaction initiator
		var userInfo map[string]interface{}
		if tx.InitiatedBy != "" {
			userQuery := `
				SELECT id, first_name, last_name, email, phone
				FROM users WHERE id = ?
			`
			var userID, firstName, lastName, email, phone string
			err := db.(*sql.DB).QueryRow(userQuery, tx.InitiatedBy).Scan(&userID, &firstName, &lastName, &email, &phone)
			if err == nil {
				userInfo = map[string]interface{}{
					"id":         userID,
					"firstName":  firstName,
					"lastName":   lastName,
					"fullName":   firstName + " " + lastName,
					"email":      email,
					"phone":      phone,
				}
			}
		}

		// Handle pointer fields safely
		description := ""
		if tx.Description != nil {
			description = *tx.Description
		}

		reference := ""
		if tx.Reference != nil {
			reference = *tx.Reference
		}

		recipientID := ""
		if tx.RecipientID != nil {
			recipientID = *tx.RecipientID
		}

		// Format transaction for frontend
		formattedTx := map[string]interface{}{
			"id":            tx.ID,
			"type":          tx.Type,
			"status":        tx.Status,
			"amount":        tx.Amount,
			"currency":      tx.Currency,
			"description":   description,
			"reference":     reference,
			"paymentMethod": tx.PaymentMethod,
			"fees":          tx.Fees,
			"initiatedBy":   tx.InitiatedBy,
			"recipientId":   recipientID,
			"createdAt":     tx.CreatedAt.Format(time.RFC3339),
			"updatedAt":     tx.UpdatedAt.Format(time.RFC3339),
			"metadata":      tx.Metadata,
		}

		// Add user information if available
		if userInfo != nil {
			formattedTx["user"] = userInfo
		}

		formattedTransactions = append(formattedTransactions, formattedTx)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    formattedTransactions,
		"meta": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"count":  len(formattedTransactions),
		},
	})
}

// updateTransactionStatus updates transaction status
func updateTransactionStatus(db *sql.DB, transactionID string, status models.TransactionStatus) error {
	updateQuery := "UPDATE transactions SET status = ?, updated_at = ? WHERE id = ?"
	result, err := db.Exec(updateQuery, status, utils.NowEAT(), transactionID)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	log.Printf("Successfully updated transaction %s status to %s", transactionID, status)
	return nil
}

// updateTransactionCheckoutRequestID updates transaction with checkout request ID
func updateTransactionCheckoutRequestID(db *sql.DB, transactionID string, checkoutRequestID string) {
	log.Printf("📝 Updating transaction %s checkout_request_id to: %s", transactionID, checkoutRequestID)
	updateQuery := "UPDATE transactions SET checkout_request_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	result, err := db.Exec(updateQuery, checkoutRequestID, transactionID)
	if err != nil {
		log.Printf("❌ Failed to update transaction checkout request ID: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		log.Printf("✅ Successfully updated transaction %s checkout_request_id", transactionID)
	} else {
		log.Printf("⚠️ No rows affected when updating transaction %s checkout_request_id", transactionID)
	}
}

// updateTransactionReference updates transaction reference
func updateTransactionReference(db *sql.DB, transactionID, reference string) {
	log.Printf("📝 Updating transaction %s reference to: %s", transactionID, reference)
	updateQuery := "UPDATE transactions SET reference = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	result, err := db.Exec(updateQuery, reference, transactionID)
	if err != nil {
		log.Printf("❌ Failed to update transaction reference: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		log.Printf("✅ Successfully updated transaction %s reference", transactionID)
	} else {
		log.Printf("⚠️ No rows affected when updating transaction %s reference", transactionID)
	}
}
