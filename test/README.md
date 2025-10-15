# VaultKe Backend Test Suite

This directory contains comprehensive unit tests for the VaultKe backend API handlers. The tests are designed to validate all major functionality of the API endpoints.

## Test Structure

### Core Test Files

1. **`setup_test.go`** - Common test setup and database utilities
2. **`simple_test.go`** - Basic API endpoint tests (working examples)
3. **`basic_test.go`** - More comprehensive API tests
4. **`auth_test.go`** - Authentication and authorization tests
5. **`user_test.go`** - User management endpoint tests
6. **`chama_test.go`** - Chama (group) management tests
7. **`wallet_test.go`** - Wallet and transaction tests
8. **`marketplace_test.go`** - Marketplace endpoint tests
9. **`meeting_test.go`** - Meeting management tests
10. **`notification_test.go`** - Notification system tests
11. **`loan_test.go`** - Loan management tests
12. **`reminder_test.go`** - Reminder system tests

### Test Coverage

The test suite covers the following areas:

#### 1. Authentication & Authorization
- User registration and login
- Password reset functionality
- Email and phone verification
- JWT token management
- Session management

#### 2. User Management
- User profile CRUD operations
- User search and filtering
- Avatar upload functionality
- User role management
- Account status management

#### 3. Chama Management
- Chama creation and management
- Member management (join, leave, roles)
- Chama discovery and search
- Privacy settings and approval workflows
- Chama statistics and reporting

#### 4. Wallet & Transactions
- Wallet creation and management
- Money deposits and withdrawals
- Inter-wallet transfers
- Transaction history and filtering
- Balance management and limits
- Wallet locking/unlocking

#### 5. Marketplace
- Product catalog management
- Shopping cart functionality
- Order processing
- Product reviews and ratings
- Search and filtering
- Category management

#### 6. Meeting Management
- Meeting scheduling and management
- Meeting participation
- Meeting recordings
- Meeting notifications
- Video conferencing integration

#### 7. Notification System
- Notification creation and delivery
- Notification preferences
- Multi-channel notifications (email, SMS, push)
- Notification history and status

#### 8. Loan Management
- Loan application processing
- Loan approval workflows
- Guarantor management
- Loan repayment tracking
- Loan statistics and reporting

#### 9. Reminder System
- Reminder creation and management
- Recurring reminders
- Reminder notifications
- Reminder completion tracking

## Test Types

### 1. Unit Tests
Each test file contains comprehensive unit tests covering:
- **Happy Path Tests** - Valid inputs and expected outputs
- **Validation Tests** - Invalid inputs and error handling
- **Edge Cases** - Boundary conditions and unusual scenarios
- **Authentication Tests** - Access control and permissions
- **Database Tests** - Data persistence and retrieval
- **Error Handling** - Proper error responses and status codes

### 2. Integration Tests
Tests that verify the interaction between different components:
- Database integration
- Service layer integration
- Middleware functionality
- Request/response flow

### 3. Concurrency Tests
Tests that verify thread safety and concurrent access:
- Concurrent user operations
- Race condition prevention
- Database transaction handling

## Running Tests

### Prerequisites
- Go 1.24.4 or higher
- SQLite3 support
- All project dependencies installed

### Running All Tests
```bash
go test ./test/... -v
```

### Running Specific Test Files
```bash
# Run simple tests only
go test ./test/simple_test.go ./test/setup_test.go -v

# Run authentication tests
go test ./test/auth_test.go ./test/setup_test.go -v

# Run user management tests
go test ./test/user_test.go ./test/setup_test.go -v
```

### Running Individual Test Functions
```bash
# Run specific test function
go test ./test/simple_test.go ./test/setup_test.go -v -run TestSimpleAPIHandlers

# Run test with pattern matching
go test ./test/... -v -run "TestAuth.*"
```

### Test Coverage
```bash
# Generate test coverage report
go test ./test/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Test Database

The tests use an in-memory SQLite database that is created fresh for each test run. The database schema includes all necessary tables for testing:

- Users and authentication
- Chamas and memberships
- Wallets and transactions
- Products and orders
- Meetings and participants
- Notifications and settings
- Loans and repayments
- Reminders

## Test Utilities

### Database Setup
- `setupTestDB()` - Creates fresh in-memory database
- `cleanupTestData()` - Cleans up test data between tests
- `insertTestData()` - Inserts common test data

### HTTP Testing
- `setupTestRouter()` - Creates test HTTP router
- Mock authentication middleware
- Request/response helpers

### Assertion Helpers
- JSON response validation
- Status code verification
- Error message checking

## Mock Data

The tests use realistic mock data that represents typical usage patterns:
- Test users with various roles and statuses
- Sample chamas with different configurations
- Transaction records with various types
- Product catalog with different categories
- Meeting schedules and participants

## Best Practices

The test suite follows Go testing best practices:

1. **Table-Driven Tests** - Using test cases with multiple scenarios
2. **Setup/Teardown** - Proper test isolation and cleanup
3. **Mocking** - Using mocks for external dependencies
4. **Error Testing** - Comprehensive error scenario coverage
5. **Concurrency Testing** - Testing thread safety
6. **Documentation** - Clear test descriptions and comments

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Ensure SQLite3 is properly installed
   - Check database permissions

2. **Import Errors**
   - Verify all dependencies are installed: `go mod tidy`
   - Check Go version compatibility

3. **Test Failures**
   - Some tests may fail due to implementation differences
   - Check actual vs expected behavior
   - Verify test data setup

### Debugging Tests

1. **Verbose Output**
   ```bash
   go test -v ./test/...
   ```

2. **Single Test Debugging**
   ```bash
   go test -v -run TestSpecificFunction ./test/...
   ```

3. **Test Coverage Analysis**
   ```bash
   go test -cover ./test/...
   ```

## Contributing

When adding new tests:

1. Follow the existing test structure
2. Include both positive and negative test cases
3. Add proper test documentation
4. Ensure tests are isolated and repeatable
5. Use meaningful test names and descriptions
6. Include edge case testing
7. Add concurrent access testing where appropriate

## Notes

- Some tests may require adjustment based on actual API implementation
- The test suite is designed to be comprehensive and may need adaptation for specific requirements
- Mock data should be updated to match actual business rules
- Authentication and authorization tests may need JWT secret configuration
