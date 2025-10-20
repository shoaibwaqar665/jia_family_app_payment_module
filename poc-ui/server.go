package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	paymentv1 "github.com/jia-app/paymentservice/api/payment/v1"
)

const (
	grpcAddress = "localhost:8081"
	httpPort    = ":8082"
	dbHost      = "ep-wild-wave-a1nsn7ul.ap-southeast-1.aws.neon.tech"
	dbPort      = 5432
	dbUser      = "neondb_owner"
	dbPassword  = "sLdJyF0w2Unv"
	dbName      = "jia_family_app"
	dbSslMode   = "require"
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

// Admin request/response types
type UpdatePlanRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	FeatureCodes []string `json:"feature_codes,omitempty"`
	BillingCycle string   `json:"billing_cycle,omitempty"`
	PriceCents   int32    `json:"price_cents,omitempty"`
	Currency     string   `json:"currency,omitempty"`
	MaxUsers     int32    `json:"max_users,omitempty"`
	Active       bool     `json:"active,omitempty"`
}

type CreatePricingZoneRequest struct {
	Country                 string  `json:"country"`
	ISOCode                 string  `json:"iso_code"`
	Zone                    string  `json:"zone"`
	ZoneName                string  `json:"zone_name"`
	WorldBankClassification string  `json:"world_bank_classification"`
	GNIPerCapitaThreshold   string  `json:"gni_per_capita_threshold"`
	PricingMultiplier       float64 `json:"pricing_multiplier"`
}

type UpdatePricingZoneRequest struct {
	ID                      string  `json:"id"`
	Country                 string  `json:"country,omitempty"`
	ISOCode                 string  `json:"iso_code,omitempty"`
	Zone                    string  `json:"zone,omitempty"`
	ZoneName                string  `json:"zone_name,omitempty"`
	WorldBankClassification string  `json:"world_bank_classification,omitempty"`
	GNIPerCapitaThreshold   string  `json:"gni_per_capita_threshold,omitempty"`
	PricingMultiplier       float64 `json:"pricing_multiplier,omitempty"`
}

