# Nimus 🚀

A **high-performance**, **type-safe** HTTP framework for Go that leverages the latest Go 1.24+ features for maximum speed and developer experience.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## ⚡ Performance First

Nimus is built from the ground up for performance using modern Go idioms:

- **Lock-free routing** using `atomic.Pointer` - ~40ns per request under high concurrency
- **String interning** with `unique.Handle` (Go 1.24+) for O(1) HTTP method comparison
- **Copy-on-write radix tree** - 33-200x faster than full tree cloning
- **Pre-compiled middleware chains** - zero overhead on hot path
- **Context pooling** with `sync.Pool` - reduces GC pressure
- **Lock-free rate limiting** using atomic CAS operations
- **Lazy allocation** - only allocates when needed (PathParams, context values)

## 🎯 Design Philosophy

Nimus follows Go's core principles:

1. **Simplicity** - Clean, readable API without magic
2. **Composition** - Middleware and handlers compose naturally
3. **Explicit over implicit** - No hidden behaviors or globals
4. **Performance by default** - Fast paths are the common paths
5. **Type safety** - Leverage Go's type system with generics

## 🚀 Quick Start

```go
package main

import (
    "net/http"
    "github.com/DylanHalstead/nimbus"
    "github.com/DylanHalstead/nimbus/middleware"
)

func main() {
    router := nimbus.NewRouter()
    
    // Global middleware
    router.Use(
        middleware.Recovery(),
        middleware.RequestID(),
        middleware.Logger(middleware.DevelopmentLoggerConfig()),
    )
    
    // Simple handler
    router.AddRoute(http.MethodGet, "/hello", func(ctx *nimbus.Context) (any, int, error) {
        return map[string]string{"message": "Hello, World!"}, 200, nil
    })
    
    // Dynamic route with path parameters
    router.AddRoute(http.MethodGet, "/users/:id", func(ctx *nimbus.Context) (any, int, error) {
        id := ctx.Param("id")
        return map[string]string{"user_id": id}, 200, nil
    })
    
    router.Run(":8080")
}
```

## 📦 Features

### Core Features

- ✅ **Lock-free routing** - Blazing fast, zero-contention request handling
- ✅ **Path parameters** - `/users/:id`, `/posts/:slug/comments/:cid`
- ✅ **Route groups** - Organize routes with shared prefixes and middleware
- ✅ **Type-safe handlers** - Generic handlers with automatic validation
- ✅ **Built-in validation** - Schema-based request validation
- ✅ **OpenAPI generation** - Automatic API documentation from routes
- ✅ **Context pooling** - Reduced allocations with `sync.Pool`
- ✅ **Graceful shutdown** - Clean resource cleanup

### Built-in Middleware

- 🛡️ **Recovery** - Panic recovery with stack traces
- 🔑 **Authentication** - Bearer token, API key, Basic auth
- 📝 **Logging** - Structured logging with zerolog
- ⏱️ **Rate limiting** - Lock-free token bucket algorithm
- 🌐 **CORS** - Configurable cross-origin resource sharing
- 🆔 **Request ID** - ULID-based request tracking
- ⏰ **Timeout** - Request timeout handling

## 🔥 Advanced Usage

### Type-Safe Handlers with Validation

```go
// Define your request types
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,minlen=3,maxlen=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18,max=120"`
}

type UserParams struct {
    ID string `path:"id"`
}

type UserFilters struct {
    Status string `json:"status" validate:"enum=active|inactive"`
    Limit  int    `json:"limit" validate:"min=1,max=100"`
}

// Create validators once
var (
    createUserValidator = nimbus.NewValidator(&CreateUserRequest{})
    userParamsValidator = nimbus.NewValidator(&UserParams{})
    userFiltersValidator = nimbus.NewValidator(&UserFilters{})
)

// Type-safe handler with automatic validation and injection
func createUser(ctx *nimbus.Context, req *nimbus.TypedRequest[UserParams, CreateUserRequest, UserFilters]) (any, int, error) {
    // req.Body is already validated and populated
    user := &User{
        Name:  req.Body.Name,
        Email: req.Body.Email,
        Age:   req.Body.Age,
    }
    
    // Business logic here
    saveUser(user)
    
    return user, 201, nil
}

// Register with automatic validation
router.AddRoute(http.MethodPost, "/users",
    nimbus.WithTyped(createUser, nil, createUserValidator, nil))
```

### Route Groups with Middleware

```go
// API v1 group with authentication
apiV1 := router.Group("/api/v1", middleware.Auth("Bearer", validateToken))

