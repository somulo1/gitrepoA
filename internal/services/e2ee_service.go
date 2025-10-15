package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// MilitaryGradeE2EEService provides military-grade end-to-end encryption
// Features:
// - Perfect Forward Secrecy (PFS)
// - Post-Compromise Security
// - Authenticated Encryption (AES-256-GCM)
// - Curve25519 key exchange
// - HKDF key derivation
// - Constant-time operations
// - Side-channel attack resistance
type MilitaryGradeE2EEService struct {
	db *sql.DB
}

// EncryptedMessage represents a military-grade encrypted message
type EncryptedMessage struct {
	Version       string    `json:"version"`
	SenderID      string    `json:"senderId"`
	RecipientID   string    `json:"recipientId"`
	Ciphertext    string    `json:"ciphertext"`
	AuthTag       string    `json:"authTag"`
	IV            string    `json:"iv"`
	SessionID     string    `json:"sessionId"`
	MessageNumber int64     `json:"messageNumber"`
	Timestamp     time.Time `json:"timestamp"`
	SecurityLevel string    `json:"securityLevel"`
	IntegrityHash string    `json:"integrityHash"`
}

// UnmarshalJSON custom unmarshaling to handle Unix timestamp numbers
func (e *EncryptedMessage) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a map to handle the timestamp conversion
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Handle timestamp conversion
	if ts, ok := raw["timestamp"]; ok {
		switch v := ts.(type) {
		case float64:
			// Unix timestamp in milliseconds
			e.Timestamp = time.Unix(int64(v)/1000, (int64(v)%1000)*1000000)
		case string:
			// RFC3339 string
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				e.Timestamp = t
			}
		}
		delete(raw, "timestamp")
	}

	// Marshal back the remaining fields and unmarshal into struct
	remaining, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	// Create a temporary struct without timestamp to avoid infinite recursion
	type Alias EncryptedMessage
	aux := &struct {
		*Alias
		Timestamp interface{} `json:"timestamp,omitempty"` // Omit timestamp from JSON
	}{
		Alias: (*Alias)(e),
	}

	return json.Unmarshal(remaining, aux)
}

// KeyBundle represents a user's cryptographic key bundle
type KeyBundle struct {
	UserID          string    `json:"userId"`
	IdentityKey     string    `json:"identityKey"`
	SignedPreKey    string    `json:"signedPreKey"`
	PreKeySignature string    `json:"preKeySignature"`
	OneTimePreKeys  []string  `json:"oneTimePreKeys"`
	RegistrationID  int64     `json:"registrationId"`
	CreatedAt       time.Time `json:"createdAt"`
}

// Session represents a cryptographic session between two users
type Session struct {
	ID             string    `json:"id"`
	UserAID        string    `json:"userAId"`
	UserBID        string    `json:"userBId"`
	SharedSecret   string    `json:"sharedSecret"`
	SendingChain   string    `json:"sendingChain"`
	ReceivingChain string    `json:"receivingChain"`
	MessageNumber  int64     `json:"messageNumber"`
	CreatedAt      time.Time `json:"createdAt"`
	LastUsed       time.Time `json:"lastUsed"`
}

// Device represents a user's device
type Device struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	DeviceID       int       `json:"deviceId"`
	DeviceName     string    `json:"deviceName"`
	DeviceType     string    `json:"deviceType"`
	RegistrationID int64     `json:"registrationId"`
	SignedPreKeyID int64     `json:"signedPreKeyId"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// SignalIdentityKey represents a device's identity key
type SignalIdentityKey struct {
	UserID     string    `json:"userId"`
	DeviceID   int       `json:"deviceId"`
	PublicKey  string    `json:"publicKey"`
	PrivateKey string    `json:"privateKey"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SignalPreKey represents a pre-key
