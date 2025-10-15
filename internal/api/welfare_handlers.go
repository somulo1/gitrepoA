package api

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Welfare handlers
func GetWelfareRequests(c *gin.Context) {
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

	// Check and close any completed welfare votes before loading data
	CheckAndCloseCompletedWelfareVotes(chamaID, db.(*sql.DB))

	// Get user ID from context for checking user votes
	currentUserID, _ := c.Get("userID")

	// Query welfare requests for the chama with vote information and beneficiary details
	rows, err := db.(*sql.DB).Query(`
		SELECT
			wr.id, wr.chama_id, wr.requester_id, COALESCE(wr.beneficiary_id, wr.requester_id) as beneficiary_id,
			wr.title, wr.description, wr.amount, wr.category, wr.urgency, wr.status,
			wr.votes_for, wr.votes_against, wr.created_at, wr.updated_at,
			u.first_name as requester_first_name, u.last_name as requester_last_name, u.email as requester_email,
			b.first_name as beneficiary_first_name, b.last_name as beneficiary_last_name, b.email as beneficiary_email,
			COALESCE(yes_votes.vote_count, 0) as actual_yes_votes,
			COALESCE(no_votes.vote_count, 0) as actual_no_votes,
			user_vote.option_text as user_vote_option
		FROM welfare_requests wr
		JOIN users u ON wr.requester_id = u.id
		LEFT JOIN users b ON COALESCE(wr.beneficiary_id, wr.requester_id) = b.id
		LEFT JOIN votes v ON v.chama_id = wr.chama_id AND v.title = 'Welfare Request: ' || wr.id AND v.type = 'welfare'
		LEFT JOIN vote_options yes_votes ON yes_votes.vote_id = v.id AND yes_votes.option_text = 'yes'
		LEFT JOIN vote_options no_votes ON no_votes.vote_id = v.id AND no_votes.option_text = 'no'
		LEFT JOIN user_votes uv ON uv.vote_id = v.id AND uv.user_id = ?
		LEFT JOIN vote_options user_vote ON user_vote.id = uv.option_id
		WHERE wr.chama_id = ?
		ORDER BY wr.created_at DESC
	`, currentUserID, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch welfare requests: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var welfareRequests []map[string]interface{}
	for rows.Next() {
		var wr struct {
			ID                   string    `json:"id"`
			ChamaID              string    `json:"chamaId"`
			RequesterID          string    `json:"requesterId"`
			BeneficiaryID        string    `json:"beneficiaryId"`
			Title                string    `json:"title"`
			Description          string    `json:"description"`
			Amount               float64   `json:"amount"`
			Category             string    `json:"category"`
			Urgency              string    `json:"urgency"`
			Status               string    `json:"status"`
			VotesFor             int       `json:"votesFor"`
			VotesAgainst         int       `json:"votesAgainst"`
			CreatedAt            time.Time `json:"createdAt"`
			UpdatedAt            time.Time `json:"updatedAt"`
			RequesterFirstName   string    `json:"requesterFirstName"`
			RequesterLastName    string    `json:"requesterLastName"`
			RequesterEmail       string    `json:"requesterEmail"`
			BeneficiaryFirstName string    `json:"beneficiaryFirstName"`
			BeneficiaryLastName  string    `json:"beneficiaryLastName"`
			BeneficiaryEmail     string    `json:"beneficiaryEmail"`
			ActualYesVotes       int       `json:"actualYesVotes"`
			ActualNoVotes        int       `json:"actualNoVotes"`
			UserVoteOption       *string   `json:"userVoteOption"` // nullable
		}

		err := rows.Scan(
			&wr.ID, &wr.ChamaID, &wr.RequesterID, &wr.BeneficiaryID, &wr.Title, &wr.Description,
			&wr.Amount, &wr.Category, &wr.Urgency, &wr.Status, &wr.VotesFor,
			&wr.VotesAgainst, &wr.CreatedAt, &wr.UpdatedAt,
			&wr.RequesterFirstName, &wr.RequesterLastName, &wr.RequesterEmail,
			&wr.BeneficiaryFirstName, &wr.BeneficiaryLastName, &wr.BeneficiaryEmail,
			&wr.ActualYesVotes, &wr.ActualNoVotes, &wr.UserVoteOption,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		// Use actual vote counts from voting tables, fallback to welfare_requests table
		actualYesVotes := wr.ActualYesVotes
		actualNoVotes := wr.ActualNoVotes
		if actualYesVotes == 0 && actualNoVotes == 0 {
			// Fallback to welfare_requests table if no votes exist yet
			actualYesVotes = wr.VotesFor
			actualNoVotes = wr.VotesAgainst
		}

		// Determine user vote status
		var userVote interface{} = nil
		if wr.UserVoteOption != nil {
			userVote = *wr.UserVoteOption // "yes" or "no"
		}

		welfareMap := map[string]interface{}{
			"id":            wr.ID,
			"chamaId":       wr.ChamaID,
			"requesterId":   wr.RequesterID,
			"beneficiaryId": wr.BeneficiaryID,
			"title":         wr.Title,
			"description":   wr.Description,
			"amount":        wr.Amount,
			"category":      wr.Category,
			"urgency":       wr.Urgency,
			"status":        wr.Status,
			"votesFor":      actualYesVotes,
			"votesAgainst":  actualNoVotes,
			"votes_for":     actualYesVotes, // Alternative field name for frontend compatibility
			"votes_against": actualNoVotes,  // Alternative field name for frontend compatibility
			"total_votes":   actualYesVotes + actualNoVotes,
			"userVote":      userVote,
			"votes": map[string]interface{}{
				"yes":   actualYesVotes,
				"no":    actualNoVotes,
				"total": actualYesVotes + actualNoVotes,
			},
			"createdAt": wr.CreatedAt.Format(time.RFC3339),
			"updatedAt": wr.UpdatedAt.Format(time.RFC3339),
			"requester": map[string]interface{}{
				"id":         wr.RequesterID,
				"firstName":  wr.RequesterFirstName,
				"lastName":   wr.RequesterLastName,
				"email":      wr.RequesterEmail,
				"fullName":   wr.RequesterFirstName + " " + wr.RequesterLastName,
				"first_name": wr.RequesterFirstName, // Alternative field name
				"last_name":  wr.RequesterLastName,  // Alternative field name
			},
			"beneficiary": map[string]interface{}{
				"id":         wr.BeneficiaryID,
				"firstName":  wr.BeneficiaryFirstName,
				"lastName":   wr.BeneficiaryLastName,
				"email":      wr.BeneficiaryEmail,
				"fullName":   wr.BeneficiaryFirstName + " " + wr.BeneficiaryLastName,
				"first_name": wr.BeneficiaryFirstName, // Alternative field name
				"last_name":  wr.BeneficiaryLastName,  // Alternative field name
			},
		}

		// Add contribution calculations for real-time progress
		welfareRequestFundID := fmt.Sprintf("fund-%s", wr.ID)
		var totalContributions float64
		var contributionCount int

		err = db.(*sql.DB).QueryRow(`
			SELECT COALESCE(SUM(amount), 0), COUNT(*)
			FROM welfare_contributions
			WHERE welfare_fund_id = ?
		`, welfareRequestFundID).Scan(&totalContributions, &contributionCount)
		if err != nil {
			// Log error but continue with zero values
			fmt.Printf("Error getting contributions for request %s: %v\n", wr.ID, err)
			totalContributions = 0
			contributionCount = 0
		}

		// Add contribution data to the welfare map
		welfareMap["totalContributions"] = totalContributions
		welfareMap["contributionCount"] = contributionCount
		welfareMap["remainingAmount"] = math.Max(0, wr.Amount-totalContributions)
		welfareMap["progressPercentage"] = math.Min(100, (totalContributions/wr.Amount)*100)

		welfareRequests = append(welfareRequests, welfareMap)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    welfareRequests,
		"message": fmt.Sprintf("Found %d welfare requests", len(welfareRequests)),
		"meta": map[string]interface{}{
			"total":   len(welfareRequests),
			"chamaId": chamaID,
		},
	})
}

func CreateWelfareRequest(c *gin.Context) {
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
		ChamaID       string  `json:"chamaId" binding:"required"`
		Title         string  `json:"title" binding:"required"`
		Description   string  `json:"description" binding:"required"`
		Amount        float64 `json:"amount" binding:"required"`
		Category      string  `json:"category" binding:"required"`
		Urgency       string  `json:"urgency" binding:"required"`
		BeneficiaryID *string `json:"beneficiaryId"` // Optional - defaults to requester if not provided
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

	// Validate urgency
	validUrgencies := map[string]bool{"low": true, "medium": true, "high": true, "emergency": true}
	if !validUrgencies[req.Urgency] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Urgency must be one of: low, medium, high, emergency",
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

	// Determine beneficiary - defaults to requester if not specified
	beneficiaryID := userID
	if req.BeneficiaryID != nil && *req.BeneficiaryID != "" {
		beneficiaryID = *req.BeneficiaryID
	}

	// Generate welfare request ID
	welfareID := fmt.Sprintf("welfare-%d", time.Now().UnixNano())

	// Insert welfare request into database
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO welfare_requests (
			id, chama_id, requester_id, beneficiary_id, title, description, amount,
			category, urgency, status, votes_for, votes_against,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, welfareID, req.ChamaID, userID, beneficiaryID, req.Title, req.Description, req.Amount, req.Category, req.Urgency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create welfare request: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Welfare request created successfully! Members will be notified to vote.",
		"data": map[string]interface{}{
			"id":            welfareID,
			"chamaId":       req.ChamaID,
			"requesterId":   userID,
			"beneficiaryId": beneficiaryID,
			"title":         req.Title,
			"description":   req.Description,
			"amount":        req.Amount,
			"category":      req.Category,
			"urgency":       req.Urgency,
			"status":        "pending",
			"votesFor":      0,
			"votesAgainst":  0,
			"createdAt":     time.Now().Format(time.RFC3339),
		},
	})
}

