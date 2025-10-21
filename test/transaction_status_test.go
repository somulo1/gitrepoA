package test

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"vaultke-backend/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

// updateTransactionStatus is the function we're testing
func updateTransactionStatus(db *sql.DB, transactionID string, status models.TransactionStatus) error {
	updateQuery := "UPDATE transactions SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	result, err := db.Exec(updateQuery, status, transactionID)
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

func TestUpdateTransactionStatus(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create transactions table
	createTableSQL := `
	CREATE TABLE transactions (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		amount REAL NOT NULL,
		currency TEXT DEFAULT 'KES',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("Failed to create transactions table: %v", err)
	}

	// Insert a test transaction
	testTransactionID := "test-txn-123"
	insertSQL := `
	INSERT INTO transactions (id, status, amount, currency)
	VALUES (?, ?, ?, ?)`

	_, err = db.Exec(insertSQL, testTransactionID, models.TransactionStatusPending, 1000.0, "KES")
	if err != nil {
		t.Fatalf("Failed to insert test transaction: %v", err)
	}

	// Test 1: Update existing transaction status
	t.Run("Update existing transaction", func(t *testing.T) {
		err := updateTransactionStatus(db, testTransactionID, models.TransactionStatusCompleted)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify the status was updated
		var status string
		var updatedAt time.Time
		err = db.QueryRow("SELECT status, updated_at FROM transactions WHERE id = ?", testTransactionID).Scan(&status, &updatedAt)
		if err != nil {
			t.Fatalf("Failed to query updated transaction: %v", err)
		}

		if status != string(models.TransactionStatusCompleted) {
			t.Errorf("Expected status %s, got %s", models.TransactionStatusCompleted, status)
		}

		// Check that updated_at was changed (should be very recent)
		if time.Since(updatedAt) > time.Minute {
			t.Errorf("updated_at was not updated recently: %v", updatedAt)
		}
	})

	// Test 2: Update non-existent transaction
	t.Run("Update non-existent transaction", func(t *testing.T) {
		err := updateTransactionStatus(db, "non-existent-txn", models.TransactionStatusFailed)
		if err == nil {
			t.Error("Expected error for non-existent transaction, got nil")
		}

		expectedError := "transaction not found: non-existent-txn"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	// Test 3: Update to different statuses
	t.Run("Update to different statuses", func(t *testing.T) {
		statuses := []models.TransactionStatus{
			models.TransactionStatusFailed,
			models.TransactionStatusCancelled,
			models.TransactionStatusProcessing,
			models.TransactionStatusPending,
		}

		for _, status := range statuses {
			err := updateTransactionStatus(db, testTransactionID, status)
			if err != nil {
				t.Errorf("Failed to update to status %s: %v", status, err)
			}

			// Verify the status was updated
			var currentStatus string
			err = db.QueryRow("SELECT status FROM transactions WHERE id = ?", testTransactionID).Scan(&currentStatus)
			if err != nil {
				t.Fatalf("Failed to query transaction status: %v", err)
			}

			if currentStatus != string(status) {
				t.Errorf("Expected status %s, got %s", status, currentStatus)
			}
		}
	})
}
