GO_VERSION=1.14.7
GOFILES = $(shell find . -type f -name '*.go' -not -path "./.git/*")
LDFLAGS = '-s -w -extldflags "-static" -X github.com/gimlet-io/gimlets/version.Version='${VERSION}

DOCKER_RUN?=
_with-docker:
	$(eval DOCKER_RUN=docker run --rm -v $(shell pwd):/go/src/github.com/gimlet-io/gimletd -w /go/src/github.com/gimlet-io/gimletd golang:$(GO_VERSION))

.PHONY: all format test build dist

all: test build

format:
	@gofmt -w ${GOFILES}

test:
	$(DOCKER_RUN) go test -race -timeout 60s $(shell go list ./... )

test-with-postgres:
	docker run --rm -e POSTGRES_PASSWORD=mysecretpassword -p 5432:5432 -d postgres

	export DATABASE_DRIVER=postgres
	export DATABASE_CONFIG=postgres://postgres:mysecretpassword@127.0.0.1:5432/postgres?sslmode=disable
	go test -timeout 60s github.com/gimlet-io/gimletd/store/...

build:
	$(DOCKER_RUN) go build -ldflags $(LDFLAGS) -o build/gimlet github.com/gimlet-io/gimletd/cmd

dist:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -a -installsuffix cgo -o bin/gimletd-linux-x86_64 github.com/gimlet-io/gimletd/cmd

start-local-env:
	docker-compose -f fixtures/k3s/docker-compose.yml up -d

stop-local-env:
	docker-compose -f fixtures/k3s/docker-compose.yml stop

clean-local-env:
	docker-compose -f fixtures/k3s/docker-compose.yml down
	docker volume rm k3s_k3s-gimlet
