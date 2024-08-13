package main

import (
	"context"
	"errors"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/internal/validator"
	"github.com/saarwasserman/auth/protogen/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (app *application) isValidAuthenticationToken(token_scope, token_plaintext string) (*data.Token, error) {
	v := validator.New()

	if data.ValidateTokenPlaintext(v, token_plaintext); !v.Valid() {
		return nil, status.Error(codes.Unauthenticated, "invalid auth token")
	}

	token, err := app.models.Tokens.GetForToken(token_scope, token_plaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return nil, status.Error(codes.Unauthenticated, "invalid auth token")
		default:
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
	}

	return token, nil
}

func (app *application) Authenticate(ctx context.Context, req *auth.AuthenticationRequest) (*auth.AuthenticationResponse, error) {
	token, err := app.isValidAuthenticationToken(req.TokenScope, req.TokenPlaintext)
	if err != nil {
		app.logger.PrintError(err, nil)
		return nil, err
	}

	return &auth.AuthenticationResponse{
		UserId: token.UserID,
	}, nil
}
