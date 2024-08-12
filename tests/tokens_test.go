package main

import (
	"context"
	"log"
	"testing"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/protogen/auth"
)

func TestCreateToken(t *testing.T) {
	authClient, err := NewAuthClient("localhost", 40020)
	if err != nil {
		log.Fatal("couldn't greet ", err.Error())
		return
	}

	res, err := authClient.CreateToken(context.Background(), &auth.TokenCreationRequest{Scope: data.ScopeAuthentication, UserId: 11})
	if err != nil {
		log.Fatal("couldn't create token", err.Error())
		return
	}

	if len(res.TokenPlaintext) != 26 {
		t.Errorf("token length is not equal to 26")
	}
}
