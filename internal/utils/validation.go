package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, ", ")
}

// ValidateStruct validates a struct using reflection and struct tags
func ValidateStruct(s interface{}) error {
	var errors ValidationErrors

	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", v.Kind())
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get validation tag
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// Parse validation rules
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if err := validateField(fieldType.Name, field, rule); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateField validates a single field against a rule
func validateField(fieldName string, field reflect.Value, rule string) *ValidationError {
	parts := strings.Split(rule, "=")
	ruleName := parts[0]
	var ruleValue string
	if len(parts) > 1 {
		ruleValue = parts[1]
	}

	switch ruleName {
	case "required":
		if isEmpty(field) {
			return &ValidationError{
				Field:   fieldName,
				Message: "is required",
			}
		}
	case "email":
		if field.Kind() == reflect.String {
			email := field.String()
			if email != "" && !IsValidEmail(email) {
				return &ValidationError{
					Field:   fieldName,
					Message: "must be a valid email address",
				}
			}
		}
	case "phone":
		if field.Kind() == reflect.String {
			phone := field.String()
			if phone != "" && !IsPhoneNumber(phone) {
				return &ValidationError{
					Field:   fieldName,
					Message: "must be a valid phone number",
				}
			}
		}
	case "min":
		if field.Kind() == reflect.String {
			if len(field.String()) < parseIntOrDefault(ruleValue, 0) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be at least %s characters", ruleValue),
				}
			}
		} else if isNumeric(field) {
			if getNumericValue(field) < float64(parseIntOrDefault(ruleValue, 0)) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be at least %s", ruleValue),
				}
			}
		}
	case "max":
		if field.Kind() == reflect.String {
			if len(field.String()) > parseIntOrDefault(ruleValue, 0) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be at most %s characters", ruleValue),
				}
			}
		} else if isNumeric(field) {
			if getNumericValue(field) > float64(parseIntOrDefault(ruleValue, 0)) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be at most %s", ruleValue),
				}
			}
		}
	case "gt":
		if isNumeric(field) {
			if getNumericValue(field) <= float64(parseIntOrDefault(ruleValue, 0)) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be greater than %s", ruleValue),
				}
			}
		}
	case "gte":
		if isNumeric(field) {
			if getNumericValue(field) < float64(parseIntOrDefault(ruleValue, 0)) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be greater than or equal to %s", ruleValue),
				}
			}
		}
	case "lt":
		if isNumeric(field) {
			if getNumericValue(field) >= float64(parseIntOrDefault(ruleValue, 0)) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be less than %s", ruleValue),
				}
			}
		}
	case "lte":
		if isNumeric(field) {
			if getNumericValue(field) > float64(parseIntOrDefault(ruleValue, 0)) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be less than or equal to %s", ruleValue),
				}
			}
		}
	case "alphanumeric":
		if field.Kind() == reflect.String {
			str := field.String()
			if str != "" && !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "must contain only letters and numbers",
				}
			}
		}
	case "alpha":
		if field.Kind() == reflect.String {
			str := field.String()
			if str != "" && !regexp.MustCompile(`^[a-zA-Z\s]+$`).MatchString(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "must contain only letters and spaces",
				}
			}
		}
	case "numeric":
		if field.Kind() == reflect.String {
			str := field.String()
			if str != "" && !regexp.MustCompile(`^[0-9]+$`).MatchString(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "must contain only numbers",
				}
			}
		}
	case "no_sql_injection":
		if field.Kind() == reflect.String {
			str := strings.ToLower(field.String())
			dangerous := []string{
				"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "exec", "execute",
				"select", "insert", "update", "delete", "drop", "create", "alter",
				"union", "script", "javascript", "vbscript", "onload", "onerror",
				"<script", "</script", "eval(", "expression(", "url(", "import(",
				"truncate", "grant", "revoke", "declare", "cast", "convert",
			}
			for _, pattern := range dangerous {
				if strings.Contains(str, pattern) {
					return &ValidationError{
						Field:   fieldName,
						Message: "contains potentially dangerous characters",
					}
				}
			}
		}
	case "no_xss":
		if field.Kind() == reflect.String {
			str := strings.ToLower(field.String())
			xssPatterns := []string{
				"<script", "</script", "javascript:", "vbscript:", "onload=", "onerror=",
				"onclick=", "onmouseover=", "onfocus=", "onblur=", "onchange=", "onsubmit=",
				"<iframe", "<object", "<embed", "<link", "<meta", "data:text/html",
				"eval(", "expression(", "url(javascript:", "&#", "&#x", "<svg",
				"<img", "onerror", "onmouseover", "onfocus", "onblur", "onchange",
			}
			for _, pattern := range xssPatterns {
				if strings.Contains(str, pattern) {
					return &ValidationError{
						Field:   fieldName,
						Message: "contains potentially malicious content",
					}
				}
			}
		}
	case "amount":
		if isNumeric(field) {
			value := getNumericValue(field)
			if value < 0 || value > 100000000 { // Max 100M KES
				return &ValidationError{
					Field:   fieldName,
					Message: "amount must be between 0 and 100,000,000",
				}
			}
		}
	case "safe_text":
		if field.Kind() == reflect.String {
			str := field.String()
			// Allow letters, numbers, spaces, and safe punctuation
			if str != "" && !regexp.MustCompile(`^[a-zA-Z0-9\s\.\,\!\?\-\_\(\)]+$`).MatchString(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "contains invalid characters",
				}
			}
		}
	case "url_safe":
		if field.Kind() == reflect.String {
			str := field.String()
			if str != "" && !regexp.MustCompile(`^[a-zA-Z0-9\-\_\.\/\:]+$`).MatchString(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "must be URL safe",
				}
			}
		}
	}

	return nil
}

