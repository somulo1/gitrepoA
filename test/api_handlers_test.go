package test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultke-backend/test/helpers"
)

func TestAuthHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("Register", func(t *testing.T) {
		registerData := map[string]interface{}{
			"email":     "newuser@example.com",
			"phone":     "+254700000200",
			"firstName": "New",
			"lastName":  "User",
			"password":  "password123",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/register", registerData, nil)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user")
	})

	t.Run("RegisterWithDuplicateEmail", func(t *testing.T) {
		registerData := map[string]interface{}{
			"email":     ts.Users["user"].Email,
			"phone":     "+254700000201",
			"firstName": "Duplicate",
			"lastName":  "User",
			"password":  "password123",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/register", registerData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("RegisterWithInvalidData", func(t *testing.T) {
		registerData := map[string]interface{}{
			"email":     "invalid-email",
			"phone":     "invalid-phone",
			"firstName": "",
			"lastName":  "",
			"password":  "123",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/register", registerData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Login", func(t *testing.T) {
		loginData := map[string]interface{}{
			"email":    ts.Users["user"].Email,
			"password": ts.Users["user"].Password,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/login", loginData, nil)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user")
	})

	t.Run("LoginWithInvalidCredentials", func(t *testing.T) {
		loginData := map[string]interface{}{
			"email":    ts.Users["user"].Email,
			"password": "wrongpassword",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/login", loginData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("LoginWithNonexistentUser", func(t *testing.T) {
		loginData := map[string]interface{}{
			"email":    "nonexistent@example.com",
			"password": "password123",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/login", loginData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("GetProfile", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/users/profile", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "user")
	})

	t.Run("GetProfileWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/users/profile", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		updateData := map[string]interface{}{
			"firstName": "Updated",
			"lastName":  "Name",
			"bio":       "Updated bio",
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/users/profile", updateData, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("UpdateProfileWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		updateData := map[string]interface{}{
			"firstName": "",
			"lastName":  "",
			"email":     "invalid-email",
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/users/profile", updateData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Logout", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/logout", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("LogoutWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/logout", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("RefreshToken", func(t *testing.T) {
		refreshData := map[string]interface{}{
			"token": ts.Users["user"].Token,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/refresh", refreshData, nil)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "token")
	})

	t.Run("RefreshTokenWithInvalidToken", func(t *testing.T) {
		refreshData := map[string]interface{}{
			"token": "invalid-token",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/auth/refresh", refreshData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}

func TestUserHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("GetUsers", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/users", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "users")
		assert.Contains(t, response, "total")
	})

	t.Run("GetUsersWithPagination", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/users?limit=5&offset=0", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "users")
		assert.Contains(t, response, "total")
	})

	t.Run("GetUsersWithSearch", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/users?search=user", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "users")
	})

	t.Run("GetUsersWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/users", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("UploadAvatar", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		avatarData := map[string]interface{}{
			"avatar": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRT/2wBDAQMEBAUEBQkFBQkUDQsNFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k=",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/users/avatar", avatarData, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("UploadAvatarWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		avatarData := map[string]interface{}{
			"avatar": "invalid-base64-data",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/users/avatar", avatarData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("UploadAvatarWithoutAuth", func(t *testing.T) {
		avatarData := map[string]interface{}{
			"avatar": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRT/2wBDAQMEBAUEBQkFBQkUDQsNFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k=",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/users/avatar", avatarData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}

func TestChamaHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("GetChamas", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/chamas", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "chamas")
		assert.Contains(t, response, "total")
	})

	t.Run("GetChamasWithFilters", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/chamas?county=Nairobi&type=savings", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "chamas")
	})

	t.Run("CreateChama", func(t *testing.T) {
		headers := ts.GetAuthHeaders("chairperson")
		chamaData := map[string]interface{}{
			"name":                  "Test API Chama",
			"description":           "Test chama for API testing",
			"type":                  "savings",
			"county":                "Nairobi",
			"town":                  "Nairobi",
			"contributionAmount":    1000.0,
			"contributionFrequency": "monthly",
			"maxMembers":            50,
			"isPublic":              true,
			"requiresApproval":      false,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas", chamaData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "chama")
	})

	t.Run("CreateChamaWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("chairperson")
		chamaData := map[string]interface{}{
			"name":                  "",
			"contributionAmount":    -100.0,
			"contributionFrequency": "invalid",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas", chamaData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("CreateChamaWithoutAuth", func(t *testing.T) {
		chamaData := map[string]interface{}{
			"name":                  "Unauthorized Chama",
			"type":                  "savings",
			"county":                "Nairobi",
			"town":                  "Nairobi",
			"contributionAmount":    1000.0,
			"contributionFrequency": "monthly",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas", chamaData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("GetUserChamas", func(t *testing.T) {
		headers := ts.GetAuthHeaders("chairperson")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/chamas/my", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "chamas")
	})

	t.Run("GetChama", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/chamas/api-test-chama", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "chama")
	})

	t.Run("GetNonexistentChama", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/chamas/nonexistent-chama", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("UpdateChama", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("update-api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("chairperson")
		updateData := map[string]interface{}{
			"name":        "Updated API Chama",
			"description": "Updated description",
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/chamas/update-api-test-chama", updateData, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("UpdateChamaWithoutPermission", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("update-unauthorized-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		updateData := map[string]interface{}{
			"name": "Unauthorized Update",
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/chamas/update-unauthorized-chama", updateData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("DeleteChama", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("delete-api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("chairperson")
		w := helpers.MakeRequest(ts.Router, "DELETE", "/api/v1/chamas/delete-api-test-chama", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("DeleteChamaWithoutPermission", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("delete-unauthorized-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "DELETE", "/api/v1/chamas/delete-unauthorized-chama", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("GetChamaMembers", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("members-api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/chamas/members-api-test-chama/members", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "members")
	})

	t.Run("JoinChama", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("join-api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas/join-api-test-chama/join", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("JoinChamaAlreadyMember", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("join-duplicate-api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("chairperson")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas/join-duplicate-api-test-chama/join", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("LeaveChama", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("leave-api-test-chama")
		require.NoError(t, err)

		// Join first
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas/leave-api-test-chama/join", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Then leave
		w = helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas/leave-api-test-chama/leave", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("LeaveChamaNotMember", func(t *testing.T) {
		// Create a test chama first
		err := ts.CreateTestChama("leave-not-member-api-test-chama")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/chamas/leave-not-member-api-test-chama/leave", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})
}

func TestWalletHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("GetWallets", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/wallets", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "wallets")
	})

	t.Run("GetWalletsWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/wallets", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("GetWallet", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("api-test-wallet", ts.Users["user"].ID, "personal", 1000.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/wallets/api-test-wallet", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "wallet")
	})

	t.Run("GetWalletWithoutPermission", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("unauthorized-wallet", ts.Users["admin"].ID, "personal", 1000.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/wallets/unauthorized-wallet", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("GetWalletBalance", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("balance-api-test-wallet", ts.Users["user"].ID, "personal", 1500.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/wallets/balance", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "balance")
	})

	t.Run("DepositMoney", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("deposit-api-test-wallet", ts.Users["user"].ID, "personal", 1000.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		depositData := map[string]interface{}{
			"amount":        500.0,
			"paymentMethod": "mpesa",
			"description":   "Test deposit",
			"reference":     "TEST123",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/wallets/deposit", depositData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "transaction")
	})

	t.Run("DepositMoneyWithInvalidAmount", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		depositData := map[string]interface{}{
			"amount":        -100.0,
			"paymentMethod": "mpesa",
			"description":   "Invalid deposit",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/wallets/deposit", depositData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("TransferMoney", func(t *testing.T) {
		// Create source and destination wallets
		err := ts.CreateTestWallet("transfer-source-wallet", ts.Users["user"].ID, "personal", 2000.0)
		require.NoError(t, err)
		err = ts.CreateTestWallet("transfer-dest-wallet", ts.Users["admin"].ID, "personal", 500.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		transferData := map[string]interface{}{
			"fromWalletId": "transfer-source-wallet",
			"toWalletId":   "transfer-dest-wallet",
			"amount":       300.0,
			"description":  "Test transfer",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/wallets/transfer", transferData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "transaction")
	})

	t.Run("TransferMoneyInsufficientFunds", func(t *testing.T) {
		// Create source and destination wallets
		err := ts.CreateTestWallet("insufficient-source-wallet", ts.Users["user"].ID, "personal", 100.0)
		require.NoError(t, err)
		err = ts.CreateTestWallet("insufficient-dest-wallet", ts.Users["admin"].ID, "personal", 500.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		transferData := map[string]interface{}{
			"fromWalletId": "insufficient-source-wallet",
			"toWalletId":   "insufficient-dest-wallet",
			"amount":       1000.0,
			"description":  "Insufficient funds transfer",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/wallets/transfer", transferData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("WithdrawMoney", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("withdraw-api-test-wallet", ts.Users["user"].ID, "personal", 1000.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		withdrawData := map[string]interface{}{
			"amount":        200.0,
			"paymentMethod": "mpesa",
			"description":   "Test withdrawal",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/wallets/withdraw", withdrawData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "transaction")
	})

	t.Run("WithdrawMoneyInsufficientFunds", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("withdraw-insufficient-wallet", ts.Users["user"].ID, "personal", 100.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		withdrawData := map[string]interface{}{
			"amount":        1000.0,
			"paymentMethod": "mpesa",
			"description":   "Insufficient funds withdrawal",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/wallets/withdraw", withdrawData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestMarketplaceHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("GetProducts", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/marketplace/products", nil, nil)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "products")
		assert.Contains(t, response, "total")
	})

	t.Run("GetProductsWithFilters", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/marketplace/products?category=electronics&county=Nairobi", nil, nil)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "products")
	})

	t.Run("GetProduct", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("api-test-product")
		require.NoError(t, err)

		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/marketplace/products/api-test-product", nil, nil)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "product")
	})

	t.Run("GetNonexistentProduct", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/marketplace/products/nonexistent-product", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("CreateProduct", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		productData := map[string]interface{}{
			"name":        "API Test Product",
			"description": "Test product for API testing",
			"category":    "electronics",
			"price":       1500.0,
			"images":      []string{"https://example.com/image1.jpg"},
			"stock":       20,
			"minOrder":    1,
			"county":      "Nairobi",
			"town":        "Nairobi",
			"tags":        []string{"electronics", "test"},
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/products", productData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "product")
	})

	t.Run("CreateProductWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		productData := map[string]interface{}{
			"name":        "",
			"description": "",
			"price":       -100.0,
			"stock":       -5,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/products", productData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("CreateProductWithoutAuth", func(t *testing.T) {
		productData := map[string]interface{}{
			"name":        "Unauthorized Product",
			"description": "Test product",
			"category":    "electronics",
			"price":       1000.0,
			"county":      "Nairobi",
			"town":        "Nairobi",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/products", productData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("UpdateProduct", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("update-api-test-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		updateData := map[string]interface{}{
			"name":        "Updated API Product",
			"description": "Updated description",
			"price":       2000.0,
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/marketplace/products/update-api-test-product", updateData, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("UpdateProductWithoutPermission", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("update-unauthorized-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("admin")
		updateData := map[string]interface{}{
			"name": "Unauthorized Update",
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/marketplace/products/update-unauthorized-product", updateData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("DeleteProduct", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("delete-api-test-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "DELETE", "/api/v1/marketplace/products/delete-api-test-product", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("DeleteProductWithoutPermission", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("delete-unauthorized-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("admin")
		w := helpers.MakeRequest(ts.Router, "DELETE", "/api/v1/marketplace/products/delete-unauthorized-product", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("GetCart", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/marketplace/cart", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "cart")
	})

	t.Run("AddToCart", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("cart-api-test-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("admin")
		cartData := map[string]interface{}{
			"productId": "cart-api-test-product",
			"quantity":  2,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/cart", cartData, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	})

	t.Run("AddToCartInvalidQuantity", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("cart-invalid-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("admin")
		cartData := map[string]interface{}{
			"productId": "cart-invalid-product",
			"quantity":  0,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/cart", cartData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("GetOrders", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/marketplace/orders", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "orders")
	})

	t.Run("CreateOrder", func(t *testing.T) {
		// Create a test product first
		err := ts.CreateTestProduct("order-api-test-product")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("admin")
		orderData := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"productId": "order-api-test-product",
					"quantity":  1,
				},
			},
			"deliveryAddress": "Test Address",
			"deliveryPhone":   "+254700000000",
			"paymentMethod":   "mpesa",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/orders", orderData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "order")
	})

	t.Run("CreateOrderWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		orderData := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"productId": "nonexistent-product",
					"quantity":  1,
				},
			},
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/marketplace/orders", orderData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestPaymentHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("InitiateMpesaSTK", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		stkData := map[string]interface{}{
			"amount":      1000.0,
			"phoneNumber": "+254700000000",
			"description": "Test STK payment",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/payments/mpesa/stk", stkData, headers)
		// This might fail without real M-Pesa credentials, so check for both success and specific error
		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			assert.Contains(t, response, "checkoutRequestId")
		} else {
			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		}
	})

	t.Run("InitiateMpesaSTKWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		stkData := map[string]interface{}{
			"amount":      -100.0,
			"phoneNumber": "invalid-phone",
			"description": "",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/payments/mpesa/stk", stkData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("InitiateMpesaSTKWithoutAuth", func(t *testing.T) {
		stkData := map[string]interface{}{
			"amount":      1000.0,
			"phoneNumber": "+254700000000",
			"description": "Test STK payment",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/payments/mpesa/stk", stkData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("GetMpesaTransactionStatus", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/payments/mpesa/status/test-checkout-request-id", nil, headers)
		// This might fail without real M-Pesa transaction, so check for both success and specific error
		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			assert.Contains(t, response, "status")
		} else {
			helpers.AssertErrorResponse(t, w, http.StatusNotFound)
		}
	})

	t.Run("HandleMpesaCallback", func(t *testing.T) {
		callbackData := map[string]interface{}{
			"ResultCode":        0,
			"ResultDesc":        "The service request is processed successfully.",
			"CheckoutRequestID": "test-checkout-request-id",
			"CallbackMetadata": map[string]interface{}{
				"Item": []map[string]interface{}{
					{
						"Name":  "Amount",
						"Value": 1000.0,
					},
					{
						"Name":  "MpesaReceiptNumber",
						"Value": "TEST123456789",
					},
					{
						"Name":  "PhoneNumber",
						"Value": 254700000000,
					},
				},
			},
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/payments/mpesa/callback", callbackData, nil)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("HandleMpesaCallbackWithInvalidData", func(t *testing.T) {
		callbackData := map[string]interface{}{
			"ResultCode": "invalid",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/payments/mpesa/callback", callbackData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestMeetingHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	// Create a test chama first
	err := ts.CreateTestChama("meeting-test-chama")
	require.NoError(t, err)

	t.Run("GetMeetings", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/meetings?chamaId=meeting-test-chama", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "meetings")
	})

	t.Run("GetMeetingsWithoutChamaId", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/meetings", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("CreateMeeting", func(t *testing.T) {
		headers := ts.GetAuthHeaders("chairperson")
		meetingData := map[string]interface{}{
			"chamaId":     "meeting-test-chama",
			"title":       "API Test Meeting",
			"description": "Test meeting for API testing",
			"scheduledAt": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"duration":    60,
			"location":    "Nairobi Office",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/meetings", meetingData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "meeting")
	})

	t.Run("CreateMeetingWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("chairperson")
		meetingData := map[string]interface{}{
			"chamaId":     "meeting-test-chama",
			"title":       "",
			"scheduledAt": "invalid-date",
			"duration":    -30,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/meetings", meetingData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("CreateMeetingWithoutAuth", func(t *testing.T) {
		meetingData := map[string]interface{}{
			"chamaId":     "meeting-test-chama",
			"title":       "Unauthorized Meeting",
			"scheduledAt": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"duration":    60,
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/meetings", meetingData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("GetMeeting", func(t *testing.T) {
		// Create a test meeting first
		meetingID := uuid.New().String()
		query := `
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, created_by, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, meetingID, "meeting-test-chama", "Test Meeting", "Test Description", time.Now().Add(24*time.Hour), 60, ts.Users["chairperson"].ID, "scheduled")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/meetings/"+meetingID, nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "meeting")
	})

	t.Run("GetNonexistentMeeting", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/meetings/nonexistent-meeting", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("UpdateMeeting", func(t *testing.T) {
		// Create a test meeting first
		meetingID := uuid.New().String()
		query := `
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, created_by, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, meetingID, "meeting-test-chama", "Update Test Meeting", "Update Test Description", time.Now().Add(24*time.Hour), 60, ts.Users["chairperson"].ID, "scheduled")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("chairperson")
		updateData := map[string]interface{}{
			"title":       "Updated Meeting Title",
			"description": "Updated meeting description",
			"duration":    90,
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/meetings/"+meetingID, updateData, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("UpdateMeetingWithoutPermission", func(t *testing.T) {
		// Create a test meeting first
		meetingID := uuid.New().String()
		query := `
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, created_by, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, meetingID, "meeting-test-chama", "Unauthorized Update Test Meeting", "Test Description", time.Now().Add(24*time.Hour), 60, ts.Users["chairperson"].ID, "scheduled")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		updateData := map[string]interface{}{
			"title": "Unauthorized Update",
		}

		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/meetings/"+meetingID, updateData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("DeleteMeeting", func(t *testing.T) {
		// Create a test meeting first
		meetingID := uuid.New().String()
		query := `
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, created_by, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, meetingID, "meeting-test-chama", "Delete Test Meeting", "Test Description", time.Now().Add(24*time.Hour), 60, ts.Users["chairperson"].ID, "scheduled")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("chairperson")
		w := helpers.MakeRequest(ts.Router, "DELETE", "/api/v1/meetings/"+meetingID, nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("DeleteMeetingWithoutPermission", func(t *testing.T) {
		// Create a test meeting first
		meetingID := uuid.New().String()
		query := `
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, created_by, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, meetingID, "meeting-test-chama", "Unauthorized Delete Test Meeting", "Test Description", time.Now().Add(24*time.Hour), 60, ts.Users["chairperson"].ID, "scheduled")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "DELETE", "/api/v1/meetings/"+meetingID, nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("JoinMeeting", func(t *testing.T) {
		// Create a test meeting first
		meetingID := uuid.New().String()
		query := `
			INSERT INTO meetings (id, chama_id, title, description, scheduled_at, duration, created_by, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, meetingID, "meeting-test-chama", "Join Test Meeting", "Test Description", time.Now().Add(24*time.Hour), 60, ts.Users["chairperson"].ID, "scheduled")
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/meetings/"+meetingID+"/join", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("JoinNonexistentMeeting", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/meetings/nonexistent-meeting/join", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})
}

func TestNotificationHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("GetNotifications", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/notifications", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "notifications")
	})

	t.Run("GetNotificationsWithPagination", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/notifications?limit=5&offset=0", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "notifications")
	})

	t.Run("GetNotificationsWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/notifications", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("MarkNotificationAsRead", func(t *testing.T) {
		// Create a test notification first
		notificationID := uuid.New().String()
		query := `
			INSERT INTO notifications (id, user_id, type, title, message, is_read, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, notificationID, ts.Users["user"].ID, "info", "Test Notification", "Test message", false, time.Now())
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/notifications/"+notificationID+"/read", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("MarkNonexistentNotificationAsRead", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "PUT", "/api/v1/notifications/nonexistent-notification/read", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("MarkAllNotificationsAsRead", func(t *testing.T) {
		// Create multiple test notifications
		for i := 0; i < 3; i++ {
			notificationID := uuid.New().String()
			query := `
				INSERT INTO notifications (id, user_id, type, title, message, is_read, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`
			_, err := ts.DB.DB.Exec(query, notificationID, ts.Users["user"].ID, "info", "Bulk Test Notification "+strconv.Itoa(i), "Test message", false, time.Now())
			require.NoError(t, err)
		}

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/notifications/read-all", nil, headers)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("MarkAllNotificationsAsReadWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/notifications/read-all", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}

func TestContributionHandlers(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	// Create a test chama first
	err := ts.CreateTestChama("contribution-test-chama")
	require.NoError(t, err)

	t.Run("GetContributions", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/contributions?chamaId=contribution-test-chama", nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "contributions")
	})

	t.Run("GetContributionsWithoutChamaId", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/contributions", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("GetContributionsWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/contributions?chamaId=contribution-test-chama", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("MakeContribution", func(t *testing.T) {
		// Create a test wallet first
		err := ts.CreateTestWallet("contribution-wallet", ts.Users["user"].ID, "personal", 5000.0)
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		contributionData := map[string]interface{}{
			"chamaId":       "contribution-test-chama",
			"amount":        1000.0,
			"description":   "Monthly contribution",
			"type":          "regular",
			"paymentMethod": "wallet",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/contributions", contributionData, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		assert.Contains(t, response, "contribution")
	})

	t.Run("MakeContributionWithInvalidData", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		contributionData := map[string]interface{}{
			"chamaId":       "contribution-test-chama",
			"amount":        -100.0,
			"paymentMethod": "invalid",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/contributions", contributionData, headers)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("MakeContributionWithoutAuth", func(t *testing.T) {
		contributionData := map[string]interface{}{
			"chamaId":       "contribution-test-chama",
			"amount":        1000.0,
			"description":   "Unauthorized contribution",
			"paymentMethod": "wallet",
		}

		w := helpers.MakeRequest(ts.Router, "POST", "/api/v1/contributions", contributionData, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("GetContribution", func(t *testing.T) {
		// Create a test contribution first
		contributionID := uuid.New().String()
		query := `
			INSERT INTO contributions (id, chama_id, user_id, amount, description, type, payment_method, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := ts.DB.DB.Exec(query, contributionID, "contribution-test-chama", ts.Users["user"].ID, 1000.0, "Test contribution", "regular", "wallet", "completed", time.Now())
		require.NoError(t, err)

		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/contributions/"+contributionID, nil, headers)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		assert.Contains(t, response, "contribution")
	})

	t.Run("GetNonexistentContribution", func(t *testing.T) {
		headers := ts.GetAuthHeaders("user")
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/contributions/nonexistent-contribution", nil, headers)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("GetContributionWithoutAuth", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/api/v1/contributions/some-contribution", nil, nil)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})
}

func TestHealthEndpoint(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("HealthCheck", func(t *testing.T) {
		w := helpers.MakeRequest(ts.Router, "GET", "/health", nil, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
		assert.Contains(t, response, "message")
		assert.Contains(t, response, "version")
	})
}
