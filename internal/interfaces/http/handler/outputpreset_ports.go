package handler

import (
	"context"

	domainpreset "go-api/internal/domain/outputpreset"
)

type outputPresetListHandler interface {
	Handle(ctx context.Context) ([]domainpreset.Preset, error)
}
