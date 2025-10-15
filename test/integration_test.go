package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultke-backend/test/helpers"
)

func TestSimpleIntegration(t *testing.T) {
	ts := helpers.NewTestSuite(t)
	defer ts.Cleanup()

	t.Run("Basic_Setup_Test", func(t *testing.T) {
		// Test that we can create a test suite
		assert.NotNil(t, ts)
		assert.NotNil(t, ts.DB)
		assert.NotNil(t, ts.Router)
		assert.NotNil(t, ts.Config)
		assert.NotNil(t, ts.Users)

		// Test that test users were created
		assert.Contains(t, ts.Users, "admin")
		assert.Contains(t, ts.Users, "user")
		assert.Contains(t, ts.Users, "chairperson")

		// Test that we can get auth headers
		userHeaders := ts.GetAuthHeaders("user")
		assert.NotNil(t, userHeaders)
		assert.Contains(t, userHeaders, "Authorization")
		assert.NotEmpty(t, userHeaders["Authorization"])

		// Test that we can create a wallet
		err := ts.CreateTestWallet("simple-test-wallet", ts.Users["user"].ID, "personal", 1000.0)
		require.NoError(t, err)

		// Test that we can create a chama
		err = ts.CreateTestChama("simple-test-chama")
		require.NoError(t, err)

		// Test that we can create a product
		err = ts.CreateTestProduct("simple-test-product")
		require.NoError(t, err)
	})
}
