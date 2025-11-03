package nimbus

import (
	"errors"
	"strings"
	"testing"
)

// Test structs for schema validation
type TestUser struct {
	Name     string `json:"name" validate:"required,minlen=2,maxlen=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=18,max=120"`
	Role     string `json:"role" validate:"enum=user|admin|moderator"`
	Password string `json:"password" validate:"required,minlen=8"`
}

func (u *TestUser) Validate() error {
	if u.Role == "admin" && u.Age < 21 {
		return errors.New("admin must be at least 21 years old")
	}
	return nil
}

type TestProduct struct {
	Name        string  `json:"name" validate:"required,maxlen=100"`
	Price       float64 `json:"price" validate:"min=0"`
	Category    string  `json:"category" validate:"required"`
	Description string  `json:"description" validate:"maxlen=500"`
}

type TestContact struct {
	Name       string `json:"name" validate:"required,minlen=2,maxlen=50"`
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone" validate:"pattern=^\\d{3}-\\d{3}-\\d{4}$"`
	Website    string `json:"website" validate:"pattern=^https?://[a-zA-Z0-9.-]+\\.[a-zA-Z]{2}[a-zA-Z]*(/.*)?$"`
	PostalCode string `json:"postal_code" validate:"pattern=^\\d{5}(-\\d{4})?$"`
	HexColor   string `json:"hex_color" validate:"pattern=^#[0-9A-Fa-f]{6}$"`
}

func TestNewSchema(t *testing.T) {
	schema := NewSchema(TestUser{})

	if schema == nil {
		t.Fatal("Expected schema to be created")
	}

	if len(schema.fields) != 5 {
		t.Errorf("Expected 5 fields in schema, got %d", len(schema.fields))
	}

	// Test that fields are properly parsed
	nameField, exists := schema.fields["name"]
	if !exists {
		t.Error("Expected 'name' field to exist in schema")
	}

	if !nameField.required {
		t.Error("Expected 'name' field to be required")
	}

	if nameField.minLength != 2 {
		t.Errorf("Expected 'name' minLength to be 2, got %d", nameField.minLength)
	}

	if nameField.maxLength != 50 {
		t.Errorf("Expected 'name' maxLength to be 50, got %d", nameField.maxLength)
	}
}

func TestSchema_Validate_Success(t *testing.T) {
	schema := NewSchema(TestUser{})

	user := TestUser{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Role:     "user",
		Password: "password123",
	}

	errors := schema.Validate(user)
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}
}

func TestSchema_Validate_Required(t *testing.T) {
	schema := NewSchema(TestUser{})

	user := TestUser{
		Name:  "", // Required field missing
		Email: "john@example.com",
		Age:   25,
		Role:  "user",
	}

	errors := schema.Validate(user)
	if len(errors) != 2 { // name and password required
		t.Errorf("Expected 2 validation errors, got %d: %v", len(errors), errors)
	}

	// Check specific error
	found := false
	for _, err := range errors {
		if err.Field == "name" && err.Tag == "required" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected required validation error for 'name' field")
	}
}

func TestSchema_Validate_Email(t *testing.T) {
	schema := NewSchema(TestUser{})

	user := TestUser{
		Name:     "John Doe",
		Email:    "invalid-email",
		Age:      25,
		Role:     "user",
		Password: "password123",
	}

	errors := schema.Validate(user)
	if len(errors) != 1 {
		t.Errorf("Expected 1 validation error, got %d: %v", len(errors), errors)
	}

	if errors[0].Field != "email" || errors[0].Tag != "email" {
		t.Errorf("Expected email validation error, got: %v", errors[0])
	}
}

func TestSchema_Validate_MinMax(t *testing.T) {
	schema := NewSchema(TestUser{})

	user := TestUser{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      15, // Below minimum
		Role:     "user",
		Password: "password123",
	}

	errors := schema.Validate(user)
	if len(errors) != 1 {
		t.Errorf("Expected 1 validation error, got %d: %v", len(errors), errors)
	}

	if errors[0].Field != "age" || errors[0].Tag != "min" {
		t.Errorf("Expected min validation error for age, got: %v", errors[0])
	}
}

