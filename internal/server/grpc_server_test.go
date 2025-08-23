package server

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/jia-app/paymentservice/internal/config"
)

func TestNewGRPCServer(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		GRPC: config.GRPCConfig{
			Address: ":0", // Use port 0 for testing
		},
	}

	// Test with nil dependencies
	server := NewGRPCServer(cfg, nil, nil)
	
	if server == nil {
		t.Fatal("Expected server to be created")
	}
	
	if server.server == nil {
		t.Fatal("Expected gRPC server to be created")
	}
	
	if server.healthServer == nil {
		t.Fatal("Expected health server to be created")
	}
	
	if server.config != cfg {
		t.Fatal("Expected config to be set")
	}
}

func TestHealthMonitoring(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		GRPC: config.GRPCConfig{
			Address: ":0",
		},
	}

	// Create server with nil dependencies
	server := NewGRPCServer(cfg, nil, nil)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Start health monitoring
	server.StartHealthMonitoring(ctx)
	
	// Wait a bit for health check to run
	time.Sleep(50 * time.Millisecond)
	
	// Check that health status is NOT_SERVING (since dependencies are nil)
	resp, err := server.healthServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Errorf("Expected health status to be NOT_SERVING, got %v", resp.Status)
	}
}

func TestHealthCheckMethods(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		GRPC: config.GRPCConfig{
			Address: ":0",
		},
	}

	// Create server with nil dependencies
	server := NewGRPCServer(cfg, nil, nil)
	
	// Test database health check with nil pool
	dbHealthy := server.checkDatabase(context.Background())
	if dbHealthy {
		t.Error("Expected database health check to fail with nil pool")
	}
	
	// Test Redis health check with nil client
	redisHealthy := server.checkRedis(context.Background())
	if redisHealthy {
		t.Error("Expected Redis health check to fail with nil client")
	}
	
	// Test dependency check
	server.checkDependencies()
	
	// Should still be NOT_SERVING since both dependencies are unhealthy
	resp, err := server.healthServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Errorf("Expected health status to be NOT_SERVING, got %v", resp.Status)
	}
}

func TestGRPCServerMethods(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		GRPC: config.GRPCConfig{
			Address: ":0",
		},
	}

	// Create server
	server := NewGRPCServer(cfg, nil, nil)
	
	// Test GetServer method
	grpcServer := server.GetServer()
	if grpcServer == nil {
		t.Fatal("Expected gRPC server to be returned")
	}
	
	// Test RegisterService method exists (we can't test with nil values as it would panic)
	// The method should exist and be callable with proper parameters
	// Note: We can't easily test the actual registration without proto definitions
}
