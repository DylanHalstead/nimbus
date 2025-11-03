package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DylanHalstead/nimbus"
)

func TestAuth_MissingAuthHeader(t *testing.T) {
	// Create a mock validator that should never be called
	validateToken := func(token string) (any, error) {
		t.Fatal("validateToken should not be called when auth header is missing")
		return nil, nil
	}

	middleware := Auth(validateToken)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		t.Fatal("next handler should not be called when auth header is missing")
		return nil, 0, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	if statusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, statusCode)
	}

	if err == nil {
		t.Error("expected error, got nil")
	}

	apiErr, ok := err.(*nimbus.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
	}
	if apiErr.Code != "unauthorized" {
		t.Errorf("expected error code 'unauthorized', got '%s'", apiErr.Code)
	}
}

func TestAuth_InvalidAuthHeaderFormat(t *testing.T) {
	testCases := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "sometoken123"},
		{"only bearer", "Bearer"},
		{"wrong prefix", "Basic sometoken123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateToken := func(token string) (any, error) {
				t.Fatal("validateToken should not be called with invalid format")
				return nil, nil
			}

			middleware := Auth(validateToken)
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				t.Fatal("next handler should not be called with invalid format")
				return nil, 0, nil
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			_, statusCode, err := handler(ctx)

			if statusCode != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, statusCode)
			}

			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestAuth_ValidTokenButValidationFails(t *testing.T) {
	expectedError := errors.New("invalid token")

	validateToken := func(token string) (any, error) {
		if token != "validtoken123" {
			return nil, expectedError
		}
		return nil, expectedError // Return error even for the token we're testing
	}

	middleware := Auth(validateToken)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		t.Fatal("next handler should not be called when token validation fails")
		return nil, 0, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	w := httptest.NewRecorder()
	ctx := nimbus.NewContext(w, req)

	_, statusCode, err := handler(ctx)

	if statusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, statusCode)
	}

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestAuth_ValidToken(t *testing.T) {
	expectedUser := map[string]any{
		"id":   "123",
		"name": "Test User",
		"role": "admin",
	}

	validateToken := func(token string) (any, error) {
		if token == "validtoken123" {
			return expectedUser, nil
		}
		return nil, errors.New("invalid token")
	}

	nextCalled := false
	middleware := Auth(validateToken)
	handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
		nextCalled = true

		// Check that user was stored in context
		user, exists := ctx.Get("user")
		if !exists {
			t.Error("user not found in context")
		}

		userMap, ok := user.(map[string]any)
		if !ok {
			t.Errorf("expected user to be map[string]any, got %T", user)
		}

		// Verify user data
		if userMap["id"] != expectedUser["id"] {
			t.Errorf("expected user id %v, got %v", expectedUser["id"], userMap["id"])
		}

		return map[string]string{"message": "success"}, http.StatusOK, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer validtoken123")
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

func TestAuth_DifferentTokens(t *testing.T) {
	users := map[string]any{
		"token1": map[string]string{"id": "1", "name": "User 1"},
		"token2": map[string]string{"id": "2", "name": "User 2"},
		"token3": map[string]string{"id": "3", "name": "User 3"},
	}

	validateToken := func(token string) (any, error) {
		if user, ok := users[token]; ok {
			return user, nil
		}
		return nil, errors.New("invalid token")
	}

	middleware := Auth(validateToken)

	for token, expectedUser := range users {
		t.Run("token_"+token, func(t *testing.T) {
			handler := middleware(func(ctx *nimbus.Context) (any, int, error) {
				user, _ := ctx.Get("user")
				userMap, ok := user.(map[string]string)
				if !ok {
					t.Errorf("expected user to be map[string]string, got %T", user)
				}
				expectedMap, _ := expectedUser.(map[string]string)
				if userMap["id"] != expectedMap["id"] {
					t.Errorf("expected user id %v, got %v", expectedMap["id"], userMap["id"])
				}
				return nil, http.StatusOK, nil
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			ctx := nimbus.NewContext(w, req)

			_, statusCode, err := handler(ctx)

			if statusCode != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, statusCode)
			}

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
