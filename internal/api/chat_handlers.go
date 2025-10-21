package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// GetChatRooms retrieves chat rooms for the authenticated user (OPTIMIZED)
func GetChatRooms(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// fmt.Printf("🔍 GetChatRooms: Starting request for user %s\n", userID)

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Use the proper chat service with timeout protection
	chatService := services.NewChatService(db.(*sql.DB))

	fmt.Printf("🔄 GetChatRooms: Calling GetUserChatRooms for user %s\n", userID)
	rooms, err := chatService.GetUserChatRooms(userID)
	if err != nil {
		fmt.Printf("❌ GetChatRooms: Failed for user %s: %v\n", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve chat rooms: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ GetChatRooms: Success for user %s, returning %d rooms\n", userID, len(rooms))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rooms,
	})
}

// CreateChatRoom creates a new chat room
func CreateChatRoom(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		Type        string                 `json:"type" binding:"required"`
		Name        string                 `json:"name"`
		ChamaID     string                 `json:"chamaId"`
		UserIDs     []string               `json:"userIds"`
		RecipientID string                 `json:"recipientId"`
		Context     map[string]interface{} `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ Failed to bind JSON in CreateChatRoom: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 CreateChatRoom request - UserID: %s, Type: %s, RecipientID: %s\n", userID, req.Type, req.RecipientID)

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	chatService := services.NewChatService(db.(*sql.DB))

	switch req.Type {
	case "private":
		if req.RecipientID == "" {
			fmt.Printf("❌ Missing RecipientID for private chat\n")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Recipient ID required for private chat",
			})
			return
		}

		fmt.Printf("🔄 Creating private chat between %s and %s\n", userID, req.RecipientID)

		// Create or get existing private chat room
		room, err := chatService.CreatePrivateChat(userID, req.RecipientID)
		if err != nil {
			fmt.Printf("❌ Failed to create private chat: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create private chat: " + err.Error(),
			})
			return
		}

		fmt.Printf("✅ Private chat created successfully: %s\n", room.ID)

		// If context is provided (e.g., product inquiry), send an initial message
		if req.Context != nil {
			if productID, exists := req.Context["productId"]; exists {
				if productName, nameExists := req.Context["productName"]; nameExists {
					// Send initial context message about the product
					contextMessage := fmt.Sprintf("Hi! I'm interested in your product: %s", productName)
					metadata := map[string]interface{}{
						"type":      "product_inquiry",
						"productId": productID,
					}

					_, err := chatService.SendMessage(room.ID, userID, services.MessageTypeText, contextMessage, metadata, nil)
					if err != nil {
						// Log error but don't fail the chat creation
						fmt.Printf("Failed to send initial product inquiry message: %v\n", err)
					}
				}
			}
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    room,
		})
		return

	case "chama":
		if req.ChamaID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Chama ID required for chama chat",
			})
			return
		}

		// Create chama chat room
		room, err := chatService.CreateChamaChat(req.ChamaID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create chama chat: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    room,
		})
		return

	case "support":
		// For support chats, we need a user ID (the user who needs support)
		supportUserID := req.RecipientID
		if supportUserID == "" && len(req.UserIDs) > 0 {
			// Try to get it from the UserID field if RecipientID is not provided
			supportUserID = req.UserIDs[0]
		}

		// Check if we have a userId in the request body
		if supportUserID == "" {
			if userIDFromBody, ok := req.Context["userId"].(string); ok {
				supportUserID = userIDFromBody
			}
		}

		if supportUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "User ID required for support chat",
			})
			return
		}

		fmt.Printf("🔄 Creating support chat for user %s (admin: %s)\n", supportUserID, userID)

		// Create support chat room (similar to private chat but allows admin to chat with user)
		room, err := chatService.CreateSupportChat(userID, supportUserID, req.Context)
		if err != nil {
			fmt.Printf("❌ Failed to create support chat: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create support chat: " + err.Error(),
			})
			return
		}

		fmt.Printf("✅ Support chat created successfully: %s\n", room.ID)
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    room,
		})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid chat room type",
		})
		return
	}
}

// GetChatRoom retrieves a specific chat room
func GetChatRoom(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// For now, return a simple response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":     roomID,
			"name":   "Chat Room",
			"type":   "private",
			"userId": userID,
		},
	})
}

// JoinChatRoom allows a user to join a chat room
func JoinChatRoom(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Room ID is required",
		})
		return
	}

	fmt.Printf("🔄 JoinChatRoom: User %s joining room %s\n", userID, roomID)

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	chatService := services.NewChatService(db.(*sql.DB))

	// Check if user is already a member
	isMember, err := chatService.IsUserMemberOfRoom(roomID, userID)
	if err != nil {
		fmt.Printf("❌ JoinChatRoom: Error checking membership for user %s in room %s: %v\n", userID, roomID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check room membership: " + err.Error(),
		})
		return
	}

	if isMember {
		fmt.Printf("✅ JoinChatRoom: User %s already member of room %s\n", userID, roomID)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User is already a member of this room",
		})
		return
	}

	// Add user to room
	err = chatService.AddUserToRoom(roomID, userID)
	if err != nil {
		fmt.Printf("❌ JoinChatRoom: Failed to add user %s to room %s: %v\n", userID, roomID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to join room: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ JoinChatRoom: User %s successfully joined room %s\n", userID, roomID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully joined room",
	})
}

// GetChatMessages retrieves messages for a chat room
func GetChatMessages(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Use the proper chat service
	chatService := services.NewChatService(db.(*sql.DB))
	messages, err := chatService.GetRoomMessages(roomID, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve messages: " + err.Error(),
		})
		return
	}

	// Get E2EE service for decryption
	e2eeService, e2eeExists := c.Get("e2eeService")
	if !e2eeExists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	// Decrypt encrypted messages
	for _, message := range messages {
		if message.Metadata != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(message.Metadata), &meta); err == nil {
				if encrypted, exists := meta["encrypted"]; exists && encrypted.(bool) {
					fmt.Printf("🔓 Decrypting message %s for user %s\n", message.ID, userID)

					// Check if message needs decryption
					if needsDecryption, ok := meta["needsDecryption"].(bool); ok && needsDecryption {
						securityLevel := getStringFromMeta(meta, "securityLevel", "")

						if securityLevel == "GROUP_ENCRYPTED" {
							// Group message decryption - use the roomID from the function parameter
							if roomID == "" {
								fmt.Printf("❌ Missing roomID parameter for group message decryption\n")
								message.Content = "[Failed to decrypt message - missing room ID]"
								meta["decryptionError"] = "Missing room ID parameter for group decryption"
							} else {
								// Prepare encrypted message structure for group decryption
								encryptedMsg := services.EncryptedMessage{
									Version:       "1.0",
									SenderID:      getStringFromMeta(meta, "senderId", ""),
									RecipientID:   roomID,
									Ciphertext:    getStringFromMeta(meta, "ciphertext", ""),
									IV:           getStringFromMeta(meta, "iv", ""),
									AuthTag:      getStringFromMeta(meta, "authTag", ""),
									SessionID:    roomID,
									MessageNumber: 0,
									Timestamp:    time.Unix(int64(getFloatFromMeta(meta, "timestamp", float64(time.Now().Unix())))/1000, 0),
									SecurityLevel: "GROUP_ENCRYPTED",
									IntegrityHash: getStringFromMeta(meta, "integrityHash", ""),
								}

								// Decrypt the group message
								decryptedText, decryptedMeta, err := e2eeService.(*services.MilitaryGradeE2EEService).DecryptGroupMessage(roomID, &encryptedMsg)
								if err != nil {
									fmt.Printf("❌ Failed to decrypt group message %s: %v\n", message.ID, err)
									message.Content = "[Failed to decrypt message]"
									meta["decryptionError"] = err.Error()
								} else {
									fmt.Printf("✅ Group message %s decrypted successfully\n", message.ID)
									message.Content = decryptedText

									// Update metadata to reflect successful decryption
									meta["decrypted"] = true
									meta["decryptionTimestamp"] = time.Now().Unix()
									if decryptedMeta != nil {
										// Merge decrypted metadata
										for k, v := range decryptedMeta {
											meta[k] = v
										}
									}
								}
							}
						} else {
							// Private message decryption (existing logic)
							encryptedMsg := services.EncryptedMessage{
								Version:       "1.0",
								SenderID:      getStringFromMeta(meta, "senderId", ""),
								RecipientID:   getStringFromMeta(meta, "recipientId", ""),
								Ciphertext:    getStringFromMeta(meta, "ciphertext", ""),
								IV:           getStringFromMeta(meta, "iv", ""),
								AuthTag:      getStringFromMeta(meta, "authTag", ""),
								SessionID:    getStringFromMeta(meta, "sessionId", ""),
								MessageNumber: int64(getFloatFromMeta(meta, "messageNumber", 0)),
								Timestamp:    time.Unix(int64(getFloatFromMeta(meta, "timestamp", float64(time.Now().Unix())))/1000, 0),
								SecurityLevel: getStringFromMeta(meta, "securityLevel", "MILITARY_GRADE"),
								IntegrityHash: getStringFromMeta(meta, "integrityHash", ""),
							}

							// Decrypt the message
							decryptedText, decryptedMeta, err := e2eeService.(*services.MilitaryGradeE2EEService).DecryptMessage(&encryptedMsg)
							if err != nil {
								fmt.Printf("❌ Failed to decrypt message %s: %v\n", message.ID, err)
								// Keep encrypted content but mark as decryption failed
								message.Content = "[Failed to decrypt message]"
								meta["decryptionError"] = err.Error()
							} else {
								fmt.Printf("✅ Message %s decrypted successfully\n", message.ID)
								message.Content = decryptedText

								// Update metadata to reflect successful decryption
								meta["decrypted"] = true
								meta["decryptionTimestamp"] = time.Now().Unix()
								if decryptedMeta != nil {
									// Merge decrypted metadata
									for k, v := range decryptedMeta {
										meta[k] = v
									}
								}
							}
						}

						// Update message metadata
						if updatedMeta, err := json.Marshal(meta); err == nil {
							message.Metadata = string(updatedMeta)
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    messages,
	})
}

// SendMessage sends a message to a chat room
func SendMessage(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Check if this is a multipart/form-data request (file upload)
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		handleFileMessage(c, userID, roomID)
		return
	}

	// Handle regular JSON message
	var req struct {
		Type          string            `json:"type" binding:"required"`
		Content       string            `json:"content" binding:"required"`
		Metadata      json.RawMessage   `json:"metadata"`
		ReplyToID     *string           `json:"replyToId"`
		IsEncrypted   bool              `json:"isEncrypted"`
		SecurityLevel string            `json:"securityLevel"`
		RecipientID   string            `json:"recipientId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	// Parse metadata
	var metadataMap map[string]interface{}
	if len(req.Metadata) > 0 {
		if err := json.Unmarshal(req.Metadata, &metadataMap); err != nil {
			fmt.Printf("Failed to parse metadata JSON: %v\n", err)
			metadataMap = make(map[string]interface{})
		}
	} else {
		metadataMap = make(map[string]interface{})
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get E2EE service from context
	e2eeService, e2eeExists := c.Get("e2eeService")
	if !e2eeExists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	// Get WebSocket service from context
	wsService, wsExists := c.Get("wsService")

	// Use the proper chat service
	chatService := services.NewChatService(db.(*sql.DB))

	var finalContent string
	var finalMetadata map[string]interface{}

	// Convert type string to MessageType
	var messageType services.MessageType
	switch req.Type {
	case "text":
		messageType = services.MessageTypeText
	case "image":
		messageType = services.MessageTypeImage
	case "file":
		messageType = services.MessageTypeFile
	case "military_encrypted_text":
		messageType = services.MessageTypeText // Store as text but mark as encrypted
	default:
		messageType = services.MessageTypeText
	}

	// ALWAYS ENCRYPT MESSAGES FOR SECURITY
	fmt.Printf("🔐 ENCRYPTING MESSAGE: User %s in room %s\n", userID, roomID)

	// Get room information to determine chat type
	room, err := chatService.GetChatRoomByID(roomID)
	if err != nil {
		fmt.Printf("❌ Failed to get room information: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get room information",
		})
		return
	}

	// Handle encryption based on actual room type from database
	if room.Type == "private" {
		// Get room members for private chat
		roomMembers, err := chatService.GetRoomMembers(roomID, userID)
		if err != nil {
			fmt.Printf("❌ Failed to get room members for private chat: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to prepare message encryption",
			})
			return
		}

		// Find the other user
		var recipientID string
		for _, member := range roomMembers {
			if member.UserID != userID {
				recipientID = member.UserID
				break
			}
		}

		if recipientID == "" {
			fmt.Printf("❌ No recipient found for private chat encryption\n")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Cannot determine message recipient for encryption",
			})
			return
		}

		// Encrypt the message using E2EE service
		encryptedMessage, err := e2eeService.(*services.MilitaryGradeE2EEService).EncryptMessage(userID, recipientID, req.Content, metadataMap)
		if err != nil {
			fmt.Printf("❌ Failed to encrypt message: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to encrypt message: " + err.Error(),
			})
			return
		}

		// Store the encrypted content
		finalContent = encryptedMessage.Ciphertext
		finalMetadata = map[string]interface{}{
			"encrypted":       true,
			"securityLevel":   "MILITARY_GRADE",
			"needsDecryption": true,
			"ciphertext":      encryptedMessage.Ciphertext,
			"iv":             encryptedMessage.IV,
			"authTag":        encryptedMessage.AuthTag,
			"sessionId":      encryptedMessage.SessionID,
			"messageNumber":  encryptedMessage.MessageNumber,
			"integrityHash":  encryptedMessage.IntegrityHash,
			"timestamp":      encryptedMessage.Timestamp,
			"messageId":      fmt.Sprintf("msg_%d_%s", time.Now().Unix(), userID),
			"senderId":       userID,
			"recipientId":    recipientID,
		}
	} else {
		// Group chat (group, chama, support) - encrypt with group E2EE
		fmt.Printf("🔐 %s chat message - encrypting with group E2EE\n", room.Type)

		// Encrypt the message using group encryption
		encryptedMessage, err := e2eeService.(*services.MilitaryGradeE2EEService).EncryptGroupMessage(roomID, userID, req.Content, metadataMap)
		if err != nil {
			fmt.Printf("❌ Failed to encrypt group message: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to encrypt group message: " + err.Error(),
			})
			return
		}

		// Store the encrypted content
		finalContent = encryptedMessage.Ciphertext
		finalMetadata = map[string]interface{}{
			"encrypted":       true,
			"securityLevel":   "GROUP_ENCRYPTED",
			"needsDecryption": true,
			"chatType":        string(room.Type),
			"ciphertext":      encryptedMessage.Ciphertext,
			"iv":             encryptedMessage.IV,
			"authTag":        encryptedMessage.AuthTag,
			"sessionId":      encryptedMessage.SessionID,
			"integrityHash":  encryptedMessage.IntegrityHash,
			"timestamp":      encryptedMessage.Timestamp,
			"messageId":      fmt.Sprintf("group_msg_%d_%s", time.Now().Unix(), userID),
			"senderId":       userID,
			"roomId":         roomID,
		}
	}

	// Merge any additional metadata
	for k, v := range metadataMap {
		if k != "encrypted" && k != "securityLevel" { // Don't override encryption metadata
			finalMetadata[k] = v
		}
	}

	fmt.Printf("✅ Message encrypted successfully with military-grade E2EE\n")

	// Send message using chat service
	message, err := chatService.SendMessage(roomID, userID, messageType, finalContent, finalMetadata, req.ReplyToID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send message: " + err.Error(),
		})
		return
	}

	// Send real-time notification via WebSocket with decrypted content
	if wsExists {
		// Create a copy of the message with decrypted content for WebSocket broadcast
		wsMessage := *message // Copy the message

		// Decrypt the content for WebSocket broadcast (similar to GetChatMessages)
		if finalMetadata["encrypted"].(bool) {
			if room.Type == "private" {
				// Get room members for private chat to find recipient
				roomMembers, err := chatService.GetRoomMembers(roomID, userID)
				if err != nil {
					fmt.Printf("❌ Failed to get room members for WebSocket decryption: %v\n", err)
					wsMessage.Content = "[Failed to decrypt message]"
				} else {
					// Find the other user
					var wsRecipientID string
					for _, member := range roomMembers {
						if member.UserID != userID {
							wsRecipientID = member.UserID
							break
						}
					}

					if wsRecipientID == "" {
						fmt.Printf("❌ No recipient found for WebSocket decryption\n")
						wsMessage.Content = "[Failed to decrypt message]"
					} else {
						// For private messages, decrypt using the stored encrypted message
						encryptedMsg := services.EncryptedMessage{
							Version:       "1.0",
							SenderID:      userID,
							RecipientID:   wsRecipientID,
							Ciphertext:    finalMetadata["ciphertext"].(string),
							IV:           finalMetadata["iv"].(string),
							AuthTag:      finalMetadata["authTag"].(string),
							SessionID:    finalMetadata["sessionId"].(string),
							MessageNumber: int64(finalMetadata["messageNumber"].(float64)),
							Timestamp:    time.Unix(int64(finalMetadata["timestamp"].(float64))/1000, 0),
							SecurityLevel: finalMetadata["securityLevel"].(string),
							IntegrityHash: finalMetadata["integrityHash"].(string),
						}

						decryptedText, _, err := e2eeService.(*services.MilitaryGradeE2EEService).DecryptMessage(&encryptedMsg)
						if err != nil {
							fmt.Printf("❌ Failed to decrypt message for WebSocket: %v\n", err)
							wsMessage.Content = "[Failed to decrypt message]"
						} else {
							wsMessage.Content = decryptedText
						}
					}
				}
			} else {
				// For group messages, decrypt using group decryption
				encryptedMsg := services.EncryptedMessage{
					Version:       "1.0",
					SenderID:      userID,
					RecipientID:   roomID,
					Ciphertext:    finalMetadata["ciphertext"].(string),
					IV:           finalMetadata["iv"].(string),
					AuthTag:      finalMetadata["authTag"].(string),
					SessionID:    roomID,
					MessageNumber: 0,
					Timestamp:    time.Unix(int64(finalMetadata["timestamp"].(float64))/1000, 0),
					SecurityLevel: "GROUP_ENCRYPTED",
					IntegrityHash: finalMetadata["integrityHash"].(string),
				}

				decryptedText, _, err := e2eeService.(*services.MilitaryGradeE2EEService).DecryptGroupMessage(roomID, &encryptedMsg)
				if err != nil {
					fmt.Printf("❌ Failed to decrypt group message for WebSocket: %v\n", err)
					wsMessage.Content = "[Failed to decrypt message]"
				} else {
					wsMessage.Content = decryptedText
				}
			}

			// Update metadata to reflect decryption
			wsMessage.Metadata = `{"decrypted": true, "decryptionTimestamp": ` + fmt.Sprintf("%d", time.Now().Unix()) + `}`
		}

		wsMsg := services.WebSocketMessage{
			Type:   "new_message",
			RoomID: roomID,
			Data:   &wsMessage,
		}
		wsService.(*services.WebSocketService).BroadcastToRoom(roomID, wsMsg)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    message,
	})
}

