package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"vaultke-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// ReceiptHandlers handles receipt-related API endpoints
type ReceiptHandlers struct {
	db *sql.DB
}

// NewReceiptHandlers creates a new receipt handlers instance
func NewReceiptHandlers(db *sql.DB) *ReceiptHandlers {
	return &ReceiptHandlers{db: db}
}

// CompanyInfo represents company information for receipts
type CompanyInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Website string `json:"website"`
	Logo    string `json:"logo"`
}

// ReceiptData represents the complete receipt data
type ReceiptData struct {
	ReceiptID   string                 `json:"receiptId"`
	CompanyInfo CompanyInfo            `json:"companyInfo"`
	Transaction models.Transaction     `json:"transaction"`
	UserInfo    map[string]interface{} `json:"userInfo"`
	GeneratedAt time.Time              `json:"generatedAt"`
	Version     string                 `json:"version"`
}

// GetTransactionReceipt generates and returns a transaction receipt
func (h *ReceiptHandlers) GetTransactionReceipt(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	transactionID := c.Param("transactionId")
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Transaction ID is required",
		})
		return
	}

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "html" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Supported formats: json, html",
		})
		return
	}

	// Get transaction details
	transaction, err := h.getTransactionByID(transactionID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Transaction not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve transaction",
		})
		return
	}

	// Get user information
	userInfo, err := h.getUserInfo(userID)
	if err != nil {
		// Continue without user info if not found
		userInfo = make(map[string]interface{})
	}

	// Generate receipt data
	receiptData := h.generateReceiptData(transaction, userInfo)

	switch format {
	case "json":
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    receiptData,
		})
	case "html":
		html := h.generateReceiptHTML(receiptData)
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	}
}

// DownloadTransactionReceipt generates and serves a downloadable receipt
func (h *ReceiptHandlers) DownloadTransactionReceipt(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	transactionID := c.Param("transactionId")
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Transaction ID is required",
		})
		return
	}

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "html" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Supported formats: json, html",
		})
		return
	}

	// Get transaction details
	transaction, err := h.getTransactionByID(transactionID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Transaction not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve transaction",
		})
		return
	}

	// Get user information
	userInfo, err := h.getUserInfo(userID)
	if err != nil {
		userInfo = make(map[string]interface{})
	}

	// Generate receipt data
	receiptData := h.generateReceiptData(transaction, userInfo)
	receiptID := fmt.Sprintf("RCP-%s", transaction.ID[:8])
	fileName := fmt.Sprintf("VaultKe_Receipt_%s_%s.%s",
		receiptID,
		time.Now().Format("2006-01-02"),
		format)

	switch format {
	case "json":
		jsonData, _ := json.MarshalIndent(receiptData, "", "  ")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, string(jsonData))
	case "html":
		html := h.generateReceiptHTML(receiptData)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	}
}

// getTransactionByID retrieves a transaction by ID for a specific user
func (h *ReceiptHandlers) getTransactionByID(transactionID, userID string) (*models.Transaction, error) {
	query := `
		SELECT id, from_wallet_id, to_wallet_id, type, status, amount, currency,
			   description, reference, payment_method, metadata, fees, initiated_by,
			   approved_by, requires_approval, approval_deadline, created_at, updated_at
		FROM transactions 
		WHERE id = ? AND (initiated_by = ? OR approved_by = ?)
	`

	var transaction models.Transaction
	var fromWalletID, toWalletID, approvedBy sql.NullString
	var approvalDeadline sql.NullTime
	var metadataJSON string

	err := h.db.QueryRow(query, transactionID, userID, userID).Scan(
		&transaction.ID,
		&fromWalletID,
		&toWalletID,
		&transaction.Type,
		&transaction.Status,
		&transaction.Amount,
		&transaction.Currency,
		&transaction.Description,
		&transaction.Reference,
		&transaction.PaymentMethod,
		&metadataJSON,
		&transaction.Fees,
		&transaction.InitiatedBy,
		&approvedBy,
		&transaction.RequiresApproval,
		&approvalDeadline,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if fromWalletID.Valid {
		transaction.FromWalletID = &fromWalletID.String
	}
	if toWalletID.Valid {
		transaction.ToWalletID = &toWalletID.String
	}
	if approvedBy.Valid {
		transaction.ApprovedBy = &approvedBy.String
	}
	if approvalDeadline.Valid {
		transaction.ApprovalDeadline = &approvalDeadline.Time
	}

	// Parse metadata
	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &transaction.Metadata)
	}

	return &transaction, nil
}

