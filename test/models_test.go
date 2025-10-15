package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"vaultke-backend/internal/models"
)

// TestUserModel tests the User model methods
func TestUserModel(t *testing.T) {
	t.Run("UserModelMethods", func(t *testing.T) {
		user := models.User{
			ID:              "test-user-model",
			Email:           "modeltest@example.com",
			Phone:           "+254700000099",
			FirstName:       "Model",
			LastName:        "Test",
			Role:            models.UserRoleUser,
			Status:          models.UserStatusActive,
			IsEmailVerified: true,
			IsPhoneVerified: true,
		}

		// Test GetFullName method
		assert.Equal(t, "Model Test", user.GetFullName())

		// Test IsActive method
		assert.True(t, user.IsActive())

		// Test IsVerified method
		assert.True(t, user.IsVerified())
	})

	t.Run("UserLocationMethods", func(t *testing.T) {
		county := "Nairobi"
		town := "Nairobi"
		lat := -1.2921
		lng := 36.8219

		user := models.User{
			County:    &county,
			Town:      &town,
			Latitude:  &lat,
			Longitude: &lng,
		}

		// Test GetLocation method
		assert.Equal(t, "Nairobi, Nairobi", user.GetLocation())

		// Test HasCoordinates method
		assert.True(t, user.HasCoordinates())

		// Test GetCoordinates method
		latitude, longitude := user.GetCoordinates()
		assert.Equal(t, -1.2921, latitude)
		assert.Equal(t, 36.8219, longitude)
	})

	t.Run("UserStatusMethods", func(t *testing.T) {
		// Test active user
		activeUser := models.User{Status: models.UserStatusActive}
		assert.True(t, activeUser.IsActive())

		// Test verified user
		verifiedUser := models.User{Status: models.UserStatusVerified}
		assert.True(t, verifiedUser.IsActive())

		// Test pending user
		pendingUser := models.User{Status: models.UserStatusPending}
		assert.False(t, pendingUser.IsActive())

		// Test suspended user
		suspendedUser := models.User{Status: models.UserStatusSuspended}
		assert.False(t, suspendedUser.IsActive())
	})

	t.Run("UserVerificationMethods", func(t *testing.T) {
		// Test fully verified user
		verifiedUser := models.User{
			IsEmailVerified: true,
			IsPhoneVerified: true,
		}
		assert.True(t, verifiedUser.IsVerified())

		// Test partially verified user
		partialUser := models.User{
			IsEmailVerified: true,
			IsPhoneVerified: false,
		}
		assert.False(t, partialUser.IsVerified())

		// Test unverified user
		unverifiedUser := models.User{
			IsEmailVerified: false,
			IsPhoneVerified: false,
		}
		assert.False(t, unverifiedUser.IsVerified())
	})
}

// TestChamaModel tests the Chama model methods
func TestChamaModel(t *testing.T) {
	t.Run("ChamaModelMethods", func(t *testing.T) {
		description := "A test chama"
		chama := models.Chama{
			ID:          "test-chama",
			Name:        "Test Chama",
			Description: &description,
			Type:        models.ChamaTypeSavings,
			Status:      models.ChamaStatusActive,
		}

		// Test basic properties
		assert.Equal(t, "test-chama", chama.ID)
		assert.Equal(t, "Test Chama", chama.Name)
		assert.Equal(t, "A test chama", *chama.Description)
		assert.Equal(t, models.ChamaTypeSavings, chama.Type)
		assert.Equal(t, models.ChamaStatusActive, chama.Status)
	})
}

// TestWalletModel tests the Wallet model methods
func TestWalletModel(t *testing.T) {
	t.Run("WalletModelMethods", func(t *testing.T) {
		wallet := models.Wallet{
			ID:       "test-wallet",
			OwnerID:  "test-user",
			Type:     models.WalletTypePersonal,
			Balance:  1000.0,
			Currency: "KES",
			IsActive: true,
			IsLocked: false,
		}

		// Test basic properties
		assert.Equal(t, "test-wallet", wallet.ID)
		assert.Equal(t, "test-user", wallet.OwnerID)
		assert.Equal(t, models.WalletTypePersonal, wallet.Type)
		assert.Equal(t, 1000.0, wallet.Balance)
		assert.Equal(t, "KES", wallet.Currency)
		assert.True(t, wallet.IsActive)
		assert.False(t, wallet.IsLocked)
	})
}

