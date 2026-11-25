from __future__ import annotations

import json
import logging
import os
import shutil
import threading
from pathlib import Path

import boto3
import pika
from botocore.client import Config as BotoConfig

from reframe import envelope
from reframe.detect import FaceTracker

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("worker-reframe")

EXCHANGE_JOBS = "studio.jobs"
EXCHANGE_RESULTS = "studio.results"
EXCHANGE_PROGRESS = "studio.progress"


def env(name: str, default: str = "") -> str:
    return os.environ.get(name, default)


def s3_client():
    endpoint = env("S3_ENDPOINT", "http://minio:9000")
    return boto3.client(
        "s3",
        endpoint_url=endpoint,
        aws_access_key_id=env("S3_ACCESS_KEY", "devuser"),
        aws_secret_access_key=env("S3_SECRET_KEY", "devpassword"),
        region_name=env("S3_REGION", "us-east-1"),
        config=BotoConfig(signature_version="s3v4", s3={"addressing_style": "path"}),
    )


def declare_topology(ch: pika.channel.Channel) -> None:
    # Match go-api/internal/studio/topology (studio.jobs is direct).
    ch.exchange_declare(exchange=EXCHANGE_JOBS, exchange_type="direct", durable=True)
    ch.exchange_declare(exchange=EXCHANGE_RESULTS, exchange_type="topic", durable=True)
    ch.exchange_declare(exchange=EXCHANGE_PROGRESS, exchange_type="fanout", durable=True)

    main = "q.reframe"
    retry = "q.reframe.retry"
    dlq = "q.reframe.dlq"
    ch.queue_declare(queue=dlq, durable=True)
    ch.queue_declare(
        queue=main,
        durable=True,
        arguments={
            "x-dead-letter-exchange": "",
            "x-dead-letter-routing-key": dlq,
        },
    )
    ch.queue_declare(
        queue=retry,
        durable=True,
        arguments={
            "x-message-ttl": 30000,
            "x-dead-letter-exchange": "",
            "x-dead-letter-routing-key": main,
        },
    )
    ch.queue_bind(queue=main, exchange=EXCHANGE_JOBS, routing_key="job.reframe")


def publish(
    ch: pika.channel.Channel,
    exchange: str,
    routing_key: str,
    body: bytes,
    message_id: str,
    correlation_id: str,
) -> None:
    ch.basic_publish(
        exchange=exchange,
        routing_key=routing_key,
        body=body,
        properties=pika.BasicProperties(
            content_type="application/json",
            delivery_mode=2,
            message_id=message_id,
            correlation_id=correlation_id,
        ),
        mandatory=False,
    )


def _run_track_keepalive(
    ch: pika.channel.Channel,
    video_path: str,
    sample_fps: float,
    target_aspect: float | None = None,
) -> dict:
    """Run face tracking off the I/O thread while pumping AMQP heartbeats."""
    box: dict = {}

    def target() -> None:
        tracker = FaceTracker()
        try:
            box["doc"] = tracker.track(video_path, sample_fps=sample_fps, target_aspect=target_aspect)
        except Exception as exc:  # noqa: BLE001 — surfaced to caller
            box["err"] = exc
        finally:
            tracker.close()

    thread = threading.Thread(target=target, name="reframe-track", daemon=True)
    thread.start()
    while thread.is_alive():
        ch.connection.process_data_events(time_limit=1)
    thread.join()
    if "err" in box:
        raise box["err"]
    return box["doc"]


