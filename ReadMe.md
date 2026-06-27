# go-load-balancer

A production-grade HTTP load balancer built from scratch in Go. Implements four routing algorithms, active health checking, per-IP rate limiting, connection pooling, consistent hashing, and Prometheus observability — all without any third-party load balancing libraries.

Built as a deep-dive into Go systems programming and distributed systems fundamentals, with every engineering decision made deliberately.

---

## Features

- **Four routing algorithms** — Round Robin, Least Connections, Weighted, Consistent Hashing
- **Active health checking** — background goroutine pings backends on a configurable interval, automatically removes dead backends and recovers them when they come back
- **Per-IP rate limiting** — token bucket algorithm via middleware, fully composable
- **Connection pooling** — tuned `http.Transport` with configurable idle connection limits
- **Consistent hashing** — FNV32 hash ring with 150 virtual nodes per backend for uniform distribution and minimal redistribution on backend changes
- **Prometheus metrics** — request rate, p99 latency, and active connections per backend exposed at `/metrics`
- **Grafana dashboards** — Docker Compose setup for full local observability stack
- **Graceful shutdown** — drains in-flight requests on SIGTERM/SIGINT before exiting
- **YAML config** — all parameters user-configurable, with sane validated defaults

---

## Architecture

```
                         ┌─────────────────────────────────────┐
                         │           Load Balancer              │
                         │                                      │
Client ──HTTP──►  :8080  │  RateLimiter → Proxy → Balancer     │ ──► Backend :9090
                         │                   ↕                  │ ──► Backend :9091
                         │             HealthChecker            │ ──► Backend :909N
                         │                                      │
                         │  /metrics ──► Prometheus ──► Grafana │
                         └─────────────────────────────────────┘
```

### Package structure

```
load-balancer/
├── cmd/load-balancer/
│   └── main.go              # wires everything together, handles OS signals
├── internal/
│   ├── config/              # Config and BackendConfig structs
│   ├── parser/              # YAML → Config, with validation
│   ├── proxy/               # http.Handler, owns request lifecycle
│   ├── balancer/            # Balancer interface + 4 implementations
│   │   ├── roundrobin.go
│   │   ├── leastconn.go
│   │   ├── weighted.go
│   │   └── consistent.go
│   ├── health/              # Background health checker
│   ├── middleware/          # Rate limiter middleware
│   ├── metrics/             # Prometheus recorder
│   └── server/              # TCP server + /metrics endpoint
└── deploy/
    ├── docker-compose.yml   # Prometheus + Grafana
    └── prometheus.yml       # scrape config
```

Every package has a single responsibility. The `Balancer` interface decouples routing logic from HTTP concerns — algorithms implement `Next(*http.Request)`, know nothing about `ResponseWriter`.

---

## Routing algorithms

### Round Robin
Distributes requests sequentially across all active backends using an atomic counter. Lock-free — `sync/atomic` on a single `uint64` means zero mutex contention under concurrent load.

```
Request 1 → backend 0
Request 2 → backend 1
Request 3 → backend 0
```

### Least Connections
Routes each request to the backend with the fewest active connections. Uses a `sync.Mutex` to protect connection counts — increments on `Next()`, decrements via a returned `done()` closure that is always called via `defer`, guaranteeing cleanup even on panics or client disconnects.

```go
target, backend, done := balancer.Next(r)
defer done()  // always decrements, no matter what
```

### Weighted
Expands the backend list by weight at construction time — a backend with weight 3 gets 3 slots in the rotation. Round robin then runs over the expanded list. Same proxy pointer is reused per backend, so memory usage is O(backends) not O(total weight).

### Consistent Hashing
Builds an FNV32 hash ring with 150 virtual nodes per backend. Client requests are routed by hashing `r.RemoteAddr` — same client always hits the same backend. Binary search (`sort.Search`) gives O(log N) lookup per request.

When a backend goes down, only ~1/N of clients are rerouted — versus 100% rerouting with naive `hash % N` approaches. Critical for cache affinity use cases.

---

## Health checking

A background goroutine pings each backend's `/health` endpoint (or `/` if not available) on a configurable interval. Uses a dedicated `*http.Client` with a configurable timeout — never hangs on unresponsive backends.

```
health_interval: 10s   # how often to ping
health_timeout:  3s    # must be < health_interval (validated at startup)
```

Health state is maintained as `[]bool` protected by `sync.RWMutex` — reads (every request) take a read lock, writes (every health check) take a write lock. Multiple concurrent requests never block each other.

Dead backends are automatically re-added to rotation when they recover — zero manual intervention required.

---

## Rate limiting

Token bucket per client IP via `golang.org/x/time/rate`. Implemented as middleware — wraps any `http.Handler`, completely decoupled from routing logic.

