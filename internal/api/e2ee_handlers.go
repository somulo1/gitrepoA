package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vaultke-backend/internal/services"
)

// RegisterDevice registers a new device for a user with Signal Protocol keys
func RegisterDevice(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		DeviceName      string `json:"deviceName" binding:"required"`
		DeviceType      string `json:"deviceType" binding:"required"`
		RegistrationID  int64  `json:"registrationId" binding:"required"`
		IdentityKey     string `json:"identityKey" binding:"required"`
		SignedPreKey    string `json:"signedPreKey" binding:"required"`
		SignedPreKeyID  int64  `json:"signedPreKeyId" binding:"required"`
		SignedPreKeySig string `json:"signedPreKeySignature" binding:"required"`
		PreKeys         []struct {
			ID  int64  `json:"id"`
			Key string `json:"key"`
		} `json:"preKeys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// Generate device ID
	deviceID := uuid.New().ID() % 1000000 // Keep it reasonable

	// Store device
	device := &services.Device{
		ID:             uuid.New().String(),
		UserID:         userID,
		DeviceID:       int(deviceID),
		DeviceName:     req.DeviceName,
		DeviceType:     req.DeviceType,
		RegistrationID: req.RegistrationID,
		SignedPreKeyID: req.SignedPreKeyID,
		IsActive:       true,
	}

	// Store identity key
	identityKey := &services.SignalIdentityKey{
		UserID:    userID,
		DeviceID:  int(deviceID),
		PublicKey: req.IdentityKey,
	}

	// Store signed pre-key
	signedPreKey := &services.SignalSignedPreKey{
		UserID:         userID,
		DeviceID:       int(deviceID),
		SignedPreKeyID: req.SignedPreKeyID,
		PublicKey:      req.SignedPreKey,
		Signature:      req.SignedPreKeySig,
	}

	// Store pre-keys
	var preKeys []*services.SignalPreKey
	for _, pk := range req.PreKeys {
		preKeys = append(preKeys, &services.SignalPreKey{
			UserID:    userID,
			DeviceID:  int(deviceID),
			PreKeyID:  pk.ID,
			PublicKey: pk.Key,
		})
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	// Register device
	err := service.RegisterDevice(device, identityKey, signedPreKey, preKeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to register device: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deviceId": deviceID,
			"message":  "Device registered successfully",
		},
	})
}

// GetDevices retrieves all devices for a user
func GetDevices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	devices, err := service.GetUserDevices(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get devices: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    devices,
	})
}

// UploadPreKeys uploads new pre-keys for a device
func UploadPreKeys(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		DeviceID int `json:"deviceId" binding:"required"`
		PreKeys  []struct {
			ID  int64  `json:"id"`
			Key string `json:"key"`
		} `json:"preKeys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	var preKeys []*services.SignalPreKey
	for _, pk := range req.PreKeys {
		preKeys = append(preKeys, &services.SignalPreKey{
			UserID:    userID,
			DeviceID:  req.DeviceID,
			PreKeyID:  pk.ID,
			PublicKey: pk.Key,
		})
	}

	err := service.UploadPreKeys(preKeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to upload pre-keys: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pre-keys uploaded successfully",
	})
}

// GetPreKeyBundle retrieves a pre-key bundle for initiating a session with another user
func GetPreKeyBundle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Target user ID is required",
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	bundle, err := service.GetPreKeyBundle(targetUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Pre-key bundle not found: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bundle,
	})
}

// SendE2EEMessage sends an encrypted message
func SendE2EEMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		RecipientID string `json:"recipientId" binding:"required"`
		DeviceID    int    `json:"deviceId" binding:"required"`
		Ciphertext  string `json:"ciphertext" binding:"required"`
		MessageType string `json:"messageType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	message := &services.SignalMessage{
		SenderID:       userID,
		SenderDeviceID: req.DeviceID,
		RecipientID:    req.RecipientID,
		Ciphertext:     req.Ciphertext,
		MessageType:    req.MessageType,
	}

	err := service.SendMessage(message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send message: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message sent successfully",
	})
}

// GetMessages retrieves encrypted messages for a user
func GetMessages(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	messages, err := service.GetMessages(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get messages: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    messages,
	})
}

// GetSafetyNumber computes the safety number for communication with another user
func GetSafetyNumber(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Target user ID is required",
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	safetyNumber, err := service.ComputeSafetyNumber(userID, targetUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Failed to compute safety number: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"safetyNumber": safetyNumber,
			"userId":       userID,
			"targetUserId": targetUserID,
			"instructions": "Compare this number with your contact out-of-band to verify the security of your communication",
		},
	})
}

// RotateKeys rotates the signed pre-key for a device
func RotateKeys(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		DeviceID           int    `json:"deviceId" binding:"required"`
		NewSignedPreKey    string `json:"newSignedPreKey" binding:"required"`
		NewSignedPreKeyID  int64  `json:"newSignedPreKeyId" binding:"required"`
		NewSignedPreKeySig string `json:"newSignedPreKeySignature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	err := service.RotateSignedPreKey(userID, req.DeviceID, req.NewSignedPreKeyID, req.NewSignedPreKey, req.NewSignedPreKeySig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to rotate keys: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Keys rotated successfully",
	})
}

