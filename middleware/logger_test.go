package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DylanHalstead/nimbus"
	"github.com/rs/zerolog"
)

func TestLogger_BasicLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	config := LoggerConfig{
		Logger: &logger,
	}

	nextCalled := false
	middleware := Logger(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		nextCalled = true
		return map[string]string{"message": "success"}, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if !nextCalled {
		t.Error("next handler was not called")
	}

	if statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, statusCode)
	}

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if data == nil {
		t.Error("expected data, got nil")
	}

	// Check log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "method") {
		t.Error("log should contain 'method'")
	}
	if !strings.Contains(logOutput, "GET") {
		t.Error("log should contain 'GET'")
	}
	if !strings.Contains(logOutput, "/test") {
		t.Error("log should contain '/test'")
	}
	if !strings.Contains(logOutput, "duration") {
		t.Error("log should contain 'duration'")
	}
	if !strings.Contains(logOutput, "status") {
		t.Error("log should contain 'status'")
	}
}

func TestLogger_SkipPaths(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	config := LoggerConfig{
		Logger:    &logger,
		SkipPaths: []string{"/health", "/metrics"},
	}

	middleware := Logger(config)

	testCases := []struct {
		path          string
		shouldLog     bool
		expectedCount int
	}{
		{"/health", false, 0},
		{"/metrics", false, 0},
		{"/api/users", true, 1},
		{"/test", true, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			buf.Reset()

			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				return nil, http.StatusOK, nil
			})

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			handler(ctx)

			logOutput := buf.String()
			hasLog := len(logOutput) > 0 && strings.Contains(logOutput, "HTTP request")

			if tc.shouldLog && !hasLog {
				t.Errorf("expected log for path %s, but got none", tc.path)
			}
			if !tc.shouldLog && hasLog {
				t.Errorf("expected no log for path %s, but got: %s", tc.path, logOutput)
			}
		})
	}
}

func TestLogger_LogIP(t *testing.T) {
	testCases := []struct {
		name   string
		logIP  bool
		ipAddr string
	}{
		{"with IP logging", true, "192.168.1.1:12345"},
		{"without IP logging", false, "10.0.0.1:54321"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf).With().Timestamp().Logger()

			config := LoggerConfig{
				Logger: &logger,
				LogIP:  tc.logIP,
			}

			middleware := Logger(config)
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				return nil, http.StatusOK, nil
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tc.ipAddr
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			handler(ctx)

			logOutput := buf.String()
			containsIP := strings.Contains(logOutput, tc.ipAddr)

			if tc.logIP && !containsIP {
				t.Errorf("expected IP address in log, but got: %s", logOutput)
			}
			if !tc.logIP && containsIP {
				t.Errorf("expected no IP address in log, but got: %s", logOutput)
			}
		})
	}
}

func TestLogger_LogUserAgent(t *testing.T) {
	testCases := []struct {
		name         string
		logUserAgent bool
		userAgent    string
	}{
		{"with user agent logging", true, "Mozilla/5.0 Test Browser"},
		{"without user agent logging", false, "curl/7.68.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf).With().Timestamp().Logger()

			config := LoggerConfig{
				Logger:       &logger,
				LogUserAgent: tc.logUserAgent,
			}

			middleware := Logger(config)
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				return nil, http.StatusOK, nil
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			handler(ctx)

			logOutput := buf.String()
			containsUA := strings.Contains(logOutput, "user_agent")

			if tc.logUserAgent && !containsUA {
				t.Errorf("expected user agent in log, but got: %s", logOutput)
			}
			if !tc.logUserAgent && containsUA {
				t.Errorf("expected no user agent in log, but got: %s", logOutput)
			}
		})
	}
}

