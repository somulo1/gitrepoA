package api

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
	// "strings"
	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	meetingService      *services.MeetingService
	schedulerService    *services.SchedulerService
	notificationService *services.NotificationService
)

// Meeting handlers
func GetMeetings(c *gin.Context) {
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

	// Query meetings for the chama
	rows, err := db.(*sql.DB).Query(`
		SELECT
			m.id, m.chama_id, m.title, m.description, m.scheduled_at,
			m.duration, m.location, m.meeting_url, m.meeting_type, m.status, m.created_by, m.created_at,
			u.first_name, u.last_name, u.email
		FROM meetings m
		JOIN users u ON m.created_by = u.id
		WHERE m.chama_id = ?
		ORDER BY m.scheduled_at DESC
	`, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch meetings: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var meetings []map[string]interface{}
	for rows.Next() {
		var meeting struct {
			ID               string    `json:"id"`
			ChamaID          string    `json:"chamaId"`
			Title            string    `json:"title"`
			Description      string    `json:"description"`
			ScheduledAt      time.Time `json:"scheduledAt"`
			Duration         int       `json:"duration"`
			Location         string    `json:"location"`
			MeetingURL       string    `json:"meetingUrl"`
			MeetingType      string    `json:"meetingType"`
			Status           string    `json:"status"`
			CreatedBy        string    `json:"createdBy"`
			CreatedAt        time.Time `json:"createdAt"`
			CreatorFirstName string    `json:"creatorFirstName"`
			CreatorLastName  string    `json:"creatorLastName"`
			CreatorEmail     string    `json:"creatorEmail"`
		}

		err := rows.Scan(
			&meeting.ID, &meeting.ChamaID, &meeting.Title, &meeting.Description, &meeting.ScheduledAt,
			&meeting.Duration, &meeting.Location, &meeting.MeetingURL, &meeting.MeetingType, &meeting.Status, &meeting.CreatedBy, &meeting.CreatedAt,
			&meeting.CreatorFirstName, &meeting.CreatorLastName, &meeting.CreatorEmail,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		// Determine meeting status based on scheduled time and duration
		now := time.Now()
		status := meeting.Status
		meetingEndTime := meeting.ScheduledAt.Add(time.Duration(meeting.Duration) * time.Minute)

		// Only mark as completed if the meeting has actually ended (not just started)
		if meetingEndTime.Before(now) && status == "scheduled" {
			status = "completed"
		}

		meetingMap := map[string]interface{}{
			"id":          meeting.ID,
			"chamaId":     meeting.ChamaID,
			"title":       meeting.Title,
			"description": meeting.Description,
			"scheduledAt": meeting.ScheduledAt.Format(time.RFC3339),
			"duration":    meeting.Duration,
			"location":    meeting.Location,
			"meetingUrl":  meeting.MeetingURL,
			"meetingType": meeting.MeetingType,
			"type":        meeting.MeetingType, // Also include as 'type' for compatibility
			"status":      status,
			"createdBy":   meeting.CreatedBy,
			"createdAt":   meeting.CreatedAt.Format(time.RFC3339),
			"creator": map[string]interface{}{
				"id":        meeting.CreatedBy,
				"firstName": meeting.CreatorFirstName,
				"lastName":  meeting.CreatorLastName,
				"email":     meeting.CreatorEmail,
				"fullName":  meeting.CreatorFirstName + " " + meeting.CreatorLastName,
			},
		}

		meetings = append(meetings, meetingMap)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    meetings,
		"message": fmt.Sprintf("Found %d meetings", len(meetings)),
		"meta": map[string]interface{}{
			"total":   len(meetings),
			"chamaId": chamaID,
		},
	})
}

// GetUserMeetings gets all meetings for the current user across all chamas they belong to
func GetUserMeetings(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
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

	// Parse pagination parameters
	limit := 50
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Query meetings for all chamas the user belongs to
	rows, err := db.(*sql.DB).Query(`
		SELECT DISTINCT
			m.id, m.chama_id, m.title, m.description, m.scheduled_at,
			m.duration, m.location, m.meeting_url, m.meeting_type, m.status, m.created_by, m.created_at,
			u.first_name, u.last_name, u.email,
			c.name as chama_name
		FROM meetings m
		JOIN users u ON m.created_by = u.id
		JOIN chama_members cm ON m.chama_id = cm.chama_id
		JOIN chamas c ON m.chama_id = c.id
		WHERE cm.user_id = ? AND cm.is_active = TRUE
		ORDER BY m.scheduled_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)

	if err != nil {
		log.Printf("❌ Error fetching user meetings for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch user meetings: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var meetings []map[string]interface{}
	chamaCount := make(map[string]bool)

	for rows.Next() {
		var meeting struct {
			ID          string
			ChamaID     string
			Title       string
			Description sql.NullString
			ScheduledAt time.Time
			Duration    sql.NullInt64
			Location    sql.NullString
			MeetingURL  sql.NullString
			MeetingType sql.NullString
			Status      string
			CreatedBy   string
			CreatedAt   time.Time
			CreatorFirstName string
			CreatorLastName  string
			CreatorEmail     string
			ChamaName        string
		}

		err := rows.Scan(
			&meeting.ID, &meeting.ChamaID, &meeting.Title, &meeting.Description,
			&meeting.ScheduledAt, &meeting.Duration, &meeting.Location,
			&meeting.MeetingURL, &meeting.MeetingType, &meeting.Status,
			&meeting.CreatedBy, &meeting.CreatedAt,
			&meeting.CreatorFirstName, &meeting.CreatorLastName, &meeting.CreatorEmail,
			&meeting.ChamaName,
		)
		if err != nil {
			continue
		}

		// Track unique chamas
		chamaCount[meeting.ChamaID] = true

		meetingData := map[string]interface{}{
			"id":          meeting.ID,
			"chamaId":     meeting.ChamaID,
			"chamaName":   meeting.ChamaName,
			"title":       meeting.Title,
			"description": "",
			"scheduledAt": meeting.ScheduledAt.Format(time.RFC3339),
			"duration":    0,
			"location":    "",
			"meetingUrl":  "",
			"meetingType": "physical",
			"status":      meeting.Status,
			"createdBy":   meeting.CreatedBy,
			"createdAt":   meeting.CreatedAt.Format(time.RFC3339),
			"creator": map[string]interface{}{
				"firstName": meeting.CreatorFirstName,
				"lastName":  meeting.CreatorLastName,
				"email":     meeting.CreatorEmail,
			},
		}

		// Handle nullable fields
		if meeting.Description.Valid {
			meetingData["description"] = meeting.Description.String
		}
		if meeting.Duration.Valid {
			meetingData["duration"] = meeting.Duration.Int64
		}
		if meeting.Location.Valid {
			meetingData["location"] = meeting.Location.String
		}
		if meeting.MeetingURL.Valid {
			meetingData["meetingUrl"] = meeting.MeetingURL.String
		}
		if meeting.MeetingType.Valid {
			meetingData["meetingType"] = meeting.MeetingType.String
		}

		meetings = append(meetings, meetingData)
	}

	// Get total count for pagination
	var totalCount int
	err = db.(*sql.DB).QueryRow(`
		SELECT COUNT(DISTINCT m.id)
		FROM meetings m
		JOIN chama_members cm ON m.chama_id = cm.chama_id
		WHERE cm.user_id = ? AND cm.is_active = TRUE
	`, userID).Scan(&totalCount)

	if err != nil {
		totalCount = len(meetings)
	}

	log.Printf("✅ Successfully fetched %d meetings for user %s from %d chamas", len(meetings), userID, len(chamaCount))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    meetings,
		"message": fmt.Sprintf("Found %d meetings from %d chamas", len(meetings), len(chamaCount)),
		"meta": map[string]interface{}{
			"total":  totalCount,
			"limit":  limit,
			"offset": offset,
			"chamas": len(chamaCount),
		},
	})
}

func CreateMeeting(c *gin.Context) {
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
		ChamaID     string `json:"chamaId" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		ScheduledAt string `json:"scheduledAt" binding:"required"`
		Duration    int    `json:"duration"`
		Location    string `json:"location"`
		MeetingURL  string `json:"meetingUrl"`
		Type        string `json:"type"`
		Agenda      string `json:"agenda"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Validate date time format
	meetingTime, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid date time format. Use RFC3339 format (e.g., 2024-01-01T15:30:00Z)",
		})
		return
	}

	// Check if meeting is in the future
	if meetingTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting date and time must be in the future",
		})
		return
	}

	// Calculate end time based on duration
	duration := req.Duration
	if duration <= 0 {
		duration = 60 // Default to 60 minutes
	}
	endTime := meetingTime.Add(time.Duration(duration) * time.Minute)

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

	// Generate meeting ID
	meetingID := fmt.Sprintf("meeting-%d", time.Now().UnixNano())

	// Set default values
	meetingType := req.Type
	if meetingType == "" {
		meetingType = "regular"
	}

	location := req.Location
	if location == "" {
		location = "TBD"
	}

	// Insert meeting into database
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO meetings (
			id, chama_id, title, description, scheduled_at, duration, location,
			meeting_url, status, created_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'scheduled', ?, CURRENT_TIMESTAMP)
	`, meetingID, req.ChamaID, req.Title, req.Description, meetingTime, duration, location, req.MeetingURL, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create meeting: " + err.Error(),
		})
		return
	}

	// Notify all chama members about the new meeting
	if notificationService != nil {
		log.Printf("Creating notifications for meeting %s in chama %s", meetingID, req.ChamaID)
		go func() {
			// Get all chama members
			rows, err := db.(*sql.DB).Query(`
				SELECT user_id FROM chama_members
				WHERE chama_id = ? AND user_id != ? AND is_active = TRUE
			`, req.ChamaID, userID)
			if err != nil {
				log.Printf("Failed to get chama members for notification: %v", err)
				return
			}
			defer rows.Close()

			// Get chama name for notification
			var chamaName string
			err = db.(*sql.DB).QueryRow("SELECT name FROM chamas WHERE id = ?", req.ChamaID).Scan(&chamaName)
			if err != nil {
				log.Printf("Failed to get chama name: %v, using default", err)
				chamaName = "Chama"
			}

			memberCount := 0
			// Send notification to each member
			for rows.Next() {
				var memberID string
				if err := rows.Scan(&memberID); err != nil {
					log.Printf("Failed to scan member ID: %v", err)
					continue
				}

				memberCount++
				log.Printf("Creating notification for member %s (%d/%d)", memberID, memberCount, 0) // We'll count total later

				// Create notification data
				data := map[string]interface{}{
					"meetingId":    meetingID,
					"chamaId":      req.ChamaID,
					"chamaName":    chamaName,
					"meetingTitle": req.Title,
					"description":  req.Description,
					"scheduledAt":  req.ScheduledAt,
					"duration":     duration,
					"location":     location,
					"meetingUrl":   req.MeetingURL,
					"meetingType":  meetingType,
				}

				// Send notification
				notification, err := notificationService.CreateNotification(
					memberID,
					services.NotificationTypeMeeting,
					fmt.Sprintf("New Meeting: %s", req.Title),
					fmt.Sprintf("A new meeting '%s' has been scheduled for %s in %s", req.Title, chamaName, meetingTime.Format("Jan 2, 2006 at 3:04 PM")),
					data,
					true, // sendPush
					false, // sendEmail
					false, // sendSMS
				)
				if err != nil {
					log.Printf("Failed to send meeting notification to user %s: %v", memberID, err)
				} else {
					log.Printf("Successfully created notification %s for user %s", notification.ID, memberID)
				}
			}
			log.Printf("Finished creating notifications for %d members", memberCount)
		}()
	} else {
		log.Printf("Notification service is nil, skipping meeting notifications")
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Meeting scheduled successfully! All chama members will be notified.",
		"data": map[string]interface{}{
			"id":          meetingID,
			"chamaId":     req.ChamaID,
			"title":       req.Title,
			"description": req.Description,
			"scheduledAt": req.ScheduledAt,
			"endsAt":      endTime.Format(time.RFC3339), // Calculated end time
			"duration":    duration,
			"location":    location,
			"meetingUrl":  req.MeetingURL,
			"status":      "scheduled",
			"createdBy":   userID,
			"createdAt":   time.Now().Format(time.RFC3339),
		},
	})
}