func main() {
	// Create database connection
	dbConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// Create gRPC connection
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := paymentv1.NewPaymentServiceClient(conn)

	// Public HTTP handlers
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

	// Admin HTTP handlers
	http.HandleFunc("/api/admin/plans", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleAdminListPlans(w, r, db)
		case http.MethodPost:
			handleAdminCreatePlan(w, r, db)
		case http.MethodPut:
			handleAdminUpdatePlan(w, r, db)
		case http.MethodDelete:
			handleAdminDeletePlan(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/admin/pricing-zones", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleAdminListPricingZones(w, r, db)
		case http.MethodPost:
			handleAdminCreatePricingZone(w, r, db)
		case http.MethodPut:
			handleAdminUpdatePricingZone(w, r, db)
		case http.MethodDelete:
			handleAdminDeletePricingZone(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/admin/purchases", func(w http.ResponseWriter, r *http.Request) {
		handleAdminListPurchases(w, r, db)
	})

	http.HandleFunc("/api/admin/entitlements", func(w http.ResponseWriter, r *http.Request) {
		handleAdminListEntitlements(w, r, db)
	})

	// Handle payment success page
	http.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		handlePaymentSuccess(w, r, client)
	})

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Printf("HTTP server starting on port %s", httpPort)
	log.Printf("Serving POC UI at http://localhost%s", httpPort)
	log.Println("Admin API available at:")
	log.Println("  GET/POST/PUT/DELETE /api/admin/plans")
	log.Println("  GET/POST/PUT/DELETE /api/admin/pricing-zones")
	log.Println("  GET /api/admin/purchases")
	log.Println("  GET /api/admin/entitlements")
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
	if req.UserID == "" || req.PlanID == "" {
		http.Error(w, "Missing required fields: user_id, plan_id", http.StatusBadRequest)
		return
	}

	// Create a webhook payload to simulate entitlement creation
	webhookPayload := map[string]interface{}{
		"session_id": req.SubscriptionID,
		"user_id":    req.UserID,
		"plan_id":    req.PlanID,
		"amount":     19.99, // Default amount
		"currency":   "USD",
		"status":     req.Status,
		"expires_at": req.ExpiresAt,
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

	// Get plan_id from query parameters
	planID := r.URL.Query().Get("plan_id")
	if planID == "" {
		planID = "pro_monthly" // Default plan for POC
	}

	// Get family_id from query parameters
	familyID := r.URL.Query().Get("family_id")
	if familyID == "" {
		familyID = userID // Use userID as family ID for individual plans
	}

	// Create a webhook payload to simulate payment completion
	webhookPayload := map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"plan_id":    planID, // Use dynamic plan ID
		"amount":     19.99,
		"currency":   "USD",
		"status":     "completed",
		"expires_at": nil,
		"metadata": map[string]interface{}{
			"family_id":      familyID, // Use dynamic family ID
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

// ==================== ADMIN HANDLERS ====================

// handleAdminListPlans - List all plans (admin view with all plans including inactive)
func handleAdminListPlans(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	query := `SELECT id, name, description, feature_codes, billing_cycle, price_cents, 
	          currency, max_users, usage_limits, metadata, active, created_at, updated_at 
	          FROM plans ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list plans: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	plans := []map[string]interface{}{}
	for rows.Next() {
		var id, name, currency string
		var description, billingCycle sql.NullString
		var featureCodes []string
		var priceCents, maxUsers sql.NullInt32
		var usageLimits, metadata []byte
		var active bool
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &name, &description, pq.Array(&featureCodes), &billingCycle, &priceCents,
			&currency, &maxUsers, &usageLimits, &metadata, &active, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		plan := map[string]interface{}{
			"id":         id,
			"name":       name,
			"currency":   currency,
			"active":     active,
			"created_at": createdAt,
			"updated_at": updatedAt,
		}

		if description.Valid {
			plan["description"] = description.String
		}
		if billingCycle.Valid {
			plan["billing_cycle"] = billingCycle.String
		}
		if priceCents.Valid {
			plan["price_cents"] = priceCents.Int32
			plan["price_dollars"] = float64(priceCents.Int32) / 100.0
		}
		if maxUsers.Valid {
			plan["max_users"] = maxUsers.Int32
		}
		if featureCodes != nil {
			plan["feature_codes"] = featureCodes
		}

		plans = append(plans, plan)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plans": plans,
		"total": len(plans),
	})
}

// handleAdminCreatePlan - Create a new plan
func handleAdminCreatePlan(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		FeatureCodes []string `json:"feature_codes"`
		BillingCycle string   `json:"billing_cycle"`
		PriceCents   int32    `json:"price_cents"`
		Currency     string   `json:"currency"`
		MaxUsers     int32    `json:"max_users"`
		Active       bool     `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO plans (id, name, description, feature_codes, billing_cycle, 
	          price_cents, currency, max_users, active) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	var id string
	err := db.QueryRow(query, req.ID, req.Name, req.Description, pq.Array(req.FeatureCodes), req.BillingCycle,
		req.PriceCents, req.Currency, req.MaxUsers, req.Active).Scan(&id)

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create plan: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Plan created successfully",
		"id":      id,
	})
}

// handleAdminUpdatePlan - Update an existing plan
func handleAdminUpdatePlan(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, req.Name)
		argPos++
	}
	if req.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argPos))
		args = append(args, req.Description)
		argPos++
	}
	if len(req.FeatureCodes) > 0 {
		updates = append(updates, fmt.Sprintf("feature_codes = $%d", argPos))
		args = append(args, pq.Array(req.FeatureCodes))
		argPos++
	}
	if req.BillingCycle != "" {
		updates = append(updates, fmt.Sprintf("billing_cycle = $%d", argPos))
		args = append(args, req.BillingCycle)
		argPos++
	}
	if req.PriceCents != 0 {
		updates = append(updates, fmt.Sprintf("price_cents = $%d", argPos))
		args = append(args, req.PriceCents)
		argPos++
	}
	if req.Currency != "" {
		updates = append(updates, fmt.Sprintf("currency = $%d", argPos))
		args = append(args, req.Currency)
		argPos++
	}
	if req.MaxUsers != 0 {
		updates = append(updates, fmt.Sprintf("max_users = $%d", argPos))
		args = append(args, req.MaxUsers)
		argPos++
	}

	updates = append(updates, "updated_at = NOW()")
	updates = append(updates, fmt.Sprintf("active = $%d", argPos))
	args = append(args, req.Active)
	argPos++

	args = append(args, req.ID)

	query := fmt.Sprintf("UPDATE plans SET %s WHERE id = $%d",
		joinStrings(updates, ", "), argPos)

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update plan: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       rowsAffected > 0,
		"message":       "Plan updated successfully",
		"rows_affected": rowsAffected,
	})
}

