VERSION=v1.0.0
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOLINT=golangci-lint run
BUILD_PLATFORM=linux/amd64
PACKAGE_PLATFORM=$(BUILD_PLATFORM)
VERSION_MAJOR=$(shell echo $(VERSION) | cut -f1 -d.)
VERSION_MINOR=$(shell echo $(VERSION) | cut -f2 -d.)
BINARY_NAME=xcontest-rss-extractor
GO_PACKAGE=fahy.xyz/xcontest-rss-extractor
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
# Setup indexing
ES_CLUSTER_URL=http://localhost:9200
SETUP_PATH=dev/docker/setup_indexing/Dockerfile
SETUP_IMG=setup_indexing

all: ensure build package

build-setup:
	docker build -f $(SETUP_PATH) -t $(SETUP_IMG) .
	docker run --env "ES_CLUSTER_URL=$(ES_CLUSTER_URL)" --network="host" $(SETUP_IMG)

ensure:
	env GOOS=linux $(GOCMD) mod download

clean:
	$(GOCLEAN)

lint:
	$(GOLINT) ./cmd/*.go

build:
	env GOOS=linux CGO_ENABLED=0 $(GOCMD) mod download && \
	env GOOS=linux CGO_ENABLED=0 \
		$(GOBUILD) \
		-o $(BINARY_NAME) \
		./cmd/

package: build
	docker buildx build -f Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t ${GO_PACKAGE}:$(VERSION) \
		-t ${GO_PACKAGE}:$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t ${GO_PACKAGE}:$(VERSION_MAJOR) \
		--load \
		.

test:
	$(GOTEST) ./...

