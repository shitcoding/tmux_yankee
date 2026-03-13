#!/usr/bin/env bash
# Creates content files and launcher scripts for VHS demo recordings.
# Usage: setup.sh <scenario|all>
# Only creates content files in /tmp/ — tmux/yankee handled by VHS tapes.
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"

create_hero() {
    cat > /tmp/yankee-demo-hero.txt << 'EOF'
[2026-03-02 10:14:11] INFO  server    │ Starting HTTP server on :8080
[2026-03-02 10:14:11] INFO  server    │ Loading configuration from config.yaml
[2026-03-02 10:14:12] INFO  database  │ Connected to PostgreSQL (pool_size=10)
[2026-03-02 10:14:12] INFO  cache     │ Redis connection established (addr=localhost:6379)
[2026-03-02 10:14:13] WARN  auth      │ Token refresh failed for session=a847f2, retrying...
[2026-03-02 10:14:14] INFO  auth      │ Token refreshed successfully for session=a847f2
[2026-03-02 10:14:15] INFO  worker    │ Background job scheduler started (workers=4)
[2026-03-02 10:14:15] ERROR handler   │ timeout waiting for upstream service=payments (2500ms)
[2026-03-02 10:14:16] WARN  handler   │ Retry 1/3 for request_id=9f2c4b (service=payments)
[2026-03-02 10:14:17] ERROR handler   │ timeout waiting for upstream service=payments (2500ms)
[2026-03-02 10:14:18] WARN  handler   │ Retry 2/3 for request_id=9f2c4b (service=payments)
[2026-03-02 10:14:19] ERROR handler   │ timeout waiting for upstream service=payments (2500ms)
[2026-03-02 10:14:19] ERROR handler   │ Max retries exceeded for request_id=9f2c4b
[2026-03-02 10:14:20] ERROR handler   │ panic: assignment to entry in nil map
[2026-03-02 10:14:20] ERROR handler   │   goroutine 284 [running]:
[2026-03-02 10:14:20] ERROR handler   │   main.(*cacheHandler).Invalidate(0x0, {0x7f...})
[2026-03-02 10:14:20] ERROR handler   │       /app/handlers/cache.go:67
[2026-03-02 10:14:21] INFO  server    │ Recovered from panic, continuing to serve
[2026-03-02 10:14:22] INFO  handler   │ GET /api/v1/users (200, 42ms)
[2026-03-02 10:14:23] INFO  handler   │ POST /api/v1/orders (201, 127ms)
[2026-03-02 10:14:24] WARN  metrics   │ Latency p99=340ms exceeds limit=200ms
[2026-03-02 10:14:23] INFO  handler   │ GET /api/v1/health (200, 3ms)
[2026-03-02 10:14:24] ERROR database  │ Connection pool exhausted (active=10, waiting=3)
[2026-03-02 10:14:25] WARN  database  │ Scaling pool to max_connections=20
[2026-03-02 10:14:26] INFO  database  │ Pool scaled successfully (active=10, max=20)
[2026-03-02 10:14:27] INFO  handler   │ PUT /api/v1/orders/42 (200, 89ms)
[2026-03-02 10:14:28] INFO  handler   │ GET /api/v1/users/15 (200, 31ms)
[2026-03-02 10:14:29] ERROR handler   │ timeout waiting for upstream service=search (3000ms)
[2026-03-02 10:14:30] WARN  handler   │ Retry 1/3 for request_id=bb41a7 (service=search)
[2026-03-02 10:14:31] INFO  handler   │ GET /api/v1/search?q=orders (200, 284ms)
[2026-03-02 10:14:32] INFO  worker    │ Job completed: sync_inventory (duration=1.2s)
[2026-03-02 10:14:33] INFO  worker    │ Job completed: flush_metrics (duration=340ms)
[2026-03-02 10:14:34] INFO  handler   │ DELETE /api/v1/sessions/expired (200, 15ms)
[2026-03-02 10:14:35] WARN  auth      │ Rate limit approaching for client_id=app-frontend (85/100)
[2026-03-02 10:14:36] INFO  handler   │ POST /api/v1/webhooks (202, 8ms)
[2026-03-02 10:14:37] INFO  handler   │ GET /api/v1/orders?status=pending (200, 156ms)
[2026-03-02 10:14:38] ERROR handler   │ panic: nil pointer dereference in orderHandler.Process
[2026-03-02 10:14:38] ERROR handler   │   goroutine 847 [running]:
[2026-03-02 10:14:38] ERROR handler   │   main.(*orderHandler).Process(0x0, 0xc000284600)
[2026-03-02 10:14:38] ERROR handler   │       /app/handlers/orders.go:142
[2026-03-02 10:14:38] ERROR handler   │   main.(*Server).ServeHTTP(0xc0001a8000, {0x7f...}, 0xc...)
[2026-03-02 10:14:38] ERROR handler   │       /app/server.go:87
[2026-03-02 10:14:39] INFO  server    │ Recovered from panic, continuing to serve
[2026-03-02 10:14:40] INFO  handler   │ GET /api/v1/health (200, 2ms)
[2026-03-02 10:14:41] INFO  handler   │ POST /api/v1/auth/refresh (200, 45ms)
[2026-03-02 10:14:42] INFO  metrics   │ Exported 2847 metrics to Prometheus (took 12ms)
[2026-03-02 10:14:43] INFO  handler   │ GET /api/v1/users (200, 38ms)
[2026-03-02 10:14:44] WARN  cache     │ Cache miss rate at 23% (threshold=15%)
[2026-03-02 10:14:45] INFO  worker    │ Job started: generate_reports
[2026-03-02 10:14:46] INFO  handler   │ GET /api/v1/reports/daily (200, 892ms)
[2026-03-02 10:14:47] INFO  worker    │ Job completed: generate_reports (duration=1.8s)
[2026-03-02 10:14:48] INFO  handler   │ POST /api/v1/orders (201, 104ms)
[2026-03-02 10:14:49] INFO  handler   │ GET /api/v1/health (200, 2ms)
EOF
}

