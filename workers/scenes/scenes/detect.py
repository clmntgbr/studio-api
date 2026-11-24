from __future__ import annotations

import re
import subprocess
from dataclasses import dataclass
from typing import Any

from scenedetect import SceneManager, open_video
from scenedetect.detectors import ContentDetector


@dataclass
class Interval:
    start_ms: int
    end_ms: int
    confidence: float
    reason: str
    cut_count: int = 0


def _ms(seconds: float) -> int:
    return max(0, int(round(seconds * 1000)))


def detect_scene_cuts(video_path: str, threshold: float = 27.0) -> list[dict[str, Any]]:
    """Return scene boundaries from PySceneDetect ContentDetector."""
    video = open_video(video_path)
    manager = SceneManager()
    manager.add_detector(ContentDetector(threshold=threshold))
    manager.detect_scenes(video)
    scenes = manager.get_scene_list()
    out: list[dict[str, Any]] = []
    for idx, (start, end) in enumerate(scenes):
        out.append(
            {
                "index": idx,
                "startMs": _ms(start.get_seconds()),
                "endMs": _ms(end.get_seconds()),
            }
        )
    return out


def _run_ffmpeg_detect(video_path: str, filter_expr: str) -> str:
    # silencedetect is audio (-af); blackdetect is video (-vf).
    flag = "-af" if "silencedetect" in filter_expr else "-vf"
    cmd = [
        "ffmpeg",
        "-hide_banner",
        "-nostats",
        "-i",
        video_path,
        flag,
        filter_expr,
        "-f",
        "null",
        "-",
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True, check=False)
    # Missing audio on proxy makes silencedetect fail — treat as empty, not fatal.
    if proc.returncode != 0 and "silencedetect" not in filter_expr:
        raise RuntimeError(f"ffmpeg detect failed (code={proc.returncode}): {(proc.stderr or '')[-500:]}")
    return (proc.stderr or "") + (proc.stdout or "")


def detect_black_intervals(video_path: str, min_duration_s: float = 0.3) -> list[tuple[int, int]]:
    log = _run_ffmpeg_detect(video_path, f"blackdetect=d={min_duration_s}:pix_th=0.10")
    starts: list[float] = []
    ends: list[float] = []
    for line in log.splitlines():
        if "black_start:" in line:
            m = re.search(r"black_start:\s*([0-9.]+)", line)
            if m:
                starts.append(float(m.group(1)))
        if "black_end:" in line:
            m = re.search(r"black_end:\s*([0-9.]+)", line)
            if m:
                ends.append(float(m.group(1)))
    out: list[tuple[int, int]] = []
    for i, start in enumerate(starts):
        end = ends[i] if i < len(ends) else start
        if end > start:
            out.append((_ms(start), _ms(end)))
    return out


def detect_silence_intervals(video_path: str, noise_db: float = -40.0, min_duration_s: float = 0.3) -> list[tuple[int, int]]:
    log = _run_ffmpeg_detect(video_path, f"silencedetect=noise={noise_db}dB:d={min_duration_s}")
    starts: list[float] = []
    ends: list[float] = []
    for line in log.splitlines():
        if "silence_start:" in line:
            m = re.search(r"silence_start:\s*([0-9.]+)", line)
            if m:
                starts.append(float(m.group(1)))
        if "silence_end:" in line:
            m = re.search(r"silence_end:\s*([0-9.]+)", line)
            if m:
                ends.append(float(m.group(1)))
    out: list[tuple[int, int]] = []
    for i, start in enumerate(starts):
        end = ends[i] if i < len(ends) else start
        if end > start:
            out.append((_ms(start), _ms(end)))
    return out


def _overlap_ms(a: tuple[int, int], b: tuple[int, int]) -> int:
    start = max(a[0], b[0])
    end = min(a[1], b[1])
    return max(0, end - start)


def _covers(intervals: list[tuple[int, int]], window: tuple[int, int], min_ratio: float = 0.35) -> bool:
    if window[1] <= window[0]:
        return False
    covered = sum(_overlap_ms(window, iv) for iv in intervals)
    return (covered / (window[1] - window[0])) >= min_ratio


