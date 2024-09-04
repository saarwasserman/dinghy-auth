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


func (app *application) Authenticate(ctx context.Context, req *auth.AuthenticationRequest) (*auth.AuthenticationResponse, error) {

	tokenPlaintext := req.TokenPlaintext
	tokenScope := req.TokenScope

	v := validator.New()

	if data.ValidateTokenPlaintext(v, tokenPlaintext); !v.Valid() {
		return nil, status.Error(codes.Unauthenticated, "invalid auth token")
	}


	// check in cache
	token := app.models.Tokens.GetTokenFromCache(tokenScope, tokenPlaintext)
	var err error
	// if not in cance go to db
	if token == nil {
		token, err = app.models.Tokens.GetForToken(tokenScope, tokenPlaintext)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				return nil, status.Error(codes.Unauthenticated, "invalid auth token")
			default:
				return nil, status.Error(codes.Unauthenticated, err.Error())
			}
		} else {
			go func() {
				app.models.Tokens.UpdateCache(token)
			}()
		}
	}

	return &auth.AuthenticationResponse{
		UserId: token.UserID,
	}, nil
}