// getUserInfo retrieves user information for the receipt
func (h *ReceiptHandlers) getUserInfo(userID string) (map[string]interface{}, error) {
	query := `
		SELECT first_name, last_name, email, phone, county, town
		FROM users 
		WHERE id = ?
	`

	var firstName, lastName, email, phone, county, town sql.NullString
	err := h.db.QueryRow(query, userID).Scan(
		&firstName, &lastName, &email, &phone, &county, &town,
	)

	if err != nil {
		return nil, err
	}

	userInfo := make(map[string]interface{})
	if firstName.Valid {
		userInfo["firstName"] = firstName.String
	}
	if lastName.Valid {
		userInfo["lastName"] = lastName.String
	}
	if email.Valid {
		userInfo["email"] = email.String
	}
	if phone.Valid {
		userInfo["phone"] = phone.String
	}
	if county.Valid {
		userInfo["county"] = county.String
	}
	if town.Valid {
		userInfo["town"] = town.String
	}

	return userInfo, nil
}

// generateReceiptData creates the complete receipt data structure
func (h *ReceiptHandlers) generateReceiptData(transaction *models.Transaction, userInfo map[string]interface{}) *ReceiptData {
	receiptID := fmt.Sprintf("RCP-%s", transaction.ID[:8])

	companyInfo := CompanyInfo{
		Name:    "VaultKe",
		Address: "Kisumu, Kenya",
		Phone:   "+254 700 000 000",
		Email:   "support@vaultke.co.ke",
		Website: "www.vaultke.co.ke",
		Logo:    "https://vaultke.co.ke/logo.png",
	}

	return &ReceiptData{
		ReceiptID:   receiptID,
		CompanyInfo: companyInfo,
		Transaction: *transaction,
		UserInfo:    userInfo,
		GeneratedAt: time.Now(),
		Version:     "1.0",
	}
}

