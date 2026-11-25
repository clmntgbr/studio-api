# Cutflow-like Edit → Single Timeline → Ship Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One editable timeline (no versions), no scenes / early reframe; Ship enqueues preset-driven reframe (if needed) then master render.

**Architecture:** Stop `enqueue_scenes` / `enqueue_reframe` on `proxy_ready`. Rebuild replaces segments/decisions on the project head timeline in place. `POST …/renders` no longer requires validated status; when the preset’s `cropMode` needs tracking, publish a reframe job with target width/height, keep the render `pending` until crop path exists, then publish the render job. Front Ship calls create-render only; drop version chrome.

**Tech Stack:** Go (Fiber), RabbitMQ studio jobs, Python reframe worker, Next.js studio-app, Centrifugo realtime (already wired)

**Spec:** `docs/superpowers/specs/2026-11-25-cutflow-like-edit-ship-design.md`

## Global Constraints

- Editing must never enqueue master encode
- Default ingest must not enqueue `scenes` or `reframe`
- Ads/sponsors come from LLM only
- One active timeline per project; no `NextVersion` / `SupersedeDrafts` on rebuild
- Reframe target geometry comes from the front-selected output preset (width/height/aspect), not a hard-coded 9:16
- Proxy at ingest stays for editor playback

## File map

| File | Responsibility |
|------|----------------|
| `cmd/worker/di/container.go` | Unregister scenes/reframe-on-proxy handlers |
| `internal/application/command/timeline/rebuild.go` | In-place replace head timeline |
| `internal/domain/timeline/repository.go` + `…/write/timeline_repository.go` | `ReplaceContents` (delete+insert segments/decisions, update head row) |
| `internal/application/command/render/create.go` | Drop validated gate; optional reframe-then-render |
| `internal/domain/port/jobs.go` + `studiojobs/publisher.go` | `PublishReframeJob` with target W/H |
| `workers/reframe/reframe/worker.py` (+ detect if needed) | Honor target width/height/aspect |
| `internal/application/command/media/apply_reframe_result.go` | After crop saved, start pending renders |
| `studio-app` timeline editor / hooks / pipeline | Ship without validate; hide versions |

---

### Task 1: Stop scenes + reframe on proxy_ready

**Files:**
- Modify: `cmd/worker/di/container.go` (remove or comment-out the two `reg.Register(…MediaProxyReady…)` blocks for `enqueue_scenes_on_proxy_ready` and `enqueue_reframe_on_proxy_ready`)
- Test: add/adjust a small worker wiring test if one exists; otherwise add `internal/application/event/media/enqueue_pipeline_test.go` asserting handlers are not registered — skip if too heavy; prefer a focused unit test that `EnqueueScenesHandler` / `EnqueueReframeHandler` are unused by documenting removal and a compile check

**Interfaces:**
- Consumes: existing `domainmedia.EventTypeMediaProxyReady` registrations
- Produces: proxy_ready no longer starts scenes/reframe

- [ ] **Step 1: Write a failing regression note as a test on worker registration list**

In `cmd/worker/di/container_test.go` (create if missing), or simpler: add `TestProxyReadyDoesNotEnqueueScenesReframe` as a documentation test in `internal/application/event/media/enqueue_scenes_test.go`:

```go
package media_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerDI_ProxyReadyDoesNotEnqueueScenesOrReframe(t *testing.T) {
	root := filepath.Join("..", "..", "..", "cmd", "worker", "di", "container.go")
	b, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if strings.Contains(src, `"enqueue_scenes_on_proxy_ready"`) {
		t.Fatal("scenes must not be registered on proxy_ready")
	}
	if strings.Contains(src, `"enqueue_reframe_on_proxy_ready"`) {
		t.Fatal("reframe must not be registered on proxy_ready")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd studio-api && go test ./internal/application/event/media/ -run ProxyReady -count=1
```

Expected: FAIL with scenes/reframe still registered.

- [ ] **Step 3: Remove both registrations from `cmd/worker/di/container.go`**

