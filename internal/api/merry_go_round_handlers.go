package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// checkAndAdvanceMerryGoRound is a utility function that can be called directly
// to check and advance a merry-go-round without requiring a Gin context
func checkAndAdvanceMerryGoRound(db *sql.DB, merryGoRoundID, chamaID, userID string) error {
	fmt.Printf("🔄 Checking if merry-go-round %s should advance...\n", merryGoRoundID)

	// Get current round information
	var currentRound, totalParticipants int
	var status string
	err := db.QueryRow(`
		SELECT current_round, total_participants, status
		FROM merry_go_rounds
		WHERE id = ? AND chama_id = ?
	`, merryGoRoundID, chamaID).Scan(&currentRound, &totalParticipants, &status)
	if err != nil {
		return fmt.Errorf("failed to get merry-go-round information: %v", err)
	}

	if status != "active" {
		return fmt.Errorf("merry-go-round is not active")
	}

	// Count contributions for the current round (contributions made TO the current recipient)
	// Exclude contributions from the recipient to themselves
	var contributionCount int
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT t.initiated_by)
		FROM transactions t
		WHERE t.type = 'contribution'
			AND json_extract(t.metadata, '$.contributionType') = 'merry-go-round'
			AND json_extract(t.metadata, '$.merryGoRoundId') = ?
			AND json_extract(t.metadata, '$.roundNumber') = ?
			AND json_extract(t.metadata, '$.chamaId') = ?
			AND t.status = 'completed'
	`, merryGoRoundID, currentRound, chamaID).Scan(&contributionCount)

	if err != nil {
		return fmt.Errorf("failed to count contributions: %v", err)
	}

	fmt.Printf("🔍 Round %d: %d/%d contributions completed\n", currentRound, contributionCount, totalParticipants-1)

	// Check if all non-recipient participants have contributed (100% completion)
	if contributionCount >= (totalParticipants - 1) {
		// Advance to next round
		nextRound := currentRound + 1

		// Check if we've completed all rounds
		if nextRound > totalParticipants {
			// Mark merry-go-round as completed
			_, err = db.Exec(`
				UPDATE merry_go_rounds
				SET status = 'completed', updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, merryGoRoundID)

			if err != nil {
				return fmt.Errorf("failed to complete merry-go-round: %v", err)
			}

			fmt.Printf("✅ Merry-go-round %s completed! All rounds finished.\n", merryGoRoundID)
			return nil
		}

		// Calculate next payout date
		var frequency string
		var startDate time.Time
		err = db.QueryRow(`
			SELECT frequency, start_date FROM merry_go_rounds WHERE id = ?
		`, merryGoRoundID).Scan(&frequency, &startDate)
		if err != nil {
			return fmt.Errorf("failed to get merry-go-round schedule: %v", err)
		}

		var nextPayoutDate time.Time
		if frequency == "weekly" {
			nextPayoutDate = startDate.AddDate(0, 0, nextRound*7)
		} else { // monthly
			nextPayoutDate = startDate.AddDate(0, nextRound, 0)
		}

		// Advance to next round
		_, err = db.Exec(`
			UPDATE merry_go_rounds
			SET current_round = ?, next_payout_date = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, nextRound, nextPayoutDate, merryGoRoundID)

		if err != nil {
			return fmt.Errorf("failed to advance to next round: %v", err)
		}

		fmt.Printf("✅ Advanced merry-go-round %s to round %d\n", merryGoRoundID, nextRound)
		return nil
	}

	// Round not yet complete
	fmt.Printf("⏳ Round %d in progress: %d/%d contributions completed\n", currentRound, contributionCount, totalParticipants)
	return nil
}

// Merry-Go-Round handlers
func GetMerryGoRounds(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	chamaID := c.Query("chamaId")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "chamaId parameter is required",
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

	// Check if user is a member of the chama
	var membershipExists bool
	err := db.(*sql.DB).QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM chama_members
			WHERE chama_id = ? AND user_id = ? AND is_active = TRUE
		)
	`, chamaID, userID).Scan(&membershipExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify chama membership",
		})
		return
	}

	if !membershipExists {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Access denied. You are not a member of this chama.",
		})
		return
	}

	// Query merry-go-rounds for the chama
	rows, err := db.(*sql.DB).Query(`
		SELECT
			mgr.id, mgr.chama_id, mgr.name, mgr.description, mgr.amount_per_round,
			mgr.frequency, mgr.total_participants, mgr.current_round, mgr.status,
			mgr.start_date, mgr.next_payout_date, mgr.created_by, mgr.created_at,
			u.first_name, u.last_name, u.email
		FROM merry_go_rounds mgr
		JOIN users u ON mgr.created_by = u.id
		WHERE mgr.chama_id = ?
		ORDER BY mgr.created_at DESC
	`, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch merry-go-rounds: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var merryGoRounds []map[string]interface{}
	for rows.Next() {
		var mgr struct {
			ID                string     `json:"id"`
			ChamaID           string     `json:"chamaId"`
			Name              string     `json:"name"`
			Description       string     `json:"description"`
			AmountPerRound    float64    `json:"amountPerRound"`
			Frequency         string     `json:"frequency"`
			TotalParticipants int        `json:"totalParticipants"`
			CurrentRound      int        `json:"currentRound"`
			Status            string     `json:"status"`
			StartDate         time.Time  `json:"startDate"`
			NextPayoutDate    *time.Time `json:"nextPayoutDate"`
			CreatedBy         string     `json:"createdBy"`
			CreatedAt         time.Time  `json:"createdAt"`
			CreatorFirstName  string     `json:"creatorFirstName"`
			CreatorLastName   string     `json:"creatorLastName"`
			CreatorEmail      string     `json:"creatorEmail"`
		}

		err := rows.Scan(
			&mgr.ID, &mgr.ChamaID, &mgr.Name, &mgr.Description, &mgr.AmountPerRound,
			&mgr.Frequency, &mgr.TotalParticipants, &mgr.CurrentRound, &mgr.Status,
			&mgr.StartDate, &mgr.NextPayoutDate, &mgr.CreatedBy, &mgr.CreatedAt,
			&mgr.CreatorFirstName, &mgr.CreatorLastName, &mgr.CreatorEmail,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		nextPayoutDateStr := ""
		if mgr.NextPayoutDate != nil {
			nextPayoutDateStr = mgr.NextPayoutDate.Format(time.RFC3339)
		}

		// For active merry-go-rounds, check contribution count and round completion
		var contributionCount int
		var totalParticipants int
		var roundComplete bool
		if mgr.Status == "active" {
			// Count all contributions for the current round (same as contribution page logic)
			err = db.(*sql.DB).QueryRow(`
				SELECT COUNT(DISTINCT t.initiated_by)
				FROM transactions t
				WHERE t.type = 'contribution'
					AND json_extract(t.metadata, '$.contributionType') = 'merry-go-round'
					AND json_extract(t.metadata, '$.merryGoRoundId') = ?
					AND json_extract(t.metadata, '$.roundNumber') = ?
					AND json_extract(t.metadata, '$.chamaId') = ?
					AND t.status = 'completed'
			`, mgr.ID, mgr.CurrentRound, mgr.ChamaID).Scan(&contributionCount)

			if err == nil {
				// Get total participants count
				err = db.(*sql.DB).QueryRow(`
					SELECT COUNT(*)
					FROM merry_go_round_participants
					WHERE merry_go_round_id = ?
				`, mgr.ID).Scan(&totalParticipants)

				if err == nil {
					// Round is complete if all non-recipient participants have contributed
					roundComplete = contributionCount >= (totalParticipants - 1)
					fmt.Printf("🎯 Round completion check for %s: %d/%d contributions needed, complete: %v\n", mgr.ID, contributionCount, totalParticipants-1, roundComplete)
				} else {
					fmt.Printf("❌ Error counting participants for %s: %v\n", mgr.ID, err)
					roundComplete = false
					totalParticipants = 0
				}
			} else {
				fmt.Printf("❌ Error checking round completion for %s: %v\n", mgr.ID, err)
				roundComplete = false
				contributionCount = 0
				totalParticipants = 0
			}
		} else {
			contributionCount = 0
			totalParticipants = 0
			roundComplete = false
		}

		// Get participants for this merry-go-round
		participantRows, err := db.(*sql.DB).Query(`
			SELECT
				mgrp.user_id, mgrp.position, mgrp.has_received,
				u.first_name, u.last_name, u.email
			FROM merry_go_round_participants mgrp
			JOIN users u ON mgrp.user_id = u.id
			WHERE mgrp.merry_go_round_id = ?
			ORDER BY mgrp.position ASC
		`, mgr.ID)

		var participants []map[string]interface{}
		if err == nil {
			defer participantRows.Close()
			for participantRows.Next() {
				var p struct {
					UserID      string `json:"userId"`
					Position    int    `json:"position"`
					HasReceived bool   `json:"hasReceived"`
					FirstName   string `json:"firstName"`
					LastName    string `json:"lastName"`
					Email       string `json:"email"`
				}

				err := participantRows.Scan(&p.UserID, &p.Position, &p.HasReceived, &p.FirstName, &p.LastName, &p.Email)
				if err == nil {
					// Determine status based on position, current round, and contribution count
					var status string
					if p.Position < mgr.CurrentRound {
						status = "completed"
					} else if p.Position == mgr.CurrentRound {
						if contributionCount >= (totalParticipants - 1) {
							status = "completed"
						} else {
							status = "current"
						}
					} else if p.Position == mgr.CurrentRound+1 && contributionCount >= (totalParticipants-1) {
						status = "current"
					} else {
						status = "pending"
					}

					participant := map[string]interface{}{
						"id":           fmt.Sprintf("%s-%d", mgr.ID, p.Position), // Unique participant ID
						"user_id":      p.UserID,
						"position":     p.Position,
						"status":       status,
						"has_received": p.HasReceived,
						"user": map[string]interface{}{
							"id":         p.UserID,
							"first_name": p.FirstName,
							"last_name":  p.LastName,
							"email":      p.Email,
							"username":   p.Email, // Use email as username fallback
							"full_name":  p.FirstName + " " + p.LastName,
						},
						"has_contributed_this_cycle": false, // Mock - would need real tracking
					}
					participants = append(participants, participant)
				}
			}
		} else {
			fmt.Printf("❌ Failed to query participants for merry-go-round %s: %v\n", mgr.ID, err)
		}

		// Calculate total payout (amount per round * number of participants)
		totalPayout := mgr.AmountPerRound * float64(len(participants))

		mgrMap := map[string]interface{}{
			"id":                 mgr.ID,
			"chamaId":            mgr.ChamaID,
			"name":               mgr.Name,
			"description":        mgr.Description,
			"amountPerRound":     mgr.AmountPerRound,
			"amount_per_round":   mgr.AmountPerRound, // Alternative field name
			"frequency":          mgr.Frequency,
			"totalParticipants":  mgr.TotalParticipants,
			"total_participants": len(participants), // Use actual participant count
			"currentRound":       mgr.CurrentRound,
			"current_round":      mgr.CurrentRound,     // Alternative field name
			"current_position":   mgr.CurrentRound - 1, // Zero-based position
			"status":             mgr.Status,
			"startDate":          mgr.StartDate.Format("2006-01-02"),
			"start_date":         mgr.StartDate.Format("2006-01-02"), // Alternative field name
			"nextPayoutDate":     nextPayoutDateStr,
			"next_payout_date":   nextPayoutDateStr, // Alternative field name
			"createdBy":          mgr.CreatedBy,
			"created_by":         mgr.CreatedBy, // Alternative field name
			"createdAt":          mgr.CreatedAt.Format(time.RFC3339),
			"created_at":         mgr.CreatedAt.Format(time.RFC3339), // Alternative field name
			"total_payout":       totalPayout,
			"members":            participants,
			"participants":       participants, // Alternative field name
			"roundComplete":      roundComplete,
			"creator": map[string]interface{}{
				"id":        mgr.CreatedBy,
				"firstName": mgr.CreatorFirstName,
				"lastName":  mgr.CreatorLastName,
				"email":     mgr.CreatorEmail,
				"fullName":  mgr.CreatorFirstName + " " + mgr.CreatorLastName,
			},
		}

		merryGoRounds = append(merryGoRounds, mgrMap)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    merryGoRounds,
		"message": fmt.Sprintf("Found %d merry-go-rounds", len(merryGoRounds)),
		"meta": map[string]interface{}{
			"total":   len(merryGoRounds),
			"chamaId": chamaID,
		},
	})
}

