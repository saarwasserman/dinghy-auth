# User and Authentication

Requirements:
    1. postgres
    2. redis


# GRPC commands

protoc --go_out=grpcgen/{proto_entity} --go_opt=paths=source_relative --go-grpc_out=/{proto_entity}--go-grpc_opt=paths=source_relative ../proto/{proto_entity}.proto

 note: proto is a sibling dir/repo in the file system

## import usage:
import (
    users "saarwasserman.com/auth/grpcgen/users/proto"
)