Delete the `reg.Register(domainmedia.EventTypeMediaProxyReady, … enqueue_scenes …)` and `… enqueue_reframe …` blocks. Leave handler types on disk (still usable from Ship / reanalyze).

- [ ] **Step 4: Re-run test — expect PASS**

```bash
cd studio-api && go test ./internal/application/event/media/ -run ProxyReady -count=1
```

- [ ] **Step 5: Commit**

```bash
cd studio-api && git add cmd/worker/di/container.go internal/application/event/media/enqueue_scenes_test.go
git commit -m "fix(pipeline): do not enqueue scenes/reframe on proxy_ready"
```

---

### Task 2: Timeline write — ReplaceContents

**Files:**
- Modify: `internal/domain/timeline/repository.go`
- Modify: `internal/infrastructure/persistence/write/timeline_repository.go`
- Create: `internal/infrastructure/persistence/write/timeline_repository_replace_test.go` (or package test with sqlite/postgres if project already has DB tests — prefer table-driven unit with mocked DB only if pattern exists; otherwise integration-style with existing test helpers)

**Interfaces:**
- Produces: `WriteRepository.ReplaceContents(ctx, t *Timeline, segments []Segment, decisions []Decision) error`  
  Semantics: update timeline row fields (fingerprint, durationMs, status stays `draft`); `DELETE` segments/decisions for `t.ID`; `INSERT` new ones. Do **not** change `t.ID` or `t.Version` (keep version=1 forever or leave existing head version untouched).

- [ ] **Step 1: Add interface method**

```go
// ReplaceContents updates the timeline row and replaces all segments/decisions.
ReplaceContents(ctx context.Context, t *Timeline, segments []Segment, decisions []Decision) error
```

- [ ] **Step 2: Implement in write repo**

```go
func (r *timelineWriteRepository) ReplaceContents(ctx context.Context, t *domaintimeline.Timeline, segments []domaintimeline.Segment, decisions []domaintimeline.Decision) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Model(&TimelineModel{}).Where("id = ?", t.ID).Updates(map[string]any{
		"fingerprint": t.Fingerprint,
		"duration_ms": t.DurationMs,
		"status":      string(t.Status),
	}).Error; err != nil {
		return err
	}
	if err := db.Where("timeline_id = ?", t.ID).Delete(&TimelineSegmentModel{}).Error; err != nil {
		return err
	}
	if err := db.Where("timeline_id = ?", t.ID).Delete(&TimelineDecisionModel{}).Error; err != nil {
		return err
	}
	for i := range segments {
		seg := segments[i]
		seg.TimelineID = t.ID
		if seg.ID == uuid.Nil {
			seg.ID = uuid.New()
		}
		if err := db.Create(timelineSegmentModelFromDomain(&seg)).Error; err != nil {
			return err
		}
	}
	for i := range decisions {
		dec := decisions[i]
		dec.TimelineID = t.ID
		if err := db.Create(timelineDecisionModelFromDomain(&dec)).Error; err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 3: Compile**

```bash
cd studio-api && go build ./internal/infrastructure/persistence/write/
```

Expected: PASS (or fix missing model names to match existing `TimelineSegmentModel` / `TimelineDecisionModel`).

- [ ] **Step 4: Commit**

```bash
cd studio-api && git add internal/domain/timeline/repository.go internal/infrastructure/persistence/write/timeline_repository.go
git commit -m "feat(timeline): ReplaceContents for single-timeline updates"
```

---

### Task 3: RebuildHandler uses single timeline

**Files:**
- Modify: `internal/application/command/timeline/rebuild.go`
- Create: `internal/application/command/timeline/rebuild_test.go` (table test with fakes)

**Interfaces:**
- Consumes: `ReplaceContents`, `FindHeadByProject`, `Save` (first timeline only)
- Produces: rebuild never calls `NextVersion` / `SupersedeDrafts`

- [ ] **Step 1: Write failing unit test with fakes**

Fake repos recording calls. Assert:
1. First rebuild with nil head → `Save` once, version forced to `1`.
2. Second rebuild different fingerprint → `ReplaceContents` on same ID, **no** `NextVersion` / `SupersedeDrafts`.
3. Same fingerprint → no Save/Replace.

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd studio-api && go test ./internal/application/command/timeline/ -run Rebuild -count=1
```

