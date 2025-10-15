package services

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

// MeetingService handles meeting-related operations
type MeetingService struct {
	db              *sql.DB
	livekitService  *LiveKitService
	calendarService *CalendarService
	roomGenerator   *RoomNameGenerator
}

// RoomNameGenerator provides methods for generating consistent room names
type RoomNameGenerator struct{}

// NewRoomNameGenerator creates a new room name generator
func NewRoomNameGenerator() *RoomNameGenerator {
	return &RoomNameGenerator{}
}

// GenerateRoomName generates a consistent room name for a meeting
func (g *RoomNameGenerator) GenerateRoomName(chamaID, meetingID string) string {
	// Clean the IDs to ensure they're safe for room names
	cleanChamaID := g.cleanID(chamaID)
	cleanMeetingID := g.cleanID(meetingID)

	roomName := fmt.Sprintf("chama_%s_meeting_%s", cleanChamaID, cleanMeetingID)

	// Ensure the room name is valid (alphanumeric, hyphens, underscores only)
	return g.sanitizeRoomName(roomName)
}

// cleanID removes special characters and keeps only alphanumeric and safe characters
func (g *RoomNameGenerator) cleanID(id string) string {
	// Remove common prefixes
	id = strings.TrimPrefix(id, "meeting-")
	id = strings.TrimPrefix(id, "chama-")

	// Keep only alphanumeric characters and hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-]`)
	cleaned := reg.ReplaceAllString(id, "")

	// Remove consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	cleaned = reg.ReplaceAllString(cleaned, "-")

	// Trim hyphens from start and end
	cleaned = strings.Trim(cleaned, "-")

	return cleaned
}

// sanitizeRoomName ensures the room name meets LiveKit requirements
func (g *RoomNameGenerator) sanitizeRoomName(roomName string) string {
	// LiveKit room names should be alphanumeric with underscores and hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	sanitized := reg.ReplaceAllString(roomName, "_")

	// Remove consecutive underscores
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_-")

	// Ensure minimum length
	if len(sanitized) < 3 {
		sanitized = sanitized + "_room"
	}

	// Ensure maximum length (LiveKit has limits)
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}

	return sanitized
}

// NewMeetingService creates a new meeting service instance
func NewMeetingService(db *sql.DB, livekitService *LiveKitService, calendarService *CalendarService) *MeetingService {
	return &MeetingService{
		db:              db,
		livekitService:  livekitService,
		calendarService: calendarService,
		roomGenerator:   NewRoomNameGenerator(),
	}
}