func GetMeeting(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Get meeting endpoint - coming soon",
	})
}

func UpdateMeeting(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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

	var req struct {
		Status        string    `json:"status"`
		ConductedAt   time.Time `json:"conductedAt"`
		AttendeeCount int       `json:"attendeeCount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Update meeting in database - only update provided fields
	query := `UPDATE meetings SET updated_at = CURRENT_TIMESTAMP`
	params := []interface{}{}
	paramCount := 1

	if req.Status != "" {
		query += `, status = ?`
		params = append(params, req.Status)
		paramCount++
	}

	if !req.ConductedAt.IsZero() {
		query += `, conducted_at = ?`
		params = append(params, req.ConductedAt)
		paramCount++
	}

	if req.AttendeeCount > 0 {
		query += `, attendee_count = ?`
		params = append(params, req.AttendeeCount)
		paramCount++
	}

	query += ` WHERE id = ?`
	params = append(params, meetingID)

	result, err := db.(*sql.DB).Exec(query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update meeting: " + err.Error(),
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check update result: " + err.Error(),
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Meeting not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Meeting updated successfully",
	})
}

func DeleteMeeting(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delete meeting endpoint - coming soon",
	})
}

// Join meeting endpoint - fully functional for production

func JoinMeeting(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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

	// Get meeting details
	var meeting struct {
		ID          string    `json:"id"`
		ChamaID     string    `json:"chamaId"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		ScheduledAt time.Time `json:"scheduledAt"`
		Duration    int       `json:"duration"`
		Location    string    `json:"location"`
		MeetingURL  string    `json:"meetingUrl"`
		MeetingType string    `json:"meetingType"`
		Status      string    `json:"status"`
		CreatedBy   string    `json:"createdBy"`
	}

	err := db.(*sql.DB).QueryRow(`
		SELECT id, chama_id, title, description, scheduled_at, duration,
			   location, meeting_url, meeting_type, status, created_by
		FROM meetings
		WHERE id = ?
	`, meetingID).Scan(
		&meeting.ID, &meeting.ChamaID, &meeting.Title, &meeting.Description,
		&meeting.ScheduledAt, &meeting.Duration, &meeting.Location,
		&meeting.MeetingURL, &meeting.MeetingType, &meeting.Status, &meeting.CreatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Meeting not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch meeting details: " + err.Error(),
			})
		}
		return
	}

	// Check if user is a member of the chama
	var membershipExists bool
	err = db.(*sql.DB).QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM chama_members
			WHERE chama_id = ? AND user_id = ? AND is_active = TRUE
		)
	`, meeting.ChamaID, userID).Scan(&membershipExists)
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

	// Check if meeting is active or can be joined
	now := time.Now()
	meetingEndTime := meeting.ScheduledAt.Add(time.Duration(meeting.Duration) * time.Minute)
	tenMinutesBefore := meeting.ScheduledAt.Add(-10 * time.Minute) // Updated to match frontend

	// Enhanced join logic - users can join from 10 minutes before until meeting ends
	canJoin := now.After(tenMinutesBefore) && now.Before(meetingEndTime.Add(1*time.Minute)) // Add 1 minute buffer
	if !canJoin {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting cannot be joined at this time. Join window is 10 minutes before start until meeting ends.",
			"data": map[string]interface{}{
				"meetingStart":     meeting.ScheduledAt.Format(time.RFC3339),
				"meetingEnd":       meetingEndTime.Format(time.RFC3339),
				"joinWindowStart":  tenMinutesBefore.Format(time.RFC3339),
				"joinWindowEnd":    meetingEndTime.Add(1 * time.Minute).Format(time.RFC3339),
				"currentTime":      now.Format(time.RFC3339),
				"isMeetingActive":  now.After(meeting.ScheduledAt) && now.Before(meetingEndTime),
				"remainingMinutes": int(meetingEndTime.Sub(now).Minutes()),
			},
		})
		return
	}

	// Return meeting join information based on meeting type
	response := gin.H{
		"success": true,
		"message": "Meeting join information retrieved successfully",
		"data": map[string]interface{}{
			"meeting": map[string]interface{}{
				"id":          meeting.ID,
				"title":       meeting.Title,
				"description": meeting.Description,
				"scheduledAt": meeting.ScheduledAt.Format(time.RFC3339),
				"duration":    meeting.Duration,
				"location":    meeting.Location,
				"meetingUrl":  meeting.MeetingURL,
				"meetingType": meeting.MeetingType,
				"type":        meeting.MeetingType,
				"status":      meeting.Status,
			},
			"joinInfo": map[string]interface{}{
				"canJoin":          true,
				"joinWindowStart":  tenMinutesBefore.Format(time.RFC3339),
				"joinWindowEnd":    meetingEndTime.Add(1 * time.Minute).Format(time.RFC3339),
				"currentTime":      now.Format(time.RFC3339),
				"isMeetingActive":  now.After(meeting.ScheduledAt) && now.Before(meetingEndTime),
				"remainingMinutes": int(meetingEndTime.Sub(now).Minutes()),
			},
		},
	}

	// Add specific join instructions based on meeting type
	switch meeting.MeetingType {
	case "virtual":
		if meeting.MeetingURL != "" {
			response["data"].(map[string]interface{})["joinInstructions"] = map[string]interface{}{
				"type":        "virtual",
				"instruction": "Click the meeting URL to join the virtual meeting",
				"meetingUrl":  meeting.MeetingURL,
			}
		} else {
			response["data"].(map[string]interface{})["joinInstructions"] = map[string]interface{}{
				"type":        "virtual",
				"instruction": "Virtual meeting details will be provided by the meeting organizer",
			}
		}
	case "physical":
		response["data"].(map[string]interface{})["joinInstructions"] = map[string]interface{}{
			"type":        "physical",
			"instruction": "Please arrive at the specified location",
			"location":    meeting.Location,
		}
	case "hybrid":
		response["data"].(map[string]interface{})["joinInstructions"] = map[string]interface{}{
			"type":        "hybrid",
			"instruction": "You can join either virtually or physically",
			"location":    meeting.Location,
			"meetingUrl":  meeting.MeetingURL,
		}
	default:
		response["data"].(map[string]interface{})["joinInstructions"] = map[string]interface{}{
			"type":        "unknown",
			"instruction": "Please contact the meeting organizer for join instructions",
		}
	}

	c.JSON(http.StatusOK, response)
}

// InitializeMeetingService initializes the meeting service with LiveKit integration
func InitializeMeetingService(db *sql.DB, notifService *services.NotificationService) {
	// Set the notification service
	notificationService = notifService

	// Get LiveKit configuration
	config := services.GetLiveKitConfig()

	// Initialize LiveKit service
	livekitService := services.NewLiveKitService(config.WSURL, config.APIKey, config.APISecret)

	// Initialize calendar service (optional)
	var calendarService *services.CalendarService
	// TODO: Initialize calendar service with credentials when available

	// Initialize meeting service
	meetingService = services.NewMeetingService(db, livekitService, calendarService)

	// Initialize and start scheduler service
	schedulerService = services.NewSchedulerService(db, meetingService)
	schedulerService.Start(time.Minute) // Check every minute

	// Fix existing meetings without room names
	// log.Println("Ensuring all virtual meetings have room names...")
	err := meetingService.EnsureVirtualMeetingsHaveRoomNames()
	if err != nil {
		log.Printf("Warning: Failed to ensure room names for existing meetings: %v", err)
	}

	log.Println("Meeting service and scheduler initialized successfully")
}

// CreateMeetingWithLiveKit creates a new meeting with LiveKit integration
func CreateMeetingWithLiveKit(c *gin.Context) {
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
		ChamaID          string `json:"chamaId" binding:"required"`
		Title            string `json:"title" binding:"required"`
		Description      string `json:"description"`
		ScheduledAt      string `json:"scheduledAt" binding:"required"`
		Duration         int    `json:"duration"`
		Location         string `json:"location"`
		MeetingURL       string `json:"meetingUrl"`
		MeetingType      string `json:"meetingType" binding:"required"` // 'physical', 'virtual', 'hybrid'
		RecordingEnabled bool   `json:"recordingEnabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Validate meeting type
	if req.MeetingType != "physical" && req.MeetingType != "virtual" && req.MeetingType != "hybrid" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid meeting type. Must be 'physical', 'virtual', or 'hybrid'",
		})
		return
	}

	// Parse scheduled time
	meetingTime, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid scheduled time format. Use RFC3339 format",
		})
		return
	}

	// Check if meeting is in the future
	if meetingTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting cannot be scheduled in the past",
		})
		return
	}

	// Set default duration if not provided
	duration := req.Duration
	if duration <= 0 {
		duration = 60 // Default 1 hour
	}

	// Create meeting object
	meetingID := uuid.New().String()
	meeting := &services.Meeting{
		ID:               meetingID,
		ChamaID:          req.ChamaID,
		Title:            req.Title,
		Description:      req.Description,
		ScheduledAt:      meetingTime,
		Duration:         duration,
		Location:         req.Location,
		MeetingURL:       req.MeetingURL,
		MeetingType:      req.MeetingType,
		Status:           "scheduled",
		RecordingEnabled: req.RecordingEnabled,
		CreatedBy:        userID.(string),
	}

	// Create meeting with LiveKit integration
	err = meetingService.CreateVirtualMeeting(meeting)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create meeting: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Meeting created successfully with LiveKit integration!",
		"data": gin.H{
			"id":               meeting.ID,
			"chamaId":          meeting.ChamaID,
			"title":            meeting.Title,
			"description":      meeting.Description,
			"scheduledAt":      meeting.ScheduledAt.Format(time.RFC3339),
			"duration":         meeting.Duration,
			"location":         meeting.Location,
			"meetingUrl":       meeting.MeetingURL,
			"meetingType":      meeting.MeetingType,
			"livekitRoomName":  meeting.LiveKitRoomName,
			"status":           meeting.Status,
			"recordingEnabled": meeting.RecordingEnabled,
			"createdBy":        meeting.CreatedBy,
		},
	})
}

