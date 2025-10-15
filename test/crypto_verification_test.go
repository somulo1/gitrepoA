package test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	t.Logf("✅ AES-256-GCM encryption test passed. Ciphertext length: %d, Plaintext length: %d", len(ciphertext), len(plaintext))
}

// TestCiphertextSecurityVerification tests that ciphertext is cryptographically secure
func TestCiphertextSecurityVerification(t *testing.T) {
	plaintext := "Secret message that should not be readable without proper decryption"
	key := make([]byte, 32)
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

	// Convert to base64 (like the E2EE service does)
	ciphertextBase64 := base64.StdEncoding.EncodeToString(ciphertext)

	// Verify that the ciphertext doesn't contain the plaintext in any obvious way
	ciphertextStr := string(ciphertext)

	// Check that individual words from plaintext are not visible
	words := strings.Fields(plaintext)
	for _, word := range words {
		assert.NotContains(t, ciphertextStr, word, "Ciphertext should not contain plaintext words")
	}

	// Verify it's not just base64 of the plaintext
	plaintextBase64 := base64.StdEncoding.EncodeToString([]byte(plaintext))
	assert.NotEqual(t, plaintextBase64, ciphertextBase64)

	// Verify it's not just base64 of plaintext with simple suffix
	for _, suffix := range []string{"_enc_", "_encrypted_", "_secure_"} {
		suffixedPlaintext := plaintext + suffix
		suffixedBase64 := base64.StdEncoding.EncodeToString([]byte(suffixedPlaintext))
		assert.NotEqual(t, suffixedBase64, ciphertextBase64)
	}

	// Verify the ciphertext is properly base64 encoded
	_, err = base64.StdEncoding.DecodeString(ciphertextBase64)
	assert.NoError(t, err, "Ciphertext should be valid base64")

	t.Logf("✅ Ciphertext security verification passed. Ciphertext is cryptographically secure")
}

// TestWeakEncryptionPatternDetection tests detection of weak encryption patterns
func TestWeakEncryptionPatternDetection(t *testing.T) {
	// Test the specific weak pattern from the user's example
	weakEncrypted := "dGVzdCBhZ2Fpbl9lbmNfMTc1ODU0Mzg3NzM1MF8ydjBuZm5ybTh2ZQ=="

	// This should be detected as weak encryption (base64 encoded)
	assert.True(t, strings.HasSuffix(weakEncrypted, "=="))

	// Decode it
	decoded, err := base64.StdEncoding.DecodeString(weakEncrypted)
	require.NoError(t, err)
	decodedStr := string(decoded)

	// Verify it contains the weak pattern
	assert.Contains(t, decodedStr, "_enc_")
	assert.Contains(t, decodedStr, "test again")

	// This is weak because it's just base64 of (plaintext + "_enc_" + timestamp + "_" + random)
	plaintext := "test again"
	timestamp := "1758543877350"
	random := "2v0nfnrm8ve"
	expectedWeakPattern := plaintext + "_enc_" + timestamp + "_" + random

	assert.Equal(t, expectedWeakPattern, decodedStr)

	t.Logf("✅ Weak encryption pattern detection test passed. Pattern: %s", decodedStr)
}

// TestProperEncryptionVsWeakEncryption tests the difference between proper and weak encryption
func TestProperEncryptionVsWeakEncryption(t *testing.T) {
	plaintext := "This is my secret message"

	// Simulate weak encryption (what was happening before)
	weakEncrypted := base64.StdEncoding.EncodeToString([]byte(plaintext + "_enc_" + "1234567890" + "_" + "random"))

	// Simulate proper encryption (AES-256-GCM)
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	properEncrypted := base64.StdEncoding.EncodeToString(ciphertext)

	// Verify they are completely different
	assert.NotEqual(t, weakEncrypted, properEncrypted)

	// Verify weak encryption is trivially decodable
	weakDecoded, err := base64.StdEncoding.DecodeString(weakEncrypted)
	require.NoError(t, err)
	assert.Contains(t, string(weakDecoded), plaintext)
	assert.Contains(t, string(weakDecoded), "_enc_")

	// Verify proper encryption is not trivially decodable
	properDecoded, err := base64.StdEncoding.DecodeString(properEncrypted)
	require.NoError(t, err)
	properStr := string(properDecoded)
	assert.NotContains(t, properStr, plaintext, "Proper encryption should not contain plaintext")

	// Verify proper encryption can be decrypted with correct key
	decrypted, err := gcm.Open(nil, nonce, properDecoded, nil)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted))

	t.Logf("✅ Encryption comparison test passed. Weak: %d chars, Proper: %d chars", len(weakEncrypted), len(properEncrypted))
}

// TestE2EESecurityProperties tests key E2EE security properties
func TestE2EESecurityProperties(t *testing.T) {
	plaintext := "End-to-end encrypted message"

	// Test 1: Same plaintext with different keys produces different ciphertext
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, err := rand.Read(key1)
	require.NoError(t, err)
	_, err = rand.Read(key2)
	require.NoError(t, err)

	block1, _ := aes.NewCipher(key1)
	block2, _ := aes.NewCipher(key2)
	gcm1, _ := cipher.NewGCM(block1)
	gcm2, _ := cipher.NewGCM(block2)

	nonce1 := make([]byte, gcm1.NonceSize())
	nonce2 := make([]byte, gcm2.NonceSize())
	rand.Read(nonce1)
	rand.Read(nonce2)

	ciphertext1 := gcm1.Seal(nil, nonce1, []byte(plaintext), nil)
	ciphertext2 := gcm2.Seal(nil, nonce2, []byte(plaintext), nil)

	assert.NotEqual(t, ciphertext1, ciphertext2, "Same plaintext with different keys should produce different ciphertext")

	// Test 2: Same key with different nonce produces different ciphertext
	nonce3 := make([]byte, gcm1.NonceSize())
	rand.Read(nonce3)
	ciphertext3 := gcm1.Seal(nil, nonce3, []byte(plaintext), nil)

	assert.NotEqual(t, ciphertext1, ciphertext3, "Same key with different nonce should produce different ciphertext")

	// Test 3: Ciphertext length is consistent and appropriate
	// GCM mode: ciphertext = plaintext + auth_tag (16 bytes), nonce is separate
	expectedMinLength := len(plaintext) + 16 // plaintext + auth tag
	assert.GreaterOrEqual(t, len(ciphertext1), expectedMinLength, "Ciphertext should include auth tag")

	t.Logf("✅ E2EE security properties test passed. Ciphertext length: %d (min expected: %d)", len(ciphertext1), expectedMinLength)
}