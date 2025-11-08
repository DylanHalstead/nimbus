package nimbus

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   any    `json:"value"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	if len(ve) == 1 {
		return ve[0].Message
	}
	return fmt.Sprintf("validation failed on %d fields", len(ve))
}

// Schema represents a validation schema for a struct
type Schema struct {
	structType reflect.Type
	fields     map[string]fieldRule
}

type fieldRule struct {
	jsonTag   string
	required  bool
	minLength int
	maxLength int
	min       *int
	max       *int
	email     bool
	pattern   *regexp.Regexp
	enum      []string
	custom    func(any) error
}

// NewSchema creates a new validation schema from a struct type
func NewSchema(structPtr any) *Schema {
	t := reflect.TypeOf(structPtr)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		panic("NewSchema expects a struct or pointer to struct")
	}

	schema := &Schema{
		structType: t,
		fields:     make(map[string]fieldRule),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		validateTag := field.Tag.Get("validate")

		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag to get field name
		jsonName := strings.Split(jsonTag, ",")[0]

		// Parse validation rules
		rule := parseValidationTag(validateTag)
		rule.jsonTag = jsonName

		schema.fields[jsonName] = rule
	}

	return schema
}

// AddCustomValidator adds a custom validation function for a specific field (by JSON name)
func (s *Schema) AddCustomValidator(fieldName string, validator func(any) error) *Schema {
	if rule, exists := s.fields[fieldName]; exists {
		rule.custom = validator
		s.fields[fieldName] = rule
	} else {
		panic(fmt.Sprintf("field %s not found", fieldName))
	}
	return s
}

// parseValidationTag parses validation rules from struct tag
func parseValidationTag(tag string) fieldRule {
	rule := fieldRule{
		minLength: -1,
		maxLength: -1,
	}

	if tag == "" {
		return rule
	}

	rules := strings.Split(tag, ",")
	for _, r := range rules {
		r = strings.TrimSpace(r)

		switch {
		case r == "required":
			rule.required = true
		case r == "email":
			rule.email = true
		case strings.HasPrefix(r, "min="):
			if val, err := strconv.Atoi(r[4:]); err == nil {
				rule.min = &val
			}
		case strings.HasPrefix(r, "max="):
			if val, err := strconv.Atoi(r[4:]); err == nil {
				rule.max = &val
			}
		case strings.HasPrefix(r, "minlen="):
			if val, err := strconv.Atoi(r[7:]); err == nil {
				rule.minLength = val
			}
		case strings.HasPrefix(r, "maxlen="):
			if val, err := strconv.Atoi(r[7:]); err == nil {
				rule.maxLength = val
			}
		case strings.HasPrefix(r, "pattern="):
			if regex, err := regexp.Compile(r[8:]); err == nil {
				rule.pattern = regex
			}
		case strings.HasPrefix(r, "enum="):
			enumStr := r[5:]
			rule.enum = strings.Split(enumStr, "|")
		}
	}

	return rule
}

// Validate validates a struct against the schema
func (s *Schema) Validate(data any) ValidationErrors {
	var errors ValidationErrors

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ValidationErrors{{
			Field:   "root",
			Message: "expected struct type",
		}}
	}

	// Check each field in the schema
	for fieldName, rule := range s.fields {
		fieldValue := v.FieldByName(getStructFieldName(s.structType, fieldName))

		if !fieldValue.IsValid() {
			if rule.required {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Tag:     "required",
					Message: fmt.Sprintf("%s is required", fieldName),
				})
			}
			continue
		}

		// Validate the field
		if fieldErrors := s.validateField(fieldName, fieldValue.Interface(), rule); len(fieldErrors) > 0 {
			errors = append(errors, fieldErrors...)
		}
	}

	return errors
}

// validateField validates a single field against its rule
func (s *Schema) validateField(fieldName string, value any, rule fieldRule) ValidationErrors {
	var errors ValidationErrors

	// Handle nil/empty values
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.String && value.(string) == "") {
		if rule.required {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "required",
				Message: fmt.Sprintf("%s is required", fieldName),
			})
		}
		return errors
	}

	// String validations
	if str, ok := value.(string); ok {
		if rule.minLength >= 0 && len(str) < rule.minLength {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "minlen",
				Message: fmt.Sprintf("%s must be at least %d characters", fieldName, rule.minLength),
			})
		}

		if rule.maxLength >= 0 && len(str) > rule.maxLength {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "maxlen",
				Message: fmt.Sprintf("%s must be at most %d characters", fieldName, rule.maxLength),
			})
		}

		if rule.email {
			if !emailRegex.MatchString(str) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Value:   value,
					Tag:     "email",
					Message: fmt.Sprintf("%s must be a valid email", fieldName),
				})
			}
		}

		if rule.pattern != nil && !rule.pattern.MatchString(str) {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "pattern",
				Message: fmt.Sprintf("%s format is invalid", fieldName),
			})
		}

		if len(rule.enum) > 0 {
			found := false
			for _, allowed := range rule.enum {
				if str == allowed {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Value:   value,
					Tag:     "enum",
					Message: fmt.Sprintf("%s must be one of: %s", fieldName, strings.Join(rule.enum, ", ")),
				})
			}
		}
	}

	// Numeric validations
	if num, ok := convertToInt(value); ok {
		if rule.min != nil && num < *rule.min {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "min",
				Message: fmt.Sprintf("%s must be at least %d", fieldName, *rule.min),
			})
		}

		if rule.max != nil && num > *rule.max {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "max",
				Message: fmt.Sprintf("%s must be at most %d", fieldName, *rule.max),
			})
		}
	}

	// Custom validation
	if rule.custom != nil {
		if err := rule.custom(value); err != nil {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     "custom",
				Message: err.Error(),
			})
		}
	}

	return errors
}

// Helper function to get struct field name from JSON tag
func getStructFieldName(t reflect.Type, jsonName string) string {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName == jsonName {
				return field.Name
			}
		}
	}
	return ""
}

// Helper function to convert various numeric types to int
func convertToInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// ValidatedStruct is an interface that structs can implement for custom validation
type ValidatedStruct interface {
	Validate() error
}

// Validator bundles a validation schema with a factory function for creating instances.
// This provides a cleaner API by ensuring schema and factory are always paired correctly.
type Validator[T any] struct {
	Schema  *Schema
	Factory func() *T
}

// NewValidator creates a new Validator from an example struct.
// The factory will create new instances using new(T).
func NewValidator[T any](example *T) *Validator[T] {
	return &Validator[T]{
		Schema:  NewSchema(example),
		Factory: func() *T { return new(T) },
	}
}

// ValidateJSON validates JSON data against a schema and unmarshal it
func ValidateJSON(data []byte, target any, schema *Schema) error {
	// First unmarshal into a map to check for missing/extra fields
	var jsonData map[string]any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Unmarshal into the target struct
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("JSON unmarshal error: %w", err)
	}

	// Validate using schema
	if errors := schema.Validate(target); len(errors) > 0 {
		return errors
	}

	// Check if the struct implements ValidatedStruct for custom validation
	if validator, ok := target.(ValidatedStruct); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateQuery validates query parameters against a schema and binds them to a struct
func ValidateQuery(queryParams url.Values, target any, schema *Schema) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer to struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	// Bind query parameters to struct fields
	for fieldName, rule := range schema.fields {
		structFieldName := getStructFieldName(schema.structType, fieldName)
		if structFieldName == "" {
			continue
		}

		fieldValue := v.FieldByName(structFieldName)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		// Get the query parameter value (use query tag or json tag)
		structField, ok := schema.structType.FieldByName(structFieldName)
		if !ok {
			continue
		}

		queryTag := structField.Tag.Get("query")
		if queryTag == "" {
			queryTag = rule.jsonTag
		}

		paramValue := queryParams.Get(queryTag)

		// Skip if empty and not required
		if paramValue == "" {
			continue
		}

		// Convert and set the value based on field type
		if err := setFieldValue(fieldValue, paramValue); err != nil {
			return fmt.Errorf("error setting field %s: %w", fieldName, err)
		}
	}

	// Validate using schema
	if errors := schema.Validate(target); len(errors) > 0 {
		return errors
	}

	// Check if the struct implements ValidatedStruct for custom validation
	if validator, ok := target.(ValidatedStruct); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// setFieldValue sets a struct field value from a string
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// WithBodyValidation wraps a handler with automatic JSON body validation
// The validated body will be stored in the context with key ContextKeyValidatedBody
func WithBodyValidation[T any](validator *Validator[T]) func(Handler) Handler {
	return func(handler Handler) Handler {
		return func(ctx *Context) (any, int, error) {
			// Create a new instance of the body struct
			body := validator.Factory()
			if body == nil {
				return nil, 400, NewAPIError("invalid_request", "body factory returned nil")
			}

			// Validate the request body
			if err := ctx.BindAndValidateJSON(body, validator.Schema); err != nil {
				if validationErrs, ok := err.(ValidationErrors); ok {
					return ctx.SendValidationError(validationErrs)
				}
				return nil, 400, NewAPIError("invalid_request", err.Error())
			}

			// Store validated body in context
			ctx.Set(ContextKeyValidatedBody, body)

			// Call the original handler
			return handler(ctx)
		}
	}
}

// WithQueryValidation wraps a handler with automatic query parameter validation
// The validated query params will be stored in the context with key ContextKeyValidatedQuery
func WithQueryValidation[T any](validator *Validator[T]) func(Handler) Handler {
	return func(handler Handler) Handler {
		return func(ctx *Context) (any, int, error) {
			// Create a new instance of the query struct
			query := validator.Factory()

			// Validate the query parameters
			if err := ctx.BindAndValidateQuery(query, validator.Schema); err != nil {
				if validationErrs, ok := err.(ValidationErrors); ok {
					return ctx.SendValidationError(validationErrs)
				}
				return nil, 400, NewAPIError("invalid_request", err.Error())
			}

			// Store validated query in context
			ctx.Set(ContextKeyValidatedQuery, query)

			// Call the original handler
			return handler(ctx)
		}
	}
}

// populatePathParams populates a struct from path parameters using the "path" tag
func populatePathParams(pathParams map[string]string, target any) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the path tag
		pathTag := fieldType.Tag.Get("path")
		if pathTag == "" {
			continue
		}

		// Get the value from path params
		paramValue, exists := pathParams[pathTag]
		if !exists {
			return fmt.Errorf("required path parameter '%s' not found", pathTag)
		}

		// Set the field value
		if field.Kind() == reflect.String {
			field.SetString(paramValue)
		} else {
			return fmt.Errorf("path parameter '%s' has unsupported type %s (only string is supported)", pathTag, field.Kind())
		}
	}

	return nil
}

// WithPathParams wraps a handler with automatic path parameter extraction
// The params struct will be populated from path parameters and stored in context with key ContextKeyValidatedParams
// Example:
//
//	type UserParams struct {
//	    ID string `path:"id"`
//	}
//	userParamsValidator := api.NewValidator(&UserParams{})
//	WithPathParams(userParamsValidator)
func WithPathParams[T any](validator *Validator[T]) func(Handler) Handler {
	return func(handler Handler) Handler {
		return func(ctx *Context) (any, int, error) {
			// Create a new instance of the params struct
			params := validator.Factory()
			if params == nil {
				return nil, 400, NewAPIError("invalid_request", "params factory returned nil")
			}

			// Extract path parameters and populate the struct
			if err := populatePathParams(ctx.PathParams, params); err != nil {
				return nil, 400, NewAPIError("invalid_path_params", err.Error())
			}

			// Store validated params in context
			ctx.Set(ContextKeyValidatedParams, params)

			// Call the original handler
			return handler(ctx)
		}
	}
}

// ============================================================================
// Typed Handler Wrappers - Automatic Parameter Injection
// ============================================================================

// WithTyped wraps a typed handler with automatic validation and injection of parameters.
// Pass nil for any validator you don't need. Unused fields in TypedRequest will be nil.
//
// Parameters:
//   - handler: Your typed handler function
//   - params: Validator for path params (nil if not needed)
//   - body: Validator for request body (nil if not needed)
//   - query: Validator for query params (nil if not needed)
//
// Examples:
//
//	// Define validators once
//	var (
//	    userParamsValidator = api.NewValidator(&UserParams{})
//	    createUserValidator = api.NewValidator(&CreateUserRequest{})
//	    userFiltersValidator = api.NewValidator(&UserFilters{})
//	)
//
//	// Only path params
//	func getUser(ctx *api.Context, req *api.TypedRequest[UserParams, CreateUserRequest, UserFilters]) (any, int, error) {
//	    return users[req.Params.ID], 200, nil
//	}
//	router.AddRoute(http.MethodGet, "/users/:id",
//	    api.WithTyped(getUser, userParamsValidator, nil, nil))
//
//	// Only body
//	func createUser(ctx *api.Context, req *api.TypedRequest[UserParams, CreateUserRequest, UserFilters]) (any, int, error) {
//	    return createUser(req.Body), 201, nil
//	}
//	router.AddRoute(http.MethodPost, "/users",
//	    api.WithTyped(createUser, nil, createUserValidator, nil))
//
//	// Only query params
//	func listUsers(ctx *api.Context, req *api.TypedRequest[UserParams, CreateUserRequest, UserFilters]) (any, int, error) {
//	    return filterUsers(req.Query), 200, nil
//	}
//	router.AddRoute(http.MethodGet, "/users",
//	    api.WithTyped(listUsers, nil, nil, userFiltersValidator))
//
//	// All three (params + body + query)
//	func updateUser(ctx *api.Context, req *api.TypedRequest[UserParams, UpdateUserRequest, UserFilters]) (any, int, error) {
//	    return updateUser(req.Params.ID, req.Body, req.Query), 200, nil
//	}
//	router.AddRoute(http.MethodPut, "/users/:id",
//	    api.WithTyped(updateUser, userParamsValidator, updateUserValidator, userFiltersValidator))
func WithTyped[P any, B any, Q any](
	handler HandlerFuncTyped[P, B, Q],
	params *Validator[P],
	body *Validator[B],
	query *Validator[Q],
) Handler {
	return func(ctx *Context) (any, int, error) {
		var paramsPtr *P
		var bodyPtr *B
		var queryPtr *Q

		// Handle path parameters
		if params != nil {
			paramsPtr = params.Factory()
			if paramsPtr == nil {
				return nil, 400, NewAPIError("invalid_request", "params factory returned nil")
			}
			if err := populatePathParams(ctx.PathParams, paramsPtr); err != nil {
				return nil, 400, NewAPIError("invalid_path_params", err.Error())
			}
			ctx.Set(ContextKeyValidatedParams, paramsPtr)
		}

		// Handle request body
		if body != nil {
			bodyPtr = body.Factory()
			if bodyPtr == nil {
				return nil, 400, NewAPIError("invalid_request", "body factory returned nil")
			}
			if err := ctx.BindAndValidateJSON(bodyPtr, body.Schema); err != nil {
				return nil, 400, NewAPIError("invalid_request", err.Error())
			}
			ctx.Set(ContextKeyValidatedBody, bodyPtr)
		}

		// Handle query parameters
		if query != nil {
			queryPtr = query.Factory()
			if queryPtr == nil {
				return nil, 400, NewAPIError("invalid_request", "query factory returned nil")
			}
			if err := ctx.BindAndValidateQuery(queryPtr, query.Schema); err != nil {
				if validationErrs, ok := err.(ValidationErrors); ok {
					return ctx.SendValidationError(validationErrs)
				}
				return nil, 400, NewAPIError("invalid_request", err.Error())
			}
			ctx.Set(ContextKeyValidatedQuery, queryPtr)
		}

		// Build TypedRequest and call handler
		req := &TypedRequest[P, B, Q]{
			Params: paramsPtr,
			Body:   bodyPtr,
			Query:  queryPtr,
		}

		return handler(ctx, req)
	}
}
