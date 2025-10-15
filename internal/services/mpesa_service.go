package services

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"vaultke-backend/config"
	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

// MpesaService handles M-Pesa payment integration
type MpesaService struct {
	db     *sql.DB
	config *config.Config
	client *http.Client
}

// getBaseURL returns the appropriate M-Pesa API base URL based on environment
func (s *MpesaService) getBaseURL() string {
	if s.config.Environment == "production" {
		return "https://api.safaricom.co.ke"
	}
	return "https://sandbox.safaricom.co.ke"
}

// NewMpesaService creates a new M-Pesa service
func NewMpesaService(db *sql.DB, cfg *config.Config) *MpesaService {
	return &MpesaService{
		db:     db,
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// MpesaTokenResponse represents M-Pesa access token response
type MpesaTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// MpesaSTKPushRequest represents STK push request
type MpesaSTKPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// MpesaSTKPushResponse represents STK push response
type MpesaSTKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// B2CRequest represents M-Pesa B2C (Business to Customer) request
type B2CRequest struct {
	InitiatorName      string  `json:"InitiatorName"`
	SecurityCredential string  `json:"SecurityCredential"`
	CommandID          string  `json:"CommandID"`
	Amount             float64 `json:"Amount"`
	PartyA             string  `json:"PartyA"`
	PartyB             string  `json:"PartyB"`
	Remarks            string  `json:"Remarks"`
	QueueTimeOutURL    string  `json:"QueueTimeOutURL"`
	ResultURL          string  `json:"ResultURL"`
	Occasion           string  `json:"Occasion"`
}

// B2CResponse represents M-Pesa B2C response
type B2CResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// GetAccessToken gets M-Pesa access token
func (s *MpesaService) GetAccessToken() (string, error) {
	// Create basic auth header
	auth := base64.StdEncoding.EncodeToString(
		[]byte(s.config.MpesaConsumerKey + ":" + s.config.MpesaConsumerSecret),
	)

	// Create request with environment-specific URL
	tokenURL := s.getBaseURL() + "/oauth/v1/generate?grant_type=client_credentials"
	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var tokenResp MpesaTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token received")
	}

	return tokenResp.AccessToken, nil
}

// GeneratePassword generates M-Pesa password
func (s *MpesaService) GeneratePassword(timestamp string) string {
	password := s.config.MpesaShortcode + s.config.MpesaPasskey + timestamp
	return base64.StdEncoding.EncodeToString([]byte(password))
}

// InitiateSTKPush initiates M-Pesa STK push
func (s *MpesaService) InitiateSTKPush(transaction *models.MpesaTransaction) (*MpesaSTKPushResponse, error) {
	// Get access token
	accessToken, err := s.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Generate timestamp and password
	timestamp := time.Now().Format("20060102150405")
	password := s.GeneratePassword(timestamp)

	// Create STK push request
	stkRequest := MpesaSTKPushRequest{
		BusinessShortCode: s.config.MpesaShortcode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            fmt.Sprintf("%.0f", transaction.Amount),
		PartyA:            transaction.PhoneNumber,
		PartyB:            s.config.MpesaShortcode,
		PhoneNumber:       transaction.PhoneNumber,
		CallBackURL:       s.config.MpesaCallbackURL,
		AccountReference:  transaction.AccountReference,
		TransactionDesc:   transaction.TransactionDesc,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(stkRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal STK request: %w", err)
	}

	// Create HTTP request with environment-specific URL
	stkURL := s.getBaseURL() + "/mpesa/stkpush/v1/processrequest"
	req, err := http.NewRequest("POST", stkURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create STK request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate STK push: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read STK response: %w", err)
	}

	// Parse response
	var stkResp MpesaSTKPushResponse
	if err := json.Unmarshal(body, &stkResp); err != nil {
		return nil, fmt.Errorf("failed to decode STK response: %w", err)
	}

	// Check if request was successful
	if stkResp.ResponseCode != "0" {
		return nil, fmt.Errorf("STK push failed: %s", stkResp.ResponseDescription)
	}

	return &stkResp, nil
}

// ProcessMpesaCallback processes M-Pesa callback
func (s *MpesaService) ProcessMpesaCallback(callback *models.MpesaCallback) error {
	log.Printf("🔍 Processing M-Pesa callback: CheckoutRequestID=%s, ResultCode=%d",
		callback.CheckoutRequestID, callback.ResultCode)

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if payment was successful
	if callback.ResultCode == 0 {
		// Payment successful
		log.Printf("✅ M-Pesa payment successful for CheckoutRequestID: %s", callback.CheckoutRequestID)
		amount := callback.GetMpesaAmount()
		receiptNumber := callback.GetMpesaReceiptNumber()
		phoneNumber := callback.GetMpesaPhoneNumber()
		log.Printf("📊 Payment details: Amount=%.2f, Receipt=%s, Phone=%s", amount, receiptNumber, phoneNumber)

		// Find the pending transaction - try multiple reference formats
		var transactionID string
		var findErr error

		// Try finding by checkout request ID in reference field
		findQuery := `
			SELECT id FROM transactions
			WHERE reference = ? AND status = ? AND payment_method = ?
		`
		findErr = tx.QueryRow(findQuery, callback.CheckoutRequestID, models.TransactionStatusPending, models.PaymentMethodMpesa).Scan(&transactionID)

		if findErr == sql.ErrNoRows {
			// Also try finding by checkout request ID in checkout_request_id field if it exists
			findQuery2 := `
				SELECT id FROM transactions
				WHERE checkout_request_id = ? AND status = ? AND payment_method = ?
			`
			findErr = tx.QueryRow(findQuery2, callback.CheckoutRequestID, models.TransactionStatusPending, models.PaymentMethodMpesa).Scan(&transactionID)
		}

		if findErr != nil {
			if findErr == sql.ErrNoRows {
				log.Printf("⚠️ No pending transaction found for CheckoutRequestID: %s, creating new transaction", callback.CheckoutRequestID)
				// Create new transaction if not found
				transactionID, err = s.createMpesaTransaction(tx, callback, amount, receiptNumber, phoneNumber)
				if err != nil {
					return fmt.Errorf("failed to create M-Pesa transaction: %w", err)
				}
			} else {
				return fmt.Errorf("failed to find transaction: %w", findErr)
			}
		} else {
			log.Printf("✅ Found pending transaction: %s, updating to completed", transactionID)
			// Update existing transaction
			err = s.updateMpesaTransaction(tx, transactionID, receiptNumber, phoneNumber)
			if err != nil {
				return fmt.Errorf("failed to update M-Pesa transaction: %w", err)
			}
		}

		// Process the transaction (update wallet balance)
		// Note: ProcessTransaction will handle its own transaction context
		// We need to commit our transaction first, then process the wallet update
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit M-Pesa transaction: %w", err)
		}

		// Now process the wallet transaction
		walletService := NewWalletService(s.db)
		err = walletService.ProcessTransaction(transactionID)
		if err != nil {
			return fmt.Errorf("failed to process transaction: %w", err)
		}

		return nil
	} else {
		// Payment failed
		log.Printf("❌ M-Pesa payment failed for CheckoutRequestID: %s, ResultCode: %d",
			callback.CheckoutRequestID, callback.ResultCode)

		// Find transaction by checkout request ID and update status
		var transactionID string
		var findErr error

		// Try finding by checkout request ID in reference field
		findErr = tx.QueryRow("SELECT id FROM transactions WHERE reference = ? AND payment_method = ?",
			callback.CheckoutRequestID, models.PaymentMethodMpesa).Scan(&transactionID)

		if findErr == sql.ErrNoRows {
			// Also try finding by checkout request ID in checkout_request_id field
			findErr = tx.QueryRow("SELECT id FROM transactions WHERE checkout_request_id = ? AND payment_method = ?",
				callback.CheckoutRequestID, models.PaymentMethodMpesa).Scan(&transactionID)
		}

		if findErr != nil {
			log.Printf("⚠️ No transaction found for failed payment CheckoutRequestID: %s", callback.CheckoutRequestID)
			return fmt.Errorf("failed to find transaction for failed payment: %w", findErr)
		}

		log.Printf("📝 Marking transaction %s as failed", transactionID)

		// Commit transaction first to release the lock
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Update transaction status using centralized function
		err = s.updateTransactionStatus(transactionID, models.TransactionStatusFailed)
		if err != nil {
			return fmt.Errorf("failed to update failed transaction status: %w", err)
		}

		log.Printf("✅ Successfully marked transaction %s as failed", transactionID)
		return nil
	}
}

// createMpesaTransaction creates a new M-Pesa transaction
func (s *MpesaService) createMpesaTransaction(tx *sql.Tx, callback *models.MpesaCallback, amount float64, receiptNumber, phoneNumber string) (string, error) {
	// Find user by phone number
	var userID string
	userQuery := "SELECT id FROM users WHERE phone = ?"
	err := tx.QueryRow(userQuery, phoneNumber).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user by phone: %w", err)
	}

	// Get user's personal wallet
	var walletID string
	walletQuery := "SELECT id FROM wallets WHERE owner_id = ? AND type = ?"
	err = tx.QueryRow(walletQuery, userID, models.WalletTypePersonal).Scan(&walletID)
	if err != nil {
		return "", fmt.Errorf("failed to find user wallet: %w", err)
	}

	// Create transaction
	transactionID := generateTransactionID()
	metadata := map[string]interface{}{
		"mpesa_receipt_number": receiptNumber,
		"mpesa_phone_number":   phoneNumber,
		"checkout_request_id":  callback.CheckoutRequestID,
		"merchant_request_id":  callback.MerchantRequestID,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	insertQuery := `
		INSERT INTO transactions (
			id, to_wallet_id, type, status, amount, currency, description,
			reference, payment_method, metadata, fees, initiated_by,
			requires_approval, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(insertQuery,
		transactionID, walletID, models.TransactionTypeDeposit, models.TransactionStatusCompleted,
		amount, "KES", "M-Pesa deposit", callback.CheckoutRequestID, models.PaymentMethodMpesa,
		string(metadataJSON), 0, userID, false, time.Now(), time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert transaction: %w", err)
	}

	return transactionID, nil
}

// updateTransactionStatus updates transaction status
func (s *MpesaService) updateTransactionStatus(transactionID string, status models.TransactionStatus) error {
	updateQuery := "UPDATE transactions SET status = ?, updated_at = ? WHERE id = ?"
	result, err := s.db.Exec(updateQuery, status, utils.NowEAT(), transactionID)
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

// updateMpesaTransaction updates an existing M-Pesa transaction
func (s *MpesaService) updateMpesaTransaction(tx *sql.Tx, transactionID, receiptNumber, phoneNumber string) error {
	log.Printf("📝 Updating M-Pesa transaction %s: Receipt=%s, Phone=%s", transactionID, receiptNumber, phoneNumber)

	// Update transaction metadata
	metadata := map[string]interface{}{
		"mpesa_receipt_number": receiptNumber,
		"mpesa_phone_number":   phoneNumber,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	updateQuery := `
		UPDATE transactions
		SET status = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := tx.Exec(updateQuery, models.TransactionStatusCompleted, string(metadataJSON), time.Now(), transactionID)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no transaction found with ID: %s", transactionID)
	}

	log.Printf("✅ Successfully updated transaction %s to completed status", transactionID)
	return nil
}

