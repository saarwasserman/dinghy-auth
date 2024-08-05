package main

import (
	"context"
)

type ContextKey string

const userIdContextKey = ContextKey("userId")

func (app *application) contextSetUserId(ctx context.Context, userId int64) context.Context {
	ctx = context.WithValue(ctx, userIdContextKey, userId)
	return ctx
}

func (app *application) contextGetUserId(ctx context.Context) int64 {
	userId, ok := ctx.Value(userIdContextKey).(int64)
	if !ok {
		panic("missing userId value in request context")
	}

	return userId
}