// TestReminderModel tests the Reminder model methods
func TestReminderModel(t *testing.T) {
	t.Run("ReminderModelMethods", func(t *testing.T) {
		description := "This is a test reminder"
		reminder := models.Reminder{
			ID:               "test-reminder",
			UserID:           "test-user",
			Title:            "Test Reminder",
			Description:      &description,
			ReminderType:     models.ReminderTypeOnce,
			ScheduledAt:      time.Now().Add(time.Hour),
			IsEnabled:        true,
			IsCompleted:      false,
			NotificationSent: false,
		}

		// Test basic properties
		assert.Equal(t, "test-reminder", reminder.ID)
		assert.Equal(t, "test-user", reminder.UserID)
		assert.Equal(t, "Test Reminder", reminder.Title)
		assert.Equal(t, "This is a test reminder", *reminder.Description)
		assert.Equal(t, models.ReminderTypeOnce, reminder.ReminderType)
		assert.True(t, reminder.IsEnabled)
		assert.False(t, reminder.IsCompleted)
		assert.False(t, reminder.NotificationSent)
	})
}

// TestFlexibleDate tests the FlexibleDate custom type
func TestFlexibleDate(t *testing.T) {
	t.Run("FlexibleDateUnmarshaling", func(t *testing.T) {
		// Test date only format
		dateJSON := `"2023-01-15"`
		var fd models.FlexibleDate
		err := fd.UnmarshalJSON([]byte(dateJSON))
		assert.NoError(t, err)

		// Test datetime format
		datetimeJSON := `"2023-01-15T10:30:00Z"`
		var fd2 models.FlexibleDate
		err = fd2.UnmarshalJSON([]byte(datetimeJSON))
		assert.NoError(t, err)

		// Test empty string
		emptyJSON := `""`
		var fd3 models.FlexibleDate
		err = fd3.UnmarshalJSON([]byte(emptyJSON))
		assert.NoError(t, err)
	})
}

// TestUserRegistration tests the UserRegistration struct
func TestUserRegistration(t *testing.T) {
	t.Run("UserRegistrationStruct", func(t *testing.T) {
		registration := models.UserRegistration{
			Email:     "test@example.com",
			Phone:     "+254712345678",
			FirstName: "Test",
			LastName:  "User",
			Password:  "SecurePass123!",
			Language:  "en",
		}

		// Test basic properties
		assert.Equal(t, "test@example.com", registration.Email)
		assert.Equal(t, "+254712345678", registration.Phone)
		assert.Equal(t, "Test", registration.FirstName)
		assert.Equal(t, "User", registration.LastName)
		assert.Equal(t, "en", registration.Language)
	})
}

// TestUserLogin tests the UserLogin struct
func TestUserLogin(t *testing.T) {
	t.Run("UserLoginStruct", func(t *testing.T) {
		login := models.UserLogin{
			Identifier: "test@example.com",
			Password:   "SecurePass123!",
		}

		// Test basic properties
		assert.Equal(t, "test@example.com", login.Identifier)
		assert.Equal(t, "SecurePass123!", login.Password)
	})
}

// TestUserProfileUpdate tests the UserProfileUpdate struct
func TestUserProfileUpdate(t *testing.T) {
	t.Run("UserProfileUpdateStruct", func(t *testing.T) {
		firstName := "Updated"
		lastName := "Name"
		language := "sw"

		update := models.UserProfileUpdate{
			FirstName: &firstName,
			LastName:  &lastName,
			Language:  &language,
		}

		// Test basic properties
		assert.Equal(t, "Updated", *update.FirstName)
		assert.Equal(t, "Name", *update.LastName)
		assert.Equal(t, "sw", *update.Language)
	})
}
