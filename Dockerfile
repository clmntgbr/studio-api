# ============================================
# Base stage - Common dependencies
# ============================================
FROM golang:1.25-alpine AS base

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .


# ============================================
# Development stage - With Air hot reload
# ============================================
FROM base AS development

RUN apk add --no-cache ffmpeg curl ca-certificates python3 \
    && curl -fsSL -o /usr/local/bin/yt-dlp \
      https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp

RUN go install github.com/air-verse/air@v1.67.1
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
ENV PATH="/go/bin:${PATH}"

EXPOSE 3000

CMD ["air", "-c", ".air.toml"]


# ============================================
# Builder stage - Build binaries
# ============================================
FROM base AS builder

RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o api \
    ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o worker \
    ./cmd/worker

RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o cli \
    ./cmd/cli

RUN set -eu; \
    for name in ingest audio denoise transcribe scenes detect analyze reframe render translate; do \
      CGO_ENABLED=0 GOOS=linux go build \
        -a -installsuffix cgo \
        -ldflags="-w -s" \
        -o "worker-${name}" \
        "./cmd/worker-${name}"; \
    done


# ============================================
# Runtime base - shared by every production image
# ============================================
FROM alpine:latest AS runtime

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /home/appuser
USER appuser


# ============================================
# Production: all binaries in one image (cutflow-style).
# Compose selects the process via command: ["./worker-ingest"], etc.
# ============================================
FROM runtime AS production

COPY --from=builder --chown=appuser:appuser /app/api .
COPY --from=builder --chown=appuser:appuser /app/worker .
COPY --from=builder --chown=appuser:appuser /app/cli .
COPY --from=builder --chown=appuser:appuser /app/worker-ingest .
COPY --from=builder --chown=appuser:appuser /app/worker-audio .
COPY --from=builder --chown=appuser:appuser /app/worker-denoise .
COPY --from=builder --chown=appuser:appuser /app/worker-transcribe .
COPY --from=builder --chown=appuser:appuser /app/worker-scenes .
COPY --from=builder --chown=appuser:appuser /app/worker-detect .
COPY --from=builder --chown=appuser:appuser /app/worker-analyze .
COPY --from=builder --chown=appuser:appuser /app/worker-reframe .
COPY --from=builder --chown=appuser:appuser /app/worker-render .
COPY --from=builder --chown=appuser:appuser /app/worker-translate .

EXPOSE 3000
CMD ["./api"]

# Ingest needs ffmpeg + yt-dlp (Python zipapp). Keep them off the API image.
FROM production AS worker-ingest
USER root
RUN apk --no-cache add ffmpeg curl ca-certificates python3 \
    && curl -fsSL -o /usr/local/bin/yt-dlp \
      https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp
USER appuser
CMD ["./worker-ingest"]


FROM production AS api
CMD ["./api"]

FROM production AS worker
CMD ["./worker"]

FROM production AS cli
CMD ["./cli"]
