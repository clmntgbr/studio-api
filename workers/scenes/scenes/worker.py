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

from scenes import envelope
from scenes.detect import analyze

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("worker-scenes")

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
    ch.exchange_declare(exchange=EXCHANGE_JOBS, exchange_type="direct", durable=True)
    ch.exchange_declare(exchange=EXCHANGE_RESULTS, exchange_type="topic", durable=True)
    ch.exchange_declare(exchange=EXCHANGE_PROGRESS, exchange_type="fanout", durable=True)

    main = "q.scenes"
    retry = "q.scenes.retry"
    dlq = "q.scenes.dlq"
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
    ch.queue_bind(queue=main, exchange=EXCHANGE_JOBS, routing_key="job.scenes")


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


def _run_analyze_keepalive(ch: pika.channel.Channel, video_path: str, threshold: float) -> dict:
    """Run heavy detection off the I/O thread while pumping heartbeats."""
    box: dict = {}

    def target() -> None:
        try:
            box["doc"] = analyze(video_path, content_threshold=threshold)
        except Exception as exc:  # noqa: BLE001 — surfaced to caller
            box["err"] = exc

    thread = threading.Thread(target=target, name="scenes-analyze", daemon=True)
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
    if not media_id or not proxy_key:
        m, out, rk = envelope.new_result(
            "scenes", corr, msg["messageId"], attempt, False, None, "invalid_payload", "mediaId/proxyKey required", False
        )
        publish(ch, EXCHANGE_RESULTS, rk, out, m["messageId"], corr)
        ch.basic_ack(delivery_tag=method.delivery_tag)
        return

    job_dir = work_dir / f"scenes-{media_id}"
    if job_dir.exists():
        shutil.rmtree(job_dir, ignore_errors=True)
    job_dir.mkdir(parents=True, exist_ok=True)

    def progress(pct: float, stage: str, message: str) -> None:
        m, body_p = envelope.new_progress("scenes", corr, msg["messageId"], attempt, pct, stage, message)
        publish(ch, EXCHANGE_PROGRESS, "", body_p, m["messageId"], corr)

    try:
        progress(5, "download", "Downloading proxy")
        local = job_dir / "proxy.mp4"
        key = proxy_key.lstrip("/")
        store.download_file(bucket, key, str(local))

        progress(25, "detect", "Detecting scenes and ad candidates")
        threshold = float(env("SCENES_CONTENT_THRESHOLD", "27"))
        doc = _run_analyze_keepalive(ch, str(local), threshold)

        progress(80, "upload", "Uploading scenes artifacts")
        scenes_key = f"media/{media_id}/analysis/scenes.json"
        ads_key = f"media/{media_id}/analysis/ad_candidates.json"
        scenes_local = job_dir / "scenes.json"
        ads_local = job_dir / "ad_candidates.json"
        scenes_local.write_text(
            json.dumps(
                {
                    "detector": doc["detector"],
                    "scenes": doc["scenes"],
                    "blackIntervals": doc["blackIntervals"],
                    "silenceIntervals": doc["silenceIntervals"],
                },
                separators=(",", ":"),
            ),
            encoding="utf-8",
        )
        ads_local.write_text(json.dumps({"adCandidates": doc["adCandidates"]}, separators=(",", ":")), encoding="utf-8")
        store.upload_file(str(scenes_local), bucket, scenes_key, ExtraArgs={"ContentType": "application/json"})
        store.upload_file(str(ads_local), bucket, ads_key, ExtraArgs={"ContentType": "application/json"})

        progress(100, "done", "Scenes completed")
        m, out, rk = envelope.new_result(
            "scenes",
            corr,
            msg["messageId"],
            attempt,
            True,
            {
                "scenesKey": scenes_key,
                "adCandidatesKey": ads_key,
                "detector": doc["detector"],
                "sceneCount": len(doc["scenes"]),
                "adCandidates": doc["adCandidates"],
            },
        )
        publish(ch, EXCHANGE_RESULTS, rk, out, m["messageId"], corr)
        ch.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as exc:
        log.exception("scenes failed media=%s: %s", media_id, exc)
        m, out, rk = envelope.new_result(
            "scenes",
            corr,
            msg["messageId"],
            attempt,
            False,
            None,
            "scenes_error",
            "scenes processing failed",
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
    # Long TV proxies: keep connection alive while detection runs on a side thread.
    params.heartbeat = int(env("SCENES_AMQP_HEARTBEAT", "600"))
    params.blocked_connection_timeout = 300

    while True:
        connection = pika.BlockingConnection(params)
        ch = connection.channel()
        declare_topology(ch)
        ch.basic_qos(prefetch_count=1)
        store = s3_client()
        log.info("worker-scenes started bucket=%s heartbeat=%s", bucket, params.heartbeat)
        try:
            for method, _properties, body in ch.consume("q.scenes", inactivity_timeout=1, auto_ack=False):
                if body is None:
                    continue
                handle(ch, method, body, store, bucket, work_dir)
        except KeyboardInterrupt:
            log.info("worker-scenes stopping")
            try:
                ch.cancel()
            except Exception:  # noqa: BLE001
                pass
            connection.close()
            return
        except Exception:
            log.exception("worker-scenes connection lost; reconnecting")
            try:
                connection.close()
            except Exception:  # noqa: BLE001
                pass