// StartMeeting starts a meeting
func StartMeeting(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
		})
		return
	}

	err := meetingService.StartMeeting(meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start meeting: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Meeting started successfully",
	})
}

// EndMeeting ends a meeting
func EndMeeting(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
		})
		return
	}

	// Get user ID from context
	userID, userExists := c.Get("userID")
	if !userExists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get DB
	db, dbExists := c.Get("db")
	if !dbExists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Load meeting to get chamaId and current status
	meeting, getErr := meetingService.GetMeeting(meetingID)
	if getErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to get meeting: " + getErr.Error(),
		})
		return
	}

	// Strict role check: only chairperson or secretary can end meetings
	var role string
	roleErr := db.(*sql.DB).QueryRow(`
		SELECT role FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = TRUE
	`, meeting.ChamaID, userID.(string)).Scan(&role)
	if roleErr != nil {
		if roleErr == sql.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Only chairperson or secretary can end meetings",
				"code":    "FORBIDDEN",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify user role: " + roleErr.Error(),
		})
		return
	}

	if role != "chairperson" && role != "secretary" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson or secretary can end meetings",
			"code":    "FORBIDDEN",
		})
		return
	}

	// Minutes approval check removed per updated rule: if minutes were approved earlier, great;
	// otherwise, only attendance requirement is enforced before ending.

	// Prereq 2: At least one attendance record exists
	var attendanceCount int
	attErr := db.(*sql.DB).QueryRow(`
		SELECT COUNT(*) FROM meeting_attendance WHERE meeting_id = ?
	`, meetingID).Scan(&attendanceCount)
	if attErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify attendance: " + attErr.Error(),
		})
		return
	}
	if attendanceCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Attendance must be marked before ending the meeting",
			"code":    "ATTENDANCE_REQUIRED",
		})
		return
	}

	err := meetingService.EndMeeting(meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to end meeting: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Meeting ended successfully",
	})
}

