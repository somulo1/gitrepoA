package services

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	RoomID  string      `json:"roomId,omitempty"`
	UserID  string      `json:"userId,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	ID               string
	UserID           string
	Conn             *websocket.Conn
	Send             chan WebSocketMessage
	Hub              *Hub
	WebSocketService *WebSocketService
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan WebSocketMessage

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Room subscriptions - maps roomID to clients
	rooms map[string]map[*Client]bool

	// User connections - maps userID to clients
	users map[string]*Client

	mutex sync.RWMutex
}

// WebSocketService handles WebSocket connections and real-time messaging
type WebSocketService struct {
	hub         *Hub
	upgrader    websocket.Upgrader
	chatService *ChatService
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService(db *sql.DB) *WebSocketService {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan WebSocketMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
		users:      make(map[string]*Client),
	}

	service := &WebSocketService{
		hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin in development
				// In production, you should check the origin properly
				return true
			},
		},
		chatService: NewChatService(db),
	}

	// Start the hub
	go hub.run()

	return service
}

// HandleWebSocket handles WebSocket connections
func (s *WebSocketService) HandleWebSocket(c *gin.Context) {
	// Try to get user ID from context first (set by auth middleware)
	userID, exists := c.Get("userID")

	// If not in context, try to get from query parameter
	if !exists {
		token := c.Query("token")
		if token == "" {
			log.Printf("WebSocket connection rejected: no token provided")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - no token"})
			return
		}

		// Here you would validate the token and extract userID
		// For now, we'll use a simple approach - in production, use proper JWT validation
		tokenPreview := token
		if len(token) > 20 {
			tokenPreview = token[:20] + "..."
		}
		log.Printf("WebSocket token received: %s", tokenPreview)

		// Set a placeholder userID - in production, extract from validated JWT
		userID = "temp_user_from_token"
		c.Set("userID", userID)
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &Client{
		ID:               generateClientID(),
		UserID:           userID.(string),
		Conn:             conn,
		Send:             make(chan WebSocketMessage, 256),
		Hub:              s.hub,
		WebSocketService: s,
	}

	// Register client
	s.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// BroadcastToRoom sends a message to all clients in a specific room
func (s *WebSocketService) BroadcastToRoom(roomID string, message WebSocketMessage) {
	message.RoomID = roomID
	s.hub.broadcast <- message
}

// SendToUser sends a message to a specific user
func (s *WebSocketService) SendToUser(userID string, message WebSocketMessage) {
	s.hub.mutex.RLock()
	client, exists := s.hub.users[userID]
	s.hub.mutex.RUnlock()

	if exists {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(s.hub.users, userID)
		}
	}
}

// Hub methods
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.users[client.UserID] = client
			h.mutex.Unlock()

			// Send connection confirmation
			select {
			case client.Send <- WebSocketMessage{Type: "connected", Message: "Connected to chat server"}:
			default:
				close(client.Send)
				delete(h.clients, client)
				delete(h.users, client.UserID)
			}

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.users, client.UserID)
				close(client.Send)

				// Remove from all rooms
				for roomID, roomClients := range h.rooms {
					if _, inRoom := roomClients[client]; inRoom {
						delete(roomClients, client)
						if len(roomClients) == 0 {
							delete(h.rooms, roomID)
						}
					}
				}
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			if message.RoomID != "" {
				// Broadcast to specific room
				if roomClients, exists := h.rooms[message.RoomID]; exists {
					for client := range roomClients {
						select {
						case client.Send <- message:
						default:
							close(client.Send)
							delete(h.clients, client)
							delete(h.users, client.UserID)
							delete(roomClients, client)
						}
					}
				}
			} else {
				// Broadcast to all clients
				for client := range h.clients {
					select {
					case client.Send <- message:
					default:
						close(client.Send)
						delete(h.clients, client)
						delete(h.users, client.UserID)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if roomClients, exists := h.rooms[roomID]; exists {
		delete(roomClients, client)
		if len(roomClients) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

// Client methods
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		var message WebSocketMessage
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		switch message.Type {
		case "join_room":
			if message.RoomID != "" {
				c.Hub.JoinRoom(c, message.RoomID)
			}
		case "leave_room":
			if message.RoomID != "" {
				c.Hub.LeaveRoom(c, message.RoomID)
			}
		case "send_message":
			// Handle message sending - process and save to database
			c.handleSendMessage(c.WebSocketService, message)
		case "ping":
			// Send pong response
			select {
			case c.Send <- WebSocketMessage{Type: "pong"}:
			default:
				return
			}
		}
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

// handleSendMessage processes a message sent via WebSocket and saves it to the database
func (c *Client) handleSendMessage(wsService *WebSocketService, message WebSocketMessage) {
	// Extract message data
	messageData, ok := message.Data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid message data format for send_message")
		return
	}

	// Extract required fields
	roomID, ok := messageData["roomId"].(string)
	if !ok || roomID == "" {
		log.Printf("Missing or invalid roomId in send_message")
		return
	}

	recipientID, ok := messageData["recipientId"].(string)
	if !ok || recipientID == "" {
		log.Printf("Missing or invalid recipientId in send_message")
		return
	}

	senderID, ok := messageData["senderId"].(string)
	if !ok || senderID == "" {
		log.Printf("Missing or invalid senderId in send_message")
		return
	}

	messageType, ok := messageData["type"].(string)
	if !ok {
		messageType = "military_encrypted_text" // Default to encrypted text
	}

	content, ok := messageData["content"].(string)
	if !ok {
		log.Printf("Missing or invalid content in send_message")
		return
	}

	// Extract metadata
	metadata := make(map[string]interface{})
	if meta, exists := messageData["metadata"]; exists {
		if metaMap, ok := meta.(map[string]interface{}); ok {
			metadata = metaMap
		}
	}

	// Create message payload for database using chat service SendMessage method
	messageObj, err := wsService.chatService.SendMessage(
		roomID,
		senderID,
		MessageType(messageType),
		content,
		metadata,
		nil, // replyToID
	)
	if err != nil {
		log.Printf("Failed to save message to database: %v", err)
		return
	}

	log.Printf("Message saved to database: %v", messageObj.ID)

	// Convert to map for WebSocket broadcast
	savedMessage := map[string]interface{}{
		"id":        messageObj.ID,
		"roomId":    messageObj.RoomID,
		"senderId":  messageObj.SenderID,
		"type":      messageObj.Type,
		"content":   messageObj.Content,
		"metadata":  messageObj.Metadata,
		"fileUrl":   messageObj.FileURL,
		"isEdited":  messageObj.IsEdited,
		"isDeleted": messageObj.IsDeleted,
		"replyToId": messageObj.ReplyToID,
		"createdAt": messageObj.CreatedAt,
		"updatedAt": messageObj.UpdatedAt,
		"sender": map[string]interface{}{
			"id":        messageObj.Sender.ID,
			"firstName": messageObj.Sender.FirstName,
			"lastName":  messageObj.Sender.LastName,
			"avatar":    messageObj.Sender.Avatar,
		},
	}

	// Broadcast the message to all clients in the room
	broadcastMessage := WebSocketMessage{
		Type:   "new_message",
		RoomID: roomID,
		UserID: senderID,
		Data:   savedMessage,
	}

	c.Hub.broadcast <- broadcastMessage
}

// Helper function to generate client ID
func generateClientID() string {
	// Simple client ID generation - in production, use UUID
	return "client_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
