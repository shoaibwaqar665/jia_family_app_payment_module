# Admin API Quick Start Guide

## 🚀 Quick Start

### 1. Start the Server
```bash
cd poc-ui
go run server.go
```

### 2. Open Admin Dashboard
```
http://localhost:8082/admin.html
```

### 3. Test the API
```bash
./test-admin-api.sh
```

---

## 📋 API Endpoints Quick Reference

### Plans Management

#### List All Plans
```bash
curl http://localhost:8082/api/admin/plans
```

#### Create Plan
```bash
curl -X POST http://localhost:8082/api/admin/plans \
  -H "Content-Type: application/json" \
  -d '{
    "id": "premium_monthly",
    "name": "Premium Monthly",
    "description": "Premium plan",
    "feature_codes": ["feature1", "feature2"],
    "billing_cycle": "monthly",
    "price_cents": 1999,
    "currency": "USD",
    "max_users": 5,
    "active": true
  }'
```

#### Update Plan
```bash
curl -X PUT http://localhost:8082/api/admin/plans \
  -H "Content-Type: application/json" \
  -d '{
    "id": "premium_monthly",
    "price_cents": 2499,
    "active": true
  }'
```

#### Delete Plan
```bash
curl -X DELETE "http://localhost:8082/api/admin/plans?id=premium_monthly"
```

---

### Pricing Zones Management

#### List All Zones
```bash
curl http://localhost:8082/api/admin/pricing-zones
```

#### Create Zone
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

#### Update Zone
```bash
curl -X PUT http://localhost:8082/api/admin/pricing-zones \
  -H "Content-Type: application/json" \
  -d '{
    "id": "1",
    "pricing_multiplier": 0.45
  }'
```

#### Delete Zone
```bash
curl -X DELETE "http://localhost:8082/api/admin/pricing-zones?id=1"
```

---

### Purchases & Entitlements

#### List Purchases
```bash
curl http://localhost:8082/api/admin/purchases?limit=10&offset=0
```

#### List Entitlements
```bash
curl http://localhost:8082/api/admin/entitlements?limit=10&offset=0
```

---

## 🎨 Admin Dashboard Features

### Plans Tab
- ✅ Create new plans
- ✅ View all plans
- ✅ Edit plans
- ✅ Delete plans
- ✅ Active/Inactive status

### Pricing Zones Tab
- ✅ Create new zones
- ✅ View all zones
- ✅ Edit zones
- ✅ Delete zones
- ✅ Zone classification (A/B/C/D)

### Purchases Tab
- ✅ View all payments
- ✅ Payment status
- ✅ Pagination

### Entitlements Tab
- ✅ View all entitlements
- ✅ User information
- ✅ Expiration dates
- ✅ Pagination

---

## 🔧 Configuration

### Database Settings (server.go)
```go
const (
    dbHost     = "localhost"
    dbPort     = 5432
    dbUser     = "postgres"
    dbPassword = "postgres"
    dbName     = "jia_payment_service"
)
```

### Server Settings
```go
const (
    grpcAddress = "localhost:8081"
    httpPort    = ":8082"
)
```

---

## 📊 Pricing Zone Classifications

| Zone | Name | Multiplier | Description |
|------|------|------------|-------------|
| A | Premium | 1.00 (100%) | High-income |
| B | Mid-High | 0.70 (70%) | Upper-middle-income |
| C | Mid-Low | 0.40 (40%) | Lower-middle-income |
| D | Low-Income | 0.20 (20%) | Low-income |

---

## 🧪 Testing

### Automated Test Script
```bash
./test-admin-api.sh
```

### Manual Testing
1. Start server: `go run server.go`
2. Open UI: `http://localhost:8082/admin.html`
3. Test all operations

---

## 📚 Documentation

- **[README.md](./README.md)** - Main documentation
- **[ADMIN_API_README.md](./ADMIN_API_README.md)** - Detailed API docs
- **[IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)** - Implementation details

---

## ⚠️ Security Notes

**Current Status**: No authentication implemented

**For Production**:
1. Add authentication middleware
2. Implement RBAC
3. Add rate limiting
4. Enable HTTPS
5. Add audit logging

---

## 🐛 Troubleshooting

### Database Connection
```bash
# Check if PostgreSQL is running
docker-compose ps

# Test connection
psql -h localhost -U postgres -d jia_payment_service
```

### Port Issues
```bash
# Check port usage
lsof -i :8082

# Kill process
kill -9 <PID>
```

### Dependencies
```bash
go get github.com/lib/pq
go mod tidy
```

---

## 💡 Tips

1. **Use the Admin UI** - It's easier than cURL for most operations
2. **Check Logs** - Server logs show all requests and errors
3. **Test First** - Use the test script before manual testing
4. **Read Docs** - Full documentation is available

---

## 🎯 Common Tasks

### Create a New Plan
1. Go to Plans tab
2. Fill in the form
3. Click "Create Plan"

### Update Pricing
1. Go to Pricing Zones tab
2. Click "Edit" on a zone
3. Update multiplier
4. Submit

### View All Purchases
1. Go to Purchases tab
2. View table
3. Use pagination if needed

### View All Entitlements
1. Go to Entitlements tab
2. View table
3. Check expiration dates

---

## 📞 Support

For issues:
1. Check server logs
2. Review documentation
3. Test with cURL
4. Verify database

---

**Happy Admin-ing! 🚀**

