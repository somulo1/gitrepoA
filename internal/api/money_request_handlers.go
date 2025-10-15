package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MoneyRequestHandlers struct {
	db *sql.DB
}

func NewMoneyRequestHandlers(db *sql.DB) *MoneyRequestHandlers {
	return &MoneyRequestHandlers{db: db}
}

// normalizePhoneNumber removes spaces, dashes and normalizes phone number format
func normalizePhoneNumber(phone string) string {
	// Remove all non-digit characters except +
	re := regexp.MustCompile(`[^\d+]`)
	normalized := re.ReplaceAllString(phone, "")

	// Remove leading + if present
	normalized = strings.TrimPrefix(normalized, "+")

	// If it starts with 254, keep as is
	if strings.HasPrefix(normalized, "254") {
		return normalized
	}

	// If it starts with 0, replace with 254
	if strings.HasPrefix(normalized, "0") {
		return "254" + normalized[1:]
	}

	// If it's 9 digits starting with 7, add 254
	if len(normalized) == 9 && strings.HasPrefix(normalized, "7") {
		return "254" + normalized
	}

	return normalized
}

type CreateMoneyRequestRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Reason      string  `json:"reason"`
	RequestType string  `json:"requestType" binding:"required,oneof=qr_code direct"`
}

type SendMoneyRequestRequest struct {
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	Reason       string  `json:"reason"`
	TargetUserID string  `json:"targetUserId"`
	TargetPhone  string  `json:"targetPhone"`
	RequestType  string  `json:"requestType" binding:"required,oneof=direct"`
}