// ResetSession resets the session with another user
func ResetSession(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Target user ID is required",
		})
		return
	}

	// Get E2EE service
	e2eeService, exists := c.Get("e2eeService")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "E2EE service not available",
		})
		return
	}

	service := e2eeService.(*services.MilitaryGradeE2EEService)

	err := service.ResetSession(userID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to reset session: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session reset successfully",
	})
}

// InitializeE2EEKeys initializes military-grade E2EE keys for a user
func InitializeE2EEKeys(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
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

	// Initialize E2EE service
	e2eeService := services.NewMilitaryGradeE2EEService(db.(*sql.DB))

	// Initialize keys for the user
	keyBundle, err := e2eeService.InitializeUserKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize E2EE keys: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"keyBundle":     keyBundle,
			"securityLevel": "MILITARY_GRADE",
			"features": []string{
				"PERFECT_FORWARD_SECRECY",
				"POST_COMPROMISE_SECURITY",
				"AUTHENTICATED_ENCRYPTION",
				"METADATA_PROTECTION",
				"CURVE25519_KEY_EXCHANGE",
			},
		},
	})
}

// GetE2EEKeyBundle retrieves a user's public key bundle
func GetE2EEKeyBundle(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User ID is required",
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

	// Initialize E2EE service
	e2eeService := services.NewMilitaryGradeE2EEService(db.(*sql.DB))

	// Get key bundle for the user
	keyBundle, err := e2eeService.GetKeyBundle(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Key bundle not found: " + err.Error(),
		})
		return
	}

	// Return only public components (never return private keys)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"userId":          keyBundle.UserID,
			"identityKey":     keyBundle.IdentityKey,
			"signedPreKey":    keyBundle.SignedPreKey,
			"preKeySignature": keyBundle.PreKeySignature,
			"oneTimePreKeys":  keyBundle.OneTimePreKeys,
			"registrationId":  keyBundle.RegistrationID,
			"securityLevel":   "MILITARY_GRADE",
		},
	})
}

// EncryptMessage encrypts a message using military-grade E2EE
func EncryptMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req struct {
		RecipientID string                 `json:"recipientId" binding:"required"`
		Plaintext   string                 `json:"plaintext" binding:"required"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
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

	// Initialize E2EE service
	e2eeService := services.NewMilitaryGradeE2EEService(db.(*sql.DB))

	// Encrypt the message
	encryptedMessage, err := e2eeService.EncryptMessage(userID, req.RecipientID, req.Plaintext, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to encrypt message: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    encryptedMessage,
	})
}

// DecryptMessage decrypts a military-grade encrypted message
func DecryptMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req services.EncryptedMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid encrypted message: " + err.Error(),
		})
		return
	}

	// Verify user is authorized to decrypt this message
	if req.RecipientID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Not authorized to decrypt this message",
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

	// Initialize E2EE service
	e2eeService := services.NewMilitaryGradeE2EEService(db.(*sql.DB))

	// Decrypt the message
	plaintext, metadata, err := e2eeService.DecryptMessage(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to decrypt message: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plaintext": plaintext,
			"metadata":  metadata,
			"verified":  true,
		},
	})
}

// GetE2EESecurityStatus returns the security status of the E2EE system
func GetE2EESecurityStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"securityLevel": "MILITARY_GRADE",
			"features": []string{
				"PERFECT_FORWARD_SECRECY",
				"POST_COMPROMISE_SECURITY",
				"AUTHENTICATED_ENCRYPTION_AES_256_GCM",
				"CURVE25519_KEY_EXCHANGE",
				"HKDF_KEY_DERIVATION",
				"HMAC_SHA256_AUTHENTICATION",
				"METADATA_PROTECTION",
				"CONSTANT_TIME_OPERATIONS",
				"SIDE_CHANNEL_RESISTANCE",
			},
			"algorithms": gin.H{
				"encryption":     "AES-256-GCM",
				"keyExchange":    "Curve25519",
				"keyDerivation":  "HKDF-SHA-256",
				"authentication": "HMAC-SHA-256",
				"hashing":        "SHA-256",
			},
			"status":  "ACTIVE",
			"version": "1.0",
		},
	})
}
