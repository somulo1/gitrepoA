package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetNotifications retrieves all types of notifications for the authenticated user
func GetNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get all notifications from different sources
	allNotifications := []map[string]interface{}{}

	// Get deleted virtual notification IDs for filtering
	deletedVirtualNotifications := getDeletedVirtualNotificationIDs(db.(*sql.DB), userID)

	// 1. Get system notifications
	systemNotifications, err := getSystemNotifications(db.(*sql.DB), userID)
	if err == nil {
		allNotifications = append(allNotifications, systemNotifications...)
	}

	// 2. Get chama invitations
	invitationNotifications, err := getChamaInvitationNotifications(db.(*sql.DB), userID)
	if err == nil {
		// Filter out deleted virtual notifications
		filteredInvitations := filterDeletedNotifications(invitationNotifications, deletedVirtualNotifications)
		allNotifications = append(allNotifications, filteredInvitations...)
	}

	// 3. Get meeting notifications
	meetingNotifications, err := getMeetingNotifications(db.(*sql.DB), userID)
	if err == nil {
		// Filter out deleted virtual notifications
		filteredMeetings := filterDeletedNotifications(meetingNotifications, deletedVirtualNotifications)
		allNotifications = append(allNotifications, filteredMeetings...)
	}

	// 4. Get financial notifications
	financialNotifications, err := getFinancialNotifications(db.(*sql.DB), userID)
	if err == nil {
		// Filter out deleted virtual notifications
		filteredFinancial := filterDeletedNotifications(financialNotifications, deletedVirtualNotifications)
		allNotifications = append(allNotifications, filteredFinancial...)
	}

	// 5. Get chama activity notifications
	chamaNotifications, err := getChamaActivityNotifications(db.(*sql.DB), userID)
	if err == nil {
		// Filter out deleted virtual notifications
		filteredChama := filterDeletedNotifications(chamaNotifications, deletedVirtualNotifications)
		allNotifications = append(allNotifications, filteredChama...)
	}

	// 6. Get support request notifications
	supportNotifications, err := getSupportRequestNotifications(db.(*sql.DB), userID)
	if err == nil {
		// Filter out deleted virtual notifications
		filteredSupport := filterDeletedNotifications(supportNotifications, deletedVirtualNotifications)
		allNotifications = append(allNotifications, filteredSupport...)
	}

	// Sort all notifications by created_at (most recent first)
	sortNotificationsByDate(allNotifications)

	// Apply pagination
	totalCount := len(allNotifications)
	start := offset
	end := offset + limit
	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	paginatedNotifications := allNotifications[start:end]

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    paginatedNotifications,
		"meta": map[string]interface{}{
			"total":  totalCount,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetUnreadNotificationCount returns the count of unread notifications for the authenticated user
func GetUnreadNotificationCount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// For now, return a simple count from the notifications table
	// This can be enhanced later to include counts from other notification sources
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = ? AND is_read = false
	`

	var count int
	err := db.(*sql.DB).QueryRow(query, userID).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get unread notification count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"count": count,
		},
	})
}

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	notificationID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Handle different notification sources
	if handleSpecialNotificationRead(db.(*sql.DB), notificationID, userID) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Notification marked as read",
		})
		return
	}

	// Update notification in the notifications table
	query := `
		UPDATE notifications
		SET is_read = true
		WHERE id = ? AND user_id = ?
	`

	result, err := db.(*sql.DB).Exec(query, notificationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to mark notification as read",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Notification not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification marked as read",
	})
}

// MarkAllNotificationsAsRead marks all notifications as read for a user
func MarkAllNotificationsAsRead(c *gin.Context) {
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Update all notifications for the user in the notifications table
	query := `
		UPDATE notifications
		SET is_read = true
		WHERE user_id = ? AND is_read = false
	`

	result, err := db.(*sql.DB).Exec(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to mark notifications as read",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	// For virtual notifications (chama activities, meetings, etc.), we'll just return success
	// since they don't need persistent read status tracking for now
	totalMarked := rowsAffected

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("All notifications marked as read (updated %d system notifications)", totalMarked),
	})
}

// DeleteNotification deletes a notification
func DeleteNotification(c *gin.Context) {
	userID := c.GetString("userID")
	notificationID := c.Param("id")

	fmt.Printf("🗑️ DELETE NOTIFICATION: userID=%s, notificationID=%s\n", userID, notificationID)

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// First, check if this notification exists in the database
	var count int
	checkQuery := "SELECT COUNT(*) FROM notifications WHERE id = ? AND user_id = ?"
	err := db.(*sql.DB).QueryRow(checkQuery, notificationID, userID).Scan(&count)
	if err != nil {
		fmt.Printf("🗑️ Error checking notification existence: %v\n", err)
	} else {
		fmt.Printf("🗑️ Notification count in DB: %d\n", count)
	}

	// Handle different notification sources
	if handleSpecialNotificationDelete(db.(*sql.DB), notificationID, userID) {
		fmt.Printf("🗑️ Successfully handled as special notification\n")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Notification deleted successfully",
			"data":    nil, // Include data field for consistency with frontend expectations
		})
		return
	}

	// Delete notification from the notifications table
	query := `
		DELETE FROM notifications
		WHERE id = ? AND user_id = ?
	`

	fmt.Printf("🗑️ Attempting to delete from notifications table with query: %s\n", query)
	fmt.Printf("🗑️ Parameters: notificationID=%s, userID=%s\n", notificationID, userID)

	result, err := db.(*sql.DB).Exec(query, notificationID, userID)
	if err != nil {
		fmt.Printf("🗑️ Error deleting notification from database: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete notification from database",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("🗑️ Database deletion completed - Rows affected: %d\n", rowsAffected)

	if rowsAffected == 0 {
		fmt.Printf("🗑️ No rows affected - notification not found in database\n")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Notification not found in database",
		})
		return
	}

	fmt.Printf("🗑️ Notification successfully deleted from database\n")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification deleted successfully",
		"data":    nil, // Include data field for consistency with frontend expectations
	})
}

// Helper functions to get different types of notifications

// getSystemNotifications retrieves system notifications from the notifications table
func getSystemNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, user_id, title, message, type, data, is_read, created_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var notification struct {
			ID        string         `json:"id"`
			UserID    string         `json:"userId"`
			Title     string         `json:"title"`
			Message   string         `json:"message"`
			Type      string         `json:"type"`
			Data      sql.NullString `json:"data"`
			IsRead    bool           `json:"isRead"`
			CreatedAt string         `json:"createdAt"`
		}

		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Title,
			&notification.Message,
			&notification.Type,
			&notification.Data,
			&notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			continue
		}

		notificationMap := map[string]interface{}{
			"id":        notification.ID,
			"userId":    notification.UserID,
			"title":     notification.Title,
			"message":   notification.Message,
			"type":      notification.Type,
			"isRead":    notification.IsRead,
			"createdAt": notification.CreatedAt,
			"source":    "system",
		}

		if notification.Data.Valid {
			notificationMap["data"] = notification.Data.String
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// getChamaInvitationNotifications retrieves chama invitation notifications
func getChamaInvitationNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	// Get user's email first
	var userEmail string
	err := db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			ci.id, ci.chama_id, ci.email, ci.message, ci.created_at, ci.expires_at,
			c.name as chama_name, c.description as chama_description,
			c.contribution_amount, c.contribution_frequency,
			u.first_name as inviter_first_name, u.last_name as inviter_last_name
		FROM chama_invitations ci
		INNER JOIN chamas c ON ci.chama_id = c.id
		INNER JOIN users u ON ci.inviter_id = u.id
		WHERE ci.email = ? AND ci.status = 'pending' AND ci.expires_at > ?
		ORDER BY ci.created_at DESC
	`

	rows, err := db.Query(query, userEmail, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, email, chamaName, chamaDescription          string
			contributionFrequency, inviterFirstName, inviterLastName string
			createdAt, expiresAt                                     string
			message                                                  *string
			contributionAmount                                       float64
		)

		err := rows.Scan(
			&id, &chamaID, &email, &message, &createdAt, &expiresAt,
			&chamaName, &chamaDescription, &contributionAmount, &contributionFrequency,
			&inviterFirstName, &inviterLastName,
		)
		if err != nil {
			continue
		}

		inviterName := fmt.Sprintf("%s %s", inviterFirstName, inviterLastName)
		title := fmt.Sprintf("Chama Invitation: %s", chamaName)
		messageText := fmt.Sprintf("%s invited you to join %s", inviterName, chamaName)
		if message != nil && *message != "" {
			messageText = *message
		}

		notificationMap := map[string]interface{}{
			"id":        id,
			"userId":    userID,
			"title":     title,
			"message":   messageText,
			"type":      "chama_invitation",
			"isRead":    false, // Invitations are always unread until responded
			"createdAt": createdAt,
			"source":    "chama_invitation",
			"data": map[string]interface{}{
				"invitationId":          id,
				"chamaId":               chamaID,
				"chamaName":             chamaName,
				"chamaDescription":      chamaDescription,
				"contributionAmount":    contributionAmount,
				"contributionFrequency": contributionFrequency,
				"inviterName":           inviterName,
				"expiresAt":             expiresAt,
			},
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// getMeetingNotifications retrieves meeting-related notifications
func getMeetingNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			m.id, m.chama_id, m.title, m.description, m.scheduled_at,
			m.meeting_type, m.status, m.created_at,
			c.name as chama_name,
			u.first_name as creator_first_name, u.last_name as creator_last_name
		FROM meetings m
		INNER JOIN chamas c ON m.chama_id = c.id
		INNER JOIN users u ON m.created_by = u.id
		INNER JOIN chama_members cm ON c.id = cm.chama_id
		WHERE cm.user_id = ? AND cm.is_active = true
		AND (
			(m.status = 'scheduled' AND m.scheduled_at > datetime('now', '-1 day'))
			OR (m.status = 'active')
			OR (m.created_at > datetime('now', '-7 days'))
		)
		ORDER BY m.scheduled_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, title, meetingType, status, createdAt string
			chamaName, creatorFirstName, creatorLastName       string
			description, scheduledAt                           *string
		)

		err := rows.Scan(
			&id, &chamaID, &title, &description, &scheduledAt,
			&meetingType, &status, &createdAt,
			&chamaName, &creatorFirstName, &creatorLastName,
		)
		if err != nil {
			continue
		}

		creatorName := fmt.Sprintf("%s %s", creatorFirstName, creatorLastName)

		var notificationTitle, messageText, notificationType string

		switch status {
		case "scheduled":
			notificationTitle = fmt.Sprintf("Upcoming Meeting: %s", title)
			messageText = fmt.Sprintf("Meeting '%s' in %s is scheduled", title, chamaName)
			notificationType = "meeting_scheduled"
		case "active":
			notificationTitle = fmt.Sprintf("Meeting Started: %s", title)
			messageText = fmt.Sprintf("Meeting '%s' in %s has started", title, chamaName)
			notificationType = "meeting_started"
		default:
			notificationTitle = fmt.Sprintf("Meeting Created: %s", title)
			messageText = fmt.Sprintf("%s created a new meeting '%s'  in %s", creatorName, title, chamaName)
			notificationType = "meeting_created"
		}

		notificationMap := map[string]interface{}{
			"id":        fmt.Sprintf("meeting_%s", id),
			"userId":    userID,
			"title":     notificationTitle,
			"message":   messageText,
			"type":      notificationType,
			"isRead":    false,
			"createdAt": createdAt,
			"source":    "meeting",
			"data": map[string]interface{}{
				"meetingId":    id,
				"chamaId":      chamaID,
				"chamaName":    chamaName,
				"meetingTitle": title,
				"description":  description,
				"scheduledAt":  scheduledAt,
				"meetingType":  meetingType,
				"status":       status,
				"creatorName":  creatorName,
			},
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// getFinancialNotifications retrieves financial-related notifications
func getFinancialNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	var notifications []map[string]interface{}

	// 1. Get loan notifications (applications, approvals, disbursements)
	loanNotifications, err := getLoanNotifications(db, userID)
	if err == nil {
		notifications = append(notifications, loanNotifications...)
	}

	// 2. Get welfare request notifications
	welfareNotifications, err := getWelfareNotifications(db, userID)
	if err == nil {
		notifications = append(notifications, welfareNotifications...)
	}

	// 3. Get transaction notifications (contributions, payments)
	transactionNotifications, err := getTransactionNotifications(db, userID)
	if err == nil {
		notifications = append(notifications, transactionNotifications...)
	}

	return notifications, nil
}

// getLoanNotifications retrieves loan-related notifications
func getLoanNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			l.id, l.chama_id, l.applicant_id, l.amount, l.purpose,
			l.status, l.created_at, l.approved_at, l.disbursed_at,
			c.name as chama_name,
			u.first_name as applicant_first_name, u.last_name as applicant_last_name
		FROM loans l
		INNER JOIN chamas c ON l.chama_id = c.id
		INNER JOIN users u ON l.applicant_id = u.id
		INNER JOIN chama_members cm ON c.id = cm.chama_id
		WHERE (cm.user_id = ? OR l.applicant_id = ?) AND cm.is_active = true
		AND l.created_at > datetime('now', '-30 days')
		ORDER BY l.created_at DESC
	`

	rows, err := db.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, applicantID, purpose, status, createdAt string
			chamaName, applicantFirstName, applicantLastName     string
			amount                                               float64
			approvedAt, disbursedAt                              *string
		)

		err := rows.Scan(
			&id, &chamaID, &applicantID, &amount, &purpose,
			&status, &createdAt, &approvedAt, &disbursedAt,
			&chamaName, &applicantFirstName, &applicantLastName,
		)
		if err != nil {
			continue
		}

		applicantName := fmt.Sprintf("%s %s", applicantFirstName, applicantLastName)

		var notificationTitle, messageText, notificationType string

		if applicantID == userID {
			// Notifications for the loan applicant
			switch status {
			case "pending":
				notificationTitle = "Loan Application Submitted"
				messageText = fmt.Sprintf("Your loan application for KES %.2f in %s is under review", amount, chamaName)
				notificationType = "loan_application_submitted"
			case "approved":
				notificationTitle = "Loan Application Approved"
				messageText = fmt.Sprintf("Your loan application for KES %.2f in %s has been approved", amount, chamaName)
				notificationType = "loan_approved"
			case "disbursed":
				notificationTitle = "Loan Disbursed"
				messageText = fmt.Sprintf("Your loan of KES %.2f from %s has been disbursed", amount, chamaName)
				notificationType = "loan_disbursed"
			case "rejected":
				notificationTitle = "Loan Application Rejected"
				messageText = fmt.Sprintf("Your loan application for KES %.2f in %s was rejected", amount, chamaName)
				notificationType = "loan_rejected"
			default:
				continue
			}
		} else {
			// Notifications for other chama members
			switch status {
			case "pending":
				notificationTitle = "New Loan Application"
				messageText = fmt.Sprintf("%s applied for a loan of KES %.2f in %s", applicantName, amount, chamaName)
				notificationType = "loan_application_new"
			case "approved":
				notificationTitle = "Loan Application Approved"
				messageText = fmt.Sprintf("%s's loan application for KES %.2f in %s was approved", applicantName, amount, chamaName)
				notificationType = "loan_approved_member"
			default:
				continue
			}
		}

		notificationMap := map[string]interface{}{
			"id":        fmt.Sprintf("loan_%s", id),
			"userId":    userID,
			"title":     notificationTitle,
			"message":   messageText,
			"type":      notificationType,
			"isRead":    false,
			"createdAt": createdAt,
			"source":    "loan",
			"data": map[string]interface{}{
				"loanId":        id,
				"chamaId":       chamaID,
				"chamaName":     chamaName,
				"applicantName": applicantName,
				"amount":        amount,
				"purpose":       purpose,
				"status":        status,
				"approvedAt":    approvedAt,
				"disbursedAt":   disbursedAt,
			},
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// getWelfareNotifications retrieves welfare-related notifications
func getWelfareNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			wr.id, wr.chama_id, wr.beneficiary_id, wr.amount, wr.reason,
			wr.status, wr.created_at, wr.approved_at,
			c.name as chama_name,
			u.first_name as beneficiary_first_name, u.last_name as beneficiary_last_name,
			creator.first_name as creator_first_name, creator.last_name as creator_last_name
		FROM welfare_requests wr
		INNER JOIN chamas c ON wr.chama_id = c.id
		INNER JOIN users u ON wr.beneficiary_id = u.id
		INNER JOIN users creator ON wr.created_by = creator.id
		INNER JOIN chama_members cm ON c.id = cm.chama_id
		WHERE (cm.user_id = ? OR wr.beneficiary_id = ? OR wr.created_by = ?) AND cm.is_active = true
		AND wr.created_at > datetime('now', '-30 days')
		ORDER BY wr.created_at DESC
	`

	rows, err := db.Query(query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, beneficiaryID, reason, status, createdAt string
			chamaName, beneficiaryFirstName, beneficiaryLastName  string
			creatorFirstName, creatorLastName                     string
			amount                                                float64
			approvedAt                                            *string
		)

		err := rows.Scan(
			&id, &chamaID, &beneficiaryID, &amount, &reason,
			&status, &createdAt, &approvedAt,
			&chamaName, &beneficiaryFirstName, &beneficiaryLastName,
			&creatorFirstName, &creatorLastName,
		)
		if err != nil {
			continue
		}

		beneficiaryName := fmt.Sprintf("%s %s", beneficiaryFirstName, beneficiaryLastName)
		creatorName := fmt.Sprintf("%s %s", creatorFirstName, creatorLastName)

		var notificationTitle, messageText, notificationType string

		if beneficiaryID == userID {
			// Notifications for the beneficiary
			switch status {
			case "pending":
				notificationTitle = "Welfare Request Created"
				messageText = fmt.Sprintf("A welfare request for KES %.2f has been created for you in %s", amount, chamaName)
				notificationType = "welfare_request_created"
			case "approved":
				notificationTitle = "Welfare Request Approved"
				messageText = fmt.Sprintf("Your welfare request for KES %.2f in %s has been approved", amount, chamaName)
				notificationType = "welfare_approved"
			case "rejected":
				notificationTitle = "Welfare Request Rejected"
				messageText = fmt.Sprintf("Your welfare request for KES %.2f in %s was rejected", amount, chamaName)
				notificationType = "welfare_rejected"
			default:
				continue
			}
		} else {
			// Notifications for other chama members
			switch status {
			case "pending":
				notificationTitle = "New Welfare Request"
				messageText = fmt.Sprintf("%s created a welfare request for %s (KES %.2f) in %s", creatorName, beneficiaryName, amount, chamaName)
				notificationType = "welfare_request_new"
			case "approved":
				notificationTitle = "Welfare Request Approved"
				messageText = fmt.Sprintf("Welfare request for %s (KES %.2f) in %s was approved", beneficiaryName, amount, chamaName)
				notificationType = "welfare_approved_member"
			default:
				continue
			}
		}

		notificationMap := map[string]interface{}{
			"id":        fmt.Sprintf("welfare_%s", id),
			"userId":    userID,
			"title":     notificationTitle,
			"message":   messageText,
			"type":      notificationType,
			"isRead":    false,
			"createdAt": createdAt,
			"source":    "welfare",
			"data": map[string]interface{}{
				"welfareId":       id,
				"chamaId":         chamaID,
				"chamaName":       chamaName,
				"beneficiaryName": beneficiaryName,
				"creatorName":     creatorName,
				"amount":          amount,
				"reason":          reason,
				"status":          status,
				"approvedAt":      approvedAt,
			},
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// getTransactionNotifications retrieves transaction-related notifications
func getTransactionNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			t.id, t.chama_id, t.user_id, t.amount, t.type, t.description,
			t.status, t.created_at,
			c.name as chama_name,
			u.first_name as user_first_name, u.last_name as user_last_name
		FROM transactions t
		INNER JOIN chamas c ON t.chama_id = c.id
		INNER JOIN users u ON t.user_id = u.id
		INNER JOIN chama_members cm ON c.id = cm.chama_id
		WHERE cm.user_id = ? AND cm.is_active = true
		AND t.created_at > datetime('now', '-7 days')
		AND t.type IN ('contribution', 'welfare_contribution', 'loan_payment')
		ORDER BY t.created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, transactionUserID, transactionType, status, createdAt string
			chamaName, userFirstName, userLastName                             string
			amount                                                             float64
			description                                                        *string
		)

		err := rows.Scan(
			&id, &chamaID, &transactionUserID, &amount, &transactionType, &description,
			&status, &createdAt,
			&chamaName, &userFirstName, &userLastName,
		)
		if err != nil {
			continue
		}

		userName := fmt.Sprintf("%s %s", userFirstName, userLastName)

		var notificationTitle, messageText, notificationType string

		if transactionUserID == userID {
			// Notifications for the transaction creator
			switch transactionType {
			case "contribution":
				notificationTitle = "Contribution Recorded"
				messageText = fmt.Sprintf("Your contribution of KES %.2f to %s has been recorded", amount, chamaName)
				notificationType = "contribution_recorded"
			case "welfare_contribution":
				notificationTitle = "Welfare Contribution Recorded"
				messageText = fmt.Sprintf("Your welfare contribution of KES %.2f to %s has been recorded", amount, chamaName)
				notificationType = "welfare_contribution_recorded"
			case "loan_payment":
				notificationTitle = "Loan Payment Recorded"
				messageText = fmt.Sprintf("Your loan payment of KES %.2f to %s has been recorded", amount, chamaName)
				notificationType = "loan_payment_recorded"
			default:
				continue
			}
		} else {
			// Notifications for other chama members (only for significant transactions)
			if amount >= 1000 { // Only notify for transactions >= 1000 KES
				switch transactionType {
				case "contribution":
					notificationTitle = "Member Contribution"
					messageText = fmt.Sprintf("%s made a contribution of KES %.2f to %s", userName, amount, chamaName)
					notificationType = "member_contribution"
				case "welfare_contribution":
					notificationTitle = "Welfare Contribution"
					messageText = fmt.Sprintf("%s made a welfare contribution of KES %.2f to %s", userName, amount, chamaName)
					notificationType = "member_welfare_contribution"
				default:
					continue
				}
			} else {
				continue
			}
		}

		notificationMap := map[string]interface{}{
			"id":        fmt.Sprintf("transaction_%s", id),
			"userId":    userID,
			"title":     notificationTitle,
			"message":   messageText,
			"type":      notificationType,
			"isRead":    false,
			"createdAt": createdAt,
			"source":    "transaction",
			"data": map[string]interface{}{
				"transactionId":   id,
				"chamaId":         chamaID,
				"chamaName":       chamaName,
				"userName":        userName,
				"amount":          amount,
				"transactionType": transactionType,
				"description":     description,
				"status":          status,
			},
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// getChamaActivityNotifications retrieves chama activity notifications (member joins, role changes, etc.)
func getChamaActivityNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			cm.id, cm.chama_id, cm.user_id, cm.role, cm.joined_at,
			c.name as chama_name,
			u.first_name as user_first_name, u.last_name as user_last_name
		FROM chama_members cm
		INNER JOIN chamas c ON cm.chama_id = c.id
		INNER JOIN users u ON cm.user_id = u.id
		INNER JOIN chama_members my_membership ON c.id = my_membership.chama_id
		WHERE my_membership.user_id = ? AND my_membership.is_active = true
		AND cm.user_id != ? -- Don't notify about own activities
		AND cm.joined_at > datetime('now', '-7 days')
		AND cm.is_active = true
		ORDER BY cm.joined_at DESC
	`

	rows, err := db.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, memberUserID, role, joinedAt string
			chamaName, userFirstName, userLastName    string
		)

		err := rows.Scan(
			&id, &chamaID, &memberUserID, &role, &joinedAt,
			&chamaName, &userFirstName, &userLastName,
		)
		if err != nil {
			continue
		}

		userName := fmt.Sprintf("%s %s", userFirstName, userLastName)

		notificationTitle := "New Member Joined"
		messageText := fmt.Sprintf("%s joined %s", userName, chamaName)
		if role != "member" {
			messageText = fmt.Sprintf("%s joined %s as %s", userName, chamaName, role)
		}

		notificationMap := map[string]interface{}{
			"id":        fmt.Sprintf("chama_activity_%s", id),
			"userId":    userID,
			"title":     notificationTitle,
			"message":   messageText,
			"type":      "member_joined",
			"isRead":    false,
			"createdAt": joinedAt,
			"source":    "chama_activity",
			"data": map[string]interface{}{
				"chamaId":    chamaID,
				"chamaName":  chamaName,
				"memberName": userName,
				"memberRole": role,
				"joinedAt":   joinedAt,
			},
		}

		notifications = append(notifications, notificationMap)
	}

	return notifications, nil
}

// sortNotificationsByDate sorts notifications by created_at in descending order (most recent first)
func sortNotificationsByDate(notifications []map[string]interface{}) {
	// Simple bubble sort for small datasets
	n := len(notifications)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			date1, ok1 := notifications[j]["createdAt"].(string)
			date2, ok2 := notifications[j+1]["createdAt"].(string)

			if ok1 && ok2 {
				// Parse dates and compare
				time1, err1 := time.Parse("2006-01-02 15:04:05", date1)
				time2, err2 := time.Parse("2006-01-02 15:04:05", date2)

				if err1 == nil && err2 == nil && time1.Before(time2) {
					// Swap if j is older than j+1
					notifications[j], notifications[j+1] = notifications[j+1], notifications[j]
				}
			}
		}
	}
}

// handleSpecialNotificationRead handles marking special notification types as read
func handleSpecialNotificationRead(db *sql.DB, notificationID, userID string) bool {
	// Handle prefixed notification IDs from aggregated notifications

	// Check for chama activity notifications
	if strings.HasPrefix(notificationID, "chama_activity_") {
		// These are virtual notifications based on chama member activities
		// We don't need to store read status for these, just return success
		return true
	}

	// Check for meeting notifications
	if strings.HasPrefix(notificationID, "meeting_") {
		// These are virtual notifications based on meetings
		// We don't need to store read status for these, just return success
		return true
	}

	// Check for loan notifications
	if strings.HasPrefix(notificationID, "loan_") {
		// These are virtual notifications based on loans
		// We don't need to store read status for these, just return success
		return true
	}

	// Check for welfare notifications
	if strings.HasPrefix(notificationID, "welfare_") {
		// These are virtual notifications based on welfare requests
		// We don't need to store read status for these, just return success
		return true
	}

	// Check for transaction notifications
	if strings.HasPrefix(notificationID, "transaction_") {
		// These are virtual notifications based on transactions
		// We don't need to store read status for these, just return success
		return true
	}

	// Check if this is a chama invitation notification
	if len(notificationID) > 0 {
		// Chama invitations are handled by accept/reject, so we consider them "read" when accessed
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM chama_invitations WHERE id = ?", notificationID).Scan(&count)
		if err == nil && count > 0 {
			// This is a chama invitation, consider it handled
			return true
		}
	}

	return false
}

// handleSpecialNotificationDelete handles deleting special notification types
func handleSpecialNotificationDelete(db *sql.DB, notificationID, userID string) bool {
	fmt.Printf("🗑️ SPECIAL DELETE: Checking notification ID: %s\n", notificationID)

	// First, check if this notification actually exists in the database
	// If it exists, we should delete it normally, not treat it as virtual
	var count int
	checkQuery := "SELECT COUNT(*) FROM notifications WHERE id = ? AND user_id = ?"
	err := db.QueryRow(checkQuery, notificationID, userID).Scan(&count)
	if err == nil && count > 0 {
		fmt.Printf("🗑️ SPECIAL DELETE: Notification exists in database (count=%d), allowing normal deletion\n", count)
		return false // Let normal deletion process handle it
	} else {
		fmt.Printf("🗑️ SPECIAL DELETE: Notification not found in database (count=%d, err=%v), checking for virtual notification patterns\n", count, err)
	}

	// Handle prefixed notification IDs from aggregated notifications
	// These are virtual notifications that don't exist in the main notifications table
	// We'll mark them as "deleted" by creating a deletion record or just return success

	// Check for chama activity notifications
	if strings.HasPrefix(notificationID, "chama_activity_") {
		fmt.Printf("🗑️ SPECIAL DELETE: Virtual chama activity notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "chama_activity")
	}

	// Check for meeting notifications (only if they don't exist in database)
	if strings.HasPrefix(notificationID, "meeting_") {
		fmt.Printf("🗑️ SPECIAL DELETE: Virtual meeting notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "meeting")
	}

	// Check for loan notifications (only if they don't exist in database)
	if strings.HasPrefix(notificationID, "loan_") {
		fmt.Printf("🗑️ SPECIAL DELETE: Virtual loan notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "loan")
	}

	// Check for welfare notifications (only if they don't exist in database)
	if strings.HasPrefix(notificationID, "welfare_") {
		fmt.Printf("🗑️ SPECIAL DELETE: Virtual welfare notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "welfare")
	}

	// Check for transaction notifications (only if they don't exist in database)
	if strings.HasPrefix(notificationID, "transaction_") {
		fmt.Printf("🗑️ SPECIAL DELETE: Virtual transaction notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "transaction")
	}

	// Check for support request notifications
	if strings.HasPrefix(notificationID, "support_update_") {
		fmt.Printf("🗑️ SPECIAL DELETE: Support request notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "support_update")
	}

	// Check for new support request notifications (for admins)
	if strings.HasPrefix(notificationID, "support_new_") {
		fmt.Printf("🗑️ SPECIAL DELETE: New support request notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "support_new")
	}

	// Check for timestamp-based notification IDs (format: YYYYMMDDHHMMSS or YYYYMMDDHHMMSS-XXXXX)
	// These are often generated notifications that might not be in the main notifications table
	if matched, _ := regexp.MatchString(`^\d{14}(-[a-zA-Z0-9]+)?$`, notificationID); matched {
		fmt.Printf("🗑️ SPECIAL DELETE: Virtual timestamp-based notification - storing deletion record\n")
		return storeVirtualNotificationDeletion(db, userID, notificationID, "timestamp_based")
	}

	// Check if this is a chama invitation notification
	if len(notificationID) > 0 {
		// For chama invitations, try to delete from chama_invitations table
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM chama_invitations WHERE id = ?", notificationID).Scan(&count)
		if err == nil && count > 0 {
			fmt.Printf("🗑️ SPECIAL DELETE: Chama invitation found - attempting to delete\n")

			// Actually delete the chama invitation
			deleteQuery := "DELETE FROM chama_invitations WHERE id = ? AND invited_email = (SELECT email FROM users WHERE id = ?)"
			result, err := db.Exec(deleteQuery, notificationID, userID)
			if err != nil {
				fmt.Printf("🗑️ SPECIAL DELETE: Failed to delete chama invitation: %v\n", err)
				return false
			}

			rowsAffected, _ := result.RowsAffected()
			fmt.Printf("🗑️ SPECIAL DELETE: Chama invitation deleted, rows affected: %d\n", rowsAffected)
			return rowsAffected > 0
		}
	}

	// If we reach here, the notification doesn't exist in the database and doesn't match known patterns
	// This could be a stale/cached notification that was already deleted or a virtual notification
	// we don't recognize. For better UX, we'll treat it as successfully deleted.
	fmt.Printf("🗑️ SPECIAL DELETE: Unknown notification pattern, treating as virtual notification\n")
	return storeVirtualNotificationDeletion(db, userID, notificationID, "unknown")
}

// storeVirtualNotificationDeletion stores a record that a virtual notification was deleted
func storeVirtualNotificationDeletion(db *sql.DB, userID, notificationID, notificationType string) bool {
	// Create a table to track deleted virtual notifications if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS deleted_virtual_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			notification_id TEXT NOT NULL,
			notification_type TEXT NOT NULL,
			deleted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, notification_id)
		)
	`

	_, err := db.Exec(createTableQuery)
	if err != nil {
		fmt.Printf("🗑️ Failed to create deleted_virtual_notifications table: %v\n", err)
		// Even if table creation fails, we can still return success for virtual notifications
		return true
	}

	// Insert deletion record
	insertQuery := `
		INSERT OR REPLACE INTO deleted_virtual_notifications
		(user_id, notification_id, notification_type)
		VALUES (?, ?, ?)
	`

	_, err = db.Exec(insertQuery, userID, notificationID, notificationType)
	if err != nil {
		fmt.Printf("🗑️ Failed to store virtual notification deletion: %v\n", err)
		// Even if storage fails, we can still return success for virtual notifications
		return true
	}

	fmt.Printf("🗑️ Virtual notification deletion stored successfully\n")
	return true
}

// getDeletedVirtualNotificationIDs retrieves the IDs of deleted virtual notifications for a user
func getDeletedVirtualNotificationIDs(db *sql.DB, userID string) map[string]bool {
	deletedIDs := make(map[string]bool)

	// Create the table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS deleted_virtual_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			notification_id TEXT NOT NULL,
			notification_type TEXT NOT NULL,
			deleted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, notification_id)
		)
	`

	_, err := db.Exec(createTableQuery)
	if err != nil {
		fmt.Printf("🗑️ Failed to create deleted_virtual_notifications table: %v\n", err)
		return deletedIDs
	}

	query := `
		SELECT notification_id
		FROM deleted_virtual_notifications
		WHERE user_id = ?
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		fmt.Printf("🗑️ Failed to get deleted virtual notifications: %v\n", err)
		return deletedIDs
	}
	defer rows.Close()

	for rows.Next() {
		var notificationID string
		if err := rows.Scan(&notificationID); err == nil {
			deletedIDs[notificationID] = true
		}
	}

	return deletedIDs
}

