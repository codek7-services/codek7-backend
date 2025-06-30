package main

import (
	"log"
	"net"

	"github.com/lumbrjx/codek7/repo/internal/handler"
	"github.com/lumbrjx/codek7/repo/internal/repository"
	"github.com/lumbrjx/codek7/repo/internal/service"
	"github.com/lumbrjx/codek7/repo/internal/storage"
	"github.com/lumbrjx/codek7/repo/pkg/pb/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// === Config ===
	minioEndpoint := "localhost:9000"
	minioAccessKey := "minioadmin"
	minioSecretKey := "minioadmin"
	minioBucket := "videos"
	minioUseSSL := false

	postgresDSN := "postgres://user:password@localhost:5432/codek7?sslmode=disable"

	// === MinIO ===
	minioClient, err := storage.New(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioUseSSL)
	if err != nil {
		log.Fatalf("Failed to init MinIO: %v", err)
	}

	// === PostgreSQL ===
	conn, err := repository.NewPostgresPool(postgresDSN)
	if err != nil {
		log.Fatalf("Failed to init Postgres: %v", err)
	}

	// === Repositories ===

	// video repository
	vr := repository.NewVideoRepository(conn)
	if err != nil {
		log.Fatalf("Failed to init Postgres: %v", err)
	}

	ur := repository.NewUserRepository(conn)
	if err != nil {
		log.Fatalf("Failed to init Postgres: %v", err)
	}

	// === Services ===
	videoService := service.NewVideoService(vr, minioClient)
	userService := service.NewUserService(ur)

	// === Handler ===
	repoHandler := handler.NewRepoHandler(userService, videoService)

	// === gRPC Server ===
	grpcServer := grpc.NewServer()
	pb.RegisterRepoServiceServer(grpcServer, repoHandler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
