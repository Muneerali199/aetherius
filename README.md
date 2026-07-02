# Aetherius — Decentralized AI Cloud Platform

> Orchestrates idle GPU compute resources across the globe into a single, self-managing infrastructure layer.

## Architecture

```
frontend/     → React 19 + Three.js + Tailwind CSS (Vite)
services/     → 13 Go microservices (auth, node, scheduler, deployment, ...)
agent/        → Go node agent (hardware detection, heartbeat, container mgmt)
pkg/          → Shared Go libraries (JWT, PostgreSQL, RabbitMQ)
proto/        → Protobuf service definitions
api-gateway/  → Envoy config (routing, auth, rate limiting)
k8s/          → Kubernetes manifests (Kustomize)
```

## Quick Start

### Backend (Docker Compose)

```bash
cp .env.example .env
docker compose up -d
```

### Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

### Development (without Docker)

Each service runs standalone with environment variables:

```bash
# Start a single service
cd services/auth
go run ./cmd/server

# Start all services
make dev
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| API Gateway | 8080 | Envoy reverse proxy |
| Auth | 8081 | JWT, OAuth, MFA, RBAC |
| Node | 8082 | GPU registration, heartbeats, resource tracking |
| Scheduler | 8083 | K8s-inspired multi-dimensional bin packing |
| User | 8084 | Orgs, teams, API keys, activity timeline |
| Deployment | 8085 | Container lifecycle, GPU passthrough |
| Marketplace | 8086 | GPU listing, search, pricing |
| Billing | 8087 | Wallet, per-second usage, invoices, payouts |
| Storage | 8088 | Volumes, snapshots, MinIO integration |
| Networking | 8089 | Private networks, firewalls, public IPs |
| AI | 8090 | Model registry, inference (vLLM/TensorRT) |
| Notification | 8091 | Email, WebSocket push, webhooks |
| Support | 8092 | Tickets system |
| Monitoring | 8093 | Prometheus metrics, health dashboards |

## Tech Stack

- **Backend**: Go 1.22+, chi router, pgx (PostgreSQL), RabbitMQ, Redis
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS, Three.js
- **Gateway**: Envoy Proxy
- **Infrastructure**: Docker, Kubernetes, Terraform, ArgoCD
- **Observability**: Prometheus, Grafana, OpenTelemetry
- **Security**: JWT, mTLS, Vault, Casbin RBAC

## License

MIT
