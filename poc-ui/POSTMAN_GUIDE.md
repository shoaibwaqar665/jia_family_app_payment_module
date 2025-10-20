# Postman Collection Guide

## 📦 Import the Collection

### Method 1: Import from File
1. Open Postman
2. Click **Import** button (top left)
3. Select **Upload Files**
4. Choose `Admin_API.postman_collection.json`
5. Click **Import**

### Method 2: Import from URL
1. Open Postman
2. Click **Import** button
3. Select **Link**
4. Paste the file path or URL
5. Click **Continue**

---

## 📋 Collection Structure

The collection is organized into the following folders:

### 1. Plans Management
- ✅ List All Plans
- ✅ Create Plan - Premium Monthly
- ✅ Create Plan - Basic Yearly
- ✅ Update Plan - Change Price
- ✅ Update Plan - Change Features
- ✅ Delete Plan - Deactivate

### 2. Pricing Zones Management
- ✅ List All Pricing Zones
- ✅ Create Zone - Premium (Zone A)
- ✅ Create Zone - Mid-High (Zone B)
- ✅ Create Zone - Mid-Low (Zone C)
- ✅ Create Zone - Low Income (Zone D)
- ✅ Update Zone - Change Multiplier
- ✅ Update Zone - Change Classification
- ✅ Delete Pricing Zone

### 3. Purchases Management
- ✅ List All Purchases
- ✅ List Purchases - First Page
- ✅ List Purchases - Second Page

### 4. Entitlements Management
- ✅ List All Entitlements
- ✅ List Entitlements - First Page
- ✅ List Entitlements - Second Page

### 5. Examples - Common Scenarios
- ✅ Create Complete Pricing Setup
- ✅ Update Pricing - Price Increase
- ✅ View All Data

---

## 🚀 Quick Start

### 1. Set Up Environment Variables (Optional)
You can create an environment in Postman with:
```
base_url: http://localhost:8082
admin_path: /api/admin
```

### 2. Start the Server
```bash
cd poc-ui
go run server.go
```

### 3. Test the API
1. Open the collection in Postman
2. Select a request
3. Click **Send**
4. View the response

---

## 📝 Using the Requests

### Create a Plan
1. Navigate to **Plans Management** folder
2. Select **Create Plan - Premium Monthly**
3. Review the request body (already filled in)
4. Click **Send**
5. Check the response for success

### Update a Plan
1. Navigate to **Plans Management** folder
2. Select **Update Plan - Change Price**
3. Modify the request body:
   ```json
   {
     "id": "premium_monthly",
     "price_cents": 2499,
     "active": true
   }
   ```
4. Click **Send**

### Create a Pricing Zone
1. Navigate to **Pricing Zones Management** folder
2. Select **Create Zone - Mid-Low (Zone C)**
3. Review the request body
4. Click **Send**

### View Purchases
1. Navigate to **Purchases Management** folder
2. Select **List All Purchases**
3. Click **Send**
4. Review the paginated results

---

## 🎯 Example Workflows

### Workflow 1: Create Complete Pricing Setup
1. Open **Examples - Common Scenarios** folder
2. Select **Create Complete Pricing Setup**
3. Run the collection:
   - 1. Create Premium Plan
   - 2. Create Zone A (Premium)
   - 3. Create Zone C (Mid-Low)

### Workflow 2: Update Pricing
1. Open **Examples - Common Scenarios** folder
2. Select **Update Pricing - Price Increase**
3. Run the collection:
   - 1. List Current Plans
   - 2. Update Plan Price
   - 3. Verify Updated Plan

### Workflow 3: View All Data
1. Open **Examples - Common Scenarios** folder
2. Select **View All Data**
3. Run the collection to view:
   - All Plans
   - All Pricing Zones
   - All Purchases
   - All Entitlements

---

## 🔧 Customizing Requests

### Change the Base URL
If your server is running on a different port or host:

1. Click on the collection name
2. Go to **Variables** tab
3. Update `base_url` variable
4. All requests will use the new URL

### Modify Request Bodies
All requests have pre-filled example data. To customize:

1. Select a request
2. Go to **Body** tab
3. Edit the JSON
4. Click **Send**

### Add Authentication
To add authentication headers:

