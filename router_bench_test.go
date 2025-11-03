package nimbus

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkRouter_StaticRoute(b *testing.B) {
	router := NewRouter()
	router.AddRoute(http.MethodGet, "/test", func(ctx *Context) (any, int, error) {
		return map[string]any{"status": "ok"}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_ParameterRoute(b *testing.B) {
	router := NewRouter()
	router.AddRoute(http.MethodGet, "/users/:id", func(ctx *Context) (any, int, error) {
		id := ctx.PathParams["id"]
		return map[string]any{"id": id}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_MultipleParameters(b *testing.B) {
	router := NewRouter()
	router.AddRoute(http.MethodGet, "/posts/:postId/comments/:commentId", func(ctx *Context) (any, int, error) {
		postId := ctx.PathParams["postId"]
		commentId := ctx.PathParams["commentId"]
		return map[string]any{"post": postId, "comment": commentId}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/posts/123/comments/456", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_WithMiddleware(b *testing.B) {
	router := NewRouter()

	// Add middleware
	router.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (any, int, error) {
			ctx.Set("processed", true)
			return next(ctx)
		}
	})

	router.AddRoute(http.MethodGet, "/test", func(ctx *Context) (any, int, error) {
		return map[string]any{"status": "ok"}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_WithMultipleMiddleware(b *testing.B) {
	router := NewRouter()

	// Add multiple middleware
	for i := 0; i < 5; i++ {
		router.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) (any, int, error) {
				return next(ctx)
			}
		})
	}

	router.AddRoute(http.MethodGet, "/test", func(ctx *Context) (any, int, error) {
		return map[string]any{"status": "ok"}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Groups(b *testing.B) {
	router := NewRouter()

	api := router.Group("/api/v1")
	api.AddRoute(http.MethodGet, "/users", func(ctx *Context) (any, int, error) {
		return map[string]any{"users": []string{}}, http.StatusOK, nil
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkContext_JSON(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := NewContext(w, req)

	data := map[string]any{"status": "ok", "message": "test"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		ctx.Writer = w
		ctx.JSON(http.StatusOK, data)
	}
}

func BenchmarkContext_Param(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/123", nil)
	ctx := NewContext(w, req)
	ctx.PathParams = map[string]string{"id": "123"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ctx.PathParams["id"]
	}
}

func BenchmarkContext_Query(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?page=1&limit=10", nil)
	ctx := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ctx.Query("page")
	}
}

func BenchmarkContext_SetGet(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := NewContext(w, req)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx.Set("key", "value")
		_, _ = ctx.Get("key")
	}
}