// GetGoogleCalendarAddEventURL returns a pre-filled Google Calendar event URL for a meeting
func GetGoogleCalendarAddEventURL(c *gin.Context) {
    meetingID := c.Param("id")
    if meetingID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Meeting ID is required",
        })
        return
    }

    // Get meeting details
    meeting, err := meetingService.GetMeeting(meetingID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Failed to get meeting: " + err.Error(),
        })
        return
    }

    // Build a summary and description (include chama name)
    // Fetch chama name for better labeling
    db, dbExists := c.Get("db")
    var chamaName string
    if dbExists {
        _ = db.(*sql.DB).QueryRow("SELECT name FROM chamas WHERE id = ?", meeting.ChamaID).Scan(&chamaName)
    }
    if chamaName == "" {
        chamaName = "Chama"
    }

    summary := meeting.Title
    if summary == "" {
        summary = "Chama Meeting"
    }
    summary = fmt.Sprintf("%s — %s", summary, chamaName)

    description := fmt.Sprintf("%s\n\nLocation: %s.", "Chama meeting.", meeting.Location)
    if meeting.MeetingURL != "" {
        description += "\\nJoin: " + meeting.MeetingURL
    }

    // Compute start/end using scheduled time and duration in EAT
    eat, _ := time.LoadLocation("Africa/Nairobi")
    start := meeting.ScheduledAt.In(eat)
    end := meeting.ScheduledAt.In(eat).Add(time.Duration(max(1, meeting.Duration)) * time.Minute)

    // Google Calendar template URL
    // https://calendar.google.com/calendar/render?action=TEMPLATE&text=...&dates=YYYYMMDDTHHMMSSZ/YYY...&details=...&location=...
    const template = "https://calendar.google.com/calendar/render"
    params := url.Values{}
    params.Set("action", "TEMPLATE")
    params.Set("text", summary)
    params.Set("details", description)
    if meeting.Location != "" {
        params.Set("location", meeting.Location)
    }

    // Provide local datetime without Z and set ctz to Africa/Nairobi for accurate display
    toLocal := func(t time.Time) string { return t.Format("20060102T150405") }
    params.Set("dates", fmt.Sprintf("%s/%s", toLocal(start), toLocal(end)))
    params.Set("ctz", "Africa/Nairobi")

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "url": template + "?" + params.Encode(),
        },
    })
}