func TestSchema_Validate_Enum(t *testing.T) {
	schema := NewSchema(TestUser{})

	user := TestUser{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Role:     "invalid_role",
		Password: "password123",
	}

	errors := schema.Validate(user)
	if len(errors) != 1 {
		t.Errorf("Expected 1 validation error, got %d: %v", len(errors), errors)
	}

	if errors[0].Field != "role" || errors[0].Tag != "enum" {
		t.Errorf("Expected enum validation error for role, got: %v", errors[0])
	}
}

func TestSchema_Validate_StringLength(t *testing.T) {
	schema := NewSchema(TestUser{})

	user := TestUser{
		Name:     "J", // Too short
		Email:    "john@example.com",
		Age:      25,
		Role:     "user",
		Password: "short", // Too short
	}

	errors := schema.Validate(user)
	if len(errors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d: %v", len(errors), errors)
	}

	// Check for minlen errors
	nameError := false
	passwordError := false
	for _, err := range errors {
		if err.Field == "name" && err.Tag == "minlen" {
			nameError = true
		}
		if err.Field == "password" && err.Tag == "minlen" {
			passwordError = true
		}
	}

	if !nameError {
		t.Error("Expected minlen validation error for name")
	}
	if !passwordError {
		t.Error("Expected minlen validation error for password")
	}
}

func TestValidateJSON_Success(t *testing.T) {
	schema := NewSchema(TestUser{})

	jsonData := `{
		"name": "John Doe",
		"email": "john@example.com",
		"age": 25,
		"role": "user",
		"password": "password123"
	}`

	var user TestUser
	err := ValidateJSON([]byte(jsonData), &user, schema)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if user.Name != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got '%s'", user.Name)
	}
}

