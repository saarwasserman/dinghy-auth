package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	users "saarwasserman.com/auth/grpcgen/users/proto"
)

func main() {

	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:8089", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := users.NewUsersServiceClient(conn)
	// res, err := client.RegisterUserHandler(context.Background(), &users.UserRequest{ Email: "test2@test2.com",
	// 	Name: "Test2", Password: "43sdf!4fd",})

	// res, err := client.ActivateUserHandler(context.Background(), &users.ActivationRequest{ TokenPlaintext: "5QWCJ6JKPGIZQ3WFNGMPJ3BHSI"})

	res, err := client.CreateAuthenticationTokenHandler(context.Background(), &users.AuthenticationRequest{Email: "test2@test2.com", Password: "43sdf!4fd"})
	if err != nil {
		log.Fatal("couldn't greet ", err.Error())
		return
	}

	fmt.Println(res)
}
