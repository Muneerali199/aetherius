# Aetherius — Decentralized AI Cloud Platform Architecture

## Overview

Aetherius is a decentralized AI cloud platform that connects GPU/node providers with developers
who need compute. The architecture follows cloud-native best practices, microservices patterns,
and enterprise-grade security principles.

## Technology Stack & Justifications

| Technology | Purpose | Why |
|---|---|---|
| **Go 1.22+** | All microservices | Superior concurrency (goroutines), fast compilation, small binaries, excellent stdlib. Ideal for high-throughput infrastructure services. |
| **PostgreSQL 16** | Primary database | Mature, ACID-compliant, excellent JSON support, powerful indexing (GIN, BRIN), logical replication for read replicas. |
| **Redis 7** | Caching + queues | Sub-millisecond latency, pub/sub, streams for reliable messaging, built-in rate limiting with sliding window. |
| **RabbitMQ** | Message broker | AMQP 0-9-1 guarantees at-least-once delivery, dead-letter exchanges, delayed queues. Superior to Kafka for job-oriented workloads where every message must be processed exactly once. |
| **MinIO** | Object storage | S3-compatible, self-hosted, erasure coding for durability, supports versioning, encryption at rest. No vendor lock-in. |
| **gRPC** | Inter-service communication | Strongly typed (protobuf), bidirectional streaming, multiplexing over HTTP/2. 10x faster than REST for internal calls. |
| **Connect-Go** | gRPC implementation | Pure Go gRPC client/server, no CGo, works with net/http. Better than standard gRPC-Go for simplicity. |
| **Envoy** | API Gateway + Sidecar | L7 routing, rate limiting, circuit breaking, observability, hot reload. Battle-tested at Lyft/Google. |
| **Docker + containerd** | Container runtime | OCI standard, GPU passthrough (nvidia-container-toolkit), cgroups v2 isolation. |
| **Kubernetes** | Orchestration | Self-healing, horizontal scaling, service discovery, rolling updates. The standard for production container orchestration. |
| **Prometheus + Grafana** | Monitoring + Alerting | Pull-based metrics, powerful query language (PromQL), wide ecosystem, battle-tested at scale. |
| **OpenTelemetry** | Distributed tracing | Vendor-neutral, W3C trace context propagation, auto-instrumentation for Go. |
| **Ory Kratos** | Identity management | Open-source, passwordless, WebAuthn, passkeys, MFA, social sign-in. Self-hosted, no vendor lock-in. |
| **Casbin** | Authorization | RBAC, ABAC, multi-tenant policies. Sub-millisecond policy evaluation. Used by Docker, Intel. |
| **HashiCorp Vault** | Secrets management | Dynamic secrets, encryption-as-a-service, audit logging, auto-unseal. |
| **Terraform** | Infrastructure as Code | Declarative, immutable infrastructure. State management, plan/apply workflow. |
| **Helm** | K8s package manager | Chart templating, dependency management, release history. |
| **ArgoCD** | GitOps deployment | Declarative, pull-based, automatic sync with git, multi-cluster support. |
| **Prometheus Alertmanager** | Alerting | Deduplication, silencing, inhibition, routing to Slack/PagerDuty/Opsgenie. |

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Internet                            │
├─────────────────────────────────────────────────────────────┤
│                      Envoy API Gateway                      │
│              Rate Limiting · Auth · TLS · Routing            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐   │
│  │   Auth    │ │   User    │ │   Node    │ │Scheduler  │   │
│  │  Service  │ │  Service  │ │  Service  │ │  Service  │   │
│  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └─────┬─────┘   │
│        │              │              │              │        │
│  ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐   │
│  │Deployment │ │Marketplace│ │ Billing   │ │ Storage   │   │
│  │  Service  │ │  Service  │ │  Service  │ │  Service  │   │
│  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └─────┬─────┘   │
│        │              │              │              │        │
│  ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐   │
│  │Networking│ │Monitoring │ │  AI       │ │Notif.     │   │
│  │  Service  │ │  Service  │ │  Service  │ │  Service  │   │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘   │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                    Message Broker (RabbitMQ)                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  PostgreSQL (Primary + Read Replicas)                        │
│  Redis (Cache + Sessions + Rate Limit)                       │
│  MinIO (Object Storage)                                      │
│  Vault (Secrets)                                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Service Diagram

