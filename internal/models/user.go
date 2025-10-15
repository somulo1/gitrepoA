package models

import (
	"fmt"
	"time"
)

// UserRole represents user roles in the system
type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// UserStatus represents user account status
type UserStatus string

const (
	UserStatusPending   UserStatus = "pending"
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusVerified  UserStatus = "verified"
)

// ChamaRole represents roles within a chama
type ChamaRole string

const (
	ChamaRoleChairperson ChamaRole = "chairperson"
	ChamaRoleTreasurer   ChamaRole = "treasurer"
	ChamaRoleSecretary   ChamaRole = "secretary"
	ChamaRoleMember      ChamaRole = "member"
	ChamaRoleAssistant   ChamaRole = "assistant"
)

// User represents a user in the VaultKe system
type User struct {
	ID                  string     `json:"id" db:"id"`
	Email               string     `json:"email" db:"email"`
	Phone               string     `json:"phone" db:"phone"`
	FirstName           string     `json:"firstName" db:"first_name"`
	LastName            string     `json:"lastName" db:"last_name"`
	PasswordHash        string     `json:"-" db:"password_hash"`
	Avatar              *string    `json:"avatar,omitempty" db:"avatar"`
	Role                UserRole   `json:"role" db:"role"`
	Status              UserStatus `json:"status" db:"status"`
	IsEmailVerified     bool       `json:"isEmailVerified" db:"is_email_verified"`
	IsPhoneVerified     bool       `json:"isPhoneVerified" db:"is_phone_verified"`
	Language            string     `json:"language" db:"language"`
	Theme               string     `json:"theme" db:"theme"`
	County              *string    `json:"county,omitempty" db:"county"`
	Town                *string    `json:"town,omitempty" db:"town"`
	Latitude            *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude           *float64   `json:"longitude,omitempty" db:"longitude"`
	BusinessType        *string    `json:"businessType,omitempty" db:"business_type"`
	BusinessDescription *string    `json:"businessDescription,omitempty" db:"business_description"`
	Bio                 *string    `json:"bio,omitempty" db:"bio"`
	Occupation          *string    `json:"occupation,omitempty" db:"occupation"`
	DateOfBirth         *string    `json:"dateOfBirth,omitempty" db:"date_of_birth"`
	Gender              *string    `json:"gender,omitempty" db:"gender"`
	Rating              float64    `json:"rating" db:"rating"`
	TotalRatings        int        `json:"totalRatings" db:"total_ratings"`
	CreatedAt           time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time  `json:"updatedAt" db:"updated_at"`
}

// UserRegistration represents user registration data
type UserRegistration struct {
	Email     string  `json:"email" validate:"required,email,max=100,no_sql_injection,no_xss"`
	Phone     string  `json:"phone" validate:"required,phone"`
	FirstName string  `json:"firstName" validate:"required,min=2,max=50,alpha,no_sql_injection,no_xss"`
	LastName  string  `json:"lastName" validate:"required,min=2,max=50,alpha,no_sql_injection,no_xss"`
	Password  string  `json:"password" validate:"required,min=8,max=128"`
	Gender    *string `json:"gender,omitempty" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Language  string  `json:"language" validate:"alpha,max=5"`
}

// UserLogin represents user login data
type UserLogin struct {
	Identifier string `json:"identifier" validate:"required,max=100,no_sql_injection,no_xss"` // email or phone
	Password   string `json:"password" validate:"required,max=128"`
}

// UserProfileUpdate represents user profile update data
type UserProfileUpdate struct {
	FirstName           *string       `json:"firstName,omitempty"`
	LastName            *string       `json:"lastName,omitempty"`
	Phone               *string       `json:"phone,omitempty"`
	Avatar              *string       `json:"avatar,omitempty"`
	Language            *string       `json:"language,omitempty"`
	Theme               *string       `json:"theme,omitempty"`
	County              *string       `json:"county,omitempty"`
	Town                *string       `json:"town,omitempty"`
	Bio                 *string       `json:"bio,omitempty"`
	Occupation          *string       `json:"occupation,omitempty"`
	DateOfBirth         *FlexibleDate `json:"dateOfBirth,omitempty"`
	Gender              *string       `json:"gender,omitempty"`
	Latitude            *float64      `json:"latitude,omitempty"`
	Longitude           *float64      `json:"longitude,omitempty"`
	BusinessType        *string       `json:"businessType,omitempty"`
	BusinessDescription *string       `json:"businessDescription,omitempty"`
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// IsActive checks if the user account is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive || u.Status == UserStatusVerified
}

// IsVerified checks if both email and phone are verified
func (u *User) IsVerified() bool {
	return u.IsEmailVerified && u.IsPhoneVerified
}

// GetLocation returns the user's location as a formatted string
func (u *User) GetLocation() string {
	if u.Town != nil && u.County != nil {
		return *u.Town + ", " + *u.County
	}
	if u.County != nil {
		return *u.County
	}
	return ""
}

// HasCoordinates checks if the user has location coordinates
func (u *User) HasCoordinates() bool {
	return u.Latitude != nil && u.Longitude != nil
}

// GetCoordinates returns the user's coordinates
func (u *User) GetCoordinates() (float64, float64) {
	if u.HasCoordinates() {
		return *u.Latitude, *u.Longitude
	}
	return 0, 0
}

// FlexibleDate is a custom type that can handle both date and datetime formats
type FlexibleDate struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for flexible date parsing
func (fd *FlexibleDate) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// Skip empty strings
	if str == "" {
		return nil
	}

	// Try different date formats
	formats := []string{
		"2006-01-02",                // Date only (YYYY-MM-DD)
		"2006-01-02T15:04:05Z07:00", // Full datetime with timezone
		"2006-01-02T15:04:05Z",      // Full datetime UTC
		"2006-01-02T15:04:05",       // Full datetime without timezone
		"2006-01-02 15:04:05",       // Date and time with space
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			fd.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse date: %s", str)
}