- [ ] **Step 3: Rewrite `RebuildHandler.Handle`**

Pseudo:

```go
head, err := h.timelineR.FindHeadByProject(ctx, cmd.ProjectID)
// ...
if head != nil && head.Fingerprint == out.Fingerprint {
  return head, out, nil
}
segments := mapSegments(out)
decisions := mapDecisions(out)
if head == nil {
  t := domaintimeline.New(cmd.ProjectID, 1, out.Fingerprint, out.DurationMs)
  err = h.timelines.WithTransaction(ctx, func(txCtx context.Context) error {
    return h.timelines.Save(txCtx, t, segments, decisions)
  })
  return t, out, err
}
head.Fingerprint = out.Fingerprint
head.DurationMs = out.DurationMs
head.Status = domaintimeline.StatusDraft
err = h.timelines.WithTransaction(ctx, func(txCtx context.Context) error {
  return h.timelines.ReplaceContents(txCtx, head, segments, decisions)
})
return head, out, err
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd studio-api && go test ./internal/application/command/timeline/ -count=1
```

- [ ] **Step 5: Commit**

```bash
cd studio-api && git add internal/application/command/timeline/
git commit -m "feat(timeline): rebuild updates single project timeline in place"
```

---

### Task 4: Render create — drop validated gate

**Files:**
- Modify: `internal/application/command/render/create.go` (remove lines checking `StatusValidated`)
- Modify: `internal/interfaces/http/handler/test/render/handler_test.go` — rename/repurpose `TestRenderHandler_Create_NotValidated` to assert create **succeeds** with draft timeline (or delete that case)
- Modify: `internal/interfaces/http/handler/test/clip/handler_test.go` if same gate exists for clip render

**Interfaces:**
- Consumes: head timeline with any status ∈ {draft, validated} (prefer: any non-empty head)
- Produces: create render without validate

- [ ] **Step 1: Change the failing expectation**

Update `TestRenderHandler_Create_NotValidated` → `TestRenderHandler_Create_DraftTimeline_OK` expecting 201/200 instead of 400.

- [ ] **Step 2: Run — expect FAIL** (handler still returns must be validated)

```bash
cd studio-api && go test ./internal/interfaces/http/handler/test/render/ -run DraftTimeline -count=1
```

- [ ] **Step 3: Remove validated check in `create.go`**

Delete:

```go
if t.Status != domaintimeline.StatusValidated {
  return nil, errors.New("timeline must be validated")
}
```

Keep `timeline not found` / empty segments checks.

- [ ] **Step 4: Run render handler tests — PASS**

```bash
cd studio-api && go test ./internal/interfaces/http/handler/test/render/ ./internal/interfaces/http/handler/test/clip/ -count=1
```

- [ ] **Step 5: Commit**

```bash
cd studio-api && git add internal/application/command/render/create.go internal/interfaces/http/handler/test/render/ internal/interfaces/http/handler/test/clip/
git commit -m "feat(render): allow Ship without validated timeline status"
```

---

### Task 5: Reframe job carries preset geometry

**Files:**
- Modify: `internal/domain/port/jobs.go` — extend signature  
  `PublishReframeJob(ctx, mediaID, proxyKey string, targetWidth, targetHeight int) error`
- Modify: `internal/infrastructure/messaging/studiojobs/publisher.go` — add `targetWidth` / `targetHeight` to payload
- Modify all call sites: `enqueue_reframe.go`, `reanalyze.go` (pass `0,0` meaning “source aspect / legacy”), later Ship passes preset sizes
- Modify: `workers/reframe/reframe/worker.py` — read optional `targetWidth`/`targetHeight`; if set, store in crop_path doc metadata and bias tracker envelope aspect toward `targetWidth/targetHeight`

