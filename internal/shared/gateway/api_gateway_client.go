package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/jia-app/paymentservice/internal/shared/log"
)

// GatewayConfig holds API Gateway configuration
type GatewayConfig struct {
	// Gateway address (e.g., "api-gateway:8080")
	Address string

	// Enable TLS
	EnableTLS bool

	// Certificate for TLS (optional, for mutual TLS)
	CertFile string
	KeyFile  string
	CAFile   string

	// Timeouts
	DialTimeout    time.Duration
	RequestTimeout time.Duration

	// API Key for gateway authentication (optional)
	APIKey string

	// Logger
	Logger *zap.Logger
}

// DefaultGatewayConfig returns default gateway configuration
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		EnableTLS:      true,
		DialTimeout:    30 * time.Second,
		RequestTimeout: 30 * time.Second,
	}
}

// APIGatewayClient represents a client for the API Gateway
type APIGatewayClient struct {
	conn   *grpc.ClientConn
	config GatewayConfig
	logger *zap.Logger
}

// NewAPIGatewayClient creates a new API Gateway client
func NewAPIGatewayClient(config GatewayConfig) (*APIGatewayClient, error) {
	logger := config.Logger
	if logger == nil {
		logger = log.L(context.Background())
	}

	// Build dial options
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	// Setup TLS
	if config.EnableTLS {
		var creds credentials.TransportCredentials
		if config.CertFile != "" && config.KeyFile != "" {
			// Mutual TLS
			cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			creds = credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			})
		} else {
			// Standard TLS
			creds = credentials.NewTLS(&tls.Config{
				MinVersion: tls.VersionTLS12,
			})
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
		logger.Info("TLS enabled for API Gateway client",
			zap.String("address", config.Address))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		logger.Warn("TLS disabled for API Gateway client - using insecure connection",
			zap.String("address", config.Address))
	}

	// Connect to gateway
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial API Gateway %s: %w", config.Address, err)
	}

	logger.Info("API Gateway client connected",
		zap.String("address", config.Address))

	return &APIGatewayClient{
		conn:   conn,
		config: config,
		logger: logger,
	}, nil
}

// CallService makes a gRPC call through the API Gateway
func (c *APIGatewayClient) CallService(ctx context.Context, serviceName, method string, req interface{}, resp interface{}) error {
	// Add API key to metadata if configured
	md := metadata.New(nil)
	if c.config.APIKey != "" {
		md.Set("x-api-key", c.config.APIKey)
	}

	// Add service name to metadata for gateway routing
	md.Set("x-service-name", serviceName)

	ctx = metadata.NewOutgoingContext(ctx, md)

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	// Make the call
	err := c.conn.Invoke(ctx, method, req, resp)
	if err != nil {
		c.logger.Error("API Gateway call failed",
			zap.String("service", serviceName),
			zap.String("method", method),
			zap.Error(err))
		return fmt.Errorf("API Gateway call failed: %w", err)
	}

	c.logger.Info("API Gateway call successful",
		zap.String("service", serviceName),
		zap.String("method", method))

	return nil
}

// GetConn returns the underlying gRPC connection
func (c *APIGatewayClient) GetConn() *grpc.ClientConn {
	return c.conn
}

// Close closes the API Gateway connection
func (c *APIGatewayClient) Close() error {
	if c.conn != nil {
		c.logger.Info("Closing API Gateway client connection",
			zap.String("address", c.config.Address))
		return c.conn.Close()
	}
	return nil
}

// IsHealthy checks if the connection is healthy
func (c *APIGatewayClient) IsHealthy(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() == "TRANSIENT_FAILURE" || state.String() == "SHUTDOWN" {
		return fmt.Errorf("connection is not healthy: %s", state.String())
	}
	return nil
}
