package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/DylanHalstead/nimbus"
)

// CORSConfig defines CORS configuration options
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           3600,
	}
}

// CORS returns a CORS middleware with optional custom configuration
// If no config is provided, uses default configuration
func CORS(configs ...CORSConfig) nimbus.Middleware {
	config := DefaultCORSConfig()
	if len(configs) > 0 {
		config = configs[0]
	}

	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			origin := ctx.GetHeader("Origin")

			// Check if origin is allowed
			allowedOrigin := ""
			if len(config.AllowOrigins) > 0 {
				if config.AllowOrigins[0] == "*" {
					allowedOrigin = "*"
				} else {
					for _, o := range config.AllowOrigins {
						if o == origin {
							allowedOrigin = origin
							break
						}
					}
				}
			}

			// Set CORS headers
			if allowedOrigin != "" {
				ctx.Header("Access-Control-Allow-Origin", allowedOrigin)
			}

			if config.AllowCredentials {
				ctx.Header("Access-Control-Allow-Credentials", "true")
			}

			if len(config.ExposeHeaders) > 0 {
				ctx.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			}

			// Handle preflight requests
			if ctx.Request.Method == http.MethodOptions {
				if len(config.AllowMethods) > 0 {
					ctx.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
				}

				if len(config.AllowHeaders) > 0 {
					ctx.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
				}

				if config.MaxAge > 0 {
					ctx.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}

				// Return no content for preflight
				return nil, http.StatusNoContent, nil
			}

			// Call next handler
			return next(ctx)
		}
	}
}
