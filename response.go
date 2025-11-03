package nimbus

// APIError represents a custom API error with code and message
type APIError struct {
	Code    string
	Message string
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new API error
func NewAPIError(code, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code int, err string, message ...string) *ErrorResponse {
	resp := &ErrorResponse{
		Error: err,
		Code:  code,
	}
	if len(message) > 0 {
		resp.Message = message[0]
	}
	return resp
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(data any, message ...string) *SuccessResponse {
	resp := &SuccessResponse{
		Success: true,
		Data:    data,
	}
	if len(message) > 0 {
		resp.Message = message[0]
	}
	return resp
}
