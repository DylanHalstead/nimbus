package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DylanHalstead/nimbus"
)

func TestCORS_DefaultConfig(t *testing.T) {
	middleware := CORS()
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return map[string]string{"message": "success"}, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	if statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, statusCode)
	}

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

func TestCORS_WildcardOrigin(t *testing.T) {
	config := DefaultCORSConfig()
	config.AllowOrigins = []string{"*"}

	middleware := CORS(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	handler(ctx)

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

func TestCORS_SpecificOrigins(t *testing.T) {
	allowedOrigins := []string{"http://example.com", "https://app.example.com"}
	config := CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost},
		AllowHeaders: []string{"Content-Type"},
	}

	middleware := CORS(config)

	testCases := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{"allowed origin 1", "http://example.com", "http://example.com"},
		{"allowed origin 2", "https://app.example.com", "https://app.example.com"},
		{"not allowed origin", "http://evil.com", ""},
		{"no origin header", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				return nil, http.StatusOK, nil
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			handler(ctx)

			origin := w.Header().Get("Access-Control-Allow-Origin")
			if origin != tc.expectedOrigin {
				t.Errorf("expected Access-Control-Allow-Origin '%s', got '%s'", tc.expectedOrigin, origin)
			}
		})
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"http://example.com"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           7200,
	}

	nextCalled := false
	middleware := CORS(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		nextCalled = true
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	// Preflight should return 204 No Content
	if statusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, statusCode)
	}

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Next handler should not be called for preflight
	if nextCalled {
		t.Error("next handler should not be called for preflight request")
	}

	// Check CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://example.com', got '%s'", origin)
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods != strings.Join(config.AllowMethods, ", ") {
		t.Errorf("expected Access-Control-Allow-Methods '%s', got '%s'", strings.Join(config.AllowMethods, ", "), methods)
	}

	if headers := w.Header().Get("Access-Control-Allow-Headers"); headers != strings.Join(config.AllowHeaders, ", ") {
		t.Errorf("expected Access-Control-Allow-Headers '%s', got '%s'", strings.Join(config.AllowHeaders, ", "), headers)
	}

	if creds := w.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials 'true', got '%s'", creds)
	}
}

func TestCORS_AllowCredentials(t *testing.T) {
	testCases := []struct {
		name             string
		allowCredentials bool
		expectedHeader   string
	}{
		{"credentials allowed", true, "true"},
		{"credentials not allowed", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := CORSConfig{
				AllowOrigins:     []string{"http://example.com"},
				AllowCredentials: tc.allowCredentials,
			}

			middleware := CORS(config)
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				return nil, http.StatusOK, nil
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "http://example.com")
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			handler(ctx)

			creds := w.Header().Get("Access-Control-Allow-Credentials")
			if creds != tc.expectedHeader {
				t.Errorf("expected Access-Control-Allow-Credentials '%s', got '%s'", tc.expectedHeader, creds)
			}
		})
	}
}

func TestCORS_ExposeHeaders(t *testing.T) {
	exposeHeaders := []string{"X-Request-ID", "X-Custom-Header"}
	config := CORSConfig{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: exposeHeaders,
	}

	middleware := CORS(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	handler(ctx)

	exposed := w.Header().Get("Access-Control-Expose-Headers")
	expected := strings.Join(exposeHeaders, ", ")
	if exposed != expected {
		t.Errorf("expected Access-Control-Expose-Headers '%s', got '%s'", expected, exposed)
	}
}

func TestCORS_NoExposeHeaders(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{},
	}

	middleware := CORS(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	handler(ctx)

	exposed := w.Header().Get("Access-Control-Expose-Headers")
	if exposed != "" {
		t.Errorf("expected no Access-Control-Expose-Headers, got '%s'", exposed)
	}
}

func TestCORS_ActualRequestAfterPreflight(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"http://example.com"},
		AllowMethods: []string{http.MethodPost},
		AllowHeaders: []string{"Content-Type"},
	}

	nextCalled := false
	middleware := CORS(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		nextCalled = true
		return map[string]string{"result": "ok"}, http.StatusOK, nil
	})

	// Actual request (not OPTIONS)
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	// Should call next handler for actual request
	if !nextCalled {
		t.Error("next handler should be called for actual request")
	}

	if statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, statusCode)
	}

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if data == nil {
		t.Error("expected data from next handler")
	}

	// Should still set CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://example.com', got '%s'", origin)
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	if len(config.AllowOrigins) != 1 || config.AllowOrigins[0] != "*" {
		t.Errorf("expected AllowOrigins ['*'], got %v", config.AllowOrigins)
	}

	expectedMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}
	if len(config.AllowMethods) != len(expectedMethods) {
		t.Errorf("expected %d methods, got %d", len(expectedMethods), len(config.AllowMethods))
	}

	if config.AllowCredentials {
		t.Error("expected AllowCredentials to be false by default")
	}

	if config.MaxAge != 3600 {
		t.Errorf("expected MaxAge 3600, got %d", config.MaxAge)
	}
}

func TestCORS_MaxAge(t *testing.T) {
	tests := []struct {
		name        string
		maxAge      int
		expected    string
		shouldExist bool
	}{
		{
			name:        "MaxAge set to 3600",
			maxAge:      3600,
			expected:    "3600",
			shouldExist: true,
		},
		{
			name:        "MaxAge set to 7200",
			maxAge:      7200,
			expected:    "7200",
			shouldExist: true,
		},
		{
			name:        "MaxAge set to 0 (not sent)",
			maxAge:      0,
			expected:    "",
			shouldExist: false,
		},
		{
			name:        "MaxAge set to -1 (not sent)",
			maxAge:      -1,
			expected:    "",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CORSConfig{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{http.MethodGet, http.MethodOptions},
				MaxAge:       tt.maxAge,
			}

			middleware := CORS(config)
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				return nil, http.StatusOK, nil
			})

			// Send OPTIONS request (preflight)
			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			req.Header.Set("Origin", "http://example.com")
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			handler(ctx)

			maxAgeHeader := w.Header().Get("Access-Control-Max-Age")

			if tt.shouldExist {
				if maxAgeHeader != tt.expected {
					t.Errorf("expected Access-Control-Max-Age '%s', got '%s'", tt.expected, maxAgeHeader)
				}
			} else {
				if maxAgeHeader != "" {
					t.Errorf("expected no Access-Control-Max-Age header, got '%s'", maxAgeHeader)
				}
			}
		})
	}
}

func TestCORS_MaxAge_NotOnActualRequest(t *testing.T) {
	// MaxAge should only be sent on preflight (OPTIONS) requests, not actual requests
	config := CORSConfig{
		AllowOrigins: []string{"*"},
		MaxAge:       3600,
	}

	middleware := CORS(config)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusOK, nil
	})

	// Send GET request (not preflight)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	handler(ctx)

	maxAgeHeader := w.Header().Get("Access-Control-Max-Age")
	if maxAgeHeader != "" {
		t.Errorf("expected no Access-Control-Max-Age on non-preflight request, got '%s'", maxAgeHeader)
	}
}
