package project

import (
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft      Status = "draft"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusRendering  Status = "rendering"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Project struct {
	ID        uuid.UUID
	Name      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time

	events []event.DomainEvent
}

func NewProject(name string) *Project {
	now := time.Now().UTC()
	p := &Project{
		ID:        uuid.New(),
		Name:      name,
		Status:    StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p.recordEvent(ProjectCreated{
		ID:        uuid.New().String(),
		ProjectID: p.ID.String(),
		Name:      name,
		Timestamp: now,
	})
	return p
}

func (p *Project) PullEvents() []event.DomainEvent {
	ev := p.events
	p.events = nil
	return ev
}

func (p *Project) recordEvent(e event.DomainEvent) {
	p.events = append(p.events, e)
}

func (p *Project) Rename(name string) {
	p.Name = name
	p.UpdatedAt = time.Now().UTC()
}

func (p *Project) SetStatus(status Status) {
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
}
