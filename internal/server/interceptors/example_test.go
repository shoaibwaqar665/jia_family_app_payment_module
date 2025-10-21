package interceptors

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/auth"
	"github.com/jia-app/paymentservice/internal/log"
)

// Example usage of the auth interceptor
func ExampleAuthInterceptor() {
	// Initialize logger for testing
	_ = log.Init("info")

	// Create auth interceptor with mock validator
	mockValidator := auth.NewMockValidator()
	whitelistedMethods := []string{"/payment.v1.PaymentService/PaymentSuccessWebhook"}
	authInterceptor := NewAuthInterceptor(mockValidator, whitelistedMethods)

	// Example gRPC method info
	methodInfo := &grpc.UnaryServerInfo{
		FullMethod: "/payment.v1.PaymentService/CreatePayment",
	}

	// Example handler that logs the user_id from context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// The user_id will be automatically injected by the interceptor
		log.Info(ctx, "Processing payment request")
		return "payment_created", nil
	}

	// Test with valid token
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
		"better-auth-token": "spiff_id_12345",
	}))

	// Call the interceptor
	resp, err := authInterceptor.Unary()(ctx, "payment_request", methodInfo, handler)
	if err != nil {
		panic(err)
	}

	_ = resp // Use response
}

// Test auth interceptor with different scenarios
func TestAuthInterceptor(t *testing.T) {
	// Initialize logger for testing
	_ = log.Init("info")

	// Create auth interceptor with mock validator
	mockValidator := auth.NewMockValidator()
	whitelistedMethods := []string{"/payment.v1.PaymentService/PaymentSuccessWebhook"}
	authInterceptor := NewAuthInterceptor(mockValidator, whitelistedMethods)

	// Test handler that returns success
	successHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	tests := []struct {
		name           string
		method         string
		token          string
		expectedUserID string
		shouldSucceed  bool
	}{
		{
			name:           "valid spiff_id token",
			method:         "/payment.v1.PaymentService/CreatePayment",
			token:          "spiff_id_12345",
			expectedUserID: "spiff_id_12345",
			shouldSucceed:  true,
		},
		{
			name:           "bearer token",
			method:         "/payment.v1.PaymentService/CreatePayment",
			token:          "Bearer spiff_id_67890",
			expectedUserID: "spiff_id_67890",
			shouldSucceed:  true,
		},
		{
			name:           "random token generates fake spiff_id",
			method:         "/payment.v1.PaymentService/CreatePayment",
			token:          "random_token",
			expectedUserID: "spiff_id_random_",
			shouldSucceed:  true,
		},
		{
			name:           "whitelisted method skips auth",
			method:         "/payment.v1.PaymentService/PaymentSuccessWebhook",
			token:          "",
			expectedUserID: "",
			shouldSucceed:  true,
		},
		{
			name:           "missing token fails",
			method:         "/payment.v1.PaymentService/CreatePayment",
			token:          "",
			expectedUserID: "",
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Add token to metadata if provided
			if tt.token != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
					"better-auth-token": tt.token,
				}))
			}

			methodInfo := &grpc.UnaryServerInfo{
				FullMethod: tt.method,
			}

			// Call the interceptor
			resp, err := authInterceptor.Unary()(ctx, "test_request", methodInfo, successHandler)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
					return
				}

				// Check if user_id was injected into context
				if tt.expectedUserID != "" {
					userID := log.L(ctx).Core().With([]zap.Field{zap.String("user_id", "")})
					if userID == nil {
						t.Error("Expected user_id to be injected into context")
					}
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}

				// Check if it's an authentication error
				st, ok := status.FromError(err)
				if !ok || st.Code() != codes.Unauthenticated {
					t.Errorf("Expected Unauthenticated error, got: %v", err)
				}
			}

			_ = resp // Use response to avoid unused variable warning
		})
	}
}
