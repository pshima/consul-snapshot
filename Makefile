all: build test

deps:
	go get github.com/aws/aws-sdk-go
	go get github.com/hashicorp/consul
	go get github.com/Pallinder/go-randomdata
	go get github.com/mitchellh/cli
	go get github.com/golang/lint/golint
	go get cloud.google.com/go/storage
	go get golang.org/x/net/context
	go get

updatedeps:
	go get -u -v github.com/aws/aws-sdk-go
	go get -u -v github.com/hashicorp/consul
	go get -u -v github.com/Pallinder/go-randomdata
	go get -u -v github.com/mitchellh/cli
	go get -u -v github.com/golang/lint/golint
	go get -u -v cloud.google.com/go/storage
	go get -u -v golang.org/x/net/context

build: deps
	go build

fmt:
	go fmt `go list ./... | grep -v vendor`

bootstrap:
	go get -u -v  github.com/golang/lint/golint
	go get -u -v github.com/mitchellh/gox

test: bootstrap
	go test `go list ./... | grep -v /vendor/`
	go vet `go list ./... | grep -v /vendor/`
	go list ./... |grep -v /vendor/ | xargs -L1 golint

build-all: test
	gox -arch="386 amd64 arm" -os="darwin linux windows" github.com/pshima/consul-snapshot

install:
	go install

cov:
	gocov test `go list ./... |grep -v /vendor/` | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html
