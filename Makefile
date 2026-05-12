BINARY := forge
PKG    := ./cmd/forge
MODULE := github.com/sametkarademir/forge

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(MODULE)/internal/build.Version=$(VERSION) \
	-X $(MODULE)/internal/build.Commit=$(COMMIT) \
	-X $(MODULE)/internal/build.Date=$(DATE)

.PHONY: build install vet fmt smoke

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install:
	go install -ldflags "$(LDFLAGS)" $(PKG)

vet:
	go vet ./...

fmt:
	gofmt -l ./...

smoke:
	bash test/smoke/docker_create.sh
	bash test/smoke/docker_reset.sh
	bash test/smoke/docker_remove.sh
