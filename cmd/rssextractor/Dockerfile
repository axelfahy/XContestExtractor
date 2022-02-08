FROM --platform=$BUILDPLATFORM golang:alpine as builder

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT
ARG GIT_DIRTY
ARG VERSION
ARG BUILD_DATE

COPY . /src

WORKDIR /src

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go mod download && \
    env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o xcontest-rss-extractor \
    -ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} \
			  -X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE} \
              -X github.com/sqooba/go-common/version.Version=${VERSION}" \
    ./cmd/rssextractor/main.go

FROM --platform=$BUILDPLATFORM alpine

COPY --from=builder /src/xcontest-rss-extractor /xcontest-rss-extractor

#HEALTHCHECK --interval=900s --timeout=30s --retries=1 --start-period=30s CMD ["/xcontest-arch-extractor", "--health-check"]
ENTRYPOINT ["/xcontest-rss-extractor"]
