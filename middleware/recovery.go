package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/DylanHalstead/nimbus"
)

// Recovery is a middleware that recovers from panics
func Recovery() nimbus.MiddlewareFunc {
	return func(next nimbus.HandlerFunc) nimbus.HandlerFunc {
		return func(ctx *nimbus.Context) (data any, statusCode int, err error) {
			defer func() {
				if r := recover(); r != nil {
					// Log the error and stack trace
					log.Printf("PANIC: %v\n%s", r, debug.Stack())

					// Return error response
					data = nil
					statusCode = http.StatusInternalServerError
					err = nimbus.NewAPIError("internal_server_error", "An unexpected error occurred")
				}
			}()

			// Call next handler
			return next(ctx)
		}
	}
}

// DetailedRecovery returns a recovery middleware that includes error details in the response
func DetailedRecovery() nimbus.MiddlewareFunc {
	return func(next nimbus.HandlerFunc) nimbus.HandlerFunc {
		return func(ctx *nimbus.Context) (data any, statusCode int, err error) {
			defer func() {
				if r := recover(); r != nil {
					// Log the error and stack trace
					log.Printf("PANIC: %v\n%s", r, debug.Stack())

					// Return detailed error response
					message := fmt.Sprintf("Panic recovered: %v", r)
					data = nil
					statusCode = http.StatusInternalServerError
					err = nimbus.NewAPIError("internal_server_error", message)
				}
			}()

			// Call next handler
			return next(ctx)
		}
	}
}
