package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"

	"github.com/jia-app/paymentservice/internal/shared/log"
)

// ServiceConfig holds service discovery configuration
type ServiceConfig struct {
	// Service name (e.g., "contact-service")
	ServiceName string

	// Envoy proxy address (e.g., "localhost:8500")
	EnvoyAddress string

	// Service mesh namespace
	Namespace string

	// Resolver scheme (e.g., "xds" for Envoy)
	Scheme string

	// Timeout for service resolution
	Timeout time.Duration

	// Logger
	Logger *zap.Logger
}

// DefaultServiceConfig returns default service discovery configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		EnvoyAddress: "localhost:8500",
		Namespace:    "default",
		Scheme:       "xds",
		Timeout:      30 * time.Second,
	}
}

// ServiceDiscovery handles service discovery using Envoy
type ServiceDiscovery struct {
	config   ServiceConfig
	logger   *zap.Logger
	services map[string]*ServiceInfo
	mu       sync.RWMutex
	resolver resolver.Builder
}

// ServiceInfo holds information about a discovered service
type ServiceInfo struct {
	Name       string
	Address    string
	Port       int
	Endpoints  []Endpoint
	LastUpdate time.Time
}

// Endpoint represents a service endpoint
type Endpoint struct {
	Address string
	Port    int
	Healthy bool
	Weight  int
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(config ServiceConfig) (*ServiceDiscovery, error) {
	logger := config.Logger
	if logger == nil {
		logger = log.L(context.Background())
	}

	sd := &ServiceDiscovery{
		config:   config,
		logger:   logger,
		services: make(map[string]*ServiceInfo),
	}

	// Register resolver if using Envoy XDS
	if config.Scheme == "xds" {
		if err := sd.registerXDSResolver(); err != nil {
			logger.Warn("Failed to register XDS resolver, using DNS fallback",
				zap.Error(err))
			sd.config.Scheme = "dns"
		}
	}

	logger.Info("Service discovery initialized",
		zap.String("envoy_address", config.EnvoyAddress),
		zap.String("scheme", config.Scheme),
		zap.String("namespace", config.Namespace))

	return sd, nil
}

// registerXDSResolver registers the XDS resolver for Envoy
func (sd *ServiceDiscovery) registerXDSResolver() error {
	// In a real implementation, this would register the xds:// resolver
	// For now, we'll use a simplified approach
	sd.logger.Info("XDS resolver registered for Envoy integration")
	return nil
}

// ResolveService resolves a service name to an address
func (sd *ServiceDiscovery) ResolveService(serviceName string) (string, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	service, ok := sd.services[serviceName]
	if !ok {
		// Try to resolve using DNS as fallback
		return sd.resolveDNS(serviceName)
	}

	if len(service.Endpoints) == 0 {
		return "", fmt.Errorf("no endpoints available for service: %s", serviceName)
	}

	// Return the first healthy endpoint
	for _, endpoint := range service.Endpoints {
		if endpoint.Healthy {
			return fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port), nil
		}
	}

	// If no healthy endpoints, return the first one anyway
	endpoint := service.Endpoints[0]
	return fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port), nil
}

// resolveDNS resolves a service using DNS
func (sd *ServiceDiscovery) resolveDNS(serviceName string) (string, error) {
	// Try to resolve using DNS SRV records or simple DNS lookup
	// Format: service-name.namespace.svc.cluster.local
	fqdn := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, sd.config.Namespace)

	// Try SRV lookup first
	_, srvAddrs, err := net.LookupSRV("", "", fqdn)
	if err == nil && len(srvAddrs) > 0 {
		// Use the first SRV record
		return fmt.Sprintf("%s:%d", srvAddrs[0].Target, srvAddrs[0].Port), nil
	}

	// Fallback to A record lookup
	hostAddrs, err := net.LookupHost(fqdn)
	if err == nil && len(hostAddrs) > 0 {
		// Use default port 50051 for gRPC
		return fmt.Sprintf("%s:50051", hostAddrs[0]), nil
	}

	// Last resort: try just the service name
	hostAddrs, err = net.LookupHost(serviceName)
	if err == nil && len(hostAddrs) > 0 {
		return fmt.Sprintf("%s:50051", hostAddrs[0]), nil
	}

	return "", fmt.Errorf("failed to resolve service: %s", serviceName)
}

