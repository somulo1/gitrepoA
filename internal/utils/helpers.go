package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// East African Time timezone
var EATLocation *time.Location

func init() {
	// Load East African Time timezone (UTC+3)
	var err error
	EATLocation, err = time.LoadLocation("Africa/Nairobi")
	if err != nil {
		// Fallback to fixed offset if timezone data is not available
		EATLocation = time.FixedZone("EAT", 3*60*60) // UTC+3
		log.Printf("Warning: Could not load Africa/Nairobi timezone, using fixed offset: %v", err)
	}
}

// NowEAT returns the current time in East African Time
func NowEAT() time.Time {
	return time.Now().In(EATLocation)
}

// ToEAT converts a time to East African Time
func ToEAT(t time.Time) time.Time {
	return t.In(EATLocation)
}

// ParseTimeEAT parses a time string and returns it in EAT
func ParseTimeEAT(layout, value string) (time.Time, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(EATLocation), nil
}

// FormatTimeEAT formats a time in EAT with the given layout
func FormatTimeEAT(t time.Time, layout string) string {
	return t.In(EATLocation).Format(layout)
}

// FormatCurrency formats a number as Kenyan Shilling currency
func FormatCurrency(amount float64) string {
	return fmt.Sprintf("KSh %.2f", amount)
}

// FormatPhoneNumber formats a phone number to international format
func FormatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Handle different formats
	if strings.HasPrefix(cleaned, "254") {
		return "+" + cleaned
	} else if strings.HasPrefix(cleaned, "0") && len(cleaned) == 10 {
		return "+254" + cleaned[1:]
	} else if len(cleaned) == 9 {
		return "+254" + cleaned
	}

	// Return as-is if format is unclear
	return phone
}

// ParsePhoneNumber extracts the phone number without country code
func ParsePhoneNumber(phone string) string {
	formatted := FormatPhoneNumber(phone)
	if strings.HasPrefix(formatted, "+254") {
		return "0" + formatted[4:]
	}
	return phone
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}

// GenerateOTP generates a numeric OTP of specified length
func GenerateOTP(length int) string {
	if length <= 0 {
		length = 6
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "123456" // Fallback
	}

	otp := ""
	for i := 0; i < length; i++ {
		otp += strconv.Itoa(int(bytes[i]) % 10)
	}

	return otp
}

// Slugify converts a string to a URL-friendly slug
func Slugify(text string) string {
	// Convert to lowercase
	slug := strings.ToLower(text)

	// Replace spaces and special characters with hyphens
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// TruncateString truncates a string to specified length with ellipsis
func TruncateString(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	if maxLength <= 3 {
		return text[:maxLength]
	}

	return text[:maxLength-3] + "..."
}

// CalculateDistance calculates distance between two coordinates in kilometers
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// FormatDate formats a time.Time to a readable string in EAT
func FormatDate(t time.Time, format string) string {
	// Convert to EAT first
	eatTime := ToEAT(t)

	switch format {
	case "short":
		return eatTime.Format("02/01/2006")
	case "long":
		return eatTime.Format("02 January 2006")
	case "datetime":
		return eatTime.Format("02/01/2006 15:04")
	case "time":
		return eatTime.Format("15:04")
	case "iso":
		return eatTime.Format(time.RFC3339)
	case "api":
		// Format for API responses - ISO format but in EAT
		return eatTime.Format("2006-01-02T15:04:05Z07:00")
	default:
		return eatTime.Format("02/01/2006")
	}
}

// ParseDate parses a date string in various formats
func ParseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"2006-01-02 15:04:05",
		"02/01/2006 15:04",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// GetTimeAgo returns a human-readable time difference in EAT
func GetTimeAgo(t time.Time) string {
	now := NowEAT()
	eatTime := ToEAT(t)
	diff := now.Sub(eatTime)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(diff.Hours() / (24 * 365))
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// RoundToDecimalPlaces rounds a float to specified decimal places
func RoundToDecimalPlaces(value float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(value*multiplier) / multiplier
}

// CalculatePercentage calculates percentage of part from total
func CalculatePercentage(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (part / total) * 100
}

// CalculatePercentageChange calculates percentage change between old and new values
func CalculatePercentageChange(oldValue, newValue float64) float64 {
	if oldValue == 0 {
		if newValue == 0 {
			return 0
		}
		return 100 // 100% increase from 0
	}
	return ((newValue - oldValue) / oldValue) * 100
}

// IsWeekend checks if a date is weekend (Saturday or Sunday)
func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// IsBusinessDay checks if a date is a business day (Monday to Friday)
func IsBusinessDay(t time.Time) bool {
	return !IsWeekend(t)
}

// GetNextBusinessDay returns the next business day
func GetNextBusinessDay(t time.Time) time.Time {
	next := t.AddDate(0, 0, 1)
	for IsWeekend(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// GetStartOfDay returns the start of day (00:00:00) for given time
func GetStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// GetEndOfDay returns the end of day (23:59:59) for given time
func GetEndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// GetStartOfMonth returns the start of month for given time
func GetStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// GetEndOfMonth returns the end of month for given time
func GetEndOfMonth(t time.Time) time.Time {
	return GetStartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// Contains checks if a slice contains a specific item
func Contains[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate items from a slice
func RemoveDuplicates[T comparable](slice []T) []T {
	keys := make(map[T]bool)
	var result []T

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// ChunkSlice splits a slice into chunks of specified size
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		return nil
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// SafeStringPointer safely converts string to *string
func SafeStringPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// SafeIntPointer safely converts int to *int
func SafeIntPointer(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// SafeFloat64Pointer safely converts float64 to *float64
func SafeFloat64Pointer(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

// DerefString safely dereferences a string pointer
func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// DerefInt safely dereferences an int pointer
func DerefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// DerefFloat64 safely dereferences a float64 pointer
func DerefFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// JSONMarshal marshals data to JSON
func JSONMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// JSONUnmarshal unmarshals JSON data
func JSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