func TestLogger_LogHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	config := LoggerConfig{
		Logger:     &logger,
		LogHeaders: []string{"X-Custom-Header", "X-Request-Source"},
	}

	middleware := Logger(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("X-Request-Source", "mobile-app")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	handler(ctx)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "header_X-Custom-Header") {
		t.Error("log should contain custom header key")
	}
	if !strings.Contains(logOutput, "custom-value") {
		t.Error("log should contain custom header value")
	}
	if !strings.Contains(logOutput, "header_X-Request-Source") {
		t.Error("log should contain request source header key")
	}
	if !strings.Contains(logOutput, "mobile-app") {
		t.Error("log should contain request source header value")
	}
}

func TestLogger_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	config := LoggerConfig{
		Logger: &logger,
	}

	middleware := Logger(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	// Set a request ID in the context (as would be done by RequestID middleware)
	ctx.Set("request_id", "test-request-id-12345")

	handler(ctx)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "request_id") {
		t.Error("log should contain request_id field")
	}
	if !strings.Contains(logOutput, "test-request-id-12345") {
		t.Error("log should contain the actual request ID value")
	}
}

func TestLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	config := LoggerConfig{
		Logger: &logger,
	}

	middleware := Logger(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusInternalServerError, nimbus.NewAPIError("test_error", "Something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error, got nil")
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "error") {
		t.Error("log should contain error field")
	}
	if !strings.Contains(logOutput, "500") {
		t.Error("log should contain status 500")
	}
}

func TestLogger_Duration(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	config := LoggerConfig{
		Logger: &logger,
	}

	middleware := Logger(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	handler(ctx)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "duration") {
		t.Error("log should contain duration field")
	}
}

func TestProductionLoggerConfig(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	// Use production config but with our test logger
	config := ProductionLoggerConfig()
	config.Logger = &logger

	middleware := Logger(config)

	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	// Test regular path
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "TestClient/1.0")
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	if statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, statusCode)
	}

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "ip") {
		t.Error("production config should log IP")
	}
	if !strings.Contains(logOutput, "user_agent") {
		t.Error("production config should log user agent")
	}
	if !strings.Contains(logOutput, "header_Authorization") {
		t.Error("production config should log Authorization header")
	}
}

func TestDevelopmentLoggerConfig(t *testing.T) {
	config := DevelopmentLoggerConfig()

	if config.Logger == nil {
		t.Error("expected Logger to be set")
	}

	if len(config.SkipPaths) != 0 {
		t.Errorf("expected SkipPaths to be empty, got %v", config.SkipPaths)
	}

	if config.LogIP {
		t.Error("expected LogIP to be false for development")
	}

	if config.LogUserAgent {
		t.Error("expected LogUserAgent to be false for development")
	}

	if len(config.LogHeaders) != 0 {
		t.Errorf("expected LogHeaders to be empty, got %v", config.LogHeaders)
	}
}

func TestMinimalLoggerConfig(t *testing.T) {
	config := MinimalLoggerConfig()

	if config.Logger == nil {
		t.Error("expected Logger to be set")
	}

	if config.LogIP {
		t.Error("minimal config should not log IP")
	}

	if config.LogUserAgent {
		t.Error("minimal config should not log user agent")
	}

	expectedSkipPaths := []string{"/health", "/metrics", "/favicon.ico"}
	if len(config.SkipPaths) != len(expectedSkipPaths) {
		t.Errorf("expected %d skip paths, got %d", len(expectedSkipPaths), len(config.SkipPaths))
	}
}

func TestVerboseLoggerConfig(t *testing.T) {
	config := VerboseLoggerConfig()

	if config.Logger == nil {
		t.Error("expected Logger to be set")
	}

	if !config.LogIP {
		t.Error("verbose config should log IP")
	}

	if !config.LogUserAgent {
		t.Error("verbose config should log user agent")
	}

	if len(config.LogHeaders) == 0 {
		t.Error("verbose config should log headers")
	}

	// Verbose should not skip any paths
	if len(config.SkipPaths) != 0 {
		t.Error("verbose config should not skip paths")
	}
}