type MoneyRequest struct {
	ID           string    `json:"id"`
	RequesterID  string    `json:"requesterId"`
	TargetUserID *string   `json:"targetUserId,omitempty"`
	Amount       float64   `json:"amount"`
	Reason       string    `json:"reason"`
	RequestType  string    `json:"requestType"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// CreateMoneyRequest creates a new money request (for QR codes)
func (h *MoneyRequestHandlers) CreateMoneyRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req CreateMoneyRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Generate unique request ID
	requestID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour) // Expires in 24 hours

	// Insert money request into database
	query := `
		INSERT INTO money_requests (id, requester_id, amount, reason, request_type, status, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?, ?)
	`

	_, err := h.db.Exec(query, requestID, userID, req.Amount, req.Reason, req.RequestType, time.Now(), expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create money request: " + err.Error(),
		})
		return
	}

	// Return the created request
	moneyRequest := MoneyRequest{
		ID:          requestID,
		RequesterID: userID,
		Amount:      req.Amount,
		Reason:      req.Reason,
		RequestType: req.RequestType,
		Status:      "pending",
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    moneyRequest,
		"message": "Money request created successfully",
	})
}

// SendMoneyRequest sends a direct money request to a specific user
func (h *MoneyRequestHandlers) SendMoneyRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req SendMoneyRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Find target user by ID or phone
	var targetUserID string
	if req.TargetUserID != "" {
		targetUserID = req.TargetUserID
	} else if req.TargetPhone != "" {
		// Normalize phone number (remove spaces, dashes, and ensure proper format)
		normalizedPhone := normalizePhoneNumber(req.TargetPhone)

		// Find user by phone number - try multiple formats
		query := `
			SELECT id FROM users
			WHERE phone = ? OR phone = ? OR phone = ? OR phone = ?
		`

		// Try different phone number formats
		formats := []string{
			normalizedPhone,              // As provided (normalized)
			req.TargetPhone,              // Original format
			"+254" + normalizedPhone[1:], // Add country code if missing
			"254" + normalizedPhone[1:],  // Add country code without +
		}

		err := h.db.QueryRow(query, formats[0], formats[1], formats[2], formats[3]).Scan(&targetUserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   fmt.Sprintf("User not found with phone number: %s (tried formats: %v)", req.TargetPhone, formats),
			})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Either targetUserId or targetPhone must be provided",
		})
		return
	}

	// Generate unique request ID
	requestID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour) // Expires in 24 hours

	// Insert money request into database
	query := `
		INSERT INTO money_requests (id, requester_id, target_user_id, amount, reason, request_type, status, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)
	`

	_, err := h.db.Exec(query, requestID, userID, targetUserID, req.Amount, req.Reason, req.RequestType, time.Now(), expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send money request: " + err.Error(),
		})
		return
	}

	// Create in-app notification for target user
	notificationID := uuid.New().String()
	notificationQuery := `
		INSERT INTO notifications (id, user_id, type, title, message, data, created_at, is_read)
		VALUES (?, ?, 'money_request', ?, ?, ?, ?, false)
	`

	// Get requester name for notification
	var requesterName string
	err = h.db.QueryRow("SELECT CONCAT(COALESCE(first_name, ''), ' ', COALESCE(last_name, '')) FROM users WHERE id = ?", userID).Scan(&requesterName)
	if err != nil {
		requesterName = "Someone"
	}

	notificationTitle := "Money Request"
	notificationMessage := fmt.Sprintf("%s has requested KES %.2f from you", requesterName, req.Amount)
	notificationData := fmt.Sprintf(`{"requestId": "%s", "amount": %.2f, "reason": "%s", "requesterId": "%s"}`, requestID, req.Amount, req.Reason, userID)

	_, err = h.db.Exec(notificationQuery, notificationID, targetUserID, notificationTitle, notificationMessage, notificationData, time.Now())
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to create notification: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Money request sent successfully",
		"data": gin.H{
			"requestId":    requestID,
			"targetUserId": targetUserID,
		},
	})
}

// GetRecentContacts returns users who have previously sent money to the current user
func (h *MoneyRequestHandlers) GetRecentContacts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Query for users who have sent money to this user before
	// Check both recipient_id and initiated_by fields for transaction history
	query := `
		SELECT DISTINCT
			u.id,
			u.first_name,
			u.last_name,
			u.phone,
			u.email,
			COALESCE(MAX(t.amount), 1000.0) as last_amount,
			COALESCE(MAX(t.created_at), datetime('now', '-1 day')) as last_transaction_date
		FROM users u
		LEFT JOIN transactions t ON (u.id = t.recipient_id OR u.id = t.initiated_by)
		WHERE u.id != ?
			AND (t.recipient_id = ? OR t.initiated_by != ? OR t.id IS NULL)
			AND (t.type IN ('transfer', 'payment', 'deposit') OR t.id IS NULL)
			AND (t.status = 'completed' OR t.id IS NULL)
		GROUP BY u.id, u.first_name, u.last_name, u.phone, u.email
		ORDER BY last_transaction_date DESC
		LIMIT 5
	`

	rows, err := h.db.Query(query, userID, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch recent contacts: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var contacts []gin.H
	for rows.Next() {
		var contact struct {
			ID                  string    `db:"id"`
			FirstName           *string   `db:"first_name"`
			LastName            *string   `db:"last_name"`
			Phone               *string   `db:"phone"`
			Email               *string   `db:"email"`
			LastAmount          float64   `db:"last_amount"`
			LastTransactionDate time.Time `db:"last_transaction_date"`
		}

		err := rows.Scan(
			&contact.ID,
			&contact.FirstName,
			&contact.LastName,
			&contact.Phone,
			&contact.Email,
			&contact.LastAmount,
			&contact.LastTransactionDate,
		)
		if err != nil {
			continue
		}

		contacts = append(contacts, gin.H{
			"id":                  contact.ID,
			"first_name":          contact.FirstName,
			"last_name":           contact.LastName,
			"phone":               contact.Phone,
			"email":               contact.Email,
			"lastAmount":          contact.LastAmount,
			"lastTransactionDate": contact.LastTransactionDate,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    contacts,
	})
}