// CreateGoogleCalendarEvent creates the event in the user's Google Calendar with 30/10/0 minute reminders
func CreateGoogleCalendarEvent(c *gin.Context) {
    meetingID := c.Param("id")
    if meetingID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Meeting ID is required"})
        return
    }

    // Auth user
    userID := c.GetString("userID")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
        return
    }

    // Get DB and services
    db, exists := c.Get("db")
    if !exists {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Database connection not available"})
        return
    }

    // Load meeting and chama name
    meeting, err := meetingService.GetMeeting(meetingID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to get meeting: " + err.Error()})
        return
    }

    // Fetch chama name
    var chamaName string
    err = db.(*sql.DB).QueryRow("SELECT name FROM chamas WHERE id = ?", meeting.ChamaID).Scan(&chamaName)
    if err != nil {
        chamaName = "Chama"
    }

    // Get the user's stored Google tokens via GoogleDriveService storage (shared token store)
    driveService := services.NewGoogleDriveService(db.(*sql.DB))
    token, err := driveService.GetUserTokens(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Google account not connected for this user"})
        return
    }

    // Initialize CalendarService with credentials from env JSON
    creds := os.Getenv("GOOGLE_CALENDAR_CREDENTIALS_JSON")
    if creds == "" {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Calendar credentials not configured"})
        return
    }
    calService, err := services.NewCalendarService([]byte(creds))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to init calendar service: " + err.Error()})
        return
    }
    if err := calService.InitializeWithToken(token); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to authorize calendar: " + err.Error()})
        return
    }

    // Build event with accurate start/end and chama name in title
    title := fmt.Sprintf("%s — %s", meeting.Title, chamaName)
    desc := meeting.Description
    if meeting.MeetingURL != "" {
        desc = fmt.Sprintf("%s\n\nJoin: %s", desc, meeting.MeetingURL)
    }

    ev := &services.CalendarEvent{
        Title:       title,
        Description: desc,
        StartTime:   meeting.ScheduledAt,
        EndTime:     meeting.ScheduledAt.Add(time.Duration(max(1, meeting.Duration)) * time.Minute),
        Location:    meeting.Location,
        MeetingURL:  meeting.MeetingURL,
    }

    // Use primary calendar and reminders 30,10,0 minutes
    created, err := calService.CreateEventWithReminders("primary", ev, []int{30, 10, 0})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create calendar event: " + err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"eventId": created.Id, "htmlLink": created.HtmlLink}})
}

// JoinMeetingWithLiveKit generates a LiveKit access token for joining a meeting
func JoinMeetingWithLiveKit(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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
		UserRole string `json:"userRole"` // 'chairperson', 'secretary', 'treasurer', 'member'
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to member role if not specified
		req.UserRole = "member"
	}

	// Generate LiveKit access token
	token, err := meetingService.GenerateJoinToken(meetingID, userID.(string), req.UserRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to generate join token: " + err.Error(),
		})
		return
	}

	// Get meeting details
	meeting, err := meetingService.GetMeeting(meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get meeting details: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Join token generated successfully",
		"data": gin.H{
			"token":            token,
			"roomName":         meeting.LiveKitRoomName,
			"wsUrl":            services.GetLiveKitConfig().WSURL,
			"meetingId":        meetingID,
			"userRole":         req.UserRole,
			"meetingTitle":     meeting.Title,
			"meetingType":      meeting.MeetingType,
			"recordingEnabled": meeting.RecordingEnabled,
		},
	})
}

