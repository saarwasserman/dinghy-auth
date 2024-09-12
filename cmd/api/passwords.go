package main

import (
	"context"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/protogen/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (app *application) SetPassword(ctx context.Context, req *auth.SetPasswordRequest) (*auth.SetPasswordResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	app.models.Passwords.CreatePasswordForUserId(req.UserId, hash)

	// after password change delete all tokens for user
	app.models.Tokens.DeleteAllForUser(data.ScopeAuthentication, req.UserId)
	app.models.Tokens.DeleteTokensCacheForUser(data.ScopeAuthentication, req.UserId)

	return &auth.SetPasswordResponse{}, nil
}
