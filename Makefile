.PHONY: run build tidy

run:
	go run ./cmd/main.go

build:
	go build -o bin/server ./cmd/main.go

tidy:
	go mod tidy

.DEFAULT_GOAL := run
