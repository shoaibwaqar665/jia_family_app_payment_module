package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// GRPCServer represents a gRPC server
type GRPCServer struct {
	server *grpc.Server
	port   int
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer(port int) *GRPCServer {
	// Create gRPC server with middleware
	server := grpc.NewServer()
	
	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	
	// Register reflection service for development
	reflection.Register(server)
	
	// Set serving status
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	
	return &GRPCServer{
		server: server,
		port:   port,
	}
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	
	fmt.Printf("gRPC server listening on port %d\n", s.port)
	
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	
	return nil
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// GetServer returns the underlying gRPC server
func (s *GRPCServer) GetServer() *grpc.Server {
	return s.server
}
