# Cloudflare Tunnel Inspector

A developer tool that exposes local HTTP applications through Cloudflare Tunnel (`cloudflared`) and captures incoming HTTP requests and outgoing HTTP responses in a real-time web dashboard.

Built local-first and packaged as a single self-contained executable.

```text
Tunnel Inspector

✓ Application    http://localhost:8000
✓ Inspector      http://localhost:4040
✓ Cloudflared    connected

Public URL
https://abc123.trycloudflare.com

Dashboard
http://localhost:4040

Waiting for requests...
```

---

## Architecture

```text
                         INTERNET
                            │
                            ▼
                    ┌───────────────┐
                    │   Cloudflare  │
                    └───────┬───────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  cloudflared  │
                    └───────┬───────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │      Go Agent       │
                 │                     │
                 │  Tunnel Manager     │
                 │  HTTP Proxy         │
                 │  Request Capture    │
                 │  SQLite (local)     │
                 │  WebSocket Hub      │
                 │  Dashboard Server   │
                 └──────────┬──────────┘
                            │
                            ▼
                    localhost:8000
                      User's App


                 ┌─────────────────────┐
                 │    React + Vite     │
                 │      Dashboard      │
                 └──────────▲──────────┘
                            │
                        WebSocket
                            │
                        Go Agent
```

* **Inspection Proxy (`:4041`)**: Receives public traffic from `cloudflared`, captures request & response streams, redacts sensitive headers, writes to local SQLite, and forwards to the target app (`:8000`).
* **Dashboard Server (`:4040`)**: Serves the embedded React dashboard, REST API, and WebSocket hub. Kept strictly on `localhost` for security.
* **Storage**: Local SQLite database with WAL mode and indexing. No external database required.
* **Real-time updates**: WebSocket hub pushes `request.completed` events instantly to all open dashboard tabs.

---

## Features

- **Zero-Config Quick Tunnel**: Launches `cloudflared tunnel --url ...` and auto-detects the public `https://*.trycloudflare.com` URL.
- **Real-Time Web Dashboard**: Fast React + Tailwind + Zustand UI with split-pane master-detail view.
- **Rich Request Inspection**:
  - Method, path, query parameters table
  - Request headers and response headers
  - JSON body formatter with raw text toggle and word wrap
  - Response status code, size, and duration
- **Secret Redaction**: Automatically redacts sensitive headers (`Authorization`, `Cookie`, `Set-Cookie`, `X-API-Key`, `X-Auth-Token`) before persistence and display.
- **Request Replay**: One-click replay or customized replay with editable payload directly to upstream.
- **Search & Filters**: Instant client-side filtering by HTTP method, status code class (2xx, 3xx, 4xx, 5xx), and URL path search.
- **Offline / Local Mode**: `--no-tunnel` flag allows using the inspection proxy locally without external Cloudflare dependencies.
- **Self-Contained Executable**: Frontend static assets are embedded via Go `//go:embed`.

---

## Prerequisites

1. **`cloudflared`**:
   Install via Homebrew on macOS:
   ```bash
   brew install cloudflared
   ```
   Or download from [Cloudflare](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/).

2. **Go** (1.22+) and **Node.js** (18+) for building from source.

---

## Installation

### Option 1: One-Line Installer (macOS & Linux, Any Shell)

```bash
curl -fsSL https://raw.githubusercontent.com/dela-dels/tunnel-inspector/main/install.sh | sh
```

This installs both `tunnel-inspector` and the shorthand alias `ti` into `~/.local/bin` and automatically configures your shell (`bash`, `zsh`, `fish`, etc.) if needed.

### Option 2: Build & Install from Source

```bash
make install
```

This installs `tunnel-inspector` and `ti` into `~/.local/bin`.

---

## Quick Start

Expose an application running on port 8000:

```bash
ti start --port 8000
# or
tunnel-inspector start --port 8000
```

Or with custom ports:

```bash
./bin/tunnel-inspector start \
  --port 8000 \
  --host localhost \
  --dashboard-port 4040 \
  --proxy-port 4041
```

---

## CLI Options

| Flag | Default | Description |
|------|---------|-------------|
| `--port`, `-p` | `8000` | Target upstream application port |
| `--host` | `localhost` | Target upstream application host |
| `--dashboard-port` | `4040` | Inspector web dashboard port |
| `--proxy-port` | `4041` | Inspection proxy port for cloudflared |
| `--db` | `~/.tunnel-inspector/requests.db` | SQLite database file location |
| `--max-body` | `1048576` (1 MB) | Maximum request/response body capture size |
| `--no-tunnel` | `false` | Run local proxy and dashboard without cloudflared |

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/status` | Current status, tunnel URL, and total request count |
| `GET` | `/api/tunnel` | Active tunnel metadata and target URL |
| `GET` | `/api/requests?page=1&limit=50` | Paginated list of request summaries |
| `GET` | `/api/requests/:id` | Full request & response details including bodies |
| `DELETE` | `/api/requests` | Clear all stored requests |
| `DELETE` | `/api/requests/:id` | Delete a single stored request |
| `POST` | `/api/requests/:id/replay` | Replay original or modified request to upstream |
| `GET` | `/ws` | WebSocket connection for real-time events |

---

## Local Development Mode

During frontend development, run the Go agent and Vite dev server concurrently:

1. **Start the Go Agent**:
   ```bash
   go run ./cmd/tunnel-inspector start --port 8000 --no-tunnel
   ```

2. **Start the Vite Dev Server**:
   ```bash
   cd web
   npm run dev
   ```
   Open [http://localhost:5173](http://localhost:5173). Vite proxies `/api` and `/ws` to `http://localhost:4040`.

---

## Running Tests

Run all unit tests:

```bash
make test
```