// JoinMeetingWithJitsi generates a Jitsi Meet room URL and authentication data for joining a meeting
func JoinMeetingWithJitsi(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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
		UserRole string `json:"userRole"` // 'chairperson', 'secretary', 'treasurer', 'member'
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to member role if not specified
		req.UserRole = "member"
	}

	// Get meeting details
	meeting, err := meetingService.GetMeeting(meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get meeting details: " + err.Error(),
		})
		return
	}

	// Generate Jitsi room data
	jitsiData, err := meetingService.GenerateJitsiRoomData(meetingID, userID.(string), req.UserRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to generate Jitsi room data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Jitsi room data generated successfully",
		"data": gin.H{
			"roomName":         jitsiData.RoomName,
			"joinUrl":          jitsiData.JoinURL,
			"roomPassword":     jitsiData.RoomPassword,
			"isModerator":      jitsiData.IsModerator,
			"displayName":      jitsiData.DisplayName,
			"chamaId":          meeting.ChamaID,
			"meetingId":        meetingID,
			"userRole":         req.UserRole,
			"meetingTitle":     meeting.Title,
			"meetingType":      meeting.MeetingType,
			"recordingEnabled": meeting.RecordingEnabled,
		},
	})
}

// MarkAttendance marks a user's attendance for a meeting
func MarkAttendance(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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
		AttendanceType string `json:"attendanceType" binding:"required"` // 'physical', 'virtual'
		IsPresent      bool   `json:"isPresent"`
		UserID         string `json:"userId"` // Optional: for secretary to mark attendance for others
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Determine which user's attendance to mark
	targetUserID := userID.(string) // Default to current user
	if req.UserID != "" {
		// Secretary is marking attendance for another user
		targetUserID = req.UserID

		// Verify that the current user has permission to mark attendance for others
		// This should be a secretary or chairperson
		// TODO: Add proper role verification here
		log.Printf("User %s is marking attendance for user %s", userID.(string), targetUserID)
	}

	err := meetingService.MarkAttendance(meetingID, targetUserID, req.AttendanceType, req.IsPresent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to mark attendance: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Attendance marked successfully",
	})
}

// GetMeetingAttendance retrieves attendance records for a meeting
func GetMeetingAttendance(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
		})
		return
	}

	attendances, err := meetingService.GetMeetingAttendance(meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get meeting attendance: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    attendances,
	})
}

// UploadMeetingDocument handles document uploads for meetings
func UploadMeetingDocument(c *gin.Context) {
	log.Printf("📄 UploadMeetingDocument called")

	meetingID := c.Param("id")
	if meetingID == "" {
		log.Printf("📄 Missing meeting ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
		})
		return
	}

	log.Printf("📄 Meeting ID: %s", meetingID)

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Debug: Log request details
	log.Printf("📄 Document upload request - Content-Type: %s", c.Request.Header.Get("Content-Type"))
	log.Printf("📄 Document upload request - Content-Length: %d", c.Request.ContentLength)
	log.Printf("📄 Document upload request - Method: %s", c.Request.Method)

	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		log.Printf("📄 Failed to parse multipart form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to parse multipart form: " + err.Error(),
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No file provided: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file size (10MB max)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "File too large. Maximum size is 10MB",
		})
		return
	}

	// Create uploads directory
	uploadsDir := "./uploads/meetings"
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create upload directory: " + err.Error(),
		})
		return
	}

	// Generate unique filename
	fileExt := filepath.Ext(header.Filename)
	fileName := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), fileExt)
	filePath := filepath.Join(uploadsDir, fileName)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save file: " + err.Error(),
		})
		return
	}

	// Create file URL
	fileURL := fmt.Sprintf("/uploads/meetings/%s", fileName)

	// Get form values
	documentType := c.PostForm("documentType")
	description := c.PostForm("description")

	if documentType == "" {
		documentType = "meeting_document"
	}

	// Save document info to database
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	documentID := uuid.New().String()
	_, err = db.(*sql.DB).Exec(`
		INSERT INTO meeting_documents (
			id, meeting_id, uploaded_by, file_name, file_path, file_url,
			file_size, file_type, document_type, description, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, documentID, meetingID, userID, header.Filename, filePath, fileURL,
		header.Size, header.Header.Get("Content-Type"), documentType, description)
	if err != nil {
		// Clean up uploaded file if database insert fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save document info: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document uploaded successfully",
		"data": map[string]interface{}{
			"id":           documentID,
			"meetingId":    meetingID,
			"fileName":     header.Filename,
			"fileSize":     header.Size,
			"fileType":     header.Header.Get("Content-Type"),
			"documentType": documentType,
			"description":  description,
			"url":          fileURL,
			"uploadedBy":   userID,
			"uploadedAt":   time.Now().Format(time.RFC3339),
		},
	})
}

// GetMeetingDocuments retrieves all documents for a meeting
func GetMeetingDocuments(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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

	// Query documents
	rows, err := db.(*sql.DB).Query(`
		SELECT id, meeting_id, uploaded_by, file_name, file_url, file_size,
			   file_type, document_type, description, created_at
		FROM meeting_documents
		WHERE meeting_id = ?
		ORDER BY created_at DESC
	`, meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch documents: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var documents []map[string]interface{}
	for rows.Next() {
		var doc struct {
			ID           string    `json:"id"`
			MeetingID    string    `json:"meetingId"`
			UploadedBy   string    `json:"uploadedBy"`
			FileName     string    `json:"fileName"`
			FileURL      string    `json:"fileUrl"`
			FileSize     int64     `json:"fileSize"`
			FileType     string    `json:"fileType"`
			DocumentType string    `json:"documentType"`
			Description  string    `json:"description"`
			CreatedAt    time.Time `json:"createdAt"`
		}

		err := rows.Scan(&doc.ID, &doc.MeetingID, &doc.UploadedBy, &doc.FileName,
			&doc.FileURL, &doc.FileSize, &doc.FileType, &doc.DocumentType,
			&doc.Description, &doc.CreatedAt)
		if err != nil {
			continue
		}

		documents = append(documents, map[string]interface{}{
			"id":           doc.ID,
			"meetingId":    doc.MeetingID,
			"uploadedBy":   doc.UploadedBy,
			"name":         doc.FileName,
			"url":          doc.FileURL,
			"size":         doc.FileSize,
			"type":         doc.FileType,
			"documentType": doc.DocumentType,
			"description":  doc.Description,
			"uploadedAt":   doc.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    documents,
	})
}

// DeleteMeetingDocument deletes a document from a meeting
func DeleteMeetingDocument(c *gin.Context) {
	meetingID := c.Param("id")
	documentID := c.Param("docId")

	if meetingID == "" || documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID and Document ID are required",
		})
		return
	}

	// Get user ID from context for authentication
	_, exists := c.Get("userID")
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

	// Get document info first to delete the file
	var filePath string
	err := db.(*sql.DB).QueryRow(`
		SELECT file_path FROM meeting_documents
		WHERE id = ? AND meeting_id = ?
	`, documentID, meetingID).Scan(&filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Document not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to find document: " + err.Error(),
			})
		}
		return
	}

	// Delete from database first
	_, err = db.(*sql.DB).Exec(`
		DELETE FROM meeting_documents
		WHERE id = ? AND meeting_id = ?
	`, documentID, meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete document from database: " + err.Error(),
		})
		return
	}

	// Delete physical file
	if filePath != "" {
		os.Remove(filePath) // Ignore error if file doesn't exist
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document deleted successfully",
	})
}

// SaveMeetingMinutes saves meeting notes/minutes
func SaveMeetingMinutes(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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
		Content   string `json:"content" binding:"required"`
		Status    string `json:"status"`
		MeetingID string `json:"meetingId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Set default status
	if req.Status == "" {
		req.Status = "draft"
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

	// Check if minutes already exist for this meeting
	var existingID string
	err := db.(*sql.DB).QueryRow(`
		SELECT id FROM meeting_minutes WHERE meeting_id = ?
	`, meetingID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new minutes
		minutesID := uuid.New().String()
		_, err = db.(*sql.DB).Exec(`
			INSERT INTO meeting_minutes (
				id, meeting_id, content, status, taken_by, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, minutesID, meetingID, req.Content, req.Status, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to save meeting minutes: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Meeting minutes saved successfully",
			"data": map[string]interface{}{
				"id":        minutesID,
				"meetingId": meetingID,
				"content":   req.Content,
				"status":    req.Status,
				"takenBy":   userID,
				"createdAt": time.Now().Format(time.RFC3339),
			},
		})
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check existing minutes: " + err.Error(),
		})
		return
	} else {
		// Update existing minutes
		_, err = db.(*sql.DB).Exec(`
			UPDATE meeting_minutes
			SET content = ?, status = ?, taken_by = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, req.Content, req.Status, userID, existingID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update meeting minutes: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Meeting minutes updated successfully",
			"data": map[string]interface{}{
				"id":        existingID,
				"meetingId": meetingID,
				"content":   req.Content,
				"status":    req.Status,
				"takenBy":   userID,
				"updatedAt": time.Now().Format(time.RFC3339),
			},
		})
	}
}

