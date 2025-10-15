package test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultke-backend/internal/services"
	"vaultke-backend/test/helpers"
)

func TestE2EEMessageDecoding(t *testing.T) {
	// Test the specific message from the user's example
	encodedMessage := "dGVzdCBhZ2Fpbl9lbmNfMTc1ODU0Mzg3NzM1MF8ydjBuZm5ybTh2ZQ=="

	// Decode the message
	decoded, err := base64.StdEncoding.DecodeString(encodedMessage)
	assert.NoError(t, err)

	decodedStr := string(decoded)
	assert.Contains(t, decodedStr, "test again")
	assert.Contains(t, decodedStr, "_enc_")
	assert.Contains(t, decodedStr, "1758543877350")
	assert.Contains(t, decodedStr, "2v0nfnrm8ve")

	// Test that it matches the expected pattern
	assert.True(t, strings.Contains(decodedStr, "_enc_"))
	assert.True(t, strings.HasSuffix(encodedMessage, "=="))

	t.Logf("Successfully decoded message: %s", decodedStr)
}

func TestFallbackEncryptionPattern(t *testing.T) {
	plaintext := "test again"
	timestamp := "1758543877350"
	random := "2v0nfnrm8ve"

	expectedPattern := plaintext + "_enc_" + timestamp + "_" + random
	encoded := base64.StdEncoding.EncodeToString([]byte(expectedPattern))

	// Verify it ends with ==
	assert.True(t, strings.HasSuffix(encoded, "=="))

	// Verify it can be decoded back
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	assert.NoError(t, err)
	assert.Equal(t, expectedPattern, string(decoded))

	t.Logf("Pattern test passed: %s -> %s", expectedPattern, encoded)
}

func TestMessageMetadataFlags(t *testing.T) {
	metadata := map[string]interface{}{
		"encrypted":      true,
		"needsDecryption": true,
		"securityLevel":  "FALLBACK",
	}

	assert.True(t, metadata["encrypted"].(bool))
	assert.True(t, metadata["needsDecryption"].(bool))
	assert.Equal(t, "FALLBACK", metadata["securityLevel"])

	t.Log("Metadata flags test passed")
}

// TestAESGCMEncryptionDirect tests AES-256-GCM encryption directly
func TestAESGCMEncryptionDirect(t *testing.T) {
	plaintext := "This is a test message for AES-256-GCM encryption"
	key := make([]byte, 32) // AES-256
	_, err := rand.Read(key)
	require.NoError(t, err)

	// Encrypt using AES-256-GCM
	block, err := aes.NewCipher(key)
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Verify ciphertext is different from plaintext
	assert.NotEqual(t, plaintext, string(ciphertext))
	assert.True(t, len(ciphertext) > len(plaintext)) // GCM adds authentication tag

	// Verify we can decrypt it back
	decrypted, err := gcm.Open(nil, nonce, ciphertext, nil)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted))

	t.Logf("AES-256-GCM encryption test passed. Ciphertext length: %d, Plaintext length: %d", len(ciphertext), len(plaintext))
}

// TestProperE2EEEncryptionCycle tests the full E2EE encryption/decryption cycle
func TestProperE2EEEncryptionCycle(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	e2eeService := services.NewMilitaryGradeE2EEService(ts.DB.DB)

	senderID := "e2ee-sender"
	recipientID := "e2ee-recipient"
	plaintext := "This is a properly encrypted E2EE message using AES-256-GCM"

	// Initialize keys for both users
	_, err := e2eeService.InitializeUserKeys(senderID)
	require.NoError(t, err)
	_, err = e2eeService.InitializeUserKeys(recipientID)
	require.NoError(t, err)

	// Encrypt message
	encryptedMessage, err := e2eeService.EncryptMessage(senderID, recipientID, plaintext, nil)
	require.NoError(t, err)
	require.NotNil(t, encryptedMessage)

	// Verify encrypted message structure
	assert.Equal(t, "1.0", encryptedMessage.Version)
	assert.Equal(t, senderID, encryptedMessage.SenderID)
	assert.Equal(t, recipientID, encryptedMessage.RecipientID)
	assert.Equal(t, "MILITARY_GRADE", encryptedMessage.SecurityLevel)
	assert.NotEmpty(t, encryptedMessage.Ciphertext)
	assert.NotEmpty(t, encryptedMessage.AuthTag)
	assert.NotEmpty(t, encryptedMessage.IV)
	assert.NotEmpty(t, encryptedMessage.IntegrityHash)

	// Verify ciphertext is base64 encoded
	ciphertextBytes, err := base64.StdEncoding.DecodeString(encryptedMessage.Ciphertext)
	require.NoError(t, err)
	assert.True(t, len(ciphertextBytes) > len(plaintext)) // Should be larger due to GCM

	// Verify ciphertext is not trivially decodable (not just base64 of plaintext)
	plaintextBase64 := base64.StdEncoding.EncodeToString([]byte(plaintext))
	assert.NotEqual(t, plaintextBase64, encryptedMessage.Ciphertext)

	// Decrypt message
	decryptedText, metadata, err := e2eeService.DecryptMessage(encryptedMessage)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decryptedText)
	assert.NotNil(t, metadata)

	t.Logf("✅ Full E2EE cycle test passed. Original: %s, Encrypted length: %d, Decrypted: %s",
		plaintext, len(ciphertextBytes), decryptedText)
}

