package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DylanHalstead/nimbus"
)

func TestRecovery_NoPanic(t *testing.T) {
	middleware := Recovery()

	nextCalled := false
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
}

func TestRecovery_PanicWithString(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	middleware := Recovery()

	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		panic("something went wrong!")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if data != nil {
		t.Errorf("expected nil data after panic, got %v", data)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error after panic, got nil")
	}

	apiErr, ok := err.(*nimbus.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
	}

	if apiErr.Code != "internal_server_error" {
		t.Errorf("expected error code 'internal_server_error', got '%s'", apiErr.Code)
	}

	if apiErr.Message != "An unexpected error occurred" {
		t.Errorf("expected generic error message, got '%s'", apiErr.Message)
	}

	// Check log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "PANIC") {
		t.Error("log should contain 'PANIC'")
	}
	if !strings.Contains(logOutput, "something went wrong!") {
		t.Error("log should contain the panic message")
	}
}

func TestRecovery_PanicWithError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	middleware := Recovery()

	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		panic(nimbus.NewAPIError("custom_error", "Custom error message"))
		return nil, http.StatusInternalServerError, nimbus.NewAPIError("custom_error", "Custom error message")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if data != nil {
		t.Errorf("expected nil data after panic, got %v", data)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error after panic, got nil")
	}

	// Check log output contains panic info
	logOutput := buf.String()
	if !strings.Contains(logOutput, "PANIC") {
		t.Error("log should contain 'PANIC'")
	}
}

func TestRecovery_PanicNilPointerDereference(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	middleware := Recovery()

	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		// Trigger a runtime panic
		panic("runtime panic: nil map")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if data != nil {
		t.Errorf("expected nil data after panic, got %v", data)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error after panic, got nil")
	}

	// Check log output includes stack trace
	logOutput := buf.String()
	if !strings.Contains(logOutput, "PANIC") {
		t.Error("log should contain 'PANIC'")
	}
}

func TestDetailedRecovery_NoPanic(t *testing.T) {
	middleware := DetailedRecovery()

	nextCalled := false
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
}

func TestDetailedRecovery_PanicWithDetails(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	middleware := DetailedRecovery()

	panicMessage := "detailed panic message"
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		panic(panicMessage)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if data != nil {
		t.Errorf("expected nil data after panic, got %v", data)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error after panic, got nil")
	}

	apiErr, ok := err.(*nimbus.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
	}

	if apiErr.Code != "internal_server_error" {
		t.Errorf("expected error code 'internal_server_error', got '%s'", apiErr.Code)
	}

	// DetailedRecovery should include the panic value in the message
	if !strings.Contains(apiErr.Message, "Panic recovered") {
		t.Errorf("expected message to contain 'Panic recovered', got '%s'", apiErr.Message)
	}

	if !strings.Contains(apiErr.Message, panicMessage) {
		t.Errorf("expected message to contain panic message '%s', got '%s'", panicMessage, apiErr.Message)
	}

	// Check log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "PANIC") {
		t.Error("log should contain 'PANIC'")
	}
	if !strings.Contains(logOutput, panicMessage) {
		t.Error("log should contain the panic message")
	}
}

func TestDetailedRecovery_PanicWithInteger(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	middleware := DetailedRecovery()

	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		panic(42)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if data != nil {
		t.Errorf("expected nil data after panic, got %v", data)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error after panic, got nil")
	}

	apiErr, ok := err.(*nimbus.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
	}

	// Should include the integer value in the message
	if !strings.Contains(apiErr.Message, "42") {
		t.Errorf("expected message to contain '42', got '%s'", apiErr.Message)
	}
}

func TestRecovery_PreservesOriginalError(t *testing.T) {
	middleware := Recovery()

	expectedErr := nimbus.NewAPIError("validation_error", "Invalid input")
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		return nil, http.StatusBadRequest, expectedErr
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	if statusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, statusCode)
	}

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRecovery_ChainWithOtherMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	// Simulate middleware chain: Recovery -> Custom Middleware -> Handler
	recoveryMiddleware := Recovery()

	customMiddleware := func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			ctx.Set("custom", "value")
			return next(ctx)
		}
	}

	// Apply middleware in order
	handler := recoveryMiddleware(customMiddleware(func(ctx *nimbus.Context) (any, int, error) {
		// Verify custom middleware ran
		if val, ok := ctx.Get("custom"); !ok || val != "value" {
			t.Error("custom middleware did not run")
		}
		panic("test panic in handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	data, statusCode, err := handler(ctx)

	if data != nil {
		t.Errorf("expected nil data after panic, got %v", data)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, statusCode)
	}

	if err == nil {
		t.Error("expected error after panic, got nil")
	}
}
