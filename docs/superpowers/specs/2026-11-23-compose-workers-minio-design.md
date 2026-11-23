# Design: compose, networks, MinIO, stub workers

Date: 2026-11-23  
Status: approved direction (approach 2 + `worker-xxxx` naming)

## Goal

Align `compose.dev.yaml` / `compose.yaml` with `docs/features.md` and sibling projects (uptime networks, cutflow/isaiornot multi-binary Dockerfile, MinIO like uptime/isaiornot). Scaffold pipeline workers as stubs: connect to RabbitMQ, consume, log, ack — no business logic.

## Non-goals

- Full workerkit / contracts / ffmpeg / Python workers
- Migrating API to feature-complete jobs topology (`jobs` exchange, tiered retries)
- Frontend / Caddyfile rewrite beyond what compose needs

## Naming

| Compose service | Binary / `cmd/` | Queue consumed |
|---|---|---|
| `api` | `./api` · `cmd/api` | (HTTP + existing outbox path via `worker`) |
| `worker` | `./worker` · `cmd/worker` | existing domain outbox consumer (DB) |
| `worker-ingest` | `./worker-ingest` · `cmd/worker-ingest` | `q.ingest` |
| `worker-audio` | `./worker-audio` · `cmd/worker-audio` | `q.audio` |
| `worker-denoise` | `./worker-denoise` · `cmd/worker-denoise` | `q.denoise` (compose profile `denoise`) |
| `worker-transcribe` | `./worker-transcribe` · `cmd/worker-transcribe` | `q.transcribe` |
| `worker-scenes` | `./worker-scenes` · `cmd/worker-scenes` | `q.scenes` |
| `worker-detect` | `./worker-detect` · `cmd/worker-detect` | `q.detect` |
| `worker-analyze` | `./worker-analyze` · `cmd/worker-analyze` | `q.analyze` |
| `worker-reframe` | `./worker-reframe` · `cmd/worker-reframe` | `q.reframe` |
| `worker-render` | `./worker-render` · `cmd/worker-render` | `q.render` |
| `worker-translate` | `./worker-translate` · `cmd/worker-translate` | `q.translate` |
| `worker-upscale` | `./worker-upscale` · `cmd/worker-upscale` | `q.upscale` (profile `gpu`) |
| `worker-dub` | `./worker-dub` · `cmd/worker-dub` | `q.dub` (profile `gpu`) |

No Kafka. No `worker-calls` / `worker-scheduler` / `worker-notify`.

## Code shape (cutflow-style)

- Shared helper: `internal/workerstub` (or similar) — dial AMQP, declare simple durable queue `q.<name>`, consume (prefetch 1), log delivery, ack, graceful shutdown.
- Each `cmd/worker-<name>/main.go` is a thin `main` that calls `workerstub.Run("<name>")`.
- Stubs **do not** open Postgres or MinIO.
- Queue declare is minimal for local boot (not full §7 topology yet). Init/migrate-mq can come later.

## Dockerfile (shared, like cutflow)

One `Dockerfile`:

- `development` — Air / `go run` (compose commands point at `./cmd/worker-xxxx`)
- `builder` — build `api`, `worker`, `cli`, and every `worker-*`
- `production` (or per-binary targets if we keep alpine runtime stages) — copy all binaries; compose `command: ["./worker-ingest"]` etc.

Remove uptime-only builds (`worker-scheduler`, `worker-calls`, `worker-notify`).

## Networks (uptime-style)

| Network | Prod | Who |
|---|---|---|
| `edge` | public | caddy, api, centrifugo |
| `backend` | `internal: true` | rabbitmq, minio, api, worker, all `worker-*`, centrifugo |
| `data` | `internal: true` | postgres; only `api` + `worker` |
| `egress` | public (outbound) | stubs that will call external APIs later: ingest, transcribe, analyze, translate, scenes (+ optional denoise/reframe/render/upscale/dub) |

Dev: same membership; `backend`/`data` need not be internal so host ports work easily (mirror current uptime-dev looseness).

## Infra services

- **postgres** — `data` only  
- **rabbitmq** — `backend` (+ management ports in dev)  
- **minio** + **mc** (init) — `backend`  
  - Buckets: `media`, `work`, `renders`, `assets`  
  - Lifecycle: expire `work` after 3 days  
  - Versioning: enable on `assets`  
  - Image: same as uptime (`ghcr.io/coollabsio/minio:…`) for healthcheck/`mc` parity  
- **centrifugo** — `edge` + `backend`  
- **caddy** — `edge` (keep if present; create minimal Caddyfile only if missing and required)  
- **ngrok** (dev) — `edge`  
- Drop: kafka, kc, mailpit (unless still referenced — drop with notify worker), heartbeat network  

## Env (`.env.dist`)

Add S3 block aligned with features.md naming (or map to existing `STORAGE_*` if already used elsewhere — prefer `S3_ENDPOINT`, `S3_PUBLIC_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_REGION`, bucket names). Keep RabbitMQ vars; stubs use `RABBITMQ_URL` + hard-coded `q.<name>` (or `WORKER_QUEUE` override).

## Compose patterns

**Dev** (`compose.dev.yaml`):

```yaml
x-go-dev: &go-dev
  build: { context: ., dockerfile: Dockerfile, target: development }
  volumes: [.:/app, go-modules:/go/pkg/mod, go-build-cache:/root/.cache/go-build]
  restart: unless-stopped

x-pipeline-worker: &pipeline-worker
  <<: *go-dev
  depends_on:
    rabbitmq: { condition: service_healthy }
    mc: { condition: service_completed_successfully }  # minio ready + buckets
  networks: [backend]  # + egress where needed
```

Each stub: `command: ["go", "run", "./cmd/worker-ingest"]` etc.

**Prod** (`compose.yaml`): build shared image / run `./worker-xxxx`, networks as table above, `deploy.replicas` optional later.

## Verification

1. `docker compose -f compose.dev.yaml config` validates.  
2. `go build ./cmd/worker-ingest` (and siblings) compile.  
3. Bring up stack: rabbitmq healthy, minio buckets exist, each stub logs “consuming q.<name>”.  
4. Publish a test message to `q.ingest` → stub logs body and acks.

## Out of scope follow-ups

- Full RabbitMQ definitions from features §7  
- Python images for scenes/denoise/reframe  
- Presign / CORS MinIO wiring in API code  
- GPU compose override file  
