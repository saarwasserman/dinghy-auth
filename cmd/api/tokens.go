package main

import (
	"context"
	"time"

	"github.com/saarwasserman/auth/protogen/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (app *application) CreateToken(ctx context.Context, req *auth.TokenCreationRequest) (*auth.TokenCreationResponse, error) {
	app.models.Tokens.DeleteAllForUser(req.Scope, req.UserId)

	token, err := app.models.Tokens.New(req.UserId, 24*time.Hour, req.Scope)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &auth.TokenCreationResponse{
		TokenPlaintext: token.Plaintext,
		Expiry:         token.Expiry.UnixMilli(),
	}, nil
}

func (app *application) DeleteAllTokensForUser(ctx context.Context, req *auth.TokensDeletionRequest) (*auth.TokensDeletionRequest, error) {

	err := app.models.Tokens.DeleteAllForUser(req.Scope, req.UserId)
	if err != nil {
		app.logger.PrintError(err, nil)
		return nil, err
	}

	return &auth.TokensDeletionRequest{}, nil
}
