COMPOSE_DEV := docker compose -f compose.dev.yaml

# Always-on app processes (excludes profile-only denoise/gpu workers)
APP_SERVICES := api worker \
	worker-ingest worker-audio worker-transcribe worker-scenes \
	worker-detect worker-analyze worker-reframe worker-render worker-translate

PIPELINE_WORKERS := worker-ingest worker-audio worker-transcribe worker-scenes \
	worker-detect worker-analyze worker-reframe worker-render worker-translate

# ============================================
# Development (compose.dev.yaml)
# ============================================

dev:
	$(COMPOSE_DEV) up -d

restart:
	$(COMPOSE_DEV) restart $(APP_SERVICES)

build:
	$(COMPOSE_DEV) up -d --build

dev-down:
	$(COMPOSE_DEV) down

dev-logs:
	$(COMPOSE_DEV) logs -f $(APP_SERVICES)

api-logs:
	$(COMPOSE_DEV) logs -f api

worker-logs:
	$(COMPOSE_DEV) logs -f worker

pipeline-logs:
	$(COMPOSE_DEV) logs -f $(PIPELINE_WORKERS)

worker-ingest-logs:
	$(COMPOSE_DEV) logs -f worker-ingest

worker-audio-logs:
	$(COMPOSE_DEV) logs -f worker-audio

worker-denoise-logs:
	$(COMPOSE_DEV) --profile denoise logs -f worker-denoise

worker-transcribe-logs:
	$(COMPOSE_DEV) logs -f worker-transcribe

worker-scenes-logs:
	$(COMPOSE_DEV) logs -f worker-scenes

worker-detect-logs:
	$(COMPOSE_DEV) logs -f worker-detect

worker-analyze-logs:
	$(COMPOSE_DEV) logs -f worker-analyze

worker-reframe-logs:
	$(COMPOSE_DEV) logs -f worker-reframe

worker-render-logs:
	$(COMPOSE_DEV) logs -f worker-render

worker-translate-logs:
	$(COMPOSE_DEV) logs -f worker-translate

dev-restart:
	$(COMPOSE_DEV) restart $(APP_SERVICES)

# Optional profile workers
dev-denoise:
	$(COMPOSE_DEV) --profile denoise up -d worker-denoise

lint:
	$(COMPOSE_DEV) exec api golangci-lint run --fix

# ============================================
# Tests (handler HTTP, via api container)
# ============================================

tests:
	$(COMPOSE_DEV) exec api go test ./internal/interfaces/http/handler/test/... -v -count=1

coverage:
	$(COMPOSE_DEV) exec api go test -coverprofile=coverage.out ./internal/interfaces/http/handler/test/... -coverpkg=./internal/interfaces/http/handler/...

coverage-html: coverage
	$(COMPOSE_DEV) exec api go tool cover -html=coverage.out -o coverage.html

unit-tests:
	$(COMPOSE_DEV) exec api go test ./internal/domain/... ./internal/application/... ./internal/infrastructure/... ./internal/workerstub/... ./cmd/... -race -count=1

# ============================================
# CLI Commands (via Docker)
# ============================================

cli-build:
	@$(COMPOSE_DEV) exec api go build -o bin/cli ./cmd/cli

migrate: cli-build
	@echo "Applying SQL migrations..."
	@$(COMPOSE_DEV) exec api ./bin/cli migrate up
	@echo "Checking model/DB schema..."
	@$(COMPOSE_DEV) exec api ./bin/cli migrate check

migrate-status: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli migrate status

migrate-down: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli migrate down

migrate-check: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli migrate check

shell:
	$(COMPOSE_DEV) exec api sh

# ============================================
# MinIO
# ============================================

minio-init:
	$(COMPOSE_DEV) up mc --force-recreate

minio-ls:
	@$(COMPOSE_DEV) exec minio sh -c 'mc alias set local http://127.0.0.1:9000 "$$MINIO_ROOT_USER" "$$MINIO_ROOT_PASSWORD" >/dev/null && mc ls local'
