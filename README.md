# Studio API

Backend Go for **Studio**: ingest a video, cut silences, propose viral clips, then **render** a master (+ optional vertical clips) with FFmpeg.

Product design lives in [docs/docs.md](docs/docs.md).

## Stack

| Layer | Tech |
| --- | --- |
| API | Go 1.25+, Fiber v3, GORM, Goose |
| DB | PostgreSQL |
| Jobs | RabbitMQ (`studio.jobs` / `q.*`) |
| Storage | MinIO (S3) |
| Realtime | Centrifugo |
| Auth | Clerk |
| Encode | FFmpeg + libass (DejaVu fonts in worker image) |

## Workers

| Container | Queue | Role |
| --- | --- | --- |
| `worker-fetch` | `q.fetch` | yt-dlp |
| `worker-media` | probe / audio / proxy / waveform | Media prep |
| `worker-silence` | `q.silence` | Silence detection |
| `worker-transcribe` | `q.transcribe` | AssemblyAI (or mock) |
| `worker-text` | `q.text` | Fillers / false starts |
| `worker-llm` | `q.llm` | Sponsors, chapters, viral clips |
| `worker-scenes` | scenes | Ad-ish cuts (Python) |
| `worker-reframe` | `q.reframe` | Face / speaker crop path |
| **`worker-render`** | `q.render` | Master / clip encode (**serial**, prefetch 1, ffmpeg `-threads 1`) |

Plus `api` (HTTP) and `worker` (outbox / domain events / results).

## Getting started

```bash
cp .env.dist .env
# Clerk, S3, RabbitMQ, optional ASSEMBLYAI_API_KEY / LLM_* 

make dev
make migrate
```

| Service | Port |
| --- | --- |
| API | `4000` |
| Postgres | `9543` (compose mapping may vary) |
| RabbitMQ UI | `15672` |
| MinIO | `9000` / console `9003` |
| Centrifugo | `8000` |

Dev mocks (default): `TRANSCRIBE_PROVIDER=mock`, `LLM_PROVIDER=mock` → `testdata/mocks/external/`.

## Render (current behaviour)

### Global render (editor)

1. User picks **format** + optional **burn SRT** + **master split** (persisted on `media_configuration`).
2. `POST /api/media/:id/renders` with `force: true` → **master** job.
3. For each selected clip → `POST /api/clips/:id/render` (queued **one after another** from the app).
4. `worker-render` runs **one job at a time** (QoS 1).

### Media configuration (persisted prefs)

| Column | Values | Effect |
| --- | --- | --- |
| `output_preset_id` | UUID | Last render format |
| `burn_subtitles` | bool | Burn ASS captions into master + clips when rendering |
| `master_split` | `""` / `off` / `parts:N` / `durationMs:N` | After master encode, split into parts for TikTok-style upload |

- `parts:5` → 5 equal parts (N ∈ 2…50).
- `durationMs:600000` → parts of at most 10 minutes.
- Full master stays at `storage_key`; parts in `render.output_parts`.
- Download: `GET /api/renders/:id/download` and `?part=1`…`N`.

### Clips

- LLM viral band: ~**25–90 s** (prefer ~45 s); mock expands fixture windows.
- Clip encode uses a **continuous** source window (no silence KEEP cuts).
- Hook overlay via `drawtext` + `textfile=` (avoids FR apostrophe breakage) and DejaVu `fontfile=`.

### Fingerprint / force

- Same inputs reuse a queued/running/completed render unless `force: true` (Render always forces a new encode).

## Useful Makefile targets

| Command | Description |
| --- | --- |
| `make dev` | Compose stack |
| `make migrate` | Goose + schema drift check |
| `make tests` | Handler tests in API container |
| `make lint` | golangci-lint |
| `make shell` | Shell into API container |

## Key HTTP surfaces

- Media: upload / from-URL, configuration (`PATCH …/configuration`), reanalyze, playback
- Timeline: get / preview / validate / rebuild / exports / subtitles
- Clips: list / create / patch (incl. `selected`) / analyze / render
- Renders: create / list / cancel / **download** (streamed proxy, no MinIO signature for the browser)
- Output presets: list

Realtime: Centrifugo project channel (`render.progress` / completed / failed, timeline updates).

## Architecture sketch

```
cmd/api | worker | worker-* 
internal/
  domain/           aggregates + ports
  application/      command | query | event
  renderplan/       job payload for worker-render
  mastersplit/      parts:N | durationMs:N
  renderworker/     FFmpeg encode + optional master split
  llm/              OpenAI-compatible + file/heuristic mock
  infrastructure/   GORM, S3, RabbitMQ, Centrifugo
  interfaces/http/  Fiber handlers
migrations/         Goose SQL
```

Conventions: [`.cursor/rules/`](.cursor/rules/) when present; deep product notes in [docs/docs.md](docs/docs.md).
