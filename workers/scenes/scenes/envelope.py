from __future__ import annotations

import json
import uuid
from datetime import datetime, timezone
from typing import Any


def parse(body: bytes) -> dict[str, Any]:
    msg = json.loads(body.decode("utf-8"))
    if not msg.get("messageId") or not msg.get("type"):
        raise ValueError("envelope messageId and type are required")
    msg.setdefault("version", 1)
    msg.setdefault("attempt", 1)
    return msg


def _build(msg_type: str, correlation_id: str, causation_id: str, attempt: int, payload: Any) -> tuple[dict[str, Any], bytes]:
    msg = {
        "messageId": str(uuid.uuid4()),
        "type": msg_type,
        "version": 1,
        "correlationId": correlation_id,
        "causationId": causation_id or "",
        "occurredAt": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "attempt": attempt if attempt > 0 else 1,
        "payload": payload,
    }
    return msg, json.dumps(msg, separators=(",", ":")).encode("utf-8")


def new_progress(job_type: str, correlation_id: str, causation_id: str, attempt: int, pct: float, stage: str, message: str) -> tuple[dict[str, Any], bytes]:
    return _build(
        f"progress.{job_type}",
        correlation_id,
        causation_id,
        attempt,
        {"pct": pct, "stage": stage, "message": message},
    )


def new_result(
    job_type: str,
    correlation_id: str,
    causation_id: str,
    attempt: int,
    ok: bool,
    output: Any | None = None,
    code: str = "",
    err_msg: str = "",
    retryable: bool = False,
) -> tuple[dict[str, Any], bytes, str]:
    if ok:
        payload: dict[str, Any] = {"status": "succeeded"}
        if output is not None:
            payload["output"] = output
        routing = f"result.{job_type}.succeeded"
    else:
        payload = {
            "status": "failed",
            "error": {"code": code or "ERROR", "message": err_msg or "failed", "retryable": retryable},
        }
        routing = f"result.{job_type}.failed"
    msg, body = _build(routing, correlation_id, causation_id, attempt, payload)
    return msg, body, routing