func TestValidateJSON_ValidationError(t *testing.T) {
	schema := NewSchema(TestUser{})

	jsonData := `{
		"name": "",
		"email": "invalid-email",
		"age": 15,
		"role": "invalid_role"
	}`

	var user TestUser
	err := ValidateJSON([]byte(jsonData), &user, schema)

	if err == nil {
		t.Error("Expected validation error")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	if len(validationErrors) < 4 {
		t.Errorf("Expected at least 4 validation errors, got %d", len(validationErrors))
	}
}

func TestValidateJSON_CustomValidation(t *testing.T) {
	schema := NewSchema(TestUser{})

	jsonData := `{
		"name": "Admin User",
		"email": "admin@example.com",
		"age": 20,
		"role": "admin",
		"password": "password123"
	}`

	var user TestUser
	err := ValidateJSON([]byte(jsonData), &user, schema)

	if err == nil {
		t.Error("Expected custom validation error")
	}

	expectedMsg := "admin must be at least 21 years old"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidateJSON_InvalidJSON(t *testing.T) {
	schema := NewSchema(TestUser{})

	jsonData := `{invalid json}`

	var user TestUser
	err := ValidateJSON([]byte(jsonData), &user, schema)

	if err == nil {
		t.Error("Expected JSON parsing error")
	}
}

func TestSchema_Validate_Pattern_Success(t *testing.T) {
	schema := NewSchema(TestContact{})

	contact := TestContact{
		Name:       "John Doe",
		Email:      "john@example.com",
		Phone:      "123-456-7890",
		Website:    "https://example.com",
		PostalCode: "12345",
		HexColor:   "#FF5733",
	}

	errors := schema.Validate(contact)
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}
}

func TestSchema_Validate_Pattern_Failures(t *testing.T) {
	schema := NewSchema(TestContact{})

	tests := []struct {
		name     string
		contact  TestContact
		expected []string // Expected failing fields
	}{
		{
			name: "Invalid phone pattern",
			contact: TestContact{
				Name:       "John Doe",
				Email:      "john@example.com",
				Phone:      "123-45-7890", // Wrong format
				Website:    "https://example.com",
				PostalCode: "12345",
				HexColor:   "#FF5733",
			},
			expected: []string{"phone"},
		},
		{
			name: "Invalid website pattern",
			contact: TestContact{
				Name:       "John Doe",
				Email:      "john@example.com",
				Phone:      "123-456-7890",
				Website:    "ftp://example.com", // Wrong protocol
				PostalCode: "12345",
				HexColor:   "#FF5733",
			},
			expected: []string{"website"},
		},
		{
			name: "Invalid postal code pattern",
			contact: TestContact{
				Name:       "John Doe",
				Email:      "john@example.com",
				Phone:      "123-456-7890",
				Website:    "https://example.com",
				PostalCode: "1234", // Too short
				HexColor:   "#FF5733",
			},
			expected: []string{"postal_code"},
		},
		{
			name: "Invalid hex color pattern",
			contact: TestContact{
				Name:       "John Doe",
				Email:      "john@example.com",
				Phone:      "123-456-7890",
				Website:    "https://example.com",
				PostalCode: "12345",
				HexColor:   "#GG5733", // Invalid hex characters
			},
			expected: []string{"hex_color"},
		},
		{
			name: "Multiple pattern failures",
			contact: TestContact{
				Name:       "John Doe",
				Email:      "john@example.com",
				Phone:      "invalid-phone",
				Website:    "not-a-url",
				PostalCode: "abc",
				HexColor:   "not-hex",
			},
			expected: []string{"phone", "website", "postal_code", "hex_color"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := schema.Validate(tt.contact)

			if len(errors) != len(tt.expected) {
				t.Errorf("Expected %d validation errors, got %d: %v",
					len(tt.expected), len(errors), errors)
			}

			// Check that all expected fields have pattern errors
			errorFields := make(map[string]bool)
			for _, err := range errors {
				if err.Tag == "pattern" {
					errorFields[err.Field] = true
				}
			}

			for _, expectedField := range tt.expected {
				if !errorFields[expectedField] {
					t.Errorf("Expected pattern validation error for field '%s'", expectedField)
				}
			}
		})
	}
}

func TestSchema_Validate_Pattern_OptionalFields(t *testing.T) {
	// Test that empty optional pattern fields don't trigger validation
	schema := NewSchema(TestContact{})

	contact := TestContact{
		Name:  "John Doe",
		Email: "john@example.com",
		// All pattern fields left empty (they're not required)
	}

	errors := schema.Validate(contact)

	// Should only have errors for required fields (name, email), not pattern fields
	patternErrors := 0
	for _, err := range errors {
		if err.Tag == "pattern" {
			patternErrors++
		}
	}

	if patternErrors > 0 {
		t.Errorf("Expected no pattern validation errors for empty optional fields, got %d", patternErrors)
	}
}

func TestValidateJSON_Pattern_Success(t *testing.T) {
	schema := NewSchema(TestContact{})

	jsonData := `{
		"name": "John Doe",
		"email": "john@example.com",
		"phone": "123-456-7890",
		"website": "https://example.com/path",
		"postal_code": "12345-6789",
		"hex_color": "#FF5733"
	}`

	var contact TestContact
	err := ValidateJSON([]byte(jsonData), &contact, schema)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if contact.Phone != "123-456-7890" {
		t.Errorf("Expected phone to be '123-456-7890', got '%s'", contact.Phone)
	}
}

func TestValidateJSON_Pattern_Failure(t *testing.T) {
	schema := NewSchema(TestContact{})

	jsonData := `{
		"name": "John Doe",
		"email": "john@example.com",
		"phone": "invalid-phone",
		"website": "not-a-url",
		"postal_code": "abc",
		"hex_color": "not-hex"
	}`

	var contact TestContact
	err := ValidateJSON([]byte(jsonData), &contact, schema)

	if err == nil {
		t.Error("Expected validation error")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	// Should have 4 pattern validation errors
	patternErrors := 0
	for _, verr := range validationErrors {
		if verr.Tag == "pattern" {
			patternErrors++
		}
	}

	if patternErrors != 4 {
		t.Errorf("Expected 4 pattern validation errors, got %d", patternErrors)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errors := ValidationErrors{
		{Field: "name", Message: "name is required"},
		{Field: "email", Message: "email is invalid"},
	}

	errorMsg := errors.Error()
	expected := "validation failed on 2 fields"
	if errorMsg != expected {
		t.Errorf("Expected '%s', got '%s'", expected, errorMsg)
	}

	// Test single error
	singleError := ValidationErrors{
		{Field: "name", Message: "name is required"},
	}

	errorMsg = singleError.Error()
	expected = "name is required"
	if errorMsg != expected {
		t.Errorf("Expected '%s', got '%s'", expected, errorMsg)
	}

	// Test empty errors
	emptyErrors := ValidationErrors{}
	if emptyErrors.Error() != "" {
		t.Errorf("Expected empty string for no errors, got '%s'", emptyErrors.Error())
	}
}

// Test structs for query parameter validation
type TestSearchQuery struct {
	Query    string `json:"query" validate:"required,minlen=2,maxlen=100"`
	Category string `json:"category" validate:"enum=electronics|clothing|books"`
	MinPrice int    `json:"min_price" validate:"min=0"`
	MaxPrice int    `json:"max_price" validate:"min=0,max=100000"`
	Page     int    `json:"page" validate:"min=1"`
	Limit    int    `json:"limit" validate:"min=1,max=100"`
}

func TestValidateQuery_Success(t *testing.T) {
	schema := NewSchema(TestSearchQuery{})

	// Create URL query values
	queryParams := map[string][]string{
		"query":     {"laptop"},
		"category":  {"electronics"},
		"min_price": {"100"},
		"max_price": {"2000"},
		"page":      {"1"},
		"limit":     {"10"},
	}

	var query TestSearchQuery
	err := ValidateQuery(queryParams, &query, schema)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if query.Query != "laptop" {
		t.Errorf("Expected query to be 'laptop', got '%s'", query.Query)
	}

	if query.Category != "electronics" {
		t.Errorf("Expected category to be 'electronics', got '%s'", query.Category)
	}

	if query.MinPrice != 100 {
		t.Errorf("Expected min_price to be 100, got %d", query.MinPrice)
	}

	if query.MaxPrice != 2000 {
		t.Errorf("Expected max_price to be 2000, got %d", query.MaxPrice)
	}

	if query.Page != 1 {
		t.Errorf("Expected page to be 1, got %d", query.Page)
	}

	if query.Limit != 10 {
		t.Errorf("Expected limit to be 10, got %d", query.Limit)
	}
}

func TestValidateQuery_RequiredValidation(t *testing.T) {
	schema := NewSchema(TestSearchQuery{})

	// Missing required field 'query'
	queryParams := map[string][]string{
		"category": {"electronics"},
	}

	var query TestSearchQuery
	err := ValidateQuery(queryParams, &query, schema)

	if err == nil {
		t.Error("Expected validation error for missing required field")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	foundRequiredError := false
	for _, verr := range validationErrors {
		if verr.Field == "query" && verr.Tag == "required" {
			foundRequiredError = true
			break
		}
	}

	if !foundRequiredError {
		t.Error("Expected required validation error for 'query' field")
	}
}

func TestValidateQuery_EnumValidation(t *testing.T) {
	schema := NewSchema(TestSearchQuery{})

	queryParams := map[string][]string{
		"query":    {"laptop"},
		"category": {"invalid-category"},
	}

	var query TestSearchQuery
	err := ValidateQuery(queryParams, &query, schema)

	if err == nil {
		t.Error("Expected validation error for invalid enum value")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	foundEnumError := false
	for _, verr := range validationErrors {
		if verr.Field == "category" && verr.Tag == "enum" {
			foundEnumError = true
			break
		}
	}

	if !foundEnumError {
		t.Error("Expected enum validation error for 'category' field")
	}
}

func TestValidateQuery_MinMaxValidation(t *testing.T) {
	schema := NewSchema(TestSearchQuery{})

	queryParams := map[string][]string{
		"query":     {"laptop"},
		"min_price": {"-10"},    // Invalid: below min
		"max_price": {"200000"}, // Invalid: above max
		"page":      {"0"},      // Invalid: below min
		"limit":     {"200"},    // Invalid: above max
	}

	var query TestSearchQuery
	err := ValidateQuery(queryParams, &query, schema)

	if err == nil {
		t.Error("Expected validation errors")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	// Should have multiple min/max validation errors
	minMaxErrors := 0
	for _, verr := range validationErrors {
		if verr.Tag == "min" || verr.Tag == "max" {
			minMaxErrors++
		}
	}

	if minMaxErrors < 2 {
		t.Errorf("Expected at least 2 min/max validation errors, got %d", minMaxErrors)
	}
}

func TestValidateQuery_StringLengthValidation(t *testing.T) {
	schema := NewSchema(TestSearchQuery{})

	queryParams := map[string][]string{
		"query": {"a"}, // Too short (minlen=2)
	}

	var query TestSearchQuery
	err := ValidateQuery(queryParams, &query, schema)

	if err == nil {
		t.Error("Expected validation error for string length")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	foundMinlenError := false
	for _, verr := range validationErrors {
		if verr.Field == "query" && verr.Tag == "minlen" {
			foundMinlenError = true
			break
		}
	}

	if !foundMinlenError {
		t.Error("Expected minlen validation error for 'query' field")
	}
}

// Test struct for custom validator tests
type TestCustomUser struct {
	Username string `json:"username" validate:"required,minlen=3,maxlen=20"`
	Password string `json:"password" validate:"required,minlen=8"`
	Age      int    `json:"age" validate:"min=0,max=150"`
}

func TestAddCustomValidator_Success(t *testing.T) {
	schema := NewSchema(TestCustomUser{})

	// Add custom validator for username - no spaces allowed
	schema.AddCustomValidator("username", func(value any) error {
		username, ok := value.(string)
		if !ok {
			return errors.New("username must be a string")
		}
		if strings.Contains(username, " ") {
			return errors.New("username cannot contain spaces")
		}
		return nil
	})

	// Test valid username
	user := TestCustomUser{
		Username: "validuser",
		Password: "password123",
		Age:      25,
	}

	validationErrors := schema.Validate(user)
	if len(validationErrors) != 0 {
		t.Errorf("Expected no validation errors, got: %v", validationErrors)
	}
}

func TestAddCustomValidator_Failure(t *testing.T) {
	schema := NewSchema(TestCustomUser{})

	// Add custom validator for username - no spaces allowed
	schema.AddCustomValidator("username", func(value any) error {
		username, ok := value.(string)
		if !ok {
			return errors.New("username must be a string")
		}
		if strings.Contains(username, " ") {
			return errors.New("username cannot contain spaces")
		}
		return nil
	})

	// Test invalid username with space
	user := TestCustomUser{
		Username: "invalid user",
		Password: "password123",
		Age:      25,
	}

	validationErrors := schema.Validate(user)
	if len(validationErrors) != 1 {
		t.Errorf("Expected 1 validation error, got %d: %v", len(validationErrors), validationErrors)
	}

	if validationErrors[0].Field != "username" || validationErrors[0].Tag != "custom" {
		t.Errorf("Expected custom validation error for username, got: %v", validationErrors[0])
	}

	expectedMsg := "username cannot contain spaces"
	if validationErrors[0].Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, validationErrors[0].Message)
	}
}

func TestAddCustomValidator_MultipleFields(t *testing.T) {
	schema := NewSchema(TestCustomUser{})

	// Add custom validator for username
	schema.AddCustomValidator("username", func(value any) error {
		username, ok := value.(string)
		if !ok {
			return errors.New("username must be a string")
		}
		if strings.HasPrefix(username, "admin") {
			return errors.New("username cannot start with 'admin'")
		}
		return nil
	})

	// Add custom validator for password
	schema.AddCustomValidator("password", func(value any) error {
		password, ok := value.(string)
		if !ok {
			return errors.New("password must be a string")
		}
		// Check for at least one digit
		hasDigit := false
		for _, ch := range password {
			if ch >= '0' && ch <= '9' {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			return errors.New("password must contain at least one digit")
		}
		return nil
	})

	// Test both validators fail
	user := TestCustomUser{
		Username: "adminuser",
		Password: "passwordonly",
		Age:      25,
	}

	validationErrors := schema.Validate(user)
	if len(validationErrors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d: %v", len(validationErrors), validationErrors)
	}

	// Check both custom validators triggered
	foundUsernameError := false
	foundPasswordError := false
	for _, err := range validationErrors {
		if err.Field == "username" && err.Tag == "custom" {
			foundUsernameError = true
		}
		if err.Field == "password" && err.Tag == "custom" {
			foundPasswordError = true
		}
	}

	if !foundUsernameError {
		t.Error("Expected custom validation error for username")
	}
	if !foundPasswordError {
		t.Error("Expected custom validation error for password")
	}
}

func TestAddCustomValidator_Chaining(t *testing.T) {
	schema := NewSchema(TestCustomUser{}).
		AddCustomValidator("username", func(value any) error {
			username, ok := value.(string)
			if !ok {
				return errors.New("username must be a string")
			}
			if strings.Contains(username, "_") {
				return errors.New("username cannot contain underscores")
			}
			return nil
		}).
		AddCustomValidator("age", func(value any) error {
			age, ok := convertToInt(value)
			if !ok {
				return errors.New("age must be a number")
			}
			if age < 18 {
				return errors.New("user must be 18 or older")
			}
			return nil
		})

	// Test with invalid data
	user := TestCustomUser{
		Username: "user_name",
		Password: "password123",
		Age:      16,
	}

	validationErrors := schema.Validate(user)
	if len(validationErrors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d: %v", len(validationErrors), validationErrors)
	}
}

func TestAddCustomValidator_NonExistentField_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when adding validator to non-existent field")
		}
	}()

	schema := NewSchema(TestCustomUser{})
	schema.AddCustomValidator("nonexistent", func(value any) error {
		return nil
	})
}

func TestAddCustomValidator_WithJSON(t *testing.T) {
	schema := NewSchema(TestCustomUser{})

	// Add custom validator for password strength
	schema.AddCustomValidator("password", func(value any) error {
		password, ok := value.(string)
		if !ok {
			return errors.New("password must be a string")
		}
		hasUpper := false
		hasLower := false
		hasDigit := false
		for _, ch := range password {
			if ch >= 'A' && ch <= 'Z' {
				hasUpper = true
			}
			if ch >= 'a' && ch <= 'z' {
				hasLower = true
			}
			if ch >= '0' && ch <= '9' {
				hasDigit = true
			}
		}
		if !hasUpper || !hasLower || !hasDigit {
			return errors.New("password must contain uppercase, lowercase, and digit")
		}
		return nil
	})

	// Test with weak password
	jsonData := `{
		"username": "testuser",
		"password": "weakpass",
		"age": 25
	}`

	var user TestCustomUser
	err := ValidateJSON([]byte(jsonData), &user, schema)

	if err == nil {
		t.Error("Expected validation error for weak password")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("Expected ValidationErrors, got %T", err)
	}

	foundPasswordError := false
	for _, verr := range validationErrors {
		if verr.Field == "password" && verr.Tag == "custom" {
			foundPasswordError = true
			break
		}
	}

	if !foundPasswordError {
		t.Error("Expected custom validation error for password")
	}

	// Test with strong password
	jsonData = `{
		"username": "testuser",
		"password": "StrongPass123",
		"age": 25
	}`

	var user2 TestCustomUser
	err = ValidateJSON([]byte(jsonData), &user2, schema)

	if err != nil {
		t.Errorf("Expected no error with strong password, got: %v", err)
	}
}

func TestAddCustomValidator_CombinedWithBuiltIn(t *testing.T) {
	schema := NewSchema(TestCustomUser{})

	// Add custom validator that works alongside built-in validators
	schema.AddCustomValidator("username", func(value any) error {
		username, ok := value.(string)
		if !ok {
			return errors.New("username must be a string")
		}
		// Custom rule: must be alphanumeric only
		for _, ch := range username {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return errors.New("username must be alphanumeric only")
			}
		}
		return nil
	})

	// Test username that's too short (built-in) and has special chars (custom)
	user := TestCustomUser{
		Username: "a!",
		Password: "password123",
		Age:      25,
	}

	validationErrors := schema.Validate(user)

	// Should have both minlen (built-in) and custom validation errors
	if len(validationErrors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d: %v", len(validationErrors), validationErrors)
	}

	foundMinlenError := false
	foundCustomError := false
	for _, err := range validationErrors {
		if err.Field == "username" && err.Tag == "minlen" {
			foundMinlenError = true
		}
		if err.Field == "username" && err.Tag == "custom" {
			foundCustomError = true
		}
	}

	if !foundMinlenError {
		t.Error("Expected minlen validation error for username")
	}
	if !foundCustomError {
		t.Error("Expected custom validation error for username")
	}
}
