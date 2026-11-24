from scenes.detect import Interval, _merge_intervals, propose_ad_candidates


def test_merge_overlapping():
    merged = _merge_intervals(
        [
            Interval(0, 10_000, 0.5, "a", 2),
            Interval(9_000, 20_000, 0.8, "b", 5),
            Interval(40_000, 50_000, 0.6, "c", 3),
        ]
    )
    assert len(merged) == 2
    assert merged[0].start_ms == 0
    assert merged[0].end_ms == 20_000
    assert merged[0].confidence == 0.8
    assert merged[1].start_ms == 40_000


def test_propose_ad_candidates_requires_cue():
    scenes = [{"index": i, "startMs": i * 2000, "endMs": (i + 1) * 2000} for i in range(30)]
    # Density alone is not enough.
    assert propose_ad_candidates(scenes, [], [], window_ms=20_000, min_cuts=4, min_duration_ms=5_000) == []
    black = [(12_000, 13_000)]
    silence = [(11_500, 12_500)]
    ads = propose_ad_candidates(scenes, black, silence, window_ms=20_000, min_cuts=4, min_duration_ms=5_000)
    assert len(ads) >= 1
    assert ads[0].end_ms > ads[0].start_ms
    assert ads[0].confidence >= 0.55