**Interfaces:**
- Produces: job payload `{ mediaId, proxyKey, targetWidth?, targetHeight? }`

- [ ] **Step 1: Update port + publisher + compile all call sites with `0, 0`**

```go
func (p *Publisher) PublishReframeJob(ctx context.Context, mediaID, proxyKey string, targetWidth, targetHeight int) error {
	payload := map[string]any{
		"mediaId":  mediaID,
		"proxyKey": proxyKey,
	}
	if targetWidth > 0 && targetHeight > 0 {
		payload["targetWidth"] = targetWidth
		payload["targetHeight"] = targetHeight
	}
	// ...
}
```

- [ ] **Step 2: Worker reads targets**

In `handle` after parsing payload:

```python
target_w = int(payload.get("targetWidth") or 0)
target_h = int(payload.get("targetHeight") or 0)
# pass into FaceTracker.track(..., target_aspect=(target_w/target_h) if target_w and target_h else None)
# include targetWidth/targetHeight in uploaded crop_path.json
```

If `detect.py` has no aspect parameter yet, add optional `target_aspect: float | None` and use it when computing crop window width/height ratio (fallback: frame aspect).

- [ ] **Step 3: Smoke compile / import**

```bash
cd studio-api && go build ./...
python -m py_compile workers/reframe/reframe/worker.py workers/reframe/reframe/detect.py
```

- [ ] **Step 4: Commit**

```bash
cd studio-api && git add internal/domain/port/jobs.go internal/infrastructure/messaging/studiojobs/publisher.go internal/application/event/media/enqueue_reframe.go internal/application/command/media/reanalyze.go workers/reframe/
git commit -m "feat(reframe): pass target width/height from preset into job"
```

---

### Task 6: Ship = create render + conditional reframe chain

**Files:**
- Modify: `internal/application/command/render/create.go`
- Modify: `internal/application/command/media/apply_reframe_result.go`
- Possibly: `internal/domain/render` status already has `pending` — use it
- Store link: render settings JSON or fingerprint already includes preset; add optional `media_file_id` wait flag via existing fields if present — else add `renders.meta` / use `Settings` blob `{ "awaitingReframe": true, "mediaId": "..." }`

**Logic:**

```text
needsTrack := preset.CropMode == "face_track" || preset.CropMode == "speaker_track"
hasCrop := cropPathKey != nil
if needsTrack && !hasCrop {
  // create render row StatusPending, do NOT PublishRenderJob yet
  // QueueStep(reframe) + PublishReframeJob(media, proxy, preset.Width, preset.Height)
  return render, nil
}
// else existing path: StatusQueued + PublishRenderJob
```

On `ApplyReframeResultHandler` success:
- Find project renders in `pending` with awaitingReframe for this mediaId
- Build plan (same as create) or call shared `publishRenderJob(render)`
- Set status queued and publish

**Helper:** extract `needsReframe(preset, media) (bool, proxyKey string, err)` and `startRenderJob(...)` from create.go to avoid duplication.

- [ ] **Step 1: Unit-test CreateHandler with fake jobs** — when cropMode face_track and no crop asset, expect `PublishReframeJob` called with preset W/H and **no** `PublishRenderJob`; render status pending.

- [ ] **Step 2: Implement create.go branch + apply_reframe kickoff**

- [ ] **Step 3: Tests PASS**

```bash
cd studio-api && go test ./internal/application/command/render/ ./internal/application/command/media/ -count=1
```

- [ ] **Step 4: Commit**

```bash
cd studio-api && git add internal/application/command/render/ internal/application/command/media/apply_reframe_result.go
git commit -m "feat(ship): reframe-on-demand then render for tracked presets"
```

---

### Task 7: Front — Ship without validate; single timeline UI