create_block_select() {
    cat > /tmp/yankee-demo-block-select.txt << 'EOF'
  PID   %CPU  %MEM   STATE    STARTED     CMD
 1042    0.4   1.2   RUN      10:14:11    api-server --port 8080
 1188    3.7   2.8   WAIT     10:14:12    worker-pool --workers 4
 1301    8.1   4.4   RUN      10:14:13    indexer --batch-size 1000
 1442    1.1   1.0   IDLE     10:14:14    scheduler --interval 30s
 1567    0.2   0.5   RUN      10:14:15    cache-warmer --ttl 3600
 1698    2.3   3.1   WAIT     10:14:16    log-collector --flush 5s
 1823    0.8   0.7   RUN      10:14:17    health-checker --timeout 10s
 1956    5.2   6.3   RUN      10:14:18    data-pipeline --mode stream
 2101    0.1   0.3   IDLE     10:14:19    cron-runner --config jobs.yaml
 2234    1.5   2.0   RUN      10:14:20    auth-proxy --upstream :8080
 2367    0.9   1.1   WAIT     10:14:21    notification-svc --queue amqp
 2498    4.3   5.8   RUN      10:14:22    ml-inference --model v2.1
 2631    0.6   0.4   RUN      10:14:23    dns-resolver --cache-ttl 300
 2764    1.8   2.5   RUN      10:14:24    rate-limiter --window 60s
 2897    0.3   0.2   IDLE     10:14:25    config-watcher --dir /etc/app

CONTAINER ID    IMAGE                        STATUS
a1b2c3d4e5f6    api-server:2.4.1             Up 3 hours
b2c3d4e5f6a7    worker-pool:2.4.1            Up 3 hours
c3d4e5f6a7b8    postgres:16-alpine           Up 3 hours
d4e5f6a7b8c9    redis:7-alpine               Up 3 hours
e5f6a7b8c9d0    nginx:1.25-alpine            Up 3 hours
f6a7b8c9d0e1    prometheus:v2.48             Up 3 hours
a7b8c9d0e1f2    grafana:10.2                 Up 3 hours

NAMESPACE       NAME                         READY   STATUS
default         api-server-7d8f9b6c4-x2k9l   1/1     Running
default         worker-pool-5c6d7e8f9-m3n4o   1/1     Running
default         indexer-3a4b5c6d7-p8q9r       1/1     Running
monitoring      prometheus-0                  1/1     Running
monitoring      grafana-6f7a8b9c0-w1x2y       1/1     Running
kube-system     coredns-5d78c9869d-z4a5b      1/1     Running
kube-system     etcd-control-plane-0          1/1     Running
EOF
}

