package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	paymentv1 "github.com/jia-app/paymentservice/api/payment/v1"
)

const (
	grpcAddress = "localhost:8081"
	httpPort    = ":8082"
)

type CheckoutRequest struct {
	PlanID      string  `json:"plan_id"`
	UserID      string  `json:"user_id"`
	FamilyID    string  `json:"family_id,omitempty"`
	CountryCode string  `json:"country_code"`
	BasePrice   float64 `json:"base_price"`
	Currency    string  `json:"currency"`
	SuccessURL  string  `json:"success_url"`
	CancelURL   string  `json:"cancel_url"`
}

type CheckoutResponse struct {
	SessionID string    `json:"session_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type WebhookRequest struct {
	Payload   interface{} `json:"payload"`
	Signature string      `json:"signature"`
	Provider  string      `json:"provider"`
}

type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type CreateEntitlementRequest struct {
	UserID         string                 `json:"user_id"`
	FamilyID       string                 `json:"family_id,omitempty"`
	FeatureCode    string                 `json:"feature_code"`
	PlanID         string                 `json:"plan_id"`
	SubscriptionID string                 `json:"subscription_id,omitempty"`
	Status         string                 `json:"status"`
	GrantedAt      string                 `json:"granted_at"`
	ExpiresAt      *string                `json:"expires_at,omitempty"`
	UsageLimits    map[string]interface{} `json:"usage_limits,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type CreateEntitlementResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	EntitlementID string `json:"entitlement_id,omitempty"`
}

func main() {
	// Create gRPC connection
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := paymentv1.NewPaymentServiceClient(conn)

	// HTTP handlers
	http.HandleFunc("/api/checkout", func(w http.ResponseWriter, r *http.Request) {
		handleCheckout(w, r, client)
	})

	http.HandleFunc("/api/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(w, r, client)
	})

	http.HandleFunc("/api/payments", func(w http.ResponseWriter, r *http.Request) {
		handleListPayments(w, r, client)
	})

	http.HandleFunc("/api/simulate-payment", func(w http.ResponseWriter, r *http.Request) {
		handleSimulatePayment(w, r, client)
	})

	http.HandleFunc("/api/entitlements", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleListEntitlements(w, r, client)
		} else if r.Method == http.MethodPost {
			handleCreateEntitlement(w, r, client)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/pricing-zones", func(w http.ResponseWriter, r *http.Request) {
		handleListPricingZones(w, r, client)
	})

	// Handle payment success page
	http.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		handlePaymentSuccess(w, r, client)
	})

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Printf("HTTP server starting on port %s", httpPort)
	log.Printf("Serving POC UI at http://localhost%s", httpPort)
	log.Fatal(http.ListenAndServe(httpPort, nil))
}

