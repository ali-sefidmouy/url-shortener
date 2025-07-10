package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"url-shortener/proto"

	"url-shortener/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type urlShortenerServer struct {
	proto.UnimplementedURLShortenerServer
}

// gRPC Shorten method
func (s *urlShortenerServer) Shorten(ctx context.Context, req *proto.ShortenRequest) (*proto.ShortenResponse, error) {
	original := req.GetUrl()
	if original == "" {
		return nil, status.Error(codes.InvalidArgument, "URL cannot be empty")
	}

	var existingCode string
	err := DB.QueryRow("SELECT short_code FROM urls WHERE original_url = $1", original).Scan(&existingCode)
	if err == nil {
		return &proto.ShortenResponse{ShortUrl: "http://localhost:8080/" + existingCode}, nil
	} else if err != sql.ErrNoRows {
		return nil, status.Error(codes.Internal, "database error")
	}

	var short string
	for {
		short = generateShortURL()
		err = DB.QueryRow("SELECT short_code FROM urls WHERE short_code = $1", short).Scan(&existingCode)
		if err == sql.ErrNoRows {
			break // is unique, so break
		}
	}

	_, err = DB.Exec("INSERT INTO urls (original_url, short_code) VALUES ($1, $2)", original, short)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not insert URL")
	}

	return &proto.ShortenResponse{ShortUrl: "http://localhost:8080/" + short}, nil
}

// gRPC Resolve method
func (s *urlShortenerServer) Resolve(ctx context.Context, req *proto.ResolveRequest) (*proto.ResolveResponse, error) {
	code := req.GetShortCode()
	if code == "" {
		return nil, status.Error(codes.InvalidArgument, "short code cannot be empty")
	}

	var original string
	err := DB.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", code).Scan(&original)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "short code not found")
	} else if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	return &proto.ResolveResponse{OriginalUrl: original}, nil
}

func startGRPCServer() {
	config.LoadEnvOrFail()

	lis, err := net.Listen("tcp", ":" + os.Getenv("GRPC_SERVER_PORT"))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterURLShortenerServer(grpcServer, &urlShortenerServer{})

	log.Println("gRPC server listening on :", os.Getenv("GRPC_SERVER_PORT"))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve gRPC: %v", err)
	}
}
