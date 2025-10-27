run:
	go run ./cmd/xtop

build:
	go build ./cmd/xtop

lint:
	golangci-lint run

format:
	golangci-lint fmt
