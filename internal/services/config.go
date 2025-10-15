package services

import (
	"os"
)

// LiveKitConfig holds LiveKit configuration
type LiveKitConfig struct {
	WSURL     string
	APIKey    string
	APISecret string
}

// GetLiveKitConfig returns LiveKit configuration from environment variables
func GetLiveKitConfig() *LiveKitConfig {
	return &LiveKitConfig{
		WSURL:     getEnvOrDefault("LIVEKIT_WS_URL", "ws://localhost:7880"),
		APIKey:    getEnvOrDefault("LIVEKIT_API_KEY", ""),
		APISecret: getEnvOrDefault("LIVEKIT_API_SECRET", ""),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
