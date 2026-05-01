# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.25.5

FROM --platform=$BUILDPLATFORM node:22-alpine AS ui-builder
WORKDIR /src/ui
COPY ui/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY ui/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags='-s -w' -o /out/auction-server ./cmd/server

FROM alpine:3.21 AS runtime
WORKDIR /app
ENV PORT=8082
RUN apk add --no-cache ca-certificates curl tzdata && \
    addgroup -g 10001 -S app && adduser -u 10001 -S -G app app
COPY --from=go-builder --chown=10001:10001 /out/auction-server /app/auction-server
COPY --from=ui-builder --chown=10001:10001 /src/ui/dist /app/ui/dist
USER 10001:10001

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -fsS "http://127.0.0.1:${PORT:-8082}/health" || exit 1
EXPOSE 8082
ENTRYPOINT ["/app/auction-server"]
