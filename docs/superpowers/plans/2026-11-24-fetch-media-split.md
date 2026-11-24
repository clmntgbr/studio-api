# Fetch / Media Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `worker-ingest` with `worker-fetch` + `worker-media` on `studio.*` RabbitMQ topology, keeping Clerk and three audio tracks in one `job.audio`.

**Architecture:** Shared `internal/studio/{envelope,topology}`; fetch consumes `q.fetch`; media consumes `q.probe|audio|proxy|waveform` in one process with per-queue prefetch; results go to `studio.results`; no worker-side job chaining.

**Tech Stack:** Go, amqp091-go, ffmpeg, yt-dlp, MinIO/S3, Docker Compose

**Spec:** `docs/superpowers/specs/2026-11-24-fetch-media-split-design.md`

## Global Constraints

- Never `Nack(requeue=true)`
- Channel AMQP ops serialized when concurrent
- JSON camelCase on wire envelopes
- Domain/app layers stay free of RabbitMQ adapters
- Clerk unchanged

---

### Task 1: Envelope + topology packages

**Files:**
- Create: `internal/studio/envelope/envelope.go`
- Create: `internal/studio/envelope/envelope_test.go`
- Create: `internal/studio/topology/topology.go`

- [ ] **Step 1:** Implement `envelope.Message` with camelCase JSON fields and helpers `NewJob`, `Parse`, `NewResult`, `NewProgress`.
- [ ] **Step 2:** Unit test round-trip marshal/parse for `job.fetch`.
- [ ] **Step 3:** Implement `topology.DeclareStudio(ch)` (exchanges) and `DeclareJobQueue(ch, jobType string)` (queue + retry + dlq + bindings).
- [ ] **Step 4:** Implement `Publisher` with confirm + mutex for `PublishResult` / `PublishProgress` / `PublishJob`.

### Task 2: Shared media/ffmpeg helpers

**Files:**
- Create: `internal/pipeline/ffprobe.go`, `ffmpeg.go`, `ssrf.go`, `ytdlp.go`, `hash.go`, `errors.go`, `storage` reuse from `internal/infrastructure/storage/s3`
- Move/adapt from `internal/ingest/*`
- Restore `MakeProxy`
- Add `MakeWaveformPeaks` → `peaks.bin` (100 Hz, interleaved min/max uint8)
- Keep three audio extractors

- [ ] **Step 1:** Move helpers out of `internal/ingest` into `internal/pipeline`.
- [ ] **Step 2:** Restore proxy + add waveform; keep SSRF tests green under new package path.
- [ ] **Step 3:** Delete obsolete `internal/ingest` once fetch/media compile.

### Task 3: worker-fetch

**Files:**
- Create: `internal/fetch/worker.go`
- Create: `cmd/worker-fetch/main.go`

- [ ] **Step 1:** Consume `q.fetch`, prefetch 2 (env `FETCH_CONCURRENCY`).
- [ ] **Step 2:** Validate URL (SSRF), yt-dlp download, upload `media/{id}/source/original.ext`, publish `result.fetch.succeeded|failed`, ack.
- [ ] **Step 3:** Wire `cmd/worker-fetch/main.go`.

### Task 4: worker-media

**Files:**
- Create: `internal/media/worker.go`
- Create: `cmd/worker-media/main.go`

- [ ] **Step 1:** One process, four consumers (or one channel + four Consume) for probe/audio/proxy/waveform with documented prefetch.
- [ ] **Step 2:** Each job downloads `sourceKey` (or `audioKey` for waveform), runs ffmpeg/ffprobe, uploads derived key(s), publishes result.
- [ ] **Step 3:** Wire main.

### Task 5: Compose / Docker / Makefile cleanup

**Files:**
- Modify: `compose.dev.yaml`, `compose.yaml`, `Dockerfile`, `Makefile`, `.env.dist`

- [ ] **Step 1:** Replace `worker-ingest` with `worker-fetch` + `worker-media`; remove `worker-audio` stub.
- [ ] **Step 2:** Dockerfile builds `worker-fetch` / `worker-media`; targets with ffmpeg (+ yt-dlp on fetch).
- [ ] **Step 3:** Makefile APP_SERVICES + log targets + `fetch-sample`.
- [ ] **Step 4:** Smoke: `go test ./internal/...` and recreate workers; publish sample fetch job.

### Task 6: Retire worker-ingest

- [ ] **Step 1:** Remove `cmd/worker-ingest`, ingest package leftovers, Dockerfile `worker-ingest` stage.
- [ ] **Step 2:** Confirm `go build ./cmd/worker-fetch ./cmd/worker-media` succeeds.
