package yotei

import "context"

// Handler determines an interface of something
// that can be handled.
type Handler interface {
	// Handle is the callback to execute the task action.
	Handle(context.Context) Action
}

// HandlerFunc is a simple type to quickly transform a func
// into a [Handler].
type HandlerFunc func(context.Context) Action

// Hande runs the action
func (handler HandlerFunc) Handle(ctx context.Context) Action {
	return handler(ctx)
}
