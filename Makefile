.PHONY: build run dev test lint tidy docker-up docker-down

BINARY := documind
CMD     := ./cmd/server

build:
	go build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

dev:
	@which air > /dev/null 2>&1 || go install github.com/air-verse/air@latest
	air

test:
	go test ./... -v -count=1

lint:
	@which golangci-lint > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

tidy:
	go mod tidy

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down