// handleAdminDeletePlan - Soft delete a plan (set active to false)
func handleAdminDeletePlan(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	planID := r.URL.Query().Get("id")
	if planID == "" {
		http.Error(w, "Plan ID is required", http.StatusBadRequest)
		return
	}

	query := "UPDATE plans SET active = false, updated_at = NOW() WHERE id = $1"
	result, err := db.Exec(query, planID)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete plan: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       rowsAffected > 0,
		"message":       "Plan deactivated successfully",
		"rows_affected": rowsAffected,
	})
}

// handleAdminListPricingZones - List all pricing zones
func handleAdminListPricingZones(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	query := `SELECT id, country, iso_code, zone, zone_name, world_bank_classification,
	          gni_per_capita_threshold, pricing_multiplier, created_at, updated_at 
	          FROM pricing_zones ORDER BY zone, country`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list pricing zones: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	zones := []map[string]interface{}{}
	for rows.Next() {
		var id, country, isoCode, zone, zoneName, worldBankClass, gniThreshold string
		var pricingMultiplier float64
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &country, &isoCode, &zone, &zoneName, &worldBankClass,
			&gniThreshold, &pricingMultiplier, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		zones = append(zones, map[string]interface{}{
			"id":                        id,
			"country":                   country,
			"iso_code":                  isoCode,
			"zone":                      zone,
			"zone_name":                 zoneName,
			"world_bank_classification": worldBankClass,
			"gni_per_capita_threshold":  gniThreshold,
			"pricing_multiplier":        pricingMultiplier,
			"created_at":                createdAt,
			"updated_at":                updatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pricing_zones": zones,
		"total":         len(zones),
	})
}

// handleAdminCreatePricingZone - Create a new pricing zone
func handleAdminCreatePricingZone(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req CreatePricingZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO pricing_zones (country, iso_code, zone, zone_name, 
	          world_bank_classification, gni_per_capita_threshold, pricing_multiplier) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	var id string
	err := db.QueryRow(query, req.Country, req.ISOCode, req.Zone, req.ZoneName,
		req.WorldBankClassification, req.GNIPerCapitaThreshold, req.PricingMultiplier).Scan(&id)

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create pricing zone: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pricing zone created successfully",
		"id":      id,
	})
}

// handleAdminUpdatePricingZone - Update an existing pricing zone
func handleAdminUpdatePricingZone(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req UpdatePricingZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Country != "" {
		updates = append(updates, fmt.Sprintf("country = $%d", argPos))
		args = append(args, req.Country)
		argPos++
	}
	if req.ISOCode != "" {
		updates = append(updates, fmt.Sprintf("iso_code = $%d", argPos))
		args = append(args, req.ISOCode)
		argPos++
	}
	if req.Zone != "" {
		updates = append(updates, fmt.Sprintf("zone = $%d", argPos))
		args = append(args, req.Zone)
		argPos++
	}
	if req.ZoneName != "" {
		updates = append(updates, fmt.Sprintf("zone_name = $%d", argPos))
		args = append(args, req.ZoneName)
		argPos++
	}
	if req.WorldBankClassification != "" {
		updates = append(updates, fmt.Sprintf("world_bank_classification = $%d", argPos))
		args = append(args, req.WorldBankClassification)
		argPos++
	}
	if req.GNIPerCapitaThreshold != "" {
		updates = append(updates, fmt.Sprintf("gni_per_capita_threshold = $%d", argPos))
		args = append(args, req.GNIPerCapitaThreshold)
		argPos++
	}
	if req.PricingMultiplier != 0 {
		updates = append(updates, fmt.Sprintf("pricing_multiplier = $%d", argPos))
		args = append(args, req.PricingMultiplier)
		argPos++
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, req.ID)

	query := fmt.Sprintf("UPDATE pricing_zones SET %s WHERE id = $%d",
		joinStrings(updates, ", "), argPos)

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update pricing zone: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       rowsAffected > 0,
		"message":       "Pricing zone updated successfully",
		"rows_affected": rowsAffected,
	})
}