func CreateMerryGoRound(c *gin.Context) {
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
		ChamaID           string  `json:"chamaId" binding:"required"`
		Name              string  `json:"name" binding:"required"`
		Description       string  `json:"description"`
		AmountPerRound    float64 `json:"amountPerRound" binding:"required"`
		Frequency         string  `json:"frequency" binding:"required"`
		TotalParticipants int     `json:"totalParticipants" binding:"required"`
		StartDate         string  `json:"startDate" binding:"required"`
		Participants      []struct {
			UserID   string `json:"userId"`
			Position int    `json:"position"`
			Name     string `json:"name"`
			Email    string `json:"email"`
		} `json:"participants"`
		ParticipantOrder string `json:"participantOrder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Validate amount
	if req.AmountPerRound <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Amount per round must be greater than 0",
		})
		return
	}

	// Validate total participants
	if req.TotalParticipants < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least 2 participants are required",
		})
		return
	}

	// Validate frequency
	validFrequencies := map[string]bool{"weekly": true, "monthly": true}
	if !validFrequencies[req.Frequency] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Frequency must be 'weekly' or 'monthly'",
		})
		return
	}

	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid start date format. Use YYYY-MM-DD",
		})
		return
	}

	// Check if start date is not too far in the past (allow today and future dates)
	today := time.Now().Truncate(24 * time.Hour)
	if startDate.Before(today) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Start date cannot be in the past. Please select today or a future date.",
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

	// Check if user is a member of the chama
	var membershipExists bool
	err = db.(*sql.DB).QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM chama_members
			WHERE chama_id = ? AND user_id = ? AND is_active = TRUE
		)
	`, req.ChamaID, userID).Scan(&membershipExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify chama membership",
		})
		return
	}

	if !membershipExists {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Access denied. You are not a member of this chama.",
		})
		return
	}

	// Generate merry-go-round ID
	mgrID := fmt.Sprintf("mgr-%d", time.Now().UnixNano())

	// Calculate next payout date
	var nextPayoutDate time.Time
	if req.Frequency == "weekly" {
		nextPayoutDate = startDate.AddDate(0, 0, 7)
	} else { // monthly
		nextPayoutDate = startDate.AddDate(0, 1, 0)
	}

	// Insert merry-go-round into database
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO merry_go_rounds (
			id, chama_id, name, description, amount_per_round, frequency,
			total_participants, current_round, status, start_date, next_payout_date,
			created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 1, 'active', ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, mgrID, req.ChamaID, req.Name, req.Description, req.AmountPerRound, req.Frequency, req.TotalParticipants, startDate, nextPayoutDate, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create merry-go-round: " + err.Error(),
		})
		return
	}

	// Insert participants into merry_go_round_participants table
	if len(req.Participants) > 0 {
		fmt.Printf("🔄 Inserting %d participants into merry_go_round_participants table\n", len(req.Participants))

		for i, participant := range req.Participants {
			participantID := fmt.Sprintf("mgrp-%d-%d", time.Now().UnixNano(), i)

			fmt.Printf("📝 Adding participant: ID=%s, UserID=%s, Position=%d\n", participantID, participant.UserID, participant.Position)

			_, err = db.(*sql.DB).Exec(`
				INSERT INTO merry_go_round_participants (
					id, merry_go_round_id, user_id, position, has_received, received_at,
					total_contributed, joined_at
				) VALUES (?, ?, ?, ?, FALSE, NULL, 0, CURRENT_TIMESTAMP)
			`, participantID, mgrID, participant.UserID, participant.Position)

			if err != nil {
				fmt.Printf("❌ Failed to add participant %s (UserID: %s): %v\n", participantID, participant.UserID, err)
				// Continue with other participants instead of failing completely
			} else {
				fmt.Printf("✅ Successfully added participant %s\n", participantID)
			}
		}

		// Verify participants were inserted
		var participantCount int
		err = db.(*sql.DB).QueryRow(`
			SELECT COUNT(*) FROM merry_go_round_participants
			WHERE merry_go_round_id = ?
		`, mgrID).Scan(&participantCount)

		if err == nil {
			fmt.Printf("🔍 Verification: %d participants found in database for merry-go-round %s\n", participantCount, mgrID)
		}
	} else {
		fmt.Printf("⚠️ No participants provided in request\n")
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Merry-go-round created successfully! Members can now join.",
		"data": map[string]interface{}{
			"id":                mgrID,
			"chamaId":           req.ChamaID,
			"name":              req.Name,
			"description":       req.Description,
			"amountPerRound":    req.AmountPerRound,
			"frequency":         req.Frequency,
			"totalParticipants": req.TotalParticipants,
			"currentRound":      1,
			"status":            "active",
			"startDate":         req.StartDate,
			"nextPayoutDate":    nextPayoutDate.Format("2006-01-02"),
			"createdBy":         userID,
			"createdAt":         time.Now().Format(time.RFC3339),
		},
	})
}

