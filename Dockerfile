FROM --platform=$BUILDPLATFORM golang:alpine as builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG BUILD_DATE

COPY . /src

WORKDIR /src

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go mod download && \
#export GIT_COMMIT=$(git rev-parse HEAD) && \
#export GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "+CHANGES" || true) && \
#export env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o xcontest-rss-extractor \
    ./cmd/

FROM --platform=$BUILDPLATFORM alpine

COPY --from=builder /src/xcontest-rss-extractor /xcontest-rss-extractor

#HEALTHCHECK --interval=900s --timeout=30s --retries=1 --start-period=30s CMD ["/xcontest-rss-extractor", "--health-check"]
ENTRYPOINT ["/xcontest-rss-extractor"]
