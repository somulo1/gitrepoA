package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Migration to fix existing E2EE messages
func migrateE2EEMessages(db *sql.DB) error {
	fmt.Println("🔄 Starting E2EE message migration...")

	// Get all messages that contain encrypted content
	query := `
		SELECT id, content, metadata
		FROM chat_messages
		WHERE content LIKE '%_enc_%' AND content LIKE '%=='
	`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query encrypted messages: %w", err)
	}
	defer rows.Close()

	migratedCount := 0
	for rows.Next() {
		var id string
		var content string
		var metadataStr string

		if err := rows.Scan(&id, &content, &metadataStr); err != nil {
			log.Printf("Error scanning message %s: %v", id, err)
			continue
		}

		// Parse existing metadata
		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		// Mark as encrypted and needing decryption
		metadata["encrypted"] = true
		metadata["needsDecryption"] = true
		metadata["securityLevel"] = "FALLBACK"
		metadata["migrated"] = true

		// Update the message metadata
		updatedMetadata, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("Error marshaling metadata for message %s: %v", id, err)
			continue
		}

		updateQuery := `
			UPDATE chat_messages
			SET metadata = ?
			WHERE id = ?
		`

		_, err = db.Exec(updateQuery, string(updatedMetadata), id)
		if err != nil {
			log.Printf("Error updating message %s: %v", id, err)
			continue
		}

		migratedCount++
		if migratedCount%10 == 0 {
			fmt.Printf("🔄 Migrated %d messages...\n", migratedCount)
		}
	}

	fmt.Printf("✅ Migration completed! Updated %d encrypted messages\n", migratedCount)
	return nil
}

func main() {
	// This is a standalone migration script
	// In production, this would be integrated into your migration system

	db, err := sql.Open("sqlite3", "./vaultke.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	if err := migrateE2EEMessages(db); err != nil {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("🎉 E2EE message migration completed successfully!")
}