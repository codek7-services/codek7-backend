.PHONY: dev docker-prod docker-stop docker-dev docker-dev-stop rebuild logs proto test fmt lint clean

dev:
	@echo " > Running app locally with Docker-based Postgres + MinIO...  "
	docker compose --profile dev up -d
	go run cmd/main.go

docker-prod:
	@echo " > Running full stack in Docker (app + postgres + minio)...  "
	docker compose --profile app up -d --build

docker-stop:
	@echo " > Stopping only the Docker app (keeping postgres/minio running)...  "
	docker compose stop app

docker-dev:
	@echo " > Starting only Docker dev dependencies (postgres + minio)...  "
	docker compose --profile dev up -d

docker-dev-stop:
	@echo " > Stopping Docker dev dependencies (postgres + minio)...  "
	docker compose stop minio postgres

logs:
	@docker logs -f repo-service

rebuild:
	docker compose --profile app down
	docker compose --profile app up -d --build

proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/pb/repo.proto

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

clean:
	docker compose down -v
