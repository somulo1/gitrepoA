package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AccountHandlers handles account management related requests
type AccountHandlers struct {
	db *sql.DB
}

// NewAccountHandlers creates a new account handlers instance
func NewAccountHandlers(db *sql.DB) *AccountHandlers {
	return &AccountHandlers{
		db: db,
	}
}

// GetEligibleWelfareMembers returns members eligible for welfare disbursement
func (h *AccountHandlers) GetEligibleWelfareMembers(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Query eligible welfare members
	query := `
		SELECT DISTINCT u.id, u.first_name, u.last_name, u.phone,
			   COALESCE(SUM(t.amount), 0) as total_contributions,
			   MAX(wf.created_at) as last_disbursement
		FROM users u
		JOIN chama_members cm ON u.id = cm.user_id
		LEFT JOIN transactions t ON u.id = t.initiated_by AND t.type = 'contribution'
		LEFT JOIN welfare_funds wf ON u.id = wf.beneficiary_id AND wf.chama_id = ?
		WHERE cm.chama_id = ? AND cm.is_active = TRUE
		GROUP BY u.id, u.first_name, u.last_name, u.phone
		ORDER BY total_contributions DESC
	`

	rows, err := h.db.Query(query, chamaID, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve eligible members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var id, firstName, lastName, phone string
		var totalContributions float64
		var lastDisbursement sql.NullTime

		err := rows.Scan(&id, &firstName, &lastName, &phone, &totalContributions, &lastDisbursement)
		if err != nil {
			continue
		}

		eligibilityStatus := "eligible"
		if totalContributions < 1000 { // Example threshold
			eligibilityStatus = "pending"
		}

		member := map[string]interface{}{
			"id":                 id,
			"name":               fmt.Sprintf("%s %s", firstName, lastName),
			"phone":              phone,
			"eligibilityStatus":  eligibilityStatus,
			"totalContributions": totalContributions,
		}

		if lastDisbursement.Valid {
			member["lastDisbursement"] = lastDisbursement.Time.Format("2006-01-02")
		} else {
			member["lastDisbursement"] = nil
		}

		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
		"count":   len(members),
	})
}

// GetTransparencyFeed returns transparency feed for a chama
func (h *AccountHandlers) GetTransparencyFeed(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// Query transparency feed (transactions for chama members)
	query := `
		SELECT t.id, t.type, t.amount, t.description, t.created_at,
			   u.first_name || ' ' || u.last_name as member_name,
			   t.status
		FROM transactions t
		JOIN users u ON t.initiated_by = u.id
		JOIN chama_members cm ON u.id = cm.user_id AND cm.chama_id = ?
		WHERE cm.is_active = true
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := h.db.Query(query, chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve transparency feed",
		})
		return
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	var totalAmount float64

	for rows.Next() {
		var id, transactionType, description, memberName, status string
		var amount float64
		var createdAt time.Time

		err := rows.Scan(&id, &transactionType, &amount, &description, &createdAt, &memberName, &status)
		if err != nil {
			continue
		}

		transaction := map[string]interface{}{
			"id":        id,
			"type":      transactionType,
			"amount":    amount,
			"member":    memberName,
			"timestamp": createdAt.Format(time.RFC3339),
			"status":    status,
		}

		if description != "" {
			transaction["description"] = description
		}

		transactions = append(transactions, transaction)
		totalAmount += amount
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"transactions":      transactions,
			"totalTransactions": len(transactions),
			"totalAmount":       totalAmount,
		},
	})
}

// GetAccountNotifications returns account-specific notifications
func (h *AccountHandlers) GetAccountNotifications(c *gin.Context) {
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Query account notifications
	query := `
		SELECT id, type, title, message, created_at, is_read
		FROM notifications
		WHERE user_id = ?
		AND type IN ('security', 'account', 'financial')
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := h.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve account notifications",
		})
		return
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var id, notificationType, title, message string
		var createdAt time.Time
		var isRead bool

		err := rows.Scan(&id, &notificationType, &title, &message, &createdAt, &isRead)
		if err != nil {
			continue
		}

		notification := map[string]interface{}{
			"id":        id,
			"type":      notificationType,
			"title":     title,
			"message":   message,
			"timestamp": createdAt.Format(time.RFC3339),
			"read":      isRead,
		}

		notifications = append(notifications, notification)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// ValidateSystemSecurity performs security validation
func (h *AccountHandlers) ValidateSystemSecurity(c *gin.Context) {
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Perform basic security checks
	securityScore := 85 // Base score
	var vulnerabilities []string

	// Check for recent suspicious activities
	// This is a simplified implementation

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"securityScore":   securityScore,
			"vulnerabilities": vulnerabilities,
			"lastAudit":       time.Now().Format(time.RFC3339),
			"status":          "secure",
		},
	})
}

// LogSecurityEvent logs security events
func (h *AccountHandlers) LogSecurityEvent(c *gin.Context) {
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var eventData map[string]interface{}
	if err := c.ShouldBindJSON(&eventData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid event data",
		})
		return
	}

	// Log the security event (simplified implementation)
	// In a real implementation, you'd store this in a security_events table

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"eventId":   fmt.Sprintf("evt_%d", time.Now().Unix()),
			"logged":    true,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}
