VERSION=v1.1.0
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOLINT=golangci-lint run
BUILD_PLATFORM=linux/amd64
PACKAGE_PLATFORM=$(BUILD_PLATFORM)
VERSION_MAJOR=$(shell echo $(VERSION) | cut -f1 -d.)
VERSION_MINOR=$(shell echo $(VERSION) | cut -f2 -d.)
BINARY_NAME_ARCH=xcontest-arch-extractor
BINARY_NAME_RSS=xcontest-rss-extractor
GO_PACKAGE_ARCH=fahy.xyz/xcontest-arch-extractor
GO_PACKAGE_RSS=fahy.xyz/xcontest-rss-extractor
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
# Setup indexing
ES_CLUSTER_URL=http://localhost:9200
SETUP_PATH=docker/setup_indexing/Dockerfile
SETUP_IMG=setup_indexing

all: ensure package_arch_extractor package_rss_extractor

ensure:
	env GOOS=linux $(GOCMD) mod download

clean:
	$(GOCLEAN)

lint:
	$(GOLINT) ./...

build_setup_indexing:
	docker build -f $(SETUP_PATH) -t $(SETUP_IMG) .

run_setup_indexing: build_setup_indexing
	docker run --env "ES_CLUSTER_URL=$(ES_CLUSTER_URL)" --network="host" $(SETUP_IMG)

build_weekly_stats:
	docker build -f ./docker/stats/Dockerfile -t fahy.xyz/xcontest-weekly-stats:${VERSION_MAJOR} .

build_rss_extractor:
	env GOOS=linux CGO_ENABLED=0 $(GOCMD) mod download && \
	env GOOS=linux CGO_ENABLED=0 \
		$(GOBUILD) \
		-ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY}" \
			"-X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE}" \
			"-X github.com/sqooba/go-common/version.Version=${VERSION}" \
		-o $(BINARY_NAME_RSS) \
		./cmd/rssextractor/

package_rss_extractor:
	docker buildx build -f ./cmd/rssextractor/Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t ${GO_PACKAGE_RSS}:$(VERSION) \
		-t ${GO_PACKAGE_RSS}:$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t ${GO_PACKAGE_RSS}:$(VERSION_MAJOR) \
		--load \
		.

build_arch_extractor:
	env GOOS=linux CGO_ENABLED=0 $(GOCMD) mod download && \
	env GOOS=linux CGO_ENABLED=0 \
		$(GOBUILD) \
		-ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY}" \
			"-X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE}" \
			"-X github.com/sqooba/go-common/version.Version=${VERSION}" \
		-o $(BINARY_NAME_ARCH) \
		./cmd/archextractor/

package_arch_extractor:
	docker buildx build -f ./cmd/archextractor/Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t ${GO_PACKAGE_ARCH}:$(VERSION) \
		-t ${GO_PACKAGE_ARCH}:$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t ${GO_PACKAGE_ARCH}:$(VERSION_MAJOR) \
		--load \
		.

test:
	$(GOTEST) ./...