func GetMerryGoRound(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Get merry-go-round endpoint - coming soon",
	})
}

func UpdateMerryGoRound(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Update merry-go-round endpoint - coming soon",
	})
}

func DeleteMerryGoRound(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delete merry-go-round endpoint - coming soon",
	})
}

func JoinMerryGoRound(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Join merry-go-round endpoint - coming soon",
	})
}

// CheckUserContributionStatus checks if a user has already contributed to the current round
func CheckUserContributionStatus(c *gin.Context) {
	fmt.Printf("🔍 [CONTRIBUTION STATUS] CheckUserContributionStatus handler called\n")

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	chamaID := c.Param("chamaId")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
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

	// Check if a specific roundId is provided via query parameter
	roundID := c.Query("roundId")
	var merryGoRoundID string
	var currentRound int
	var amountPerRound float64
	var status string

	if roundID != "" {
		// Use the specific merry-go-round
		fmt.Printf("🎯 Using specific merry-go-round: %s\n", roundID)
		err := db.(*sql.DB).QueryRow(`
			SELECT id, current_round, amount_per_round, status
			FROM merry_go_rounds
			WHERE id = ? AND chama_id = ?
		`, roundID, chamaID).Scan(&merryGoRoundID, &currentRound, &amountPerRound, &status)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"error":   "Merry-go-round not found",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to get merry-go-round information",
			})
			return
		}
	} else {
		// Get the active merry-go-round for this chama (fallback behavior)
		fmt.Printf("🎯 Using active merry-go-round for chama: %s\n", chamaID)
		err := db.(*sql.DB).QueryRow(`
			SELECT id, current_round, amount_per_round, status
			FROM merry_go_rounds
			WHERE chama_id = ? AND status = 'active'
			ORDER BY created_at DESC
			LIMIT 1
		`, chamaID).Scan(&merryGoRoundID, &currentRound, &amountPerRound, &status)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"hasActiveMerryGoRound": false,
						"hasContributed":        false,
						"canContribute":         false,
						"message":               "No active merry-go-round found",
					},
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to check merry-go-round status",
			})
			return
		}
	}

	// Check if user has already contributed to this round with timeout
	var hasContributed bool
	queryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.(*sql.DB).QueryRowContext(queryCtx, `
		SELECT EXISTS(
			SELECT 1 FROM transactions t
			WHERE t.type = 'contribution'
				AND json_extract(t.metadata, '$.contributionType') = 'merry-go-round'
				AND json_extract(t.metadata, '$.merryGoRoundId') = ?
				AND json_extract(t.metadata, '$.roundNumber') = ?
				AND json_extract(t.metadata, '$.chamaId') = ?
				AND t.initiated_by = ?
				AND t.status = 'completed'
		)
	`, merryGoRoundID, currentRound, chamaID, userID).Scan(&hasContributed)

	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Printf("⚠️ Contribution status query timed out for user %s\n", userID)
			c.JSON(http.StatusRequestTimeout, gin.H{
				"success": false,
				"error":   "Request timed out. Please try again.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check contribution status",
		})
		return
	}

	// Get current recipient info
	var currentRecipientID string
	var recipientName string
	err = db.(*sql.DB).QueryRow(`
		SELECT mgrp.user_id, COALESCE(u.first_name || ' ' || u.last_name, u.first_name, u.last_name, 'Member')
		FROM merry_go_round_participants mgrp
		JOIN users u ON mgrp.user_id = u.id
		WHERE mgrp.merry_go_round_id = ? AND mgrp.position = ?
	`, merryGoRoundID, currentRound).Scan(&currentRecipientID, &recipientName)

	if err != nil {
		fmt.Printf("⚠️ Failed to get recipient info: %v\n", err)
		recipientName = "Unknown"
	}

	// Count total contributions for this round
	var totalContributions int
	err = db.(*sql.DB).QueryRow(`
		SELECT COUNT(DISTINCT t.initiated_by)
		FROM transactions t
		WHERE t.type = 'contribution'
			AND json_extract(t.metadata, '$.contributionType') = 'merry-go-round'
			AND json_extract(t.metadata, '$.merryGoRoundId') = ?
			AND json_extract(t.metadata, '$.roundNumber') = ?
			AND json_extract(t.metadata, '$.chamaId') = ?
			AND t.status = 'completed'
	`, merryGoRoundID, currentRound, chamaID).Scan(&totalContributions)

	if err != nil {
		fmt.Printf("⚠️ Failed to count contributions: %v\n", err)
		totalContributions = 0
	}

	// Get total participants
	var totalParticipants int
	err = db.(*sql.DB).QueryRow(`
		SELECT COUNT(*)
		FROM merry_go_round_participants
		WHERE merry_go_round_id = ?
	`, merryGoRoundID).Scan(&totalParticipants)

	if err != nil {
		fmt.Printf("⚠️ Failed to count participants: %v\n", err)
		totalParticipants = 0
	}

	// Check if user is a participant in this merry-go-round
	var isParticipant bool
	err = db.(*sql.DB).QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM merry_go_round_participants
			WHERE merry_go_round_id = ? AND user_id = ?
		)
	`, merryGoRoundID, userID).Scan(&isParticipant)

	if err != nil {
		fmt.Printf("⚠️ Failed to check if user is participant: %v\n", err)
		isParticipant = false
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"hasActiveMerryGoRound": true,
			"hasContributed":        hasContributed,
			"canContribute":         !hasContributed,
			"isParticipant":         isParticipant,
			"currentRound":          currentRound,
			"amountPerRound":        amountPerRound,
			"currentRecipient": gin.H{
				"id":   currentRecipientID,
				"name": recipientName,
			},
			"contributionStats": gin.H{
				"totalContributions": totalContributions,
				"totalParticipants":  totalParticipants,
				"progressPercentage": func() float64 {
					if totalParticipants > 0 {
						return (float64(totalContributions) * 100) / float64(totalParticipants)
					}
					return 0
				}(),
			},
			"roundComplete": totalContributions >= totalParticipants && totalParticipants > 0,
		},
	})
}

// CheckAndAdvanceRound checks if all members have contributed to the current recipient
// and automatically advances to the next member if the condition is met
func CheckAndAdvanceRound(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	chamaID := c.Param("chamaId")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	merryGoRoundID := c.Param("merryGoRoundId")
	if merryGoRoundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Merry-go-round ID is required",
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

	// Check if user is a member of the chama
	var membershipExists bool
	err := db.(*sql.DB).QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM chama_members
			WHERE chama_id = ? AND user_id = ? AND is_active = TRUE
		)
	`, chamaID, userID).Scan(&membershipExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify chama membership",
		})
		return
	}

	if !membershipExists {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Access denied. You are not a member of this chama.",
		})
		return
	}

	// Get current round information for response
	var currentRound, totalParticipants int
	var status string
	err = db.(*sql.DB).QueryRow(`
		SELECT current_round, total_participants, status
		FROM merry_go_rounds
		WHERE id = ? AND chama_id = ?
	`, merryGoRoundID, chamaID).Scan(&currentRound, &totalParticipants, &status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get merry-go-round information",
		})
		return
	}

	if status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Merry-go-round is not active",
		})
		return
	}

	// Use the utility function to check and advance
	err = checkAndAdvanceMerryGoRound(db.(*sql.DB), merryGoRoundID, chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to check round advancement: %v", err),
		})
		return
	}

	// Get updated round information after potential advancement
	var updatedCurrentRound int
	var updatedStatus string
	err = db.(*sql.DB).QueryRow(`
		SELECT current_round, status
		FROM merry_go_rounds
		WHERE id = ? AND chama_id = ?
	`, merryGoRoundID, chamaID).Scan(&updatedCurrentRound, &updatedStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get updated merry-go-round information",
		})
		return
	}

	// Count current contributions (contributions made TO the current recipient)
	// Exclude contributions from the recipient to themselves
	var contributionCount int
	err = db.(*sql.DB).QueryRow(`
		SELECT COUNT(DISTINCT t.initiated_by)
		FROM transactions t
		WHERE t.type = 'contribution'
			AND json_extract(t.metadata, '$.contributionType') = 'merry-go-round'
			AND json_extract(t.metadata, '$.merryGoRoundId') = ?
			AND json_extract(t.metadata, '$.roundNumber') = ?
			AND json_extract(t.metadata, '$.chamaId') = ?
			AND t.status = 'completed'
	`, merryGoRoundID, currentRound, chamaID).Scan(&contributionCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to count contributions",
		})
		return
	}

	// Return appropriate response based on status
	if updatedStatus == "completed" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Merry-go-round completed successfully!",
			"data": map[string]interface{}{
				"merryGoRoundId": merryGoRoundID,
				"status":         "completed",
				"finalRound":     updatedCurrentRound,
				"totalRounds":    totalParticipants,
			},
		})
	} else if updatedCurrentRound > currentRound {
		// Round was advanced
		var nextPayoutDate time.Time
		err = db.(*sql.DB).QueryRow(`
			SELECT next_payout_date FROM merry_go_rounds WHERE id = ?
		`, merryGoRoundID).Scan(&nextPayoutDate)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("Round %d completed! Advanced to round %d", currentRound, updatedCurrentRound),
			"data": map[string]interface{}{
				"merryGoRoundId": merryGoRoundID,
				"previousRound":  currentRound,
				"currentRound":   updatedCurrentRound,
				"nextPayoutDate": nextPayoutDate.Format("2006-01-02"),
				"totalRounds":    totalParticipants,
			},
		})
	} else {
		// Round not yet complete
		totalContributionsNeeded := totalParticipants - 1
		remainingContributions := totalContributionsNeeded - contributionCount
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("Round %d in progress: %d/%d contributions completed", updatedCurrentRound, contributionCount, totalContributionsNeeded),
			"data": map[string]interface{}{
				"merryGoRoundId":           merryGoRoundID,
				"currentRound":             updatedCurrentRound,
				"contributionsCompleted":   contributionCount,
				"totalContributionsNeeded": totalContributionsNeeded,
				"remainingContributions":   remainingContributions,
				"progressPercentage":       (contributionCount * 100) / totalContributionsNeeded,
			},
		})
	}
}

