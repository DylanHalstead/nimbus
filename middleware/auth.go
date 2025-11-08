package middleware

import (
	"net/http"
	"strings"

	"github.com/DylanHalstead/nimbus"
)

// Auth middleware validates authentication token
// This is a simple example - in production, use proper JWT validation
func Auth(validateToken func(string) (any, error)) nimbus.Middleware {
	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			authHeader := ctx.GetHeader("Authorization")

			if authHeader == "" {
				return nil, http.StatusUnauthorized, nimbus.NewAPIError("unauthorized", "Missing authorization header")
			}

			// Expect "Bearer <token>" format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return nil, http.StatusUnauthorized, nimbus.NewAPIError("unauthorized", "Invalid authorization header format")
			}

			token := parts[1]

			// Validate token
			user, err := validateToken(token)
			if err != nil {
				return nil, http.StatusUnauthorized, nimbus.NewAPIError("unauthorized", err.Error())
			}

			// Store user in context
			ctx.Set("user", user)

			// Call next handler
			return next(ctx)
		}
	}
}
