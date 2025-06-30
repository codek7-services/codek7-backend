.PHONY: dev dev-with-app setup-video-chunks-topic setup-kafka

dev:
	docker compose up -d

dev-with-app:
	docker compose --profile app up -d --build

setup-video-chunks-topic:
	docker exec -it $(docker ps --format '{{.Names}}' | grep -i 'kafka') \
  /usr/bin/kafka-topics --bootstrap-server localhost:9092 \
  --create --topic video-chunks --partitions 3 --replication-factor 1

setup-kafka:
	setup-video-chunks-topic

proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/pb/repo.proto