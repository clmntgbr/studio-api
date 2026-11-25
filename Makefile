COMPOSE_DEV := docker compose -f compose.dev.yaml

# Always-on app processes (excludes profile-only denoise/gpu workers)
APP_SERVICES := api worker \
	worker-fetch worker-media worker-silence worker-transcribe worker-text worker-llm worker-scenes \
	worker-analyze worker-reframe worker-render worker-translate

PIPELINE_WORKERS := worker-fetch worker-media worker-silence worker-transcribe worker-text worker-llm worker-scenes \
	worker-analyze worker-reframe worker-render worker-translate

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

worker-fetch-logs:
	$(COMPOSE_DEV) logs -f worker-fetch

worker-media-logs:
	$(COMPOSE_DEV) logs -f worker-media

worker-denoise-logs:
	$(COMPOSE_DEV) --profile denoise logs -f worker-denoise

worker-transcribe-logs:
	$(COMPOSE_DEV) logs -f worker-transcribe

worker-text-logs:
	$(COMPOSE_DEV) logs -f worker-text

worker-llm-logs:
	$(COMPOSE_DEV) logs -f worker-llm

worker-scenes-logs:
	$(COMPOSE_DEV) logs -f worker-scenes

worker-silence-logs:
	$(COMPOSE_DEV) logs -f worker-silence

worker-detect-logs:
	$(COMPOSE_DEV) --profile legacy logs -f worker-detect

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

# Publish a sample job.fetch to studio.jobs (requires rabbitmqadmin / curl + python).
# Usage: make fetch-sample URL='https://www.youtube.com/watch?v=...'
fetch-sample:
	@test -n "$(URL)" || (echo "URL is required"; exit 1)
	@MEDIA_ID=$${MEDIA_ID:-$$(uuidgen | tr '[:upper:]' '[:lower:]')}; \
	BODY=$$(python3 -c "import json,uuid,datetime; media='$$MEDIA_ID'; url='$(URL)'; print(json.dumps({'messageId':str(uuid.uuid4()),'type':'job.fetch','version':1,'correlationId':media,'occurredAt':datetime.datetime.utcnow().isoformat()+'Z','attempt':1,'payload':{'mediaId':media,'url':url}}))"); \
	docker exec broker rabbitmqadmin -u user -p password publish exchange=studio.jobs routing_key=job.fetch payload="$$BODY"; \
	echo "published job.fetch media_id=$$MEDIA_ID"

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
	$(COMPOSE_DEV) exec api go test ./internal/domain/... ./internal/application/... ./internal/infrastructure/... ./internal/pipeline/... ./internal/studio/... ./internal/fetch/... ./internal/media/... ./internal/workerstub/... ./cmd/... -race -count=1

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

# Empty project/media/pipeline tables; keeps users, goose_db_version, output_preset.
# Usage: make purge
purge: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli purge --force

# Retention: delete derived media objects older than N days (default 7).
# Usage: make purge-retention  OR  make purge-retention DAYS=14 DRY_RUN=1
purge-retention: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli purge retention \
		--days=$${DAYS:-7} \
		$$( [ "$${DRY_RUN}" = "1" ] && echo --dry-run || true )

# Empty media + renders S3 buckets (local MinIO). Refuses without --force via CLI.
# Usage: make purge-s3
purge-s3: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli purge s3 --force

shell:
	$(COMPOSE_DEV) exec api sh
