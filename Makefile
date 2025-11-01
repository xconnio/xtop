run:
	go run ./cmd/xtop

build:
	CGO_ENABLED=0 go build ./cmd/xtop

lint:
	golangci-lint run

format:
	golangci-lint fmt
