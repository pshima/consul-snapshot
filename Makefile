all: build test

deps:
	go get github.com/aws/aws-sdk-go
	go get github.com/hashicorp/consul
	go get github.com/Pallinder/go-randomdata
	go get github.com/mitchellh/cli
	go get github.com/golang/lint/golint
	go get golang.org/x/net/context
	go get google.golang.org/cloud/storage
	go get

updatedeps:
	go get -u -v github.com/aws/aws-sdk-go
	go get -u -v github.com/hashicorp/consul
	go get -u -v github.com/Pallinder/go-randomdata
	go get -u -v github.com/mitchellh/cli
	go get -u -v github.com/golang/lint/golint
	go get -u -v golang.org/x/net/context
	go get -u -v google.golang.org/cloud/storage

build: deps
	go build

fmt:
	go fmt `go list ./... | grep -v vendor`

test:
	go test ./...
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

cov:
	gocov test ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html
