package ingest

import (
	"encoding/json"
	"testing"
)

func TestParseRequest_URLAndDefaults(t *testing.T) {
	raw := json.RawMessage(`{
		"source": {"kind": "url", "url": "https://example.com/v"},
		"normalize_cfr": true
	}`)
	req, err := ParseRequest(raw)
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Proxy.Height != 720 || req.Proxy.CRF != 26 {
		t.Fatalf("unexpected proxy defaults: %+v", req.Proxy)
	}
	if req.Audio.Analysis.SampleRate != 16000 {
		t.Fatalf("unexpected analysis sample rate: %d", req.Audio.Analysis.SampleRate)
	}
	if !req.NormalizeCFR {
		t.Fatal("normalize_cfr should default to true")
	}
}

func TestParseRequest_UploadRequiresKey(t *testing.T) {
	raw := json.RawMessage(`{"source": {"kind": "upload", "bucket": "media"}}`)
	if _, err := ParseRequest(raw); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestParseEnvelope_RequiresStepID(t *testing.T) {
	if _, err := ParseEnvelope([]byte(`{"schema":"envelope.v1","payload":{}}`)); err == nil {
		t.Fatal("expected error for missing step_id")
	}
}