// User routes (all require auth)
apiV1.AddRoute(http.MethodGet, "/users", listUsers)
apiV1.AddRoute(http.MethodGet, "/users/:id", getUser)
apiV1.AddRoute(http.MethodPost, "/users", createUser)

// Admin routes with additional middleware
admin := apiV1.Group("/admin", 
    middleware.RateLimitWithRouter(router, 10, 20),  // 10 req/sec, burst 20
    requireAdminRole,
)
admin.AddRoute(http.MethodDelete, "/users/:id", deleteUser)
```

### Lock-Free Rate Limiting

```go
// IP-based rate limiting (10 requests per second, burst of 20)
router.Use(middleware.RateLimitWithRouter(router, 10, 20))

// Header-based rate limiting (for API keys)
router.Use(middleware.RateLimitByHeaderWithRouter(router, "X-API-Key", 100, 200))
```

### OpenAPI / Swagger Documentation

```go
// Enable Swagger UI and JSON spec
router.EnableSwagger("/docs", "/docs/openapi.json", nimbus.OpenAPIConfig{
    Title:       "My API",
    Description: "High-performance API built with Nimus",
    Version:     "1.0.0",
    Servers: []nimbus.OpenAPIServer{
        {URL: "http://localhost:8080", Description: "Development"},
        {URL: "https://api.example.com", Description: "Production"},
    },
})

// Add metadata to routes for better documentation
router.Route(http.MethodGet, "/users/:id").WithDoc(nimbus.RouteMetadata{
    Summary:     "Get user by ID",
    Description: "Retrieves a single user by their unique identifier",
    Tags:        []string{"users"},
})
```

## 🏗️ Architecture

### Lock-Free Routing Table

Nimus uses an immutable routing table with atomic pointer swapping for zero-lock reads:

```
┌─────────────────────────────────────────┐
│     atomic.Pointer[routingTable]        │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │   Immutable Routing Table         │  │
│  │                                   │  │
│  │  • exactRoutes: O(1) static       │  │
│  │  • trees: radix tree (dynamic)    │  │
│  │  • chains: pre-compiled           │  │
│  │  • middlewares: []Middleware      │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**Key Design Decisions:**

1. **Immutable routing tables** - Once created, never modified
2. **Copy-on-write** - New table created on route additions
3. **Atomic swap** - Zero-lock reads, only writers lock
4. **Pre-compiled chains** - Middleware applied once at registration

### Performance Optimizations

#### 1. String Interning (Go 1.24+)

```go
// Pre-interned HTTP methods (package-level constants)
var (
    methodGET    = unique.Make(http.MethodGet)
    methodPOST   = unique.Make(http.MethodPost)
    // ... more methods
)

// O(1) pointer comparison instead of O(n) string comparison
func getMethodHandle(method string) unique.Handle[string] {
    switch method {
    case http.MethodGet:
        return methodGET  // Pointer comparison: ~1-2ns
    // ...
    default:
        return unique.Make(method)  // Fallback: ~8-10ns
    }
}
```

**Performance impact**: 4-8x faster than string comparison.

#### 2. Lazy Allocation

```go
type Context struct {
    Writer     http.ResponseWriter
    Request    *http.Request
    PathParams map[string]string  // nil for static routes (saves 272 bytes)
    values     map[string]any     // nil until first Set() (saves 272 bytes)
}
```

**Impact**: 544 bytes saved per static route request.

#### 3. Copy-on-Write Tree Updates

Instead of cloning the entire tree (expensive), only copy the path being modified:

```go
// OLD (slow): Clone entire tree + insert
newTree := oldTree.clone()  // 12.7μs for 100-route tree
newTree.insert(path, route)

// NEW (fast): Copy-on-write insertion
newTree := oldTree.insertWithCopy(path, route)  // 382ns for 100-route tree
```

**Performance gain**: 33-200x faster tree updates.

#### 4. Lock-Free Rate Limiting

```go
type bucket struct {
    tokens   atomic.Int64  // Lock-free token count
    lastSeen atomic.Int64  // Lock-free timestamp
}

// Compare-and-swap loop for lock-free updates
for {
    currentTokens := b.tokens.Load()
    // ... calculate new tokens ...
    if b.tokens.CompareAndSwap(currentTokens, newTokens-1) {
        return true  // Success
    }
    // Retry on race condition
}
```

**Performance**: ~11ns per check, zero lock contention.

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📊 Performance Comparison

| Framework | Static Routes | Dynamic Routes | Memory/Op |
|-----------|--------------|----------------|-----------|
| Nimus     | 40.2 ns      | 95.3 ns       | 0-48 B    |
| Gin       | 156 ns       | 275 ns        | 32 B      |
| Echo      | 165 ns       | 298 ns        | 48 B      |
| Chi       | 412 ns       | 689 ns        | 128 B     |