create_search_nav() {
    # Merged search + navigation content: 200+ line config/log with repeated "timeout" token
    cat > /tmp/yankee-demo-search-nav.txt << 'EOF'
[10:14:11] INFO  Starting batch processor (batch_id=7a92f)
[10:14:12] INFO  Loading 2,847 records from staging table
[10:14:13] INFO  Validating schema: payments_v2
[10:14:14] WARN  Schema field "currency" missing default value
[10:14:15] INFO  Processing chunk 1/12 (237 records)
[10:14:16] INFO  Processing chunk 2/12 (237 records)
[10:14:17] ERROR Validation failed: amount=-42.50 (must be positive)
[10:14:17] ERROR   record_id=inv-2024-00891, field=amount, value=-42.50
[10:14:18] INFO  Processing chunk 3/12 (237 records)
[10:14:19] INFO  Processing chunk 4/12 (237 records)
[10:14:20] ERROR Duplicate key constraint: order_id=ORD-48271
[10:14:20] ERROR   table=payments, constraint=payments_order_id_key
[10:14:21] INFO  Processing chunk 5/12 (237 records)
[10:14:22] WARN  Slow query detected: 847ms (threshold=500ms)
[10:14:22] WARN    query=SELECT * FROM payments WHERE status='pending' ORDER BY created_at
[10:14:23] INFO  Processing chunk 6/12 (237 records)
[10:14:24] INFO  Processing chunk 7/12 (237 records)
[10:14:25] ERROR Connection timeout: upstream=payment-provider (timeout=5000ms)
[10:14:25] ERROR   retrying in 2s (attempt 1/3)
[10:14:27] ERROR Connection timeout: upstream=payment-provider (timeout=5000ms)
[10:14:27] ERROR   retrying in 4s (attempt 2/3)
[10:14:31] INFO  Connection restored to payment-provider
[10:14:32] INFO  Processing chunk 8/12 (237 records)
[10:14:33] INFO  Processing chunk 9/12 (237 records)
[10:14:34] ERROR Foreign key violation: customer_id=CUST-00000 not found
[10:14:34] ERROR   record_id=inv-2024-01102, table=customers
[10:14:35] INFO  Processing chunk 10/12 (237 records)
[10:14:36] INFO  Processing chunk 11/12 (237 records)
[10:14:37] INFO  Processing chunk 12/12 (240 records)
[10:14:38] WARN  3 records skipped due to validation errors
[10:14:38] WARN  2 records skipped due to constraint violations
[10:14:39] INFO  Batch complete: 2842/2847 records processed
[10:14:39] INFO  Duration: 28.4s, throughput: 100.1 records/s
[10:14:40] INFO  Committing transaction (batch_id=7a92f)
[10:14:41] INFO  Transaction committed successfully
[10:14:42] INFO  Sending completion webhook to https://hooks.example.com/batch
[10:14:43] INFO  Webhook delivered (status=200, latency=89ms)
[10:14:44] INFO  ─────────────────────────────────────────────────
[10:14:44] INFO  Starting batch processor (batch_id=8b03g)
[10:14:45] INFO  Loading 1,523 records from staging table
[10:14:46] INFO  Validating schema: refunds_v1
[10:14:47] INFO  Processing chunk 1/7 (217 records)
[10:14:48] INFO  Processing chunk 2/7 (217 records)
[10:14:49] WARN  Slow query detected: 1247ms (threshold=500ms)
[10:14:49] WARN    query=SELECT * FROM refunds JOIN orders ON refunds.order_id=orders.id
[10:14:50] INFO  Processing chunk 3/7 (217 records)
[10:14:51] ERROR Connection timeout: upstream=refund-service (timeout=3000ms)
[10:14:51] ERROR   retrying in 2s (attempt 1/3)
[10:14:53] INFO  Connection restored to refund-service
[10:14:54] INFO  Processing chunk 4/7 (217 records)
[10:14:55] INFO  Processing chunk 5/7 (217 records)
[10:14:56] ERROR Validation failed: refund_amount exceeds original (order=ORD-91234)
[10:14:56] ERROR   original=150.00, refund_requested=175.00
[10:14:57] INFO  Processing chunk 6/7 (217 records)
[10:14:58] INFO  Processing chunk 7/7 (221 records)
[10:14:59] INFO  Batch complete: 1521/1523 records processed
[10:14:59] INFO  Duration: 15.2s, throughput: 100.1 records/s
[10:15:00] INFO  Committing transaction (batch_id=8b03g)
[10:15:01] INFO  Transaction committed successfully
EOF
}