// generateReceiptHTML creates HTML receipt from receipt data
func (h *ReceiptHandlers) generateReceiptHTML(receiptData *ReceiptData) string {
	formatCurrency := func(amount float64) string {
		return fmt.Sprintf("KES %.2f", amount)
	}

	getTransactionTypeLabel := func(txType models.TransactionType) string {
		switch txType {
		case models.TransactionTypeDeposit:
			return "Deposit"
		case models.TransactionTypeWithdrawal:
			return "Withdrawal"
		case models.TransactionTypeTransfer:
			return "Transfer"
		case models.TransactionTypeContribution:
			return "Contribution"
		case models.TransactionTypeLoan:
			return "Loan"
		case models.TransactionTypeLoanRepayment:
			return "Loan Repayment"
		case models.TransactionTypePurchase:
			return "Purchase"
		case models.TransactionTypeRefund:
			return "Refund"
		case models.TransactionTypeFee:
			return "Fee"
		default:
			return "Transaction"
		}
	}

	transactionDate := receiptData.Transaction.CreatedAt.Format("January 2, 2006 at 3:04:05 PM")
	amount := formatCurrency(receiptData.Transaction.Amount)
	fees := formatCurrency(receiptData.Transaction.Fees)
	totalAmount := formatCurrency(receiptData.Transaction.Amount + receiptData.Transaction.Fees)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Transaction Receipt - %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; background: #f5f5f5; padding: 20px; }
        .receipt-container { max-width: 600px; margin: 0 auto; background: white; border-radius: 10px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); overflow: hidden; }
        .receipt-header { background: linear-gradient(135deg, #6366F1, #8B5CF6); color: white; padding: 30px; text-align: center; }
        .company-logo { width: 60px; height: 60px; margin: 0 auto 15px; background: white; border-radius: 50%%; display: flex; align-items: center; justify-content: center; font-size: 24px; font-weight: bold; color: #6366F1; }
        .company-name { font-size: 28px; font-weight: bold; margin-bottom: 5px; }
        .receipt-title { font-size: 18px; opacity: 0.9; }
        .receipt-body { padding: 30px; }
        .receipt-info { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 30px; padding: 20px; background: #f8f9fa; border-radius: 8px; }
        .info-item { display: flex; flex-direction: column; }
        .info-label { font-size: 12px; color: #666; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 5px; }
        .info-value { font-size: 14px; font-weight: 600; color: #333; }
        .transaction-details { border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; margin-bottom: 30px; }
        .detail-row { display: flex; justify-content: space-between; align-items: center; padding: 15px 20px; border-bottom: 1px solid #e5e7eb; }
        .detail-row:last-child { border-bottom: none; }
        .detail-row.total { background: #f8f9fa; font-weight: bold; font-size: 16px; }
        .detail-label { color: #666; }
        .detail-value { font-weight: 600; color: #333; }
        .status-badge { display: inline-block; padding: 6px 12px; border-radius: 20px; font-size: 12px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
        .status-completed { background: #dcfce7; color: #166534; }
        .status-pending { background: #fef3c7; color: #92400e; }
        .status-failed { background: #fee2e2; color: #991b1b; }
        .receipt-footer { background: #f8f9fa; padding: 20px 30px; text-align: center; border-top: 1px solid #e5e7eb; }
        .company-details { font-size: 12px; color: #666; line-height: 1.5; }
        .thank-you { font-size: 14px; color: #333; margin-bottom: 10px; font-weight: 600; }
        @media print { body { background: white; padding: 0; } .receipt-container { box-shadow: none; border-radius: 0; } }
    </style>
</head>
<body>
    <div class="receipt-container">
        <div class="receipt-header">
            <div class="company-logo">VK</div>
            <div class="company-name">%s</div>
            <div class="receipt-title">Transaction Receipt</div>
        </div>

        <div class="receipt-body">
            <div class="receipt-info">
                <div class="info-item">
                    <div class="info-label">Receipt ID</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Transaction ID</div>
                    <div class="info-value">%s...</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Date & Time</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Status</div>
                    <div class="info-value">
                        <span class="status-badge status-%s">%s</span>
                    </div>
                </div>
            </div>

            <div class="transaction-details">
                <div class="detail-row">
                    <span class="detail-label">Transaction Type</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Description</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Amount</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Transaction Fees</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row total">
                    <span class="detail-label">Total Amount</span>
                    <span class="detail-value">%s</span>
                </div>`,
		receiptData.ReceiptID,
		receiptData.CompanyInfo.Name,
		receiptData.ReceiptID,
		receiptData.Transaction.ID[:16],
		transactionDate,
		receiptData.Transaction.Status,
		receiptData.Transaction.Status,
		getTransactionTypeLabel(receiptData.Transaction.Type),
		func() string {
			if receiptData.Transaction.Description != nil {
				return *receiptData.Transaction.Description
			}
			return "N/A"
		}(),
		amount,
		fees,
		totalAmount,
	)

	// Add optional fields
	if receiptData.Transaction.Reference != nil && *receiptData.Transaction.Reference != "" {
		html += fmt.Sprintf(`
                <div class="detail-row">
                    <span class="detail-label">Reference</span>
                    <span class="detail-value">%s</span>
                </div>`, *receiptData.Transaction.Reference)
	}

	if receiptData.Transaction.PaymentMethod != "" {
		html += fmt.Sprintf(`
                <div class="detail-row">
                    <span class="detail-label">Payment Method</span>
                    <span class="detail-value">%s</span>
                </div>`, receiptData.Transaction.PaymentMethod)
	}

	// Close the HTML
	html += fmt.Sprintf(`
            </div>
        </div>

        <div class="receipt-footer">
            <div class="thank-you">Thank you for using %s!</div>
            <div class="company-details">
                %s<br>
                Phone: %s | Email: %s<br>
                %s
            </div>
        </div>
    </div>
</body>
</html>`,
		receiptData.CompanyInfo.Name,
		receiptData.CompanyInfo.Address,
		receiptData.CompanyInfo.Phone,
		receiptData.CompanyInfo.Email,
		receiptData.CompanyInfo.Website,
	)

	return html
}
