package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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
	PlanID      string `json:"plan_id"`
	UserID      string `json:"user_id"`
	FamilyID    string `json:"family_id,omitempty"`
	CountryCode string `json:"country_code"`
	BasePrice   int64  `json:"base_price"`
	Currency    string `json:"currency"`
	SuccessURL  string `json:"success_url"`
	CancelURL   string `json:"cancel_url"`
}

type CheckoutResponse struct {
	SessionID string    `json:"session_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type WebhookRequest struct {
	Payload   []byte `json:"payload"`
	Signature string `json:"signature"`
	Provider  string `json:"provider"`
}

type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
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
		handleListEntitlements(w, r, client)
	})

	http.HandleFunc("/api/pricing-zones", func(w http.ResponseWriter, r *http.Request) {
		handleListPricingZones(w, r, client)
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

	// Convert to gRPC request
	grpcReq := &paymentv1.CreateCheckoutSessionRequest{
		PlanId:      req.PlanID,
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

	// Convert to gRPC request
	grpcReq := &paymentv1.ProcessWebhookRequest{
		Payload:   req.Payload,
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