// handleAdminDeletePricingZone - Delete a pricing zone
func handleAdminDeletePricingZone(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	zoneID := r.URL.Query().Get("id")
	if zoneID == "" {
		http.Error(w, "Pricing zone ID is required", http.StatusBadRequest)
		return
	}

	query := "DELETE FROM pricing_zones WHERE id = $1"
	result, err := db.Exec(query, zoneID)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete pricing zone: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       rowsAffected > 0,
		"message":       "Pricing zone deleted successfully",
		"rows_affected": rowsAffected,
	})
}

// handleAdminListPurchases - List all purchases/payments (admin view)
func handleAdminListPurchases(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	query := `SELECT id, amount, currency, status, payment_method, customer_id, 
	          order_id, description, external_payment_id, failure_reason, 
	          created_at, updated_at 
	          FROM payments 
	          ORDER BY created_at DESC 
	          LIMIT $1 OFFSET $2`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list purchases: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	purchases := []map[string]interface{}{}
	for rows.Next() {
		var id, currency, status, paymentMethod, customerID, orderID, description string
		var externalPaymentID, failureReason sql.NullString
		var amount float64
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &amount, &currency, &status, &paymentMethod, &customerID,
			&orderID, &description, &externalPaymentID, &failureReason, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		purchase := map[string]interface{}{
			"id":             id,
			"amount":         amount,
			"currency":       currency,
			"status":         status,
			"payment_method": paymentMethod,
			"customer_id":    customerID,
			"order_id":       orderID,
			"description":    description,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
		}

		if externalPaymentID.Valid {
			purchase["external_payment_id"] = externalPaymentID.String
		}
		if failureReason.Valid {
			purchase["failure_reason"] = failureReason.String
		}

		purchases = append(purchases, purchase)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"purchases": purchases,
		"total":     len(purchases),
		"limit":     limit,
		"offset":    offset,
	})
}

// handleAdminListEntitlements - List all entitlements (admin view)
func handleAdminListEntitlements(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	query := `SELECT id, user_id, family_id, feature_code, plan_id, subscription_id,
	          status, granted_at, expires_at, usage_limits, metadata, created_at, updated_at 
	          FROM entitlements 
	          ORDER BY created_at DESC 
	          LIMIT $1 OFFSET $2`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list entitlements: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entitlements := []map[string]interface{}{}
	for rows.Next() {
		var id, userID, featureCode, planID, status string
		var familyID, subscriptionID sql.NullString
		var usageLimits, metadata []byte
		var grantedAt, createdAt, updatedAt time.Time
		var expiresAt sql.NullTime

		err := rows.Scan(&id, &userID, &familyID, &featureCode, &planID, &subscriptionID,
			&status, &grantedAt, &expiresAt, &usageLimits, &metadata, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		entitlement := map[string]interface{}{
			"id":           id,
			"user_id":      userID,
			"feature_code": featureCode,
			"plan_id":      planID,
			"status":       status,
			"granted_at":   grantedAt,
			"created_at":   createdAt,
			"updated_at":   updatedAt,
		}

		if familyID.Valid {
			entitlement["family_id"] = familyID.String
		}
		if subscriptionID.Valid {
			entitlement["subscription_id"] = subscriptionID.String
		}
		if expiresAt.Valid {
			entitlement["expires_at"] = expiresAt.Time
		}

		entitlements = append(entitlements, entitlement)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entitlements": entitlements,
		"total":        len(entitlements),
		"limit":        limit,
		"offset":       offset,
	})
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
