package main

import (
	"net"
	"os"

	"codek7/common/pb"

	"github.com/joho/godotenv"
	"github.com/lumbrjx/codek7/repo/internal/handler"
	"github.com/lumbrjx/codek7/repo/internal/repository"
	"github.com/lumbrjx/codek7/repo/internal/service"
	"github.com/lumbrjx/codek7/repo/internal/storage"
	"github.com/lumbrjx/codek7/repo/pkg/logger"

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

	logger.Logger.Info("Loading environment configuration")

	minioEndpoint = os.Getenv("MINIO_ENDPOINT")
	minioAccessKey = os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey = os.Getenv("MINIO_SECRET_KEY")
	minioBucket = os.Getenv("MINIO_BUCKET")
	minioUseSSL = os.Getenv("MINIO_USE_SSL") == "true"

	if minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" || minioBucket == "" {
		logger.Logger.Error("MinIO environment variables are not set",
			"minio_endpoint", minioEndpoint,
			"minio_access_key", minioAccessKey != "",
			"minio_secret_key", minioSecretKey != "",
			"minio_bucket", minioBucket,
		)
		os.Exit(1)
	}

	postgresDSN = os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		logger.Logger.Error("POSTGRES_DSN environment variable is not set")
		os.Exit(1)
	}

	logger.Logger.Info("Environment configuration loaded successfully",
		"minio_endpoint", minioEndpoint,
		"minio_bucket", minioBucket,
		"minio_use_ssl", minioUseSSL,
		"postgres_dsn_set", postgresDSN != "",
	)
}

func main() {
	logger.Logger.Info("Starting repo service")

	// === MinIO ===
	logger.Logger.Info("Initializing MinIO client")
	minioClient, err := storage.New(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioUseSSL)
	if err != nil {
		logger.Logger.Error("Failed to initialize MinIO",
			"error", err.Error(),
			"endpoint", minioEndpoint,
			"bucket", minioBucket,
		)
		os.Exit(1)
	}
	logger.Logger.Info("MinIO client initialized successfully")

	// === PostgreSQL ===
	logger.Logger.Info("Initializing PostgreSQL connection pool")
	conn, err := repository.NewPostgresPool(postgresDSN)
	if err != nil {
		logger.Logger.Error("Failed to initialize PostgreSQL",
			"error", err.Error(),
		)
		os.Exit(1)
	}
	logger.Logger.Info("PostgreSQL connection pool initialized successfully")

	// === Repositories ===
	logger.Logger.Info("Initializing repositories")
	vr := repository.NewVideoRepository(conn)
	ur := repository.NewUserRepository(conn)

	// === Services ===
	logger.Logger.Info("Initializing services")
	videoService := service.NewVideoService(vr, minioClient)
	userService := service.NewUserService(ur)

	// === Handler ===
	logger.Logger.Info("Initializing gRPC handler")
	repoHandler := handler.NewRepoHandler(userService, videoService)

	// === gRPC Server ===
	logger.Logger.Info("Initializing gRPC server")
	grpcServer := grpc.NewServer()
	pb.RegisterRepoServiceServer(grpcServer, repoHandler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Logger.Error("Failed to listen on port 50051",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	logger.Logger.Info("gRPC server listening", "port", 50051)
	if err := grpcServer.Serve(lis); err != nil {
		logger.Logger.Error("Failed to serve gRPC",
			"error", err.Error(),
		)
		os.Exit(1)
	}
}