create_text_objects() {
    cat > /tmp/yankee-demo-text-objects.txt << 'EOF'
{
  "service": {
    "name": "payment-gateway",
    "version": "2.4.1",
    "environment": "production"
  },
  "server": {
    "host": "0.0.0.0",
    "port": 8443,
    "tls": {
      "cert_path": "/etc/ssl/certs/server.crt",
      "key_path": "/etc/ssl/private/server.key"
    }
  },
  "database": {
    "driver": "postgresql",
    "host": "db.internal.example.com",
    "port": 5432,
    "name": "payments_prod",
    "pool_size": 20,
    "connection_timeout": "5s"
  },
  "rate_limit": {
    "requests_per_minute": 100,
    "burst_size": 20,
    "whitelist": ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  },
  "features": {
    "flags": ["enable_webhooks", "enable_idempotency", "request_signing"],
    "maintenance": {
      "window": "(02:00-04:00 UTC)",
      "notify": "(ops-team@example.com, oncall@example.com)"
    }
  }
}
EOF
}

create_navigation() {
    cat > /tmp/yankee-demo-navigation.txt << 'EOF'
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config holds application configuration loaded from environment.
type Config struct {
	Port         int           `env:"PORT" default:"8080"`
	DatabaseURL  string        `env:"DATABASE_URL,required"`
	RedisAddr    string        `env:"REDIS_ADDR" default:"localhost:6379"`
	LogLevel     string        `env:"LOG_LEVEL" default:"info"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"15s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"15s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" default:"60s"`
}

// Server wraps the HTTP server with middleware and graceful shutdown.
type Server struct {
	config *Config
	router *http.ServeMux
	logger *log.Logger
}

// NewServer creates a configured server instance.
func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
		router: http.NewServeMux(),
		logger: log.New(os.Stdout, "[server] ", log.LstdFlags),
	}
}

// setupRoutes registers all API endpoint handlers.
func (s *Server) setupRoutes() {
	s.router.HandleFunc("/api/v1/health", s.handleHealth)
	s.router.HandleFunc("/api/v1/users", s.handleUsers)
	s.router.HandleFunc("/api/v1/orders", s.handleOrders)
	s.router.HandleFunc("/api/v1/payments", s.handlePayments)
}

// handleHealth returns server health status with uptime.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","uptime":"%s"}`, time.Since(startTime))
}

// handleUsers manages user CRUD operations.
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_ = ctx
	w.WriteHeader(http.StatusOK)
}

// handleOrders processes order creation and retrieval.
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.logger.Printf("Creating new order from %s", r.RemoteAddr)
	}
	w.WriteHeader(http.StatusOK)
}

// handlePayments processes payment transactions with retry logic.
func (s *Server) handlePayments(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Processing payment request from %s", r.RemoteAddr)
	w.WriteHeader(http.StatusAccepted)
}

