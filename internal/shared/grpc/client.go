package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/jia-app/paymentservice/internal/shared/log"
)

// ClientConfig holds configuration for gRPC client connections
type ClientConfig struct {
	// Target address (e.g., "contact-service:50051")
	Target string

	// Service name for service discovery
	ServiceName string

	// Enable mTLS
	EnableMTLS bool

	// Certificate paths
	CertFile string
	KeyFile  string
	CAFile   string

	// Spiffe ID for this service
	SpiffeID string

	// Timeouts
	DialTimeout    time.Duration
	RequestTimeout time.Duration

	// Keepalive settings
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration

	// Logger
	Logger *zap.Logger
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		EnableMTLS:       true,
		DialTimeout:      30 * time.Second,
		RequestTimeout:   10 * time.Second,
		KeepaliveTime:    30 * time.Second,
		KeepaliveTimeout: 5 * time.Second,
	}
}

// Client represents a gRPC client with mTLS and spiffe authentication
type Client struct {
	conn     *grpc.ClientConn
	config   ClientConfig
	logger   *zap.Logger
	spiffeID string
}

// NewClient creates a new gRPC client with mTLS and spiffe authentication
func NewClient(config ClientConfig) (*Client, error) {
	logger := config.Logger
	if logger == nil {
		logger = log.L(context.Background())
	}

	// Build dial options
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepaliveTime,
			Timeout:             config.KeepaliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	// Setup TLS/mTLS
	if config.EnableMTLS {
		tlsConfig, err := loadTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
		logger.Info("mTLS enabled for gRPC client",
			zap.String("target", config.Target),
			zap.String("service", config.ServiceName))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		logger.Warn("mTLS disabled for gRPC client - using insecure connection",
			zap.String("target", config.Target))
	}

	// Connect to target
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", config.Target, err)
	}

	logger.Info("gRPC client connected",
		zap.String("target", config.Target),
		zap.String("service", config.ServiceName),
		zap.String("spiffe_id", config.SpiffeID))

	return &Client{
		conn:     conn,
		config:   config,
		logger:   logger,
		spiffeID: config.SpiffeID,
	}, nil
}

// loadTLSConfig loads TLS configuration for mTLS
func loadTLSConfig(config ClientConfig) (*tls.Config, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
		// Verify server certificate
		ServerName: config.ServiceName,
	}

	return tlsConfig, nil
}

// GetConn returns the underlying gRPC connection
func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}

// CallWithSpiffe makes a gRPC call with spiffe authentication
func (c *Client) CallWithSpiffe(ctx context.Context, method string, req interface{}, resp interface{}) error {
	// Add spiffe ID to context metadata
	md := metadata.Pairs("spiffe-id", c.spiffeID)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	// Make the call
	return c.conn.Invoke(ctx, method, req, resp)
}

// CallWithContext makes a gRPC call with custom context
func (c *Client) CallWithContext(ctx context.Context, method string, req interface{}, resp interface{}) error {
	// Add spiffe ID if not already present
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	if len(md.Get("spiffe-id")) == 0 {
		md = md.Copy()
		md.Set("spiffe-id", c.spiffeID)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	// Make the call
	return c.conn.Invoke(ctx, method, req, resp)
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		c.logger.Info("Closing gRPC client connection",
			zap.String("target", c.config.Target),
			zap.String("service", c.config.ServiceName))
		return c.conn.Close()
	}
	return nil
}

// IsHealthy checks if the connection is healthy
func (c *Client) IsHealthy(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() == "TRANSIENT_FAILURE" || state.String() == "SHUTDOWN" {
		return fmt.Errorf("connection is not healthy: %s", state.String())
	}
	return nil
}

// GetSpiffeID returns the spiffe ID of this client
func (c *Client) GetSpiffeID() string {
	return c.spiffeID
}

// ClientPool manages multiple gRPC clients for different services
type ClientPool struct {
	clients map[string]*Client
	logger  *zap.Logger
}

// NewClientPool creates a new client pool
func NewClientPool(logger *zap.Logger) *ClientPool {
	return &ClientPool{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// AddClient adds a client to the pool
func (p *ClientPool) AddClient(serviceName string, client *Client) {
	p.clients[serviceName] = client
	p.logger.Info("Added client to pool",
		zap.String("service", serviceName),
		zap.String("spiffe_id", client.GetSpiffeID()))
}

// GetClient retrieves a client from the pool
func (p *ClientPool) GetClient(serviceName string) (*Client, error) {
	client, ok := p.clients[serviceName]
	if !ok {
		return nil, fmt.Errorf("client not found for service: %s", serviceName)
	}
	return client, nil
}

// CloseAll closes all clients in the pool
func (p *ClientPool) CloseAll() error {
	var errs []error
	for serviceName, client := range p.clients {
		if err := client.Close(); err != nil {
			p.logger.Error("Failed to close client",
				zap.String("service", serviceName),
				zap.Error(err))
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close some clients: %v", errs)
	}
	return nil
}

// HealthCheck performs health check on all clients
func (p *ClientPool) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for serviceName, client := range p.clients {
		results[serviceName] = client.IsHealthy(ctx)
	}
	return results
}
