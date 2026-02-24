# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM node:22-alpine AS ui-builder
WORKDIR /src/ui
COPY ui/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY ui/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS go-builder
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
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=go-builder /out/auction-server /app/auction-server
COPY --from=ui-builder /src/ui/dist /app/ui/dist
RUN chown -R app:app /app
USER app

EXPOSE 8082
ENTRYPOINT ["/app/auction-server"]
