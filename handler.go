package yotei

import "context"

type HandlerFunc func(context.Context) Action

func (handler HandlerFunc) Handle(ctx context.Context) Action {
	return handler(ctx)
}
