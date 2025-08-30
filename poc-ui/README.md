# Jia Family App - Payment Service POC

This is a complete Proof of Concept (POC) demonstrating the payment service integration with Stripe, including a full UI and backend implementation.

## 🚀 Features

- **Complete Payment Flow**: Plan selection → Checkout → Payment processing → Entitlement creation
- **Real Stripe Integration**: Uses actual Stripe API for checkout sessions
- **Dynamic Pricing**: Country-based pricing zones with multipliers
- **Webhook Processing**: Handles Stripe webhook events
- **Entitlement Management**: Creates user entitlements after successful payments
- **Modern UI**: Beautiful, responsive interface with real-time updates

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   POC UI        │    │   HTTP Server   │    │   gRPC Service  │    │   Stripe API    │
│   (Frontend)    │◄──►│   (Bridge)      │◄──►│   (Backend)     │◄──►│   (Payment)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 📁 Files Structure

```
poc-ui/
├── index.html          # Main POC interface
├── success.html        # Payment success page
├── cancel.html         # Payment cancellation page
├── styles.css          # UI styling
├── script.js           # Frontend JavaScript
├── server.go           # HTTP-to-gRPC bridge server
└── README.md           # This file
```

## 🛠️ Setup Instructions

### 1. Start the Payment Service

```bash
# In the main project directory
go run cmd/paymentservice/main.go
```

### 2. Start the POC HTTP Server

```bash
# In the poc-ui directory
go run server.go
```

### 3. Access the POC

Open your browser and navigate to: `http://localhost:8080`

## 🎯 How to Use the POC

### 1. **Select a Plan**
- Choose from Basic ($9.99), Pro ($19.99), or Family ($29.99) plans
- Each plan has different features and pricing

### 2. **Configure Payment**
- Select your country (affects pricing via pricing zones)
- Optionally enter a Family ID
- Review the pricing calculation

### 3. **Process Payment**
- Click "Pay Now" to create a Stripe checkout session
- You'll be redirected to Stripe's secure checkout
- Complete the payment with test card details

### 4. **View Results**
- After successful payment, you'll see the payment status
- Entitlements are automatically created
- View your entitlements in the footer

## 🧪 Testing

### Test Card Numbers (Stripe Test Mode)

- **Success**: `4242 4242 4242 4242`
- **Decline**: `4000 0000 0000 0002`
- **Insufficient Funds**: `4000 0000 0000 9995`

Use any future expiry date and any 3-digit CVC.

### Pricing Zones

The POC demonstrates dynamic pricing based on country:

- **Premium** (US, GB, DE): 1.00x multiplier
- **Mid-High** (CN, BR): 0.70x multiplier  
- **Mid-Low** (IN): 0.40x multiplier
- **Low-Income** (AF): 0.20x multiplier

## 🔧 API Endpoints

The HTTP server provides these endpoints:

- `POST /api/checkout` - Create Stripe checkout session
- `POST /api/webhook` - Process Stripe webhooks
- `GET /api/payments` - List payments

## 📊 What Happens Behind the Scenes

### 1. **Checkout Session Creation**
```javascript
// Frontend calls HTTP API
POST /api/checkout
{
  "plan_id": "pro_monthly",
  "user_id": "spiff_id_test_user_123",
  "country_code": "US",
  "base_price": 1999,
  "currency": "USD"
}
```

### 2. **Stripe Integration**
```go
// Backend creates real Stripe checkout session
session, err := session.New(&stripe.CheckoutSessionParams{
    PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
    LineItems: lineItems,
    Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
    SuccessURL: stripe.String(req.SuccessURL),
    CancelURL: stripe.String(req.CancelURL),
    Metadata: metadata,
})
```

### 3. **Webhook Processing**
```go
// Stripe sends webhook when payment completes
func (a *Adapter) handleCheckoutSessionCompleted(event stripe.Event) {
    // Extract user_id, plan_id from metadata
    // Create entitlement in database
    // Publish event to Kafka
}
```

### 4. **Entitlement Creation**
```go
// System creates user entitlement
entitlement := domain.Entitlement{
    UserID: userID,
    FeatureCode: "pro_storage",
    PlanID: planID,
    Status: "active",
    GrantedAt: time.Now(),
}
```

## 🎨 UI Features

- **Responsive Design**: Works on desktop and mobile
- **Real-time Updates**: Live pricing calculations
- **Loading States**: Visual feedback during operations
- **Toast Notifications**: Success/error messages
- **Modern Styling**: Beautiful gradients and animations

## 🔍 Monitoring

Check the logs for detailed information:

```bash
# Payment service logs
tail -f logs/payment-service.log

# HTTP server logs
# (displayed in terminal where server.go is running)
```

## 🚨 Troubleshooting

### Common Issues

1. **"Connection refused"**: Make sure the payment service is running on port 8081
2. **"Stripe error"**: Check your Stripe API keys in config.yaml
3. **"Webhook failed"**: Ensure webhook endpoint is accessible

### Debug Mode

Enable debug logging by setting `LOG_LEVEL=debug` in your environment.

## 🔐 Security Notes

- This POC uses test Stripe keys (safe for demonstration)
- Webhook signature validation is disabled for POC simplicity
- In production, always validate webhook signatures
- Use HTTPS in production environments

## 📈 Next Steps

To make this production-ready:

1. **Add webhook signature validation**
2. **Implement proper error handling**
3. **Add rate limiting**
4. **Set up monitoring and alerting**
5. **Add comprehensive tests**
6. **Implement proper logging**

## 🎉 Demo Flow

1. **Start both services**
2. **Open http://localhost:8080**
3. **Select Pro Plan**
4. **Choose United States**
5. **Click "Pay Now"**
6. **Use test card: 4242 4242 4242 4242**
7. **Complete payment**
8. **View success page and entitlements**

This POC demonstrates a complete, working payment system with real Stripe integration! 🚀
