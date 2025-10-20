package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/discovery"
	"github.com/jia-app/paymentservice/internal/shared/grpc"
)

// ContactServiceClient is a client for the Contact and Relationship service
type ContactServiceClient struct {
	client      *grpc.Client
	serviceName string
	discovery   *discovery.ServiceDiscovery
	logger      *zap.Logger
}

// NewContactServiceClient creates a new contact service client
func NewContactServiceClient(cfg *config.Config, discovery *discovery.ServiceDiscovery, logger *zap.Logger) (*ContactServiceClient, error) {
	// Get service configuration
	serviceConfig := cfg.ExternalServices.ContactService

	// Get target address from service discovery
	var target string
	if discovery != nil {
		target = discovery.GetTargetAddress(serviceConfig.Name)
	} else {
		target = serviceConfig.Address
	}

	// Create gRPC client configuration
	clientConfig := grpc.DefaultClientConfig()
	clientConfig.Target = target
	clientConfig.ServiceName = serviceConfig.Name
	clientConfig.EnableMTLS = cfg.MTLS.Enabled
	clientConfig.SpiffeID = cfg.ServiceMesh.SpiffeID
	clientConfig.CertFile = cfg.MTLS.CertFile
	clientConfig.KeyFile = cfg.MTLS.KeyFile
	clientConfig.CAFile = cfg.MTLS.CAFile
	clientConfig.RequestTimeout = time.Duration(serviceConfig.TimeoutSec) * time.Second
	clientConfig.Logger = logger

	// Create gRPC client
	client, err := grpc.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact service client: %w", err)
	}

	logger.Info("Contact service client created",
		zap.String("target", target),
		zap.String("spiffe_id", cfg.ServiceMesh.SpiffeID))

	return &ContactServiceClient{
		client:      client,
		serviceName: serviceConfig.Name,
		discovery:   discovery,
		logger:      logger,
	}, nil
}

// GetUserInfo retrieves user information from the contact service
func (c *ContactServiceClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	// Example gRPC call - adjust method name and request/response types based on actual proto
	req := &GetUserInfoRequest{
		UserID: userID,
	}

	resp := &GetUserInfoResponse{}

	// Make the gRPC call with spiffe authentication
	err := c.client.CallWithSpiffe(ctx, "/contact.v1.ContactService/GetUserInfo", req, resp)
	if err != nil {
		c.logger.Error("Failed to get user info",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	c.logger.Info("User info retrieved",
		zap.String("user_id", userID),
		zap.String("name", resp.User.Name))

	return resp.User, nil
}

// GetFamilyMembers retrieves family members for a family ID
func (c *ContactServiceClient) GetFamilyMembers(ctx context.Context, familyID string) ([]*UserInfo, error) {
	req := &GetFamilyMembersRequest{
		FamilyID: familyID,
	}

	resp := &GetFamilyMembersResponse{}

	err := c.client.CallWithSpiffe(ctx, "/contact.v1.ContactService/GetFamilyMembers", req, resp)
	if err != nil {
		c.logger.Error("Failed to get family members",
			zap.String("family_id", familyID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get family members: %w", err)
	}

	c.logger.Info("Family members retrieved",
		zap.String("family_id", familyID),
		zap.Int("count", len(resp.Members)))

	return resp.Members, nil
}

// Close closes the client connection
func (c *ContactServiceClient) Close() error {
	return c.client.Close()
}

// IsHealthy checks if the service is healthy
func (c *ContactServiceClient) IsHealthy(ctx context.Context) error {
	return c.client.IsHealthy(ctx)
}

// Request/Response types (replace with actual proto-generated types)
type GetUserInfoRequest struct {
	UserID string `json:"user_id"`
}

type GetUserInfoResponse struct {
	User *UserInfo `json:"user"`
}

type GetFamilyMembersRequest struct {
	FamilyID string `json:"family_id"`
}

type GetFamilyMembersResponse struct {
	Members []*UserInfo `json:"members"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	FamilyID string `json:"family_id,omitempty"`
}
