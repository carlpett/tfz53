VERSION  = $(shell git describe --always --tags --dirty=-dirty)
REVISION = $(shell git rev-parse --short=8 HEAD)
BRANCH   = $(shell git rev-parse --abbrev-ref HEAD)

all: build test

build:
	@echo ">> building bzfttr53rdutil"
	@go build -ldflags "\
            -X main.Version=${VERSION} \
            -X main.Revision=${REVISION} \
            -X main.Branch=${BRANCH} \
            -X main.BuildUser=$(USER)@$(HOSTNAME) \
            -X main.BuildDate=$(shell date +%Y-%m-%dT%T%z)"

test:
	@go test -v -cover

release: build bin/github-release
	@echo ">> uploading release ${VERSION}"
	@./bin/github-release upload -t ${VERSION} -n bzfttr53rdutil -f bzfttr53rdutil

bin/github-release:
	@mkdir -p bin
	@curl -sL 'https://github.com/aktau/github-release/releases/download/v0.6.2/linux-amd64-github-release.tar.bz2' | tar xjf - --strip-components 3 -C bin

.PHONY: all build release
