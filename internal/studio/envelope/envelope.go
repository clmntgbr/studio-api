package envelope

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message is the shared AMQP envelope (docs/docs.md contracts).
type Message struct {
	MessageID     string          `json:"messageId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	CorrelationID string          `json:"correlationId"`
	CausationID   string          `json:"causationId,omitempty"`
	OccurredAt    string          `json:"occurredAt"`
	Attempt       int             `json:"attempt"`
	Payload       json.RawMessage `json:"payload"`
}

type ProgressPayload struct {
	Pct     float64 `json:"pct"`
	Stage   string  `json:"stage"`
	Message string  `json:"message"`
}

type ResultPayload struct {
	Status string          `json:"status"` // succeeded | failed
	Output json.RawMessage `json:"output,omitempty"`
	Error  *ResultError    `json:"error,omitempty"`
}

type ResultError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func Parse(body []byte) (*Message, error) {
	var m Message
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("invalid envelope: %w", err)
	}
	if m.MessageID == "" || m.Type == "" {
		return nil, fmt.Errorf("envelope messageId and type are required")
	}
	if m.Version == 0 {
		m.Version = 1
	}
	if m.Attempt == 0 {
		m.Attempt = 1
	}
	return &m, nil
}

func NewJob(jobType, correlationID, causationID string, attempt int, payload any) (*Message, []byte, error) {
	return build("job."+jobType, correlationID, causationID, attempt, payload)
}

func NewProgress(jobType, correlationID, causationID string, attempt int, pct float64, stage, message string) (*Message, []byte, error) {
	return build("progress."+jobType, correlationID, causationID, attempt, ProgressPayload{
		Pct: pct, Stage: stage, Message: message,
	})
}

func NewResult(jobType, correlationID, causationID string, attempt int, succeeded bool, output any, errCode, errMsg string, retryable bool) (*Message, []byte, error) {
	rp := ResultPayload{Status: "succeeded"}
	routingSuffix := "succeeded"
	if !succeeded {
		rp.Status = "failed"
		routingSuffix = "failed"
		rp.Error = &ResultError{Code: errCode, Message: errMsg, Retryable: retryable}
	} else if output != nil {
		raw, err := json.Marshal(output)
		if err != nil {
			return nil, nil, err
		}
		rp.Output = raw
	}
	m, body, err := build("result."+jobType+"."+routingSuffix, correlationID, causationID, attempt, rp)
	if err != nil {
		return nil, nil, err
	}
	// Type field already set by build from full type string.
	_ = routingSuffix
	return m, body, nil
}

func ResultRoutingKey(jobType string, succeeded bool) string {
	if succeeded {
		return "result." + jobType + ".succeeded"
	}
	return "result." + jobType + ".failed"
}

func build(msgType, correlationID, causationID string, attempt int, payload any) (*Message, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	if attempt <= 0 {
		attempt = 1
	}
	m := &Message{
		MessageID:     uuid.NewString(),
		Type:          msgType,
		Version:       1,
		CorrelationID: correlationID,
		CausationID:   causationID,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Attempt:       attempt,
		Payload:       raw,
	}
	body, err := json.Marshal(m)
	if err != nil {
		return nil, nil, err
	}
	return m, body, nil
}