// Run starts the server and blocks until shutdown signal is received.
func (s *Server) Run() error {
	s.setupRoutes()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		s.logger.Printf("Listening on :%d", s.config.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	s.logger.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

var startTime = time.Now()

func main() {
	cfg := &Config{
		Port:         8080,
		DatabaseURL:  "postgres://localhost:5432/myapp",
		RedisAddr:    "localhost:6379",
		LogLevel:     "info",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv := NewServer(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
EOF
}

create_flash() {
    # Dense code/log text with many repeated substrings so flash labels appear everywhere
    cat > /tmp/yankee-demo-flash.txt << 'EOF'
func (s *Store) ProcessOrders(ctx context.Context) error {
    orders, err := s.db.ListPendingOrders(ctx)
    if err != nil {
        return fmt.Errorf("list orders: %w", err)
    }

    for _, order := range orders {
        if err := s.validateOrder(order); err != nil {
            s.log.Warnf("invalid order %s: %v", order.ID, err)
            continue
        }

        payment, err := s.payments.Charge(ctx, order.Amount, order.Currency)
        if err != nil {
            s.log.Errorf("charge failed for order %s: %v", order.ID, err)
            s.metrics.Inc("payment_failures")
            continue
        }

        order.Status = StatusPaid
        order.PaymentID = payment.ID
        order.PaidAt = time.Now()

        if err := s.db.UpdateOrder(ctx, order); err != nil {
            s.log.Errorf("update failed for order %s: %v", order.ID, err)
            s.refund(ctx, payment.ID, order.Amount)
            continue
        }

        s.log.Infof("order %s processed (payment=%s)", order.ID, payment.ID)
        s.metrics.Inc("orders_processed")
        s.notify(ctx, order.CustomerID, order.ID)
    }

    return nil
}

func (s *Store) validateOrder(o Order) error {
    if o.Amount <= 0 {
        return errors.New("amount must be positive")
    }
    if o.Currency == "" {
        return errors.New("currency is required")
    }
    if len(o.Items) == 0 {
        return errors.New("order must have items")
    }
    return nil
}

func (s *Store) refund(ctx context.Context, paymentID string, amount int64) {
    if err := s.payments.Refund(ctx, paymentID, amount); err != nil {
        s.log.Errorf("refund failed for payment %s: %v", paymentID, err)
    }
}

func (s *Store) notify(ctx context.Context, customerID, orderID string) {
    msg := Notification{
        CustomerID: customerID,
        Type:       "order_confirmed",
        OrderID:    orderID,
        CreatedAt:  time.Now(),
    }
    if err := s.notifier.Send(ctx, msg); err != nil {
        s.log.Warnf("notification failed for customer %s: %v", customerID, err)
    }
}
EOF
}

create_flash_visual() {
    # Long changelog/function list where extending selection to a distant target is dramatic
    cat > /tmp/yankee-demo-flash-visual.txt << 'EOF'
## Changelog

### v2.4.1 (2026-03-08)
- fix: visual block cursor past end-of-line for rectangular selection
- fix: pane-swap for zoomed panes, restore tmux keybindings on exit
- feat: theme cycling via Alt+t in normal mode (5 themes)

### v2.4.0 (2026-03-07)
- feat: flash navigation with labeled jump targets (s key)
- feat: flash integration with visual selection extension
- feat: auto-jump for single-match flash patterns
- feat: configurable flash jump positions (match-start, match-end)
- fix: flash label disambiguation with forbidden characters
- fix: label visibility at line boundaries

### v2.3.0 (2026-03-01)
- feat: percentage jump ([count]%) for proportional navigation
- feat: colon go-to-line (:42) for direct line access
- feat: prefix rebinding with X-Y notation (g-g, z-t, y-y)
- feat: mode-specific keymaps (@yankee_nbind_*, @yankee_vbind_*)
- fix: bracket text objects with backward search fallback
- fix: viewport clamp for text objects and search-select motions

### v2.2.0 (2026-02-26)
- feat: incremental search (/ and ?) with real-time highlighting
- feat: search navigation (n/N) from cursor position
- feat: word search (* and #) under cursor
- feat: configurable keybindings via @yankee_bindings option
- fix: text object expansion in visual mode
- fix: ge/gE word-end backward motion

### v2.1.0 (2026-02-24)
- feat: Powerline status bar with vim-airline theme integration
- feat: mouse click-drag text selection with SGR coordinate mapping
- feat: visual block mode (Ctrl-V) for rectangular selection
- feat: paragraph motions ({/}) for jumping between blocks
- perf: zero-alloc SGR parser, render cache, lazy yank
- perf: gutter sprintf elimination and startup batching
- fix: zoom pane UX (borderless popup promotion + viewport init)

### v2.0.0 (2026-02-19)
- feat: word wrap mode with display-line navigation (gj/gk/gw)
- feat: horizontal scroll for full-width content
- feat: demo mode with 4 pages and theme preview
- feat: mouse scroll integration (scroll-up launches yankee)
- feat: f/t/F/T/;/, character search motions
- feat: ESC single-press flush (no double-press needed)
- feat: yy yank-line in normal mode

### v1.0.0 (2026-02-15)
- feat: three line number modes (absolute, relative, hybrid)
- feat: overlay display mode with pane-swap
- feat: vim motions (hjkl, w/b/e, gg/G, 0/$, Ctrl-u/d, zt/zz/zb)
- feat: visual selection (v, V) with clean gutter stripping
- feat: clipboard integration (pbcopy, xclip, xsel, wl-copy)
- feat: count prefixes (5j, 3w, 10G)
- feat: 5 built-in themes (default, dracula, gruvbox, nord, solarized)
EOF
}

create_linenums() {
    # ~80 line file with cursor around the middle to show relative offsets
    cat > /tmp/yankee-demo-linenums.txt << 'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Router struct {
	mux     *http.ServeMux
	logger  *log.Logger
	prefix  string
}

func NewRouter(prefix string) *Router {
	return &Router{
		mux:    http.NewServeMux(),
		logger: log.New(os.Stdout, "[router] ", log.LstdFlags),
		prefix: strings.TrimRight(prefix, "/"),
	}
}

func (r *Router) Handle(pattern string, handler http.Handler) {
	path := r.prefix + pattern
	r.logger.Printf("registering route: %s", path)
	r.mux.Handle(path, r.withLogging(handler))
}

func (r *Router) HandleFunc(pattern string, fn http.HandlerFunc) {
	r.Handle(pattern, fn)
}

func (r *Router) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		r.logger.Printf("%s %s %v", req.Method, req.URL.Path, time.Since(start))
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

type APIResponse struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Status: "ok",
		Data:   data,
	})
}

func errorResponse(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Status: "error",
		Error:  msg,
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{
		"uptime":  fmt.Sprintf("%v", time.Since(startTime)),
		"version": "2.4.1",
	})
}

var startTime = time.Now()

func main() {
	router := NewRouter("/api/v1")
	router.HandleFunc("/health", healthHandler)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
EOF
}

create_launcher() {
    local scenario="$1"
    local theme="${2:-dracula}"
    local start_pos="${3:-top}"
    local scrollback="${4:-500}"
    cat > "/tmp/yankee-launch-${scenario}.sh" << SCRIPT
#!/usr/bin/env bash
cd '${PROJECT_DIR}'
pane=\$(tmux display-message -p '#{pane_id}')
exec ./bin/tmux-yankee --pane "\$pane" --theme ${theme} --scrollback-lines ${scrollback} --exit-on-yank off --start-position ${start_pos}
SCRIPT
    chmod +x "/tmp/yankee-launch-${scenario}.sh"
}

# Hero clipboard-proof launcher: exits after yank so we can run pbpaste
create_hero_clipboard_launcher() {
    cat > "/tmp/yankee-launch-hero.sh" << SCRIPT
#!/usr/bin/env bash
cd '${PROJECT_DIR}'
pane=\$(tmux display-message -p '#{pane_id}')
exec ./bin/tmux-yankee --pane "\$pane" --theme dracula --scrollback-lines 500 --start-position bottom
SCRIPT
    chmod +x "/tmp/yankee-launch-hero.sh"
}

SCENARIO="${1:?Usage: setup.sh <scenario|all>}"

case "$SCENARIO" in
    hero)
        create_hero
        create_hero_clipboard_launcher
        ;;
    block-select)
        create_block_select
        create_launcher block-select dracula top
        ;;
    search-nav)
        create_search_nav
        create_launcher search-nav dracula top
        ;;
    text-objects)
        create_text_objects
        create_launcher text-objects dracula top
        ;;
    navigation)
        create_navigation
        create_launcher navigation dracula top
        ;;
    flash)
        create_flash
        create_launcher flash dracula bottom
        ;;
    flash-visual)
        create_flash_visual
        create_launcher flash-visual dracula bottom
        ;;
    linenums)
        create_linenums
        create_launcher linenums dracula top
        ;;
    all)
        create_hero
        create_block_select
        create_search_nav
        create_text_objects
        create_navigation
        create_flash
        create_flash_visual
        create_linenums
        create_hero_clipboard_launcher
        create_launcher block-select dracula top
        create_launcher search-nav dracula top
        create_launcher text-objects dracula top
        create_launcher navigation dracula top
        create_launcher flash dracula bottom
        create_launcher flash-visual dracula bottom
        create_launcher linenums dracula top
        echo "All content files and launchers created in /tmp/"
        ;;
    *)
        echo "Unknown scenario: $SCENARIO" >&2
        exit 1
        ;;
esac