// Meeting represents a meeting with LiveKit integration
type Meeting struct {
	ID               string     `json:"id"`
	ChamaID          string     `json:"chamaId"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	ScheduledAt      time.Time  `json:"scheduledAt"`
	Duration         int        `json:"duration"`
	Location         string     `json:"location"`
	MeetingURL       string     `json:"meetingUrl"`
	MeetingType      string     `json:"meetingType"`
	LiveKitRoomName  string     `json:"livekitRoomName"`
	LiveKitRoomID    string     `json:"livekitRoomId"`
	Status           string     `json:"status"`
	StartedAt        *time.Time `json:"startedAt"`
	EndedAt          *time.Time `json:"endedAt"`
	RecordingEnabled bool       `json:"recordingEnabled"`
	RecordingURL     *string    `json:"recordingUrl"`
	TranscriptURL    *string    `json:"transcriptUrl"`
	CreatedBy        string     `json:"createdBy"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// MeetingAttendance represents attendance tracking
type MeetingAttendance struct {
	ID              string         `json:"id"`
	MeetingID       string         `json:"meetingId"`
	UserID          string         `json:"userId"`
	AttendanceType  string         `json:"attendanceType"`
	JoinedAt        *time.Time     `json:"joinedAt"`
	LeftAt          *time.Time     `json:"leftAt"`
	DurationMinutes int            `json:"durationMinutes"`
	IsPresent       bool           `json:"isPresent"`
	Notes           sql.NullString `json:"-"` // Exclude from JSON, use custom marshaling
	NotesString     string         `json:"notes"` // For JSON serialization
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// GetNotes returns the notes as a string (empty if NULL)
func (ma *MeetingAttendance) GetNotes() string {
	if ma.Notes.Valid {
		return ma.Notes.String
	}
	return ""
}

// SetNotes sets the notes field properly
func (ma *MeetingAttendance) SetNotes(notes string) {
	if notes == "" {
		ma.Notes = sql.NullString{Valid: false}
	} else {
		ma.Notes = sql.NullString{String: notes, Valid: true}
	}
	ma.NotesString = notes
}

// CreateVirtualMeeting creates a new virtual meeting with LiveKit room
func (s *MeetingService) CreateVirtualMeeting(meeting *Meeting) error {
	// Generate unique room name using the room generator
	roomName := s.roomGenerator.GenerateRoomName(meeting.ChamaID, meeting.ID)
	meeting.LiveKitRoomName = roomName

	log.Printf("Generated room name for meeting %s: %s", meeting.ID, roomName)

	// Create LiveKit room if it's a virtual or hybrid meeting
	if meeting.MeetingType == "virtual" || meeting.MeetingType == "hybrid" {
		room, err := s.livekitService.CreateRoom(roomName, 50) // Max 50 participants
		if err != nil {
			return fmt.Errorf("failed to create LiveKit room: %w", err)
		}
		meeting.LiveKitRoomID = room.Sid
	}

	// Save meeting to database
	query := `
		INSERT INTO meetings (
			id, chama_id, title, description, scheduled_at, duration, location,
			meeting_url, meeting_type, livekit_room_name, livekit_room_id,
			status, recording_enabled, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := s.db.Exec(query,
		meeting.ID, meeting.ChamaID, meeting.Title, meeting.Description,
		meeting.ScheduledAt, meeting.Duration, meeting.Location,
		meeting.MeetingURL, meeting.MeetingType, meeting.LiveKitRoomName,
		meeting.LiveKitRoomID, meeting.Status, meeting.RecordingEnabled,
		meeting.CreatedBy,
	)
	if err != nil {
		// Clean up LiveKit room if database insert fails
		if meeting.LiveKitRoomName != "" {
			s.livekitService.DeleteRoom(meeting.LiveKitRoomName)
		}
		return fmt.Errorf("failed to save meeting: %w", err)
	}

	log.Printf("Created meeting: %s with LiveKit room: %s", meeting.ID, meeting.LiveKitRoomName)
	return nil
}

// CreateMeetingWithCalendar creates a new meeting with LiveKit room and Google Calendar integration
func (s *MeetingService) CreateMeetingWithCalendar(meeting *Meeting, attendeeEmails []string, calendarID string) error {
	// First create the virtual meeting
	err := s.CreateVirtualMeeting(meeting)
	if err != nil {
		return err
	}

	// Create Google Calendar event if calendar service is available
	if s.calendarService != nil && calendarID != "" && len(attendeeEmails) > 0 {
		calendarEvent, err := s.calendarService.CreateMeetingEvent(calendarID, meeting, attendeeEmails)
		if err != nil {
			// Log error but don't fail the meeting creation
			log.Printf("Failed to create calendar event: %v", err)
		} else {
			log.Printf("Created calendar event: %s for meeting: %s", calendarEvent.Id, meeting.ID)
		}
	}

	return nil
}

// StartMeeting starts a meeting and updates its status
func (s *MeetingService) StartMeeting(meetingID string) error {
	now := time.Now()

	// Get current status first
	meeting, err := s.GetMeeting(meetingID)
	if err != nil {
		return fmt.Errorf("failed to get meeting: %w", err)
	}

	status := strings.ToLower(meeting.Status)
	if status == "ended" || status == "completed" {
		// Treat as idempotent success: starting an already ended meeting is a no-op
		log.Printf("Meeting %s already ended; start request treated as no-op", meetingID)
		return nil
	}
	if status == "active" || status == "in_progress" {
		log.Printf("Meeting %s already active", meetingID)
		return nil
	}

	query := `
		UPDATE meetings 
		SET status = 'active', started_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := s.db.Exec(query, now, meetingID)
	if err != nil {
		return fmt.Errorf("failed to start meeting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("meeting not found")
	}

	log.Printf("Started meeting: %s", meetingID)
	return nil
}

// EndMeeting ends a meeting and updates its status
func (s *MeetingService) EndMeeting(meetingID string) error {
	now := time.Now()

	// Get meeting details first
	meeting, err := s.GetMeeting(meetingID)
	if err != nil {
		return fmt.Errorf("failed to get meeting: %w", err)
	}

	// If already ended, treat as success
	if strings.ToLower(meeting.Status) == "ended" || strings.ToLower(meeting.Status) == "completed" {
		return nil
	}

	// Update meeting status regardless of current (as long as it exists)
	query := `
		UPDATE meetings 
		SET status = 'ended', ended_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := s.db.Exec(query, now, meetingID)
	if err != nil {
		return fmt.Errorf("failed to end meeting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("meeting not found")
	}

	// Delete LiveKit room if it exists
	if meeting.LiveKitRoomName != "" {
		err = s.livekitService.DeleteRoom(meeting.LiveKitRoomName)
		if err != nil {
			log.Printf("Warning: failed to delete LiveKit room %s: %v", meeting.LiveKitRoomName, err)
		}
	}

	log.Printf("Ended meeting: %s", meetingID)
	return nil
}

// GetMeeting retrieves a meeting by ID
func (s *MeetingService) GetMeeting(meetingID string) (*Meeting, error) {
	log.Printf("Attempting to get meeting with ID: %s", meetingID)

	query := `
		SELECT id, chama_id, title, description, scheduled_at, duration, location,
			   meeting_url, meeting_type, livekit_room_name, livekit_room_id,
			   status, started_at, ended_at, recording_enabled, recording_url,
			   transcript_url, created_by, created_at, updated_at
		FROM meetings
		WHERE id = ?
	`

	row := s.db.QueryRow(query, meetingID)

	meeting := &Meeting{}
	var startedAt, endedAt sql.NullTime
	var recordingURL, transcriptURL, livekitRoomName, livekitRoomID sql.NullString

	err := row.Scan(
		&meeting.ID, &meeting.ChamaID, &meeting.Title, &meeting.Description,
		&meeting.ScheduledAt, &meeting.Duration, &meeting.Location,
		&meeting.MeetingURL, &meeting.MeetingType, &livekitRoomName,
		&livekitRoomID, &meeting.Status, &startedAt, &endedAt,
		&meeting.RecordingEnabled, &recordingURL, &transcriptURL,
		&meeting.CreatedBy, &meeting.CreatedAt, &meeting.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error scanning meeting %s: %v", meetingID, err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("meeting not found")
		}
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	log.Printf("Successfully retrieved meeting %s: type=%s, title=%s", meetingID, meeting.MeetingType, meeting.Title)

	if startedAt.Valid {
		meeting.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		meeting.EndedAt = &endedAt.Time
	}

	// Handle nullable string fields
	if recordingURL.Valid {
		meeting.RecordingURL = &recordingURL.String
	}
	if transcriptURL.Valid {
		meeting.TranscriptURL = &transcriptURL.String
	}
	if livekitRoomName.Valid {
		meeting.LiveKitRoomName = livekitRoomName.String
	}
	if livekitRoomID.Valid {
		meeting.LiveKitRoomID = livekitRoomID.String
	}

	return meeting, nil
}

// GenerateJoinToken generates a LiveKit access token for a user to join a meeting
func (s *MeetingService) GenerateJoinToken(meetingID, userID, userRole string) (string, error) {
	meeting, err := s.GetMeeting(meetingID)
	if err != nil {
		return "", err
	}

	if meeting.LiveKitRoomName == "" {
		return "", fmt.Errorf("meeting does not have a LiveKit room")
	}

	// Generate participant name (you might want to get actual user name from database)
	participantName := fmt.Sprintf("user_%s", userID)

	token, err := s.livekitService.GenerateAccessToken(
		meeting.LiveKitRoomName,
		participantName,
		userRole,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate join token: %w", err)
	}

	return token, nil
}

// GeneratePreviewToken generates a special preview token for chairpersons and secretaries
func (s *MeetingService) GeneratePreviewToken(meetingID, userID, userRole string) (string, error) {
	meeting, err := s.GetMeeting(meetingID)
	if err != nil {
		return "", err
	}

	// Only allow chairpersons and secretaries to preview
	if userRole != "chairperson" && userRole != "secretary" {
		return "", fmt.Errorf("only chairpersons and secretaries can preview meetings")
	}

	// For physical meetings, we don't need LiveKit tokens
	if meeting.MeetingType == "physical" {
		return "", fmt.Errorf("physical meetings do not require LiveKit tokens")
	}

	if meeting.LiveKitRoomName == "" {
		return "", fmt.Errorf("meeting does not have a LiveKit room")
	}

	// Generate preview token with admin privileges
	token, err := s.livekitService.GenerateAccessToken(meeting.LiveKitRoomName, userID, userRole)
	if err != nil {
		return "", fmt.Errorf("failed to generate preview token: %w", err)
	}

	log.Printf("Generated preview token for user %s (role: %s) in meeting %s", userID, userRole, meetingID)
	return token, nil
}

// GetMeetingPreviewInfo returns information needed for meeting preview
func (s *MeetingService) GetMeetingPreviewInfo(meetingID, userID, userRole string) (map[string]interface{}, error) {
	// Only allow chairpersons, secretaries, and treasurers to preview
	if userRole != "chairperson" && userRole != "secretary" && userRole != "treasurer" {
		return nil, fmt.Errorf("only chairpersons, secretaries, and treasurers can preview meetings")
	}

	meeting, err := s.GetMeeting(meetingID)
	if err != nil {
		return nil, err
	}

	previewInfo := map[string]interface{}{
		"meeting":   meeting,
		"isPreview": true,
		"userRole":  userRole,
		"canRecord": userRole == "chairperson",
		"canMute":   true,
		"canKick":   userRole == "chairperson",
	}

	// Handle different meeting types
	switch meeting.MeetingType {
	case "virtual", "hybrid":
		// For virtual/hybrid meetings, check if LiveKit room exists
		if meeting.LiveKitRoomName == "" {
			// Create LiveKit room on-demand for preview using room generator
			roomName := s.roomGenerator.GenerateRoomName(meeting.ChamaID, meeting.ID)
			log.Printf("Generating room name for preview: %s", roomName)
			room, err := s.livekitService.CreateRoom(roomName, 50)
			if err != nil {
				log.Printf("Warning: Failed to create LiveKit room for preview: %v", err)
				// Fallback to basic virtual meeting preview without LiveKit
				previewInfo["meetingType"] = "virtual"
				previewInfo["previewMessage"] = "This is a virtual meeting. LiveKit room will be created when the meeting starts."
				previewInfo["meetingUrl"] = meeting.MeetingURL
				previewInfo["fallbackMode"] = true
			} else {
				// Update meeting with LiveKit room info
				meeting.LiveKitRoomName = roomName
				meeting.LiveKitRoomID = room.Sid

				// Update database with LiveKit room info
				updateQuery := `
					UPDATE meetings
					SET livekit_room_name = ?, livekit_room_id = ?, updated_at = CURRENT_TIMESTAMP
					WHERE id = ?
				`
				_, err = s.db.Exec(updateQuery, roomName, room.Sid, meeting.ID)
				if err != nil {
					log.Printf("Warning: Failed to update meeting with LiveKit room info: %v", err)
				}

				// Generate preview token using the room name directly
				token, err := s.livekitService.GenerateAccessToken(roomName, userID, userRole)
				if err != nil {
					log.Printf("Warning: Failed to generate preview token: %v", err)
					return nil, fmt.Errorf("failed to generate preview token: %w", err)
				}

				// Get LiveKit WebSocket URL
				wsURL := s.livekitService.GetWSURL()

				previewInfo["accessToken"] = token
				previewInfo["wsURL"] = wsURL
				previewInfo["roomName"] = meeting.LiveKitRoomName
				previewInfo["meetingType"] = "virtual"
			}
		} else {
			// LiveKit room already exists
			// Generate preview token using the existing room name
			token, err := s.livekitService.GenerateAccessToken(meeting.LiveKitRoomName, userID, userRole)
			if err != nil {
				log.Printf("Warning: Failed to generate preview token for existing room: %v", err)
				return nil, fmt.Errorf("failed to generate preview token: %w", err)
			}

			// Get LiveKit WebSocket URL
			wsURL := s.livekitService.GetWSURL()

			previewInfo["accessToken"] = token
			previewInfo["wsURL"] = wsURL
			previewInfo["roomName"] = meeting.LiveKitRoomName
			previewInfo["meetingType"] = "virtual"
		}

	case "physical":
		// For physical meetings, provide location and setup info
		previewInfo["meetingType"] = "physical"
		previewInfo["location"] = meeting.Location
		previewInfo["previewMessage"] = "This is a physical meeting. Use this preview to review meeting details and prepare for the session."
		previewInfo["setupInstructions"] = []string{
			"Ensure the meeting venue is properly set up",
			"Check that all necessary materials are available",
			"Verify attendance tracking is ready",
			"Prepare meeting agenda and documents",
		}

	default:
		return nil, fmt.Errorf("unsupported meeting type: %s", meeting.MeetingType)
	}

	return previewInfo, nil
}

// MarkAttendance marks a user's attendance for a meeting
func (s *MeetingService) MarkAttendance(meetingID, userID, attendanceType string, isPresent bool) error {
	attendanceID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT OR REPLACE INTO meeting_attendance (
			id, meeting_id, user_id, attendance_type, joined_at, is_present,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := s.db.Exec(query, attendanceID, meetingID, userID, attendanceType, now, isPresent)
	if err != nil {
		return fmt.Errorf("failed to mark attendance: %w", err)
	}

	log.Printf("Marked attendance for user %s in meeting %s", userID, meetingID)
	return nil
}

// GetMeetingAttendance retrieves attendance records for a meeting
func (s *MeetingService) GetMeetingAttendance(meetingID string) ([]*MeetingAttendance, error) {
	query := `
		SELECT id, meeting_id, user_id, attendance_type, joined_at, left_at,
			   duration_minutes, is_present, notes, created_at, updated_at
		FROM meeting_attendance
		WHERE meeting_id = ?
		ORDER BY joined_at DESC
	`

	rows, err := s.db.Query(query, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meeting attendance: %w", err)
	}
	defer rows.Close()

	var attendances []*MeetingAttendance

	for rows.Next() {
		attendance := &MeetingAttendance{}
		var joinedAt, leftAt sql.NullTime

		err := rows.Scan(
			&attendance.ID, &attendance.MeetingID, &attendance.UserID,
			&attendance.AttendanceType, &joinedAt, &leftAt,
			&attendance.DurationMinutes, &attendance.IsPresent, &attendance.Notes,
			&attendance.CreatedAt, &attendance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attendance: %w", err)
		}

		if joinedAt.Valid {
			attendance.JoinedAt = &joinedAt.Time
		}
		if leftAt.Valid {
			attendance.LeftAt = &leftAt.Time
		}

		// Set the NotesString field for JSON serialization
		attendance.NotesString = attendance.GetNotes()

		attendances = append(attendances, attendance)
	}

	return attendances, nil
}

// EnsureVirtualMeetingsHaveRoomNames ensures all virtual meetings have proper room names
func (s *MeetingService) EnsureVirtualMeetingsHaveRoomNames() error {
	log.Printf("Ensuring all virtual meetings have room names...")

	// Find all virtual/hybrid meetings without room names
	query := `
		SELECT id, chama_id, title, meeting_type
		FROM meetings
		WHERE (meeting_type = 'virtual' OR meeting_type = 'hybrid')
		AND (livekit_room_name IS NULL OR livekit_room_name = '')
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query meetings without room names: %w", err)
	}
	defer rows.Close()

	updateCount := 0
	for rows.Next() {
		var meetingID, chamaID, title, meetingType string
		err := rows.Scan(&meetingID, &chamaID, &title, &meetingType)
		if err != nil {
			log.Printf("Error scanning meeting: %v", err)
			continue
		}

		// Generate room name
		roomName := s.roomGenerator.GenerateRoomName(chamaID, meetingID)

		// Update the meeting with the room name
		updateQuery := `
			UPDATE meetings
			SET livekit_room_name = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`

		_, err = s.db.Exec(updateQuery, roomName, meetingID)
		if err != nil {
			log.Printf("Failed to update room name for meeting %s: %v", meetingID, err)
			continue
		}

		log.Printf("Updated meeting %s (%s) with room name: %s", meetingID, title, roomName)
		updateCount++
	}

	log.Printf("Updated %d meetings with room names", updateCount)
	return nil
}

// JitsiRoomData represents the data needed to join a Jitsi Meet room
type JitsiRoomData struct {
	RoomName     string `json:"roomName"`
	JoinURL      string `json:"joinUrl"`
	RoomPassword string `json:"roomPassword"`
	IsModerator  bool   `json:"isModerator"`
	DisplayName  string `json:"displayName"`
	UserEmail    string `json:"userEmail"`
	// JaaS fields
	Domain       string `json:"domain,omitempty"`
	AppID        string `json:"appId,omitempty"`
	JWT          string `json:"jwt,omitempty"`
}

// GenerateJitsiRoomData generates Jitsi Meet room data for a user to join a meeting
func (s *MeetingService) GenerateJitsiRoomData(meetingID, userID, userRole string) (*JitsiRoomData, error) {
	meeting, err := s.GetMeeting(meetingID)
	if err != nil {
		return nil, err
	}

	// SECURITY CHECK: Verify user is a member of the chama that owns this meeting
	isMember, err := s.verifyUserChamaMembership(userID, meeting.ChamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify chama membership: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("access denied: user is not a member of this chama")
	}

	// Get actual user information from database
	var displayName, userEmail string

	userQuery := `
		SELECT first_name, last_name, username, email
		FROM users
		WHERE id = ?
	`
	var firstName, lastName, username, email sql.NullString
	err = s.db.QueryRow(userQuery, userID).Scan(&firstName, &lastName, &username, &email)

	if err != nil {
		log.Printf("Warning: Could not fetch user details for %s: %v", userID, err)
		// Fallback to basic user data
		displayName = fmt.Sprintf("User_%s", userID)
		userEmail = fmt.Sprintf("user_%s@vaultke.app", userID)
	} else {
		// Build proper display name from user data
		displayName = buildUserDisplayName(firstName.String, lastName.String, username.String)
		userEmail = email.String
		if userEmail == "" {
			userEmail = fmt.Sprintf("user_%s@vaultke.app", userID)
		}
	}

	// Generate secure room name if not already set
	roomName := meeting.LiveKitRoomName // Reuse the room name field
	if roomName == "" {
		// Generate highly secure room name with chama isolation
		timestamp := time.Now().Unix()
		randomSuffix := generateSecureRandomString(8)
		roomName = fmt.Sprintf("vaultke_chama_%s_meeting_%s_%d_%s",
			meeting.ChamaID, meeting.ID, timestamp, randomSuffix)

		// Update the meeting with the room name
		updateQuery := `UPDATE meetings SET livekit_room_name = ? WHERE id = ?`
		_, err = s.db.Exec(updateQuery, roomName, meetingID)
		if err != nil {
			log.Printf("Warning: Failed to update meeting with room name: %v", err)
		}
	}

	// Generate room password using a deterministic method
	roomPassword := s.generateJitsiRoomPassword(roomName, userRole)

	// Determine if user should be a moderator (chairperson, secretary, treasurer)
	isModerator := userRole == "chairperson" || userRole == "secretary" || userRole == "treasurer"

	// Try JaaS (8x8.vc) JWT flow if configured
	jaasAppID := os.Getenv("JITSI_APP_ID") // e.g. vpaas-magic-cookie-... (AppID)
	jaasKID := os.Getenv("JITSI_KID")      // e.g. vpaas-magic-cookie-.../<keyId>
	jaasPrivateKey := os.Getenv("JITSI_PRIVATE_KEY_PEM")
	useJaaS := jaasAppID != "" && jaasKID != "" && jaasPrivateKey != ""

	if useJaaS {
		jwtToken, err := s.generateJaaSJWT(jaasAppID, jaasKID, roomName, displayName, userEmail, isModerator, userID)
		if err == nil {
			return &JitsiRoomData{
				RoomName:    fmt.Sprintf("%s/%s", jaasAppID, roomName),
				JoinURL:     "", // frontend will embed via external_api.js
				IsModerator: isModerator,
				DisplayName: displayName,
				UserEmail:   userEmail,
				Domain:      "8x8.vc",
				AppID:       jaasAppID,
				JWT:         jwtToken,
			}, nil
		}
		log.Printf("Warning: failed to generate JaaS JWT, falling back to public Jitsi URL: %v", err)
	}

	// Build public Jitsi Meet URL fallback
	joinURL := s.buildJitsiMeetURL(roomName, displayName, userEmail, roomPassword, isModerator)

	jitsiData := &JitsiRoomData{
		RoomName:     roomName,
		JoinURL:      joinURL,
		RoomPassword: roomPassword,
		IsModerator:  isModerator,
		DisplayName:  displayName,
		UserEmail:    userEmail,
	}

	log.Printf("Generated Jitsi room data for user %s (role: %s) in meeting %s", userID, userRole, meetingID)
	return jitsiData, nil
}

// generateJaaSJWT builds a signed RS256 token for Jitsi as a Service
func (s *MeetingService) generateJaaSJWT(appID, kid, roomName, displayName, email string, isModerator bool, userID string) (string, error) {
    // Parse RSA private key
    block, _ := pem.Decode([]byte(os.Getenv("JITSI_PRIVATE_KEY_PEM")))
    if block == nil {
        return "", fmt.Errorf("invalid JITSI_PRIVATE_KEY_PEM")
    }
    pk, err := x509.ParsePKCS8PrivateKey(block.Bytes)
    if err != nil {
        // try PKCS1
        if rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
            pk = rsaKey
        } else {
            return "", fmt.Errorf("failed to parse private key: %w", err)
        }
    }

    now := time.Now()
    claims := jwt.MapClaims{
        "aud": "jitsi",
        "iss": "chat",
        "sub": appID,
        "room": "*",
        "nbf": now.Unix(),
        "exp": now.Add(4 * time.Hour).Unix(),
        "context": map[string]interface{}{
            "user": map[string]interface{}{
                "id":        userID,
                "name":      displayName,
                "email":     email,
                "moderator": isModerator,
            },
            "features": map[string]interface{}{
                "livestreaming":   false,
                "recording":       false,
                "transcription":   false,
                "outbound-call":   false,
                "inbound-call":    false,
                "file-upload":     false,
                "list-visitors":   false,
            },
            "room": map[string]interface{}{
                "regex": false,
            },
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = kid
    signed, err := token.SignedString(pk)
    if err != nil {
        return "", err
    }
    return signed, nil
}

// generateJitsiRoomPassword generates a secure room password
func (s *MeetingService) generateJitsiRoomPassword(roomName, userRole string) string {
	// Create a deterministic but secure password
	// In production, you might want to use a more sophisticated method
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(roomName+userRole+"vaultke-secret")))
	return hash[:12] // Use first 12 characters
}

// buildJitsiMeetURL builds the complete Jitsi Meet URL with parameters
func (s *MeetingService) buildJitsiMeetURL(roomName, displayName, userEmail, roomPassword string, isModerator bool) string {
	// Use Jitsi as a Service (JaaS) for full control
	jitsiAppID := os.Getenv("JITSI_APP_ID")
	jitsiDomain := os.Getenv("JITSI_DOMAIN")

	if jitsiAppID != "" && jitsiDomain != "" {
		// Use JaaS with your App ID for complete control
		return s.buildJaaSURL(roomName, displayName, userEmail, isModerator)
	}

	// Fallback to public Jitsi Meet (not recommended for production)
	baseURL := "https://meet.jit.si"
	encodedRoomName := url.QueryEscape(roomName)

	// Build URL with query parameters
	params := url.Values{}

	// USER IDENTITY - Pre-populate with system data
	params.Add("userInfo.displayName", displayName)
	params.Add("userInfo.email", userEmail)

	// MEETING BEHAVIOR
	params.Add("config.startWithAudioMuted", "false")
	params.Add("config.startWithVideoMuted", "false")
	params.Add("config.requireDisplayName", "true")
	params.Add("config.enableWelcomePage", "false")
	params.Add("config.enableClosePage", "false")
	params.Add("config.prejoinPageEnabled", "false") // Skip pre-join screen

	// CRITICAL: Remove authentication prompts
	params.Add("config.disableDeepLinking", "true")     // Remove mobile app prompts
	params.Add("config.hideDisplayName", "false")       // Keep display names visible
	params.Add("config.hideEmailInSettings", "true")    // Hide email from settings
	params.Add("config.hideInviteMoreHeader", "true")   // Remove "invite more" header
	params.Add("config.disableInviteFunctions", "true") // Disable invite functions
	params.Add("config.doNotStoreRoom", "true")         // Don't store room in browser

	// FORCE ANONYMOUS MODE - This removes login prompts
	params.Add("config.enableUserRolesBasedOnToken", "false") // Disable token-based auth
	params.Add("config.enableFeaturesBasedOnToken", "false")  // Disable feature tokens
	params.Add("config.disableProfile", "true")               // Disable profile editing
	params.Add("config.readOnlyName", "true")                 // Make name read-only
	params.Add("config.enableClosePage", "false")             // No close page
	params.Add("config.enableWelcomePage", "false")           // No welcome page

	// TOOLBAR CUSTOMIZATION - Remove unwanted buttons
	params.Add("config.toolbarButtons", "["+
		"'microphone','camera','desktop','fullscreen',"+
		"'hangup','chat','recording','settings','raisehand',"+
		"'videoquality','filmstrip','tileview','mute-everyone'"+
		"]") // Removed: invite, feedback, stats, shortcuts, help, profile, etc.

	// HIDE/DISABLE UNWANTED FEATURES
	params.Add("config.disableRemoteMute", "false")        // Allow moderators to mute
	params.Add("config.enableEmailInStats", "false")       // Hide email in stats
	params.Add("config.disableThirdPartyRequests", "true") // Block external requests
	params.Add("config.disableLocalVideoFlip", "false")    // Allow video flip
	params.Add("config.disableSimulcast", "false")         // Keep simulcast for quality

	// BRANDING & CUSTOMIZATION
	params.Add("config.defaultLanguage", "en")          // Set default language
	params.Add("config.disableGoogleAnalytics", "true") // No tracking
	params.Add("config.disableRtx", "false")            // Keep RTX for quality
	params.Add("config.channelLastN", "20")             // Limit video streams

	// CRITICAL INTERFACE CONTROLS
	params.Add("config.disableJoinLeaveSounds", "false") // Keep join/leave sounds
	params.Add("config.hideConferenceSubject", "true")   // Hide room name display
	params.Add("config.hideConferenceTimer", "false")    // Keep timer
	params.Add("config.hideParticipantsStats", "true")   // Hide participant stats
	params.Add("config.disableShortcuts", "true")        // Disable keyboard shortcuts overlay

	// FORCE ANONYMOUS ACCESS - Critical for removing login prompts
	params.Add("config.enableAuthenticationUI", "false")     // Hide authentication UI
	params.Add("config.disableAuthenticationPrompt", "true") // Disable auth prompts
	params.Add("config.enableGuestDomain", "true")           // Enable guest access
	params.Add("config.guestDomain", "guest.meet.jit.si")    // Use guest domain

	// SECURITY & PRIVACY
	params.Add("config.enableNoAudioDetection", "true")
	params.Add("config.enableNoisyMicDetection", "true")
	params.Add("config.enableLayerSuspension", "true") // Better performance

	if roomPassword != "" {
		params.Add("config.roomPassword", roomPassword)
	}

	if isModerator {
		// MODERATORS: Direct access, no waiting room
		params.Add("config.isModerator", "true")
		params.Add("config.enableLobby", "false")
		params.Add("config.enableLobbyChat", "false")
	} else {
		// NON-MODERATORS: Must wait for approval
		params.Add("config.enableLobby", "true")
		params.Add("config.enableLobbyChat", "true")
	}

	return fmt.Sprintf("%s/%s?%s", baseURL, encodedRoomName, params.Encode())
}

// buildJitsiURLWithJWT builds a Jitsi Meet URL with JWT token for full control
func (s *MeetingService) buildJitsiURLWithJWT(baseURL, roomName, displayName, userEmail string, isModerator bool) string {
	// For now, use your existing JWT token approach
	// You can implement JWT generation here later if needed

	// Build URL with minimal parameters for JWT-based auth
	params := url.Values{}
	params.Add("jwt", "YOUR_JWT_TOKEN_HERE") // You'll replace this with actual JWT

	// Essential user info
	params.Add("userInfo.displayName", displayName)
	params.Add("userInfo.email", userEmail)

	// Force clean interface
	params.Add("config.prejoinPageEnabled", "false")
	params.Add("config.enableWelcomePage", "false")
	params.Add("config.enableClosePage", "false")

	return fmt.Sprintf("%s/%s?%s", baseURL, roomName, params.Encode())
}

// buildJaaSURL builds a Jitsi as a Service URL with proper configuration
// SECURITY: Each chama gets completely isolated meeting rooms with cryptographic separation
func (s *MeetingService) buildJaaSURL(roomName, displayName, userEmail string, isModerator bool) string {
	jitsiAppID := os.Getenv("JITSI_APP_ID")
	jitsiDomain := os.Getenv("JITSI_DOMAIN")

	// SECURITY ENHANCEMENT: Create cryptographically secure room isolation
	// Each chama gets a unique namespace that cannot be guessed or accessed by other chamas

	// Extract chama ID from room name (format: chama_<chamaId>_meeting_<meetingId>)
	chamaID := extractChamaIDFromRoomName(roomName)

	// Generate secure room identifier with multiple layers of isolation:
	// 1. App ID prefix (your JaaS account isolation)
	// 2. Chama-specific cryptographic hash (prevents cross-chama access)
	// 3. Meeting-specific identifier (prevents meeting collision)
	secureRoomName := generateSecureRoomName(jitsiAppID, chamaID, roomName)

	// Build the JaaS URL - this will be used by the frontend External API
	baseURL := fmt.Sprintf("https://%s", jitsiDomain)

	// For JaaS, we return a special URL that the frontend will use with External API
	// The frontend will handle the JWT and configuration
	return fmt.Sprintf("%s/jaas?room=%s&name=%s&email=%s&moderator=%t",
		baseURL,
		url.QueryEscape(secureRoomName),
		url.QueryEscape(displayName),
		url.QueryEscape(userEmail),
		isModerator,
	)
}

// extractChamaIDFromRoomName extracts the chama ID from the room name for security isolation
func extractChamaIDFromRoomName(roomName string) string {
	// Room name format: chama_<chamaId>_meeting_<meetingId>
	parts := strings.Split(roomName, "_")
	if len(parts) >= 2 && parts[0] == "chama" {
		return parts[1] // Return the chama ID
	}
	// Fallback: use a hash of the room name if format is unexpected
	hash := sha256.Sum256([]byte(roomName))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes as fallback ID
}

// generateSecureRoomName creates a cryptographically secure room name that ensures complete isolation
func generateSecureRoomName(appID, chamaID, originalRoomName string) string {
	// SECURITY LAYERS:
	// 1. App ID prefix - isolates your organization from other JaaS users
	// 2. Chama-specific salt - ensures chamas cannot access each other's meetings
	// 3. Room-specific hash - prevents meeting name collisions

	// Create chama-specific salt using cryptographic hash
	chamaSalt := sha256.Sum256([]byte(fmt.Sprintf("VAULTKE_CHAMA_SALT_%s_%s", chamaID, appID)))
	chamaSaltHex := hex.EncodeToString(chamaSalt[:16]) // Use first 16 bytes

	// Create room-specific hash
	roomHash := sha256.Sum256([]byte(fmt.Sprintf("%s_%s_%s", originalRoomName, chamaID, chamaSaltHex)))
	roomHashHex := hex.EncodeToString(roomHash[:12]) // Use first 12 bytes

	// Combine all security layers
	secureRoomName := fmt.Sprintf("%s/CHAMA_%s_ROOM_%s", appID, chamaSaltHex, roomHashHex)

	log.Printf("🔒 Generated secure room: %s -> %s", originalRoomName, secureRoomName[:50]+"...")
	return secureRoomName
}

// buildUserDisplayName creates a proper display name from user data
func buildUserDisplayName(firstName, lastName, username string) string {
	// Priority: FirstName LastName > FirstName > Username > fallback
	if firstName != "" && lastName != "" {
		return fmt.Sprintf("%s %s", firstName, lastName)
	}
	if firstName != "" {
		return firstName
	}
	if username != "" {
		return username
	}
	return "VaultKe User"
}

// verifyUserChamaMembership checks if a user is a member of a specific chama
func (s *MeetingService) verifyUserChamaMembership(userID, chamaID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM chama_members
		WHERE user_id = ? AND chama_id = ? AND is_active = TRUE
	`

	var count int
	err := s.db.QueryRow(query, userID, chamaID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check chama membership: %w", err)
	}

	return count > 0, nil
}

// generateSecureRandomString generates a cryptographically secure random string
func generateSecureRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based random if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())[0:length]
	}
	return hex.EncodeToString(bytes)[:length]
}