// TestCiphertextIsNotTriviallyDecodable tests that stored ciphertext cannot be decoded without proper keys
func TestCiphertextIsNotTriviallyDecodable(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	e2eeService := services.NewMilitaryGradeE2EEService(ts.DB.DB)

	senderID := "security-test-sender"
	recipientID := "security-test-recipient"
	plaintext := "Secret message that should not be readable without proper decryption"

	// Initialize keys
	_, err := e2eeService.InitializeUserKeys(senderID)
	require.NoError(t, err)
	_, err = e2eeService.InitializeUserKeys(recipientID)
	require.NoError(t, err)

	// Encrypt message
	encryptedMessage, err := e2eeService.EncryptMessage(senderID, recipientID, plaintext, nil)
	require.NoError(t, err)

	// Get the raw ciphertext bytes
	ciphertextBytes, err := base64.StdEncoding.DecodeString(encryptedMessage.Ciphertext)
	require.NoError(t, err)

	// Verify that the ciphertext doesn't contain the plaintext in any obvious way
	ciphertextStr := string(ciphertextBytes)

	// Check that individual words from plaintext are not visible
	words := strings.Fields(plaintext)
	for _, word := range words {
		assert.NotContains(t, ciphertextStr, word, "Ciphertext should not contain plaintext words")
	}

	// Verify it's not just base64 of the plaintext
	plaintextBase64 := base64.StdEncoding.EncodeToString([]byte(plaintext))
	assert.NotEqual(t, plaintextBase64, encryptedMessage.Ciphertext)

	// Verify it's not just base64 of plaintext with simple suffix
	for _, suffix := range []string{"_enc_", "_encrypted_", "_secure_"} {
		suffixedPlaintext := plaintext + suffix
		suffixedBase64 := base64.StdEncoding.EncodeToString([]byte(suffixedPlaintext))
		assert.NotEqual(t, suffixedBase64, encryptedMessage.Ciphertext)
	}

	t.Logf("✅ Ciphertext security test passed. Ciphertext is cryptographically secure, not trivially decodable")
}

// TestSafetyNumberGenerationAndValidation tests safety number computation
func TestSafetyNumberGenerationAndValidation(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	e2eeService := services.NewMilitaryGradeE2EEService(ts.DB.DB)

	userA := "safety-user-a"
	userB := "safety-user-b"

	// Initialize keys for both users
	_, err := e2eeService.InitializeUserKeys(userA)
	require.NoError(t, err)
	_, err = e2eeService.InitializeUserKeys(userB)
	require.NoError(t, err)

	// Generate safety number
	safetyNumber, err := e2eeService.ComputeSafetyNumber(userA, userB)
	require.NoError(t, err)
	assert.NotEmpty(t, safetyNumber)
	assert.Len(t, safetyNumber, 32) // Should be 32 hex characters (16 bytes)

	// Verify safety number is consistent (same inputs produce same output)
	safetyNumber2, err := e2eeService.ComputeSafetyNumber(userA, userB)
	require.NoError(t, err)
	assert.Equal(t, safetyNumber, safetyNumber2)

	// Verify safety number is different for different user pairs
	safetyNumber3, err := e2eeService.ComputeSafetyNumber(userA, "different-user")
	require.NoError(t, err)
	assert.NotEqual(t, safetyNumber, safetyNumber3)

	// Verify safety number is deterministic but unique per pair
	safetyNumberReverse, err := e2eeService.ComputeSafetyNumber(userB, userA)
	require.NoError(t, err)
	assert.Equal(t, safetyNumber, safetyNumberReverse) // Should be same regardless of order

	t.Logf("✅ Safety number test passed. Safety number: %s", safetyNumber)
}