func GetWelfareRequest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Get welfare request endpoint - coming soon",
	})
}

func UpdateWelfareRequest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Update welfare request endpoint - coming soon",
	})
}

func DeleteWelfareRequest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delete welfare request endpoint - coming soon",
	})
}

func VoteOnWelfareRequest(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get welfare request ID from URL parameter
	welfareID := c.Param("id")
	if welfareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Welfare request ID is required",
		})
		return
	}

	var req struct {
		Vote string `json:"vote" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Validate vote value
	if req.Vote != "yes" && req.Vote != "no" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Vote must be 'yes' or 'no'",
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

	// Check if welfare request exists
	var welfareRequest struct {
		ID           string
		ChamaID      string
		RequesterID  string
		Status       string
		VotesFor     int
		VotesAgainst int
	}

	err := db.(*sql.DB).QueryRow(`
		SELECT id, chama_id, requester_id, status, votes_for, votes_against
		FROM welfare_requests
		WHERE id = ?
	`, welfareID).Scan(&welfareRequest.ID, &welfareRequest.ChamaID, &welfareRequest.RequesterID, &welfareRequest.Status, &welfareRequest.VotesFor, &welfareRequest.VotesAgainst)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Welfare request not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch welfare request: " + err.Error(),
		})
		return
	}

	// Check if welfare request is still pending (can only vote on pending requests)
	if welfareRequest.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Cannot vote on this welfare request. Status: " + welfareRequest.Status,
		})
		return
	}

	// Check if there's already a vote for this welfare request
	var voteID string
	err = db.(*sql.DB).QueryRow(`
		SELECT id FROM votes
		WHERE chama_id = ? AND title = ? AND type = 'welfare'
	`, welfareRequest.ChamaID, "Welfare Request: "+welfareID).Scan(&voteID)

	// If no vote exists, create one
	if err == sql.ErrNoRows {
		voteID = fmt.Sprintf("vote-%d", time.Now().UnixNano())
		_, err = db.(*sql.DB).Exec(`
			INSERT INTO votes (id, chama_id, title, description, type, status, ends_at, created_by, created_at)
			VALUES (?, ?, ?, ?, 'welfare', 'active', datetime('now', '+7 days'), ?, CURRENT_TIMESTAMP)
		`, voteID, welfareRequest.ChamaID, "Welfare Request: "+welfareID, "Vote on welfare request", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create vote: " + err.Error(),
			})
			return
		}

		// Create vote options (Yes/No)
		yesOptionID := fmt.Sprintf("option-%d-yes", time.Now().UnixNano())
		noOptionID := fmt.Sprintf("option-%d-no", time.Now().UnixNano())

		_, err = db.(*sql.DB).Exec(`
			INSERT INTO vote_options (id, vote_id, option_text, vote_count) VALUES
			(?, ?, 'yes', 0),
			(?, ?, 'no', 0)
		`, yesOptionID, voteID, noOptionID, voteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create vote options: " + err.Error(),
			})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check existing vote: " + err.Error(),
		})
		return
	}

	// Get the appropriate option ID
	var optionID string
	err = db.(*sql.DB).QueryRow(`
		SELECT id FROM vote_options
		WHERE vote_id = ? AND option_text = ?
	`, voteID, req.Vote).Scan(&optionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to find vote option: " + err.Error(),
		})
		return
	}

	// Check if user has already voted
	var existingUserVote string
	err = db.(*sql.DB).QueryRow(`
		SELECT id FROM user_votes
		WHERE vote_id = ? AND user_id = ?
	`, voteID, userID).Scan(&existingUserVote)

	if existingUserVote != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "You have already voted on this welfare request",
		})
		return
	}

	// Insert the user vote
	userVoteID := fmt.Sprintf("uv-%d", time.Now().UnixNano())
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO user_votes (id, vote_id, user_id, option_id, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, userVoteID, voteID, userID, optionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to record vote: " + err.Error(),
		})
		return
	}

	// Update vote count in vote_options
	_, err = db.(*sql.DB).Exec(`
		UPDATE vote_options
		SET vote_count = vote_count + 1
		WHERE id = ?
	`, optionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update vote count: " + err.Error(),
		})
		return
	}

	// Get updated vote counts from vote_options
	var yesVotes, noVotes int
	err = db.(*sql.DB).QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN option_text = 'yes' THEN vote_count ELSE 0 END), 0) as yes_votes,
			COALESCE(SUM(CASE WHEN option_text = 'no' THEN vote_count ELSE 0 END), 0) as no_votes
		FROM vote_options
		WHERE vote_id = ?
	`, voteID).Scan(&yesVotes, &noVotes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get vote counts: " + err.Error(),
		})
		return
	}

	// Update vote counts in welfare_requests table for consistency
	_, err = db.(*sql.DB).Exec(`
		UPDATE welfare_requests
		SET votes_for = ?, votes_against = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, yesVotes, noVotes, welfareID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update welfare request vote counts: " + err.Error(),
		})
		return
	}

	// Check if all members have voted and close voting if necessary
	var totalMembers int
	err = db.(*sql.DB).QueryRow(`
		SELECT COUNT(*) FROM chama_members
		WHERE chama_id = ? AND is_active = TRUE
	`, welfareRequest.ChamaID).Scan(&totalMembers)
	if err != nil {
		// Log error but don't fail the response
		fmt.Printf("Failed to get total chama members: %v\n", err)
	} else {
		totalVotes := yesVotes + noVotes

		// Check if all members have voted
		if totalVotes >= totalMembers {
			// All members have voted, determine the result
			var newStatus string
			var statusMessage string

			// Simple majority rule: more than 50% yes votes = approved
			if yesVotes > noVotes {
				newStatus = "approved"
				statusMessage = fmt.Sprintf("Approved by majority vote (%d yes, %d no)", yesVotes, noVotes)
			} else {
				newStatus = "rejected"
				statusMessage = fmt.Sprintf("Rejected by majority vote (%d yes, %d no)", yesVotes, noVotes)
			}

			// Update welfare request status
			_, err = db.(*sql.DB).Exec(`
				UPDATE welfare_requests
				SET status = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, newStatus, welfareID)
			if err != nil {
				fmt.Printf("Failed to update welfare request status: %v\n", err)
			}

			// Close the vote
			_, err = db.(*sql.DB).Exec(`
				UPDATE votes
				SET status = 'completed', ends_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, voteID)
			if err != nil {
				fmt.Printf("Failed to close vote: %v\n", err)
			}

			// Create notification for requester about the result
			notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
			_, err = db.(*sql.DB).Exec(`
				INSERT INTO notifications (
					id, user_id, type, title, message, data,
					is_read, created_at, updated_at
				) VALUES (?, ?, 'welfare_vote_result', ?, ?, ?, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, notificationID, welfareRequest.RequesterID,
				"Welfare Vote Complete",
				statusMessage,
				fmt.Sprintf(`{"welfare_request_id": "%s", "status": "%s", "yes_votes": %d, "no_votes": %d}`,
					welfareID, newStatus, yesVotes, noVotes))
			if err != nil {
				fmt.Printf("Failed to create vote result notification: %v\n", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vote recorded successfully",
		"data": map[string]interface{}{
			"voteId":       voteID,
			"userVoteId":   userVoteID,
			"vote":         req.Vote,
			"votesFor":     yesVotes,
			"votesAgainst": noVotes,
			"totalVotes":   yesVotes + noVotes,
			"status":       "recorded",
		},
	})
}

