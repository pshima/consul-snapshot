all: build test

deps:
	go get github.com/aws/aws-sdk-go
	go get github.com/hashicorp/consul
	go get github.com/Pallinder/go-randomdata
	go get github.com/mitchellh/cli
	go get github.com/golang/lint/golint
	go get

build: deps
	go build

test:
	go test -coverprofile=coverage.out; go tool cover -html=coverage.out -o coverage.html
	go vet ./...
	golint ./...

gox:
	go get github.com/mitchellh/gox
	gox -build-toolchain

build-all: test
	which gox || make gox
	gox -arch="386 amd64 arm" -os="darwin linux windows" github.com/pshima/consul-snapshot

install: consul-snapshot
	cp consul-snapshot /usr/local/bin/consul-snapshot
