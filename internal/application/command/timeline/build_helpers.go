package timeline

import (
	"context"
	"errors"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"
	eng "go-api/internal/timeline"
)

func paginateAll() paginate.PaginateQuery {
	q := paginate.PaginateQuery{Limit: 100, Page: 1, SortBy: "position", OrderBy: paginate.OrderByAsc}
	q.Normalize()
	return q
}

func buildForMedia(
	ctx context.Context,
	media domainmedia.WriteRepository,
	detections domaindetection.ReadRepository,
	m *domainmedia.MediaFile,
) (eng.Output, error) {
	if m.DurationMs == nil || *m.DurationMs <= 0 {
		return eng.Output{}, errors.New("media duration unknown")
	}
	cfg, err := media.GetConfiguration(ctx, m.ID)
	if err != nil {
		return eng.Output{}, err
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(m.ID)
	}
	dets, err := detections.ListByMedia(ctx, m.ID)
	if err != nil {
		return eng.Output{}, err
	}
	overrides, err := detections.ListOverridesByMedia(ctx, m.ID)
	if err != nil {
		return eng.Output{}, err
	}

	in := eng.Input{
		MediaID:    m.ID,
		DurationMs: *m.DurationMs,
		Config: eng.Config{
			SilenceEnabled:       cfg.SilenceEnabled,
			SilenceMinDurationMs: cfg.SilenceMinDurationMs,
			PaddingBeforeMs:      cfg.PaddingBeforeMs,
			PaddingAfterMs:       cfg.PaddingAfterMs,
			SpeechMinDurationMs:  cfg.SpeechMinDurationMs,
			SponsorsAction:       eng.ActionRemove,
			AdsAction:            eng.ActionRemove,
		},
	}
	for _, d := range dets {
		in.Detections = append(in.Detections, eng.Detection{
			ID:            d.ID,
			Kind:          eng.DetectionKind(d.Kind),
			SourceStartMs: d.SourceStartMs,
			SourceEndMs:   d.SourceEndMs,
			Confidence:    d.Confidence,
		})
	}
	for _, o := range overrides {
		var kind *eng.DetectionKind
		if o.Kind != nil {
			k := eng.DetectionKind(*o.Kind)
			kind = &k
		}
		in.Overrides = append(in.Overrides, eng.Override{
			ID:            o.ID,
			Kind:          kind,
			Action:        eng.Action(o.Action),
			SourceStartMs: o.SourceStartMs,
			SourceEndMs:   o.SourceEndMs,
		})
	}
	return eng.Build(in), nil
}