func handleCheckout(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert string plan ID to deterministic UUID (same logic as backend)
	planID := req.PlanID
	if parsedUUID, err := uuid.Parse(req.PlanID); err != nil {
		// It's a string plan ID, generate a deterministic UUID
		planID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(req.PlanID)).String()
	} else {
		// It's already a UUID
		planID = parsedUUID.String()
	}

	// Convert to gRPC request
	grpcReq := &paymentv1.CreateCheckoutSessionRequest{
		PlanId:      planID,
		UserId:      req.UserID,
		FamilyId:    req.FamilyID,
		CountryCode: req.CountryCode,
		BasePrice:   req.BasePrice,
		Currency:    req.Currency,
		SuccessUrl:  req.SuccessURL,
		CancelUrl:   req.CancelURL,
	}

	// Call gRPC service with auth token
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add auth token to gRPC metadata
	md := metadata.New(map[string]string{
		"better-auth-token": r.Header.Get("better-auth-token"),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := client.CreateCheckoutSession(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create checkout session: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert response
	response := CheckoutResponse{
		SessionID: resp.SessionId,
		URL:       resp.Url,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleWebhook(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert payload to bytes
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		http.Error(w, "Failed to marshal payload", http.StatusBadRequest)
		return
	}

	// Convert to gRPC request
	grpcReq := &paymentv1.ProcessWebhookRequest{
		Payload:   payloadBytes,
		Signature: req.Signature,
		Provider:  req.Provider,
	}

	// Call gRPC service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ProcessWebhook(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to process webhook: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert response
	response := WebhookResponse{
		Success: resp.Success,
		Message: resp.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleListPayments(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int32(10) // default
	offset := int32(0) // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = int32(l)
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = int32(o)
		}
	}

	// Convert to gRPC request
	grpcReq := &paymentv1.ListPaymentsRequest{
		Limit:  limit,
		Offset: offset,
	}

	// Call gRPC service with auth token
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add auth token to gRPC metadata
	md := metadata.New(map[string]string{
		"better-auth-token": r.Header.Get("better-auth-token"),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := client.ListPayments(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list payments: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleSimulatePayment(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		UserID    string `json:"user_id"`
		PlanID    string `json:"plan_id"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simulate a Stripe webhook payload
	webhookPayload := fmt.Sprintf(`{
		"id": "evt_test_webhook",
		"object": "event",
		"type": "checkout.session.completed",
		"data": {
			"object": {
				"id": "%s",
				"object": "checkout.session",
				"payment_status": "paid",
				"metadata": {
					"user_id": "%s",
					"plan_id": "%s",
					"base_price": "%d",
					"currency": "%s"
				}
			}
		}
	}`, req.SessionID, req.UserID, req.PlanID, req.Amount, req.Currency)

	// Process the webhook
	grpcReq := &paymentv1.ProcessWebhookRequest{
		Payload:   []byte(webhookPayload),
		Signature: "test_signature",
		Provider:  "stripe",
	}

	// Call gRPC service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ProcessWebhook(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to process webhook: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleListEntitlements(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from query parameter
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	// Get limit and offset from query parameters
	limit := int32(10)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.ParseInt(l, 10, 32); err == nil {
			limit = int32(parsedLimit)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.ParseInt(o, 10, 32); err == nil {
			offset = int32(parsedOffset)
		}
	}

	// Call gRPC service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add authentication metadata
	md := metadata.New(map[string]string{
		"better-auth-token": r.Header.Get("better-auth-token"),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	grpcReq := &paymentv1.ListEntitlementsRequest{
		UserId: userID,
		Limit:  limit,
		Offset: offset,
	}

	resp, err := client.ListEntitlements(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list entitlements: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCreateEntitlement(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateEntitlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.FeatureCode == "" || req.PlanID == "" {
		http.Error(w, "Missing required fields: user_id, feature_code, plan_id", http.StatusBadRequest)
		return
	}

	// Create a webhook payload to simulate entitlement creation
	webhookPayload := map[string]interface{}{
		"session_id":   req.SubscriptionID,
		"user_id":      req.UserID,
		"plan_id":      req.PlanID,
		"feature_code": req.FeatureCode,
		"amount":       19.99, // Default amount
		"currency":     "USD",
		"status":       req.Status,
		"expires_at":   req.ExpiresAt,
		"metadata": map[string]interface{}{
			"family_id":      req.FamilyID,
			"country_code":   "US",
			"base_price":     19.99,
			"adjusted_price": 19.99,
		},
	}

	// Convert payload to bytes
	payloadBytes, err := json.Marshal(webhookPayload)
	if err != nil {
		http.Error(w, "Failed to marshal webhook payload", http.StatusInternalServerError)
		return
	}

	// Call gRPC service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add authentication metadata
	md := metadata.New(map[string]string{
		"better-auth-token": r.Header.Get("better-auth-token"),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Convert to gRPC request
	grpcReq := &paymentv1.ProcessWebhookRequest{
		Payload:   payloadBytes,
		Signature: "direct_entitlement_creation",
		Provider:  "stripe",
	}

	resp, err := client.ProcessWebhook(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create entitlement: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert response
	response := CreateEntitlementResponse{
		Success:       resp.Success,
		Message:       resp.Message,
		EntitlementID: "created_via_webhook", // We don't have the actual ID from webhook
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePaymentSuccess(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	// Get session_id from query parameters
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id parameter", http.StatusBadRequest)
		return
	}

	// Get user_id from query parameters or auth token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.Header.Get("better-auth-token")
	}
	if userID == "" {
		userID = "test_user_123" // Default for POC
	}

	// Create a webhook payload to simulate payment completion
	webhookPayload := map[string]interface{}{
		"session_id":   sessionID,
		"user_id":      userID,
		"plan_id":      "pro_monthly", // Default plan for POC
		"feature_code": "pro_storage",
		"amount":       19.99,
		"currency":     "USD",
		"status":       "completed",
		"expires_at":   nil,
		"metadata": map[string]interface{}{
			"family_id":      "success_page_family",
			"country_code":   "US",
			"base_price":     19.99,
			"adjusted_price": 19.99,
		},
	}

	// Convert payload to bytes
	payloadBytes, err := json.Marshal(webhookPayload)
	if err != nil {
		http.Error(w, "Failed to marshal webhook payload", http.StatusInternalServerError)
		return
	}

	// Call gRPC service to process webhook
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add authentication metadata
	md := metadata.New(map[string]string{
		"better-auth-token": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Convert to gRPC request
	grpcReq := &paymentv1.ProcessWebhookRequest{
		Payload:   payloadBytes,
		Signature: "success_page_signature_" + sessionID,
		Provider:  "stripe",
	}

	_, err = client.ProcessWebhook(ctx, grpcReq)
	if err != nil {
		log.Printf("Webhook processing error: %v", err)
		// Still show success page even if webhook fails
	}

	// Serve success page with automatic redirect to entitlements
	successPageHTML := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Payment Successful - Jia Family App</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            margin: 0;
            padding: 0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .success-container {
            background: white;
            padding: 40px;
            border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
            text-align: center;
            max-width: 500px;
            width: 90%%;
        }
        .success-icon {
            font-size: 4rem;
            color: #48bb78;
            margin-bottom: 20px;
        }
        .success-title {
            font-size: 2rem;
            color: #2d3748;
            margin-bottom: 15px;
        }
        .success-message {
            font-size: 1.1rem;
            color: #718096;
            margin-bottom: 30px;
        }
        .redirect-message {
            font-size: 0.9rem;
            color: #a0aec0;
            margin-bottom: 20px;
        }
        .btn {
            background: #667eea;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            cursor: pointer;
            text-decoration: none;
            display: inline-block;
            margin: 0 10px;
        }
        .btn:hover {
            background: #5a67d8;
        }
        .btn-secondary {
            background: #e2e8f0;
            color: #4a5568;
        }
        .btn-secondary:hover {
            background: #cbd5e0;
        }
    </style>
</head>
<body>
    <div class="success-container">
        <div class="success-icon">✅</div>
        <h1 class="success-title">Payment Successful!</h1>
        <p class="success-message">Your payment has been processed successfully and your entitlement has been created.</p>
        <p class="redirect-message">Redirecting to your entitlements in <span id="countdown">3</span> seconds...</p>
        <div>
            <a href="/" class="btn btn-secondary">Back to Plans</a>
            <a href="/" onclick="showEntitlements(); return false;" class="btn">View Entitlements</a>
        </div>
    </div>

    <script>
        // Countdown timer
        let countdown = 3;
        const countdownElement = document.getElementById('countdown');
        
        const timer = setInterval(() => {
            countdown--;
            countdownElement.textContent = countdown;
            
            if (countdown <= 0) {
                clearInterval(timer);
                // Redirect to entitlements page with a flag
                window.location.href = '/?show_entitlements=true';
            }
        }, 1000);

        // Function to show entitlements
        function showEntitlements() {
            window.location.href = '/?show_entitlements=true';
        }
    </script>
</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(successPageHTML))
}

func handleListPricingZones(w http.ResponseWriter, r *http.Request, client paymentv1.PaymentServiceClient) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call gRPC service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add authentication metadata
	md := metadata.New(map[string]string{
		"better-auth-token": r.Header.Get("better-auth-token"),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	grpcReq := &paymentv1.ListPricingZonesRequest{}

	resp, err := client.ListPricingZones(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list pricing zones: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
