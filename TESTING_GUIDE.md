# Payment Service Testing Guide

## 🚀 Quick Start

Your Payment Service is now running and ready for testing! Here's everything you need to know.

## 📍 Service Status

- **Service URL**: `localhost:8081`
- **Protocol**: gRPC
- **Status**: ✅ Running
- **Database**: ✅ PostgreSQL (with migrations applied)
- **Cache**: ✅ Redis

## 🧪 Testing with Postman

### 1. Import the Collection

1. Open Postman
2. Click "Import" 
3. Select `postman_collection_enhanced.json`
4. The collection will be imported with all endpoints organized by category

### 2. Configure Variables

The collection includes these variables (already set):
- `base_url`: `localhost:8081`
- `auth_token`: `spiff_id_test_user_123`
- `user_id`: `user_123`
- `customer_id`: `customer_123`

### 3. Available Endpoint Categories

#### 🏥 Health & Status
- **Health Check**: Verify service is running

#### 💳 Payment Operations (Available via gRPC)
- **Create Payment**: Create a new payment transaction
- **Get Payment**: Retrieve payment by ID
- **Update Payment Status**: Change payment status (pending → completed)
- **Get Payments by Customer**: List all payments for a customer
- **List Payments**: List all payments with pagination

#### ⚠️ Additional Features (Implemented but not exposed via gRPC)
The following features are implemented in the code but not yet exposed through gRPC endpoints:
- **Entitlement Management**: Check user feature access, list entitlements
- **Subscription & Checkout**: Create Stripe checkout sessions
- **Pricing Zones**: Get region-based pricing
- **Plans Management**: List and get subscription plans
- **Webhooks**: Handle payment success webhooks

**Note**: To use these features, you would need to add the service definitions to the proto file and regenerate the gRPC code.

## 🔑 Authentication

Most endpoints require authentication via the `better-auth-token` header:

```
better-auth-token: spiff_id_test_user_123
```

**Exceptions** (no auth required):
- Health Check
- Payment Success Webhook

## 📊 Sample Data

### Available Plans
- `basic_monthly` - $9.99/month
- `basic_yearly` - $95.90/year (20% discount)
- `pro_monthly` - $19.99/month
- `pro_yearly` - $191.90/year (20% discount)
- `family_monthly` - $29.99/month (up to 6 users)
- `family_yearly` - $287.90/year (20% discount)
- `enterprise_monthly` - $99.99/month (up to 100 users)
- `enterprise_yearly` - $959.90/year (20% discount)

### Feature Codes
- `basic_storage`, `basic_support`, `core_features`
- `pro_storage`, `pro_support`, `advanced_analytics`, `api_access`
- `family_storage`, `family_support`, `family_sharing`, `parental_controls`
- `enterprise_storage`, `enterprise_support`, `sso`, `audit_logs`, `custom_integrations`

### Pricing Zones
- **Zone A (Premium)**: 100% of base price (US, UK, etc.)
- **Zone B (Mid-High)**: 70% of base price
- **Zone C (Mid-Low)**: 40% of base price  
- **Zone D (Low-Income)**: 20% of base price

## 🧪 Testing Workflow

### 1. Basic Health Check
```
GET grpc://localhost:8081/grpc.health.v1.Health/Check
```

### 2. Create a Payment
```json
{
  "amount": 1999,
  "currency": "USD",
  "payment_method": "credit_card",
  "customer_id": "customer_123",
  "order_id": "order_123",
  "description": "Pro Plan Monthly Subscription"
}
```

### 3. Check User Entitlement
```json
{
  "user_id": "user_123",
  "feature_code": "pro_storage"
}
```

### 4. Create Checkout Session
```json
{
  "plan_id": "pro_monthly",
  "user_id": "user_123"
}
```

### 5. Get Pricing for Different Countries
```json
{
  "country": "United States",
  "iso_code": "US",
  "base_price": 1999
}
```

## 🔧 Troubleshooting

### Service Not Responding
1. Check if service is running: `lsof -i :8081`
2. Check service logs for errors
3. Verify PostgreSQL and Redis are running: `docker ps`

### Database Issues
1. Check PostgreSQL container: `docker logs paymentservice_postgres`
2. Run migrations again: `cd docker && docker-compose --profile migrate run --rm migrate -path=/migrations -database "postgres://app:app@postgres:5432/payments?sslmode=disable" up`

### Redis Issues
1. Check Redis container: `docker logs paymentservice_redis`
2. Restart Redis: `docker restart paymentservice_redis`

## 📝 Example Test Sequence

1. **Health Check** → Should return OK
2. **List Plans** → Should return all available plans
3. **Create Payment** → Should create a new payment with status "pending"
4. **Get Payment** → Should return the created payment
5. **Update Payment Status** → Should change status to "completed"
6. **Check Entitlement** → Should return false (no active subscription)
7. **Create Checkout Session** → Should return Stripe checkout URL
8. **Payment Success Webhook** → Should create entitlements for user

## 🎯 Key Features to Test

- ✅ Payment creation and status updates
- ✅ Entitlement checking and management
- ✅ Subscription plan management
- ✅ Regional pricing zones
- ✅ Stripe integration (checkout sessions)
- ✅ Webhook handling
- ✅ Authentication and authorization
- ✅ Caching (Redis integration)

## 📞 Support

If you encounter issues:
1. Check the service logs
2. Verify all dependencies are running
3. Test with the provided sample data
4. Use the health check endpoint to verify service status

Happy testing! 🚀
