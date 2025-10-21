package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

// Helper functions for safe metadata extraction
func getStringFromMeta(meta map[string]interface{}, key, defaultValue string) string {
	if val, exists := meta[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getFloatFromMeta(meta map[string]interface{}, key string, defaultValue float64) float64 {
	if val, exists := meta[key]; exists {
		if flt, ok := val.(float64); ok {
			return flt
		}
	}
	return defaultValue
}

// ChatService handles chat and messaging functionality
type ChatService struct {
	db          *sql.DB
	e2eeService *MilitaryGradeE2EEService
}

// NewChatService creates a new chat service
func NewChatService(db *sql.DB) *ChatService {
	return &ChatService{
		db:          db,
		e2eeService: NewMilitaryGradeE2EEService(db),
	}
}

// ChatRoomType represents the type of chat room
type ChatRoomType string

const (
	ChatRoomTypePrivate ChatRoomType = "private"
	ChatRoomTypeChama   ChatRoomType = "chama"
	ChatRoomTypeGroup   ChatRoomType = "group"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeFile     MessageType = "file"
	MessageTypeLocation MessageType = "location"
	MessageTypeSystem   MessageType = "system"
)

// ChatRoom represents a chat room
type ChatRoom struct {
	ID            string       `json:"id" db:"id"`
	Name          *string      `json:"name,omitempty" db:"name"`
	Type          ChatRoomType `json:"type" db:"type"`
	ChamaID       *string      `json:"chamaId,omitempty" db:"chama_id"`
	CreatedBy     string       `json:"createdBy" db:"created_by"`
	IsActive      bool         `json:"isActive" db:"is_active"`
	LastMessage   *string      `json:"lastMessage,omitempty" db:"last_message"`
	LastMessageAt *time.Time   `json:"lastMessageAt,omitempty" db:"last_message_at"`
	CreatedAt     time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time    `json:"updatedAt" db:"updated_at"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        string      `json:"id" db:"id"`
	RoomID    string      `json:"roomId" db:"room_id"`
	SenderID  string      `json:"senderId" db:"sender_id"`
	Type      MessageType `json:"type" db:"type"`
	Content   string      `json:"content" db:"content"`
	Metadata  string      `json:"metadata" db:"metadata"`
	FileURL   *string     `json:"fileUrl,omitempty" db:"file_url"`
	IsEdited  bool        `json:"isEdited" db:"is_edited"`
	IsDeleted bool        `json:"isDeleted" db:"is_deleted"`
	ReplyToID *string     `json:"replyToId,omitempty" db:"reply_to_id"`
	CreatedAt time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time   `json:"updatedAt" db:"updated_at"`

	// Joined data
	Sender  *models.User `json:"sender,omitempty"`
	ReplyTo *ChatMessage `json:"replyTo,omitempty"`
}

// ChatRoomMember represents a member of a chat room
type ChatRoomMember struct {
	ID         string     `json:"id" db:"id"`
	RoomID     string     `json:"roomId" db:"room_id"`
	UserID     string     `json:"userId" db:"user_id"`
	Role       string     `json:"role" db:"role"`
	JoinedAt   time.Time  `json:"joinedAt" db:"joined_at"`
	LastReadAt *time.Time `json:"lastReadAt,omitempty" db:"last_read_at"`
	IsActive   bool       `json:"isActive" db:"is_active"`
	IsMuted    bool       `json:"isMuted" db:"is_muted"`

	// Joined data
	User *models.User `json:"user,omitempty"`
}

// CreatePrivateChat creates a private chat between two users
func (s *ChatService) CreatePrivateChat(user1ID, user2ID string) (*ChatRoom, error) {
	fmt.Printf("🔍 CreatePrivateChat called with user1ID: %s, user2ID: %s\n", user1ID, user2ID)

	// Prevent users from creating chats with themselves
	if user1ID == user2ID {
		fmt.Printf("❌ Cannot create private chat with yourself: %s\n", user1ID)
		return nil, fmt.Errorf("cannot create private chat with yourself")
	}

	// Check if private chat already exists
	fmt.Printf("🔍 Checking for existing private chat room...\n")
	existingRoom, err := s.getPrivateChatRoom(user1ID, user2ID)
	if err == nil {
		fmt.Printf("✅ Found existing private chat room: %s\n", existingRoom.ID)
		return existingRoom, nil
	}
	fmt.Printf("🔍 No existing room found, creating new one. Error: %v\n", err)

	// Create new private chat room
	room := &ChatRoom{
		ID:        uuid.New().String(),
		Type:      ChatRoomTypePrivate,
		CreatedBy: user1ID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert chat room
	roomQuery := `
		INSERT INTO chat_rooms (id, name, type, chama_id, created_by, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	// For private chats, name and chama_id should be NULL
	var nameValue, chamaIDValue interface{}
	if room.Name != nil {
		nameValue = *room.Name
	} else {
		nameValue = nil
	}
	if room.ChamaID != nil {
		chamaIDValue = *room.ChamaID
	} else {
		chamaIDValue = nil
	}

	_, err = tx.Exec(roomQuery, room.ID, nameValue, room.Type, chamaIDValue, room.CreatedBy, room.IsActive, room.CreatedAt, room.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat room: %w", err)
	}

	// Add both users as members
	members := []string{user1ID, user2ID}
	for _, userID := range members {
		member := &ChatRoomMember{
			ID:       uuid.New().String(),
			RoomID:   room.ID,
			UserID:   userID,
			Role:     "member",
			JoinedAt: time.Now(),
			IsActive: true,
			IsMuted:  false,
		}

		memberQuery := `
			INSERT INTO chat_room_members (id, room_id, user_id, role, joined_at, is_active, is_muted)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err = tx.Exec(memberQuery, member.ID, member.RoomID, member.UserID, member.Role, member.JoinedAt, member.IsActive, member.IsMuted)
		if err != nil {
			return nil, fmt.Errorf("failed to add room member: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return room, nil
}

// CreateSupportChat creates a support chat between admin and user (allows same user)
func (s *ChatService) CreateSupportChat(adminID, userID string, context map[string]interface{}) (*ChatRoom, error) {
	fmt.Printf("🔍 CreateSupportChat called with adminID: %s, userID: %s\n", adminID, userID)

	// For support chats, we allow admin to chat with any user, including themselves for testing
	// Check if support chat already exists for this context
	var existingRoomID string
	if context != nil {
		if supportRequestID, ok := context["supportRequestId"].(string); ok {
			// Look for existing support chat for this support request
			query := `
				SELECT cr.id FROM chat_rooms cr
				WHERE cr.type = 'support'
				AND cr.context LIKE '%"supportRequestId":"' || ? || '"%'
				AND cr.is_active = true
				LIMIT 1
			`
			err := s.db.QueryRow(query, supportRequestID).Scan(&existingRoomID)
			if err == nil {
				fmt.Printf("✅ Found existing support chat room: %s\n", existingRoomID)
				// Get the full room details
				return s.GetChatRoomByID(existingRoomID)
			}
		}
	}

	// Create new support chat room
	room := &ChatRoom{
		ID:        uuid.New().String(),
		Type:      "support", // Use support type
		CreatedBy: adminID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Serialize context
	var contextJSON string
	if context != nil {
		contextBytes, err := json.Marshal(context)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize context: %w", err)
		}
		contextJSON = string(contextBytes)
	} else {
		contextJSON = "{}"
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert chat room with context
	roomQuery := `
		INSERT INTO chat_rooms (id, name, type, chama_id, created_by, is_active, created_at, updated_at, context)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(roomQuery, room.ID, nil, room.Type, nil, room.CreatedBy, room.IsActive, room.CreatedAt, room.UpdatedAt, contextJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create support chat room: %w", err)
	}

	// Add both admin and user as members (even if they're the same person for testing)
	members := []string{adminID}
	if userID != adminID {
		members = append(members, userID)
	}

	for _, memberID := range members {
		member := &ChatRoomMember{
			ID:       uuid.New().String(),
			RoomID:   room.ID,
			UserID:   memberID,
			Role:     "member",
			JoinedAt: time.Now(),
			IsActive: true,
			IsMuted:  false,
		}

		memberQuery := `
			INSERT INTO chat_room_members (id, room_id, user_id, role, joined_at, is_active, is_muted)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err = tx.Exec(memberQuery, member.ID, member.RoomID, member.UserID, member.Role, member.JoinedAt, member.IsActive, member.IsMuted)
		if err != nil {
			return nil, fmt.Errorf("failed to add room member: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("✅ Support chat created successfully: %s\n", room.ID)
	return room, nil
}

// CreateChamaChat creates a chat room for a chama or returns existing one
func (s *ChatService) CreateChamaChat(chamaID, createdBy string) (*ChatRoom, error) {
	// Check if chama chat room already exists
	existingRoom, err := s.getChamaChatRoom(chamaID)
	if err == nil {
		return existingRoom, nil
	}

	// Get chama details
	var chamaName string
	chamaQuery := "SELECT name FROM chamas WHERE id = ?"
	err = s.db.QueryRow(chamaQuery, chamaID).Scan(&chamaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama: %w", err)
	}

	room := &ChatRoom{
		ID:        uuid.New().String(),
		Name:      &chamaName,
		Type:      ChatRoomTypeChama,
		ChamaID:   &chamaID,
		CreatedBy: createdBy,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert chat room
	roomQuery := `
		INSERT INTO chat_rooms (id, name, type, chama_id, created_by, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	// For chama chats, both name and chama_id should have values
	var nameValue, chamaIDValue interface{}
	if room.Name != nil {
		nameValue = *room.Name
	} else {
		nameValue = nil
	}
	if room.ChamaID != nil {
		chamaIDValue = *room.ChamaID
	} else {
		chamaIDValue = nil
	}

	_, err = tx.Exec(roomQuery, room.ID, nameValue, room.Type, chamaIDValue, room.CreatedBy, room.IsActive, room.CreatedAt, room.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat room: %w", err)
	}

	// Add all chama members to the chat
	membersQuery := "SELECT user_id FROM chama_members WHERE chama_id = ? AND is_active = true"
	rows, err := tx.Query(membersQuery, chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		member := &ChatRoomMember{
			ID:       uuid.New().String(),
			RoomID:   room.ID,
			UserID:   userID,
			Role:     "member",
			JoinedAt: time.Now(),
			IsActive: true,
			IsMuted:  false,
		}

		memberQuery := `
			INSERT INTO chat_room_members (id, room_id, user_id, role, joined_at, is_active, is_muted)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err = tx.Exec(memberQuery, member.ID, member.RoomID, member.UserID, member.Role, member.JoinedAt, member.IsActive, member.IsMuted)
		if err != nil {
			return nil, fmt.Errorf("failed to add room member: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return room, nil
}

// SendMessage sends a message to a chat room
func (s *ChatService) SendMessage(roomID, senderID string, messageType MessageType, content string, metadata map[string]interface{}, replyToID *string) (*ChatMessage, error) {
	// Check if user is a member of the room
	isMember, err := s.isRoomMember(roomID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this chat room")
	}

	// Serialize metadata
	metadataJSON := "{}"
	if metadata != nil {
		metadataBytes, err := utils.JSONMarshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	message := &ChatMessage{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      messageType,
		Content:   content,
		Metadata:  metadataJSON,
		IsEdited:  false,
		IsDeleted: false,
		ReplyToID: replyToID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Extract file URL from metadata if present
	var fileURL *string
	if metadata != nil {
		if url, exists := metadata["fileUrl"]; exists {
			if urlStr, ok := url.(string); ok {
				fileURL = &urlStr
			}
		}
	}

	// Process message content
	var storedMessage string
	var encryptionMetadata string

	// Check if message is encrypted
	isEncrypted := false
	if metadata != nil {
		if encrypted, exists := metadata["encrypted"]; exists {
			if encryptedBool, ok := encrypted.(bool); ok && encryptedBool {
				isEncrypted = true
			}
		}
	}

	if isEncrypted {
		// For encrypted messages, content is already the ciphertext
		storedMessage = content
		// Metadata already contains encryption info from frontend
		encryptionMetadata = "{}" // Clear encryption_metadata as it's now in metadata
	} else {
		// For plain text messages
		storedMessage = content
		encryptionMetadata = "{}"
	}

	// Insert message with reduced redundancy
	messageQuery := `
		INSERT INTO chat_messages (
			id, room_id, sender_id, message, type, content, metadata, encryption_metadata, file_url, is_edited, is_deleted, reply_to_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(messageQuery,
		message.ID, message.RoomID, message.SenderID, storedMessage, message.Type, message.Content,
		message.Metadata, encryptionMetadata, fileURL, message.IsEdited, message.IsDeleted, message.ReplyToID,
		message.CreatedAt, message.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Update room's last message - store decrypted content for chat list preview
	var lastMessageContent string
	if isEncrypted {
		// For encrypted messages, store a placeholder for the chat list
		lastMessageContent = "[Encrypted message]"
	} else {
		// For plain text messages, store the actual content
		lastMessageContent = content
	}

	updateRoomQuery := `
		UPDATE chat_rooms
		SET last_message = ?, last_message_at = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.Exec(updateRoomQuery, lastMessageContent, message.CreatedAt, message.UpdatedAt, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to update room: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Populate sender information
	senderQuery := "SELECT first_name, last_name, avatar FROM users WHERE id = ?"
	sender := &models.User{ID: senderID}
	err = s.db.QueryRow(senderQuery, senderID).Scan(&sender.FirstName, &sender.LastName, &sender.Avatar)
	if err != nil {
		// Log error but don't fail the message send
		fmt.Printf("Warning: failed to populate sender info for message %s: %v\n", message.ID, err)
	} else {
		message.Sender = sender
	}

	return message, nil
}

// GetRoomMessages retrieves messages for a chat room (PRIVACY ENFORCED)
func (s *ChatService) GetRoomMessages(roomID, userID string, limit, offset int) ([]*ChatMessage, error) {
	// fmt.Printf("🔒 GetRoomMessages: Privacy check for user %s accessing room %s\n", userID, roomID)

	// PRIVACY: Check if user is a member of the room
	isMember, err := s.isRoomMember(roomID, userID)
	if err != nil {
		fmt.Printf("❌ Privacy check failed for user %s room %s: %v\n", userID, roomID, err)
		return nil, err
	}
	if !isMember {
		fmt.Printf("🚫 PRIVACY VIOLATION BLOCKED: User %s attempted to access room %s without membership\n", userID, roomID)
		return nil, fmt.Errorf("user is not a member of this chat room")
	}

	fmt.Printf("✅ Privacy check passed: User %s is authorized to access room %s\n", userID, roomID)

	query := `
		SELECT m.id, m.room_id, m.sender_id, m.type, m.content, m.metadata, m.file_url,
			   m.is_edited, m.is_deleted, m.reply_to_id, m.created_at, m.updated_at,
			   u.first_name, u.last_name, u.avatar
		FROM chat_messages m
		INNER JOIN users u ON m.sender_id = u.id
		WHERE m.room_id = ? AND m.is_deleted = false
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*ChatMessage
	for rows.Next() {
		message := &ChatMessage{}
		sender := &models.User{}

		err := rows.Scan(
			&message.ID, &message.RoomID, &message.SenderID, &message.Type,
			&message.Content, &message.Metadata, &message.FileURL, &message.IsEdited, &message.IsDeleted,
			&message.ReplyToID, &message.CreatedAt, &message.UpdatedAt,
			&sender.FirstName, &sender.LastName, &sender.Avatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		sender.ID = message.SenderID
		message.Sender = sender

		// Check if message is encrypted and mark it for frontend decryption
		isEncrypted := s.isMessageEncrypted(message.Metadata)
		if isEncrypted {
			// Add encryption flag to metadata for frontend
			var meta map[string]interface{}
			if message.Metadata != "" {
				json.Unmarshal([]byte(message.Metadata), &meta)
			} else {
				meta = make(map[string]interface{})
			}
			meta["encrypted"] = true
			meta["needsDecryption"] = true

			// Re-serialize metadata
			if updatedMeta, err := json.Marshal(meta); err == nil {
				message.Metadata = string(updatedMeta)
			}
		}

		// Also check for fallback encrypted content (base64 with _enc_ pattern)
		if !isEncrypted && strings.Contains(message.Content, "_enc_") && strings.HasSuffix(message.Content, "==") {
			// This is fallback encrypted content - mark it for decryption
			var meta map[string]interface{}
			if message.Metadata != "" {
				json.Unmarshal([]byte(message.Metadata), &meta)
			} else {
				meta = make(map[string]interface{})
			}
			meta["encrypted"] = true
			meta["needsDecryption"] = true
			meta["securityLevel"] = "FALLBACK"

			// Re-serialize metadata
			if updatedMeta, err := json.Marshal(meta); err == nil {
				message.Metadata = string(updatedMeta)
			}
		}

		messages = append(messages, message)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ChatRoomWithParticipants extends ChatRoom with participant information
type ChatRoomWithParticipants struct {
	*ChatRoom
	Participants    []*models.User `json:"participants,omitempty"`
	OtherUserName   *string        `json:"otherUserName,omitempty"`
	OtherUserAvatar *string        `json:"otherUserAvatar,omitempty"`
	OtherUserID     *string        `json:"otherUserId,omitempty"`
	MessageCount    int            `json:"messageCount"`
	UnreadCount     int            `json:"unreadCount"`
	MemberCount     int            `json:"memberCount"`
}

// GetUserChatRooms retrieves chat rooms for a user with participant information (PRIVACY ENFORCED)
func (s *ChatService) GetUserChatRooms(userID string) ([]*ChatRoomWithParticipants, error) {
	fmt.Printf("🔒 GetUserChatRooms: Enforcing privacy for user %s\n", userID)

	// PRIVACY: Only return rooms where user is an ACTIVE member
	query := `
		SELECT
			r.id, r.name, r.type, r.chama_id, r.created_by, r.is_active,
			r.last_message, r.last_message_at, r.created_at, r.updated_at,
			COALESCE(member_counts.member_count, 0) as member_count,
			COALESCE(message_counts.message_count, 0) as message_count
		FROM chat_rooms r
		INNER JOIN chat_room_members m ON r.id = m.room_id
		LEFT JOIN (
			SELECT room_id, COUNT(*) as member_count
			FROM chat_room_members
			WHERE is_active = true
			GROUP BY room_id
		) member_counts ON r.id = member_counts.room_id
		LEFT JOIN (
			SELECT room_id, COUNT(*) as message_count
			FROM chat_messages
			WHERE is_deleted = false
			GROUP BY room_id
		) message_counts ON r.id = message_counts.room_id
		WHERE m.user_id = ? AND m.is_active = true AND r.is_active = true
		ORDER BY r.last_message_at DESC, r.created_at DESC
		LIMIT 50
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*ChatRoomWithParticipants
	for rows.Next() {
		room := &ChatRoom{}
		var memberCount, messageCount int

		err := rows.Scan(
			&room.ID, &room.Name, &room.Type, &room.ChamaID, &room.CreatedBy,
			&room.IsActive, &room.LastMessage, &room.LastMessageAt,
			&room.CreatedAt, &room.UpdatedAt, &memberCount, &messageCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat room: %w", err)
		}

		// Handle last message decryption for chat list
		if room.LastMessage != nil && *room.LastMessage != "" {
			lastMsgStr := *room.LastMessage

			// Check if this looks like encrypted content
			if strings.Contains(lastMsgStr, "_enc_") ||
				strings.Contains(lastMsgStr, "_encrypted_") ||
				strings.Contains(lastMsgStr, "_cipher_") ||
				strings.Contains(lastMsgStr, "_secure_") ||
				(strings.Contains(lastMsgStr, "==") && strings.HasSuffix(lastMsgStr, "==")) {

				// Try to get the actual last message and decrypt it
				lastMessage, err := s.getLastMessageForRoom(room.ID, userID)
				if err == nil && lastMessage != nil {
					// Try to decrypt the message content
					decryptedContent := s.decryptMessageContentForList(lastMessage.Content, lastMessage.Metadata, room.ID, userID)
					room.LastMessage = &decryptedContent
				} else {
					// Fallback to placeholder if decryption fails
					placeholder := "[Encrypted message]"
					room.LastMessage = &placeholder
				}
			}
		}

		// Create extended room with basic info (no additional queries)
		extendedRoom := &ChatRoomWithParticipants{
			ChatRoom:     room,
			MemberCount:  memberCount,
			MessageCount: messageCount,
			UnreadCount:  0, // Skip unread count for performance
		}

		// For private chats, get the other user's information efficiently
		if room.Type == ChatRoomTypePrivate {
			otherUserQuery := `
				SELECT u.id, u.first_name, u.last_name, u.avatar
				FROM users u
				INNER JOIN chat_room_members m ON u.id = m.user_id
				WHERE m.room_id = ? AND m.user_id != ? AND m.is_active = true
				LIMIT 1
			`
			var otherUser models.User
			err := s.db.QueryRow(otherUserQuery, room.ID, userID).Scan(
				&otherUser.ID, &otherUser.FirstName, &otherUser.LastName, &otherUser.Avatar,
			)
			if err == nil {
				otherUserName := otherUser.FirstName + " " + otherUser.LastName
				extendedRoom.OtherUserName = &otherUserName
				extendedRoom.OtherUserAvatar = otherUser.Avatar
				extendedRoom.OtherUserID = &otherUser.ID
			}
		}

		rooms = append(rooms, extendedRoom)
	}

	fmt.Printf("✅ GetUserChatRooms completed: found %d rooms for user %s\n", len(rooms), userID)
	return rooms, nil
}

// IsUserMemberOfRoom checks if a user is a member of a specific chat room
func (s *ChatService) IsUserMemberOfRoom(roomID, userID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM chat_room_members
		WHERE room_id = ? AND user_id = ? AND is_active = true
	`

	var count int
	err := s.db.QueryRow(query, roomID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check room membership: %w", err)
	}

	return count > 0, nil
}

// AddUserToRoom adds a user to a chat room
func (s *ChatService) AddUserToRoom(roomID, userID string) error {
	// First check if the room exists
	var roomExists bool
	checkRoomQuery := `SELECT EXISTS(SELECT 1 FROM chat_rooms WHERE id = ? AND is_active = true)`
	err := s.db.QueryRow(checkRoomQuery, roomID).Scan(&roomExists)
	if err != nil {
		return fmt.Errorf("failed to check if room exists: %w", err)
	}

	if !roomExists {
		return fmt.Errorf("chat room does not exist or is not active")
	}

	// Check if user is already a member
	isMember, err := s.IsUserMemberOfRoom(roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to check existing membership: %w", err)
	}

	if isMember {
		return nil // User is already a member, no need to add again
	}

	// Add user to room
	memberID := uuid.New().String()
	insertQuery := `
		INSERT INTO chat_room_members (id, room_id, user_id, role, joined_at, is_active)
		VALUES (?, ?, ?, 'member', datetime('now'), true)
	`

	_, err = s.db.Exec(insertQuery, memberID, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to add user to room: %w", err)
	}

	fmt.Printf("✅ Added user %s to room %s\n", userID, roomID)
	return nil
}

// getLastMessageForRoom gets the last message for a room for decryption purposes
func (s *ChatService) getLastMessageForRoom(roomID, userID string) (*ChatMessage, error) {
	query := `
		SELECT m.id, m.room_id, m.sender_id, m.type, m.content, m.metadata, m.file_url,
			   m.is_edited, m.is_deleted, m.reply_to_id, m.created_at, m.updated_at
		FROM chat_messages m
		WHERE m.room_id = ? AND m.is_deleted = false
		ORDER BY m.created_at DESC
		LIMIT 1
	`

	message := &ChatMessage{}
	err := s.db.QueryRow(query, roomID).Scan(
		&message.ID, &message.RoomID, &message.SenderID, &message.Type,
		&message.Content, &message.Metadata, &message.FileURL, &message.IsEdited,
		&message.IsDeleted, &message.ReplyToID, &message.CreatedAt, &message.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return message, nil
}

// decryptMessageContentForList decrypts message content for chat list display
func (s *ChatService) decryptMessageContentForList(content, metadata, roomID, userID string) string {
	// Parse metadata
	var meta map[string]interface{}
	if metadata != "" {
		json.Unmarshal([]byte(metadata), &meta)
	} else {
		meta = make(map[string]interface{})
	}

	// Check if message is encrypted
	if encrypted, exists := meta["encrypted"]; exists {
		if encryptedBool, ok := encrypted.(bool); ok && encryptedBool {
			// Try to decrypt based on security level
			securityLevel, _ := meta["securityLevel"].(string)

			if securityLevel == "GROUP_ENCRYPTED" {
				// Group message - try to decrypt
				encryptedMsg := EncryptedMessage{
					Version:       "1.0",
					SenderID:      getStringFromMeta(meta, "senderId", ""),
					RecipientID:   roomID,
					Ciphertext:    getStringFromMeta(meta, "ciphertext", content),
					IV:            getStringFromMeta(meta, "iv", ""),
					AuthTag:       getStringFromMeta(meta, "authTag", ""),
					SessionID:     roomID,
					MessageNumber: 0,
					Timestamp:     time.Unix(int64(getFloatFromMeta(meta, "timestamp", float64(time.Now().Unix())))/1000, 0),
					SecurityLevel: securityLevel,
					IntegrityHash: getStringFromMeta(meta, "integrityHash", ""),
				}

				decryptedText, _, err := s.e2eeService.DecryptGroupMessage(roomID, &encryptedMsg)
				if err == nil {
					return decryptedText
				}
			} else {
				// Private message - try to decrypt
				encryptedMsg := EncryptedMessage{
					Version:       "1.0",
					SenderID:      getStringFromMeta(meta, "senderId", ""),
					RecipientID:   getStringFromMeta(meta, "recipientId", ""),
					Ciphertext:    getStringFromMeta(meta, "ciphertext", content),
					IV:            getStringFromMeta(meta, "iv", ""),
					AuthTag:       getStringFromMeta(meta, "authTag", ""),
					SessionID:     getStringFromMeta(meta, "sessionId", ""),
					MessageNumber: int64(getFloatFromMeta(meta, "messageNumber", 0)),
					Timestamp:     time.Unix(int64(getFloatFromMeta(meta, "timestamp", float64(time.Now().Unix())))/1000, 0),
					SecurityLevel: getStringFromMeta(meta, "securityLevel", "MILITARY_GRADE"),
					IntegrityHash: getStringFromMeta(meta, "integrityHash", ""),
				}

				decryptedText, _, err := s.e2eeService.DecryptMessage(&encryptedMsg)
				if err == nil {
					return decryptedText
				}
			}
		}
	}

	// If not encrypted or decryption failed, return original content with cleanup
	cleanedContent := content

	// Remove encryption artifacts
	patterns := []string{
		`_enc_\d+_[a-zA-Z0-9]+$`,
		`_encrypted_\d+_[a-zA-Z0-9]+$`,
		`_cipher_[a-zA-Z0-9]+$`,
		`_secure_[a-zA-Z0-9]+$`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, cleanedContent); matched {
			re := regexp.MustCompile(pattern)
			cleanedContent = re.ReplaceAllString(cleanedContent, "")
		}
	}

	return cleanedContent
}

// getRoomParticipants gets all participants for a room
func (s *ChatService) getRoomParticipants(roomID string) ([]*models.User, error) {
	query := `
		SELECT u.id, u.first_name, u.last_name, u.email, u.avatar, u.phone
		FROM users u
		INNER JOIN chat_room_members m ON u.id = m.user_id
		WHERE m.room_id = ? AND m.is_active = true
		ORDER BY u.first_name, u.last_name
	`

	rows, err := s.db.Query(query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room participants: %w", err)
	}
	defer rows.Close()

	var participants []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID, &user.FirstName, &user.LastName,
			&user.Email, &user.Avatar, &user.Phone,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, user)
	}

	return participants, nil
}

// getRoomMessageCount gets the total message count for a room
func (s *ChatService) getRoomMessageCount(roomID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM chat_messages WHERE room_id = ? AND is_deleted = false`
	err := s.db.QueryRow(query, roomID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}
	return count, nil
}

// getRoomUnreadCount gets the unread message count for a user in a room
func (s *ChatService) getRoomUnreadCount(roomID, userID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM chat_messages m
		LEFT JOIN chat_room_members rm ON rm.room_id = m.room_id AND rm.user_id = ?
		WHERE m.room_id = ? AND m.is_deleted = false
		AND m.sender_id != ?
		AND (rm.last_read_at IS NULL OR m.created_at > rm.last_read_at)
	`
	err := s.db.QueryRow(query, userID, roomID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	return count, nil
}

// MarkMessagesAsRead marks messages as read for a user
func (s *ChatService) MarkMessagesAsRead(roomID, userID string) error {
	now := time.Now()
	query := `
		UPDATE chat_room_members
		SET last_read_at = ?
		WHERE room_id = ? AND user_id = ?
	`

	_, err := s.db.Exec(query, now, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	return nil
}

// Helper methods

func (s *ChatService) getPrivateChatRoom(user1ID, user2ID string) (*ChatRoom, error) {
	query := `
		SELECT r.id, r.name, r.type, r.chama_id, r.created_by, r.is_active,
			   r.last_message, r.last_message_at, r.created_at, r.updated_at
		FROM chat_rooms r
		WHERE r.type = ? AND r.id IN (
			SELECT m1.room_id FROM chat_room_members m1
			INNER JOIN chat_room_members m2 ON m1.room_id = m2.room_id
			WHERE m1.user_id = ? AND m2.user_id = ? AND m1.is_active = true AND m2.is_active = true
		)
	`

	room := &ChatRoom{}
	err := s.db.QueryRow(query, ChatRoomTypePrivate, user1ID, user2ID).Scan(
		&room.ID, &room.Name, &room.Type, &room.ChamaID, &room.CreatedBy,
		&room.IsActive, &room.LastMessage, &room.LastMessageAt,
		&room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (s *ChatService) GetChatRoomByID(roomID string) (*ChatRoom, error) {
	query := `
		SELECT id, name, type, chama_id, created_by, is_active,
			   last_message, last_message_at, created_at, updated_at
		FROM chat_rooms
		WHERE id = ? AND is_active = true
	`

	room := &ChatRoom{}
	err := s.db.QueryRow(query, roomID).Scan(
		&room.ID, &room.Name, &room.Type, &room.ChamaID, &room.CreatedBy,
		&room.IsActive, &room.LastMessage, &room.LastMessageAt,
		&room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (s *ChatService) isRoomMember(roomID, userID string) (bool, error) {
	query := "SELECT COUNT(*) FROM chat_room_members WHERE room_id = ? AND user_id = ? AND is_active = true"
	var count int
	err := s.db.QueryRow(query, roomID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check room membership: %w", err)
	}
	return count > 0, nil
}

// isMessageEncrypted checks if a message is encrypted based on its metadata
func (s *ChatService) isMessageEncrypted(metadata string) bool {
	if metadata == "" {
		// fmt.Printf("🔍 Message metadata is empty\n")
		return false
	}

	// fmt.Printf("🔍 Checking metadata for encryption: %s\n", metadata)

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		fmt.Printf("❌ Failed to parse metadata JSON: %v\n", err)
		return false
	}

	// Check if explicitly marked as not encrypted
	encrypted, exists := meta["encrypted"]
	if exists {
		encryptedBool, ok := encrypted.(bool)
		if ok && !encryptedBool {
			// fmt.Printf("🔍 Message explicitly marked as not encrypted\n")
			return false
		}
	}

	// Check for plain text security level
	securityLevel, exists := meta["securityLevel"]
	if exists {
		securityLevelStr, ok := securityLevel.(string)
		if ok && securityLevelStr == "PLAIN_TEXT" {
			// fmt.Printf("🔍 Message has plain text security level\n")
			return false
		}
	}

	// Default to encrypted if encrypted field is true
	if exists {
		encryptedBool, ok := encrypted.(bool)
		if ok {
			fmt.Printf("✅ Message is encrypted: %t\n", encryptedBool)
			return encryptedBool
		}
	}

	fmt.Printf("🔍 Defaulting to not encrypted\n")
	return false
}

// isPlainTextMessage checks if a message is plain text (not requiring E2EE decryption)
func (s *ChatService) isPlainTextMessage(metadata string) bool {
	if metadata == "" {
		return false
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		return false
	}

	securityLevel, exists := meta["securityLevel"]
	if !exists {
		return false
	}

	securityLevelStr, ok := securityLevel.(string)
	if !ok {
		return false
	}

	return securityLevelStr == "PLAIN_TEXT"
}

func (s *ChatService) getChamaChatRoom(chamaID string) (*ChatRoom, error) {
	query := `
		SELECT r.id, r.name, r.type, r.chama_id, r.created_by, r.is_active,
			   r.last_message, r.last_message_at, r.created_at, r.updated_at
		FROM chat_rooms r
		WHERE r.type = ? AND r.chama_id = ? AND r.is_active = true
	`

	room := &ChatRoom{}
	err := s.db.QueryRow(query, ChatRoomTypeChama, chamaID).Scan(
		&room.ID, &room.Name, &room.Type, &room.ChamaID, &room.CreatedBy,
		&room.IsActive, &room.LastMessage, &room.LastMessageAt,
		&room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return room, nil
}

// AddUserToChamaChat adds a user to an existing chama chat room
func (s *ChatService) AddUserToChamaChat(chamaID, userID string) error {
	// Get chama chat room
	room, err := s.getChamaChatRoom(chamaID)
	if err != nil {
		// If no chat room exists, that's okay - it will be created when needed
		return nil
	}

	// Check if user is already a member
	isMember, err := s.isRoomMember(room.ID, userID)
	if err != nil {
		return fmt.Errorf("failed to check room membership: %w", err)
	}
	if isMember {
		return nil // User is already a member
	}

	// Add user to chat room
	member := &ChatRoomMember{
		ID:       uuid.New().String(),
		RoomID:   room.ID,
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
		IsActive: true,
		IsMuted:  false,
	}

	memberQuery := `
		INSERT INTO chat_room_members (id, room_id, user_id, role, joined_at, is_active, is_muted)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(memberQuery, member.ID, member.RoomID, member.UserID, member.Role, member.JoinedAt, member.IsActive, member.IsMuted)
	if err != nil {
		return fmt.Errorf("failed to add user to chat room: %w", err)
	}

	return nil
}

// GetRoomMembers retrieves members of a chat room
func (s *ChatService) GetRoomMembers(roomID, userID string) ([]*ChatRoomMember, error) {
	// PRIVACY: First check if user is a member of the room
	isMember, err := s.IsUserMemberOfRoom(roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this chat room")
	}

	query := `
		SELECT m.id, m.room_id, m.user_id, m.role, m.joined_at, m.last_read_at, m.is_active, m.is_muted,
			   u.first_name, u.last_name, u.email, u.avatar
		FROM chat_room_members m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = ? AND m.is_active = true
		ORDER BY m.joined_at ASC
	`

	rows, err := s.db.Query(query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	defer rows.Close()

	var members []*ChatRoomMember
	for rows.Next() {
		member := &ChatRoomMember{}
		user := &models.User{}

		err := rows.Scan(
			&member.ID, &member.RoomID, &member.UserID, &member.Role,
			&member.JoinedAt, &member.LastReadAt, &member.IsActive, &member.IsMuted,
			&user.FirstName, &user.LastName, &user.Email, &user.Avatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room member: %w", err)
		}

		member.User = user
		members = append(members, member)
	}

	return members, nil
}

// DeleteChatRoom removes a user from a chat room (soft delete for private chats)
func (s *ChatService) DeleteChatRoom(roomID, userID string) error {
	// Check if user is a member of the room
	isMember, err := s.isRoomMember(roomID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("user is not a member of this chat room")
	}

	// For private chats, we'll deactivate the membership
	// For group chats, we'll remove the user from the room
	query := `
		UPDATE chat_room_members
		SET is_active = false
		WHERE room_id = ? AND user_id = ?
	`

	_, err = s.db.Exec(query, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete chat room: %w", err)
	}

	return nil
}

// ClearChatRoom marks all messages as deleted for a user (soft delete)
func (s *ChatService) ClearChatRoom(roomID, userID string) error {
	// Check if user is a member of the room
	isMember, err := s.isRoomMember(roomID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("user is not a member of this chat room")
	}

	// We'll implement this as updating the user's last_read_at to current time
	// and add a flag to indicate messages before this time should be hidden
	// For now, we'll just update the last_read_at
	now := time.Now()
	query := `
		UPDATE chat_room_members
		SET last_read_at = ?
		WHERE room_id = ? AND user_id = ?
	`

	_, err = s.db.Exec(query, now, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to clear chat room: %w", err)
	}

	return nil
}