// UpdateMeetingMinutes updates meeting minutes fields (content/status). If minutes do not exist, creates them.
func UpdateMeetingMinutes(c *gin.Context) {
    meetingID := c.Param("id")
    if meetingID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Meeting ID is required",
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
        Content string `json:"content"`
        Status  string `json:"status"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Invalid request data: " + err.Error(),
        })
        return
    }

    // Default status if omitted
    if req.Status == "" {
        req.Status = "draft"
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

    // Check if minutes exist
    var existingID string
    err := db.(*sql.DB).QueryRow(`
        SELECT id FROM meeting_minutes WHERE meeting_id = ?
    `, meetingID).Scan(&existingID)

    if err == sql.ErrNoRows {
        // Create if not exists
        minutesID := uuid.New().String()
        _, err = db.(*sql.DB).Exec(`
            INSERT INTO meeting_minutes (
                id, meeting_id, content, status, taken_by, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        `, minutesID, meetingID, req.Content, req.Status, userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "success": false,
                "error":   "Failed to create meeting minutes: " + err.Error(),
            })
            return
        }

        c.JSON(http.StatusCreated, gin.H{
            "success": true,
            "message": "Meeting minutes created",
            "data": map[string]interface{}{
                "id":        minutesID,
                "meetingId": meetingID,
                "content":   req.Content,
                "status":    req.Status,
                "takenBy":   userID,
                "createdAt": time.Now().Format(time.RFC3339),
            },
        })
        return
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   "Failed to check existing minutes: " + err.Error(),
        })
        return
    }

    // Update existing
    _, err = db.(*sql.DB).Exec(`
        UPDATE meeting_minutes
        SET content = COALESCE(NULLIF(?, ''), content),
            status = ?,
            taken_by = ?,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `, req.Content, req.Status, userID, existingID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   "Failed to update meeting minutes: " + err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Meeting minutes updated",
        "data": map[string]interface{}{
            "id":        existingID,
            "meetingId": meetingID,
            "content":   req.Content,
            "status":    req.Status,
            "takenBy":   userID,
            "updatedAt": time.Now().Format(time.RFC3339),
        },
    })
}

// GetMeetingMinutes retrieves meeting minutes
func GetMeetingMinutes(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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

	var minutes struct {
		ID        string    `json:"id"`
		MeetingID string    `json:"meetingId"`
		Content   string    `json:"content"`
		Status    string    `json:"status"`
		TakenBy   string    `json:"takenBy"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	err := db.(*sql.DB).QueryRow(`
		SELECT id, meeting_id, content, status, taken_by, created_at, updated_at
		FROM meeting_minutes
		WHERE meeting_id = ?
	`, meetingID).Scan(&minutes.ID, &minutes.MeetingID, &minutes.Content, &minutes.Status,
		&minutes.TakenBy, &minutes.CreatedAt, &minutes.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    nil,
				"message": "No minutes found for this meeting",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch meeting minutes: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":        minutes.ID,
			"meetingId": minutes.MeetingID,
			"content":   minutes.Content,
			"status":    minutes.Status,
			"takenBy":   minutes.TakenBy,
			"createdAt": minutes.CreatedAt.Format(time.RFC3339),
			"updatedAt": minutes.UpdatedAt.Format(time.RFC3339),
		},
	})
}

