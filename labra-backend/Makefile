SERVER_PATH = "src"
BIN_PATH = "bin"
SERVER_NAME = "Labra-server"

generate:
	sqlc generate

run:
	cd cmd && go run .

lint:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

.PHONY: generate run lint test test-race