def handle(ch: pika.channel.Channel, method, body: bytes, store, bucket: str, work_dir: Path) -> None:
    try:
        msg = envelope.parse(body)
    except Exception as exc:
        log.error("invalid envelope: %s", exc)
        ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
        return

    corr = msg.get("correlationId") or msg["messageId"]
    attempt = int(msg.get("attempt") or 1)
    payload = msg.get("payload") or {}
    if isinstance(payload, str):
        payload = json.loads(payload)

    media_id = payload.get("mediaId") or corr
    proxy_key = payload.get("proxyKey") or ""
    target_w = int(payload.get("targetWidth") or 0)
    target_h = int(payload.get("targetHeight") or 0)
    target_aspect = (target_w / target_h) if target_w > 0 and target_h > 0 else None
    if not media_id or not proxy_key:
        m, out, rk = envelope.new_result(
            "reframe", corr, msg["messageId"], attempt, False, None, "invalid_payload", "mediaId/proxyKey required", False
        )
        publish(ch, EXCHANGE_RESULTS, rk, out, m["messageId"], corr)
        ch.basic_ack(delivery_tag=method.delivery_tag)
        return

    job_dir = work_dir / f"reframe-{media_id}"
    if job_dir.exists():
        shutil.rmtree(job_dir, ignore_errors=True)
    job_dir.mkdir(parents=True, exist_ok=True)

    def progress(pct: float, stage: str, message: str) -> None:
        m, body_p = envelope.new_progress("reframe", corr, msg["messageId"], attempt, pct, stage, message)
        publish(ch, EXCHANGE_PROGRESS, "", body_p, m["messageId"], corr)

    try:
        progress(5, "download", "Downloading proxy")
        local = job_dir / "proxy.mp4"
        key = proxy_key.lstrip("/")
        store.download_file(bucket, key, str(local))

        progress(30, "detect", "Tracking faces")
        path_doc = _run_track_keepalive(
            ch, str(local), float(env("REFRAME_SAMPLE_FPS", "5")), target_aspect=target_aspect
        )
        if target_w > 0 and target_h > 0:
            path_doc["targetWidth"] = target_w
            path_doc["targetHeight"] = target_h

        progress(80, "upload", "Uploading crop_path.json")
        crop_key = f"media/{media_id}/reframe/crop_path.json"
        crop_local = job_dir / "crop_path.json"
        crop_local.write_text(json.dumps(path_doc), encoding="utf-8")
        store.upload_file(str(crop_local), bucket, crop_key, ExtraArgs={"ContentType": "application/json"})

        progress(100, "done", "Reframe completed")
        m, out, rk = envelope.new_result(
            "reframe",
            corr,
            msg["messageId"],
            attempt,
            True,
            {
                "cropPathKey": crop_key,
                "sampleCount": len(path_doc.get("samples") or []),
                "detector": path_doc.get("detector"),
            },
        )
        publish(ch, EXCHANGE_RESULTS, rk, out, m["messageId"], corr)
        ch.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as exc:
        log.exception("reframe failed media=%s: %s", media_id, exc)
        m, out, rk = envelope.new_result(
            "reframe",
            corr,
            msg["messageId"],
            attempt,
            False,
            None,
            "reframe_error",
            "reframe processing failed",
            True,
        )
        publish(ch, EXCHANGE_RESULTS, rk, out, m["messageId"], corr)
        ch.basic_ack(delivery_tag=method.delivery_tag)
    finally:
        shutil.rmtree(job_dir, ignore_errors=True)


def main() -> None:
    rabbit = env("RABBITMQ_URL")
    if not rabbit:
        raise SystemExit("RABBITMQ_URL is required")
    bucket = env("S3_BUCKET_MEDIA", "media")
    work_dir = Path(env("WORK_DIR", "/tmp/work"))
    work_dir.mkdir(parents=True, exist_ok=True)

    params = pika.URLParameters(rabbit)
    # Long face-tracking jobs: keep the connection alive while detection runs on a side thread.
    params.heartbeat = int(env("REFRAME_AMQP_HEARTBEAT", "600"))
    params.blocked_connection_timeout = 300

    while True:
        connection = pika.BlockingConnection(params)
        ch = connection.channel()
        declare_topology(ch)
        ch.basic_qos(prefetch_count=1)
        store = s3_client()
        log.info("worker-reframe started bucket=%s heartbeat=%s", bucket, params.heartbeat)
        try:
            for method, _properties, body in ch.consume("q.reframe", inactivity_timeout=1, auto_ack=False):
                if body is None:
                    continue
                handle(ch, method, body, store, bucket, work_dir)
        except KeyboardInterrupt:
            log.info("worker-reframe stopping")
            try:
                ch.cancel()
            except Exception:  # noqa: BLE001
                pass
            connection.close()
            return
        except Exception:
            log.exception("worker-reframe connection lost; reconnecting")
            try:
                connection.close()
            except Exception:  # noqa: BLE001
                pass


if __name__ == "__main__":
    main()
