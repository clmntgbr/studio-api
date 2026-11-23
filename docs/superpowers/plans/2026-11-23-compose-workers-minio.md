# Compose workers + MinIO Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Kafka/uptime workers with RabbitMQ + MinIO + `worker-xxxx` stubs that consume and log, with uptime-style networks.

**Architecture:** Shared `internal/workerstub.Run(name)` dials AMQP, declares durable `q.<name>`, logs each delivery and acks. Thin `cmd/worker-*/main.go`. One Dockerfile builds all binaries (cutflow-style). Compose uses segmented networks and MinIO init for buckets `media|work|renders|assets`.

**Tech Stack:** Go, amqp091-go, Docker Compose, RabbitMQ, MinIO (coollabsio image), Postgres, Centrifugo.

## Global Constraints

- Worker cmds/services named `worker-xxxx` only (ingest, audio, denoise, transcribe, scenes, detect, analyze, reframe, render, translate, upscale, dub).
- Stubs: no DB, no S3 client — queue connect + log + ack only.
- No Kafka / worker-calls / worker-scheduler / worker-notify.
- Prod: `backend` and `data` networks `internal: true`.
- Buckets from features.md §10.1; work expires 3 days; assets versioning on.
- Module path remains `go-api`.
- Do not commit unless user asks.

---

### Task 1: workerstub package + cmd/worker-* stubs

**Files:**
- Create: `internal/workerstub/run.go`
- Create: `cmd/worker-ingest/main.go` (and siblings for all 12 names)
- Test: `go build ./cmd/worker-ingest ./cmd/worker-audio ...`

**Interfaces:**
- Produces: `func Run(ctx context.Context, name string) error` — queue `q.`+name, env `RABBITMQ_URL`

- [ ] **Step 1:** Implement `Run` — dial, declare queue, consume, log body, ack, exit on ctx cancel
- [ ] **Step 2:** Add thin mains calling `workerstub.Run(ctx, "ingest")` etc.
- [ ] **Step 3:** `go build` all stub cmds — expect success

### Task 2: Dockerfile

**Files:**
- Modify: `Dockerfile`

- [ ] **Step 1:** Remove scheduler/calls/notify builds; add builds for every `worker-*`; production (or runtime) stage copies all binaries like cutflow
- [ ] **Step 2:** Keep `development`, `api`, `worker`, `cli` usable from compose

### Task 3: compose.dev.yaml + compose.yaml + MinIO init

**Files:**
- Modify: `compose.dev.yaml`, `compose.yaml`
- Create: `deploy/minio/init.sh` (or inline mc entrypoint)
- Create: `caddy/Caddyfile` if missing and still referenced

- [ ] **Step 1:** Dev — rabbitmq, minio, mc, api, worker, all worker-*, networks edge/backend/data/egress
- [ ] **Step 2:** Prod — same services without host ports where appropriate; internal networks
- [ ] **Step 3:** `docker compose -f compose.dev.yaml config` validates

### Task 4: .env.dist S3 vars

**Files:**
- Modify: `.env.dist`

- [ ] **Step 1:** Add S3_ENDPOINT, S3_PUBLIC_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_REGION, bucket names
- [ ] **Step 2:** Fix DATABASE host if needed (`database` service name)

### Task 5: Verify + code review

- [ ] **Step 1:** `go build ./cmd/...`
- [ ] **Step 2:** Dispatch code-reviewer on the diff
