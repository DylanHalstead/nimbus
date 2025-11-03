package nimbus

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_GET(t *testing.T) {
	router := NewRouter()

	router.AddRoute(http.MethodGet, "/test", func(ctx *Context) (any, int, error) {
		return map[string]any{"message": "success"}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_PathParameters(t *testing.T) {
	router := NewRouter()

	router.AddRoute(http.MethodGet, "/users/:id", func(ctx *Context) (any, int, error) {
		id := ctx.PathParams["id"]
		return map[string]any{"id": id}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestRouter_Middleware(t *testing.T) {
	router := NewRouter()

	called := false
	middleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (any, int, error) {
			called = true
			return next(ctx)
		}
	}

	router.Use(middleware)
	router.AddRoute(http.MethodGet, "/test", func(ctx *Context) (any, int, error) {
		return map[string]any{"message": "ok"}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !called {
		t.Error("Middleware was not called")
	}
}

func TestRouter_Group(t *testing.T) {
	router := NewRouter()

	api := router.Group("/api/v1")
	api.AddRoute(http.MethodGet, "/users", func(ctx *Context) (any, int, error) {
		return map[string]any{"users": []string{}}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_WithPathParams(t *testing.T) {
	router := NewRouter()

	type UserParams struct {
		ID string `path:"id"`
	}

	userParamsValidator := NewValidator(&UserParams{})

	// Use WithTyped for type-safe parameter handling
	handler := func(ctx *Context, req *TypedRequest[UserParams, struct{}, struct{}]) (any, int, error) {
		return map[string]any{"id": req.Params.ID}, http.StatusOK, nil
	}

	router.AddRoute(http.MethodGet, "/users/:id", WithTyped(handler, userParamsValidator, nil, nil))

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestMatchPattern has been removed as matchPattern() function was optimized away.
// Route matching is now handled by the radix tree implementation.
// See tree_test.go for comprehensive route matching tests.
