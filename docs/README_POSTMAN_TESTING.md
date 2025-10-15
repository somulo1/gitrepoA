# VaultKe Backend API Testing with Postman

> **🚀 Quick Navigation**: [Setup](#setup-instructions) | [Authentication](#authentication-flow) | [Testing](#api-endpoints-testing) | [Troubleshooting](#troubleshooting) | [Quick Start](#quick-start-guide)

## 📋 Table of Contents
- [🚀 Setup Instructions](#setup-instructions)
- [🔧 Environment Configuration](#environment-configuration)
- [🔐 Authentication Flow](#authentication-flow)
- [📊 API Endpoints Testing](#api-endpoints-testing)
- [🧪 Test Collections](#test-collections)
- [🔍 Common Test Scripts](#common-test-scripts)
- [❌ Common Issues & Solutions](#common-issues--solutions)
- [🔄 Automated Testing](#automated-testing)
- [🔧 Advanced Testing Scenarios](#advanced-testing-scenarios)
- [📊 Performance Testing](#performance-testing)
- [🔐 Security Testing](#security-testing)
- [🧪 Data Validation Tests](#data-validation-tests)
- [📱 Mobile-Specific Testing](#mobile-specific-testing)
- [🔄 Integration Testing](#integration-testing)
- [📈 Monitoring and Reporting](#monitoring-and-reporting)
- [🚨 Error Handling Tests](#error-handling-tests)
- [🔧 Environment-Specific Testing](#environment-specific-testing)
- [🚀 Quick Start Guide](#quick-start-guide)
- [📋 Testing Checklist](#testing-checklist)
- [🔧 Troubleshooting](#troubleshooting)
- [📊 Test Reports](#test-reports)
- [📞 Support](#support)

## 🚀 Setup Instructions

### Prerequisites
1. **Postman Desktop App** or **Postman Web** installed
2. **VaultKe Backend Server** running locally
3. **Database** initialized with migrations

### Starting the Backend Server
```bash
cd apps/backend
go mod tidy
go run main.go
```

The server should start on `http://localhost:8080`

### Verify Server is Running
Open your browser and navigate to: `http://localhost:8080/health`

Expected response:
```json
{
  "status": "ok",
  "message": "VaultKe API is running",
  "version": "1.0.0"
}
```

## 🔧 Environment Configuration

### Create Postman Environment

1. **Open Postman** → Click on "Environments" → "Create Environment"
2. **Name**: `VaultKe Local`
3. **Add Variables**:

| Variable | Initial Value | Current Value |
|----------|---------------|---------------|
| `base_url` | `http://localhost:8080` | `http://localhost:8080` |
| `api_url` | `{{base_url}}/api/v1` | `{{base_url}}/api/v1` |
| `auth_token` | | (will be set automatically) |
| `user_id` | | (will be set automatically) |
| `test_email` | `test@vaultke.com` | `test@vaultke.com` |
| `test_phone` | `+254700000000` | `+254700000000` |
| `test_password` | `TestPassword123!` | `TestPassword123!` |

4. **Save Environment** and **Select** it as active

## 🔐 Authentication Flow

### 1. User Registration

**Method**: `POST`
**URL**: `{{api_url}}/auth/register`
**Headers**:
```
Content-Type: application/json
```

**Body** (JSON):
```json
{
  "email": "{{test_email}}",
  "phone": "{{test_phone}}",
  "firstName": "Test",
  "lastName": "User",
  "password": "{{test_password}}",
  "language": "en"
}
```

**Expected Response** (201):
```json
{
  "success": true,
  "message": "User registered successfully. Please verify your email and phone number.",
  "data": {
    "user": {
      "id": "user_id_here",
      "email": "test@vaultke.com",
      "firstName": "Test",
      "lastName": "User",
      "role": "user",
      "status": "pending"
    },
    "token": "jwt_token_here"
  }
}
```

**Post-Response Script** (Tests tab):
```javascript
if (pm.response.code === 201) {
    const response = pm.response.json();
    if (response.success && response.data) {
        pm.environment.set("auth_token", response.data.token);
        pm.environment.set("user_id", response.data.user.id);
        console.log("✅ Registration successful - Token saved");
    }
}

pm.test("Registration successful", function () {
    pm.response.to.have.status(201);
    pm.expect(pm.response.json().success).to.be.true;
});
```

### 2. User Login

**Method**: `POST`
**URL**: `{{api_url}}/auth/login`
**Headers**:
```
Content-Type: application/json
```

**Body** (JSON):
```json
{
  "identifier": "{{test_email}}",
  "password": "{{test_password}}"
}
```

**Expected Response** (200):
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "user": {
      "id": "user_id_here",
      "email": "test@vaultke.com",
      "firstName": "Test",
      "lastName": "User"
    },
    "token": "jwt_token_here"
  }
}
```

**Post-Response Script**:
```javascript
if (pm.response.code === 200) {
    const response = pm.response.json();
    if (response.success && response.data) {
        pm.environment.set("auth_token", response.data.token);
        pm.environment.set("user_id", response.data.user.id);
        console.log("✅ Login successful - Token updated");
    }
}

pm.test("Login successful", function () {
    pm.response.to.have.status(200);
    pm.expect(pm.response.json().success).to.be.true;
});
```

### 3. Get User Profile (Protected Route)

**Method**: `GET`
**URL**: `{{api_url}}/users/profile`
**Headers**:
```
Authorization: Bearer {{auth_token}}
Content-Type: application/json
```

**Expected Response** (200):
```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
    "user": {
      "id": "user_id_here",
      "email": "test@vaultke.com",
      "firstName": "Test",
      "lastName": "User",
      "role": "user",
      "status": "pending",
      "isEmailVerified": false,
      "isPhoneVerified": false
    }
  }
}
```

## 📊 API Endpoints Testing

### Health Check

**Method**: `GET`
**URL**: `{{base_url}}/health`
**Headers**: None

**Test Script**:
```javascript
pm.test("Health check successful", function () {
    pm.response.to.have.status(200);
    pm.expect(pm.response.json().status).to.eql("ok");
});
```

### Authentication Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/auth/register` | POST | User registration | No |
| `/auth/login` | POST | User login | No |
| `/auth/logout` | POST | User logout | Yes |
| `/auth/refresh` | POST | Refresh JWT token | No |
| `/auth/verify-email` | POST | Verify email address | Yes |
| `/auth/verify-phone` | POST | Verify phone number | Yes |
| `/auth/forgot-password` | POST | Request password reset | No |
| `/auth/reset-password` | POST | Reset password | No |

### User Management Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/users/profile` | GET | Get user profile | Yes |
| `/users/profile` | PUT | Update user profile | Yes |
| `/users/avatar` | POST | Upload user avatar | Yes |

### Chama Management Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/chamas` | GET | List all chamas | Yes |
| `/chamas` | POST | Create new chama | Yes |
| `/chamas/:id` | GET | Get chama details | Yes |
| `/chamas/:id` | PUT | Update chama | Yes |
| `/chamas/:id` | DELETE | Delete chama | Yes |
| `/chamas/:id/members` | GET | Get chama members | Yes |
| `/chamas/:id/join` | POST | Join chama | Yes |
| `/chamas/:id/leave` | POST | Leave chama | Yes |
| `/chamas/:id/transactions` | GET | Get chama transactions | Yes |

### Wallet Management Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/wallets` | GET | Get user wallets | Yes |
| `/wallets/:id` | GET | Get wallet details | Yes |
| `/wallets/:id/transactions` | GET | Get wallet transactions | Yes |
| `/wallets/transfer` | POST | Transfer money | Yes |
| `/wallets/deposit` | POST | Deposit money | Yes |
| `/wallets/withdraw` | POST | Withdraw money | Yes |

### Marketplace Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/marketplace/products` | GET | List products | Yes |
| `/marketplace/products` | POST | Create product | Yes |
| `/marketplace/products/:id` | GET | Get product details | Yes |
| `/marketplace/products/:id` | PUT | Update product | Yes |
| `/marketplace/products/:id` | DELETE | Delete product | Yes |
| `/marketplace/cart` | GET | Get shopping cart | Yes |
| `/marketplace/cart` | POST | Add to cart | Yes |
| `/marketplace/cart/:id` | DELETE | Remove from cart | Yes |
| `/marketplace/orders` | GET | List orders | Yes |
| `/marketplace/orders` | POST | Create order | Yes |
| `/marketplace/orders/:id` | GET | Get order details | Yes |
| `/marketplace/orders/:id` | PUT | Update order | Yes |
| `/marketplace/reviews` | GET | Get reviews | Yes |
| `/marketplace/reviews` | POST | Create review | Yes |

### Payment Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/payments/mpesa/stk` | POST | Initiate M-Pesa STK Push | Yes |
| `/payments/mpesa/callback` | POST | M-Pesa callback handler | No |
| `/payments/bank-transfer` | POST | Initiate bank transfer | Yes |

## 🧪 Test Collections

### Basic Authentication Flow Test

1. **Register User** → Save token
2. **Login User** → Update token
3. **Get Profile** → Verify user data
4. **Update Profile** → Test profile update
5. **Logout** → Clear session

### Chama Management Flow Test

1. **Create Chama** → Save chama ID
2. **Get Chama Details** → Verify creation
3. **Update Chama** → Test updates
4. **Join Chama** → Test membership
5. **Get Members** → Verify membership
6. **Leave Chama** → Test leaving

### Marketplace Flow Test

1. **Create Product** → Save product ID
2. **List Products** → Verify listing
3. **Add to Cart** → Test cart functionality
4. **Create Order** → Test order creation
5. **Track Order** → Test order tracking

## 🔍 Common Test Scripts

### Global Pre-Request Script
```javascript
// Auto-set authorization header if token exists
if (pm.environment.get("auth_token")) {
    pm.request.headers.add({
        key: "Authorization",
        value: "Bearer " + pm.environment.get("auth_token")
    });
}
```

### Global Test Script
```javascript
// Log response for debugging
console.log("Response:", pm.response.json());

// Test response time
pm.test("Response time is less than 2000ms", function () {
    pm.expect(pm.response.responseTime).to.be.below(2000);
});

// Test response format
pm.test("Response has success field", function () {
    pm.expect(pm.response.json()).to.have.property('success');
});
```

## ❌ Common Issues & Solutions

### Issue 1: "Authorization header required"
**Solution**: Ensure you're logged in and the `auth_token` environment variable is set.

### Issue 2: "Connection refused"
**Solution**: Verify the backend server is running on the correct port.

### Issue 3: "Invalid JSON"
**Solution**: Check request body format and Content-Type header.

### Issue 4: "User already exists"
**Solution**: Use a different email/phone or delete the existing user from the database.

### Issue 5: "Token expired"
**Solution**: Use the refresh token endpoint or login again.

## 🔄 Automated Testing

### Collection Runner Setup
1. **Import Collection** → Create from endpoints above
2. **Set Environment** → Select VaultKe Local
3. **Configure Iterations** → Set to 1 for single run
4. **Run Collection** → Monitor results

### Newman CLI Testing
```bash
# Install Newman
npm install -g newman

# Run collection
newman run VaultKe_Collection.json -e VaultKe_Local.json
```

## 📝 Notes

- Always test authentication endpoints first
- Use environment variables for dynamic data
- Implement proper error handling in tests
- Monitor response times and performance
- Keep test data consistent across runs
- Use meaningful test descriptions
- Implement cleanup procedures for test data

## 🔧 Advanced Testing Scenarios

### Chama Creation and Management

**Create Chama**:
```json
POST {{api_url}}/chamas
{
  "name": "Test Investment Chama",
  "description": "A test chama for investment purposes",
  "type": "investment",
  "contributionAmount": 5000,
  "contributionFrequency": "monthly",
  "maxMembers": 20,
  "isPublic": true,
  "rules": {
    "latePaymentFee": 100,
    "withdrawalNotice": 7,
    "meetingFrequency": "monthly"
  }
}
```

**Join Chama**:
```json
POST {{api_url}}/chamas/{{chama_id}}/join
{
  "message": "I would like to join this chama"
}
```

### Wallet Operations

**Transfer Money**:
```json
POST {{api_url}}/wallets/transfer
{
  "fromWalletId": "{{wallet_id}}",
  "toWalletId": "{{recipient_wallet_id}}",
  "amount": 1000,
  "description": "Test transfer",
  "pin": "1234"
}
```

**Deposit Money**:
```json
POST {{api_url}}/wallets/deposit
{
  "walletId": "{{wallet_id}}",
  "amount": 5000,
  "method": "mpesa",
  "reference": "TEST123456"
}
```

### Marketplace Operations

**Create Product**:
```json
POST {{api_url}}/marketplace/products
{
  "name": "Test Product",
  "description": "A test product for the marketplace",
  "price": 2500,
  "category": "electronics",
  "images": ["https://example.com/image1.jpg"],
  "stock": 10,
  "location": {
    "county": "Nairobi",
    "town": "Westlands"
  }
}
```

**Add to Cart**:
```json
POST {{api_url}}/marketplace/cart
{
  "productId": "{{product_id}}",
  "quantity": 2,
  "notes": "Please pack carefully"
}
```

**Create Order**:
```json
POST {{api_url}}/marketplace/orders
{
  "items": [
    {
      "productId": "{{product_id}}",
      "quantity": 1,
      "price": 2500
    }
  ],
  "deliveryAddress": {
    "street": "123 Test Street",
    "town": "Nairobi",
    "county": "Nairobi",
    "postalCode": "00100"
  },
  "paymentMethod": "wallet"
}
```

## 📊 Performance Testing

### Load Testing with Newman

**Create performance test script**:
```javascript
// performance-test.js
const newman = require('newman');

newman.run({
    collection: 'VaultKe_Collection.json',
    environment: 'VaultKe_Local.json',
    iterationCount: 100,
    reporters: ['cli', 'json'],
    reporter: {
        json: {
            export: './performance-results.json'
        }
    }
}, function (err) {
    if (err) { throw err; }
    console.log('Performance test completed!');
});
```

### Stress Testing Scenarios

1. **Concurrent User Registration** (50 users)
2. **Simultaneous Chama Creation** (20 chamas)
3. **High-Volume Transactions** (1000 transfers)
4. **Marketplace Load** (500 product searches)

## 🔐 Security Testing

### Authentication Security Tests

**Test Invalid Token**:
```javascript
pm.test("Invalid token rejected", function () {
    pm.sendRequest({
        url: pm.environment.get("api_url") + "/users/profile",
        method: 'GET',
        header: {
            'Authorization': 'Bearer invalid_token_here'
        }
    }, function (err, response) {
        pm.expect(response.code).to.equal(401);
    });
});
```

**Test SQL Injection**:
```json
POST {{api_url}}/auth/login
{
  "identifier": "test@example.com'; DROP TABLE users; --",
  "password": "password"
}
```

**Test XSS Prevention**:
```json
PUT {{api_url}}/users/profile
{
  "firstName": "<script>alert('XSS')</script>",
  "lastName": "Test"
}
```

## 🧪 Data Validation Tests

### Input Validation

**Test Email Validation**:
```javascript
pm.test("Invalid email rejected", function () {
    pm.sendRequest({
        url: pm.environment.get("api_url") + "/auth/register",
        method: 'POST',
        header: {
            'Content-Type': 'application/json'
        },
        body: {
            mode: 'raw',
            raw: JSON.stringify({
                email: "invalid-email",
                phone: "+254700000000",
                firstName: "Test",
                lastName: "User",
                password: "TestPassword123!"
            })
        }
    }, function (err, response) {
        pm.expect(response.code).to.equal(400);
    });
});
```

**Test Password Strength**:
```javascript
const weakPasswords = ["123", "password", "abc123"];

weakPasswords.forEach(password => {
    pm.test(`Weak password '${password}' rejected`, function () {
        pm.sendRequest({
            url: pm.environment.get("api_url") + "/auth/register",
            method: 'POST',
            header: {
                'Content-Type': 'application/json'
            },
            body: {
                mode: 'raw',
                raw: JSON.stringify({
                    email: "test@example.com",
                    phone: "+254700000000",
                    firstName: "Test",
                    lastName: "User",
                    password: password
                })
            }
        }, function (err, response) {
            pm.expect(response.code).to.equal(400);
        });
    });
});
```

## 📱 Mobile-Specific Testing

### Device Simulation Headers

Add these headers to simulate mobile requests:
```
User-Agent: VaultKe-Mobile/1.0.0 (iOS 15.0; iPhone 13)
X-Device-Type: mobile
X-App-Version: 1.0.0
X-Platform: ios
```

### Offline Sync Testing

**Test Offline Queue**:
```json
POST {{api_url}}/sync/queue
{
  "operations": [
    {
      "type": "transaction",
      "method": "POST",
      "endpoint": "/wallets/transfer",
      "data": {
        "fromWalletId": "wallet1",
        "toWalletId": "wallet2",
        "amount": 1000
      },
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

## 🔄 Integration Testing

### End-to-End User Journey

**Complete User Flow**:
1. Register → Login → Verify Email → Verify Phone
2. Create Chama → Invite Members → Accept Invitations
3. Make Contributions → Request Loan → Approve Loan
4. Create Product → Add to Cart → Checkout → Track Order
5. Rate Product → Leave Review → Update Profile

### Third-Party Integration Tests

**M-Pesa STK Push**:
```json
POST {{api_url}}/payments/mpesa/stk
{
  "phoneNumber": "254700000000",
  "amount": 1000,
  "accountReference": "VaultKe",
  "transactionDesc": "Test Payment"
}
```

**Bank Transfer Simulation**:
```json
POST {{api_url}}/payments/bank-transfer
{
  "bankCode": "01",
  "accountNumber": "1234567890",
  "amount": 5000,
  "narration": "Test transfer"
}
```

## 📈 Monitoring and Reporting

### Custom Test Reports

**Generate HTML Report**:
```bash
newman run collection.json -e environment.json -r htmlextra --reporter-htmlextra-export report.html
```

**Performance Metrics Collection**:
```javascript
// In Tests tab
pm.test("Response time acceptable", function () {
    const responseTime = pm.response.responseTime;
    pm.expect(responseTime).to.be.below(1000);

    // Log metrics
    console.log(`Endpoint: ${pm.request.url}`);
    console.log(`Response Time: ${responseTime}ms`);
    console.log(`Status: ${pm.response.code}`);
});
```

## 🚨 Error Handling Tests

### Network Error Simulation

**Test Connection Timeout**:
```javascript
pm.test("Handle connection timeout", function () {
    pm.sendRequest({
        url: "http://invalid-url:9999/api/test",
        method: 'GET'
    }, function (err, response) {
        pm.expect(err).to.not.be.null;
    });
});
```

**Test Rate Limiting**:
```javascript
// Send 100 requests rapidly
for (let i = 0; i < 100; i++) {
    pm.sendRequest({
        url: pm.environment.get("api_url") + "/health",
        method: 'GET'
    }, function (err, response) {
        if (response.code === 429) {
            console.log("Rate limiting working correctly");
        }
    });
}
```

## 🔧 Environment-Specific Testing

### Development Environment
```json
{
  "base_url": "http://localhost:8080",
  "database_reset": "true",
  "debug_mode": "true"
}
```

### Staging Environment
```json
{
  "base_url": "https://staging-api.vaultke.com",
  "database_reset": "false",
  "debug_mode": "false"
}
```

### Production Environment
```json
{
  "base_url": "https://api.vaultke.com",
  "database_reset": "false",
  "debug_mode": "false",
  "rate_limit": "true"
}
```

---

**Happy Testing! 🚀**

For issues or questions, refer to the main project documentation or contact the development team.

## 🚀 Quick Start Guide

### Option 1: Import Postman Collection (Recommended)

1. **Download Files**:
   - `VaultKe_Postman_Collection.json` - Complete API collection
   - `VaultKe_Postman_Environment.json` - Environment variables

2. **Import to Postman**:
   - Open Postman → Import → Select files
   - Import both collection and environment
   - Select "VaultKe Local Environment" as active

3. **Start Testing**:
   - Ensure backend server is running (`go run main.go`)
   - Run "Health Check" request first
   - Follow authentication flow: Register → Login → Get Profile
   - Explore other endpoints as needed

### Option 2: Command Line Testing

1. **Make script executable**:
   ```bash
   chmod +x test-api.sh
   ```

2. **Run full test suite**:
   ```bash
   ./test-api.sh full-test
   ```

3. **Run individual tests**:
   ```bash
   ./test-api.sh check-server
   ./test-api.sh login
   ./test-api.sh profile
   ```

### Option 3: Manual cURL Testing

1. **Health Check**:
   ```bash
   curl http://localhost:8080/health
   ```

2. **Register User**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"test@vaultke.com","phone":"+254700000000","firstName":"Test","lastName":"User","password":"TestPassword123!","language":"en"}'
   ```

3. **Login** (save token from response):
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"identifier":"test@vaultke.com","password":"TestPassword123!"}'
   ```

4. **Get Profile** (use token from login):
   ```bash
   curl -X GET http://localhost:8080/api/v1/users/profile \
     -H "Authorization: Bearer YOUR_TOKEN_HERE"
   ```

## 📋 Testing Checklist

### ✅ Basic Functionality
- [ ] Server health check passes
- [ ] User registration works
- [ ] User login returns valid token
- [ ] Protected routes require authentication
- [ ] Profile retrieval works
- [ ] Profile updates work

### ✅ Chama Management
- [ ] Chama creation works
- [ ] Chama listing works
- [ ] Chama details retrieval works
- [ ] Member management works
- [ ] Join/leave chama works

### ✅ Wallet Operations
- [ ] Wallet listing works
- [ ] Wallet details retrieval works
- [ ] Money transfer works
- [ ] Deposit functionality works
- [ ] Transaction history works

### ✅ Marketplace
- [ ] Product creation works
- [ ] Product listing works
- [ ] Cart operations work
- [ ] Order creation works
- [ ] Order tracking works

### ✅ Security
- [ ] Invalid tokens are rejected
- [ ] SQL injection is prevented
- [ ] XSS content is sanitized
- [ ] Rate limiting works
- [ ] Input validation works

### ✅ Performance
- [ ] Response times < 1000ms
- [ ] Concurrent requests handled
- [ ] Database queries optimized
- [ ] Memory usage acceptable

## 🔧 Troubleshooting

### Common Issues

**Issue**: "Connection refused"
```bash
# Solution: Start the backend server
cd apps/backend
go run main.go
```

**Issue**: "Invalid token"
```bash
# Solution: Login again to get fresh token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"test@vaultke.com","password":"TestPassword123!"}'
```

**Issue**: "User already exists"
```bash
# Solution: Use different email or delete existing user
# Or just login with existing credentials
```

**Issue**: "Database connection error"
```bash
# Solution: Check database configuration
# Ensure SQLite file exists and is writable
```

### Debug Mode

Enable debug logging in the backend:
```bash
export LOG_LEVEL=debug
go run main.go
```

### Reset Test Data

To reset all test data:
```bash
# Stop server
# Delete SQLite database file
rm vaultke.db
# Restart server (will recreate database)
go run main.go
```

## 📊 Test Reports

### Generate HTML Report with Newman

1. **Install Newman and HTML reporter**:
   ```bash
   npm install -g newman newman-reporter-htmlextra
   ```

2. **Run collection with HTML report**:
   ```bash
   newman run VaultKe_Postman_Collection.json \
     -e VaultKe_Postman_Environment.json \
     -r htmlextra \
     --reporter-htmlextra-export report.html
   ```

3. **Open report**:
   ```bash
   open report.html
   ```

### Continuous Integration

For CI/CD pipelines, use Newman:
```yaml
# .github/workflows/api-tests.yml
name: API Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Start Backend
        run: |
          cd apps/backend
          go run main.go &
          sleep 5
      - name: Run API Tests
        run: |
          npm install -g newman
          newman run apps/backend/VaultKe_Postman_Collection.json \
            -e apps/backend/VaultKe_Postman_Environment.json \
            --bail
```

## 📞 Support

- **Documentation**: `/docs` endpoint
- **API Status**: `/health` endpoint
- **Issues**: GitHub Issues
- **Email**: dev@vaultke.com

---

## 📁 Files Included

- `README_POSTMAN_TESTING.md` - This comprehensive guide
- `VaultKe_Postman_Collection.json` - Complete Postman collection
- `VaultKe_Postman_Environment.json` - Environment variables
- `test-api.sh` - Command-line testing script

**Ready to test! 🚀**