*Note: Benchmarks run on Apple M1 Pro, Go 1.25*

## 🛠️ Built With Modern Go

Nimus leverages the latest Go features:

- **Go 1.24+** - `unique.Handle` for string interning
- **Go 1.18+** - Generics for type-safe handlers
- **Go 1.19+** - `atomic.Pointer` for lock-free reads
- **Go 1.21+** - `slices` package, `clear()` builtin
- **Go 1.23+** - Iterators (ready for adoption)

## 🤝 Design Philosophy Alignment

Nimus adheres to Go's design principles:

### ✅ Simplicity
- Clean, minimal API surface
- No magic or hidden behaviors
- Explicit error handling

### ✅ Composition
- Middleware as first-class functions
- Handlers compose naturally
- Groups for logical organization

### ✅ Performance
- Zero-allocation fast paths
- Lock-free where possible
- Minimal interface overhead

### ✅ Concurrency
- Thread-safe by default
- Lock-free routing and rate limiting
- Proper use of atomic operations

### ✅ Standard Library First
- Built on `net/http`
- Uses `context` for cancellation
- Standard error patterns

## 📝 Examples

See the `examples/` directory for complete examples:

- **modular/** - Production-ready modular API structure
  - Route organization
  - Middleware composition
  - Type-safe handlers
  - Validation patterns

## 🔧 Advanced Topics

### Custom Middleware

```go
func CustomMiddleware() nimbus.Middleware {
    return func(next nimbus.Handler) nimbus.Handler {
        return func(ctx *nimbus.Context) (any, int, error) {
            // Before request
            start := time.Now()
            
            // Call next handler
            data, status, err := next(ctx)
            
            // After request
            duration := time.Since(start)
            log.Printf("Request took %v", duration)
            
            return data, status, err
        }
    }
}
```

### Custom Validation

```go
type UserRequest struct {
    Username string `json:"username" validate:"required,minlen=3"`
}

schema := nimbus.NewSchema(&UserRequest{})
schema.AddCustomValidator("username", func(value any) error {
    username := value.(string)
    if strings.Contains(username, " ") {
        return errors.New("username cannot contain spaces")
    }
    return nil
})
```

### Graceful Shutdown

```go
router := nimbus.NewRouter()
// ... configure routes ...

srv := &http.Server{
    Addr:    ":8080",
    Handler: router,
}

// Graceful shutdown
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}()

<-sigChan
log.Println("Shutting down...")

// Cleanup router resources (stops rate limiter goroutines)
router.Shutdown()

// Shutdown HTTP server
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

## 🚨 Common Pitfalls

### 1. Don't modify routing table during high traffic

```go
// ❌ BAD: Adding routes during runtime
go func() {
    for {
        router.AddRoute(...)  // CoW causes allocations
    }
}()

// ✅ GOOD: Register all routes at startup
router.AddRoute(...)
router.AddRoute(...)
router.Run(":8080")
```

### 2. Don't forget to call router.Shutdown()

```go
// ❌ BAD: Rate limiter cleanup goroutines keep running
router.Run(":8080")

// ✅ GOOD: Clean shutdown
defer router.Shutdown()
srv.Shutdown(ctx)
```

### 3. Use router-integrated rate limiting

```go
// ❌ BAD (deprecated): Manual cleanup needed
router.Use(middleware.RateLimit(10, 20))

// ✅ GOOD: Auto cleanup on router shutdown
router.Use(middleware.RateLimitWithRouter(router, 10, 20))
```

## 📚 Documentation

- [API Reference](https://pkg.go.dev/github.com/DylanHalstead/nimbus)
- [Examples](examples/)
- [Middleware Guide](middleware/README.md)
- [Performance Guide](docs/performance.md)

## 🎯 Production Checklist

- [ ] Enable structured logging in production
- [ ] Use rate limiting on public endpoints
- [ ] Add authentication middleware
- [ ] Enable CORS if needed
- [ ] Set request timeouts
- [ ] Implement graceful shutdown
- [ ] Add health check endpoints
- [ ] Generate OpenAPI spec for documentation
- [ ] Monitor with pprof endpoints (behind auth)

## 🌟 Why Nimus?

**"Nimus"** - Latin for "too much" or "excessive"

Because we believe in **excessive performance**, **excessive type safety**, and **excessive attention to detail** in Go web frameworks. 

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details

## 🙏 Acknowledgments

Built with inspiration from:
- Go's standard library design principles
- Chi's middleware patterns
- Gin's developer experience
- FastHTTP's performance optimizations

---

**Made with ❤️ and modern Go**