// CreateMeetingWithCalendar creates a new meeting with Google Calendar integration
func CreateMeetingWithCalendar(c *gin.Context) {
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
		ChamaID          string   `json:"chamaId" binding:"required"`
		Title            string   `json:"title" binding:"required"`
		Description      string   `json:"description"`
		ScheduledAt      string   `json:"scheduledAt" binding:"required"`
		Duration         int      `json:"duration" binding:"required"`
		Location         string   `json:"location"`
		MeetingURL       string   `json:"meetingUrl"`
		MeetingType      string   `json:"meetingType" binding:"required"`
		RecordingEnabled bool     `json:"recordingEnabled"`
		AttendeeEmails   []string `json:"attendeeEmails"`
		CalendarID       string   `json:"calendarId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Parse scheduled time
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid scheduled time format. Use RFC3339 format",
		})
		return
	}

	// Create meeting object
	meeting := &services.Meeting{
		ID:               uuid.New().String(),
		ChamaID:          req.ChamaID,
		Title:            req.Title,
		Description:      req.Description,
		ScheduledAt:      scheduledAt,
		Duration:         req.Duration,
		Location:         req.Location,
		MeetingURL:       req.MeetingURL,
		MeetingType:      req.MeetingType,
		Status:           "scheduled",
		RecordingEnabled: req.RecordingEnabled,
		CreatedBy:        userID.(string),
	}

	// Create meeting with calendar integration
	err = meetingService.CreateMeetingWithCalendar(meeting, req.AttendeeEmails, req.CalendarID)
	if err != nil {
		log.Printf("Failed to create meeting with calendar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create meeting: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Meeting created successfully with calendar integration",
		"data": gin.H{
			"id":               meeting.ID,
			"chamaId":          meeting.ChamaID,
			"title":            meeting.Title,
			"description":      meeting.Description,
			"scheduledAt":      meeting.ScheduledAt.Format(time.RFC3339),
			"duration":         meeting.Duration,
			"location":         meeting.Location,
			"meetingUrl":       meeting.MeetingURL,
			"meetingType":      meeting.MeetingType,
			"livekitRoomName":  meeting.LiveKitRoomName,
			"status":           meeting.Status,
			"recordingEnabled": meeting.RecordingEnabled,
			"createdBy":        meeting.CreatedBy,
		},
	})
}

// PreviewMeeting allows chairpersons and secretaries to preview a meeting room
func PreviewMeeting(c *gin.Context) {
	meetingID := c.Param("id")
	if meetingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Meeting ID is required",
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

	// Get user role from query parameter or request body
	userRole := c.Query("role")
	if userRole == "" {
		var req struct {
			UserRole string `json:"userRole"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			userRole = req.UserRole
		}
	}

	// Default to member if no role specified (will be rejected by service)
	if userRole == "" {
		userRole = "member"
	}

	// Debug logging before calling preview
	log.Printf("Attempting to preview meeting %s for user %s with role %s", meetingID, userID.(string), userRole)

	// Get meeting preview information
	previewInfo, err := meetingService.GetMeetingPreviewInfo(meetingID, userID.(string), userRole)
	if err != nil {
		log.Printf("Preview error for meeting %s: %v", meetingID, err)
		if err.Error() == "only chairpersons and secretaries can preview meetings" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to get meeting preview: " + err.Error(),
			})
		}
		return
	}

	// Debug logging
	log.Printf("Preview info generated for meeting %s, user %s, role %s", meetingID, userID.(string), userRole)
	log.Printf("Preview data: accessToken exists: %v, wsURL: %v, roomName: %v",
		previewInfo["accessToken"] != nil && previewInfo["accessToken"] != "",
		previewInfo["wsURL"],
		previewInfo["roomName"])

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Meeting preview information retrieved successfully",
		"data":    previewInfo,
	})
}

// TestLiveKitConnection tests the LiveKit configuration and connection
func TestLiveKitConnection(c *gin.Context) {
	// Get LiveKit configuration
	config := services.GetLiveKitConfig()

	// Initialize LiveKit service
	livekitService := services.NewLiveKitService(config.WSURL, config.APIKey, config.APISecret)

	// Validate configuration
	err := livekitService.ValidateConfig()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "LiveKit configuration invalid: " + err.Error(),
			"config": gin.H{
				"wsUrl":           config.WSURL,
				"hasApiKey":       config.APIKey != "",
				"hasApiSecret":    config.APISecret != "",
				"apiKeyLength":    len(config.APIKey),
				"apiSecretLength": len(config.APISecret),
			},
		})
		return
	}

	// Try to generate a test token
	testToken, err := livekitService.GenerateAccessToken("test-room", "test-user", "member")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to generate test token: " + err.Error(),
			"config": gin.H{
				"wsUrl":           config.WSURL,
				"hasApiKey":       config.APIKey != "",
				"hasApiSecret":    config.APISecret != "",
				"apiKeyLength":    len(config.APIKey),
				"apiSecretLength": len(config.APISecret),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "LiveKit configuration is valid",
		"config": gin.H{
			"wsUrl":           config.WSURL,
			"hasApiKey":       config.APIKey != "",
			"hasApiSecret":    config.APISecret != "",
			"apiKeyLength":    len(config.APIKey),
			"apiSecretLength": len(config.APISecret),
		},
		"testToken": gin.H{
			"generated":    true,
			"tokenLength":  len(testToken),
			"tokenPreview": testToken[:50] + "...",
		},
	})
}
