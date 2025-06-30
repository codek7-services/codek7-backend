module github.com/lai0xn/codek-gateway

go 1.24.1

require (
	codek7/common v0.0.0
	github.com/go-chi/chi/v5 v5.2.2
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/redis/go-redis/v9 v9.11.0
	github.com/segmentio/kafka-go v0.4.48
	golang.org/x/crypto v0.36.0
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
)

replace codek7/common => ../common

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
