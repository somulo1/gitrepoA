package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"vaultke-backend/test/helpers"
)

func TestDatabaseTables(t *testing.T) {
	testDB := helpers.SetupTestDatabase()
	defer testDB.Close()

	// Check if reminders table exists
	var tableName string
	err := testDB.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='reminders'").Scan(&tableName)
	
	if err != nil {
		t.Logf("Error checking reminders table: %v", err)
		
		// List all tables
		rows, err := testDB.DB.Query("SELECT name FROM sqlite_master WHERE type='table'")
		if err != nil {
			t.Fatalf("Failed to list tables: %v", err)
		}
		defer rows.Close()
		
		t.Log("Available tables:")
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				continue
			}
			t.Logf("  - %s", name)
		}
		
		t.Fatal("Reminders table does not exist")
	}
	
	assert.Equal(t, "reminders", tableName)
	t.Log("Reminders table exists successfully")
}
