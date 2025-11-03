package middleware

import (
	"os"
	"time"

	"github.com/DylanHalstead/nimbus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggerConfig defines configuration for the logger middleware
type LoggerConfig struct {
	Logger       *zerolog.Logger
	SkipPaths    []string // Paths to skip logging (e.g., health checks)
	LogIP        bool     // Whether to log IP addresses
	LogUserAgent bool     // Whether to log user agent
	LogHeaders   []string // Headers to log
}

// Preset logger configuration functions for different environments
// These functions create logger configs on-demand rather than at package init time

// DevelopmentLoggerConfig returns a human-readable console logger for development
func DevelopmentLoggerConfig() LoggerConfig {
	l := log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	return LoggerConfig{
		Logger:       &l,
		SkipPaths:    []string{},
		LogIP:        false,
		LogUserAgent: false,
		LogHeaders:   []string{},
	}
}

// ProductionLoggerConfig returns a structured JSON logger for production
func ProductionLoggerConfig() LoggerConfig {
	l := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return LoggerConfig{
		Logger:       &l,
		SkipPaths:    []string{"/health", "/metrics"},
		LogIP:        true,
		LogUserAgent: true,
		LogHeaders:   []string{"Authorization"},
	}
}

// MinimalLoggerConfig returns minimal logging (just method, path, status, duration)
func MinimalLoggerConfig() LoggerConfig {
	l := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return LoggerConfig{
		Logger:       &l,
		SkipPaths:    []string{"/health", "/metrics", "/favicon.ico"},
		LogIP:        false,
		LogUserAgent: false,
		LogHeaders:   []string{},
	}
}

// VerboseLoggerConfig returns detailed logging for debugging
func VerboseLoggerConfig() LoggerConfig {
	l := log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	return LoggerConfig{
		Logger:       &l,
		SkipPaths:    []string{},
		LogIP:        true,
		LogUserAgent: true,
		LogHeaders:   []string{"Authorization", "Content-Type", "Accept", "User-Agent", "X-Forwarded-For"},
	}
}

// Logger is a middleware that logs HTTP requests using zerolog.
// Accepts a LoggerConfig for full control over logging behavior.
// Use one of the preset configuration functions (DevelopmentLoggerConfig(), ProductionLoggerConfig(), etc.)
// or create a custom configuration.
//
// Examples:
//
//	// Use preset for development (call the function)
//	router.Use(middleware.Logger(middleware.DevelopmentLoggerConfig()))
//
//	// Use preset for production (call the function)
//	router.Use(middleware.Logger(middleware.ProductionLoggerConfig()))
//
//	// Custom configuration
//	router.Use(middleware.Logger(middleware.LoggerConfig{
//	    Logger:     myLogger,
//	    SkipPaths:  []string{"/health"},
//	    LogIP:      true,
//	}))
func Logger(config LoggerConfig) nimbus.MiddlewareFunc {
	return func(next nimbus.HandlerFunc) nimbus.HandlerFunc {
		return func(ctx *nimbus.Context) (any, int, error) {
			start := time.Now()
			path := ctx.Request.URL.Path
			method := ctx.Request.Method

			// Check if we should skip logging this path
			for _, skipPath := range config.SkipPaths {
				if path == skipPath {
					return next(ctx)
				}
			}

			// Call next handler
			data, statusCode, err := next(ctx)

			// Build log event
			duration := time.Since(start)
			event := config.Logger.Info().
				Str("method", method).
				Str("path", path).
				Dur("duration", duration).
				Int("status", statusCode)

			// Add request ID if available (automatically added by RequestID middleware)
			if requestID := ctx.GetString("request_id"); requestID != "" {
				event = event.Str("request_id", requestID)
			}

			// Add optional fields
			if config.LogIP {
				event = event.Str("ip", ctx.Request.RemoteAddr)
			}

			if config.LogUserAgent {
				event = event.Str("user_agent", ctx.Request.UserAgent())
			}

			// Log specified headers
			for _, header := range config.LogHeaders {
				if value := ctx.GetHeader(header); value != "" {
					event = event.Str("header_"+header, value)
				}
			}

			if err != nil {
				event = event.Err(err)
			}

			event.Msg("HTTP request")

			return data, statusCode, err
		}
	}
}
