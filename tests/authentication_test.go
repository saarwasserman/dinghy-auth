package main

import (
	"context"
	"log"
	"regexp"
	"testing"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/protogen/auth"
)

func Hello(name string) (string, error) {
	return name, nil
}

func TestHelloName(t *testing.T) {
	name := "Gladys"
	want := regexp.MustCompile(`\b` + name + `\b`)
	msg, err := Hello("Gladys")
	if !want.MatchString(msg) || err != nil {
		t.Fatalf(`Hello("Gladys") = %q, %v, want match for %#q, nil`, msg, err, want)
	}
}

func TestAuthenticate(t *testing.T) {
	authClient, err := NewAuthClient("localhost", 40020)
	if err != nil {
		log.Fatal("couldn't authenticate", err.Error())
		return
	}

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
