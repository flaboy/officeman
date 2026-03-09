FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /src

ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
  go build -ldflags="-s -w -X github.com/github-flaboy/officeman/internal/buildinfo.Version=${VERSION}" \
  -o /out/officeman ./cmd/officeman

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/officeman /usr/local/bin/officeman

EXPOSE 7012

ENTRYPOINT ["officeman"]
