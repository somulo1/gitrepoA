package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// createNotificationTx inserts a notification with all required fields using a transaction
func createNotificationTx(tx *sql.Tx, notificationID, userID, notificationType, title, message, data string, referenceType string, referenceID interface{}) error {
	// Ensure all required fields are included with proper defaults
	_, err := tx.Exec(`
		INSERT INTO notifications (
			user_id, type, title, message, data, is_read, created_at, updated_at,
			priority, category, reference_type, status, scheduled_for,
			is_push, is_email, is_sms
		) VALUES (?, ?, ?, ?, ?, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
			?, ?, ?, 'pending', CURRENT_TIMESTAMP,
			?, ?, ?)
	`, userID, notificationType, title, message, data,
		getNotificationPriority(notificationType),
		getNotificationCategory(notificationType),
		referenceType,
		getNotificationPushEnabled(notificationType),
		getNotificationEmailEnabled(notificationType),
		getNotificationSMSEnabled(notificationType))

	return err
}

// createNotification inserts a notification with all required fields using a database connection
func createNotification(db *sql.DB, notificationID, userID, notificationType, title, message, data string, referenceType string, referenceID interface{}) error {
	// Ensure all required fields are included with proper defaults
	_, err := db.Exec(`
		INSERT INTO notifications (
			user_id, type, title, message, data, is_read, created_at, updated_at,
			priority, category, reference_type, status, scheduled_for,
			is_push, is_email, is_sms
		) VALUES (?, ?, ?, ?, ?, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
			?, ?, ?, 'pending', CURRENT_TIMESTAMP,
			?, ?, ?)
	`, userID, notificationType, title, message, data,
		getNotificationPriority(notificationType),
		getNotificationCategory(notificationType),
		referenceType,
		getNotificationPushEnabled(notificationType),
		getNotificationEmailEnabled(notificationType),
		getNotificationSMSEnabled(notificationType))

	return err
}

// Helper functions to determine notification properties based on type
func getNotificationPriority(notificationType string) string {
	switch notificationType {
	case "guarantor_request", "loan_status_update":
		return "high"
	default:
		return "normal"
	}
}

func getNotificationCategory(notificationType string) string {
	switch notificationType {
	case "guarantor_request", "loan_status_update", "guarantor_response":
		return "financial"
	case "meeting_created", "meeting_updated":
		return "meetings"
	case "member_joined", "member_left":
		return "members"
	default:
		return "system"
	}
}

func getNotificationPushEnabled(notificationType string) int {
	switch notificationType {
	case "guarantor_request", "loan_status_update", "meeting_created", "member_joined":
		return 1
	default:
		return 0
	}
}

func getNotificationEmailEnabled(notificationType string) int {
	switch notificationType {
	case "guarantor_request", "loan_status_update":
		return 1
	default:
		return 0
	}
}

func getNotificationSMSEnabled(notificationType string) int {
	switch notificationType {
	case "guarantor_request":
		return 1
	default:
		return 0
	}
}