1. Select a request
2. Go to **Headers** tab
3. Add header:
   - Key: `Authorization`
   - Value: `Bearer YOUR_TOKEN`

---

## 📊 Understanding Responses

### Success Response (Create/Update)
```json
{
  "success": true,
  "message": "Plan created successfully",
  "id": "premium_monthly"
}
```

### Success Response (List)
```json
{
  "plans": [
    {
      "id": "premium_monthly",
      "name": "Premium Monthly",
      "price_cents": 1999,
      "active": true
    }
  ],
  "total": 1
}
```

### Error Response
```json
{
  "error": "Failed to create plan: duplicate key value"
}
```

---

## 🧪 Testing Tips

### 1. Use the Test Scripts
Some requests include test scripts that automatically verify responses:
```javascript
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Plan created successfully", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.success).to.eql(true);
});
```

### 2. Use the Collection Runner
1. Click on the collection name
2. Click **Run**
3. Select which requests to run
4. Click **Run Admin API**
5. View the test results

### 3. Save Responses
1. After sending a request
2. Click **Save Response**
3. Name your response
4. Use it as an example later

---

## 🔍 Common Use Cases

### Create a New Plan
```json
POST /api/admin/plans
{
  "id": "enterprise_monthly",
  "name": "Enterprise Monthly",
  "description": "Enterprise plan with all features",
  "feature_codes": ["all_features"],
  "billing_cycle": "monthly",
  "price_cents": 4999,
  "currency": "USD",
  "max_users": 50,
  "active": true
}
```

### Update Plan Price
```json
PUT /api/admin/plans
{
  "id": "premium_monthly",
  "price_cents": 2999,
  "active": true
}
```

### Create Pricing Zone
```json
POST /api/admin/pricing-zones
{
  "country": "Japan",
  "iso_code": "JP",
  "zone": "A",
  "zone_name": "Premium",
  "world_bank_classification": "High income",
  "gni_per_capita_threshold": "$12,536+",
  "pricing_multiplier": 1.0
}
```

### View Purchases with Pagination
```
GET /api/admin/purchases?limit=20&offset=0
```

---

## 🐛 Troubleshooting

### Connection Refused
- **Issue**: Cannot connect to server
- **Solution**: Make sure the server is running (`go run server.go`)

### 404 Not Found
- **Issue**: Endpoint not found
- **Solution**: Check the URL and ensure the server is running on port 8082

### 500 Internal Server Error
- **Issue**: Server error
- **Solution**: Check server logs for detailed error messages

### Database Connection Error
- **Issue**: Cannot connect to database
- **Solution**: 
  1. Ensure PostgreSQL is running
  2. Check database credentials in `server.go`
  3. Verify database exists

---

## 📚 Additional Resources

- **Admin API Documentation**: See `ADMIN_API_README.md`
- **Implementation Details**: See `IMPLEMENTATION_SUMMARY.md`
- **Quick Start Guide**: See `ADMIN_QUICK_START.md`

---

## 🎨 Postman Features Used

### Variables
- `base_url`: Base URL for all requests
- `admin_path`: Admin API path

### Pre-request Scripts
- Set dynamic values
- Generate timestamps
- Create unique IDs

### Test Scripts
- Verify response status
- Check response data
- Validate JSON structure

### Collection Runner
- Run multiple requests in sequence
- View test results
- Export results

---

## 💡 Best Practices

1. **Use Environments**: Create different environments for dev, staging, and production
2. **Save Examples**: Save successful responses as examples
3. **Add Tests**: Write test scripts for all requests
4. **Document Changes**: Add descriptions to requests
5. **Use Variables**: Use variables for repeated values
6. **Organize Folders**: Keep requests organized in folders
7. **Version Control**: Commit the collection to version control

---

## 🔐 Security Notes

⚠️ **Important**: The current implementation does NOT include authentication. For production:

1. Add authentication headers to all requests
2. Use environment variables for sensitive data
3. Never commit credentials to version control
4. Use HTTPS in production

---

## 📞 Support

For issues or questions:
1. Check the server logs
2. Review the documentation
3. Test with the admin UI: `http://localhost:8082/admin.html`
4. Verify database connectivity

---

**Happy Testing! 🚀**