// ContributeToWelfare handles welfare contributions
func ContributeToWelfare(c *gin.Context) {
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
		WelfareRequestID string  `json:"welfareRequestId" binding:"required"`
		Amount           float64 `json:"amount" binding:"required"`
		Message          string  `json:"message"`
		ChamaID          string  `json:"chamaId" binding:"required"`
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
			"error":   "Contribution amount must be greater than 0",
		})
		return
	}

	if req.Amount < 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minimum contribution amount is KES 10",
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

	// Check if welfare request exists and is approved
	var welfareRequest struct {
		ID            string
		ChamaID       string
		Status        string
		Amount        float64
		Title         string
		BeneficiaryID string
	}

	err = db.(*sql.DB).QueryRow(`
		SELECT id, chama_id, status, amount, title, COALESCE(beneficiary_id, requester_id) as beneficiary_id
		FROM welfare_requests
		WHERE id = ? AND chama_id = ?
	`, req.WelfareRequestID, req.ChamaID).Scan(
		&welfareRequest.ID, &welfareRequest.ChamaID, &welfareRequest.Status, &welfareRequest.Amount,
		&welfareRequest.Title, &welfareRequest.BeneficiaryID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Welfare request not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch welfare request: " + err.Error(),
		})
		return
	}

	// Only allow contributions to approved welfare requests
	if welfareRequest.Status != "approved" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Can only contribute to approved welfare requests. Current status: " + welfareRequest.Status,
		})
		return
	}

	// Allow self-contributions - users can contribute to their own welfare requests
	// This enables users to partially fund their own requests or show commitment

	// Create or get welfare fund for this request
	welfareRequestFundID := fmt.Sprintf("fund-%s", req.WelfareRequestID)

	// Check if welfare fund exists for this request
	var existingFundID string
	err = db.(*sql.DB).QueryRow(`
		SELECT id FROM welfare_funds WHERE id = ?
	`, welfareRequestFundID).Scan(&existingFundID)

	// Create welfare fund if it doesn't exist
	if err == sql.ErrNoRows {
		_, err = db.(*sql.DB).Exec(`
			INSERT INTO welfare_funds (
				id, chama_id, name, description, purpose, status, beneficiary_id, created_by, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, 'active', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, welfareRequestFundID, welfareRequest.ChamaID,
			"Fund for: "+welfareRequest.Title,
			"Welfare fund for welfare request: "+req.WelfareRequestID,
			"emergency", welfareRequest.BeneficiaryID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create welfare fund: " + err.Error(),
			})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check welfare fund: " + err.Error(),
		})
		return
	}

	// Generate contribution ID
	contributionID := fmt.Sprintf("contrib-%d", time.Now().UnixNano())

	// Insert contribution into database
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO welfare_contributions (
			id, welfare_fund_id, user_id, amount, payment_method, contributed_at
		) VALUES (?, ?, ?, ?, 'mobile_money', CURRENT_TIMESTAMP)
	`, contributionID, welfareRequestFundID, userID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to record contribution: " + err.Error(),
		})
		return
	}

	// Get total contributions for this welfare request
	var totalContributions float64
	err = db.(*sql.DB).QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM welfare_contributions
		WHERE welfare_fund_id = ?
	`, welfareRequestFundID).Scan(&totalContributions)
	if err != nil {
		// Log error but don't fail the response
		fmt.Printf("Failed to calculate total contributions: %v\n", err)
		totalContributions = req.Amount // fallback
	}

	// Create notification for beneficiary
	notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO notifications (
			id, user_id, type, title, message, data,
			is_read, created_at, updated_at
		) VALUES (?, ?, 'welfare_contribution', ?, ?, ?, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, notificationID, welfareRequest.BeneficiaryID,
		"New Welfare Contribution",
		fmt.Sprintf("You received a contribution of KES %.2f for your welfare request", req.Amount),
		fmt.Sprintf(`{"welfare_request_id": "%s", "contribution_id": "%s", "amount": %.2f}`,
			req.WelfareRequestID, contributionID, req.Amount))
	if err != nil {
		// Log error but don't fail the response
		fmt.Printf("Failed to create contribution notification: %v\n", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Contribution recorded successfully",
		"data": map[string]interface{}{
			"contributionId":     contributionID,
			"welfareRequestId":   req.WelfareRequestID,
			"amount":             req.Amount,
			"message":            req.Message,
			"totalContributions": totalContributions,
			"status":             "completed",
			"createdAt":          time.Now().Format(time.RFC3339),
		},
	})
}

// GetWelfareContributions gets all contributions for a specific welfare request
func GetWelfareContributions(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	welfareRequestID := c.Param("id")
	if welfareRequestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Welfare request ID is required",
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

	// Query contributions for the welfare request
	// Get welfare fund ID for this request
	welfareRequestFundID := fmt.Sprintf("fund-%s", welfareRequestID)

	rows, err := db.(*sql.DB).Query(`
		SELECT
			wc.id, wc.amount, COALESCE(wc.message, '') as message, 'completed' as status, wc.contributed_at,
			u.first_name, u.last_name, u.email
		FROM welfare_contributions wc
		JOIN users u ON wc.user_id = u.id
		WHERE wc.welfare_fund_id = ?
		ORDER BY wc.contributed_at DESC
	`, welfareRequestFundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch contributions: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var contributions []map[string]interface{}
	var totalAmount float64

	for rows.Next() {
		var contrib struct {
			ID                   string    `json:"id"`
			Amount               float64   `json:"amount"`
			Message              string    `json:"message"`
			Status               string    `json:"status"`
			CreatedAt            time.Time `json:"createdAt"`
			ContributorFirstName string    `json:"contributorFirstName"`
			ContributorLastName  string    `json:"contributorLastName"`
			ContributorEmail     string    `json:"contributorEmail"`
		}

		err := rows.Scan(
			&contrib.ID, &contrib.Amount, &contrib.Message, &contrib.Status, &contrib.CreatedAt,
			&contrib.ContributorFirstName, &contrib.ContributorLastName, &contrib.ContributorEmail,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		totalAmount += contrib.Amount

		contribution := map[string]interface{}{
			"id":        contrib.ID,
			"amount":    contrib.Amount,
			"message":   contrib.Message,
			"status":    contrib.Status,
			"createdAt": contrib.CreatedAt.Format(time.RFC3339),
			"contributor": map[string]interface{}{
				"firstName": contrib.ContributorFirstName,
				"lastName":  contrib.ContributorLastName,
				"email":     contrib.ContributorEmail,
				"fullName":  contrib.ContributorFirstName + " " + contrib.ContributorLastName,
			},
		}

		contributions = append(contributions, contribution)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    contributions,
		"meta": map[string]interface{}{
			"totalContributions": totalAmount,
			"contributionCount":  len(contributions),
			"welfareRequestId":   welfareRequestID,
		},
	})
}

// CheckAndCloseCompletedWelfareVotes checks for welfare votes where all members have voted
// and automatically closes them with the appropriate status
func CheckAndCloseCompletedWelfareVotes(chamaID string, db *sql.DB) {
	// Get all pending welfare requests for this chama
	rows, err := db.Query(`
		SELECT wr.id, wr.chama_id, wr.requester_id, wr.status, v.id as vote_id
		FROM welfare_requests wr
		LEFT JOIN votes v ON v.chama_id = wr.chama_id AND v.title = 'Welfare Request: ' || wr.id AND v.type = 'welfare'
		WHERE wr.chama_id = ? AND wr.status = 'pending' AND v.status = 'active'
	`, chamaID)
	if err != nil {
		fmt.Printf("Failed to get pending welfare requests: %v\n", err)
		return
	}
	defer rows.Close()

	// Get total number of active members in this chama
	var totalMembers int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM chama_members
		WHERE chama_id = ? AND is_active = TRUE
	`, chamaID).Scan(&totalMembers)
	if err != nil {
		fmt.Printf("Failed to get total chama members: %v\n", err)
		return
	}

	for rows.Next() {
		var welfareID, welfareRequestChamaID, requesterID, status, voteID string
		err := rows.Scan(&welfareID, &welfareRequestChamaID, &requesterID, &status, &voteID)
		if err != nil {
			continue
		}

		// Get current vote counts for this welfare request
		var yesVotes, noVotes int
		err = db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN option_text = 'yes' THEN vote_count ELSE 0 END), 0) as yes_votes,
				COALESCE(SUM(CASE WHEN option_text = 'no' THEN vote_count ELSE 0 END), 0) as no_votes
			FROM vote_options
			WHERE vote_id = ?
		`, voteID).Scan(&yesVotes, &noVotes)
		if err != nil {
			continue
		}

		totalVotes := yesVotes + noVotes

		// Check if all members have voted
		if totalVotes >= totalMembers {
			// Determine the result
			var newStatus string
			var statusMessage string

			if yesVotes > noVotes {
				newStatus = "approved"
				statusMessage = fmt.Sprintf("Approved by majority vote (%d yes, %d no)", yesVotes, noVotes)
			} else {
				newStatus = "rejected"
				statusMessage = fmt.Sprintf("Rejected by majority vote (%d yes, %d no)", yesVotes, noVotes)
			}

			// Update welfare request status
			_, err = db.Exec(`
				UPDATE welfare_requests
				SET status = ?, votes_for = ?, votes_against = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, newStatus, yesVotes, noVotes, welfareID)
			if err != nil {
				fmt.Printf("Failed to update welfare request status for %s: %v\n", welfareID, err)
				continue
			}

			// Close the vote
			_, err = db.Exec(`
				UPDATE votes
				SET status = 'completed', ends_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, voteID)
			if err != nil {
				fmt.Printf("Failed to close vote %s: %v\n", voteID, err)
			}

			// Create notification for requester
			notificationID := fmt.Sprintf("notif-%d", time.Now().UnixNano())
			_, err = db.Exec(`
				INSERT INTO notifications (
					id, user_id, type, title, message, data,
					is_read, created_at, updated_at
				) VALUES (?, ?, 'welfare_vote_result', ?, ?, ?, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, notificationID, requesterID,
				"Welfare Vote Complete",
				statusMessage,
				fmt.Sprintf(`{"welfare_request_id": "%s", "status": "%s", "yes_votes": %d, "no_votes": %d}`,
					welfareID, newStatus, yesVotes, noVotes))
			if err != nil {
				fmt.Printf("Failed to create vote result notification for %s: %v\n", welfareID, err)
			}

			fmt.Printf("Auto-closed welfare vote for request %s with status: %s\n", welfareID, newStatus)
		}
	}
}