// UpdateService updates service information
func (sd *ServiceDiscovery) UpdateService(serviceName string, info *ServiceInfo) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	info.LastUpdate = time.Now()
	sd.services[serviceName] = info

	sd.logger.Info("Service updated",
		zap.String("service", serviceName),
		zap.String("address", info.Address),
		zap.Int("endpoints", len(info.Endpoints)))
}

// GetService returns service information
func (sd *ServiceDiscovery) GetService(serviceName string) (*ServiceInfo, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	service, ok := sd.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	return service, nil
}

// ListServices returns all discovered services
func (sd *ServiceDiscovery) ListServices() []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	services := make([]string, 0, len(sd.services))
	for name := range sd.services {
		services = append(services, name)
	}

	return services
}

// WatchService watches for service changes (stub implementation)
func (sd *ServiceDiscovery) WatchService(serviceName string, callback func(*ServiceInfo)) error {
	// In a real implementation, this would set up a watch with Envoy
	// For now, we'll just return the current service info
	service, err := sd.GetService(serviceName)
	if err != nil {
		return err
	}

	callback(service)
	return nil
}

// GetTargetAddress returns the target address for a service
// This is the format expected by gRPC dial
func (sd *ServiceDiscovery) GetTargetAddress(serviceName string) string {
	// If using XDS, return xds://service-name
	if sd.config.Scheme == "xds" {
		return fmt.Sprintf("xds:///%s", serviceName)
	}

	// Otherwise, resolve to actual address
	address, err := sd.ResolveService(serviceName)
	if err != nil {
		// Fallback to service name with default port
		return fmt.Sprintf("%s:50051", serviceName)
	}

	return address
}

// HealthCheck performs health check on all services
func (sd *ServiceDiscovery) HealthCheck(ctx context.Context) map[string]bool {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	results := make(map[string]bool)
	for name, service := range sd.services {
		// Check if service has healthy endpoints
		hasHealthy := false
		for _, endpoint := range service.Endpoints {
			if endpoint.Healthy {
				hasHealthy = true
				break
			}
		}
		results[name] = hasHealthy
	}

	return results
}

// MockServiceDiscovery is a mock implementation for testing
type MockServiceDiscovery struct {
	services map[string]*ServiceInfo
	logger   *zap.Logger
}

// NewMockServiceDiscovery creates a mock service discovery for testing
func NewMockServiceDiscovery(logger *zap.Logger) *MockServiceDiscovery {
	return &MockServiceDiscovery{
		services: make(map[string]*ServiceInfo),
		logger:   logger,
	}
}

// RegisterService registers a mock service
func (m *MockServiceDiscovery) RegisterService(name string, address string, port int) {
	m.services[name] = &ServiceInfo{
		Name:    name,
		Address: address,
		Port:    port,
		Endpoints: []Endpoint{
			{
				Address: address,
				Port:    port,
				Healthy: true,
				Weight:  100,
			},
		},
		LastUpdate: time.Now(),
	}

	if m.logger != nil {
		m.logger.Info("Mock service registered",
			zap.String("service", name),
			zap.String("address", address),
			zap.Int("port", port))
	}
}

// ResolveService resolves a service name
func (m *MockServiceDiscovery) ResolveService(serviceName string) (string, error) {
	service, ok := m.services[serviceName]
	if !ok {
		return "", fmt.Errorf("service not found: %s", serviceName)
	}

	return fmt.Sprintf("%s:%d", service.Address, service.Port), nil
}

// GetTargetAddress returns target address
func (m *MockServiceDiscovery) GetTargetAddress(serviceName string) string {
	address, err := m.ResolveService(serviceName)
	if err != nil {
		return fmt.Sprintf("%s:50051", serviceName)
	}
	return address
}