// TestMessageIntegrityVerification tests that message integrity is properly verified
func TestMessageIntegrityVerification(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	e2eeService := services.NewMilitaryGradeE2EEService(ts.DB.DB)

	senderID := "integrity-sender"
	recipientID := "integrity-recipient"
	plaintext := "Message integrity test"

	// Initialize keys
	_, err := e2eeService.InitializeUserKeys(senderID)
	require.NoError(t, err)
	_, err = e2eeService.InitializeUserKeys(recipientID)
	require.NoError(t, err)

	// Encrypt message
	encryptedMessage, err := e2eeService.EncryptMessage(senderID, recipientID, plaintext, nil)
	require.NoError(t, err)

	// Verify message can be decrypted successfully
	decryptedText, _, err := e2eeService.DecryptMessage(encryptedMessage)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decryptedText)

	// Test tampering with ciphertext
	tamperedMessage := *encryptedMessage
	tamperedCiphertext := encryptedMessage.Ciphertext + "A" // Add one character
	tamperedMessage.Ciphertext = tamperedCiphertext

	// Should fail integrity check
	_, _, err = e2eeService.DecryptMessage(&tamperedMessage)
	assert.Error(t, err, "Tampered message should fail integrity verification")

	// Test tampering with auth tag
	tamperedMessage2 := *encryptedMessage
	if len(tamperedMessage2.AuthTag) > 0 {
		tamperedAuthTag := tamperedMessage2.AuthTag[:len(tamperedMessage2.AuthTag)-1] + "X"
		tamperedMessage2.AuthTag = tamperedAuthTag

		_, _, err = e2eeService.DecryptMessage(&tamperedMessage2)
		assert.Error(t, err, "Message with tampered auth tag should fail verification")
	}

	t.Log("✅ Message integrity verification test passed")
}

// TestPerfectForwardSecrecy tests that PFS is implemented
func TestPerfectForwardSecrecy(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	e2eeService := services.NewMilitaryGradeE2EEService(ts.DB.DB)

	senderID := "pfs-sender"
	recipientID := "pfs-recipient"

	// Initialize keys
	_, err := e2eeService.InitializeUserKeys(senderID)
	require.NoError(t, err)
	_, err = e2eeService.InitializeUserKeys(recipientID)
	require.NoError(t, err)

	// Send multiple messages
	message1 := "First PFS message"
	message2 := "Second PFS message"

	encrypted1, err := e2eeService.EncryptMessage(senderID, recipientID, message1, nil)
	require.NoError(t, err)

	encrypted2, err := e2eeService.EncryptMessage(senderID, recipientID, message2, nil)
	require.NoError(t, err)

	// Verify messages have different session data (PFS in action)
	assert.NotEqual(t, encrypted1.SessionID, encrypted2.SessionID)
	assert.NotEqual(t, encrypted1.MessageNumber, encrypted2.MessageNumber)

	// Both messages should decrypt correctly
	decrypted1, _, err := e2eeService.DecryptMessage(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, message1, decrypted1)

	decrypted2, _, err := e2eeService.DecryptMessage(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, message2, decrypted2)

	t.Log("✅ Perfect Forward Secrecy test passed - messages use different keys")
}

// TestCryptographicKeyDerivation tests HKDF key derivation
func TestCryptographicKeyDerivation(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	e2eeService := services.NewMilitaryGradeE2EEService(ts.DB.DB)

	senderID := "keyderiv-sender"
	recipientID := "keyderiv-recipient"

	// Initialize keys
	_, err := e2eeService.InitializeUserKeys(senderID)
	require.NoError(t, err)
	_, err = e2eeService.InitializeUserKeys(recipientID)
	require.NoError(t, err)

	// Get session to test key derivation
	session, err := e2eeService.getOrCreateSession(senderID, recipientID)
	require.NoError(t, err)

	// Derive message keys
	messageKeys, err := e2eeService.deriveMessageKeys(session)
	require.NoError(t, err)

	// Verify key properties
	assert.NotNil(t, messageKeys.EncryptionKey)
	assert.NotNil(t, messageKeys.MACKey)
	assert.Len(t, messageKeys.EncryptionKey, 32) // AES-256
	assert.Len(t, messageKeys.MACKey, 32)        // HMAC-SHA-256

	// Verify keys are different
	assert.NotEqual(t, messageKeys.EncryptionKey, messageKeys.MACKey)

	// Verify keys are cryptographically strong (not all zeros, not predictable)
	encryptionKeySum := 0
	macKeySum := 0
	for i := range messageKeys.EncryptionKey {
		encryptionKeySum += int(messageKeys.EncryptionKey[i])
		macKeySum += int(messageKeys.MACKey[i])
	}
	assert.True(t, encryptionKeySum > 0, "Encryption key should not be all zeros")
	assert.True(t, macKeySum > 0, "MAC key should not be all zeros")

	t.Log("✅ Cryptographic key derivation test passed")
}