// generateTransactionID generates a unique transaction ID
func generateTransactionID() string {
	// Generate random bytes
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Convert to hex string
	return fmt.Sprintf("TXN_%X", bytes)
}

// GetTransactionStatus gets M-Pesa transaction status
func (s *MpesaService) GetTransactionStatus(checkoutRequestID string) (string, error) {
	// Get access token
	accessToken, err := s.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Generate timestamp and password
	timestamp := time.Now().Format("20060102150405")
	password := s.GeneratePassword(timestamp)

	// Create query request
	queryRequest := map[string]string{
		"BusinessShortCode": s.config.MpesaShortcode,
		"Password":          password,
		"Timestamp":         timestamp,
		"CheckoutRequestID": checkoutRequestID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(queryRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal query request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://sandbox.safaricom.co.ke/mpesa/stkpushquery/v1/query", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create query request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query transaction status: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read query response: %w", err)
	}

	// Parse response
	var queryResp map[string]interface{}
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return "", fmt.Errorf("failed to decode query response: %w", err)
	}

	// Extract status
	if resultCode, ok := queryResp["ResultCode"]; ok {
		if resultCode.(float64) == 0 {
			return "completed", nil
		} else {
			return "failed", nil
		}
	}

	return "pending", nil
}

// InitiateB2C initiates a Business to Customer (B2C) transaction for withdrawals
func (s *MpesaService) InitiateB2C(phoneNumber string, amount float64, remarks string) (*B2CResponse, error) {
	// Get access token
	token, err := s.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	// Create security credential (Base64 encoded initiator password)
	// In production, this should be properly encrypted
	securityCredential := base64.StdEncoding.EncodeToString([]byte(s.config.MpesaInitiatorPassword))

	// Prepare B2C request
	b2cRequest := B2CRequest{
		InitiatorName:      s.config.MpesaInitiatorName,
		SecurityCredential: securityCredential,
		CommandID:          "BusinessPayment", // or "SalaryPayment", "PromotionPayment"
		Amount:             amount,
		PartyA:             s.config.MpesaShortcode,
		PartyB:             phoneNumber,
		Remarks:            remarks,
		QueueTimeOutURL:    s.config.BaseURL + "/api/v1/mpesa/b2c/timeout",
		ResultURL:          s.config.BaseURL + "/api/v1/mpesa/b2c/callback",
		Occasion:           "Withdrawal",
	}

	// Convert to JSON
	jsonData, err := json.Marshal(b2cRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal B2C request: %v", err)
	}

	// Create HTTP request with environment-specific URL
	b2cURL := s.getBaseURL() + "/mpesa/b2c/v1/paymentrequest"
	req, err := http.NewRequest("POST", b2cURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create B2C request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Make request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make B2C request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read B2C response: %v", err)
	}

	// Parse response
	var response B2CResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse B2C response: %v", err)
	}

	// Log the response for debugging
	log.Printf("B2C Response: %+v", response)

	return &response, nil
}
