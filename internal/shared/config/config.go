package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the payment service
type Config struct {
	AppName  string         `mapstructure:"app_name"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Billing  BillingConfig  `mapstructure:"billing"`
	Log      LogConfig      `mapstructure:"log"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Address          string `mapstructure:"address"`
	EnableReflection bool   `mapstructure:"enable_reflection"`
}

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	DSN      string `mapstructure:"dsn"`
	MaxConns int32  `mapstructure:"max_conns"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	PublicKeyPEM string `mapstructure:"public_key_pem"`
}

// BillingConfig holds billing provider configuration
type BillingConfig struct {
	Provider            string `mapstructure:"provider"`
	StripeSecret        string `mapstructure:"stripe_secret"`
	StripePublishable   string `mapstructure:"stripe_publishable"`
	StripeWebhookSecret string `mapstructure:"stripe_webhook_secret"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// LoadFromEnv loads configuration from environment variables only
func LoadFromEnv() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("app_name", "payment-service")
	viper.SetDefault("grpc.address", ":8081")
	viper.SetDefault("grpc.enable_reflection", false)
	viper.SetDefault("postgres.max_conns", 10)
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("auth.public_key_pem", "")
	viper.SetDefault("billing.provider", "stripe")
	viper.SetDefault("billing.stripe_publishable", "")
	viper.SetDefault("log.level", "info")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.AppName == "" {
		return fmt.Errorf("app_name is required")
	}
	if c.GRPC.Address == "" {
		return fmt.Errorf("grpc.address is required")
	}
	if c.Postgres.DSN == "" {
		return fmt.Errorf("postgres.dsn is required")
	}
	if c.Postgres.MaxConns <= 0 {
		return fmt.Errorf("postgres.max_conns must be greater than 0")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
	}
	return nil
}
