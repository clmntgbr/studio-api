# Fetch / Media worker split — design

Nov 24, 2026

## Goal

Replace the monolithic `worker-ingest` with `worker-fetch` + `worker-media`, and adopt the `studio.*` RabbitMQ topology from `docs/docs.md`, while keeping **Clerk** auth and the **three audio tracks** (analysis / transcription / master).

## Decisions

| Topic | Choice |
| --- | --- |
| Binaries | `worker-fetch` (yt-dlp), `worker-media` (probe, audio×3, proxy, waveform) |
| Remove | `worker-ingest` (binary, compose, Dockerfile target) |
| Audio job | **One** `job.audio` on `q.audio` produces three MinIO keys (not single Opus) |
| Proxy | Separate `job.proxy` on `q.proxy` (prefetch 1) |
| Waveform | `job.waveform` → `peaks.bin` (replaces sprite in this path) |
| Auth | Clerk unchanged (no static API key work in this change) |
| Orchestration | Workers **do not** chain jobs. They publish results to `studio.results`. Core fan-out is out of scope; smoke tests publish jobs directly. |
| Envelope | New camelCase contract aligned with `docs/docs.md` (`messageId`, `type`, `version`, `correlationId`, …) |

## Topology

Exchanges (durable):

| Exchange | Type | Purpose |
| --- | --- | --- |
| `studio.jobs` | direct | Route `job.<type>` → `q.<type>` |
| `studio.results` | topic | `result.<type>.succeeded` / `result.<type>.failed` |
| `studio.progress` | fanout | Best-effort progress |
| `studio.retry` | direct | `retry.<type>` → `q.<type>.retry` |

Per job type `x` ∈ {fetch, probe, audio, proxy, waveform, …}:

- `q.x` (quorum when broker supports it; classic durable acceptable in local if quorum unavailable)
- `q.x.retry` (TTL + DLX back to `q.x`)
- `q.x.dlq`
- Binding: `studio.jobs` / `job.x` → `q.x`

Prefetch (env overridable):

| Queue | Prefetch |
| --- | --- |
| `q.fetch` | 2 |
| `q.probe`, `q.audio`, `q.waveform` | 4 |
| `q.proxy` | 1 |

## Contracts (payloads)

Common envelope:

```json
{
  "messageId": "…",
  "type": "job.fetch",
  "version": 1,
  "correlationId": "<mediaId>",
  "causationId": "<optional>",
  "occurredAt": "…",
  "attempt": 1,
  "payload": { }
}
```

| Job | Payload in | Result payload (keys only) |
| --- | --- | --- |
| `fetch` | `mediaId`, `url`, `maxSizeBytes?`, `maxDurationSec?` | `sourceKey`, `title?`, `sizeBytes`, `sha256` |
| `probe` | `mediaId`, `sourceKey` | probe fields (durationMs, width, height, fps, codecs, vfr, …) |
| `audio` | `mediaId`, `sourceKey` | `analysisKey`, `transcriptionKey`, `masterKey`, `sha256` (of analysis) |
| `proxy` | `mediaId`, `sourceKey`, optional height/gop/crf | `proxyKey` |
| `waveform` | `mediaId`, `audioKey` (analysis wav) | `peaksKey` |

Progress messages on `studio.progress` reuse the same envelope with `type: progress.<jobType>` and payload `{ pct, stage, message }`.

## Object keys (bucket `media`)

```text
media/{mediaId}/source/original.<ext>
media/{mediaId}/audio/analysis_16k_mono.wav
media/{mediaId}/audio/transcription.m4a
media/{mediaId}/audio/master.flac
media/{mediaId}/proxy/proxy_720p.mp4
media/{mediaId}/peaks/peaks.bin
```

## Package layout

```text
internal/studio/
  envelope/     // Message, Parse, schemas
  topology/     // DeclareExchanges, DeclareJobQueue, Publish helpers
internal/fetch/ // RunWorker on q.fetch (reuse ytdlp + ssrf from ingest)
internal/media/ // RunWorker multi-queue: probe, audio, proxy, waveform
```

Reusable FFmpeg/yt-dlp helpers move from `internal/ingest` into `internal/media/ffmpeg`, `internal/fetch/ytdlp` (or shared `internal/pipeline/…`). Dead `worker-ingest` and unused sprite path are removed from the fetch/media flow.

## Compose / Makefile

- Replace `worker-ingest` with `worker-fetch` + `worker-media`.
- Drop redundant `worker-audio` stub (audio owned by `worker-media`).
- Dockerfile: build `worker-fetch` / `worker-media`; image with ffmpeg+yt-dlp for fetch; ffmpeg for media (yt-dlp optional on media image).
- Makefile targets: `worker-fetch-logs`, `worker-media-logs`, `fetch-sample URL=…`.

## Non-goals

- Core orchestration (`media.fetched` → probe → …)
- Quorum/retry TTL tuning beyond declare + basic DLX
- Renaming all remaining stubs (`detect` → `silence`, etc.)
- Clerk → API key migration
- HTTP upload / from-url API endpoints
