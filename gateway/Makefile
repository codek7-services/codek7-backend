dev-ini: # the dev profile initializes the development environment
	docker compose --profile dev up -d
dev: dev-ini
	go run cmd/main.go

app-ini: # the app profile initializes the application with all dependencies
	docker compose --profile app up -d --build

app: app-ini

proto:
	protoc \
		-I ../common \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		../common/pb/repo.proto


setup-video-chunks-topic:
	docker exec -it $(docker ps --format '{{.Names}}' | grep -i 'kafka') \
  /usr/bin/kafka-topics --bootstrap-server localhost:9092 \
  --create --topic video-chunks --partitions 3 --replication-factor 1

setup-kafka:
	setup-video-chunks-topic

.PHONY: dev dev-with-app setup-video-chunks-topic setup-kafka proto