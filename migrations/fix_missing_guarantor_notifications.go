package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// createNotification inserts a notification with all required fields
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

func main() {
	// Open database connection
	db, err := sql.Open("sqlite3", "./vaultke.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	fmt.Println("🔄 Starting migration: Fix missing guarantor notifications")

	// Find all guarantors that don't have corresponding notifications
	rows, err := db.Query(`
		SELECT g.id, g.loan_id, g.user_id, g.amount, l.amount as loan_amount, l.purpose, l.borrower_id
		FROM guarantors g
		JOIN loans l ON g.loan_id = l.id
		WHERE g.status = 'pending'
		AND NOT EXISTS (
			SELECT 1 FROM notifications n
			WHERE n.user_id = g.user_id
			AND n.type = 'guarantor_request'
			AND n.data LIKE '%' || g.id || '%'
		)
	`)
	if err != nil {
		log.Fatal("Failed to query guarantors:", err)
	}
	defer rows.Close()

	createdCount := 0
	for rows.Next() {
		var guarantorID, loanID, userID string
		var guarantorAmount, loanAmount float64
		var purpose, borrowerID string

		err := rows.Scan(&guarantorID, &loanID, &userID, &guarantorAmount, &loanAmount, &purpose, &borrowerID)
		if err != nil {
			fmt.Printf("Error scanning guarantor: %v\n", err)
			continue
		}

		// Create notification for this guarantor
		notificationID := fmt.Sprintf("notif-migration-%d", time.Now().UnixNano())
		data := fmt.Sprintf(`{"loan_id": "%s", "amount": %.2f, "purpose": "%s", "requester_id": "%s", "guarantor_id": "%s"}`,
			loanID, loanAmount, purpose, borrowerID, guarantorID)

		err = createNotification(db, notificationID, userID, "guarantor_request",
			"Guarantor Request",
			fmt.Sprintf("You have been requested to guarantee a loan of KES %.2f", loanAmount),
			data, "loan", nil) // Use nil for reference_id since it's TEXT but schema expects INTEGER

		if err != nil {
			fmt.Printf("❌ Failed to create notification for guarantor %s: %v\n", guarantorID, err)
		} else {
			fmt.Printf("✅ Created notification for guarantor %s (user: %s)\n", guarantorID, userID)
			createdCount++
		}
	}

	fmt.Printf("🎉 Migration completed! Created %d missing guarantor notifications\n", createdCount)
}