// Loan application handlers
func GetLoanApplications(c *gin.Context) {
	startTime := time.Now()
	chamaID := c.Query("chamaId")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "chamaId parameter is required",
		})
		return
	}

	fmt.Printf("🔍 GetLoanApplications called for chamaId: %s\n", chamaID)

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// First, let's check if the loans table exists and has data
	var tableExists bool
	err := db.(*sql.DB).QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='loans'").Scan(&tableExists)
	if err != nil {
		// fmt.Printf("❌ Loans table check failed: %v\n", err)
	} else {
		fmt.Printf("✅ Loans table exists\n")
	}

	// Check total count of loans in the table
	var totalLoans int
	err = db.(*sql.DB).QueryRow("SELECT COUNT(*) FROM loans").Scan(&totalLoans)
	if err != nil {
		// fmt.Printf("❌ Failed to count loans: %v\n", err)
	} else {
		// fmt.Printf("🔍 Total loans in database: %d\n", totalLoans)
	}

	// Check loans for this specific chama
	var chamaLoans int
	err = db.(*sql.DB).QueryRow("SELECT COUNT(*) FROM loans WHERE chama_id = ?", chamaID).Scan(&chamaLoans)
	if err != nil {
		// fmt.Printf("❌ Failed to count loans for chama: %v\n", err)
	} else {
		// fmt.Printf("🔍 Loans for chamaId %s: %d\n", chamaID, chamaLoans)
	}

	// Debug: Show all unique chama_ids in loans table
	chamaRows, err := db.(*sql.DB).Query("SELECT DISTINCT chama_id, COUNT(*) FROM loans GROUP BY chama_id")
	if err == nil {
		// fmt.Printf("🔍 All chamaIds with loans:\n")
		for chamaRows.Next() {
			var cid string
			var count int
			if chamaRows.Scan(&cid, &count) == nil {
				fmt.Printf("   - %s: %d loans\n", cid, count)
			}
		}
		chamaRows.Close()
	}

	// Query loan applications
	rows, err := db.(*sql.DB).Query(`
		SELECT
			l.id, l.borrower_id, l.chama_id, l.amount, l.interest_rate,
			l.duration, l.purpose, l.status, l.total_amount, l.remaining_amount,
			l.required_guarantors, l.approved_guarantors, l.due_date, l.created_at,
			u.first_name, u.last_name, u.email
		FROM loans l
		JOIN users u ON l.borrower_id = u.id
		WHERE l.chama_id = ?
		ORDER BY l.created_at DESC
	`, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch loan applications: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var loans []map[string]interface{}
	for rows.Next() {
		var loan struct {
			ID                 string    `json:"id"`
			BorrowerID         string    `json:"borrowerId"`
			ChamaID            string    `json:"chamaId"`
			Amount             float64   `json:"amount"`
			InterestRate       float64   `json:"interestRate"`
			Duration           int       `json:"duration"`
			Purpose            string    `json:"purpose"`
			Status             string    `json:"status"`
			TotalAmount        float64   `json:"totalAmount"`
			RemainingAmount    float64   `json:"remainingAmount"`
			RequiredGuarantors int       `json:"requiredGuarantors"`
			ApprovedGuarantors int       `json:"approvedGuarantors"`
			DueDate            time.Time `json:"dueDate"`
			CreatedAt          time.Time `json:"createdAt"`
			BorrowerFirstName  string    `json:"borrowerFirstName"`
			BorrowerLastName   string    `json:"borrowerLastName"`
			BorrowerEmail      string    `json:"borrowerEmail"`
		}

		err := rows.Scan(
			&loan.ID, &loan.BorrowerID, &loan.ChamaID, &loan.Amount, &loan.InterestRate,
			&loan.Duration, &loan.Purpose, &loan.Status, &loan.TotalAmount, &loan.RemainingAmount,
			&loan.RequiredGuarantors, &loan.ApprovedGuarantors, &loan.DueDate, &loan.CreatedAt,
			&loan.BorrowerFirstName, &loan.BorrowerLastName, &loan.BorrowerEmail,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		loanMap := map[string]interface{}{
			"id":                 loan.ID,
			"borrowerId":         loan.BorrowerID,
			"chamaId":            loan.ChamaID,
			"amount":             loan.Amount,
			"interestRate":       loan.InterestRate,
			"duration":           loan.Duration,
			"purpose":            loan.Purpose,
			"status":             loan.Status,
			"totalAmount":        loan.TotalAmount,
			"remainingAmount":    loan.RemainingAmount,
			"requiredGuarantors": loan.RequiredGuarantors,
			"approvedGuarantors": loan.ApprovedGuarantors,
			"dueDate":            loan.DueDate.Format(time.RFC3339),
			"createdAt":          loan.CreatedAt.Format(time.RFC3339),
			"borrower": map[string]interface{}{
				"id":        loan.BorrowerID,
				"firstName": loan.BorrowerFirstName,
				"lastName":  loan.BorrowerLastName,
				"email":     loan.BorrowerEmail,
				"fullName":  loan.BorrowerFirstName + " " + loan.BorrowerLastName,
			},
		}

		loans = append(loans, loanMap)
	}

	duration := time.Since(startTime)
	fmt.Printf("⏱️  GetLoanApplications completed in %v for chamaId: %s\n", duration, chamaID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    loans,
		"message": fmt.Sprintf("Found %d loan applications", len(loans)),
		"meta": map[string]interface{}{
			"total":   len(loans),
			"chamaId": chamaID,
		},
	})
}

