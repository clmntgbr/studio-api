package cliptest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/testutil"
)

func TestClipHandler_ListByProject_Success(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/clips", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_NumberedPack_Success(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/clips/numbered-pack", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_NumberedPack_Manifest(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/clips/numbered-pack?format=txt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListByProject_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/clips", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListByProject_InvalidProjectID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/bad/clips", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListByProject_HandlerError(t *testing.T) {
	list := &mockList{err: errors.New("boom")}
	app := mount(list, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/clips", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Create_Success(t *testing.T) {
	create := &mockCreate{}
	app := mount(nil, create, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips", map[string]any{
		"mediaFileId":   testutil.TestMediaID.String(),
		"sourceStartMs": 0,
		"sourceEndMs":   2000,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated || create.calls != 1 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, create.calls)
	}
}

func TestClipHandler_Create_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips", map[string]any{
		"mediaFileId":   testutil.TestMediaID.String(),
		"sourceStartMs": 0,
		"sourceEndMs":   2000,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Create_InvalidProjectID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/bad/clips", map[string]any{
		"mediaFileId": testutil.TestMediaID.String(), "sourceStartMs": 0, "sourceEndMs": 1,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Create_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Create_NotFound(t *testing.T) {
	create := &mockCreate{err: errors.New("media not found")}
	app := mount(nil, create, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips", map[string]any{
		"mediaFileId": testutil.TestMediaID.String(), "sourceStartMs": 0, "sourceEndMs": 1,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Create_BadBounds(t *testing.T) {
	create := &mockCreate{err: errors.New("sourceEndMs must be greater than sourceStartMs")}
	app := mount(nil, create, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips", map[string]any{
		"mediaFileId": testutil.TestMediaID.String(), "sourceStartMs": 0, "sourceEndMs": 1,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Create_HandlerError(t *testing.T) {
	create := &mockCreate{err: errors.New("boom")}
	app := mount(nil, create, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips", map[string]any{
		"mediaFileId": testutil.TestMediaID.String(), "sourceStartMs": 0, "sourceEndMs": 1,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Update_Success(t *testing.T) {
	update := &mockUpdate{}
	app := mount(nil, nil, update, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/"+testClipID.String(), map[string]any{"selected": true})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || update.calls != 1 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, update.calls)
	}
}

func TestClipHandler_Update_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/"+testClipID.String(), map[string]any{"selected": true})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Update_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/bad", map[string]any{"selected": true})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Update_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/"+testClipID.String(), map[string]any{
		"sourceEndMs": 0,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Update_NotFound(t *testing.T) {
	update := &mockUpdate{err: errors.New("clip not found")}
	app := mount(nil, nil, update, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/"+testClipID.String(), map[string]any{"selected": true})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Update_BadBounds(t *testing.T) {
	update := &mockUpdate{err: errors.New("sourceEndMs must be greater than sourceStartMs")}
	app := mount(nil, nil, update, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/"+testClipID.String(), map[string]any{"sourceEndMs": 1})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Update_HandlerError(t *testing.T) {
	update := &mockUpdate{err: errors.New("boom")}
	app := mount(nil, nil, update, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/clips/"+testClipID.String(), map[string]any{"selected": true})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Analyze_Success(t *testing.T) {
	analyze := &mockAnalyze{}
	app := mount(nil, nil, nil, analyze, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips/analyze", map[string]any{
		"mediaId": testutil.TestMediaID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted || analyze.calls != 1 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, analyze.calls)
	}
}

func TestClipHandler_Analyze_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips/analyze", map[string]any{
		"mediaId": testutil.TestMediaID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Analyze_InvalidProjectID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/bad/clips/analyze", map[string]any{"mediaId": testutil.TestMediaID.String()})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Analyze_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips/analyze", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Analyze_NotFound(t *testing.T) {
	analyze := &mockAnalyze{err: errors.New("media not found")}
	app := mount(nil, nil, nil, analyze, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips/analyze", map[string]any{
		"mediaId": testutil.TestMediaID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Analyze_NotReady(t *testing.T) {
	analyze := &mockAnalyze{err: errors.New("words not ready")}
	app := mount(nil, nil, nil, analyze, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips/analyze", map[string]any{
		"mediaId": testutil.TestMediaID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Analyze_HandlerError(t *testing.T) {
	analyze := &mockAnalyze{err: errors.New("boom")}
	app := mount(nil, nil, nil, analyze, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/clips/analyze", map[string]any{
		"mediaId": testutil.TestMediaID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListArtifactsByMedia_Success(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/llm-artifacts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListArtifactsByMedia_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/llm-artifacts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListArtifactsByMedia_InvalidMediaID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/llm-artifacts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_ListArtifactsByMedia_HandlerError(t *testing.T) {
	artifacts := &mockArtifacts{err: errors.New("boom")}
	app := mount(nil, nil, nil, nil, artifacts, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/llm-artifacts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_Success(t *testing.T) {
	render := &mockRender{}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted || render.calls != 1 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, render.calls)
	}
	if render.last.Kind != "clip" || render.last.ClipID == nil || *render.last.ClipID != testClipID {
		t.Fatalf("cmd=%+v", render.last)
	}
	if render.last.PresetID != testPresetID {
		t.Fatalf("preset=%s", render.last.PresetID)
	}
}

func TestClipHandler_Render_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/bad/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("clip not found")}
	app := mount(nil, nil, nil, nil, nil, get, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_NotValidated(t *testing.T) {
	render := &mockRender{err: errors.New("timeline must be validated")}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_HandlerError(t *testing.T) {
	render := &mockRender{err: errors.New("boom")}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_CustomPreset(t *testing.T) {
	render := &mockRender{}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{
		"presetId": "01960000-0000-7000-8000-0000000000a1",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if render.last.PresetID.String() != "01960000-0000-7000-8000-0000000000a1" {
		t.Fatalf("preset=%s", render.last.PresetID)
	}
}

func TestClipHandler_Render_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{
		"presetId": "not-a-uuid",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_GetError(t *testing.T) {
	get := &mockGet{err: errors.New("boom")}
	app := mount(nil, nil, nil, nil, nil, get, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_NoOverlap(t *testing.T) {
	render := &mockRender{err: errors.New("clip has no overlapping timeline segments")}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_BadRequestFromCreate(t *testing.T) {
	render := &mockRender{err: errors.New("clipId is required for kind clip")}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestClipHandler_Render_WithStyleID(t *testing.T) {
	render := &mockRender{}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	styleID := testutil.TestOverrideID.String()
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", map[string]any{
		"subtitleStyleId": styleID,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if render.last.SubtitleStyleID == nil || render.last.SubtitleStyleID.String() != styleID {
		t.Fatalf("cmd=%+v", render.last)
	}
}

func TestClipHandler_Render_EmptyBody(t *testing.T) {
	render := &mockRender{}
	app := mount(nil, nil, nil, nil, nil, nil, render, true)
	req, err := http.NewRequest(http.MethodPost, "/api/clips/"+testClipID.String()+"/render", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