// getSupportRequestNotifications retrieves support request related notifications
func getSupportRequestNotifications(db *sql.DB, userID string) ([]map[string]interface{}, error) {
	var notifications []map[string]interface{}

	// Get user role to determine what notifications to show
	var userRole string
	err := db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
	if err != nil {
		return notifications, err
	}

	if userRole == "admin" {
		// Admins get notifications about new support requests
		query := `
			SELECT
				sr.id, sr.user_id, sr.category, sr.subject, sr.priority, sr.created_at,
				u.first_name, u.last_name, u.email
			FROM support_requests sr
			LEFT JOIN users u ON sr.user_id = u.id
			WHERE sr.created_at >= datetime('now', '-7 days')
			AND sr.status = 'open'
			ORDER BY sr.created_at DESC
		`

		rows, err := db.Query(query)
		if err != nil {
			return notifications, err
		}
		defer rows.Close()

		for rows.Next() {
			var requestID, requestUserID, category, subject, priority, createdAt string
			var firstName, lastName, email *string

			err := rows.Scan(&requestID, &requestUserID, &category, &subject, &priority, &createdAt, &firstName, &lastName, &email)
			if err != nil {
				continue
			}

			// Create virtual notification ID
			notificationID := fmt.Sprintf("support_new_%s", requestID)

			userName := "Unknown User"
			if firstName != nil && lastName != nil {
				userName = fmt.Sprintf("%s %s", *firstName, *lastName)
			} else if email != nil {
				userName = *email
			}

			notification := map[string]interface{}{
				"id":         notificationID,
				"type":       "new_support_request",
				"title":      "New Support Request",
				"message":    fmt.Sprintf("New %s support request from %s: %s", category, userName, subject),
				"data":       fmt.Sprintf(`{"supportRequestId": "%s", "category": "%s", "priority": "%s"}`, requestID, category, priority),
				"is_read":    false,
				"created_at": createdAt,
				"user_id":    userID,
				"is_virtual": true,
			}

			notifications = append(notifications, notification)
		}
	} else {
		// Regular users get notifications about updates to their support requests
		query := `
			SELECT
				sr.id, sr.category, sr.subject, sr.description, sr.status, sr.priority,
				sr.updated_at, sr.admin_notes, sr.created_at
			FROM support_requests sr
			WHERE sr.user_id = ?
			AND sr.updated_at >= datetime('now', '-30 days')
			AND sr.status != 'open'
			ORDER BY sr.updated_at DESC
		`

		rows, err := db.Query(query, userID)
		if err != nil {
			return notifications, err
		}
		defer rows.Close()

		for rows.Next() {
			var requestID, category, subject, description, status, priority, updatedAt, createdAt string
			var adminNotes *string

			err := rows.Scan(&requestID, &category, &subject, &description, &status, &priority, &updatedAt, &adminNotes, &createdAt)
			if err != nil {
				continue
			}

			// Create virtual notification ID
			notificationID := fmt.Sprintf("support_update_%s_%s", requestID, status)

			// Create detailed status message
			statusMessage := getStatusDisplayText(status)

			// Create comprehensive message with status and details
			message := fmt.Sprintf("Your %s support request has been %s", category, statusMessage)

			// Add subject for context
			if subject != "" {
				message += fmt.Sprintf("\n\nSubject: %s", subject)
			}

			// Add current status
			message += fmt.Sprintf("\nStatus: %s", strings.ToUpper(status))

			// Add admin notes if available
			if adminNotes != nil && *adminNotes != "" && *adminNotes != "Status updated to "+status+" via quick action" {
				message += fmt.Sprintf("\n\nAdmin Response: %s", *adminNotes)
			}

			// Add priority if high or urgent
			if priority == "high" || priority == "urgent" {
				message += fmt.Sprintf("\nPriority: %s", strings.ToUpper(priority))
			}

			notification := map[string]interface{}{
				"id":         notificationID,
				"type":       "support_update",
				"title":      fmt.Sprintf("Support Request %s", strings.Title(statusMessage)),
				"message":    message,
				"data":       fmt.Sprintf(`{"supportRequestId": "%s", "status": "%s", "category": "%s", "priority": "%s", "subject": "%s"}`, requestID, status, category, priority, subject),
				"is_read":    false,
				"created_at": updatedAt,
				"user_id":    userID,
				"is_virtual": true,
			}

			notifications = append(notifications, notification)
		}
	}

	return notifications, nil
}

