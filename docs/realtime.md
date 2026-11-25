# Realtime

## Overview

**Centrifugo** delivers WebSocket updates. The API does not publish realtime events from HTTP handlers — the preferred path is:

```
outbox → RabbitMQ → worker handler → Centrifugo publisher
```

Realtime event `type` uses `entity.action` (e.g. `user.created`), not the versioned domain type (`user.created.v1`).

Publish is **best-effort**: a Centrifugo outage is logged and must not retry/DLQ domain events.

### Local / Docker config

Namespaces (`users`, `projects`) live in committed `centrifugo/config.json`, mounted as a **single file** (`./centrifugo/config.json:/centrifugo/config.json:ro`). Do not mount the whole directory — under Docker Desktop/WSL that often appears empty inside the container, Centrifugo starts with no namespaces, and publishes fail with `unknown channel: 102`. Secrets/origins stay in env (`CENTRIFUGO_API_KEY`, `CENTRIFUGO_TOKEN_HMAC_SECRET_KEY`, `CENTRIFUGO_ALLOWED_ORIGINS`).

## HTTP routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/realtime/connection` | Connection token / channel / WS URL |

Requires authentication.

## Event examples

| Realtime type | When |
|---------------|------|
| `user.created` | User created (Clerk webhook / signup) |
| `user.updated` | User updated |
| `user.deleted` | User deleted |

## Code map

| Layer | Location |
|-------|----------|
| HTTP | `internal/interfaces/http/handler/realtime_handler.go` |
| Ports | `internal/domain/port/realtime.go` |
| Helpers | `internal/application/realtime/` |
| Adapter | `internal/infrastructure/centrifugo/` |
| Event publish | `internal/application/event/user/publish_realtime.go` |