// GetMerryGoRoundCalendarAddEventURL returns a pre-filled Google Calendar event URL for a merry-go-round payout
func GetMerryGoRoundCalendarAddEventURL(c *gin.Context) {
	merryGoRoundID := c.Param("id")
	if merryGoRoundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Merry-go-round ID is required",
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

	// Get merry-go-round details
	var mgr struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		Description    string    `json:"description"`
		AmountPerRound float64   `json:"amountPerRound"`
		Frequency      string    `json:"frequency"`
		NextPayoutDate time.Time `json:"nextPayoutDate"`
		ChamaID        string    `json:"chamaId"`
	}

	err := db.(*sql.DB).QueryRow(`
		SELECT id, name, description, amount_per_round, frequency, next_payout_date, chama_id
		FROM merry_go_rounds
		WHERE id = ?
	`, merryGoRoundID).Scan(&mgr.ID, &mgr.Name, &mgr.Description, &mgr.AmountPerRound, &mgr.Frequency, &mgr.NextPayoutDate, &mgr.ChamaID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Merry-go-round not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch merry-go-round details: " + err.Error(),
			})
		}
		return
	}

	// Get chama name for better labeling
	var chamaName string
	err = db.(*sql.DB).QueryRow("SELECT name FROM chamas WHERE id = ?", mgr.ChamaID).Scan(&chamaName)
	if err != nil {
		chamaName = "Chama"
	}

	// Build event summary and description
	summary := fmt.Sprintf("%s — %s", mgr.Name, chamaName)
	description := fmt.Sprintf("Merry-Go-Round payout reminder.\n\nAmount: %.2f KES\nFrequency: %s\n\nNext payout date for the merry-go-round cycle.", mgr.AmountPerRound, mgr.Frequency)

	// Use next payout date as the event date
	// Set time to 9:00 AM EAT for the reminder
	eat, _ := time.LoadLocation("Africa/Nairobi")
	eventDate := mgr.NextPayoutDate.In(eat)
	startTime := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 9, 0, 0, 0, eat)
	endTime := startTime.Add(1 * time.Hour) // 1 hour duration

	// Google Calendar template URL
	const template = "https://calendar.google.com/calendar/render"
	params := url.Values{}
	params.Set("action", "TEMPLATE")
	params.Set("text", summary)
	params.Set("details", description)
	params.Set("location", chamaName)

	// Provide local datetime without Z and set ctz to Africa/Nairobi for accurate display
	toLocal := func(t time.Time) string { return t.Format("20060102T150405") }
	params.Set("dates", fmt.Sprintf("%s/%s", toLocal(startTime), toLocal(endTime)))
	params.Set("ctz", "Africa/Nairobi")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": template + "?" + params.Encode(),
		},
	})
}

