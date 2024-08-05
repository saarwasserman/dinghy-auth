package main

import (
	"context"
	"flag"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/protogen/auth"
)

func main() {

	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:40020", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()


	clientTokens := auth.NewAuthenticationClient(conn)
	res, err := clientTokens.CreateToken(context.Background(), &auth.TokenCreationRequest{Scope: data.ScopeAuthentication, UserId: 11})
	if err != nil {
		log.Fatal("couldn't greet ", err.Error())
		return
	}


	// with authorization header
	// md := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
	// ctx := metadata.NewOutgoingContext(context.Background(), md)

	// // Make RPC using the context with the metadata.
	// var header, trailer metadata.MD
	// r, err := c.UnaryEcho(ctx, &pb.EchoRequest{Message: message}, grpc.Header(&header), grpc.Trailer(&trailer))
	// if err != nil {
	// 	log.Fatalf("failed to call UnaryEcho: %v", err)
	// }

	// if t, ok := header["timestamp"]; ok {

}
