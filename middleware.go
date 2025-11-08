package nimbus

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler

// Chain chains multiple middleware functions together
func Chain(middlewares ...Middleware) Middleware {
	return func(handler Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}