// getStatusDisplayText converts status to user-friendly text
func getStatusDisplayText(status string) string {
	switch status {
	case "in_progress":
		return "being reviewed"
	case "resolved":
		return "resolved"
	case "closed":
		return "closed"
	case "rejected":
		return "declined"
	default:
		return "updated"
	}
}

// filterDeletedNotifications filters out notifications that have been deleted
func filterDeletedNotifications(notifications []map[string]interface{}, deletedIDs map[string]bool) []map[string]interface{} {
	filtered := []map[string]interface{}{}

	for _, notification := range notifications {
		if id, ok := notification["id"].(string); ok {
			if !deletedIDs[id] {
				filtered = append(filtered, notification)
			}
		} else {
			// If ID is not a string or doesn't exist, include the notification
			filtered = append(filtered, notification)
		}
	}

	return filtered
}

// AcceptChamaInvitation handles accepting a chama invitation
func AcceptChamaInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	invitationID := c.Param("id")
	if invitationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invitation ID is required",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get user's email
	var userEmail string
	err := db.(*sql.DB).QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user email",
		})
		return
	}

	// Verify invitation exists and belongs to user
	var invitation struct {
		ID        string `json:"id"`
		ChamaID   string `json:"chamaId"`
		Email     string `json:"email"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expiresAt"`
	}

	query := `
		SELECT id, chama_id, email, status, expires_at
		FROM chama_invitations
		WHERE id = ? AND email = ? AND status = 'pending'
	`

	err = db.(*sql.DB).QueryRow(query, invitationID, userEmail).Scan(
		&invitation.ID, &invitation.ChamaID, &invitation.Email,
		&invitation.Status, &invitation.ExpiresAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Invitation not found or already processed",
		})
		return
	}

	// Check if invitation has expired
	expiresAt, err := time.Parse("2006-01-02 15:04:05", invitation.ExpiresAt)
	if err != nil || time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invitation has expired",
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

	// Update invitation status
	_, err = tx.Exec("UPDATE chama_invitations SET status = 'accepted', responded_at = ? WHERE id = ?",
		time.Now(), invitationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update invitation",
		})
		return
	}

	// Add user to chama
	_, err = tx.Exec(`
		INSERT INTO chama_members (id, chama_id, user_id, role, joined_at, is_active)
		VALUES (?, ?, ?, 'member', ?, true)
	`, fmt.Sprintf("cm_%d", time.Now().UnixNano()), invitation.ChamaID, userID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add user to chama",
		})
		return
	}

	// Update chama member count
	_, err = tx.Exec("UPDATE chamas SET current_members = current_members + 1, updated_at = ? WHERE id = ?",
		time.Now(), invitation.ChamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update chama member count",
		})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to commit transaction",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invitation accepted successfully",
	})
}

