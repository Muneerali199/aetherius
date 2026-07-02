.PHONY: proto build test lint dev clean

proto:
	@echo "Generating protobuf code..."
	@buf generate proto

build: proto
	@echo "Building all services..."
	go build ./services/auth/cmd/server
	go build ./services/user/cmd/server
	go build ./services/node/cmd/server
	go build ./services/scheduler/cmd/server
	go build ./services/deployment/cmd/server
	go build ./services/marketplace/cmd/server
	go build ./services/billing/cmd/server
	go build ./services/storage/cmd/server
	go build ./services/networking/cmd/server
	go build ./services/monitoring/cmd/server
	go build ./services/notification/cmd/server
	go build ./services/support/cmd/server
	go build ./services/ai/cmd/server
	go build ./agent/cmd/agent

test:
	go test ./...

lint:
	golangci-lint run ./...

dev:
	docker compose up --build -d

clean:
	rm -rf dist/
	rm -rf **/tmp/

migrate-up:
	@for dir in services/*/; do \
		service=$$(basename $$dir); \
		migrate -path services/$$service/migrations -database "postgres://aetherius:password@localhost:5432/aetherius_$$service?sslmode=disable" up; \
	done

migrate-down:
	@for dir in services/*/; do \
		service=$$(basename $$dir); \
		migrate -path services/$$service/migrations -database "postgres://aetherius:password@localhost:5432/aetherius_$$service?sslmode=disable" down 1; \
	done