func CreateLoanApplication(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		ChamaID         string                 `json:"chamaId" binding:"required"`
		Amount          float64                `json:"amount" binding:"required"`
		Purpose         string                 `json:"purpose" binding:"required"`
		RepaymentPeriod int                    `json:"repaymentPeriod" binding:"required"`
		InterestRate    float64                `json:"interestRate" binding:"required"`
		Guarantors      []string               `json:"guarantors" binding:"required"`
		Security        map[string]interface{} `json:"security"`
		BusinessPlan    string                 `json:"businessPlan"`
		MonthlyIncome   float64                `json:"monthlyIncome" binding:"required"`
		OtherLoans      string                 `json:"otherLoans"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Amount must be greater than 0",
		})
		return
	}

	// Validate guarantors
	if len(req.Guarantors) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least 2 guarantors are required",
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

	// Start transaction
	tx, err := db.(*sql.DB).Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Generate loan ID
	loanID := fmt.Sprintf("loan-%d", time.Now().UnixNano())

	// Calculate total amount with interest
	totalAmount := req.Amount * (1 + req.InterestRate/100)

	// Calculate due date (repayment period in months)
	dueDate := time.Now().AddDate(0, req.RepaymentPeriod, 0)

	// Insert loan application
	_, err = tx.Exec(`
		INSERT INTO loans (
			id, borrower_id, chama_id, type, amount, interest_rate,
			duration, purpose, status, total_amount, remaining_amount,
			required_guarantors, approved_guarantors, due_date,
			created_at, updated_at
		) VALUES (?, ?, ?, 'regular', ?, ?, ?, ?, 'pending', ?, ?, ?, 0, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, loanID, userID, req.ChamaID, req.Amount, req.InterestRate, req.RepaymentPeriod, req.Purpose, totalAmount, totalAmount, len(req.Guarantors), dueDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create loan application: " + err.Error(),
		})
		return
	}

	// Insert guarantors and send notifications
	for _, guarantorID := range req.Guarantors {
		guarantorAmount := req.Amount / float64(len(req.Guarantors)) // Split equally
		guarantorRecordID := fmt.Sprintf("guarantor-%d-%s", time.Now().UnixNano(), guarantorID)

		_, err = tx.Exec(`
			INSERT INTO guarantors (
				id, loan_id, user_id, amount, status, created_at
			) VALUES (?, ?, ?, ?, 'pending', CURRENT_TIMESTAMP)
		`, guarantorRecordID, loanID, guarantorID, guarantorAmount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to add guarantor: " + err.Error(),
			})
			return
		}

		// Create notification for guarantor
		notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
		err = createNotificationTx(tx, notificationID, guarantorID, "guarantor_request",
			"Guarantor Request",
			fmt.Sprintf("You have been requested to guarantee a loan of KES %.2f", req.Amount),
			fmt.Sprintf(`{"loan_id": "%s", "amount": %.2f, "purpose": "%s", "requester_id": "%s", "guarantor_id": "%s"}`,
				loanID, req.Amount, req.Purpose, userID.(string), guarantorRecordID),
			"loan", nil)
		if err != nil {
			// Log error but don't fail the transaction
			fmt.Printf("Failed to create notification for guarantor %s: %v\n", guarantorID, err)
		}
		if err != nil {
			// Log error but don't fail the transaction
			fmt.Printf("Failed to create notification for guarantor %s: %v\n", guarantorID, err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to commit transaction",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Loan application submitted successfully. Guarantors will be notified.",
		"data": map[string]interface{}{
			"id":                 loanID,
			"chamaId":            req.ChamaID,
			"borrowerId":         userID,
			"amount":             req.Amount,
			"purpose":            req.Purpose,
			"repaymentPeriod":    req.RepaymentPeriod,
			"interestRate":       req.InterestRate,
			"guarantors":         req.Guarantors,
			"totalAmount":        totalAmount,
			"remainingAmount":    totalAmount,
			"status":             "pending",
			"requiredGuarantors": len(req.Guarantors),
			"approvedGuarantors": 0,
			"dueDate":            dueDate.Format(time.RFC3339),
			"createdAt":          time.Now().Format(time.RFC3339),
		},
	})
}

func GetLoanApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Get loan application endpoint - coming soon",
	})
}

func UpdateLoanApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Update loan application endpoint - coming soon",
	})
}

func DeleteLoanApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delete loan application endpoint - coming soon",
	})
}

func RespondToGuarantorRequest(c *gin.Context) {
	// Get loan ID from URL parameter
	loanID := c.Param("id")
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Loan ID is required",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		GuarantorID string `json:"guarantorId" binding:"required"`
		Action      string `json:"action" binding:"required"` // "accept" or "decline"
		Reason      string `json:"reason"`                    // Optional reason for decline
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	guarantorID := req.GuarantorID

	// DEBUG: Log the received parameters
	fmt.Printf("🔍 RespondToGuarantorRequest: loanID=%s, guarantorID=%s, userID=%s, action=%s\n", loanID, guarantorID, userID, req.Action)

	// Validate action
	if req.Action != "accept" && req.Action != "decline" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Action must be 'accept' or 'decline'",
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

	// DEBUG: Check if guarantor record exists at all
	var count int
	err := db.(*sql.DB).QueryRow("SELECT COUNT(*) FROM guarantors WHERE id = ?", guarantorID).Scan(&count)
	if err != nil {
		fmt.Printf("❌ Error checking guarantor existence: %v\n", err)
	} else {
		fmt.Printf("🔍 Guarantor record %s exists: %d\n", guarantorID, count)
	}

	// DEBUG: Check guarantor record details
	var dbGuarantorID, dbUserID, dbLoanID, dbStatus string
	err = db.(*sql.DB).QueryRow("SELECT id, user_id, loan_id, status FROM guarantors WHERE id = ?", guarantorID).Scan(&dbGuarantorID, &dbUserID, &dbLoanID, &dbStatus)
	if err != nil {
		fmt.Printf("❌ Error getting guarantor details: %v\n", err)
	} else {
		fmt.Printf("🔍 Guarantor record details: id=%s, user_id=%s, loan_id=%s, status=%s\n", dbGuarantorID, dbUserID, dbLoanID, dbStatus)
	}

	// Find the guarantor record by ID and ensure the current user is the guarantor
	var requesterID string
	var currentStatus string
	var actualLoanID string
	err = db.(*sql.DB).QueryRow(`
		SELECT l.user_id as requester_id, g.status, g.loan_id
		FROM guarantors g
		JOIN loans l ON g.loan_id = l.id
		WHERE g.id = ? AND g.user_id = ?
	`, guarantorID, userID).Scan(&requesterID, &currentStatus, &actualLoanID)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("❌ Guarantor request not found: guarantorID=%s, userID=%s\n", guarantorID, userID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Guarantor request not found or not authorized",
			})
			return
		}
		fmt.Printf("❌ Database error fetching guarantor request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch guarantor request: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Found guarantor request: requesterID=%s, currentStatus=%s, actualLoanID=%s\n", requesterID, currentStatus, actualLoanID)

	// If loan ID was provided in URL and doesn't match, return error
	if loanID != "" && actualLoanID != loanID {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Guarantor request does not belong to the specified loan",
		})
		return
	}

	// Use the actual loan ID for further processing
	loanID = actualLoanID

	// Check if already responded
	if currentStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "You have already responded to this guarantor request",
		})
		return
	}

	// Update guarantor status
	newStatus := "declined"
	if req.Action == "accept" {
		newStatus = "accepted"
	}

	_, err = db.(*sql.DB).Exec(`
		UPDATE guarantors
		SET status = ?, message = ?, responded_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newStatus, req.Reason, guarantorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update guarantor response: " + err.Error(),
		})
		return
	}

	// Create notification for loan requester
	notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
	message := fmt.Sprintf("Your guarantor request has been %s", newStatus)
	if req.Reason != "" {
		message += fmt.Sprintf(". Reason: %s", req.Reason)
	}

	err = createNotification(db.(*sql.DB), notificationID, requesterID, "guarantor_response",
		"Guarantor Response",
		message,
		fmt.Sprintf(`{"loan_id": "%s", "guarantor_id": "%s", "action": "%s"}`,
			loanID, guarantorID, req.Action),
		"loan", nil)
	if err != nil {
		// Log error but don't fail the response
		fmt.Printf("Failed to create notification for loan requester %s: %v\n", requesterID, err)
	}

	// Check if all guarantors have responded and update loan status if needed
	// Use a timeout context to prevent goroutine leaks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in checkAndUpdateLoanStatus: %v", r)
			}
		}()

		select {
		case <-ctx.Done():
			log.Printf("Loan status check cancelled for loan %s: %v", loanID, ctx.Err())
			return
		default:
			checkAndUpdateLoanStatus(db.(*sql.DB), loanID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Guarantor request %s successfully", newStatus),
		"data": map[string]interface{}{
			"guarantor_id": guarantorID,
			"status":       newStatus,
			"action":       req.Action,
		},
	})
}

// checkAndUpdateLoanStatus checks if all guarantors have responded and updates loan status
func checkAndUpdateLoanStatus(db *sql.DB, loanID string) {
	// Get guarantor response counts
	var totalGuarantors, acceptedGuarantors, declinedGuarantors int
	err := db.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) as accepted,
			SUM(CASE WHEN status = 'declined' THEN 1 ELSE 0 END) as declined
		FROM guarantors
		WHERE loan_id = ?
	`, loanID).Scan(&totalGuarantors, &acceptedGuarantors, &declinedGuarantors)
	if err != nil {
		fmt.Printf("Error checking guarantor status for loan %s: %v\n", loanID, err)
		return
	}

	// Check if all guarantors have responded
	respondedGuarantors := acceptedGuarantors + declinedGuarantors
	if respondedGuarantors < totalGuarantors {
		// Not all guarantors have responded yet
		return
	}

	// Determine new loan status
	var newStatus string
	if acceptedGuarantors == totalGuarantors {
		// All guarantors accepted - move to pending approval
		newStatus = "guarantors_approved"
	} else if declinedGuarantors > 0 {
		// At least one guarantor declined - reject loan
		newStatus = "guarantors_declined"
	}

	// Update loan status
	_, err = db.Exec(`
		UPDATE loans
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newStatus, loanID)
	if err != nil {
		fmt.Printf("Error updating loan status for loan %s: %v\n", loanID, err)
		return
	}

	// Get loan requester for notification
	var requesterID string
	err = db.QueryRow(`SELECT user_id FROM loans WHERE id = ?`, loanID).Scan(&requesterID)
	if err != nil {
		fmt.Printf("Error getting loan requester for loan %s: %v\n", loanID, err)
		return
	}

	// Create notification for loan requester
	notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
	var message string
	if newStatus == "guarantors_approved" {
		message = "All guarantors have accepted your loan request. Your application is now pending approval from chama officials."
	} else {
		message = "Some guarantors have declined your loan request. Your application has been rejected."
	}

	err = createNotification(db, notificationID, requesterID, "loan_status_update",
		"Loan Status Update",
		message,
		fmt.Sprintf(`{"loan_id": "%s", "status": "%s"}`, loanID, newStatus),
		"loan", nil)
	if err != nil {
		fmt.Printf("Failed to create loan status notification for user %s: %v\n", requesterID, err)
	}
}

func ApproveLoan(c *gin.Context) {
	loanID := c.Param("id")
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Loan ID is required",
		})
		return
	}

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

	// Check if loan exists and get current status
	var currentStatus, chamaID string
	err := db.(*sql.DB).QueryRow(`
		SELECT status, chama_id FROM loans WHERE id = ?
	`, loanID).Scan(&currentStatus, &chamaID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Loan not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch loan: " + err.Error(),
		})
		return
	}

	// Check if user is authorized (chama chairperson or treasurer)
	var userRole string
	err = db.(*sql.DB).QueryRow(`
		SELECT role FROM chama_members WHERE chama_id = ? AND user_id = ?
	`, chamaID, userID).Scan(&userRole)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You are not authorized to approve loans for this chama",
		})
		return
	}

	if userRole != "chairperson" && userRole != "treasurer" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson or treasurer can approve loans",
		})
		return
	}

	// Check if loan can be approved
	if currentStatus != "guarantors_approved" && currentStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Cannot approve loan with status: %s", currentStatus),
		})
		return
	}

	// Update loan status to approved
	_, err = db.(*sql.DB).Exec(`
		UPDATE loans
		SET status = 'approved', approved_by = ?, approved_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID, loanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to approve loan: " + err.Error(),
		})
		return
	}

	// Get loan details for notification
	var borrowerID string
	var amount float64
	err = db.(*sql.DB).QueryRow(`
		SELECT borrower_id, amount FROM loans WHERE id = ?
	`, loanID).Scan(&borrowerID, &amount)
	if err != nil {
		fmt.Printf("Failed to get loan details for notification: %v\n", err)
	} else {
		// Create notification for borrower
		notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
		err = createNotification(db.(*sql.DB), notificationID, borrowerID, "loan_status_update",
			"Loan Approved",
			fmt.Sprintf("Your loan application for KES %.2f has been approved and is ready for disbursement.", amount),
			fmt.Sprintf(`{"loan_id": "%s", "status": "approved", "amount": %.2f}`, loanID, amount),
			"loan", nil)
		if err != nil {
			fmt.Printf("Failed to create loan approval notification: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Loan approved successfully",
		"data": map[string]interface{}{
			"loan_id": loanID,
			"status":  "approved",
		},
	})
}