// isEmpty checks if a field is empty
func isEmpty(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return field.IsNil()
	case reflect.Invalid:
		return true
	default:
		return false
	}
}

// isNumeric checks if a field is numeric
func isNumeric(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// getNumericValue gets the numeric value as float64
func getNumericValue(field reflect.Value) float64 {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(field.Uint())
	case reflect.Float32, reflect.Float64:
		return field.Float()
	default:
		return 0
	}
}

// parseIntOrDefault parses an integer or returns default value
func parseIntOrDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}

	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// NormalizeEmail normalizes an email address for consistent comparison
func NormalizeEmail(email string) string {
	// Trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(email))
	return normalized
}

// IsPhoneNumber checks if a string looks like a phone number
func IsPhoneNumber(phone string) bool {
	// Remove spaces and common separators
	cleaned := regexp.MustCompile(`[\s\-\(\)]+`).ReplaceAllString(phone, "")

	// Check for Kenyan phone number patterns
	kenyanPhoneRegex := regexp.MustCompile(`^(\+254|254|0)[17]\d{8}$`)
	return kenyanPhoneRegex.MatchString(cleaned)
}

// ValidateKenyanPhone validates Kenyan phone number format
func ValidateKenyanPhone(phone string) bool {
	// Remove spaces and common separators
	cleaned := regexp.MustCompile(`[\s\-\(\)]+`).ReplaceAllString(phone, "")

	// Kenyan phone number patterns:
	// +254712345678, 254712345678, 0712345678
	// Networks: 7xx (Safaricom), 1xx (Airtel)
	kenyanPhoneRegex := regexp.MustCompile(`^(\+254|254|0)[17]\d{8}$`)
	return kenyanPhoneRegex.MatchString(cleaned)
}

// ValidatePassword validates password strength
func ValidatePassword(password string) []string {
	var errors []string

	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long")
	}

	if len(password) > 128 {
		errors = append(errors, "Password must be at most 128 characters long")
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	if !hasUpper {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}

	if !hasLower {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}

	if !hasNumber {
		errors = append(errors, "Password must contain at least one number")
	}

	if !hasSpecial {
		errors = append(errors, "Password must contain at least one special character")
	}

	return errors
}

// SanitizeString removes potentially harmful characters from strings
func SanitizeString(input string) string {
	// Remove null bytes and control characters
	sanitized := regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(input, "")

	// Remove potential XSS patterns
	sanitized = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(sanitized, "")

	// Remove SQL injection patterns
	dangerous := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "exec", "execute",
		"<script", "</script", "javascript:", "vbscript:", "onload=", "onerror=",
		"eval(", "expression(", "url(", "import(",
	}

	for _, pattern := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, pattern, "")
		sanitized = strings.ReplaceAll(sanitized, strings.ToUpper(pattern), "")
	}

	// Trim whitespace
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}

// SanitizeHTML removes HTML tags and dangerous content
func SanitizeHTML(input string) string {
	// Remove all HTML tags
	sanitized := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(input, "")

	// Decode HTML entities
	sanitized = strings.ReplaceAll(sanitized, "&lt;", "<")
	sanitized = strings.ReplaceAll(sanitized, "&gt;", ">")
	sanitized = strings.ReplaceAll(sanitized, "&amp;", "&")
	sanitized = strings.ReplaceAll(sanitized, "&quot;", "\"")
	sanitized = strings.ReplaceAll(sanitized, "&#x27;", "'")

	// Apply regular sanitization
	return SanitizeString(sanitized)
}

// ValidateAndSanitizeInput validates and sanitizes input with specific rules
func ValidateAndSanitizeInput(input string, maxLength int, allowSpecialChars bool) (string, error) {
	if len(input) > maxLength {
		return "", fmt.Errorf("input too long, maximum %d characters allowed", maxLength)
	}

	// Sanitize first
	sanitized := SanitizeString(input)

	// Validate based on rules
	if !allowSpecialChars {
		if !regexp.MustCompile(`^[a-zA-Z0-9\s\.\,\!\?\-\_\(\)]*$`).MatchString(sanitized) {
			return "", fmt.Errorf("input contains invalid characters")
		}
	}

	return sanitized, nil
}

// ValidateAmount validates monetary amounts
func ValidateAmount(amount float64, minAmount, maxAmount float64) error {
	if amount < minAmount {
		return fmt.Errorf("amount must be at least %.2f", minAmount)
	}
	if amount > maxAmount {
		return fmt.Errorf("amount must not exceed %.2f", maxAmount)
	}
	if amount != float64(int64(amount*100))/100 {
		return fmt.Errorf("amount can only have up to 2 decimal places")
	}
	return nil
}

// ValidateUUID validates UUID format
func ValidateUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(strings.ToLower(uuid))
}