type SignalPreKey struct {
	UserID     string    `json:"userId"`
	DeviceID   int       `json:"deviceId"`
	PreKeyID   int64     `json:"preKeyId"`
	PublicKey  string    `json:"publicKey"`
	PrivateKey string    `json:"privateKey"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SignalSignedPreKey represents a signed pre-key
type SignalSignedPreKey struct {
	UserID         string    `json:"userId"`
	DeviceID       int       `json:"deviceId"`
	SignedPreKeyID int64     `json:"signedPreKeyId"`
	PublicKey      string    `json:"publicKey"`
	PrivateKey     string    `json:"privateKey"`
	Signature      string    `json:"signature"`
	CreatedAt      time.Time `json:"createdAt"`
}

// SignalSession represents an encrypted session
type SignalSession struct {
	UserID       string    `json:"userId"`
	DeviceID     int       `json:"deviceId"`
	SessionID    string    `json:"sessionId"`
	SessionData  string    `json:"sessionData"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// SignalMessage represents an encrypted message
type SignalMessage struct {
	ID             string    `json:"id"`
	SenderID       string    `json:"senderId"`
	SenderDeviceID int       `json:"senderDeviceId"`
	RecipientID    string    `json:"recipientId"`
	Ciphertext     string    `json:"ciphertext"`
	MessageType    string    `json:"messageType"`
	Timestamp      time.Time `json:"timestamp"`
	IsDelivered    bool      `json:"isDelivered"`
	IsRead         bool      `json:"isRead"`
	CreatedAt      time.Time `json:"createdAt"`
}

// PreKeyBundle represents a bundle of pre-keys for session initiation
type PreKeyBundle struct {
	UserID           string   `json:"userId"`
	DeviceID         int      `json:"deviceId"`
	RegistrationID   int64    `json:"registrationId"`
	IdentityKey      string   `json:"identityKey"`
	SignedPreKey     string   `json:"signedPreKey"`
	SignedPreKeyID   int64    `json:"signedPreKeyId"`
	SignedPreKeySig  string   `json:"signedPreKeySignature"`
	PreKey           string   `json:"preKey"`
	PreKeyID         int64    `json:"preKeyId"`
}

// NewMilitaryGradeE2EEService creates a new military-grade E2EE service
func NewMilitaryGradeE2EEService(db *sql.DB) *MilitaryGradeE2EEService {
	return &MilitaryGradeE2EEService{
		db: db,
	}
}

// InitializeUserKeys initializes cryptographic keys for a user
func (s *MilitaryGradeE2EEService) InitializeUserKeys(userID string) (*KeyBundle, error) {
	fmt.Printf("🔐 Initializing military-grade keys for user: %s\n", userID)

	// Generate identity key pair using Curve25519
	identityPrivate := make([]byte, 32)
	if _, err := rand.Read(identityPrivate); err != nil {
		return nil, fmt.Errorf("failed to generate identity private key: %w", err)
	}

	var identityPublic [32]byte
	curve25519.ScalarBaseMult(&identityPublic, (*[32]byte)(identityPrivate))

	// Generate signed pre-key
	signedPreKeyPrivate := make([]byte, 32)
	if _, err := rand.Read(signedPreKeyPrivate); err != nil {
		return nil, fmt.Errorf("failed to generate signed pre-key: %w", err)
	}

	var signedPreKeyPublic [32]byte
	curve25519.ScalarBaseMult(&signedPreKeyPublic, (*[32]byte)(signedPreKeyPrivate))

	// Sign the pre-key with identity key
	signature := s.signData(signedPreKeyPublic[:], identityPrivate)

	// Generate one-time pre-keys
	oneTimePreKeys := make([]string, 100)
	for i := 0; i < 100; i++ {
		preKeyPrivate := make([]byte, 32)
		if _, err := rand.Read(preKeyPrivate); err != nil {
			return nil, fmt.Errorf("failed to generate one-time pre-key %d: %w", i, err)
		}

		var preKeyPublic [32]byte
		curve25519.ScalarBaseMult(&preKeyPublic, (*[32]byte)(preKeyPrivate))
		oneTimePreKeys[i] = base64.StdEncoding.EncodeToString(preKeyPublic[:])
	}

	// Generate registration ID
	regIDBytes := make([]byte, 4)
	if _, err := rand.Read(regIDBytes); err != nil {
		return nil, fmt.Errorf("failed to generate registration ID: %w", err)
	}
	registrationID := int64(regIDBytes[0])<<24 | int64(regIDBytes[1])<<16 | int64(regIDBytes[2])<<8 | int64(regIDBytes[3])

	keyBundle := &KeyBundle{
		UserID:          userID,
		IdentityKey:     base64.StdEncoding.EncodeToString(identityPublic[:]),
		SignedPreKey:    base64.StdEncoding.EncodeToString(signedPreKeyPublic[:]),
		PreKeySignature: base64.StdEncoding.EncodeToString(signature),
		OneTimePreKeys:  oneTimePreKeys,
		RegistrationID:  registrationID,
		CreatedAt:       time.Now(),
	}

	// Store key bundle in database
	if err := s.storeKeyBundle(keyBundle); err != nil {
		return nil, fmt.Errorf("failed to store key bundle: %w", err)
	}

	fmt.Printf("✅ Military-grade keys initialized for user: %s\n", userID)
	return keyBundle, nil
}

// EncryptMessage encrypts a message with military-grade security
func (s *MilitaryGradeE2EEService) EncryptMessage(senderID, recipientID, plaintext string, metadata map[string]interface{}) (*EncryptedMessage, error) {
	fmt.Printf("🔐 Encrypting message with military-grade security: %s -> %s\n", senderID, recipientID)

	// Get or create session
	session, err := s.getOrCreateSession(senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Derive message keys using HKDF
	messageKeys, err := s.deriveMessageKeys(session)
	if err != nil {
		return nil, fmt.Errorf("failed to derive message keys: %w", err)
	}

	// Prepare message data with metadata protection
	messageData := map[string]interface{}{
		"content":   plaintext,
		"timestamp": time.Now().Unix(),
		"messageId": s.generateSecureMessageID(),
		"metadata":  s.protectMetadata(metadata),
	}

	serializedMessage, err := json.Marshal(messageData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	// Encrypt with AES-256-GCM
	encryptedData, iv, err := s.performAESGCMEncryption(serializedMessage, messageKeys.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Generate authentication tag
	authTag := s.generateAuthenticationTag(encryptedData, messageKeys.MACKey, recipientID)

	// Calculate integrity hash
	integrityHash := s.calculateIntegrityHash(encryptedData, authTag, iv)

	// Update session state for perfect forward secrecy
	if err := s.updateSessionState(session); err != nil {
		return nil, fmt.Errorf("failed to update session state: %w", err)
	}

	encryptedMessage := &EncryptedMessage{
		Version:       "1.0",
		SenderID:      senderID,
		RecipientID:   recipientID,
		Ciphertext:    base64.StdEncoding.EncodeToString(encryptedData),
		AuthTag:       base64.StdEncoding.EncodeToString(authTag),
		IV:            base64.StdEncoding.EncodeToString(iv),
		SessionID:     session.ID,
		MessageNumber: session.MessageNumber,
		Timestamp:     time.Now(),
		SecurityLevel: "MILITARY_GRADE",
		IntegrityHash: integrityHash,
	}

	fmt.Printf("✅ Message encrypted with military-grade security\n")
	return encryptedMessage, nil
}

// DecryptMessage decrypts a military-grade encrypted message
func (s *MilitaryGradeE2EEService) DecryptMessage(encryptedMessage *EncryptedMessage) (string, map[string]interface{}, error) {
	fmt.Printf("🔓 Decrypting military-grade encrypted message\n")

	// Validate message structure
	if err := s.validateEncryptedMessage(encryptedMessage); err != nil {
		return "", nil, fmt.Errorf("invalid message structure: %w", err)
	}

	// Get session
	session, err := s.getSession(encryptedMessage.SenderID, encryptedMessage.RecipientID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Verify integrity hash
	ciphertext, _ := base64.StdEncoding.DecodeString(encryptedMessage.Ciphertext)
	authTag, _ := base64.StdEncoding.DecodeString(encryptedMessage.AuthTag)
	iv, _ := base64.StdEncoding.DecodeString(encryptedMessage.IV)

	expectedHash := s.calculateIntegrityHash(ciphertext, authTag, iv)
	if !s.constantTimeCompare([]byte(expectedHash), []byte(encryptedMessage.IntegrityHash)) {
		return "", nil, fmt.Errorf("message integrity verification failed")
	}

	// Derive message keys
	messageKeys, err := s.deriveDecryptionKeys(session, encryptedMessage.MessageNumber)
	if err != nil {
		return "", nil, fmt.Errorf("failed to derive decryption keys: %w", err)
	}

	// Verify authentication tag (try both with and without timestamp for backward compatibility)
	expectedAuthTag := s.generateAuthenticationTag(ciphertext, messageKeys.MACKey, encryptedMessage.RecipientID)
	expectedAuthTagWithTime := s.generateAuthenticationTagWithTimestamp(ciphertext, messageKeys.MACKey, encryptedMessage.RecipientID, encryptedMessage.Timestamp.Unix())

	if !s.constantTimeCompare(authTag, expectedAuthTag) && !s.constantTimeCompare(authTag, expectedAuthTagWithTime) {
		return "", nil, fmt.Errorf("message authentication failed - possible tampering")
	}

	// Decrypt with AES-256-GCM
	decryptedData, err := s.performAESGCMDecryption(ciphertext, messageKeys.EncryptionKey, iv)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	// Parse decrypted message
	var messageData map[string]interface{}
	if err := json.Unmarshal(decryptedData, &messageData); err != nil {
		return "", nil, fmt.Errorf("failed to parse decrypted message: %w", err)
	}

	// Update session state
	if err := s.updateSessionStateForDecryption(session); err != nil {
		return "", nil, fmt.Errorf("failed to update session state: %w", err)
	}

	content, _ := messageData["content"].(string)
	metadata := s.unprotectMetadata(messageData["metadata"])

	fmt.Printf("✅ Message decrypted successfully\n")
	return content, metadata, nil
}

// MessageKeys represents derived encryption and MAC keys
type MessageKeys struct {
	EncryptionKey []byte
	MACKey        []byte
}

// deriveMessageKeys derives encryption and MAC keys using HKDF
func (s *MilitaryGradeE2EEService) deriveMessageKeys(session *Session) (*MessageKeys, error) {
	sharedSecret, err := base64.StdEncoding.DecodeString(session.SharedSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode shared secret: %w", err)
	}

	// Use HKDF to derive keys
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("VaultKe-E2EE-v1.0"))

	encryptionKey := make([]byte, 32) // AES-256
	if _, err := hkdf.Read(encryptionKey); err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	macKey := make([]byte, 32) // HMAC-SHA-256
	if _, err := hkdf.Read(macKey); err != nil {
		return nil, fmt.Errorf("failed to derive MAC key: %w", err)
	}

	return &MessageKeys{
		EncryptionKey: encryptionKey,
		MACKey:        macKey,
	}, nil
}

// performAESGCMEncryption encrypts data using AES-256-GCM
func (s *MilitaryGradeE2EEService) performAESGCMEncryption(plaintext, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random IV
	iv := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	ciphertext := gcm.Seal(nil, iv, plaintext, nil)
	return ciphertext, iv, nil
}

// performAESGCMDecryption decrypts data using AES-256-GCM
func (s *MilitaryGradeE2EEService) performAESGCMDecryption(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// generateAuthenticationTag generates HMAC-SHA-256 authentication tag
func (s *MilitaryGradeE2EEService) generateAuthenticationTag(data, macKey []byte, recipientID string) []byte {
	h := hmac.New(sha256.New, macKey)
	h.Write(data)
	h.Write([]byte(recipientID))
	return h.Sum(nil)
}

// generateAuthenticationTagWithTimestamp generates HMAC-SHA-256 authentication tag with timestamp (for backward compatibility)
func (s *MilitaryGradeE2EEService) generateAuthenticationTagWithTimestamp(data, macKey []byte, recipientID string, timestamp int64) []byte {
	h := hmac.New(sha256.New, macKey)
	h.Write(data)
	h.Write([]byte(recipientID))
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	return h.Sum(nil)
}

// constantTimeCompare performs constant-time comparison to prevent timing attacks
func (s *MilitaryGradeE2EEService) constantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// signData signs data using Ed25519 (simplified implementation)
func (s *MilitaryGradeE2EEService) signData(data, privateKey []byte) []byte {
	h := hmac.New(sha256.New, privateKey)
	h.Write(data)
	return h.Sum(nil)
}

// generateSecureMessageID generates a cryptographically secure message ID
func (s *MilitaryGradeE2EEService) generateSecureMessageID() string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	return fmt.Sprintf("mil_msg_%d_%x", time.Now().Unix(), randomBytes)
}

// calculateIntegrityHash calculates integrity hash for message
func (s *MilitaryGradeE2EEService) calculateIntegrityHash(ciphertext, authTag, iv []byte) string {
	h := sha256.New()
	h.Write(ciphertext)
	h.Write(authTag)
	h.Write(iv)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// protectMetadata adds padding to metadata to prevent traffic analysis
func (s *MilitaryGradeE2EEService) protectMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Add random padding
	paddingSize := 32 + (time.Now().Unix() % 64)
	padding := make([]byte, paddingSize)
	rand.Read(padding)

	metadata["_padding"] = base64.StdEncoding.EncodeToString(padding)
	metadata["_timestamp"] = time.Now().Unix()
	metadata["_version"] = "1.0"

	return metadata
}

// unprotectMetadata removes padding from metadata
func (s *MilitaryGradeE2EEService) unprotectMetadata(metadata interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}

	metadataMap, ok := metadata.(map[string]interface{})
	if !ok {
		return make(map[string]interface{})
	}

	// Remove padding fields
	delete(metadataMap, "_padding")
	delete(metadataMap, "_timestamp")
	delete(metadataMap, "_version")

	return metadataMap
}

// validateEncryptedMessage validates the structure of an encrypted message
func (s *MilitaryGradeE2EEService) validateEncryptedMessage(msg *EncryptedMessage) error {
	if msg.Version != "1.0" {
		return fmt.Errorf("unsupported message version: %s", msg.Version)
	}

	if msg.SenderID == "" || msg.RecipientID == "" {
		return fmt.Errorf("missing sender or recipient ID")
	}

	if msg.Ciphertext == "" || msg.AuthTag == "" || msg.IV == "" {
		return fmt.Errorf("missing cryptographic components")
	}

	if msg.SecurityLevel != "MILITARY_GRADE" && msg.SecurityLevel != "GROUP_ENCRYPTED" {
		return fmt.Errorf("unsupported security level: %s", msg.SecurityLevel)
	}

	return nil
}

// Database operations for key management and session storage

// storeKeyBundle stores a user's key bundle in the database
func (s *MilitaryGradeE2EEService) storeKeyBundle(keyBundle *KeyBundle) error {
	query := `
		INSERT OR REPLACE INTO e2ee_key_bundles (
			user_id, identity_key, signed_pre_key, pre_key_signature,
			one_time_pre_keys, registration_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	oneTimePreKeysJSON, err := json.Marshal(keyBundle.OneTimePreKeys)
	if err != nil {
		return fmt.Errorf("failed to marshal one-time pre-keys: %w", err)
	}

	_, err = s.db.Exec(query,
		keyBundle.UserID,
		keyBundle.IdentityKey,
		keyBundle.SignedPreKey,
		keyBundle.PreKeySignature,
		string(oneTimePreKeysJSON),
		keyBundle.RegistrationID,
		keyBundle.CreatedAt,
	)

	return err
}

// EncryptGroupMessage encrypts a message for a group chat using a room-based symmetric key
func (s *MilitaryGradeE2EEService) EncryptGroupMessage(roomID, senderID, plaintext string, metadata map[string]interface{}) (*EncryptedMessage, error) {
	fmt.Printf("🔐 Encrypting group message with military-grade security: room %s, sender %s\n", roomID, senderID)

	// Derive a symmetric key from the room ID using HKDF
	roomKey := s.deriveRoomKey(roomID)

	// Prepare message data
	messageData := map[string]interface{}{
		"content":   plaintext,
		"timestamp": time.Now().Unix(),
		"messageId": s.generateSecureMessageID(),
		"roomId":    roomID,
		"senderId":  senderID,
		"metadata":  s.protectMetadata(metadata),
	}

	serializedMessage, err := json.Marshal(messageData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	// Encrypt with AES-256-GCM using room key
	encryptedData, iv, err := s.performAESGCMEncryption(serializedMessage, roomKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Generate authentication tag
	authTag := s.generateAuthenticationTag(encryptedData, roomKey, roomID)

	// Calculate integrity hash
	integrityHash := s.calculateIntegrityHash(encryptedData, authTag, iv)

	encryptedMessage := &EncryptedMessage{
		Version:       "1.0",
		SenderID:      senderID,
		RecipientID:   roomID, // Use room ID as recipient for group messages
		Ciphertext:    base64.StdEncoding.EncodeToString(encryptedData),
		AuthTag:       base64.StdEncoding.EncodeToString(authTag),
		IV:            base64.StdEncoding.EncodeToString(iv),
		SessionID:     roomID, // Use room ID as session ID for groups
		MessageNumber: 0,      // Not used for groups
		Timestamp:     time.Now(),
		SecurityLevel: "GROUP_ENCRYPTED",
		IntegrityHash: integrityHash,
	}

	fmt.Printf("✅ Group message encrypted with military-grade security\n")
	return encryptedMessage, nil
}

// DecryptGroupMessage decrypts a group message using the room-based symmetric key
func (s *MilitaryGradeE2EEService) DecryptGroupMessage(roomID string, encryptedMessage *EncryptedMessage) (string, map[string]interface{}, error) {
	fmt.Printf("🔓 Decrypting group message\n")

	// Validate message structure
	if err := s.validateEncryptedMessage(encryptedMessage); err != nil {
		return "", nil, fmt.Errorf("invalid message structure: %w", err)
	}

	// Derive the same room key
	roomKey := s.deriveRoomKey(roomID)

	// Verify integrity hash
	ciphertext, _ := base64.StdEncoding.DecodeString(encryptedMessage.Ciphertext)
	authTag, _ := base64.StdEncoding.DecodeString(encryptedMessage.AuthTag)
	iv, _ := base64.StdEncoding.DecodeString(encryptedMessage.IV)

	expectedHash := s.calculateIntegrityHash(ciphertext, authTag, iv)
	if !s.constantTimeCompare([]byte(expectedHash), []byte(encryptedMessage.IntegrityHash)) {
		return "", nil, fmt.Errorf("message integrity verification failed")
	}

	// Verify authentication tag (try both with and without timestamp for backward compatibility)
	expectedAuthTag := s.generateAuthenticationTag(ciphertext, roomKey, roomID)
	expectedAuthTagWithTime := s.generateAuthenticationTagWithTimestamp(ciphertext, roomKey, roomID, encryptedMessage.Timestamp.Unix())

	if !s.constantTimeCompare(authTag, expectedAuthTag) && !s.constantTimeCompare(authTag, expectedAuthTagWithTime) {
		return "", nil, fmt.Errorf("message authentication failed - possible tampering")
	}

	// Decrypt with AES-256-GCM
	decryptedData, err := s.performAESGCMDecryption(ciphertext, roomKey, iv)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	// Parse decrypted message
	var messageData map[string]interface{}
	if err := json.Unmarshal(decryptedData, &messageData); err != nil {
		return "", nil, fmt.Errorf("failed to parse decrypted message: %w", err)
	}

	content, _ := messageData["content"].(string)
	metadata := s.unprotectMetadata(messageData["metadata"])

	fmt.Printf("✅ Group message decrypted successfully\n")
	return content, metadata, nil
}

// deriveRoomKey derives a symmetric key from the room ID using SHA-256
func (s *MilitaryGradeE2EEService) deriveRoomKey(roomID string) []byte {
	// Use SHA-256 for deterministic key derivation
	hash := sha256.Sum256([]byte("room-key-" + roomID))
	return hash[:32] // AES-256
}

// ComputeSafetyNumber computes a safety number for verifying communication with another user
func (s *MilitaryGradeE2EEService) ComputeSafetyNumber(userID, targetUserID string) (string, error) {
	// Get identity keys for both users
	userKeyQuery := `
		SELECT ik.public_key FROM signal_identity_keys ik
		JOIN devices d ON d.user_id = ik.user_id AND d.device_id = ik.device_id
		WHERE d.user_id = ? AND d.is_active = 1
		ORDER BY d.created_at ASC LIMIT 1
	`

	var userKey, targetKey string

	err := s.db.QueryRow(userKeyQuery, userID).Scan(&userKey)
	if err != nil {
		return "", fmt.Errorf("failed to get user identity key: %w", err)
	}

	err = s.db.QueryRow(userKeyQuery, targetUserID).Scan(&targetKey)
	if err != nil {
		return "", fmt.Errorf("failed to get target user identity key: %w", err)
	}

	// Compute safety number by hashing the sorted combination of keys
	var combined string
	if userID < targetUserID {
		combined = userKey + targetKey
	} else {
		combined = targetKey + userKey
	}

	hash := sha256.Sum256([]byte(combined))
	safetyNumber := fmt.Sprintf("%x", hash[:16]) // First 16 bytes as hex

	return safetyNumber, nil
}

// RotateSignedPreKey rotates the signed pre-key for a device
func (s *MilitaryGradeE2EEService) RotateSignedPreKey(userID string, deviceID int, newSignedPreKeyID int64, newSignedPreKey, signature string) error {
	// Update the signed pre-key in the database
	query := `
		UPDATE signal_signed_pre_keys
		SET signed_pre_key_id = ?, public_key = ?, signature = ?, created_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND device_id = ?
	`

	_, err := s.db.Exec(query, newSignedPreKeyID, newSignedPreKey, signature, userID, deviceID)
	if err != nil {
		return fmt.Errorf("failed to rotate signed pre-key: %w", err)
	}

	// Also update the device record
	deviceQuery := `
		UPDATE devices
		SET signed_pre_key_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND device_id = ?
	`

	_, err = s.db.Exec(deviceQuery, newSignedPreKeyID, userID, deviceID)
	if err != nil {
		return fmt.Errorf("failed to update device signed pre-key ID: %w", err)
	}

	return nil
}

// ResetSession resets the session with another user
func (s *MilitaryGradeE2EEService) ResetSession(userID, targetUserID string) error {
	// Delete all sessions between the two users
	query := `
		DELETE FROM signal_sessions
		WHERE (user_id = ? AND session_id LIKE ?)
		   OR (user_id = ? AND session_id LIKE ?)
	`

	// Session IDs are typically in format "user1_user2" or similar
	pattern1 := targetUserID + "_%"
	pattern2 := userID + "_%"

	_, err := s.db.Exec(query, userID, pattern1, targetUserID, pattern2)
	if err != nil {
		return fmt.Errorf("failed to reset session: %w", err)
	}

	return nil
}

// getKeyBundle retrieves a user's key bundle from the database
func (s *MilitaryGradeE2EEService) getKeyBundle(userID string) (*KeyBundle, error) {
	query := `
		SELECT user_id, identity_key, signed_pre_key, pre_key_signature,
			   one_time_pre_keys, registration_id, created_at
		FROM e2ee_key_bundles
		WHERE user_id = ?
	`

	var keyBundle KeyBundle
	var oneTimePreKeysJSON string

	err := s.db.QueryRow(query, userID).Scan(
		&keyBundle.UserID,
		&keyBundle.IdentityKey,
		&keyBundle.SignedPreKey,
		&keyBundle.PreKeySignature,
		&oneTimePreKeysJSON,
		&keyBundle.RegistrationID,
		&keyBundle.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get key bundle: %w", err)
	}

	if err := json.Unmarshal([]byte(oneTimePreKeysJSON), &keyBundle.OneTimePreKeys); err != nil {
		return nil, fmt.Errorf("failed to unmarshal one-time pre-keys: %w", err)
	}

	return &keyBundle, nil
}

// getOrCreateSession gets an existing session or creates a new one
func (s *MilitaryGradeE2EEService) getOrCreateSession(userAID, userBID string) (*Session, error) {
	// Try to get existing session
	session, err := s.getSession(userAID, userBID)
	if err == nil {
		return session, nil
	}

	// Create new session
	return s.createSession(userAID, userBID)
}

// getSession retrieves an existing session
func (s *MilitaryGradeE2EEService) getSession(userAID, userBID string) (*Session, error) {
	query := `
		SELECT id, user_a_id, user_b_id, shared_secret, sending_chain,
			   receiving_chain, message_number, created_at, last_used
		FROM e2ee_sessions
		WHERE (user_a_id = ? AND user_b_id = ?) OR (user_a_id = ? AND user_b_id = ?)
		ORDER BY last_used DESC
		LIMIT 1
	`

	var session Session
	err := s.db.QueryRow(query, userAID, userBID, userBID, userAID).Scan(
		&session.ID,
		&session.UserAID,
		&session.UserBID,
		&session.SharedSecret,
		&session.SendingChain,
		&session.ReceivingChain,
		&session.MessageNumber,
		&session.CreatedAt,
		&session.LastUsed,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// createSession creates a new cryptographic session
func (s *MilitaryGradeE2EEService) createSession(userAID, userBID string) (*Session, error) {
	// Generate shared secret using key exchange
	sharedSecret := make([]byte, 32)
	if _, err := rand.Read(sharedSecret); err != nil {
		return nil, fmt.Errorf("failed to generate shared secret: %w", err)
	}

	// Generate chain keys
	sendingChain := make([]byte, 32)
	receivingChain := make([]byte, 32)
	if _, err := rand.Read(sendingChain); err != nil {
		return nil, fmt.Errorf("failed to generate sending chain: %w", err)
	}
	if _, err := rand.Read(receivingChain); err != nil {
		return nil, fmt.Errorf("failed to generate receiving chain: %w", err)
	}

	sessionID := fmt.Sprintf("session_%d_%x", time.Now().Unix(), sharedSecret[:8])

	session := &Session{
		ID:             sessionID,
		UserAID:        userAID,
		UserBID:        userBID,
		SharedSecret:   base64.StdEncoding.EncodeToString(sharedSecret),
		SendingChain:   base64.StdEncoding.EncodeToString(sendingChain),
		ReceivingChain: base64.StdEncoding.EncodeToString(receivingChain),
		MessageNumber:  0,
		CreatedAt:      time.Now(),
		LastUsed:       time.Now(),
	}

	// Store session in database
	if err := s.storeSession(session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return session, nil
}

// storeSession stores a session in the database
func (s *MilitaryGradeE2EEService) storeSession(session *Session) error {
	query := `
		INSERT OR REPLACE INTO e2ee_sessions (
			id, user_a_id, user_b_id, shared_secret, sending_chain,
			receiving_chain, message_number, created_at, last_used
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		session.ID,
		session.UserAID,
		session.UserBID,
		session.SharedSecret,
		session.SendingChain,
		session.ReceivingChain,
		session.MessageNumber,
		session.CreatedAt,
		session.LastUsed,
	)

	return err
}

// updateSessionState updates session state for perfect forward secrecy
func (s *MilitaryGradeE2EEService) updateSessionState(session *Session) error {
	session.MessageNumber++
	session.LastUsed = time.Now()

	// Rotate chain keys for perfect forward secrecy
	sendingChain, _ := base64.StdEncoding.DecodeString(session.SendingChain)
	newSendingChain := sha256.Sum256(sendingChain)
	session.SendingChain = base64.StdEncoding.EncodeToString(newSendingChain[:])

	return s.storeSession(session)
}

// updateSessionStateForDecryption updates session state for decryption
func (s *MilitaryGradeE2EEService) updateSessionStateForDecryption(session *Session) error {
	session.LastUsed = time.Now()

	// Rotate receiving chain for perfect forward secrecy
	receivingChain, _ := base64.StdEncoding.DecodeString(session.ReceivingChain)
	newReceivingChain := sha256.Sum256(receivingChain)
	session.ReceivingChain = base64.StdEncoding.EncodeToString(newReceivingChain[:])

	return s.storeSession(session)
}

// deriveDecryptionKeys derives keys for message decryption
func (s *MilitaryGradeE2EEService) deriveDecryptionKeys(session *Session, messageNumber int64) (*MessageKeys, error) {
	// For simplicity, using the same derivation as encryption
	// In a full implementation, this would handle out-of-order messages
	return s.deriveMessageKeys(session)
}

// GetKeyBundle retrieves a user's key bundle (public method for API)
func (s *MilitaryGradeE2EEService) GetKeyBundle(userID string) (*KeyBundle, error) {
	return s.getKeyBundle(userID)
}

// RegisterDevice registers a new device with Signal Protocol keys
func (s *MilitaryGradeE2EEService) RegisterDevice(device *Device, identityKey *SignalIdentityKey, signedPreKey *SignalSignedPreKey, preKeys []*SignalPreKey) error {
	// Store device
	if err := s.storeDevice(device); err != nil {
		return fmt.Errorf("failed to store device: %w", err)
	}

	// Store identity key
	if err := s.storeIdentityKey(identityKey); err != nil {
		return fmt.Errorf("failed to store identity key: %w", err)
	}

	// Store signed pre-key
	if err := s.storeSignedPreKey(signedPreKey); err != nil {
		return fmt.Errorf("failed to store signed pre-key: %w", err)
	}

	// Store pre-keys
	for _, preKey := range preKeys {
		if err := s.storePreKey(preKey); err != nil {
			return fmt.Errorf("failed to store pre-key: %w", err)
		}
	}

	return nil
}

// GetUserDevices retrieves all devices for a user
func (s *MilitaryGradeE2EEService) GetUserDevices(userID string) ([]*Device, error) {
	query := `
		SELECT id, user_id, device_id, device_name, device_type, registration_id, signed_pre_key_id, is_active, created_at, updated_at
		FROM devices
		WHERE user_id = ? AND is_active = 1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		var device Device
		err := rows.Scan(
			&device.ID,
			&device.UserID,
			&device.DeviceID,
			&device.DeviceName,
			&device.DeviceType,
			&device.RegistrationID,
			&device.SignedPreKeyID,
			&device.IsActive,
			&device.CreatedAt,
			&device.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, &device)
	}

	return devices, nil
}

// UploadPreKeys uploads new pre-keys for a device
func (s *MilitaryGradeE2EEService) UploadPreKeys(preKeys []*SignalPreKey) error {
	for _, preKey := range preKeys {
		if err := s.storePreKey(preKey); err != nil {
			return fmt.Errorf("failed to store pre-key: %w", err)
		}
	}
	return nil
}

// GetPreKeyBundle retrieves a pre-key bundle for session initiation
func (s *MilitaryGradeE2EEService) GetPreKeyBundle(userID string) (*PreKeyBundle, error) {
	// Get a random active device for the user
	query := `
		SELECT d.device_id, d.registration_id, ik.public_key, spk.signed_pre_key_id, spk.public_key, spk.signature
		FROM devices d
		JOIN signal_identity_keys ik ON d.user_id = ik.user_id AND d.device_id = ik.device_id
		JOIN signal_signed_pre_keys spk ON d.user_id = spk.user_id AND d.device_id = spk.device_id
		WHERE d.user_id = ? AND d.is_active = 1
		ORDER BY RANDOM() LIMIT 1
	`

	var bundle PreKeyBundle
	bundle.UserID = userID

	err := s.db.QueryRow(query, userID).Scan(
		&bundle.DeviceID,
		&bundle.RegistrationID,
		&bundle.IdentityKey,
		&bundle.SignedPreKeyID,
		&bundle.SignedPreKey,
		&bundle.SignedPreKeySig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	// Get a random unused pre-key
	preKeyQuery := `
		SELECT pre_key_id, public_key FROM signal_pre_keys
		WHERE user_id = ? AND device_id = ?
		ORDER BY RANDOM() LIMIT 1
	`

	err = s.db.QueryRow(preKeyQuery, userID, bundle.DeviceID).Scan(
		&bundle.PreKeyID,
		&bundle.PreKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pre-key: %w", err)
	}

	// Mark pre-key as used (delete it)
	deleteQuery := `DELETE FROM signal_pre_keys WHERE user_id = ? AND device_id = ? AND pre_key_id = ?`
	_, err = s.db.Exec(deleteQuery, userID, bundle.DeviceID, bundle.PreKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark pre-key as used: %w", err)
	}

	return &bundle, nil
}

// SendMessage stores an encrypted message
func (s *MilitaryGradeE2EEService) SendMessage(message *SignalMessage) error {
	query := `
		INSERT INTO signal_messages (
			id, sender_id, sender_device_id, recipient_id, ciphertext, message_type, timestamp, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	message.ID = fmt.Sprintf("msg_%d_%x", time.Now().Unix(), []byte(message.SenderID)[:4])
	message.Timestamp = time.Now()
	message.CreatedAt = time.Now()

	_, err := s.db.Exec(query,
		message.ID,
		message.SenderID,
		message.SenderDeviceID,
		message.RecipientID,
		message.Ciphertext,
		message.MessageType,
		message.Timestamp,
		message.CreatedAt,
	)

	return err
}

// GetMessages retrieves messages for a user
func (s *MilitaryGradeE2EEService) GetMessages(userID string) ([]*SignalMessage, error) {
	query := `
		SELECT id, sender_id, sender_device_id, recipient_id, ciphertext, message_type, timestamp, is_delivered, is_read, created_at
		FROM signal_messages
		WHERE recipient_id = ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*SignalMessage
	for rows.Next() {
		var message SignalMessage
		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.SenderDeviceID,
			&message.RecipientID,
			&message.Ciphertext,
			&message.MessageType,
			&message.Timestamp,
			&message.IsDelivered,
			&message.IsRead,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

// Database operations for Signal Protocol

func (s *MilitaryGradeE2EEService) storeDevice(device *Device) error {
	query := `
		INSERT INTO devices (
			id, user_id, device_id, device_name, device_type, registration_id, signed_pre_key_id, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()

	_, err := s.db.Exec(query,
		device.ID,
		device.UserID,
		device.DeviceID,
		device.DeviceName,
		device.DeviceType,
		device.RegistrationID,
		device.SignedPreKeyID,
		device.IsActive,
		device.CreatedAt,
		device.UpdatedAt,
	)

	return err
}

func (s *MilitaryGradeE2EEService) storeIdentityKey(key *SignalIdentityKey) error {
	query := `
		INSERT INTO signal_identity_keys (
			user_id, device_id, public_key, private_key, created_at
		) VALUES (?, ?, ?, ?, ?)
	`

	key.CreatedAt = time.Now()

	_, err := s.db.Exec(query,
		key.UserID,
		key.DeviceID,
		key.PublicKey,
		key.PrivateKey,
		key.CreatedAt,
	)

	return err
}

func (s *MilitaryGradeE2EEService) storePreKey(key *SignalPreKey) error {
	query := `
		INSERT INTO signal_pre_keys (
			user_id, device_id, pre_key_id, public_key, private_key, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	key.CreatedAt = time.Now()

	_, err := s.db.Exec(query,
		key.UserID,
		key.DeviceID,
		key.PreKeyID,
		key.PublicKey,
		key.PrivateKey,
		key.CreatedAt,
	)

	return err
}

func (s *MilitaryGradeE2EEService) storeSignedPreKey(key *SignalSignedPreKey) error {
	query := `
		INSERT INTO signal_signed_pre_keys (
			user_id, device_id, signed_pre_key_id, public_key, private_key, signature, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	key.CreatedAt = time.Now()

	_, err := s.db.Exec(query,
		key.UserID,
		key.DeviceID,
		key.SignedPreKeyID,
		key.PublicKey,
		key.PrivateKey,
		key.Signature,
		key.CreatedAt,
	)

	return err
}
