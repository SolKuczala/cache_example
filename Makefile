all: build test

build:
	GOOS=linux GOARCH=amd64 go build -o cacheserver cmd/server/*.go
	GOOS=linux GOARCH=amd64 go build -o cacheclient cmd/client/*.go

build-server:
	GOOS=linux GOARCH=amd64 go build -o cacheserver cmd/server/*.go

build-client:
	GOOS=linux GOARCH=amd64 go build -o cacheclient cmd/client/*.go

test:
	go test -v ./...

run-server: build-server
	./cacheserver

run-client: build-client
	./cacheclient

clean:
	- go clean
	- rm cacheclient cacheserver
