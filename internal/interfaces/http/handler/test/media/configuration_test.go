package mediatest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestMediaHandler_GetConfiguration_Success(t *testing.T) {
	getCfg := &mockGetConfig{out: sampleConfig()}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, getCfg, nil, nil)
	app := mount(h, true)

	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/configuration", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out presenter.MediaConfigurationResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.MediaFileID != testutil.TestMediaID.String() || !out.SilenceEnabled {
		t.Fatalf("%+v", out)
	}
	if getCfg.calls != 1 {
		t.Fatalf("calls=%d", getCfg.calls)
	}
}

func TestMediaHandler_GetConfiguration_Unauthorized(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/configuration", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetConfiguration_InvalidMediaID(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/not-a-uuid/configuration", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetConfiguration_NotFound(t *testing.T) {
	getCfg := &mockGetConfig{err: errors.New("configuration not found")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, getCfg, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/configuration", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetConfiguration_HandlerError_Internal(t *testing.T) {
	getCfg := &mockGetConfig{err: errors.New("boom")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, getCfg, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/configuration", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_Success(t *testing.T) {
	enabled := false
	updateCfg := &mockUpdateConfig{out: sampleConfig()}
	updateCfg.out.SilenceEnabled = false
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)

	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"silenceEnabled": enabled,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if updateCfg.calls != 1 || updateCfg.cmd.SilenceEnabled == nil || *updateCfg.cmd.SilenceEnabled {
		t.Fatalf("cmd=%+v", updateCfg.cmd)
	}
}

func TestMediaHandler_UpdateConfiguration_OutputPresetAndShipPrefs(t *testing.T) {
	updateCfg := &mockUpdateConfig{out: sampleConfig()}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)

	presetID := testutil.TestOverrideID.String()
	burn := true
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"outputPresetId": presetID,
		"burnSubtitles":  burn,
		"masterSplit":    "parts:5",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if updateCfg.cmd.OutputPresetID == nil || updateCfg.cmd.OutputPresetID.String() != presetID {
		t.Fatalf("preset=%v", updateCfg.cmd.OutputPresetID)
	}
	if updateCfg.cmd.ClearOutputPresetID {
		t.Fatal("expected ClearOutputPresetID=false")
	}
	if updateCfg.cmd.BurnSubtitles == nil || !*updateCfg.cmd.BurnSubtitles {
		t.Fatalf("burn=%v", updateCfg.cmd.BurnSubtitles)
	}
	if updateCfg.cmd.MasterSplit == nil || *updateCfg.cmd.MasterSplit != "parts:5" {
		t.Fatalf("split=%v", updateCfg.cmd.MasterSplit)
	}
}

func TestMediaHandler_UpdateConfiguration_ClearOutputPreset(t *testing.T) {
	updateCfg := &mockUpdateConfig{out: sampleConfig()}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)

	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"outputPresetId": "",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if !updateCfg.cmd.ClearOutputPresetID || updateCfg.cmd.OutputPresetID != nil {
		t.Fatalf("cmd=%+v", updateCfg.cmd)
	}
}

func TestMediaHandler_UpdateConfiguration_InvalidOutputPresetID(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"outputPresetId": "not-a-uuid",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_MasterSplitValidation(t *testing.T) {
	updateCfg := &mockUpdateConfig{err: errors.New("masterSplit must be empty, off, parts:N, or durationMs:N")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"masterSplit": "nope",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_Unauthorized(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"silenceEnabled": true,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_InvalidMediaID(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/bad/configuration", map[string]any{"silenceEnabled": true})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_InvalidInput(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"sponsorsAction": "explode",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_NotFound(t *testing.T) {
	updateCfg := &mockUpdateConfig{err: errors.New("media not found")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"silenceEnabled": true,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_BusinessValidation(t *testing.T) {
	updateCfg := &mockUpdateConfig{err: errors.New("thresholdMode must be auto or manual")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"silenceEnabled": true,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_UpdateConfiguration_HandlerError_Internal(t *testing.T) {
	updateCfg := &mockUpdateConfig{err: errors.New("boom")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, updateCfg, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/media/"+testutil.TestMediaID.String()+"/configuration", map[string]any{
		"silenceEnabled": true,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_Success(t *testing.T) {
	reanalyze := &mockReanalyze{}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", map[string]any{
		"steps": []string{"silence"},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if reanalyze.calls != 1 || len(reanalyze.cmd.Steps) != 1 || reanalyze.cmd.Steps[0] != "silence" {
		t.Fatalf("cmd=%+v", reanalyze.cmd)
	}
}

func TestMediaHandler_Reanalyze_Text_Success(t *testing.T) {
	reanalyze := &mockReanalyze{}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", map[string]any{
		"steps": []string{"text"},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if reanalyze.calls != 1 || len(reanalyze.cmd.Steps) != 1 || reanalyze.cmd.Steps[0] != "text" {
		t.Fatalf("cmd=%+v", reanalyze.cmd)
	}
}

func TestMediaHandler_Reanalyze_Scenes_Success(t *testing.T) {
	reanalyze := &mockReanalyze{}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", map[string]any{
		"steps": []string{"scenes"},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if reanalyze.calls != 1 || len(reanalyze.cmd.Steps) != 1 || reanalyze.cmd.Steps[0] != "scenes" {
		t.Fatalf("cmd=%+v", reanalyze.cmd)
	}
}

func TestMediaHandler_Reanalyze_Reframe_Success(t *testing.T) {
	reanalyze := &mockReanalyze{}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", map[string]any{
		"steps": []string{"reframe"},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if reanalyze.calls != 1 || len(reanalyze.cmd.Steps) != 1 || reanalyze.cmd.Steps[0] != "reframe" {
		t.Fatalf("cmd=%+v", reanalyze.cmd)
	}
}

func TestMediaHandler_Reanalyze_EmptyBody_Success(t *testing.T) {
	reanalyze := &mockReanalyze{}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_Unauthorized(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_InvalidMediaID(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/bad/reanalyze", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_InvalidInput(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", map[string]any{
		"steps": []string{"unknown"},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_NotFound(t *testing.T) {
	reanalyze := &mockReanalyze{err: errors.New("media not found")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_NotReady(t *testing.T) {
	reanalyze := &mockReanalyze{err: errors.New("audio not ready")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Reanalyze_HandlerError_Internal(t *testing.T) {
	reanalyze := &mockReanalyze{err: errors.New("boom")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, reanalyze, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/reanalyze", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
