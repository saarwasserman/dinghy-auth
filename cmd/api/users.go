package main

import (
	"context"
	"errors"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	notifications "saarwasserman.com/auth/grpcgen/notifications/proto"
	users "saarwasserman.com/auth/grpcgen/users/proto"
	"saarwasserman.com/auth/internal/data"
	"saarwasserman.com/auth/internal/validator"
)

func (app *application) RegisterUserHandler(ctx context.Context, req *users.UserRequest) (*users.UserResponse, error) {
	user := &data.User{
		Name:      req.Name,
		Email:     req.Email,
		Activated: false,
	}

	err := user.Password.Set(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "the server encountered a problem and could not process your request")
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		return nil, status.Errorf(codes.InvalidArgument, "error %s", v.Errors)
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			return nil, status.Errorf(codes.InvalidArgument, "error %s", v.Errors)
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:8090", opts...)
	if err != nil {
		app.logger.PrintFatal(err, nil)
		return nil, err
	}
	defer conn.Close()

	emailClient := notifications.NewEMailServiceClient(conn)
	_, err = emailClient.SendActivationEmail(context.Background(), &notifications.SendActivationEmailRequest{
		Recipient: user.Email,
		UserId:    strconv.FormatInt(user.ID, 10),
		Token:     token.Plaintext,
	})
	if err != nil {
		app.logger.PrintFatal(err, nil)
		return nil, err
	}

	return &users.UserResponse{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.UnixMilli(),
		Activated: user.Activated,
	}, nil
}

func (app *application) ActivateUserHandler(ctx context.Context, req *users.ActivationRequest) (*users.UserResponse, error) {

	v := validator.New()

	if data.ValidateTokenPlaintext(v, req.TokenPlaintext); !v.Valid() {
		return nil, status.Errorf(codes.InvalidArgument, "error: %s", v.Errors)
	}

	user, err := app.models.Users.GetForToken(data.ScopeActivation, req.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			return nil, status.Errorf(codes.InvalidArgument, "error: %s", v.Errors)
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	user.Activated = true

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			return nil, status.Error(codes.InvalidArgument, "unable to update the record due to an edit conflict, please try again")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &users.UserResponse{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.UnixMilli(),
		Activated: user.Activated,
	}, nil
}

func (app *application) CreateAuthenticationTokenHandler(ctx context.Context, req *users.AuthenticationRequest) (*users.AuthenticationResponse, error) {

	v := validator.New()

	data.ValidateEmail(v, req.Email)
	data.ValidatePlaintextPassword(v, req.Password)

	if !v.Valid() {
		return nil, status.Errorf(codes.InvalidArgument, "error: %s", v.Errors)
	}

	user, err := app.models.Users.GetByEmail(req.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	match, err := user.Password.Matches(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !match {
		return nil, status.Error(codes.Unauthenticated, "wrong credentials")
	}

	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &users.AuthenticationResponse{
		TokenPlaintext: token.Plaintext,
		Expiry:         token.Expiry.UnixMilli(),
	}, nil
}
