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

func TestAuthenticate(t *testing.T) {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:40020", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	authClient := auth.NewAuthenticationClient(conn)

	// get user details
	// TODO: get existing token or create a new one (fetch from tokens' tests)
	res, err := authClient.Authenticate(context.Background(), &auth.AuthenticationRequest{
		TokenScope:     data.ScopeAuthentication,
		TokenPlaintext: "VISIAMIDA5YZ4Y26N5TPLFLR44",
	})
	if err != nil {
		t.Errorf("error: %s", err.Error())
		return
	}

	if res.UserId >= 0 {
		log.Print("found a user with that token")
	}
}
