# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoLlama is a distributed LLM inference system that implements a hub-and-spoke architecture. The hub (GoLlama server) manages a pool of worker nodes, each connected to a llama.cpp instance for actual model inference. This enables distributed inference across multiple machines without relying on large LLM providers.

## Architecture

### Hub-and-Spoke Design
- **Hub (GoLlama Server)**: Central coordinator that receives chat requests, manages worker pool, and routes jobs
- **Spokes (Workers)**: Individual nodes that proxy requests to llama.cpp instances and execute inference tasks
- **Worker Pool**: Round-robin job distribution with automatic retry on failure (max 3 retries)
- **Job Processing**: 50 concurrent goroutines process jobs from a queue (default size: 5000)

### Key Components

**Server Side (`cmd/gollama/main.go`)**:
- Initializes worker pool with 5000 job queue capacity
- Starts HTTP server on port 9000
- Endpoints: `/chat`, `/connectWorker`, `/health`, `/stats`

**Worker Side (`cmd/worker/main.go`)**:
- Accepts `-port` and `-llama-port` flags for configuration
- Auto-connects to GoLlama server on startup
- Endpoints: `/execute`, `/health`, `/connect`

**Worker Pool (`internal/pool/pool.go`)**:
- Thread-safe worker management with `sync.RWMutex`
- Round-robin worker selection via `nextIdx` counter
- Automatic worker removal on failure
- Job retry mechanism (3 attempts with different workers)
- Tracks per-worker statistics (jobs completed/failed, uptime)

**Worker (`internal/worker/worker.go`)**:
- `/execute` endpoint receives dynamic endpoint + body from hub
- Proxies requests to llama.cpp (e.g., `/v1/chat/completions`)
- Uses `busyFlag` to track worker availability
- Health checks verify llama.cpp connectivity

### Request Flow
1. Client sends POST to `/chat` with `{"message": "..."}`
2. Handler creates `LlamaRequest` with message array and max_tokens
3. `WorkerJob` created with reply channel, assigned worker URL, retry count
4. Job submitted to pool queue
5. One of 50 `jobProcessor` goroutines picks up job
6. Processor calls worker's `/execute` endpoint with llama.cpp endpoint + body
7. Worker proxies to llama.cpp (e.g., `http://localhost:8080/v1/chat/completions`)
8. Response flows back through worker → pool → handler → client
9. On worker failure: stats updated, worker removed, job retried with new worker

### Data Types (`internal/types.go`)
- `ChatRequest/ChatResponse`: Client-facing API types
- `LlamaRequest/LlamaResponse`: llama.cpp API types (OpenAI-compatible)
- `WorkerJob`: Internal job representation with retry logic
- `WorkerStats`: Per-worker metrics tracking

## Configuration

The server is configured via environment variables. See `.env.example` for all available options.

**Available Configuration:**
- `GOLLAMA_PORT`: Server port (default: 9000)
- `GOLLAMA_QUEUE_SIZE`: Job queue size (default: 5000)
- `GOLLAMA_CONCURRENT_WORKERS`: Number of concurrent job processors (default: 10)
- `GOLLAMA_MAX_RETRIES`: Maximum retries per job (default: 3)
- `GOLLAMA_DEFAULT_MAX_TOKENS`: Default max tokens for LLM responses (default: 100)

**Worker Configuration:**
Workers use command-line flags:
- `-port`: Worker port (default: 9001)
- `-llama-port`: llama.cpp instance port (default: 8080)

## Development Commands

### Running the System

Start the GoLlama hub server:
```bash
# With default configuration
go run cmd/gollama/main.go

# With custom configuration
GOLLAMA_PORT=8000 GOLLAMA_CONCURRENT_WORKERS=20 go run cmd/gollama/main.go
```

Start a worker (specify ports as needed):
```bash
go run cmd/worker/main.go -port 9001 -llama-port 8080
```

### Testing

Run load test:
```bash
go run tests/loadtest/loadtest.go -requests 1000 -concurrency 50 -message "test message"
```

Run chaos test:
```bash
go run tests/chaostest/chaostest.go
```

### API Testing

Send a chat request:
```bash
curl -X POST http://localhost:9000/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Tell me a joke"}'
```

Check server health and worker count:
```bash
curl http://localhost:9000/health
```

View worker statistics:
```bash
curl http://localhost:9000/stats
```

Connect a worker manually:
```bash
curl http://localhost:9001/connect
```

## Important Implementation Details

### Concurrency & Thread Safety
- Worker pool uses `sync.RWMutex` for all map/slice operations
- Configurable concurrent `jobProcessor` goroutines handle job queue (default: 10)
- Each chat request blocks on reply channel until job completes
- Worker stats updated atomically under lock

### Error Handling & Resilience
- **Busy workers**: Skipped and job retried with different worker (worker stays in pool)
- **Unreachable workers**: Removed from pool and job retried with different worker
- **Failed workers**: Removed from pool after error response
- Failed jobs retry up to configurable max (default: 3) with different workers
- Error detection: responses starting with "Error", "error", or "Worker"
- Workers auto-reconnect on startup via `/connect` endpoint

### Known Issues
Under high load, some requests may hang while newer requests process. Potential causes noted in `pool.go:71-74`:
- Worker health check delays
- Channel/goroutine leaks
- Mutex contention from heavy stats logging

### Remaining Hard-coded Values
These values are still hard-coded and may need configuration in the future:
- llama.cpp endpoint: `/v1/chat/completions` (in `pool.go` - could support dynamic endpoints)

## Testing Strategy

The `tests/` directory contains specialized test harnesses:
- `loadtest.go`: Concurrent load testing with configurable request count and concurrency
- `chaostest.go`: Chaos engineering tests for resilience validation

Load test provides detailed metrics:
- Success/failure rates
- Latency percentiles (min, avg, max, p50, p95, p99)
- Requests per second
- Total test duration
