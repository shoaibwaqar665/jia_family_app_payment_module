package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/domain"
	paymentv1 "github.com/jia-app/paymentservice/proto/payment/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PaymentGRPCHandler wraps EnhancedPaymentService and implements the proto PaymentServiceServer interface
type PaymentGRPCHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	service *EnhancedPaymentService
}

// NewPaymentGRPCHandler creates a new gRPC handler
func NewPaymentGRPCHandler(service *EnhancedPaymentService) *PaymentGRPCHandler {
	return &PaymentGRPCHandler{
		service: service,
	}
}

// CreatePayment implements PaymentServiceServer.CreatePayment
func (h *PaymentGRPCHandler) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	// Convert proto request to domain request
	paymentReq := &domain.PaymentRequest{
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		CustomerID:    req.CustomerId,
		OrderID:       req.OrderId,
		Description:   req.Description,
	}

	// Use EnhancedPaymentService to create payment
	payment, err := h.service.CreatePayment(ctx, paymentReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	// Convert domain payment to proto response
	return &paymentv1.CreatePaymentResponse{
		Payment: toProtoPayment(payment),
	}, nil
}

// GetPayment implements PaymentServiceServer.GetPayment
func (h *PaymentGRPCHandler) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	// Parse payment ID
	paymentID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment ID format")
	}

	// Get payment using EnhancedPaymentService
	payment, err := h.service.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &paymentv1.GetPaymentResponse{
		Payment: toProtoPayment(payment),
	}, nil
}

// UpdatePaymentStatus implements PaymentServiceServer.UpdatePaymentStatus
func (h *PaymentGRPCHandler) UpdatePaymentStatus(ctx context.Context, req *paymentv1.UpdatePaymentStatusRequest) (*paymentv1.UpdatePaymentStatusResponse, error) {
	// Parse payment ID
	paymentID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment ID format")
	}

	// Update payment status using EnhancedPaymentService
	err = h.service.UpdatePaymentStatus(ctx, paymentID, req.Status)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &paymentv1.UpdatePaymentStatusResponse{}, nil
}

// GetPaymentsByCustomer implements PaymentServiceServer.GetPaymentsByCustomer
func (h *PaymentGRPCHandler) GetPaymentsByCustomer(ctx context.Context, req *paymentv1.GetPaymentsByCustomerRequest) (*paymentv1.GetPaymentsByCustomerResponse, error) {
	// Parse customer ID
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer ID format")
	}

	// Set default pagination parameters
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Get payments by customer using EnhancedPaymentService with pagination
	payments, total, err := h.service.GetPaymentsByCustomer(ctx, customerID, limit, offset)
	if err != nil {
		return nil, h.handleError(err)
	}

	// Convert domain payments to proto payments
	protoPayments := make([]*paymentv1.Payment, len(payments))
	for i, payment := range payments {
		protoPayments[i] = toProtoPayment(payment)
	}

	return &paymentv1.GetPaymentsByCustomerResponse{
		Payments: protoPayments,
		Total:    int32(total),
	}, nil
}

// ListPayments implements PaymentServiceServer.ListPayments
func (h *PaymentGRPCHandler) ListPayments(ctx context.Context, req *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	// Set default pagination parameters
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// List payments using EnhancedPaymentService
	payments, total, err := h.service.ListPayments(ctx, limit, offset)
	if err != nil {
		return nil, h.handleError(err)
	}

	// Convert domain payments to proto payments
	protoPayments := make([]*paymentv1.Payment, len(payments))
	for i, payment := range payments {
		protoPayments[i] = toProtoPayment(payment)
	}

	return &paymentv1.ListPaymentsResponse{
		Payments: protoPayments,
		Total:    int32(total),
	}, nil
}

// handleError safely handles errors and returns appropriate gRPC status
func (h *PaymentGRPCHandler) handleError(err error) error {
	if err == nil {
		return nil
	}

	// Sanitize the error to prevent information disclosure
	sanitizedErr := domain.SanitizeError(err)

	// Safely convert to GRPCError
	if grpcErr, ok := sanitizedErr.(*domain.GRPCError); ok {
		return grpcErr.ToGRPCStatus().Err()
	}

	// Fallback to internal error if conversion fails
	return status.Error(codes.Internal, "Internal server error")
}

// toProtoPayment converts a domain Payment to proto Payment
func toProtoPayment(payment *domain.Payment) *paymentv1.Payment {
	return &paymentv1.Payment{
		Id:            payment.ID.String(),
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		CustomerId:    payment.CustomerID,
		OrderId:       payment.OrderID,
		Description:   payment.Description,
		CreatedAt:     timestamppb.New(payment.CreatedAt),
		UpdatedAt:     timestamppb.New(payment.UpdatedAt),
	}
}
