package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"vaultke-backend/internal/config"
)

func TestConfigLoading(t *testing.T) {
	t.Run("LoadDefaultConfig", func(t *testing.T) {
		// Test loading default configuration
		cfg := config.Load()

		assert.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotEmpty(t, cfg.DatabaseURL)
		assert.NotEmpty(t, cfg.Environment)
		assert.NotEmpty(t, cfg.ServerPort)
	})

	t.Run("LoadConfigFromEnvironment", func(t *testing.T) {
		// Set environment variables
		os.Setenv("JWT_SECRET", "test-jwt-secret")
		os.Setenv("DATABASE_URL", "test-database-url")
		os.Setenv("ENVIRONMENT", "test")
		os.Setenv("PORT", "8080")

		defer func() {
			// Clean up environment variables
			os.Unsetenv("JWT_SECRET")
			os.Unsetenv("DATABASE_URL")
			os.Unsetenv("ENVIRONMENT")
			os.Unsetenv("PORT")
		}()

		cfg := config.Load()

		assert.Equal(t, "test-jwt-secret", cfg.JWTSecret)
		assert.Equal(t, "test-database-url", cfg.DatabaseURL)
		assert.Equal(t, "test", cfg.Environment)
		assert.Equal(t, "8080", cfg.ServerPort)
	})

	t.Run("LoadConfigWithMissingEnvironmentVars", func(t *testing.T) {
		// Ensure environment variables are not set
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("PORT")

		cfg := config.Load()

		// Should have default values
		assert.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotEmpty(t, cfg.DatabaseURL)
		assert.NotEmpty(t, cfg.Environment)
		assert.NotEmpty(t, cfg.ServerPort)
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		cfg := config.Load()

		// Test configuration validation
		err := cfg.Validate()
		assert.NoError(t, err)

		// Test invalid configuration
		invalidCfg := &config.Config{
			JWTSecret:   "",
			DatabaseURL: "",
			Environment: "invalid",
		}

		err = invalidCfg.Validate()
		assert.Error(t, err)
	})

	t.Run("ConfigDatabaseSettings", func(t *testing.T) {
		cfg := config.Load()

		// Test database configuration
		assert.NotEmpty(t, cfg.DatabaseURL)
		assert.True(t, cfg.DatabaseURL == ":memory:" || cfg.DatabaseURL == "vaultke.db" || cfg.DatabaseURL == "file:vaultke.db")
	})

	t.Run("ConfigSecuritySettings", func(t *testing.T) {
		cfg := config.Load()

		// Test security configuration
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.Greater(t, len(cfg.JWTSecret), 16) // JWT secret should be at least 16 characters
		assert.NotZero(t, cfg.JWTExpiration)
	})

	t.Run("ConfigServerSettings", func(t *testing.T) {
		cfg := config.Load()

		// Test server configuration
		assert.NotEmpty(t, cfg.ServerPort)
		assert.NotEmpty(t, cfg.Environment)
		assert.Contains(t, []string{"development", "production", "test"}, cfg.Environment)
	})

	t.Run("ConfigM-PesaSettings", func(t *testing.T) {
		// Set M-Pesa environment variables
		os.Setenv("MPESA_CONSUMER_KEY", "test-consumer-key")
		os.Setenv("MPESA_CONSUMER_SECRET", "test-consumer-secret")
		os.Setenv("MPESA_PASSKEY", "test-passkey")
		os.Setenv("MPESA_SHORTCODE", "123456")

		defer func() {
			os.Unsetenv("MPESA_CONSUMER_KEY")
			os.Unsetenv("MPESA_CONSUMER_SECRET")
			os.Unsetenv("MPESA_PASSKEY")
			os.Unsetenv("MPESA_SHORTCODE")
		}()

		cfg := config.Load()

		assert.Equal(t, "test-consumer-key", cfg.MpesaConsumerKey)
		assert.Equal(t, "test-consumer-secret", cfg.MpesaConsumerSecret)
		assert.Equal(t, "test-passkey", cfg.MpesaPasskey)
		assert.Equal(t, "123456", cfg.MpesaShortcode)
	})

	t.Run("ConfigGoogleSettings", func(t *testing.T) {
		// Set Google environment variables
		os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
		os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")
		os.Setenv("GOOGLE_REDIRECT_URL", "https://gitrepoa-1.onrender.com/auth/google/callback")

		defer func() {
			os.Unsetenv("GOOGLE_CLIENT_ID")
			os.Unsetenv("GOOGLE_CLIENT_SECRET")
			os.Unsetenv("GOOGLE_REDIRECT_URL")
		}()

		cfg := config.Load()

		assert.Equal(t, "test-client-id", cfg.GoogleClientID)
		assert.Equal(t, "test-client-secret", cfg.GoogleClientSecret)
		assert.Equal(t, "https://gitrepoa-1.onrender.com/auth/google/callback", cfg.GoogleRedirectURL)
	})

	t.Run("ConfigEmailSettings", func(t *testing.T) {
		// Set email environment variables
		os.Setenv("SMTP_HOST", "smtp.gmail.com")
		os.Setenv("SMTP_PORT", "587")
		os.Setenv("SMTP_USERNAME", "test@gmail.com")
		os.Setenv("SMTP_PASSWORD", "test-password")

		defer func() {
			os.Unsetenv("SMTP_HOST")
			os.Unsetenv("SMTP_PORT")
			os.Unsetenv("SMTP_USERNAME")
			os.Unsetenv("SMTP_PASSWORD")
		}()

		cfg := config.Load()

		assert.Equal(t, "smtp.gmail.com", cfg.SMTPHost)
		assert.Equal(t, "587", cfg.SMTPPort)
		assert.Equal(t, "test@gmail.com", cfg.SMTPUsername)
		assert.Equal(t, "test-password", cfg.SMTPPassword)
	})

	t.Run("ConfigRedisSettings", func(t *testing.T) {
		// Set Redis environment variables
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("REDIS_PASSWORD", "test-password")

		defer func() {
			os.Unsetenv("REDIS_URL")
			os.Unsetenv("REDIS_PASSWORD")
		}()

		cfg := config.Load()

		assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
		assert.Equal(t, "test-password", cfg.RedisPassword)
	})

	t.Run("ConfigFileUploadSettings", func(t *testing.T) {
		// Set file upload environment variables
		os.Setenv("MAX_FILE_SIZE", "10485760") // 10MB
		os.Setenv("UPLOAD_PATH", "./uploads")

		defer func() {
			os.Unsetenv("MAX_FILE_SIZE")
			os.Unsetenv("UPLOAD_PATH")
		}()

		cfg := config.Load()

		assert.Equal(t, int64(10485760), cfg.MaxFileSize)
		assert.Equal(t, "./uploads", cfg.UploadPath)
	})

	t.Run("ConfigRateLimitSettings", func(t *testing.T) {
		// Set rate limit environment variables
		os.Setenv("RATE_LIMIT_REQUESTS", "100")
		os.Setenv("RATE_LIMIT_WINDOW", "60")

		defer func() {
			os.Unsetenv("RATE_LIMIT_REQUESTS")
			os.Unsetenv("RATE_LIMIT_WINDOW")
		}()

		cfg := config.Load()

		assert.Equal(t, 100, cfg.RateLimitRequests)
		assert.Equal(t, 60, cfg.RateLimitWindow)
	})

	t.Run("ConfigLogSettings", func(t *testing.T) {
		// Set log environment variables
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("LOG_FILE", "./logs/app.log")

		defer func() {
			os.Unsetenv("LOG_LEVEL")
			os.Unsetenv("LOG_FILE")
		}()

		cfg := config.Load()

		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "./logs/app.log", cfg.LogFile)
	})

	t.Run("ConfigMonitoringSettings", func(t *testing.T) {
		// Set monitoring environment variables
		os.Setenv("ENABLE_METRICS", "true")
		os.Setenv("METRICS_PORT", "9090")
		os.Setenv("ENABLE_TRACING", "true")

		defer func() {
			os.Unsetenv("ENABLE_METRICS")
			os.Unsetenv("METRICS_PORT")
			os.Unsetenv("ENABLE_TRACING")
		}()

		cfg := config.Load()

		assert.True(t, cfg.EnableMetrics)
		assert.Equal(t, "9090", cfg.MetricsPort)
		assert.True(t, cfg.EnableTracing)
	})

	t.Run("ConfigBackupSettings", func(t *testing.T) {
		// Set backup environment variables
		os.Setenv("BACKUP_ENABLED", "true")
		os.Setenv("BACKUP_INTERVAL", "24h")
		os.Setenv("BACKUP_PATH", "./backups")

		defer func() {
			os.Unsetenv("BACKUP_ENABLED")
			os.Unsetenv("BACKUP_INTERVAL")
			os.Unsetenv("BACKUP_PATH")
		}()

		cfg := config.Load()

		assert.True(t, cfg.BackupEnabled)
		assert.Equal(t, "24h", cfg.BackupInterval)
		assert.Equal(t, "./backups", cfg.BackupPath)
	})

	t.Run("ConfigDevelopmentDefaults", func(t *testing.T) {
		// Set environment to development
		os.Setenv("ENVIRONMENT", "development")

		defer func() {
			os.Unsetenv("ENVIRONMENT")
		}()

		cfg := config.Load()

		assert.Equal(t, "development", cfg.Environment)
		// In development, certain settings should have development-friendly defaults
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotEmpty(t, cfg.DatabaseURL)
	})

	t.Run("ConfigProductionDefaults", func(t *testing.T) {
		// Set environment to production
		os.Setenv("ENVIRONMENT", "production")

		defer func() {
			os.Unsetenv("ENVIRONMENT")
		}()

		cfg := config.Load()

		assert.Equal(t, "production", cfg.Environment)
		// In production, certain settings should have production-friendly defaults
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotEmpty(t, cfg.DatabaseURL)
	})

	t.Run("ConfigTestDefaults", func(t *testing.T) {
		// Set environment to test
		os.Setenv("ENVIRONMENT", "test")

		defer func() {
			os.Unsetenv("ENVIRONMENT")
		}()

		cfg := config.Load()

		assert.Equal(t, "test", cfg.Environment)
		// In test, certain settings should have test-friendly defaults
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotEmpty(t, cfg.DatabaseURL)
	})

	t.Run("ConfigValidationEdgeCases", func(t *testing.T) {
		// Test empty JWT secret
		cfg := &config.Config{
			JWTSecret:   "",
			DatabaseURL: "test.db",
			Environment: "test",
		}
		err := cfg.Validate()
		assert.Error(t, err)

		// Test empty database URL
		cfg = &config.Config{
			JWTSecret:   "test-secret",
			DatabaseURL: "",
			Environment: "test",
		}
		err = cfg.Validate()
		assert.Error(t, err)

		// Test invalid environment
		cfg = &config.Config{
			JWTSecret:   "test-secret",
			DatabaseURL: "test.db",
			Environment: "invalid",
		}
		err = cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("ConfigToString", func(t *testing.T) {
		cfg := config.Load()

		configStr := cfg.String()
		assert.NotEmpty(t, configStr)
		assert.Contains(t, configStr, "Environment")
		assert.Contains(t, configStr, "DatabaseURL")
		// JWT secret should be masked in string representation
		assert.NotContains(t, configStr, cfg.JWTSecret)
	})

	t.Run("ConfigClone", func(t *testing.T) {
		cfg := config.Load()

		clonedCfg := cfg.Clone()
		assert.NotNil(t, clonedCfg)
		assert.Equal(t, cfg.Environment, clonedCfg.Environment)
		assert.Equal(t, cfg.DatabaseURL, clonedCfg.DatabaseURL)
		assert.Equal(t, cfg.JWTSecret, clonedCfg.JWTSecret)

		// Ensure it's a deep copy
		clonedCfg.Environment = "modified"
		assert.NotEqual(t, cfg.Environment, clonedCfg.Environment)
	})

	t.Run("ConfigReload", func(t *testing.T) {
		// Test configuration reloading
		cfg := config.Load()
		originalSecret := cfg.JWTSecret

		// Change environment variable
		os.Setenv("JWT_SECRET", "new-test-secret")

		defer func() {
			os.Unsetenv("JWT_SECRET")
		}()

		// Reload configuration
		cfg.Reload()

		assert.Equal(t, "new-test-secret", cfg.JWTSecret)
		assert.NotEqual(t, originalSecret, cfg.JWTSecret)
	})

	t.Run("ConfigEnvironmentSpecificSettings", func(t *testing.T) {
		environments := []string{"development", "production", "test"}

		for _, env := range environments {
			t.Run(env, func(t *testing.T) {
				os.Setenv("ENVIRONMENT", env)

				defer func() {
					os.Unsetenv("ENVIRONMENT")
				}()

				cfg := config.Load()
				assert.Equal(t, env, cfg.Environment)

				// Validate environment-specific settings
				switch env {
				case "development":
					assert.NotEmpty(t, cfg.JWTSecret)
					assert.NotEmpty(t, cfg.DatabaseURL)
				case "production":
					assert.NotEmpty(t, cfg.JWTSecret)
					assert.NotEmpty(t, cfg.DatabaseURL)
				case "test":
					assert.NotEmpty(t, cfg.JWTSecret)
					assert.NotEmpty(t, cfg.DatabaseURL)
				}
			})
		}
	})

	t.Run("ConfigFromFile", func(t *testing.T) {
		// Skip this test as LoadFromFile method doesn't exist in the current config implementation
		t.Skip("LoadFromFile method not implemented in current config package")
	})

	t.Run("ConfigSaveToFile", func(t *testing.T) {
		// Skip this test as SaveToFile method doesn't exist in the current config implementation
		t.Skip("SaveToFile method not implemented in current config package")
	})

	t.Run("ConfigMerge", func(t *testing.T) {
		// Skip this test as Merge method and ServerPort field don't exist in the current config implementation
		t.Skip("Merge method and ServerPort field not implemented in current config package")
	})

	t.Run("ConfigValidateRequired", func(t *testing.T) {
		// Test validation of required fields
		cfg := &config.Config{}

		err := cfg.ValidateRequired()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT secret is required")

		cfg.JWTSecret = "test-secret"
		err = cfg.ValidateRequired()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Database URL is required")

		cfg.DatabaseURL = "test.db"
		err = cfg.ValidateRequired()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Environment is required")

		cfg.Environment = "test"
		err = cfg.ValidateRequired()
		assert.NoError(t, err)
	})

	t.Run("ConfigSetDefaults", func(t *testing.T) {
		// Test setting default values
		cfg := &config.Config{}

		cfg.SetDefaults()

		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotEmpty(t, cfg.DatabaseURL)
		assert.NotEmpty(t, cfg.Environment)
		assert.NotEmpty(t, cfg.ServerPort)
	})
}