def propose_ad_candidates(
    scenes: list[dict[str, Any]],
    black: list[tuple[int, int]],
    silence: list[tuple[int, int]],
    *,
    window_ms: int = 45_000,
    min_cuts: int = 4,
    min_duration_ms: int = 8_000,
    max_duration_ms: int = 180_000,
) -> list[Interval]:
    """
    Score candidate ad breaks from cut density near black/silence cues.

    Typical TV ads: rapid scene changes clustered between black frames or silence pads.
    """
    if not scenes:
        return []

    cut_ms = sorted({s["startMs"] for s in scenes[1:]})  # first scene start is usually 0
    duration_ms = scenes[-1]["endMs"] if scenes else 0
    if duration_ms <= 0:
        return []

    candidates: list[Interval] = []

    # Sliding windows over cut density.
    step = max(5_000, window_ms // 4)
    for start in range(0, max(1, duration_ms - min_duration_ms + 1), step):
        end = min(duration_ms, start + window_ms)
        if end - start < min_duration_ms:
            continue
        cuts = [c for c in cut_ms if start <= c < end]
        if len(cuts) < min_cuts:
            continue

        window = (start, end)
        has_black = _covers(black, window, 0.05) or any(start - 1500 <= b0 <= end + 1500 for b0, _ in black)
        has_silence = _covers(silence, window, 0.15) or any(start - 1500 <= s0 <= end + 1500 for s0, _ in silence)
        dense = len(cuts) >= min_cuts

        # Spec: density of cuts + black/silence around breaks. Require density AND at least one cue.
        if not (dense and (has_black or has_silence)):
            continue

        confidence = min(0.95, 0.45 + 0.06 * len(cuts))
        reasons: list[str] = [f"cut_density={len(cuts)}/{(end - start) / 1000:.0f}s"]
        if has_black:
            confidence = min(0.98, confidence + 0.18)
            reasons.append("black_near")
        if has_silence:
            confidence = min(0.98, confidence + 0.12)
            reasons.append("silence_near")
        if confidence < 0.55:
            continue

        # Tighten to span of cuts (+ padding).
        tight_start = max(0, cuts[0] - 500)
        tight_end = min(duration_ms, cuts[-1] + 500)
        if tight_end - tight_start < min_duration_ms:
            tight_start = start
            tight_end = end
        if tight_end - tight_start > max_duration_ms:
            continue

        candidates.append(
            Interval(
                start_ms=tight_start,
                end_ms=tight_end,
                confidence=round(confidence, 3),
                reason=",".join(reasons),
                cut_count=len(cuts),
            )
        )

    # Black+silence pads (typical TV ad bumper) even without cut density.
    for b in black:
        if b[1] - b[0] < 1500:
            continue
        for s in silence:
            if _overlap_ms(b, s) < 800:
                continue
            pad_start = max(0, min(b[0], s[0]) - 1000)
            pad_end = min(duration_ms, max(b[1], s[1]) + 1000)
            if pad_end - pad_start < 2000:
                continue
            candidates.append(
                Interval(
                    start_ms=pad_start,
                    end_ms=pad_end,
                    confidence=0.6,
                    reason="black_silence_pad",
                    cut_count=0,
                )
            )

    # Always allow pad path even when PySceneDetect found no scenes.
    if not scenes and black and silence:
        for b in black:
            for s in silence:
                if _overlap_ms(b, s) < 800:
                    continue
                pad_start = max(0, min(b[0], s[0]) - 1000)
                pad_end = max(b[1], s[1]) + 1000
                candidates.append(
                    Interval(pad_start, pad_end, 0.6, "black_silence_pad", 0)
                )

    return _merge_intervals(candidates)


def _merge_intervals(items: list[Interval], gap_ms: int = 2000) -> list[Interval]:
    if not items:
        return []
    ordered = sorted(items, key=lambda x: (x.start_ms, -x.confidence))
    merged: list[Interval] = [ordered[0]]
    for cur in ordered[1:]:
        prev = merged[-1]
        if cur.start_ms <= prev.end_ms + gap_ms:
            prev.end_ms = max(prev.end_ms, cur.end_ms)
            prev.confidence = max(prev.confidence, cur.confidence)
            prev.cut_count = max(prev.cut_count, cur.cut_count)
            if cur.reason not in prev.reason:
                prev.reason = f"{prev.reason}|{cur.reason}"
        else:
            merged.append(cur)
    return merged


def analyze(video_path: str, content_threshold: float = 27.0) -> dict[str, Any]:
    scenes = detect_scene_cuts(video_path, threshold=content_threshold)
    black = detect_black_intervals(video_path)
    silence = detect_silence_intervals(video_path)
    ads = propose_ad_candidates(scenes, black, silence)
    return {
        "detector": "scenes.pyscenedetect@1",
        "scenes": scenes,
        "blackIntervals": [{"startMs": a, "endMs": b} for a, b in black],
        "silenceIntervals": [{"startMs": a, "endMs": b} for a, b in silence],
        "adCandidates": [
            {
                "sourceStartMs": a.start_ms,
                "sourceEndMs": a.end_ms,
                "confidence": a.confidence,
                "reason": a.reason,
                "cutCount": a.cut_count,
            }
            for a in ads
        ],
    }