// CreateMerryGoRoundCalendarEvent creates the event in the user's Google Calendar
func CreateMerryGoRoundCalendarEvent(c *gin.Context) {
	merryGoRoundID := c.Param("id")
	if merryGoRoundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Merry-go-round ID is required"})
		return
	}

	// Auth user
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}

	// Get DB and merry-go-round details
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Database connection not available"})
		return
	}

	var mgr struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		Description    string    `json:"description"`
		AmountPerRound float64   `json:"amountPerRound"`
		Frequency      string    `json:"frequency"`
		NextPayoutDate time.Time `json:"nextPayoutDate"`
		ChamaID        string    `json:"chamaId"`
	}

	err := db.(*sql.DB).QueryRow(`
		SELECT id, name, description, amount_per_round, frequency, next_payout_date, chama_id
		FROM merry_go_rounds
		WHERE id = ?
	`, merryGoRoundID).Scan(&mgr.ID, &mgr.Name, &mgr.Description, &mgr.AmountPerRound, &mgr.Frequency, &mgr.NextPayoutDate, &mgr.ChamaID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Merry-go-round not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to get merry-go-round: " + err.Error()})
		}
		return
	}

	// Get chama name
	var chamaName string
	err = db.(*sql.DB).QueryRow("SELECT name FROM chamas WHERE id = ?", mgr.ChamaID).Scan(&chamaName)
	if err != nil {
		chamaName = "Chama"
	}

	// Get the user's stored Google tokens
	driveService := services.NewGoogleDriveService(db.(*sql.DB))
	token, err := driveService.GetUserTokens(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Google account not connected for this user"})
		return
	}

	// Initialize CalendarService
	creds := os.Getenv("GOOGLE_CALENDAR_CREDENTIALS_JSON")
	if creds == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Google Calendar integration is not configured on this server. Please contact the administrator to set up Google Calendar credentials.",
			"code":    "CALENDAR_NOT_CONFIGURED",
		})
		return
	}
	calService, err := services.NewCalendarService([]byte(creds))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize calendar service. The Google Calendar credentials may be invalid.",
			"details": err.Error(),
		})
		return
	}
	if err := calService.InitializeWithToken(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to authorize calendar: " + err.Error()})
		return
	}

	// Build event with accurate start/end and EAT timezone
	title := fmt.Sprintf("%s — %s", mgr.Name, chamaName)
	desc := fmt.Sprintf("%s\n\nAmount: %.2f KES\nFrequency: %s\n\nNext payout date for the merry-go-round cycle.", mgr.Description, mgr.AmountPerRound, mgr.Frequency)

	// Set time to 9:00 AM EAT for the reminder
	eat, _ := time.LoadLocation("Africa/Nairobi")
	eventDate := mgr.NextPayoutDate.In(eat)
	startTime := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 9, 0, 0, 0, eat)
	endTime := startTime.Add(1 * time.Hour)

	ev := &services.CalendarEvent{
		Title:       title,
		Description: desc,
		StartTime:   startTime,
		EndTime:     endTime,
		Location:    chamaName,
	}

	// Use primary calendar and reminders 30,10,0 minutes
	created, err := calService.CreateEventWithReminders("primary", ev, []int{30, 10, 0})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create calendar event: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"eventId": created.Id, "htmlLink": created.HtmlLink}})
}
