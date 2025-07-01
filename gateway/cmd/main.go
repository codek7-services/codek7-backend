package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lai0xn/codek-gateway/internal/infra"
	"github.com/lai0xn/codek-gateway/internal/server"
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

}
func main() {
	godotenv.Load()
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

	err := infra.RDBConnect()
	if err != nil {
		panic(err)
	}
	err = infra.RMQConnect()
	if err != nil {
		panic(err)
	}
	err = infra.New(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioUseSSL)
	if err != nil {
		panic(err)
	}
	s := server.NewServer(os.Getenv("PORT"))
	s.Start()
}
