package nimbus

// MiddlewareFunc defines the middleware function signature
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Chain chains multiple middleware functions together
func Chain(middlewares ...MiddlewareFunc) MiddlewareFunc {
	return func(handler HandlerFunc) HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}
