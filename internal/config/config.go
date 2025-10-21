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
	Events   EventsConfig   `mapstructure:"events"`
	Log      LogConfig      `mapstructure:"log"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Secrets  SecretsConfig  `mapstructure:"secrets"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Address      string `mapstructure:"address"`        // gRPC server address (e.g., ":8081")
	TLSEnabled   bool   `mapstructure:"tls_enabled"`    // Enable TLS
	CertFile     string `mapstructure:"cert_file"`      // Path to TLS certificate file
	KeyFile      string `mapstructure:"key_file"`       // Path to TLS key file
	ClientCAFile string `mapstructure:"client_ca_file"` // Path to client CA file for mTLS
	MTLSEnabled  bool   `mapstructure:"mtls_enabled"`   // Enable mTLS (mutual TLS)
}

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	DSN      string `mapstructure:"dsn"`       // Database connection string
	MaxConns int32  `mapstructure:"max_conns"` // Maximum number of database connections
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`     // Redis server address (e.g., "localhost:6379")
	DB       int    `mapstructure:"db"`       // Redis database number
	Password string `mapstructure:"password"` // Redis password (optional)
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	PublicKeyPEM       string   `mapstructure:"public_key_pem"`      // JWT public key as PEM string
	PublicKeyPath      string   `mapstructure:"public_key_path"`     // Path to JWT public key file (fallback)
	SPIFFEEnabled      bool     `mapstructure:"spiffe_enabled"`      // Enable SPIFFE peer validation
	SPIFFETrustDomain  string   `mapstructure:"spiffe_trust_domain"` // SPIFFE trust domain
	SPIFFECACertPEM    string   `mapstructure:"spiffe_ca_cert_pem"`  // SPIFFE CA certificate as PEM string
	WhitelistedMethods []string `mapstructure:"whitelisted_methods"` // List of whitelisted gRPC methods
}

// BillingConfig holds billing provider configuration
type BillingConfig struct {
	Provider            string `mapstructure:"provider"`
	StripeSecret        string `mapstructure:"stripe_secret"`
	StripePublishable   string `mapstructure:"stripe_publishable"`
	StripeWebhookSecret string `mapstructure:"stripe_webhook_secret"`
}

// EventsConfig holds event streaming configuration
type EventsConfig struct {
	Provider string   `mapstructure:"provider"`
	Brokers  []string `mapstructure:"brokers"`
	Topic    string   `mapstructure:"topic"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
	Environment    string  `mapstructure:"environment"`
	JaegerEndpoint string  `mapstructure:"jaeger_endpoint"`
	SamplingRatio  float64 `mapstructure:"sampling_ratio"`
}

// SecretsConfig holds secret management configuration
type SecretsConfig struct {
	Provider   string `mapstructure:"provider"`    // Options: env, file, vault
	BasePath   string `mapstructure:"base_path"`   // Base path for file provider
	Prefix     string `mapstructure:"prefix"`      // Prefix for environment variables
	CacheTTL   string `mapstructure:"cache_ttl"`   // Cache TTL for secrets
	VaultAddr  string `mapstructure:"vault_addr"`  // Vault server address
	VaultToken string `mapstructure:"vault_token"` // Vault token
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

	// Process comma-separated whitelisted methods
	config.Auth.WhitelistedMethods = processWhitelistedMethods(config.Auth.WhitelistedMethods)

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

	// Process comma-separated whitelisted methods
	config.Auth.WhitelistedMethods = processWhitelistedMethods(config.Auth.WhitelistedMethods)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// processWhitelistedMethods processes comma-separated whitelisted methods
func processWhitelistedMethods(methods []string) []string {
	if len(methods) == 0 {
		return methods
	}

	// If there's only one element and it contains commas, split it
	if len(methods) == 1 && strings.Contains(methods[0], ",") {
		parts := strings.Split(methods[0], ",")
		var result []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	return methods
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("app_name", "payment-service")
	viper.SetDefault("grpc.address", ":8081")
	viper.SetDefault("grpc.tls_enabled", true)   // Enable TLS by default in production
	viper.SetDefault("grpc.mtls_enabled", false) // Disable mTLS by default
	viper.SetDefault("postgres.max_conns", 10)
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("auth.public_key_pem", "")
	viper.SetDefault("auth.public_key_path", "")
	viper.SetDefault("auth.spiffe_enabled", false)
	viper.SetDefault("auth.spiffe_trust_domain", "")
	viper.SetDefault("auth.spiffe_ca_cert_pem", "")
	viper.SetDefault("auth.whitelisted_methods", []string{"/payment.v1.PaymentService/PaymentSuccessWebhook"})
	viper.SetDefault("billing.provider", "stripe")
	viper.SetDefault("billing.stripe_publishable", "")
	viper.SetDefault("events.provider", "kafka")
	viper.SetDefault("events.topic", "payments")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("tracing.enabled", false)
	viper.SetDefault("tracing.service_name", "payment-service")
	viper.SetDefault("tracing.service_version", "1.0.0")
	viper.SetDefault("tracing.environment", "development")
	viper.SetDefault("tracing.jaeger_endpoint", "http://localhost:14268/api/traces")
	viper.SetDefault("tracing.sampling_ratio", 1.0)
	viper.SetDefault("secrets.provider", "env")
	viper.SetDefault("secrets.prefix", "PAYMENT_SERVICE")
	viper.SetDefault("secrets.cache_ttl", "5m")
	viper.SetDefault("secrets.base_path", "/etc/secrets")
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

	// Validate authentication configuration
	if c.Auth.PublicKeyPEM == "" && c.Auth.PublicKeyPath == "" {
		return fmt.Errorf("either auth.public_key_pem or auth.public_key_path is required")
	}

	// Validate TLS configuration
	if c.GRPC.TLSEnabled {
		if c.GRPC.CertFile == "" || c.GRPC.KeyFile == "" {
			return fmt.Errorf("grpc.cert_file and grpc.key_file are required when TLS is enabled")
		}
	}

	// Validate mTLS configuration
	if c.GRPC.MTLSEnabled {
		if !c.GRPC.TLSEnabled {
			return fmt.Errorf("grpc.tls_enabled must be true when mTLS is enabled")
		}
		if c.GRPC.ClientCAFile == "" {
			return fmt.Errorf("grpc.client_ca_file is required when mTLS is enabled")
		}
	}

	// Validate SPIFFE configuration
	if c.Auth.SPIFFEEnabled {
		if c.Auth.SPIFFETrustDomain == "" {
			return fmt.Errorf("auth.spiffe_trust_domain is required when SPIFFE is enabled")
		}
		if !c.GRPC.MTLSEnabled {
			return fmt.Errorf("grpc.mtls_enabled must be true when SPIFFE is enabled")
		}
	}

	return nil
}
