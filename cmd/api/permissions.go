package main

import (
	"context"

	"github.com/saarwasserman/auth/protogen/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


func (app *application) AddPermissionForUser(ctx context.Context, req *auth.AddPermissionForUserRequest) (*auth.AddPermissionForUserResponse, error) {
	err := app.models.Permissions.AddForUser(req.UserId, req.Codes...)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &auth.AddPermissionForUserResponse{}, nil 
}
	
func (app *application) RemovePermissionForUser(ctx context.Context, req *auth.RemovePermissionForUserRequest) (*auth.RemovePermissionForUserResponse, error) {
	err := app.models.Permissions.DeleteForUser(req.UserId, req.Codes...)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &auth.RemovePermissionForUserResponse{}, nil 
}
