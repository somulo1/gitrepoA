package services

import (
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
)

type LiveKitService struct {
	apiKey    string
	apiSecret string
	wsURL     string
}

func NewLiveKitService(wsURL, apiKey, apiSecret string) *LiveKitService {
	// Debug logging to verify credentials
	fmt.Printf("🔍 LiveKit Service initialized with:\n")
	fmt.Printf("  - WS URL: %s\n", wsURL)
	fmt.Printf("  - API Key: %s\n", apiKey)
	fmt.Printf("  - API Secret length: %d\n", len(apiSecret))

	return &LiveKitService{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		wsURL:     wsURL,
	}
}

// GenerateAccessToken creates a LiveKit access token for a user
func (s *LiveKitService) GenerateAccessToken(roomName, participantName, userRole string) (string, error) {
	fmt.Printf("🔍 Generating token with credentials:\n")
	fmt.Printf("  - API Key: %s\n", s.apiKey)
	fmt.Printf("  - API Secret length: %d\n", len(s.apiSecret))
	fmt.Printf("  - Room: %s\n", roomName)
	fmt.Printf("  - Participant: %s\n", participantName)
	fmt.Printf("  - Role: %s\n", userRole)

	if s.apiKey == "" || s.apiSecret == "" {
		return "", fmt.Errorf("missing API key or secret key")
	}

	// Create access token
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)

	// Set token validity (24 hours)
	at.SetValidFor(24 * time.Hour)

	// Set participant identity and name
	at.SetIdentity(participantName)
	at.SetName(participantName)

	// Set permissions based on user role
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}

	// Helper function to create bool pointers
	boolPtr := func(b bool) *bool { return &b }

	// Role-based permissions
	switch userRole {
	case "chairperson":
		grant.RoomAdmin = true
		grant.CanPublish = boolPtr(true)
		grant.CanSubscribe = boolPtr(true)
		grant.CanPublishData = boolPtr(true)
	case "secretary":
		grant.CanPublish = boolPtr(true)
		grant.CanSubscribe = boolPtr(true)
		grant.CanPublishData = boolPtr(true)
	case "treasurer":
		grant.CanPublish = boolPtr(true)
		grant.CanSubscribe = boolPtr(true)
		grant.CanPublishData = boolPtr(true)
	case "member":
		grant.CanPublish = boolPtr(true)
		grant.CanSubscribe = boolPtr(true)
		grant.CanPublishData = boolPtr(false)
	default:
		grant.CanPublish = boolPtr(false)
		grant.CanSubscribe = boolPtr(true)
		grant.CanPublishData = boolPtr(false)
	}

	at.SetVideoGrant(grant)

	// Generate the token
	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// CreateRoom creates a new LiveKit room
func (s *LiveKitService) CreateRoom(roomName string, maxParticipants uint32) (*livekit.Room, error) {
	if s.apiKey == "" || s.apiSecret == "" {
		return nil, fmt.Errorf("missing API key or secret key")
	}

	// For now, we'll return a mock room since we don't have the LiveKit server SDK
	// In a real implementation, you would use the LiveKit server SDK to create the room
	room := &livekit.Room{
		Sid:             fmt.Sprintf("room_%s_%d", roomName, time.Now().Unix()),
		Name:            roomName,
		EmptyTimeout:    300, // 5 minutes
		MaxParticipants: maxParticipants,
		CreationTime:    time.Now().Unix(),
		TurnPassword:    "",
		EnabledCodecs:   []*livekit.Codec{},
		Metadata:        "",
		NumParticipants: 0,
		NumPublishers:   0,
		ActiveRecording: false,
	}

	return room, nil
}

// GetRoomInfo gets information about a LiveKit room
func (s *LiveKitService) GetRoomInfo(roomName string) (*livekit.Room, error) {
	if s.apiKey == "" || s.apiSecret == "" {
		return nil, fmt.Errorf("missing API key or secret key")
	}

	// Mock implementation - in real scenario, you'd query the LiveKit server
	room := &livekit.Room{
		Sid:             fmt.Sprintf("room_%s", roomName),
		Name:            roomName,
		EmptyTimeout:    300,
		MaxParticipants: 50,
		CreationTime:    time.Now().Unix(),
		NumParticipants: 0,
		NumPublishers:   0,
		ActiveRecording: false,
	}

	return room, nil
}

// DeleteRoom deletes a LiveKit room
func (s *LiveKitService) DeleteRoom(roomName string) error {
	if s.apiKey == "" || s.apiSecret == "" {
		return fmt.Errorf("missing API key or secret key")
	}

	// Mock implementation - in real scenario, you'd delete the room from LiveKit server
	fmt.Printf("Room %s would be deleted\n", roomName)
	return nil
}

// GetWSURL returns the WebSocket URL for LiveKit
func (s *LiveKitService) GetWSURL() string {
	return s.wsURL
}

// ValidateConfig checks if the LiveKit configuration is valid
func (s *LiveKitService) ValidateConfig() error {
	if s.apiKey == "" {
		return fmt.Errorf("LIVEKIT_API_KEY environment variable is required")
	}
	if s.apiSecret == "" {
		return fmt.Errorf("LIVEKIT_API_SECRET environment variable is required")
	}
	if s.wsURL == "" {
		return fmt.Errorf("LIVEKIT_WS_URL environment variable is required")
	}
	return nil
}
