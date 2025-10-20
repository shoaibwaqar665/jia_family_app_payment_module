# Admin API Documentation

This document describes the Admin API endpoints for managing the Jia Payment Service.

## Overview

The Admin API provides full CRUD operations for:
- **Plans** - Subscription plans with pricing and features
- **Pricing Zones** - Dynamic pricing based on geographic regions
- **Purchases** - View all payment transactions
- **Entitlements** - View all user entitlements

## Base URL

```
http://localhost:8082/api/admin
```

## Endpoints

### Plans Management

#### 1. List All Plans
**GET** `/api/admin/plans`

Returns all plans including inactive ones.

**Response:**
```json
{
  "plans": [
    {
      "id": "pro_monthly",
      "name": "Pro Monthly",
      "description": "Professional plan with monthly billing",
      "feature_codes": ["feature1", "feature2"],
      "billing_cycle": "monthly",
      "price_cents": 1999,
      "price_dollars": 19.99,
      "currency": "USD",
      "max_users": 5,
      "active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

#### 2. Create Plan
**POST** `/api/admin/plans`

Creates a new subscription plan.

**Request Body:**
```json
{
  "id": "pro_monthly",
  "name": "Pro Monthly",
  "description": "Professional plan with monthly billing",
  "feature_codes": ["feature1", "feature2", "feature3"],
  "billing_cycle": "monthly",
  "price_cents": 1999,
  "currency": "USD",
  "max_users": 5,
  "active": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Plan created successfully",
  "id": "pro_monthly"
}
```

#### 3. Update Plan
**PUT** `/api/admin/plans`

Updates an existing plan. Only provide fields you want to update.

**Request Body:**
```json
{
  "id": "pro_monthly",
  "name": "Pro Monthly Updated",
  "price_cents": 2499,
  "active": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Plan updated successfully",
  "rows_affected": 1
}
```

#### 4. Delete Plan (Soft Delete)
**DELETE** `/api/admin/plans?id=<plan_id>`

Deactivates a plan by setting `active` to `false`.

**Response:**
```json
{
  "success": true,
  "message": "Plan deactivated successfully",
  "rows_affected": 1
}
```

---

### Pricing Zones Management

#### 1. List All Pricing Zones
**GET** `/api/admin/pricing-zones`

Returns all pricing zones.

**Response:**
```json
{
  "pricing_zones": [
    {
      "id": "1",
      "country": "United States",
      "iso_code": "US",
      "zone": "A",
      "zone_name": "Premium",
      "world_bank_classification": "High income",
      "gni_per_capita_threshold": "$12,536+",
      "pricing_multiplier": 1.0,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

#### 2. Create Pricing Zone
**POST** `/api/admin/pricing-zones`

Creates a new pricing zone.

**Request Body:**
```json
{
  "country": "United States",
  "iso_code": "US",
  "zone": "A",
  "zone_name": "Premium",
  "world_bank_classification": "High income",
  "gni_per_capita_threshold": "$12,536+",
  "pricing_multiplier": 1.0
}
```

**Response:**
```json
{
  "success": true,
  "message": "Pricing zone created successfully",
  "id": "1"
}
```

#### 3. Update Pricing Zone
**PUT** `/api/admin/pricing-zones`

Updates an existing pricing zone.

**Request Body:**
```json
{
  "id": "1",
  "country": "United States",
  "pricing_multiplier": 0.95
}
```

**Response:**
```json
{
  "success": true,
  "message": "Pricing zone updated successfully",
  "rows_affected": 1
}
```

#### 4. Delete Pricing Zone
**DELETE** `/api/admin/pricing-zones?id=<zone_id>`

Permanently deletes a pricing zone.

**Response:**
```json
{
  "success": true,
  "message": "Pricing zone deleted successfully",
  "rows_affected": 1
}
```

---

### Purchases Management

#### 1. List All Purchases
**GET** `/api/admin/purchases?limit=50&offset=0`

Returns all payment transactions with pagination.

**Query Parameters:**
- `limit` (optional): Number of records to return (default: 50)
- `offset` (optional): Number of records to skip (default: 0)

**Response:**
```json
{
  "purchases": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "amount": 19.99,
      "currency": "USD",
      "status": "completed",
      "payment_method": "credit_card",
      "customer_id": "user_123",
      "order_id": "order_456",
      "description": "Pro Monthly Subscription",
      "external_payment_id": "stripe_pi_123",
      "failure_reason": null,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

---

### Entitlements Management

#### 1. List All Entitlements
**GET** `/api/admin/entitlements?limit=50&offset=0`

Returns all user entitlements with pagination.

**Query Parameters:**
- `limit` (optional): Number of records to return (default: 50)
- `offset` (optional): Number of records to skip (default: 0)

**Response:**
```json
{
  "entitlements": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "user_123",
      "family_id": "family_456",
      "feature_code": "feature1",
      "plan_id": "pro_monthly",
      "subscription_id": "sub_789",
      "status": "active",
      "granted_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-02-01T00:00:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

---

## Pricing Zone Classifications

The system uses four pricing zones based on World Bank classifications:

| Zone | Zone Name | Multiplier | Description |
|------|-----------|------------|-------------|
| A | Premium | 1.00 (100%) | High-income countries |
| B | Mid-High | 0.70 (70%) | Upper-middle-income countries |
| C | Mid-Low | 0.40 (40%) | Lower-middle-income countries |
| D | Low-Income | 0.20 (20%) | Low-income countries |

---

## Usage Examples

### Using cURL

#### Create a Plan
```bash
curl -X POST http://localhost:8082/api/admin/plans \
  -H "Content-Type: application/json" \
  -d '{
    "id": "premium_yearly",
    "name": "Premium Yearly",
    "description": "Premium plan with yearly billing",
    "feature_codes": ["all_features"],
    "billing_cycle": "yearly",
    "price_cents": 19999,
    "currency": "USD",
    "max_users": 10,
    "active": true
  }'
```

#### Update a Plan
```bash
curl -X PUT http://localhost:8082/api/admin/plans \
  -H "Content-Type: application/json" \
  -d '{
    "id": "premium_yearly",
    "price_cents": 17999,
    "active": true
  }'
```

#### Create a Pricing Zone
```bash
curl -X POST http://localhost:8082/api/admin/pricing-zones \
  -H "Content-Type: application/json" \
  -d '{
    "country": "India",
    "iso_code": "IN",
    "zone": "C",
    "zone_name": "Mid-Low",
    "world_bank_classification": "Lower-middle income",
    "gni_per_capita_threshold": "$1,046-$4,095",
    "pricing_multiplier": 0.40
  }'
```

#### List All Purchases
```bash
curl http://localhost:8082/api/admin/purchases?limit=10&offset=0
```

#### List All Entitlements
```bash
curl http://localhost:8082/api/admin/entitlements?limit=10&offset=0
```

---

## Admin UI

A complete admin dashboard is available at:
```
http://localhost:8082/admin.html
```

The admin UI provides:
- ✅ Visual interface for all CRUD operations
- ✅ Real-time data display in tables
- ✅ Form validation
- ✅ Success/error notifications
- ✅ Responsive design

---

## Security Considerations

⚠️ **Important**: The current implementation does not include authentication/authorization. In a production environment, you should:

1. Add authentication middleware to verify admin credentials
2. Implement role-based access control (RBAC)
3. Use JWT tokens or similar for session management
4. Add rate limiting to prevent abuse
5. Log all admin actions for audit trails
6. Use HTTPS in production

Example authentication middleware:
```go
func adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Verify admin token
        token := r.Header.Get("Authorization")
        if !isValidAdminToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

---

## Database Schema

The admin API works with the following tables:

### Plans Table
- `id` (VARCHAR) - Primary key
- `name` (VARCHAR)
- `description` (TEXT)
- `feature_codes` (TEXT[])
- `billing_cycle` (VARCHAR)
- `price_cents` (INTEGER)
- `currency` (VARCHAR)
- `max_users` (INTEGER)
- `usage_limits` (JSONB)
- `metadata` (JSONB)
- `active` (BOOLEAN)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

### Pricing Zones Table
- `id` (VARCHAR) - Primary key
- `country` (VARCHAR)
- `iso_code` (VARCHAR)
- `zone` (VARCHAR)
- `zone_name` (VARCHAR)
- `world_bank_classification` (VARCHAR)
- `gni_per_capita_threshold` (VARCHAR)
- `pricing_multiplier` (DECIMAL)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

### Payments Table
- `id` (UUID) - Primary key
- `amount` (DECIMAL)
- `currency` (VARCHAR)
- `status` (VARCHAR)
- `payment_method` (VARCHAR)
- `customer_id` (VARCHAR)
- `order_id` (VARCHAR)
- `description` (TEXT)
- `external_payment_id` (VARCHAR)
- `failure_reason` (TEXT)
- `metadata` (JSONB)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

### Entitlements Table
- `id` (UUID) - Primary key
- `user_id` (VARCHAR)
- `family_id` (VARCHAR)
- `feature_code` (VARCHAR)
- `plan_id` (VARCHAR)
- `subscription_id` (VARCHAR)
- `status` (VARCHAR)
- `granted_at` (TIMESTAMP)
- `expires_at` (TIMESTAMP)
- `usage_limits` (JSONB)
- `metadata` (JSONB)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

---

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK` - Successful request
- `400 Bad Request` - Invalid input or missing required fields
- `500 Internal Server Error` - Server-side error

Error responses follow this format:
```json
{
  "error": "Error message here"
}
```

---

## Running the Server

1. Ensure PostgreSQL is running:
```bash
docker-compose up -d
```

2. Start the server:
```bash
cd poc-ui
go run server.go
```

3. Access the admin UI:
```
http://localhost:8082/admin.html
```

---

## Testing

You can test the API using the provided admin UI or with tools like:
- Postman
- cURL
- HTTPie
- Browser DevTools

---

## Support

For issues or questions, please refer to the main project documentation or create an issue in the repository.

