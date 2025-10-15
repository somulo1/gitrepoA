package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"vaultke-backend/internal/models"
)

// AuthService handles authentication-related business logic
type AuthService struct {
	jwtSecret     string
	jwtExpiration time.Duration
	// In-memory blacklist for tokens (in production, use Redis or database)
	blacklistedTokens map[string]time.Time
	blacklistMutex    sync.RWMutex
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret string, jwtExpirationSeconds int) *AuthService {
	return &AuthService{
		jwtSecret:         jwtSecret,
		jwtExpiration:     time.Duration(jwtExpirationSeconds) * time.Second,
		blacklistedTokens: make(map[string]time.Time),
	}
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vaultke",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Check if token is blacklisted first
	if s.IsTokenBlacklisted(tokenString) {
		return nil, fmt.Errorf("token has been revoked")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken generates a new token for a user (if the current token is valid)
func (s *AuthService) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Check if token is close to expiry (within 1 hour)
	if time.Until(claims.ExpiresAt.Time) > time.Hour {
		return "", fmt.Errorf("token is not close to expiry")
	}

	// Create new token with same claims but new expiry
	now := time.Now()
	newClaims := &JWTClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vaultke",
			Subject:   claims.UserID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err = token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refreshed token: %w", err)
	}

	return tokenString, nil
}

// ExtractUserIDFromToken extracts user ID from token without full validation
func (s *AuthService) ExtractUserIDFromToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// IsTokenExpired checks if a token is expired
func (s *AuthService) IsTokenExpired(tokenString string) bool {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return true
	}
	return time.Now().After(claims.ExpiresAt.Time)
}

// GetTokenExpiryTime returns the expiry time of a token
func (s *AuthService) GetTokenExpiryTime(tokenString string) (time.Time, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}

// BlacklistToken adds a token to the blacklist
func (s *AuthService) BlacklistToken(tokenString string) error {
	// Get token expiry time to know when to remove it from blacklist
	expiryTime, err := s.GetTokenExpiryTime(tokenString)
	if err != nil {
		// If we can't parse the token, still add it to blacklist with a default expiry
		expiryTime = time.Now().Add(s.jwtExpiration)
	}

	s.blacklistMutex.Lock()
	defer s.blacklistMutex.Unlock()

	s.blacklistedTokens[tokenString] = expiryTime
	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (s *AuthService) IsTokenBlacklisted(tokenString string) bool {
	s.blacklistMutex.RLock()
	defer s.blacklistMutex.RUnlock()

	expiryTime, exists := s.blacklistedTokens[tokenString]
	if !exists {
		return false
	}

	// If token has expired, remove it from blacklist and return false
	if time.Now().After(expiryTime) {
		s.blacklistMutex.RUnlock()
		s.blacklistMutex.Lock()
		delete(s.blacklistedTokens, tokenString)
		s.blacklistMutex.Unlock()
		s.blacklistMutex.RLock()
		return false
	}

	return true
}

// CleanupExpiredTokens removes expired tokens from the blacklist
func (s *AuthService) CleanupExpiredTokens() {
	s.blacklistMutex.Lock()
	defer s.blacklistMutex.Unlock()

	now := time.Now()
	for token, expiryTime := range s.blacklistedTokens {
		if now.After(expiryTime) {
			delete(s.blacklistedTokens, token)
		}
	}
}
