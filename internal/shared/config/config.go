package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the payment service
type Config struct {
	AppName          string                 `mapstructure:"app_name"`
	GRPC             GRPCConfig             `mapstructure:"grpc"`
	Postgres         PostgresConfig         `mapstructure:"postgres"`
	Redis            RedisConfig            `mapstructure:"redis"`
	Auth             AuthConfig             `mapstructure:"auth"`
	Billing          BillingConfig          `mapstructure:"billing"`
	Events           EventsConfig           `mapstructure:"events"`
	Log              LogConfig              `mapstructure:"log"`
	ServiceMesh      ServiceMeshConfig      `mapstructure:"service_mesh"`
	MTLS             MTLSConfig             `mapstructure:"mtls"`
	ExternalServices ExternalServicesConfig `mapstructure:"external_services"`
	CircuitBreaker   CircuitBreakerConfig   `mapstructure:"circuit_breaker"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Address          string `mapstructure:"address"`           // gRPC server address (e.g., ":8081")
	EnableReflection bool   `mapstructure:"enable_reflection"` // Enable gRPC reflection (for debugging)
}

// APIGatewayConfig holds API Gateway configuration
type APIGatewayConfig struct {
	Address           string `mapstructure:"address"`
	EnableTLS         bool   `mapstructure:"enable_tls"`
	APIKey            string `mapstructure:"api_key"`
	CertFile          string `mapstructure:"cert_file"`
	KeyFile           string `mapstructure:"key_file"`
	CAFile            string `mapstructure:"ca_file"`
	DialTimeoutSec    int    `mapstructure:"dial_timeout_seconds"`
	RequestTimeoutSec int    `mapstructure:"request_timeout_seconds"`
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
	PublicKeyPEM string `mapstructure:"public_key_pem"`
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

// ServiceMeshConfig holds service mesh configuration (Envoy + Spiffe)
type ServiceMeshConfig struct {
	Enabled   bool            `mapstructure:"enabled"`
	SpiffeID  string          `mapstructure:"spiffe_id"`
	Discovery DiscoveryConfig `mapstructure:"discovery"`
}

// DiscoveryConfig holds service discovery configuration
type DiscoveryConfig struct {
	EnvoyAddress string `mapstructure:"envoy_address"`
	Namespace    string `mapstructure:"namespace"`
	Scheme       string `mapstructure:"scheme"`
	TimeoutSec   int    `mapstructure:"timeout_seconds"`
}

// MTLSConfig holds mTLS configuration
type MTLSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
	CAFile     string `mapstructure:"ca_file"`
	MinVersion string `mapstructure:"min_version"`
}

// ExternalServicesConfig holds configuration for external services
type ExternalServicesConfig struct {
	ContactService  ServiceConfig `mapstructure:"contact_service"`
	FamilyService   ServiceConfig `mapstructure:"family_service"`
	DocumentService ServiceConfig `mapstructure:"document_service"`
}

// ServiceConfig holds configuration for a single external service
type ServiceConfig struct {
	Name       string `mapstructure:"name"`
	Address    string `mapstructure:"address"`
	SpiffeID   string `mapstructure:"spiffe_id"`
	TimeoutSec int    `mapstructure:"timeout_seconds"`
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	FailureThreshold int  `mapstructure:"failure_threshold"`
	SuccessThreshold int  `mapstructure:"success_threshold"`
	TimeoutSec       int  `mapstructure:"timeout_seconds"`
	HalfOpenMaxCalls int  `mapstructure:"half_open_max_calls"`
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
	viper.SetDefault("events.provider", "kafka")
	viper.SetDefault("events.topic", "payments")
	viper.SetDefault("log.level", "info")

	// API Gateway defaults
	viper.SetDefault("api_gateway.address", "api-gateway:8080")
	viper.SetDefault("api_gateway.enable_tls", true)
	viper.SetDefault("api_gateway.dial_timeout_seconds", 30)
	viper.SetDefault("api_gateway.request_timeout_seconds", 30)

	// Service Mesh defaults
	viper.SetDefault("service_mesh.enabled", false)
	viper.SetDefault("service_mesh.spiffe_id", "spiffe://jia.app/payment-service")
	viper.SetDefault("service_mesh.discovery.envoy_address", "localhost:8500")
	viper.SetDefault("service_mesh.discovery.namespace", "default")
	viper.SetDefault("service_mesh.discovery.scheme", "xds")
	viper.SetDefault("service_mesh.discovery.timeout_seconds", 30)

	// mTLS defaults
	viper.SetDefault("mtls.enabled", false)
	viper.SetDefault("mtls.min_version", "1.2")

	// Circuit Breaker defaults
	viper.SetDefault("circuit_breaker.enabled", true)
	viper.SetDefault("circuit_breaker.failure_threshold", 5)
	viper.SetDefault("circuit_breaker.success_threshold", 2)
	viper.SetDefault("circuit_breaker.timeout_seconds", 60)
	viper.SetDefault("circuit_breaker.half_open_max_calls", 3)
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
