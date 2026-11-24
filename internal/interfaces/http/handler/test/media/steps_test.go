package mediatest

import (
	"errors"
	"net/http"
	"testing"
	"time"

	querymedia "go-api/internal/application/query/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestMediaHandler_ListSteps_Success(t *testing.T) {
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	dur := int64(1500)
	list := &mockListStepRuns{out: &querymedia.ListStepRunsResult{
		Steps: []domainmedia.StepView{{
			Kind: domainmedia.StepProbe, Status: domainmedia.StepStatusCompleted, Progress: 1, DurationMs: &dur,
			StartedAt: &now, CompletedAt: &now,
		}},
		Runs: []domainmedia.StepRunView{{
			ID: testutil.TestMediaID, Kind: domainmedia.StepProbe, Attempt: 1,
			Status: domainmedia.StepStatusCompleted, StartedAt: now, CompletedAt: &now, DurationMs: &dur, CreatedAt: now,
		}},
	}}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, list)
	app := mount(h, true)

	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/steps", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out presenter.MediaStepsResponse
	testutil.DecodeJSON(t, resp, &out)
	if len(out.Steps) != 1 || out.Steps[0].DurationMs == nil || *out.Steps[0].DurationMs != 1500 {
		t.Fatalf("steps=%+v", out.Steps)
	}
	if len(out.Runs) != 1 || out.Runs[0].Attempt != 1 {
		t.Fatalf("runs=%+v", out.Runs)
	}
}

func TestMediaHandler_ListSteps_Unauthorized(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/steps", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_ListSteps_InvalidMediaID(t *testing.T) {
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/steps", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_ListSteps_HandlerError_Internal(t *testing.T) {
	list := &mockListStepRuns{err: errors.New("boom")}
	h := newHandlerWithConfig(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, list)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/steps", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
