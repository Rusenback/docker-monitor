.PHONY: build run clean test

build:
	go build -o dockermon ./cmd/dockermon

run:
	go run ./cmd/dockermon


clean:
	rm -f dockermon

test:
	go test ./...

deps:
	go mod download
	go mod tidy
