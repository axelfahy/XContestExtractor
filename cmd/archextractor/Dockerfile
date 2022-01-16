FROM --platform=$BUILDPLATFORM golang:alpine as builder

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT
ARG GIT_DIRTY
ARG VERSION
ARG BUILD_DATE

RUN apk add --no-cache ca-certificates

COPY . /src

WORKDIR /src

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go mod download && \
    env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o xcontest-arch-extractor \
    -ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE} \
              -X github.com/sqooba/go-common/version.Version=${VERSION}" \
    ./cmd/archextractor/main.go

FROM --platform=$BUILDPLATFORM chromedp/headless-shell:latest

RUN apt-get update; apt install dumb-init -y

ENTRYPOINT ["dumb-init", "--"]

# Copy the ca-certificate.crt from the build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/xcontest-arch-extractor /xcontest-arch-extractor

CMD ["/xcontest-arch-extractor"]
