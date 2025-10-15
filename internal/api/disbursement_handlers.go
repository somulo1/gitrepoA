package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DisbursementHandlers handles disbursement-related API endpoints
type DisbursementHandlers struct {
	db *sql.DB
}

// NewDisbursementHandlers creates a new instance of DisbursementHandlers
func NewDisbursementHandlers(db *sql.DB) *DisbursementHandlers {
	return &DisbursementHandlers{db: db}
}

// GetDisbursementBatches retrieves disbursement batches for a chama
func (h *DisbursementHandlers) GetDisbursementBatches(c *gin.Context) {
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

	// Query disbursement batches
	query := `
		SELECT db.id, db.chama_id, db.batch_type, db.title, db.description,
			   db.total_amount, db.total_recipients, db.status, db.scheduled_date,
			   db.processed_date, db.created_at, db.updated_at,
			   u1.first_name || ' ' || u1.last_name as initiated_by_name,
			   COALESCE(u2.first_name || ' ' || u2.last_name, '') as approved_by_name
		FROM disbursement_batches db
		LEFT JOIN users u1 ON db.initiated_by = u1.id
		LEFT JOIN users u2 ON db.approved_by = u2.id
		WHERE db.chama_id = ?
		ORDER BY db.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := h.db.Query(query, chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve disbursement batches",
		})
		return
	}
	defer rows.Close()

	var batches []map[string]interface{}
	for rows.Next() {
		var id, chamaId, batchType, title, description, status, createdAt, updatedAt, initiatedBy string
		var totalAmount float64
		var totalRecipients int
		var scheduledDate, processedDate sql.NullTime
		var approvedByName sql.NullString

		err := rows.Scan(
			&id, &chamaId, &batchType, &title,
			&description, &totalAmount, &totalRecipients,
			&status, &scheduledDate, &processedDate, &createdAt,
			&updatedAt, &initiatedBy, &approvedByName,
		)
		if err != nil {
			continue
		}

		batch := map[string]interface{}{
			"id":              id,
			"chamaId":         chamaId,
			"batchType":       batchType,
			"title":           title,
			"description":     description,
			"totalAmount":     totalAmount,
			"totalRecipients": totalRecipients,
			"status":          status,
			"createdAt":       createdAt,
			"updatedAt":       updatedAt,
			"initiatedBy":     initiatedBy,
		}

		if scheduledDate.Valid {
			batch["scheduledDate"] = scheduledDate.Time
		}
		if processedDate.Valid {
			batch["processedDate"] = processedDate.Time
		}
		if approvedByName.Valid {
			batch["approvedBy"] = approvedByName.String
		}

		batches = append(batches, batch)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    batches,
		"count":   len(batches),
	})
}

// GetTransparencyLog retrieves financial transparency log for a chama
func (h *DisbursementHandlers) GetTransparencyLog(c *gin.Context) {
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

	// Query transparency log
	query := `
		SELECT ftl.id, ftl.chama_id, ftl.activity_type, ftl.title, ftl.description,
			   ftl.amount, ftl.currency, ftl.transaction_type, ftl.reference_id,
			   ftl.reference_type, ftl.affected_members, ftl.visibility,
			   ftl.created_at, u.first_name || ' ' || u.last_name as performed_by_name
		FROM financial_transparency_log ftl
		LEFT JOIN users u ON ftl.performed_by = u.id
		WHERE ftl.chama_id = ?
		ORDER BY ftl.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := h.db.Query(query, chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve transparency log",
		})
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, chamaId, activityType, title, description, currency, transactionType, visibility, createdAt, performedBy string
		var amount float64
		var referenceID, referenceType, affectedMembers sql.NullString

		err := rows.Scan(
			&id, &chamaId, &activityType, &title,
			&description, &amount, &currency, &transactionType,
			&referenceID, &referenceType, &affectedMembers, &visibility,
			&createdAt, &performedBy,
		)
		if err != nil {
			continue
		}

		log := map[string]interface{}{
			"id":              id,
			"chamaId":         chamaId,
			"activityType":    activityType,
			"title":           title,
			"description":     description,
			"amount":          amount,
			"currency":        currency,
			"transactionType": transactionType,
			"visibility":      visibility,
			"createdAt":       createdAt,
			"performedBy":     performedBy,
		}

		if referenceID.Valid {
			log["referenceId"] = referenceID.String
		}
		if referenceType.Valid {
			log["referenceType"] = referenceType.String
		}
		if affectedMembers.Valid {
			log["affectedMembers"] = affectedMembers.String
		}

		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
		"count":   len(logs),
	})
}

// ProcessDisbursementBatch processes a disbursement batch
func (h *DisbursementHandlers) ProcessDisbursementBatch(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	batchID := c.Param("batchId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// For now, just mark as processed
	query := `UPDATE disbursement_batches SET status = 'completed', processed_date = CURRENT_TIMESTAMP WHERE id = ? AND chama_id = ?`
	_, err := h.db.Exec(query, batchID, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to process disbursement batch",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Disbursement batch processed successfully",
	})
}

// ApproveDisbursementBatch approves a disbursement batch
func (h *DisbursementHandlers) ApproveDisbursementBatch(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	batchID := c.Param("batchId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Update batch status to approved
	query := `UPDATE disbursement_batches SET status = 'approved', approved_by = ? WHERE id = ? AND chama_id = ?`
	_, err := h.db.Exec(query, userID, batchID, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to approve disbursement batch",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Disbursement batch approved successfully",
	})
}
