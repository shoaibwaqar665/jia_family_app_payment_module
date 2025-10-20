package services

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/discovery"
	"github.com/jia-app/paymentservice/internal/shared/grpc"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// ServiceManager manages all external service clients
type ServiceManager struct {
	config         *config.Config
	discovery      *discovery.ServiceDiscovery
	clientPool     *grpc.ClientPool
	contactService *ContactServiceClient
	logger         *zap.Logger
	mu             sync.RWMutex
	initialized    bool
}

// NewServiceManager creates a new service manager
func NewServiceManager(cfg *config.Config, logger *zap.Logger) (*ServiceManager, error) {
	if logger == nil {
		logger = log.L(context.Background())
	}

	// Initialize service discovery if enabled
	var sd *discovery.ServiceDiscovery
	if cfg.ServiceMesh.Enabled {
		discoveryConfig := discovery.DefaultServiceConfig()
		discoveryConfig.EnvoyAddress = cfg.ServiceMesh.Discovery.EnvoyAddress
		discoveryConfig.Namespace = cfg.ServiceMesh.Discovery.Namespace
		discoveryConfig.Scheme = cfg.ServiceMesh.Discovery.Scheme
		discoveryConfig.Logger = logger

		var err error
		sd, err = discovery.NewServiceDiscovery(discoveryConfig)
		if err != nil {
			logger.Error("Failed to initialize service discovery",
				zap.Error(err))
			// Continue without service discovery
			sd = nil
		}
	}

	clientPool := grpc.NewClientPool(logger)

	return &ServiceManager{
		config:     cfg,
		discovery:  sd,
		clientPool: clientPool,
		logger:     logger,
	}, nil
}

// Initialize initializes all service clients
func (sm *ServiceManager) Initialize(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.initialized {
		return nil
	}

	sm.logger.Info("Initializing service manager...")

	// Initialize contact service client
	if err := sm.initializeContactService(ctx); err != nil {
		sm.logger.Error("Failed to initialize contact service",
			zap.Error(err))
		// Continue with other services
	}

	sm.initialized = true
	sm.logger.Info("Service manager initialized successfully")

	return nil
}

// initializeContactService initializes the contact service client
func (sm *ServiceManager) initializeContactService(ctx context.Context) error {
	client, err := NewContactServiceClient(sm.config, sm.discovery, sm.logger)
	if err != nil {
		return fmt.Errorf("failed to create contact service client: %w", err)
	}

	sm.contactService = client
	sm.clientPool.AddClient("contact-service", client.client)

	sm.logger.Info("Contact service client initialized")
	return nil
}

// GetContactService returns the contact service client
func (sm *ServiceManager) GetContactService() (*ContactServiceClient, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.contactService == nil {
		return nil, fmt.Errorf("contact service client not initialized")
	}

	return sm.contactService, nil
}

// GetClientPool returns the gRPC client pool
func (sm *ServiceManager) GetClientPool() *grpc.ClientPool {
	return sm.clientPool
}

// GetDiscovery returns the service discovery instance
func (sm *ServiceManager) GetDiscovery() *discovery.ServiceDiscovery {
	return sm.discovery
}

// HealthCheck performs health check on all services
func (sm *ServiceManager) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// Check contact service
	if sm.contactService != nil {
		results["contact-service"] = sm.contactService.IsHealthy(ctx)
	}

	// Check all clients in pool
	poolResults := sm.clientPool.HealthCheck(ctx)
	for service, err := range poolResults {
		if err != nil {
			results[service] = err
		}
	}

	return results
}

// Close closes all service clients
func (sm *ServiceManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.logger.Info("Closing service manager...")

	var errs []error

	// Close contact service
	if sm.contactService != nil {
		if err := sm.contactService.Close(); err != nil {
			sm.logger.Error("Failed to close contact service",
				zap.Error(err))
			errs = append(errs, err)
		}
	}

	// Close all clients in pool
	if err := sm.clientPool.CloseAll(); err != nil {
		sm.logger.Error("Failed to close client pool",
			zap.Error(err))
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close some services: %v", errs)
	}

	sm.logger.Info("Service manager closed successfully")
	return nil
}

// IsInitialized returns whether the service manager is initialized
func (sm *ServiceManager) IsInitialized() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.initialized
}