### Authentication Service
- Registration, login, OAuth (Google, GitHub)
- JWT + refresh token issuance
- MFA (TOTP, WebAuthn)
- Session management
- Email verification flow

### User Service
- User profiles
- Organization management
- Team/invitation management
- RBAC (Casbin policies)
- API key management
- SSH key management
- Activity timeline

### Node Service
- Node registration & verification
- Heartbeat processing
- Hardware fingerprinting
- Node status (online/offline/maintenance)
- Provider reputation scoring
- Earnings tracking

### Scheduler Service
- K8s-inspired scheduling algorithm
- Multi-dimensional bin packing (GPU, VRAM, RAM, CPU)
- Price-aware scheduling
- Latency optimization (geo-proximity)
- Preemption and priority queues
- Automatic failover and job migration
- Autoscaling based on demand

### Deployment Service
- Container lifecycle management
- Docker/OCI image pulling
- GPU passthrough (nvidia-container-toolkit)
- Health checks and restart policies
- Log streaming
- Shell access (WebSocket)
- Private registry auth

### Marketplace Service
- GPU listing and search
- Full-text search across GPU, VRAM, region, price
- Provider reputation scores
- Latency benchmarking
- Pricing engine

### Billing Service
- Wallet system (credits)
- Usage-based billing (per-second granularity)
- Invoicing and receipts
- Subscription management
- Payout processing for providers
- Tax calculation
- Coupon/credit system
- Revenue sharing (platform cut)

### Storage Service
- Object storage (MinIO gateway)
- Persistent volume management
- Snapshot and backup
- Versioning
- CDN integration (edge caching)

