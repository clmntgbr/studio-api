package handler

import (
	"context"

	querydetection "go-api/internal/application/query/detection"
	domaindetection "go-api/internal/domain/detection"
)

type detectionListHandler interface {
	Handle(ctx context.Context, q querydetection.ListDetectionsQuery) ([]domaindetection.Detection, error)
}
