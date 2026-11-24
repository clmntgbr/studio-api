from __future__ import annotations

from dataclasses import dataclass

import cv2
import numpy as np


@dataclass
class Sample:
    t_ms: int
    cx: float
    cy: float
    zoom: float


def _smooth(samples: list[Sample], window: int = 5) -> list[Sample]:
    if len(samples) <= 2 or window <= 1:
        return samples
    half = max(1, window // 2)
    out: list[Sample] = []
    for i, s in enumerate(samples):
        lo = max(0, i - half)
        hi = min(len(samples), i + half + 1)
        chunk = samples[lo:hi]
        out.append(
            Sample(
                t_ms=s.t_ms,
                cx=sum(c.cx for c in chunk) / len(chunk),
                cy=sum(c.cy for c in chunk) / len(chunk),
                zoom=sum(c.zoom for c in chunk) / len(chunk),
            )
        )
    return out


class FaceTracker:
    """MediaPipe face detection with OpenCV Haar fallback."""

    def __init__(self) -> None:
        self._mp = None
        self._detector = None
        try:
            import mediapipe as mp

            self._mp = mp
            self._detector = mp.solutions.face_detection.FaceDetection(
                model_selection=1,
                min_detection_confidence=0.5,
            )
        except Exception:
            self._detector = None
        self._haar = cv2.CascadeClassifier(cv2.data.haarcascades + "haarcascade_frontalface_default.xml")

    def close(self) -> None:
        if self._detector is not None and hasattr(self._detector, "close"):
            self._detector.close()

    def _detect_box(self, frame_bgr: np.ndarray) -> tuple[float, float, float] | None:
        h, w = frame_bgr.shape[:2]
        if self._detector is not None and self._mp is not None:
            rgb = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2RGB)
            res = self._detector.process(rgb)
            if res.detections:
                best = max(res.detections, key=lambda d: d.score[0] if d.score else 0)
                bb = best.location_data.relative_bounding_box
                cx = bb.xmin + bb.width / 2.0
                cy = bb.ymin + bb.height / 2.0
                # Larger face → slightly tighter zoom (still >= 1).
                zoom = 1.0 + max(0.0, 0.35 - max(bb.width, bb.height))
                return (
                    float(min(1.0, max(0.0, cx))),
                    float(min(1.0, max(0.0, cy))),
                    float(max(1.0, min(1.6, zoom))),
                )

        gray = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2GRAY)
        faces = self._haar.detectMultiScale(gray, scaleFactor=1.1, minNeighbors=5, minSize=(48, 48))
        if len(faces) == 0:
            return None
        x, y, fw, fh = max(faces, key=lambda f: f[2] * f[3])
        cx = (x + fw / 2.0) / w
        cy = (y + fh / 2.0) / h
        zoom = 1.0 + max(0.0, 0.35 - max(fw / w, fh / h))
        return float(cx), float(cy), float(max(1.0, min(1.6, zoom)))

    def track(self, video_path: str, sample_fps: float = 5.0) -> dict:
        cap = cv2.VideoCapture(video_path)
        if not cap.isOpened():
            raise RuntimeError(f"cannot open video: {video_path}")

        src_fps = cap.get(cv2.CAP_PROP_FPS) or 30.0
        width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH) or 0)
        height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT) or 0)
        frame_interval = max(1, int(round(src_fps / max(sample_fps, 0.5))))

        samples: list[Sample] = []
        last = Sample(t_ms=0, cx=0.5, cy=0.42, zoom=1.05)
        idx = 0
        while True:
            ok, frame = cap.read()
            if not ok:
                break
            if idx % frame_interval == 0:
                t_ms = int(round(1000.0 * idx / src_fps))
                box = self._detect_box(frame)
                if box is None:
                    # Hold last known position (no jump to center).
                    samples.append(Sample(t_ms=t_ms, cx=last.cx, cy=last.cy, zoom=last.zoom))
                else:
                    last = Sample(t_ms=t_ms, cx=box[0], cy=box[1], zoom=box[2])
                    samples.append(last)
            idx += 1
        cap.release()

        if not samples:
            samples = [Sample(t_ms=0, cx=0.5, cy=0.42, zoom=1.05)]

        smoothed = _smooth(samples, window=5)
        return {
            "version": 1,
            "fps": sample_fps,
            "frameWidth": width,
            "frameHeight": height,
            "detector": "mediapipe" if self._detector is not None else "haar",
            "samples": [
                {"tMs": s.t_ms, "cx": round(s.cx, 5), "cy": round(s.cy, 5), "zoom": round(s.zoom, 4)}
                for s in smoothed
            ],
        }
