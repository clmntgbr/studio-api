package envelope

import (
	"encoding/json"
	"testing"
)

func TestNewJobRoundTrip(t *testing.T) {
	payload := map[string]any{
		"mediaId": "m1",
		"url":     "https://example.com/v",
	}
	msg, body, err := NewJob("fetch", "m1", "cause-1", 1, payload)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != "job.fetch" {
		t.Fatalf("type=%s", msg.Type)
	}
	parsed, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.MessageID != msg.MessageID || parsed.CorrelationID != "m1" {
		t.Fatalf("parsed mismatch: %+v", parsed)
	}
	var got map[string]any
	if err := json.Unmarshal(parsed.Payload, &got); err != nil {
		t.Fatal(err)
	}
	if got["mediaId"] != "m1" {
		t.Fatalf("payload=%v", got)
	}
}

func TestResultRoutingKey(t *testing.T) {
	if ResultRoutingKey("audio", true) != "result.audio.succeeded" {
		t.Fatal("ok key")
	}
	if ResultRoutingKey("audio", false) != "result.audio.failed" {
		t.Fatal("fail key")
	}
}
