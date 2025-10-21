package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	Environment   string
	Port          string
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration int

	// M-Pesa Configuration
	MpesaConsumerKey       string
	MpesaConsumerSecret    string
	MpesaPasskey           string
	MpesaShortcode         string
	MpesaCallbackURL       string
	MpesaInitiatorName     string
	MpesaInitiatorPassword string
	BaseURL                string

	// Firebase Configuration
	FirebaseProjectID    string
	FirebasePrivateKeyID string
	FirebasePrivateKey   string
	FirebaseClientEmail  string
	FirebaseClientID     string
	FirebaseAuthURI      string
	FirebaseTokenURI     string
	FirebaseServerKey    string

	// Google Drive Configuration
	GoogleDriveClientID     string
	GoogleDriveClientSecret string
	GoogleDriveRedirectURL  string

	// Email Configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string

	// SMS Configuration (Africa's Talking)
	ATUsername string
	ATAPIKey   string
	ATSender   string

	// File Upload Configuration
	MaxFileSize      int64
	AllowedFileTypes []string
	UploadPath       string


	// Google OAuth Configuration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Redis Configuration
	RedisURL      string
	RedisPassword string

	// Rate Limiting Configuration
	RateLimitRequests int
	RateLimitWindow   int

	// Logging Configuration
	LogLevel string
	LogFile  string

	// CORS Configuration
	AllowedOrigins []string
	AllowAllOrigins bool

	// Metrics and Monitoring Configuration
	EnableMetrics bool
	MetricsPort   string
	EnableTracing bool

	// Backup Configuration
	BackupEnabled  bool
	BackupInterval int
	BackupPath     string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Environment:   getEnv("ENVIRONMENT", "development"),
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "vaultke.db"),
		JWTSecret:     getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		JWTExpiration: getEnvAsInt("JWT_EXPIRATION", 24*60*60), // 24 hours in seconds

		// M-Pesa Configuration
		MpesaConsumerKey:       getEnv("MPESA_CONSUMER_KEY", ""),
		MpesaConsumerSecret:    getEnv("MPESA_CONSUMER_SECRET", ""),
		MpesaPasskey:           getEnv("MPESA_PASSKEY", ""),
		MpesaShortcode:         getEnv("MPESA_SHORTCODE", ""),
		MpesaCallbackURL:       getEnv("MPESA_CALLBACK_URL", ""),
		MpesaInitiatorName:     getEnv("MPESA_INITIATOR_NAME", "testapi"),
		MpesaInitiatorPassword: getEnv("MPESA_INITIATOR_PASSWORD", "Safaricom999!*!"),
		BaseURL:                getEnv("BASE_URL", "https://chama-backend-server.vercel.app"),

		// Firebase Configuration
		FirebaseProjectID:    getEnv("FIREBASE_PROJECT_ID", ""),
		FirebasePrivateKeyID: getEnv("FIREBASE_PRIVATE_KEY_ID", ""),
		FirebasePrivateKey:   getEnv("FIREBASE_PRIVATE_KEY", ""),
		FirebaseClientEmail:  getEnv("FIREBASE_CLIENT_EMAIL", ""),
		FirebaseClientID:     getEnv("FIREBASE_CLIENT_ID", ""),
		FirebaseAuthURI:      getEnv("FIREBASE_AUTH_URI", ""),
		FirebaseTokenURI:     getEnv("FIREBASE_TOKEN_URI", ""),
		FirebaseServerKey:    getEnv("FIREBASE_SERVER_KEY", ""),

		// Google Drive Configuration
		GoogleDriveClientID:     getEnv("GOOGLE_DRIVE_CLIENT_ID", ""),
		GoogleDriveClientSecret: getEnv("GOOGLE_DRIVE_CLIENT_SECRET", ""),
		GoogleDriveRedirectURL:  getEnv("GOOGLE_DRIVE_REDIRECT_URL", ""),

		// Email Configuration
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),

		// SMS Configuration
		ATUsername: getEnv("AT_USERNAME", ""),
		ATAPIKey:   getEnv("AT_API_KEY", ""),
		ATSender:   getEnv("AT_SENDER", "VaultKe"),

		// File Upload Configuration
		MaxFileSize:      getEnvAsInt64("MAX_FILE_SIZE", 5*1024*1024), // 5MB
		AllowedFileTypes: []string{"image/jpeg", "image/png", "image/webp"},
		UploadPath:       getEnv("UPLOAD_PATH", "./uploads"),


		// Google OAuth Configuration
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),

		// Redis Configuration
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		// Rate Limiting Configuration
		RateLimitRequests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvAsInt("RATE_LIMIT_WINDOW", 60),

		// Logging Configuration
		LogLevel: getEnv("LOG_LEVEL", "info"),
		LogFile:  getEnv("LOG_FILE", ""),

		// Metrics and Monitoring Configuration
		EnableMetrics: getEnvAsBool("ENABLE_METRICS", false),
		MetricsPort:   getEnv("METRICS_PORT", "9090"),
		EnableTracing: getEnvAsBool("ENABLE_TRACING", false),

		// Backup Configuration
		BackupEnabled:  getEnvAsBool("BACKUP_ENABLED", false),
		BackupInterval: getEnvAsInt("BACKUP_INTERVAL", 24),
		BackupPath:     getEnv("BACKUP_PATH", "./backups"),

		// CORS Configuration
		AllowedOrigins:  getEnvAsStringSlice("ALLOWED_ORIGINS", []string{}),
		AllowAllOrigins: getEnvAsBool("ALLOW_ALL_ORIGINS", true), // Default to true for development
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// ServerPort returns the server port (alias for Port for test compatibility)
func (c *Config) ServerPort() string {
	return c.Port
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if c.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	// Validate environment values
	validEnvs := map[string]bool{
		"development": true,
		"production":  true,
		"test":        true,
	}
	if !validEnvs[c.Environment] {
		return fmt.Errorf("invalid environment: %s", c.Environment)
	}

	return nil
}

// ValidateRequired validates only required fields
func (c *Config) ValidateRequired() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("Database URL is required")
	}
	if c.Environment == "" {
		return fmt.Errorf("Environment is required")
	}
	return nil
}

// SetDefaults sets default values for configuration
func (c *Config) SetDefaults() {
	if c.JWTSecret == "" {
		c.JWTSecret = "your-super-secret-jwt-key-change-in-production"
	}
	if c.DatabaseURL == "" {
		c.DatabaseURL = "vaultke.db"
	}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	return fmt.Sprintf("Config{Environment: %s, Port: %s, DatabaseURL: %s}", c.Environment, c.Port, c.DatabaseURL)
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c
	// Deep copy slices
	if c.AllowedFileTypes != nil {
		clone.AllowedFileTypes = make([]string, len(c.AllowedFileTypes))
		copy(clone.AllowedFileTypes, c.AllowedFileTypes)
	}
	return &clone
}

// Reload reloads the configuration from environment variables
func (c *Config) Reload() {
	newConfig := Load()
	*c = *newConfig
}