// MarkMessagesAsRead marks messages as read in a chat room
func MarkMessagesAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	chatService := services.NewChatService(db.(*sql.DB))
	err := chatService.MarkMessagesAsRead(roomID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to mark messages as read",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Messages marked as read",
	})
}

// GetChatRoomMembers retrieves members of a chat room
func GetChatRoomMembers(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	chatService := services.NewChatService(db.(*sql.DB))
	members, err := chatService.GetRoomMembers(roomID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve room members: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// DeleteChatRoom deletes a chat room for the user
func DeleteChatRoom(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	chatService := services.NewChatService(db.(*sql.DB))
	err := chatService.DeleteChatRoom(roomID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete chat room: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Chat room deleted successfully",
	})
}

// ClearChatRoom clears all messages in a chat room for the user
func ClearChatRoom(c *gin.Context) {
	userID := c.GetString("userID")
	roomID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	chatService := services.NewChatService(db.(*sql.DB))
	err := chatService.ClearChatRoom(roomID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to clear chat room: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Chat room cleared successfully",
	})
}

// handleFileMessage handles file upload messages
func handleFileMessage(c *gin.Context, userID, roomID string) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to parse multipart form: " + err.Error(),
		})
		return
	}

	// Get form values
	messageType := c.PostForm("type")
	content := c.PostForm("content")
	metadataStr := c.PostForm("metadata")

	// Parse metadata
	var metadata map[string]interface{}
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	// Handle file upload
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No file uploaded: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads/chat"
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create uploads directory: " + err.Error(),
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

	// Create file URL with full base URL for frontend access
	fileURL := fmt.Sprintf("https://gitrepoa-1.onrender.com/uploads/chat/%s", fileName)

	// Add file URL to metadata
	metadata["fileUrl"] = fileURL
	metadata["fileName"] = header.Filename
	metadata["fileSize"] = header.Size

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get WebSocket service from context
	wsService, wsExists := c.Get("wsService")

	// Use the proper chat service
	chatService := services.NewChatService(db.(*sql.DB))

	// Convert type string to MessageType
	var msgType services.MessageType
	switch messageType {
	case "image":
		msgType = services.MessageTypeImage
	case "file":
		msgType = services.MessageTypeFile
	default:
		msgType = services.MessageTypeImage // Default to image for file uploads
	}

	// Send message using chat service
	message, err := chatService.SendMessage(roomID, userID, msgType, content, metadata, nil)
	if err != nil {
		// Clean up uploaded file on error
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send message: " + err.Error(),
		})
		return
	}

	// Send real-time notification via WebSocket
	if wsExists {
		wsMsg := services.WebSocketMessage{
			Type:   "new_message",
			RoomID: roomID,
			Data:   message,
		}
		wsService.(*services.WebSocketService).BroadcastToRoom(roomID, wsMsg)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    message,
	})
}
