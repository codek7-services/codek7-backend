package main

import (
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"github.com/lumbrjx/codek7/repo/internal/handler"
	"github.com/lumbrjx/codek7/repo/internal/repository"
	"github.com/lumbrjx/codek7/repo/internal/service"
	"github.com/lumbrjx/codek7/repo/internal/storage"
	"github.com/lumbrjx/codek7/repo/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	minioEndpoint,
	minioAccessKey,
	minioSecretKey,
	minioBucket string
	minioUseSSL bool

	postgresDSN string
)

func init() {
	_ = godotenv.Load(".env")

	minioEndpoint = os.Getenv("MINIO_ENDPOINT")
	minioAccessKey = os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey = os.Getenv("MINIO_SECRET_KEY")
	minioBucket = os.Getenv("MINIO_BUCKET")
	minioUseSSL = os.Getenv("MINIO_USE_SSL") == "true"

	if minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" || minioBucket == "" {
		log.Fatal("MinIO environment variables are not set")
	}

	postgresDSN = os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		log.Fatal("POSTGRES_DSN environment variable is not set")
	}

	if minioUseSSL {
		log.Println("Using MinIO with SSL enabled")
	}

}

func main() {

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
	ur := repository.NewUserRepository(conn)

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