**Files:**
- Modify: `studio-app/components/editor/timeline-editor.tsx` — `onValidate` → `onShip`: only `createRender.mutateAsync`; remove `validate.mutateAsync`; remove VER display; simplify preview dual-mode (keep Rebuild as optional dry-run or remove if redundant)
- Modify: `studio-app/lib/timeline/hooks.ts` — stop exporting validate usage from editor (can leave API helper)
- Modify: `studio-app/components/editor/factory/pipeline-steps.tsx` — REFRAME kinds empty or ship-driven; remove scenes from mental model; SHIP active when render running / done when latest render completed
- Modify: `studio-app/lib/output-presets/hooks.ts` if needed for render list polling

**Interfaces:**
- Consumes: `POST /api/projects/:id/renders` with `{ presetId, kind }`
- Produces: no call to `/timeline/validate`

- [ ] **Step 1: Change Ship handler**

```ts
async function onShip() {
  if (!selectedPreset) {
    setActionError("Choisis un format de sortie")
    return
  }
  setActionError(null)
  setActionOk(null)
  try {
    const render = await createRender.mutateAsync({
      presetId: selectedPreset.id,
      kind: renderKindForPreset(selectedPreset),
    })
    setActionOk(`Ship · render ${render.status} (${selectedPreset.name})`)
  } catch (err) {
    setActionError(err instanceof Error ? err.message : "Ship failed")
  }
}
```

Remove `saved?.version == null` disable reason tied to validate; disable Ship only when no timeline segments / no preset / busy.

- [ ] **Step 2: Remove `VER / {version}` from header**

- [ ] **Step 3: PipelineSteps** — drop scenes from REFRAME `kinds`; pass `shipping` / `shipped` from parent based on latest render status instead of `validated`

- [ ] **Step 4: Typecheck**

```bash
cd studio-app && npx tsc --noEmit
```

Expected: PASS

- [ ] **Step 5: Commit (studio-app repo)**

```bash
cd studio-app && git add components/editor/timeline-editor.tsx components/editor/factory/pipeline-steps.tsx lib/timeline/hooks.ts
git commit -m "feat(editor): Ship without timeline versions/validate"
```

---

### Task 8: Retire validate from happy path

**Files:**
- Modify: `studio-api/cmd/api/routes.go` — remove or leave route returning 410
- Prefer: keep handler but front unused; mark Validate handler:

```go
return c.Status(fiber.StatusGone).JSON(fiber.Map{
  "message": "timeline validate removed; POST /projects/:id/renders to ship",
})
```

- Update timeline handler tests that hit validate to expect 410 **or** delete those tests if route removed.

- [ ] **Step 1: Return 410 from Validate**
- [ ] **Step 2: Fix handler tests**
- [ ] **Step 3: Commit**

```bash
cd studio-api && git add cmd/api/routes.go internal/interfaces/http/handler/timeline_handler.go internal/interfaces/http/handler/test/timeline/
git commit -m "chore(timeline): deprecate validate endpoint (gone)"
```

---

### Task 9: Restart workers / smoke

- [ ] **Step 1: Restart**

```bash
cd studio-api && docker compose -f compose.dev.yaml restart api worker worker-reframe
```

- [ ] **Step 2: Manual smoke checklist**
  1. Open editor — proxy plays
  2. Silence save → timeline.updated → timeline reloads (one timeline)
  3. Confirm no new `scenes` / `reframe` steps on fresh ingest after proxy
  4. Ship with 16:9 preset → render without reframe if cropMode none
  5. Ship with 9:16 face_track preset → reframe queued then render completes

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| No scenes on ingest | Task 1 |
| No early reframe | Task 1 + 6 |
| Single timeline in-place | Tasks 2–3 |
| Ship without validate | Tasks 4, 7, 8 |
| Reframe any front format | Tasks 5–6 |
| LLM for sponsors (unchanged path) | Task 1 (scenes off) |
| Front single timeline / Ship | Task 7 |
| Proxy kept | (no change) |