// SendSystemNotification creates a system notification
func SendSystemNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var notificationData struct {
		Message     string                 `json:"message" binding:"required"`
		Type        string                 `json:"type"`
		Data        map[string]interface{} `json:"data"`
		ChamaID     string                 `json:"chamaId"`
		RecipientID string                 `json:"recipientId"`
	}

	if err := c.ShouldBindJSON(&notificationData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid notification data",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Set default type if not provided
	if notificationData.Type == "" {
		notificationData.Type = "system"
	}

	// Notifications table should already exist from migrations
	// No need to create it here

	// Create notification record
	query := `
		INSERT INTO notifications (user_id, title, message, type, priority, category, data, is_read, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'normal', 'system', ?, false, ?, ?)
	`

	// Convert data map to JSON string
	var dataStr sql.NullString
	if notificationData.Data != nil {
		dataBytes, _ := json.Marshal(notificationData.Data)
		dataStr.String = string(dataBytes)
		dataStr.Valid = true
	}

	// Determine recipient - if specific recipient provided, use that, otherwise use current user
	recipientID := userID
	if notificationData.RecipientID != "" {
		recipientID = notificationData.RecipientID
	}

	// Extract title from message or use default
	title := "System Notification"
	if len(notificationData.Message) > 50 {
		title = notificationData.Message[:47] + "..."
	} else {
		title = notificationData.Message
	}

	now := time.Now()
	result, err := db.(*sql.DB).Exec(query, recipientID, title, notificationData.Message, notificationData.Type, dataStr, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create system notification",
		})
		return
	}

	// Get the auto-generated ID
	notificationID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get notification ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "System notification sent successfully",
		"data": gin.H{
			"notificationId": fmt.Sprintf("%d", notificationID),
		},
	})
}

// RejectChamaInvitation handles rejecting a chama invitation
func RejectChamaInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	invitationID := c.Param("id")
	if invitationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invitation ID is required",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get user's email
	var userEmail string
	err := db.(*sql.DB).QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user email",
		})
		return
	}

	// Update invitation status
	result, err := db.(*sql.DB).Exec(`
		UPDATE chama_invitations
		SET status = 'rejected', responded_at = ?
		WHERE id = ? AND email = ? AND status = 'pending'
	`, time.Now(), invitationID, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to reject invitation",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Invitation not found or already processed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invitation rejected successfully",
	})
}