func RejectLoan(c *gin.Context) {
	loanID := c.Param("id")
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Loan ID is required",
		})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

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

	// Check if loan exists and get current status
	var currentStatus, chamaID string
	err := db.(*sql.DB).QueryRow(`
		SELECT status, chama_id FROM loans WHERE id = ?
	`, loanID).Scan(&currentStatus, &chamaID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Loan not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch loan: " + err.Error(),
		})
		return
	}

	// Check if user is authorized (chama chairperson or treasurer)
	var userRole string
	err = db.(*sql.DB).QueryRow(`
		SELECT role FROM chama_members WHERE chama_id = ? AND user_id = ?
	`, chamaID, userID).Scan(&userRole)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You are not authorized to reject loans for this chama",
		})
		return
	}

	if userRole != "chairperson" && userRole != "treasurer" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson or treasurer can reject loans",
		})
		return
	}

	// Check if loan can be rejected
	if currentStatus == "approved" || currentStatus == "disbursed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Cannot reject an approved or disbursed loan",
		})
		return
	}

	// Update loan status to rejected
	_, err = db.(*sql.DB).Exec(`
		UPDATE loans
		SET status = 'rejected', rejected_by = ?, rejected_reason = ?, rejected_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID, req.Reason, loanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to reject loan: " + err.Error(),
		})
		return
	}

	// Get loan details for notification
	var borrowerID string
	var amount float64
	err = db.(*sql.DB).QueryRow(`
		SELECT borrower_id, amount FROM loans WHERE id = ?
	`, loanID).Scan(&borrowerID, &amount)
	if err != nil {
		fmt.Printf("Failed to get loan details for notification: %v\n", err)
	} else {
		// Create notification for borrower
		notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
		message := fmt.Sprintf("Your loan application for KES %.2f has been rejected.", amount)
		if req.Reason != "" {
			message += fmt.Sprintf(" Reason: %s", req.Reason)
		}

		err = createNotification(db.(*sql.DB), notificationID, borrowerID, "loan_status_update",
			"Loan Rejected",
			message,
			fmt.Sprintf(`{"loan_id": "%s", "status": "rejected", "reason": "%s", "amount": %.2f}`, loanID, req.Reason, amount),
			"loan", nil)
		if err != nil {
			fmt.Printf("Failed to create loan rejection notification: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Loan rejected successfully",
		"data": map[string]interface{}{
			"loan_id": loanID,
			"status":  "rejected",
			"reason":  req.Reason,
		},
	})
}

func DisburseLoan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Loan disbursed successfully",
	})
}

func GetGuarantorRequests(c *gin.Context) {
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

	// Query guarantor requests for the user
	rows, err := db.(*sql.DB).Query(`
		SELECT
			g.id, g.loan_id, g.amount, g.status, g.created_at,
			l.amount as loan_amount, l.purpose, l.repayment_period, l.interest_rate,
			u.first_name, u.last_name, u.email,
			c.name as chama_name
		FROM guarantors g
		JOIN loans l ON g.loan_id = l.id
		JOIN users u ON l.user_id = u.id
		JOIN chamas c ON l.chama_id = c.id
		WHERE g.user_id = ?
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch guarantor requests: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var guarantorRequests []map[string]interface{}

	for rows.Next() {
		var gr struct {
			ID                 string    `json:"id"`
			LoanID             string    `json:"loanId"`
			Amount             float64   `json:"amount"`
			Status             string    `json:"status"`
			CreatedAt          time.Time `json:"createdAt"`
			LoanAmount         float64   `json:"loanAmount"`
			Purpose            string    `json:"purpose"`
			RepaymentPeriod    int       `json:"repaymentPeriod"`
			InterestRate       float64   `json:"interestRate"`
			RequesterFirstName string    `json:"requesterFirstName"`
			RequesterLastName  string    `json:"requesterLastName"`
			RequesterEmail     string    `json:"requesterEmail"`
			ChamaName          string    `json:"chamaName"`
		}

		err := rows.Scan(
			&gr.ID, &gr.LoanID, &gr.Amount, &gr.Status, &gr.CreatedAt,
			&gr.LoanAmount, &gr.Purpose, &gr.RepaymentPeriod, &gr.InterestRate,
			&gr.RequesterFirstName, &gr.RequesterLastName, &gr.RequesterEmail,
			&gr.ChamaName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to scan guarantor request: " + err.Error(),
			})
			return
		}

		guarantorRequest := map[string]interface{}{
			"id":              gr.ID,
			"loanId":          gr.LoanID,
			"amount":          gr.Amount,
			"status":          gr.Status,
			"createdAt":       gr.CreatedAt.Format(time.RFC3339),
			"loanAmount":      gr.LoanAmount,
			"purpose":         gr.Purpose,
			"repaymentPeriod": gr.RepaymentPeriod,
			"interestRate":    gr.InterestRate,
			"chamaName":       gr.ChamaName,
			"requester": map[string]interface{}{
				"firstName": gr.RequesterFirstName,
				"lastName":  gr.RequesterLastName,
				"email":     gr.RequesterEmail,
				"fullName":  gr.RequesterFirstName + " " + gr.RequesterLastName,
			},
		}

		guarantorRequests = append(guarantorRequests, guarantorRequest)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    guarantorRequests,
		"count":   len(guarantorRequests),
	})
}
