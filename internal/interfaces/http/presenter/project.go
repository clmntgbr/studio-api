package presenter

import (
	"time"

	domainproject "go-api/internal/domain/project"
)

type ProjectResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewProjectResponseFromView(v domainproject.View) ProjectResponse {
	return ProjectResponse{
		ID:        v.ID.String(),
		Name:      v.Name,
		Status:    string(v.Status),
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}

func NewProjectResponseFromEntity(p *domainproject.Project) ProjectResponse {
	return ProjectResponse{
		ID:        p.ID.String(),
		Name:      p.Name,
		Status:    string(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func NewProjectListResponseFromViews(views []domainproject.View) []ProjectResponse {
	out := make([]ProjectResponse, 0, len(views))
	for _, v := range views {
		out = append(out, NewProjectResponseFromView(v))
	}
	return out
}
