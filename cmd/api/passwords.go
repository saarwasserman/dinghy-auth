package main

import (
	"context"

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

	return &auth.SetPasswordResponse{}, nil
}