### Networking Service
- Private network (WireGuard mesh)
- Public IP assignment
- Firewall rules
- Port forwarding
- Load balancer management
- TLS certificate provisioning (Let's Encrypt)
- Reverse proxy (per-deployment)

### Monitoring Service
- Prometheus metric collection
- GPU utilization metrics
- Node health scoring
- Distributed tracing (OpenTelemetry)
- Centralized logging (Loki/OpenSearch)
- Alert evaluation and routing

### AI Service
- One-click model deployment
- Inference endpoint management
- Model registry with versioning
- LoRA adapter management
- Hugging Face integration
- vLLM and TensorRT serving
- OpenAI-compatible API
- Autoscaling for inference

### Notification Service
- Email (SendGrid/SES)
- WebSocket real-time push
- Webhooks
- In-app notification center
- Template engine
- Preference management

## Database Schema

### Core Tables (with relationships)

```sql
-- Users & Auth
users: id, email, password_hash, display_name, avatar_url, email_verified,
       mfa_enabled, mfa_secret, mfa_backup_codes[], totp_secret, webauthn_credentials[],
       created_at, updated_at, deleted_at

sessions: id, user_id, refresh_token_hash, ip_address, user_agent,
          device_info, expires_at, created_at

oauth_accounts: id, user_id, provider (google|github), provider_user_id,
                access_token, refresh_token, token_expires_at

-- Organizations & RBAC
organizations: id, name, slug, owner_id, billing_email, plan_tier, created_at

organization_members: id, org_id, user_id, role (owner|admin|member|viewer),
                      joined_at

roles: id, org_id, name, description, is_system
permissions: id, role_id, resource, action, effect (allow|deny)
role_assignments: id, user_id, role_id, org_id, scope

-- Projects & Environments
projects: id, org_id, name, description, created_at
environments: id, project_id, name (production|staging|development), config

-- Node Provider
nodes: id, provider_id, status (pending|active|paused|maintenance|offline|banned),
       hardware_fingerprint, benchmark_score, reputation_score,
       total_gpu, available_gpu, total_vram_gb, available_vram_gb,
       total_ram_gb, available_ram_gb, total_disk_gb, available_disk_gb,
       cpu_model, cpu_cores, gpu_models[], network_speed_mbps,
       public_ip, region, country, city, latitude, longitude,
       cuda_version, rocm_version, docker_version,
       os_name, os_version, kernel_version, agent_version,
       first_seen, last_heartbeat, created_at

gpu_units: id, node_id, gpu_index, model, vram_gb, vram_type,
           cuda_cores, tensor_cores, clock_speed_mhz, power_limit_w,
           pci_generation, pci_speed_gbps, uuid, status

node_heartbeats: id, node_id, status, gpu_utilization[], gpu_temp[],
                 vram_used[], cpu_utilization, ram_used_gb, disk_used_gb,
                 network_rx_bytes, network_tx_bytes, load_average,
                 uptime_seconds, reported_at

node_verifications: id, node_id, benchmark_type (gpu|cpu|vram|disk|network|stress),
                    score, duration_ms, details (jsonb), passed, attempted_at

-- Marketplace
gpu_listings: id, node_id, gpu_unit_id, price_per_hour_usd, status,
              min_duration_hours, max_duration_hours, available,
              created_at, updated_at

-- Deployments / Jobs
deployments: id, org_id, project_id, environment_id, user_id, status,
             image, tag, command[], entrypoint, working_dir,
             gpu_count, vram_gb, ram_gb, cpu_count, disk_gb,
             port_mappings[], env_vars (encrypted), secrets (encrypted),
             volumes[], restart_policy, max_retries,
             created_at, started_at, finished_at

deployment_nodes: id, deployment_id, node_id, gpu_unit_ids[],
                  status, container_id, container_name,
                  assigned_at, started_at

containers: id, deployment_id, node_id, docker_id, image, status,
            exit_code, started_at, finished_at

-- Billing
wallets: id, user_id, org_id, balance_cents, currency, created_at

transactions: id, wallet_id, type (deposit|withdrawal|charge|refund|payout),
              amount_cents, balance_before_cents, balance_after_cents,
              description, reference_type, reference_id, created_at

usage_records: id, deployment_id, node_id, gpu_unit_id,
               start_time, end_time, duration_seconds,
               rate_per_hour_cents, total_cents, billed

invoices: id, org_id, number, status (draft|open|paid|overdue|cancelled),
          period_start, period_end, total_cents, paid_at

payouts: id, provider_id, amount_cents, status (pending|processing|completed|failed),
         payment_method, processed_at

-- Storage
volumes: id, deployment_id, name, size_gb, storage_class,
         mount_path, snapshot_schedule, created_at

snapshots: id, volume_id, size_gb, status, created_at
backups: id, volume_id, size_gb, encrypted, storage_path, created_at

-- Networking
networks: id, org_id, name, cidr_block, region, created_at
firewall_rules: id, network_id, direction, protocol, port_range,
                source_cidr, target_cidr, action, priority
load_balancers: id, org_id, name, dns_name, protocol, port, target_port,
                health_check_path, ssl_cert_id

-- Monitoring
metrics: timestamp, node_id, deployment_id, metric_name, metric_value, labels (jsonb)
alerts: id, org_id, name, metric, condition, threshold, severity,
        cooldown, enabled, last_fired_at

-- Audit & Security
audit_logs: id, timestamp, user_id, org_id, action, resource_type,
            resource_id, details (jsonb), ip_address, user_agent

api_keys: id, user_id, org_id, name, key_hash, key_prefix,
          scopes[], expires_at, last_used_at, created_at

-- AI / ML
models: id, org_id, name, description, framework, source (huggingface|custom),
        model_id (HF), base_model, task, license, size_bytes, status

model_versions: id, model_id, version, uri, format (safetensors|gguf|pt),
                config (jsonb), metrics (jsonb), created_at

inference_endpoints: id, org_id, model_id, name, status,
                     endpoint_url, runtime (vllm|tensorrt|ollama),
                     gpu_type, min_replicas, max_replicas,
                     max_tokens, temperature, created_at

-- Notifications
notifications: id, user_id, org_id, type, title, body,
               data (jsonb), read, read_at, created_at

webhooks: id, org_id, url, secret, events[], enabled, last_success_at

-- Support
support_tickets: id, user_id, org_id, subject, body, status, priority,
                 category, assigned_to, created_at, resolved_at

ticket_messages: id, ticket_id, user_id, body, attachments[], created_at

-- Activity Timeline
activities: id, user_id, org_id, project_id, type, description,
            metadata (jsonb), created_at
```

## API Structure

### External REST API (via Envoy)
```
POST   /v1/auth/register
POST   /v1/auth/login
POST   /v1/auth/refresh
POST   /v1/auth/logout
POST   /v1/auth/mfa/setup
POST   /v1/auth/mfa/verify
POST   /v1/auth/password/reset
POST   /v1/auth/oauth/{provider}

GET    /v1/users/me
PUT    /v1/users/me
GET    /v1/orgs
POST   /v1/orgs
GET    /v1/orgs/:id
PUT    /v1/orgs/:id
POST   /v1/orgs/:id/members
DELETE /v1/orgs/:id/members/:userId

GET    /v1/projects
POST   /v1/projects
GET    /v1/projects/:id
POST   /v1/projects/:id/deployments
GET    /v1/deployments
GET    /v1/deployments/:id
DELETE /v1/deployments/:id
POST   /v1/deployments/:id/restart
POST   /v1/deployments/:id/stop

GET    /v1/marketplace/gpus
GET    /v1/marketplace/gpus/:id
POST   /v1/marketplace/orders

GET    /v1/nodes
GET    /v1/nodes/:id
POST   /v1/nodes/:id/pause
POST   /v1/nodes/:id/resume
POST   /v1/nodes/:id/maintenance

GET    /v1/billing/wallet
GET    /v1/billing/invoices
POST   /v1/billing/deposit
GET    /v1/billing/usage

GET    /v1/storage/volumes
POST   /v1/storage/volumes
DELETE /v1/storage/volumes/:id
POST   /v1/storage/volumes/:id/snapshot

GET    /v1/networks
POST   /v1/networks
POST   /v1/networks/:id/rules

GET    /v1/models
POST   /v1/models
POST   /v1/models/:id/deploy

GET    /v1/notifications
POST   /v1/notifications/read

GET    /v1/support/tickets
POST   /v1/support/tickets
POST   /v1/support/tickets/:id/messages

GET    /v1/activities
GET    /v1/analytics/usage
```

### Internal gRPC Services (protobuf)

```protobuf
service AuthService {
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc ExchangeOAuth(ExchangeOAuthRequest) returns (Tokens);
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc RevokeSession(RevokeSessionRequest) returns (Empty);
}

service NodeService {
  rpc RegisterNode(RegisterNodeRequest) returns (Node);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc GetAvailableNodes(AvailableNodesRequest) returns (AvailableNodesResponse);
  rpc AssignDeployment(AssignDeploymentRequest) returns (AssignDeploymentResponse);
  rpc ReportMetrics(MetricsReport) returns (Empty);
}

service SchedulerService {
  rpc ScheduleDeployment(ScheduleRequest) returns (ScheduleResponse);
  rpc GetJobStatus(JobStatusRequest) returns (JobStatusResponse);
  rpc CancelJob(CancelJobRequest) returns (Empty);
}

service DeploymentService {
  rpc CreateDeployment(CreateDeploymentRequest) returns (Deployment);
  rpc StopDeployment(StopDeploymentRequest) returns (Empty);
  rpc GetDeploymentLogs(LogRequest) returns (stream LogLine);
  rpc GetDeploymentStatus(StatusRequest) returns (StatusResponse);
}
```

## Event Flow

### Deployment Request Flow
```
1. User POST /v1/deployments → Envoy → Auth middleware → Deployment Service
2. Deployment Service validates request → publishes "deployment.requested" event
3. Scheduler Service consumes event → queries available nodes from Node Service
4. Scheduler scores nodes (price, latency, GPU, reliability) → selects best node
5. Scheduler publishes "deployment.scheduled" event with node assignment
6. Deployment Service receives → publishes "deployment.started" event
7. Node Agent polls or receives push → pulls container image
8. Node Agent starts container with GPU passthrough
9. Node Agent sends heartbeat with container status
10. Deployment Service updates status → publishes "deployment.running"
11. User receives WebSocket notification → sees deployment live
```

### Node Heartbeat Flow
```
1. Node Agent sends Heartbeat (every 5s) via gRPC to Node Service
2. Node Service updates last_heartbeat, node status, GPU metrics
3. Node Service publishes "node.heartbeat.received" event
4. Monitoring Service consumes → evaluates alert conditions
5. If node fails to heartbeat for 30s → Node Service marks as offline
6. Node Service publishes "node.offline" event
7. Scheduler receives → reschedules affected deployments using failover
```

## Authentication Flow

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │     │  Envoy   │     │  Auth    │     │   DB     │
│          │     │ Gateway  │     │ Service  │     │          │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                 │                 │                 │
     │ POST /v1/auth/  │                 │                 │
     │   login         │                 │                 │
     ├─────────────────►                 │                 │
     │                 │ Validate req    │                 │
     │                 ├────────────────►│                 │
     │                 │                 │ Query user      │
     │                 │                 ├────────────────►│
     │                 │                 │◄────────────────┤
     │                 │                 │                 │
     │                 │                 │ Verify bcrypt   │
     │                 │                 │ Check MFA       │
     │                 │                 │                 │
     │  MFA required   │                 │                 │
     │◄────────────────┤                 │                 │
     │                 │                 │                 │
     │ POST /v1/auth/  │                 │                 │
     │   mfa/verify    │                 │                 │
     ├─────────────────►                 │                 │
     │                 │                 │ Verify TOTP     │
     │                 │                 │                 │
     │  Tokens (AT+RT) │                 │                 │
     │◄────────────────┤                 │                 │
     │                 │                 │ Store session   │
     │                 │                 ├────────────────►│
     │                 │                 │                 │
```

## Scheduler Design

The scheduler is K8s-inspired with multi-dimensional bin packing:

### Scoring Algorithm

```
Score(node) = w1 * price_factor + w2 * latency_factor + w3 * reliability_factor
           + w4 * gpu_match + w5 * utilization_balance

price_factor = 1 - (node_price / max_price)
latency_factor = 1 - (estimated_latency / max_tolerable_latency)
reliability_factor = node_uptime_percentage * historical_completion_rate
gpu_match = cosine_similarity(requested_gpu, available_gpu)
utilization_balance = 1 - abs(0.5 - current_utilization)
```

### Constraints (hard filters)
- GPU count >= requested
- VRAM >= requested + 10% buffer
- RAM >= requested
- Disk >= requested
- Region matches or latency < threshold
- Node not in maintenance/offline
- Provider not banned/flagged

### Queue Priority
```
1. Critical (production deployments)   — immediate scheduling
2. High     (paid jobs)                — within 30s
3. Normal   (standard jobs)            — within 5min
4. Low      (batch/background jobs)    — when resources available
```

## Security Architecture

```
┌─────────────────────────────────────────┐
│           Internet                       │
├─────────────────────────────────────────┤
│     DDoS Protection (Cloudflare)         │
├─────────────────────────────────────────┤
│     WAF (rate limiting, SQLi, XSS)       │
├─────────────────────────────────────────┤
│     Envoy API Gateway                    │
│     · TLS termination                    │
│     · JWT validation                     │
│     · Rate limiting (Redis sliding log) │
│     · CORS enforcement                  │
│     · Request size limits               │
├─────────────────────────────────────────┤
│     Service Mesh (Envoy sidecars)        │
│     · mTLS between services             │
│     · Circuit breaking                   │
│     · Retry budgets                      │
├─────────────────────────────────────────┤
│     Zero Trust Principles                │
│     · Every request authenticated       │
│     · Least privilege RBAC              │
│     · All traffic encrypted (mTLS)      │
│     · Audit logging everywhere          │
├─────────────────────────────────────────┤
│     Container Isolation                 │
│     · cgroups v2 + seccomp              │
│     · AppArmor profiles                 │
│     · Read-only root filesystem         │
│     · No privileged containers          │
│     · Image scanning (Trivy)            │
│     · Runtime security (Falco)          │
└─────────────────────────────────────────┘
```

## Folder Structure

```
decentralized-ai-cloud/
├── ARCHITECTURE.md
├── docker-compose.yml
├── Makefile
├── go.work
├── api-gateway/
│   ├── envoy.yaml
│   └── Dockerfile
├── services/
│   ├── auth/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   ├── repository/
│   │   │   └── model/
│   │   ├── migrations/
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── node/
│   ├── scheduler/
│   ├── deployment/
│   ├── marketplace/
│   ├── billing/
│   ├── storage/
│   ├── networking/
│   ├── monitoring/
│   ├── notification/
│   ├── support/
│   ├── ai/
│   └── user/
├── agent/
│   ├── cmd/agent/main.go
│   ├── internal/
│   │   ├── hardware/
│   │   ├── container/
│   │   ├── heartbeat/
│   │   └── updater/
│   ├── install.sh
│   ├── Dockerfile
│   └── go.mod
├── pkg/
│   ├── types/       # Shared protobuf types
│   ├── auth/        # JWT parsing, middleware
│   ├── middleware/  # gRPC interceptors
│   ├── queue/       # RabbitMQ client
│   ├── database/    # PostgreSQL pool
│   └── crypto/     # Encryption helpers
├── proto/
│   ├── auth/
│   │   └── v1/auth.proto
│   ├── node/
│   │   └── v1/node.proto
│   ├── scheduler/
│   │   └── v1/scheduler.proto
│   ├── deployment/
│   │   └── v1/deployment.proto
│   ├── billing/
│   │   └── v1/billing.proto
│   └── common/
│       └── v1/common.proto
├── db/
│   ├── migrations/
│   └── seed/
├── k8s/
│   ├── base/
│   │   ├── kustomization.yaml
│   │   ├── namespace.yaml
│   │   ├── postgres/
│   │   ├── redis/
│   │   ├── rabbitmq/
│   │   ├── minio/
│   │   └── services/
│   └── overlays/
│       ├── dev/
│       ├── staging/
│       └── production/
├── docs/
│   ├── api/        # OpenAPI/Swagger
│   └── guides/
├── scripts/
│   ├── ci.sh
│   ├── migrate.sh
│   └── deploy.sh
└── .github/
    └── workflows/
        ├── ci.yml
        └── cd.yml
```

## Development Roadmap

### Phase 1 (Weeks 1-4): Foundation
- Set up monorepo with Go workspace
- Database migrations framework (golang-migrate)
- Shared libraries (auth, queue, database)
- Authentication service (register, login, JWT)
- API Gateway (Envoy config)
- Docker Compose dev environment
- CI pipeline

### Phase 2 (Weeks 5-8): Core Services
- User service (organizations, teams, RBAC)
- Node service (registration, heartbeat)
- Node agent (hardware detection, health monitoring)
- Basic scheduler (first-fit + price)
- Container deployment service
- MinIO storage integration
- PostgreSQL with read replicas

### Phase 3 (Weeks 9-12): Billing & Marketplace
- Wallet and transaction system
- Usage-based billing (per-second)
- Invoice generation
- Provider payouts
- GPU marketplace listing & search
- Pricing engine

### Phase 4 (Weeks 13-16): Advanced Features
- Advanced scheduler (multi-dimensional bin packing)
- Auto-scaling
- Job migration and failover
- Networking service (WireGuard, public IPs)
- Monitoring and alerting (Prometheus)
- Centralized logging (OpenSearch/Loki)

### Phase 5 (Weeks 17-20): AI & Developer Platform
- Model registry and versioning
- One-click inference deployment
- OpenAI-compatible API
- SDKs (Node.js, Python, Go)
- CLI tool
- Terraform provider
- Webhooks system

### Phase 6 (Weeks 21-24): Enterprise & Scale
- Multi-region support
- CDN integration
- SOC 2 compliance prep
- Penetration testing
- Load testing (1000+ nodes)
- Chaos engineering
- Disaster recovery drills

## Monitoring Strategy

```yaml
Metrics (Prometheus):
  - Service: request_rate, latency_p50/p95/p99, error_rate, goroutines
  - Node: gpu_utilization, gpu_temp, vram_usage, cpu_usage, ram_usage
  - Business: active_deployments, total_users, revenue, node_count

Alerting (Alertmanager):
  - Critical: Node offline > 5min, deployment crash loop, billing failure
  - Warning: GPU temp > 85°C, high error rate > 5%, disk > 80%
  - Info: New node registered, deployment completed

Logging (OpenSearch/Loki):
  - Structured JSON logging (all services)
  - Log levels: debug, info, warn, error, fatal
  - Correlation ID across services (W3C trace context)
  - Retention: 7d hot, 30d warm, 90d cold

Tracing (OpenTelemetry + Jaeger):
  - All gRPC calls traced
  - Deployment lifecycle traced end-to-end
  - Sample rate: 10% production, 100% development
```

## Node Communication Protocol

```
Agent → Server:  gRPC (TLS, mTLS)
  - RegisterNode (hardware info, fingerprint)
  - Heartbeat (utilization, health, running containers)
  - ReportMetrics (GPU, CPU, memory, network)
  - PullJob (check for assigned deployments)
  - ReportJobStatus (container state, exit code)

Server → Agent:  gRPC (push via bidirectional stream)
  - AssignDeployment (image, env, GPU config, ports)
  - StopDeployment (container_id, grace_period)
  - UpdateAgent (new version URL, checksum)
  - RestartAgent

Security:
  - mTLS with client certificates
  - Certificate rotation every 30 days
  - Hardware-backed key storage (TPM where available)
  - Agent binary signed and verified
  - Secure boot attestation (future)
```

## Billing Architecture

```
Usage Record Flow:
1. Deployment starts → Billing service creates usage_record (open)
2. Every 60s → heartbeat includes running container duration
3. Node Service → publishes "node.heartbeat" with runtime_seconds
4. Billing Service → updates usage_record.duration_seconds
5. Deployment stops → Billing service closes usage_record
6. Billing service → calculates total = duration * rate
7. Deducts from wallet (user) or adds to invoice
8. Credits provider wallet (platform takes cut)

Rate Calculation:
  total_cents = duration_seconds * (rate_per_hour_cents / 3600)
  platform_cut = total_cents * platform_fee_percentage (default 15%)
  provider_earnings = total_cents - platform_cut
```

## Disaster Recovery

```
Backup Strategy:
- PostgreSQL: WAL archiving (every 5min) + full backup (daily)
- MinIO: Cross-region replication
- Vault: Auto-unseal with KMS, snapshot every hour

Recovery:
- RPO: 5 minutes (WAL logs)
- RTO: 1 hour (full restore from backup)
- Multi-region active-passive for critical services

Degradation:
- Read replicas serve during primary failover
- Queue persists messages if services are down
- API Gateway circuit-breaks failing services
- Node agent caches heartbeats if server unreachable (max 5min)
```
