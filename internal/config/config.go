package config

import (
	"fmt"
	"time"

	"log"

	"github.com/spf13/viper"
)

type Config struct {
	// Database Configuration
	DatabaseHost     string `mapstructure:"DB_HOST"`
	DatabasePort     int    `mapstructure:"DB_PORT"`
	DatabaseUsername string `mapstructure:"DB_USERNAME"`
	DatabasePassword string `mapstructure:"DB_PASSWORD"`
	DatabaseName     string `mapstructure:"DB_DATABASE_NAME"`
	DatabaseSSLMode  string `mapstructure:"DB_SSL_MODE"`
	DatabaseMaxConns int32  `mapstructure:"DB_MAX_CONNS"`
	DatabaseMinConns int32  `mapstructure:"DB_MIN_CONNS"`

	// Application Configuration
	AppPort        int    `mapstructure:"APP_PORT"`
	AppEnvironment string `mapstructure:"APP_ENVIRONMENT"`
	AppName        string `mapstructure:"APP_NAME"`

	// Redis Configuration
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	// Queue Configuration
	QueueWorkers       int `mapstructure:"QUEUE_WORKERS"`
	BurstWindowSeconds int `mapstructure:"BURST_WINDOW_SECONDS"`
	BurstThreshold     int `mapstructure:"BURST_THRESHOLD"`

	// Retry Configuration
	MaxRetries       int `mapstructure:"MAX_RETRIES"`
	RetryBaseDelayMs int `mapstructure:"RETRY_BASE_DELAY_MS"`
	RetryMaxDelayMs  int `mapstructure:"RETRY_MAX_DELAY_MS"`

	// Batching Configuration
	BatchWindowSeconds int `mapstructure:"BATCH_WINDOW_SECONDS"`
	BatchSize          int `mapstructure:"BATCH_SIZE"`

	// Rate Limiting
	RateLimitPerMinute int `mapstructure:"RATE_LIMIT_PER_MINUTE"`
	RetryMaxDelay      int `mapstruct:"RETRY_MAX_DELAY"`
	RetryBaseDelay     int `mapstruct:"RETRY_BASE_DELAY"`

	// Logging
	LogLevel  string `mapstructure:"LOG_LEVEL"`
	LogFormat string `mapstructure:"LOG_FORMAT"`

	// External API Keys
	ResendApiKey     string `mapstructure:"RESEND_API_KEY"`
	ApiAccessKey     string `mapstructure:"API_ACCESS_KEY"`
	GoogleApiKey     string `mapstructure:"GOOGLE_API_KEY"`
	SendGridApiKey   string `mapstructure:"SENDGRID_API_KEY"`
	TwilioAccountSid string `mapstructure:"TWILIO_ACCOUNT_SID"`
	TwilioAuthToken  string `mapstructure:"TWILIO_AUTH_TOKEN"`
	TwilioFromNumber string `mapstructure:"TWILIO_FROM_NUMBER"`

	// AWS Configuration (for S3, SES, etc.)
	AwsAccessKeyID     string `mapstructure:"AWS_ACCESS_KEY_ID"`
	AwsSecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	AwsRegion          string `mapstructure:"AWS_REGION"`
	AwsS3Bucket        string `mapstructure:"AWS_S3_BUCKET"`

	// Firebase Configuration (for FCM push notifications)
	FirebaseProjectID      string `mapstructure:"FIREBASE_PROJECT_ID"`
	FirebaseCredentialPath string `mapstructure:"FIREBASE_CREDENTIAL_PATH"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set config file type and name
	v.SetConfigName(".env") // Name of config file (without extension)
	v.SetConfigType("env")  // ✅ REQUIRED for .env files
	v.AddConfigPath(".")    // ✅ Look in current directory
	v.AddConfigPath("./")   // ✅ Alternative current directory
	v.AddConfigPath("../")  // Look in parent directory (if running from cmd/)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("⚠️ .env file not found, using environment variables and defaults")
		} else {
			log.Printf("⚠️ Error reading .env file: %v\n", err)
		}
	} else {
		log.Printf("✅ .env file loaded from: %s\n", v.ConfigFileUsed())
	}

	// Automatically bind environment variables
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Debug: Print loaded values
	log.Printf("📋 Config loaded - Firebase Project: %s, Cred Path: %s\n",
		cfg.FirebaseProjectID,
		cfg.FirebaseCredentialPath,
	)

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Database defaults
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USERNAME", "postgres")
	v.SetDefault("DB_PASSWORD", "root")
	v.SetDefault("DB_DATABASE_NAME", "notification")
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("DB_MAX_CONNS", 100)
	v.SetDefault("DB_MIN_CONNS", 10)

	// Application defaults
	v.SetDefault("APP_PORT", 8080)
	v.SetDefault("APP_ENVIRONMENT", "development")
	v.SetDefault("APP_NAME", "Notification System")

	// Redis defaults
	v.SetDefault("REDIS_ADDR", "localhost:6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)

	// Queue defaults
	v.SetDefault("QUEUE_WORKERS", 10)
	v.SetDefault("BURST_WINDOW_SECONDS", 120)
	v.SetDefault("BURST_THRESHOLD", 10)

	// Retry defaults
	v.SetDefault("MAX_RETRIES", 3)
	v.SetDefault("RETRY_BASE_DELAY_MS", 1000)
	v.SetDefault("RETRY_MAX_DELAY_MS", 15000)

	// Batching defaults
	v.SetDefault("BATCH_WINDOW_SECONDS", 300)
	v.SetDefault("BATCH_SIZE", 50)

	// Rate limit defaults
	v.SetDefault("RATE_LIMIT_PER_MINUTE", 1000)

	// Logging defaults
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	// Auth defaults (optional)
	v.SetDefault("AUTH_SECRET", "change-me-in-production")
	v.SetDefault("AUTH_EXPIRY_PERIOD", 3600)      // 1 hour
	v.SetDefault("REFRESH_EXPIRY_PERIOD", 604800) // 7 days

	// AWS defaults
	v.SetDefault("AWS_REGION", "us-east-1")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Database validation
	if c.DatabaseHost == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.DatabasePort <= 0 {
		return fmt.Errorf("invalid DB_PORT")
	}
	if c.DatabaseUsername == "" {
		return fmt.Errorf("DB_USERNAME is required")
	}
	if c.DatabaseName == "" {
		return fmt.Errorf("DB_DATABASE_NAME is required")
	}

	// Queue validation
	if c.QueueWorkers <= 0 {
		return fmt.Errorf("QUEUE_WORKERS must be positive")
	}

	// Retry validation
	if c.MaxRetries < 0 {
		return fmt.Errorf("MAX_RETRIES must be non-negative")
	}

	return nil
}

// Helper methods for duration conversion
func (c *Config) GetRetryBaseDelay() time.Duration {
	return time.Duration(c.RetryBaseDelayMs) * time.Millisecond
}

func (c *Config) GetRetryMaxDelay() time.Duration {
	return time.Duration(c.RetryMaxDelayMs) * time.Millisecond
}

// func (c *Config) GetAuthExpiryDuration() time.Duration {
// 	return time.Duration(c.AuthExpiryPeriod) * time.Second
// }

// func (c *Config) GetRefreshExpiryDuration() time.Duration {
// 	return time.Duration(c.RefreshExpiryPeriod) * time.Second
// }

// IsDevelopment returns true if environment is development
func (c *Config) IsDevelopment() bool {
	return c.AppEnvironment == "development"
}

// IsProduction returns true if environment is production
func (c *Config) IsProduction() bool {
	return c.AppEnvironment == "production"
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {

	dns := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DatabaseUsername,
		c.DatabasePassword,
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseName,
		c.DatabaseSSLMode,
	)
	log.Println("Database DSN", "dsn", dns)
	return dns
}
