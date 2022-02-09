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
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
# Image names
PACKAGE_SETUP_INDEXING=fahy.xyz/setup-indexing
GO_PACKAGE_ARCH=fahy.xyz/xcontest-arch-extractor
GO_PACKAGE_RSS=fahy.xyz/xcontest-rss-extractor
PACKAGE_STATS_WEEKLY=fahy.xyz/xcontest-weekly-stats
# App settings
ES_CLUSTER_URL=http://localhost:9200

all: ensure package_arch_extractor package_rss_extractor

ensure:
	env GOOS=linux $(GOCMD) mod download

clean:
	$(GOCLEAN)

lint:
	$(GOLINT) ./...

build_setup_indexing:
	docker build -f docker/setup_indexing/Dockerfile \
		-t $(PACKAGE_SETUP_INDEXING):$(VERSION) \
		-t $(PACKAGE_SETUP_INDEXING):$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(PACKAGE_SETUP_INDEXING):$(VERSION_MAJOR) \
		.

run_setup_indexing: build_setup_indexing
	docker run --env "ES_CLUSTER_URL=$(ES_CLUSTER_URL)" --network="host" $(PACKAGE_SETUP_INDEXING):$(VERSION_MAJOR)

build_weekly_stats:
	docker build -f ./docker/stats/Dockerfile \
		-t $(PACKAGE_STATS_WEEKLY):$(VERSION) \
		-t $(PACKAGE_STATS_WEEKLY):$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(PACKAGE_STATS_WEEKLY):$(VERSION_MAJOR) \
		.

package_rss_extractor:
	docker buildx build -f ./cmd/rssextractor/Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_PACKAGE_RSS):$(VERSION) \
		-t $(GO_PACKAGE_RSS):$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(GO_PACKAGE_RSS):$(VERSION_MAJOR) \
		--load \
		.

package_arch_extractor:
	docker buildx build -f ./cmd/archextractor/Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_PACKAGE_ARCH):$(VERSION) \
		-t $(GO_PACKAGE_ARCH):$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(GO_PACKAGE_ARCH):$(VERSION_MAJOR) \
		--load \
		.

test:
	$(GOTEST) ./...
