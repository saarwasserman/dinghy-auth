package main

import (
	"context"
	"log"
	"testing"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/protogen/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestCreateToken(t *testing.T) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:40020", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	authClient := auth.NewAuthenticationClient(conn)

	res, err := authClient.CreateToken(context.Background(), &auth.TokenCreationRequest{Scope: data.ScopeAuthentication, UserId: 11})
	if err != nil {
		log.Fatal("couldn't create token", err.Error())
		return
	}

	if len(res.TokenPlaintext) != 26 {
		t.Errorf("token length is not equal to 26")
	}
}
