.PHONY: build test lint fmt vet e2e run

build:
	go build -o bin/discord-mcp ./cmd/discord-mcp

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

e2e:
	scripts/e2e.sh

run:
	go run ./cmd/discord-mcp serve
