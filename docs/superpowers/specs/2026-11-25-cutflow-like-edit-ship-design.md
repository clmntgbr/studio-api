# Cutflow-like edit → single timeline → Ship

**Date:** 2026-11-25  
**Status:** approved  
**Scope:** studio-api + studio-app

## Goal

Align Studio with a Cutflow-like edit loop:

1. Before Ship: analysis + editable timeline only (no master encode, no scenes, no early reframe).
2. One timeline per project/media — no draft/validated/superseded version stack.
3. Ship: apply the front-chosen output format (reframe as needed) then encode the final video once.

## Current vs target

| Area | Today | Target |
|------|--------|--------|
| Timeline | Many rows (`draft` / `validated` / `superseded`), `NextVersion` | Single active timeline; rebuild updates in place (fingerprint reuse when unchanged) |
| Validate | Flips status to `validated`, then separate render | Merged into **Ship** = reframe (if needed) + master render |
| Scenes | Enqueued after proxy; produces `ad` detections | **Removed** from default pipeline; ads/sponsors via **LLM** only |
| Reframe | Runs early after proxy (vertical-oriented) | Runs **at Ship**, driven by the **output preset** chosen in the front (any aspect / size) |
| Proxy | At ingest | Kept at ingest — editor playback only |
| Master encode | After validate via `POST …/renders` | Only at Ship |

## Pipeline

### Ingest / edit (always)

```
source ready
  → probe, audio, proxy (720p), waveform
  → silence, transcribe, text
  → llm (sponsors, chapters, viral)
  → on detections.updated → rebuild single timeline + realtime timeline.updated
```

Not enqueued by default: **scenes**, **reframe**.

### Ship (once, user-triggered)

```
front: preset + optional subtitle options
  → API Ship / finalize
  → if preset needs crop/scale ≠ source letterbox-only: enqueue reframe with preset geometry
  → enqueue master render (FFmpeg) using active timeline + preset (+ reframe asset if produced)
  → project/render status updates + realtime
```

Editor playback stays on **proxy** until Ship completes; download uses the render output.

## Single timeline

- Studio stays **project-scoped**: one active timeline row per project (the current head).
- Rebuild: recompute segments/decisions; if fingerprint matches, keep row; else **replace** segments/decisions on that same timeline id in a transaction (bump `updated_at` / fingerprint / duration). Do **not** insert a new version row or call `SupersedeDrafts` / `NextVersion` for normal rebuilds.
- Migration: keep the latest head row; mark or ignore older version rows (no UI/API for them).
- Drop (or stop exposing) `GET …/timeline/versions` and front `VER / n` UI.
- Remove validate-as-status-gate: no requirement that timeline be `validated` before render. Render rows carry their own status; no timeline version bump on Ship.

## Reframe at Ship (any front format)

- Front sends the selected **output preset** (width, height, aspect, kind) on Ship.
- Reframe job input = proxy (or source if required) + **target geometry from the preset**, not a hard-coded 9:16 path.
- If the preset is already compatible with source framing (e.g. same aspect, no smart crop required), Ship may skip reframe and render from source/proxy + timeline cuts only.
- Render plan always receives the chosen preset; subtitles/burn-in follow existing renderplan behavior.

## LLM replaces scenes for ads

- Sponsors/ads cuts: `llm` task `sponsors` → `KindSponsor` (and config actions already on configuration).
- Do not enqueue `scenes` on `proxy_ready`.
- Existing `KindAd` / scenes assets become unused for the default path; leave schema for now, stop writing new scenes results in the happy path.
- Reanalyze “quick” path stays silence + text; LLM re-run remains an explicit/heavier step (not required on every silence save).

## API / front surface

### API

- Keep: configuration PATCH, reanalyze, detections, timeline GET, timeline rebuild-on-event, realtime `timeline.updated`.
- Change: timeline rebuild persistence (single row).
- Remove `POST …/timeline/validate` from the happy path (handler can 410 or be deleted once front stops calling it).
- Ship = existing `POST …/projects/:id/renders` (extended):
  1. Requires an active timeline.
  2. Requires `presetId` (+ render kind as today).
  3. If preset geometry needs smart crop/scale: enqueue reframe with that preset’s width/height/aspect; on reframe success, enqueue render. If no reframe needed: enqueue render immediately.
- Stop requiring `StatusValidated` in render create.

### Front (studio-app)

- Silence drawer / edits: config + reanalyze → wait `timeline.updated` → reload (already aligned).
- Remove version chrome and draft/preview dual mode if preview is redundant; show the single timeline.
- **Validate → Ship** → one action: Ship with selected format (triggers reframe-if-needed + render).
- Pipeline chips: drop scenes from REFRAME stage meaning; REFRAME/SHIP reflect Ship progress; LLM covers sponsor/caption-ish work.

## Error handling

- Ship without timeline: 400.
- Ship without preset: 400.
- Reframe failure: render not started; surface step failed on media/project.
- Render failure: existing apply-result failure path.
- Silence/text reanalyze failure: existing step failed + no false timeline.updated success.

## Out of scope

- Implementing Cutflow’s missing master render inside Cutflow.
- Dubbing pipeline.
- Migrating historical timeline version rows beyond a one-shot “keep latest head as the single timeline” migration.
- Replacing proxy with original-file playback (possible later; not required for this change).

## Testing

- Rebuild updates one timeline; repeated rebuilds with same fingerprint do not create new rows.
- Proxy ready does **not** enqueue scenes.
- Ship with landscape and portrait presets both produce a render job (reframe invoked when geometry requires it).
- Front Ship does not call validate-as-status; render create succeeds on active draft-less timeline.
- LLM sponsor detections still drive timeline cuts via rebuild.

## Success criteria

- Editing never starts a master encode.
- Scenes worker idle on normal ingest.
- Reframe only tied to Ship + front preset.
- UI shows one timeline; Ship exports once for the chosen format.
