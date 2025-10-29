package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FinancialReportsHandlers handles financial reports API endpoints
type FinancialReportsHandlers struct {
	db *sql.DB
}

// NewFinancialReportsHandlers creates a new instance of FinancialReportsHandlers
func NewFinancialReportsHandlers(db *sql.DB) *FinancialReportsHandlers {
	return &FinancialReportsHandlers{db: db}
}

// GetFinancialReports retrieves financial reports for a chama
func (h *FinancialReportsHandlers) GetFinancialReports(c *gin.Context) {
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

	// Get pagination parameters with reasonable defaults
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

	// Enforce reasonable limits to prevent memory issues
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Query financial reports
	query := `
		SELECT fr.id, fr.chama_id, fr.report_type, fr.title, fr.description,
			   fr.report_period_start, fr.report_period_end, fr.file_path,
			   fr.file_size, fr.status, fr.download_count, fr.is_public,
			   fr.created_at, fr.updated_at,
			   u.first_name || ' ' || u.last_name as generated_by_name
		FROM financial_reports fr
		LEFT JOIN users u ON fr.generated_by = u.id
		WHERE fr.chama_id = ?
		ORDER BY fr.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := h.db.Query(query, chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve financial reports",
		})
		return
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var id, chamaId, reportType, title, description, status, createdAt, updatedAt, generatedBy string
		var downloadCount int
		var isPublic bool
		var periodStart, periodEnd sql.NullTime
		var filePath sql.NullString
		var fileSize sql.NullInt64

		err := rows.Scan(
			&id, &chamaId, &reportType, &title,
			&description, &periodStart, &periodEnd, &filePath,
			&fileSize, &status, &downloadCount, &isPublic,
			&createdAt, &updatedAt, &generatedBy,
		)
		if err != nil {
			continue
		}

		report := map[string]interface{}{
			"id":            id,
			"chamaId":       chamaId,
			"reportType":    reportType,
			"title":         title,
			"description":   description,
			"status":        status,
			"downloadCount": downloadCount,
			"isPublic":      isPublic,
			"createdAt":     createdAt,
			"updatedAt":     updatedAt,
			"generatedBy":   generatedBy,
		}

		if periodStart.Valid {
			report["reportPeriodStart"] = periodStart.Time
		}
		if periodEnd.Valid {
			report["reportPeriodEnd"] = periodEnd.Time
		}
		if filePath.Valid {
			report["filePath"] = filePath.String
		}
		if fileSize.Valid {
			report["fileSize"] = float64(fileSize.Int64) / (1024 * 1024) // Convert to MB
		}

		reports = append(reports, report)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    reports,
		"count":   len(reports),
	})
}

// GenerateFinancialReport generates a new financial report
func (h *FinancialReportsHandlers) GenerateFinancialReport(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		ReportType        string    `json:"reportType" binding:"required"`
		Title             string    `json:"title"`
		Description       string    `json:"description"`
		ReportPeriodStart time.Time `json:"reportPeriodStart"`
		ReportPeriodEnd   time.Time `json:"reportPeriodEnd"`
		IsPublic          bool      `json:"isPublic"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Generate default title if not provided
	if req.Title == "" {
		switch req.ReportType {
		case "monthly_statement":
			req.Title = "Monthly Financial Statement"
		case "dividend_report":
			req.Title = "Dividend Distribution Report"
		case "transparency_report":
			req.Title = "Financial Transparency Report"
		default:
			req.Title = "Financial Report"
		}
	}

	// Create report record
	reportID := uuid.New().String()
	query := `
		INSERT INTO financial_reports (
			id, chama_id, report_type, title, description,
			report_period_start, report_period_end, generated_by,
			status, is_public, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'generating', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := h.db.Exec(query, reportID, chamaID, req.ReportType, req.Title,
		req.Description, req.ReportPeriodStart, req.ReportPeriodEnd,
		userID, req.IsPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create report",
		})
		return
	}

	// In a real implementation, you would trigger background report generation here
	// For now, we'll just mark it as ready after a short delay
	// Use a timeout context to prevent goroutine leaks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in report generation goroutine: %v", r)
			}
		}()

		select {
		case <-ctx.Done():
			log.Printf("Report generation cancelled for report %s: %v", reportID, ctx.Err())
			// Update status to failed
			updateQuery := `UPDATE financial_reports SET status = 'failed' WHERE id = ?`
			h.db.Exec(updateQuery, reportID)
			return
		default:
			time.Sleep(2 * time.Second)
			updateQuery := `UPDATE financial_reports SET status = 'ready', file_size = 2048000 WHERE id = ?`
			h.db.Exec(updateQuery, reportID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":         reportID,
			"reportType": req.ReportType,
			"title":      req.Title,
			"status":     "generating",
		},
		"message": "Report generation started",
	})
}

// DownloadFinancialReport downloads a financial report
func (h *FinancialReportsHandlers) DownloadFinancialReport(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	reportID := c.Param("reportId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if report exists and is ready
	var status string
	query := `SELECT status FROM financial_reports WHERE id = ? AND chama_id = ?`
	err := h.db.QueryRow(query, reportID, chamaID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Report not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to check report status",
			})
		}
		return
	}

	if status != "ready" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Report is not ready for download",
		})
		return
	}

	// Increment download count
	updateQuery := `UPDATE financial_reports SET download_count = download_count + 1 WHERE id = ?`
	h.db.Exec(updateQuery, reportID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report download started",
		"data": gin.H{
			"downloadUrl": "/api/v1/reports/" + reportID + "/download",
		},
	})
}
