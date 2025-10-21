package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

// RoomNameGenerator provides methods for generating consistent room names
type RoomNameGenerator struct{}

// NewRoomNameGenerator creates a new room name generator
func NewRoomNameGenerator() *RoomNameGenerator {
	return &RoomNameGenerator{}
}

// GenerateRoomName generates a consistent room name for a meeting
// Format: chama_{chamaId}_{meetingId}_{timestamp}
func (g *RoomNameGenerator) GenerateRoomName(chamaID, meetingID string) string {
	// Clean the IDs to ensure they're safe for room names
	cleanChamaID := g.cleanID(chamaID)
	cleanMeetingID := g.cleanID(meetingID)

	// Add timestamp for uniqueness
	timestamp := time.Now().Unix()

	roomName := fmt.Sprintf("chama_%s_meeting_%s_%d", cleanChamaID, cleanMeetingID, timestamp)

	// Ensure the room name is valid (alphanumeric, hyphens, underscores only)
	roomName = g.sanitizeRoomName(roomName)

	return roomName
}

// GenerateShortRoomName generates a shorter room name for better UX
// Format: chama_{shortId}_{randomSuffix}
func (g *RoomNameGenerator) GenerateShortRoomName(chamaID, meetingID string) string {
	// Take first 8 characters of chama ID
	shortChamaID := g.cleanID(chamaID)
	if len(shortChamaID) > 8 {
		shortChamaID = shortChamaID[:8]
	}

	// Take first 8 characters of meeting ID
	shortMeetingID := g.cleanID(meetingID)
	if len(shortMeetingID) > 8 {
		shortMeetingID = shortMeetingID[:8]
	}

	// Generate random suffix for uniqueness
	randomSuffix := g.generateRandomString(6)

	roomName := fmt.Sprintf("chama_%s_%s_%s", shortChamaID, shortMeetingID, randomSuffix)

	return g.sanitizeRoomName(roomName)
}

// GenerateUserFriendlyRoomName generates a human-readable room name
// Format: {ChamaName}_{MeetingTitle}_{Date}
func (g *RoomNameGenerator) GenerateUserFriendlyRoomName(chamaName, meetingTitle string, meetingDate time.Time) string {
	// Clean and truncate names
	cleanChamaName := g.cleanAndTruncate(chamaName, 15)
	cleanMeetingTitle := g.cleanAndTruncate(meetingTitle, 20)

	// Format date as YYYYMMDD
	dateStr := meetingDate.Format("20060102")

	roomName := fmt.Sprintf("%s_%s_%s", cleanChamaName, cleanMeetingTitle, dateStr)

	return g.sanitizeRoomName(roomName)
}

// GetRoomNameFromMeeting extracts or generates room name from meeting data
func (g *RoomNameGenerator) GetRoomNameFromMeeting(chamaID, meetingID, chamaName, meetingTitle string, meetingDate time.Time, existingRoomName string) string {
	// If room name already exists and is valid, use it
	if existingRoomName != "" && g.isValidRoomName(existingRoomName) {
		return existingRoomName
	}

	// Try to generate user-friendly name first
	if chamaName != "" && meetingTitle != "" {
		return g.GenerateUserFriendlyRoomName(chamaName, meetingTitle, meetingDate)
	}

	// Fallback to short room name
	return g.GenerateShortRoomName(chamaID, meetingID)
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

// cleanAndTruncate cleans a string and truncates it to maxLength
func (g *RoomNameGenerator) cleanAndTruncate(str string, maxLength int) string {
	// Convert to lowercase and replace spaces with underscores
	cleaned := strings.ToLower(str)
	cleaned = strings.ReplaceAll(cleaned, " ", "_")

	// Remove special characters except underscores and hyphens
	reg := regexp.MustCompile(`[^a-z0-9_\-]`)
	cleaned = reg.ReplaceAllString(cleaned, "")

	// Remove consecutive underscores/hyphens
	reg = regexp.MustCompile(`[_\-]+`)
	cleaned = reg.ReplaceAllString(cleaned, "_")

	// Trim underscores from start and end
	cleaned = strings.Trim(cleaned, "_-")

	// Truncate if too long
	if len(cleaned) > maxLength {
		cleaned = cleaned[:maxLength]
	}

	// Ensure it's not empty
	if cleaned == "" {
		cleaned = "meeting"
	}

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

// isValidRoomName checks if a room name is valid for LiveKit
func (g *RoomNameGenerator) isValidRoomName(roomName string) bool {
	if roomName == "" {
		return false
	}

	// Check length
	if len(roomName) < 1 || len(roomName) > 63 {
		return false
	}

	// Check characters (alphanumeric, underscores, hyphens only)
	reg := regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
	return reg.MatchString(roomName)
}

// generateRandomString generates a random alphanumeric string of given length
func (g *RoomNameGenerator) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)

	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

// GetRoomDisplayName generates a user-friendly display name for the room
func (g *RoomNameGenerator) GetRoomDisplayName(chamaName, meetingTitle string) string {
	if chamaName != "" && meetingTitle != "" {
		return fmt.Sprintf("%s - %s", chamaName, meetingTitle)
	}
	if meetingTitle != "" {
		return meetingTitle
	}
	if chamaName != "" {
		return fmt.Sprintf("%s Meeting", chamaName)
	}
	return "Meeting Room"
}
