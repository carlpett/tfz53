CROSSBUILD_OS = linux windows darwin
CROSSBUILD_ARCH = 386 amd64

VERSION  = $(shell git describe --always --tags --dirty=-dirty)
REVISION = $(shell git rev-parse --short=8 HEAD)
BRANCH   = $(shell git rev-parse --abbrev-ref HEAD)

BUILDUSER ?= $(USER)
BUILDHOST ?= $(HOSTNAME)
LDFLAGS    = -X main.Version=${VERSION} \
             -X main.Revision=${REVISION} \
             -X main.Branch=${BRANCH} \
             -X main.BuildUser=$(BUILDUSER)@$(BUILDHOST) \
             -X main.BuildDate=$(shell date +%Y-%m-%dT%T%z)

all: build test

build:
	@echo ">> building"
	@go build -ldflags "$(LDFLAGS)"

crossbuild:
	@echo ">> cross-compiling"
	@gox -arch="$(CROSSBUILD_ARCH)" -os="$(CROSSBUILD_OS)" -ldflags="$(LDFLAGS)" -output="binaries/tfz53_{{.OS}}_{{.Arch}}"

test:
	@echo ">> testing"
	@go test -v -cover

release: bin/github-release
	@echo ">> uploading release ${VERSION}"
	@for bin in binaries/*; do \
		./bin/github-release upload -t ${VERSION} -n $$(basename $${bin}) -f $${bin}; \
	done

bin/github-release:
	@mkdir -p bin
	@curl -sL 'https://github.com/aktau/github-release/releases/download/v0.6.2/linux-amd64-github-release.tar.bz2' | tar xjf - --strip-components 3 -C bin

.PHONY: all build release
