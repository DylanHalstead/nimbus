package middleware

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"time"

	"github.com/DylanHalstead/nimbus"
	"github.com/oklog/ulid/v2"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for storing request ID
	RequestIDKey = "request_id"
)

var (
	// entropy is a monotonic entropy source for ULID generation
	// This ensures ULIDs are sortable even when generated in the same millisecond
	entropy = ulid.Monotonic(rand.Reader, 0)
)

// RequestIDConfig defines configuration for the RequestID middleware
type RequestIDConfig struct {
	// HeaderName is the name of the request ID header (default: X-Request-ID)
	HeaderName string
	// Generator is a function to generate new request IDs
	Generator func() string
	// ContextKey is the key used to store the request ID in context
	ContextKey string
}

// DefaultRequestIDConfig returns a default RequestID configuration
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		HeaderName: RequestIDHeader,
		Generator:  generateRequestID,
		ContextKey: RequestIDKey,
	}
}

// RequestID is a middleware that generates or propagates request IDs
// It checks for an existing X-Request-ID header and generates one if not present
// The request ID is stored in the context and added to the response headers
func RequestID(configs ...RequestIDConfig) nimbus.Middleware {
	config := DefaultRequestIDConfig()
	if len(configs) > 0 {
		config = configs[0]
	}

	// Use defaults if not specified
	if config.HeaderName == "" {
		config.HeaderName = RequestIDHeader
	}
	if config.Generator == nil {
		config.Generator = generateRequestID
	}
	if config.ContextKey == "" {
		config.ContextKey = RequestIDKey
	}

	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			// Check if request ID exists in incoming request
			requestID := ctx.GetHeader(config.HeaderName)

			// Generate new ID if not present
			if requestID == "" {
				requestID = config.Generator()
			}

			// Store request ID in context for easy access
			ctx.Set(config.ContextKey, requestID)

			// Add request ID to response headers for tracing
			ctx.Header(config.HeaderName, requestID)

			// Call next handler
			return next(ctx)
		}
	}
}

// generateRequestID generates a UUID v4 (random UUID)
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (36 characters)
// Uses crypto/rand for better randomness and uniqueness
func generateRequestID() string {
	// Generate 16 random bytes (128 bits)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simpler method if crypto/rand fails
		return fmt.Sprintf("req_%d", mathrand.Int())
	}

	// Set version (4) and variant bits according to RFC 4122
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10

	// Format as UUID: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// GenerateULID generates a ULID (Universally Unique Lexicographically Sortable Identifier)
// ULIDs are:
// - 128-bit compatible with UUID
// - 1.21e+24 unique ULIDs per millisecond
// - Lexicographically sortable
// - Canonically encoded as a 26 character string (vs 36 for UUID)
// - URL safe (base32 encoded)
// - Case insensitive
// - No special characters (URL safe)
// - Monotonic sort order (correctly detects and handles the same millisecond)
func GenerateULID() string {
	// Use the shared monotonic entropy source
	// This ensures proper ordering even when ULIDs are generated in the same millisecond
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return id.String()
}