```
rate_limit: 100   # requests per second per IP
rate_burst: 20    # initial bucket size
```

Client IP extracted via `net.SplitHostPort` — handles IPv6 correctly. Returns `429 Too Many Requests` when limit is exceeded.

State is a `map[string]*rate.Limiter` protected by a single `sync.Mutex`. Mutex is held only for map lookup/insert (nanoseconds) — the actual `Allow()` check runs outside the lock since `rate.Limiter` is internally thread-safe.

---

## Observability

Three metrics exposed at `/metrics` in Prometheus format:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `requests_total` | Counter | `backend`, `status` | Total requests proxied |
| `request_duration_seconds` | Histogram | `backend` | Request duration with default buckets |
| `active_connections` | Gauge | `backend` | Current in-flight requests |

Status code captured via a `responseWriter` wrapper that embeds `http.ResponseWriter` and intercepts `WriteHeader` — only one method overridden, everything else inherited via embedding.

### Running the observability stack

```bash
cd deploy
docker compose up -d
```

- Prometheus UI: `http://localhost:9091`
- Grafana: `http://localhost:3000` (admin/admin)

Add Prometheus data source: `http://prometheus:9090`

Useful PromQL queries:

```promql
# requests per second per backend
rate(requests_total[1m])

# p99 latency
histogram_quantile(0.99, rate(request_duration_seconds_bucket[1m]))

# active connections
active_connections
```

---

## Configuration

```yaml
port: 8080
algorithm: round-robin     # round-robin | least-connection | weighted | consistent-hash

health_interval: 10        # seconds between health checks
health_timeout: 3          # seconds before health check times out

rate_limit: 100            # requests per second per IP
rate_burst: 20             # token bucket burst size

backends:
  - url: http://localhost:9090
    weight: 3              # used by weighted algorithm
  - url: http://localhost:9091
    weight: 1
```

Config is validated at startup — missing required fields, invalid URLs, and `health_timeout >= health_interval` all produce clear error messages and exit immediately.

---

## Running locally

```bash
# clone
git clone https://github.com/pranavbhole123/load-balancer
cd load-balancer

# run backends (two terminals)
python3 -m http.server 9090
python3 -m http.server 9091

# run load balancer
go run ./cmd/load-balancer/main.go

# send traffic
hey -n 5000 -c 50 http://localhost:8080

# check metrics
curl http://localhost:8080/metrics
```

---

## Graceful shutdown

On `SIGTERM` or `SIGINT`:

1. Stop accepting new connections
2. Wait up to 30 seconds for in-flight requests to complete
3. Cancel health checker goroutine
4. Exit cleanly

```
^C
shutdown signal received — draining in-flight requests...
shutdown complete
```

Zero dropped requests for in-flight connections. Configurable drain timeout.

---

## Key engineering decisions

**Why Go?**
Go's goroutine-per-connection model maps naturally to load balancer concurrency. The standard library's `httputil.ReverseProxy`, `net/http`, and `sync` primitives provide exactly what's needed without external dependencies for the core logic.

**Why token bucket over sliding window for rate limiting?**
Token bucket handles burst traffic gracefully — a client that's been idle accumulates tokens and can burst briefly above the rate limit. Sliding window is stricter but can feel punishing for legitimate bursty clients. Token bucket is also the industry standard (Nginx, AWS API Gateway).

**Why 150 virtual nodes for consistent hashing?**
Fewer virtual nodes mean uneven distribution — one backend could own a disproportionate arc of the ring. 150 nodes per backend gives ~1-2% standard deviation in load distribution across backends, which is acceptable for production use.

**Why interface-driven balancer design?**
The `Balancer` interface (`Next(*http.Request) (*ReverseProxy, string, func())`) means the proxy layer knows nothing about routing logic. Algorithms are swappable at startup via config — no code changes required. Adding a new algorithm is one new file and one line in the factory function.

**Why `done()` closure instead of a `Done()` interface method?**
A `Done()` method on the interface would have no way to know which specific backend to decrement for least connections under concurrent load. A closure captures the backend index at `Next()` call time — each of 1000 concurrent requests gets its own closure with its own captured state. No coordination needed.

---

## What's next

This project is a direct precursor to a distributed key-value store. Every pattern used here has a direct analogue:

| Load balancer | Distributed KV store |
|---|---|
| Health check goroutines | Raft heartbeat goroutines |
| RWMutex on backend state | RWMutex on Raft state |
| Consistent hashing | Shard routing |
| Graceful shutdown | Clean Raft node shutdown |
| Connection pooling | Peer connection management |

---

## Tech stack

- **Go 1.22+**
- `golang.org/x/time/rate` — token bucket rate limiting
- `gopkg.in/yaml.v3` — config parsing
- `github.com/prometheus/client_golang` — metrics
- Docker Compose — local observability stack