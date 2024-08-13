package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	interceptorsAuth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/internal/validator"
)

func (app *application) Authenticator(ctx context.Context) (context.Context, error) {
	token, err := interceptorsAuth.AuthFromMD(ctx, "bearer")
	if err != nil {
		fmt.Println(err.Error())
		return ctx, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, token); !v.Valid() {
		return ctx, status.Error(codes.Unauthenticated, "invalid auth token")
	}

	// token - check expiration
	userId, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return ctx, status.Error(codes.Unauthenticated, "invalid auth token")
		default:
			return ctx, status.Error(codes.Unauthenticated, err.Error())
		}
	}

	ctx = app.contextSetUserId(ctx, userId)
	return ctx, nil
}

func (app *application) AuthMatcher(ctx context.Context, callMeta interceptors.CallMeta) bool {
	// var requiredAuthenticationServices = []string{auth.UsersService_ServiceDesc.ServiceName}
	// methods := []string{auth.Authentication_ServiceDesc.Methods[1].MethodName}
	var methods []string
	return slices.Contains(methods, callMeta.Method)
}
