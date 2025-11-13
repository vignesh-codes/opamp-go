# OpAMP protocol implementation in Go

Extension of https://github.com/open-telemetry/opamp-go/ 

Added a new example here: https://github.com/vignesh-codes/opamp-go/tree/feat/v0.1/internal/examples/opamp-api-and-mcp-server

# OpAMP Topology Server

OpAMP Topology Server provides a unified, real-time view of your entire OpenTelemetry Collector fleet.  
It runs both an **OpAMP server** (for agent management) and an **API server** (for topology visualization and insights) within a single Go service.

This project helps teams understand, observe, and optimize their telemetry pipeline performance through live topology data, configuration intelligence, and AI-driven insights.

![topology graph](https://github.com/vignesh-codes/opamp-go/blob/feat/v0.1/internal/examples/opamp-api-and-mcp-server/assets/topology-graph-flow-rate.png)

---

## What It Does

- **Visualizes the OpenTelemetry fleet** – Displays all connected collectors, receivers, processors, and exporters in an interactive topology view.  
- **Monitors real-time data flow** – Tracks telemetry movement across the fleet and identifies throughput patterns.  
- **Centralizes agent management** – Uses the OpAMP protocol to discover and manage collector agents.  
- **Supports Prometheus integration (optional)** – Fetches short-term metrics for data flow and health insights.  
- **Extends with AI assistance via MCP** – Connects to Claude for configuration evaluation and improvement recommendations.

---

## Example Use Cases

- Visualize and debug complex OpenTelemetry Collector topologies  
- Measure and compare data flow rates for each component  
- Detect misconfigurations and missing capabilities across agents  
- Generate AI-based reports on configuration health and optimization paths

---

## Claude in Action

Detailed Report on path to production - https://claude.ai/public/artifacts/7fa58656-1ac9-4579-bb2f-17568e0ee5e5


```
📊 Report Highlights (From Claude):
Current State Analysis

✅ Functional 2-pipeline setup (logs + metrics)
⚠️ Development-grade configuration
🔴 4 critical security/configuration issues
🟠 6 high-priority operational gaps
🟡 5 medium-priority optimizations

Detailed Breakdown

Logs Pipeline: Filelog receiver → Transform/K8s/Resource/Filter/Batch → OTLP exporter
Metrics Pipeline: Hostmetrics + Kubeletstats → K8s/Filter/Batch → OTLP exporter
Performance profile with bottleneck analysis
Security assessment (currently unsafe for production)

Critical Issues Found

TLS disabled (cleartext data)
Kubeletstats TLS not verified (MITM vulnerable)
Memory limiter disabled (node crash risk)
Single agent (no high availability)

Actionable Improvements

Week 1 Quick Wins: 4 hours to fix low-risk issues
Week 2 Medium-term: 6 hours for TLS + storage
Weeks 3-4 Production-grade: 10 hours for HA setup
Total: ~20 hours spread over a month

Before/After Comparison

Memory: 500MB → 400MB (-20%)
CPU: 150m → 100m (-33%)
Log storage: 10GB/day → 1MB/day (-99.99%)
Availability: Single pod → 3-replica HA (99.95% uptime)
Security: ❌ Cleartext → ✅ TLS Encrypted

Implementation Timeline & Cost

20-22 hours total effort
ROI: Saves $336/year in costs + eliminates downtime
Detailed rollback procedures for each change
Success metrics defined

The report is ready to present to your team! 📋

```


Make sure to update the claude_desktop_config.json
```json
{
  "mcpServers": {
    "opamp-fleet": {
      "command": "path/internal/mcp/mcp-server",
      "args": [],
      "env": {
        "OPAMP_API_URL": "http://localhost:9081"
      }
    }
  }
}
```


## Architecture

The server follows Go best practices with a clean, modular structure:

```
mcp-server/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── deploy/                      # Helm deployment files
│   ├── otel-agent-values.yaml
│   ├── otel-gateway-values.yaml
│   ├── prometheus-values.yaml
│   ├── otel-agent-metrics-svc.yaml
│   ├── otel-gateway-metrics-svc.yaml
│   └── README.md
├── internal/
│   ├── api/                     # HTTP handlers
│   │   ├── handlers.go
│   │   └── handlers_test.go
│   ├── config/                  # Configuration parsing
│   │   ├── config.go
│   │   └── config_test.go
│   ├── metrics/                 # Metrics scraping
│   │   ├── metrics.go
│   │   └── metrics_test.go
│   ├── prometheus/             # Prometheus queries
│   │   ├── prometheus.go
│   │   └── prometheus_test.go
│   ├── server/                 # OpAMP server wrapper
│   │   ├── opamp.go
│   │   └── opamp_test.go
│   ├── topology/               # Topology graph builder
│   │   ├── topology.go
│   │   └── topology_test.go
│   ├── types/                  # Type definitions
│   │   └── types.go
│   └── view/                   # HTML view templates
│       ├── view.go
│       └── view_test.go
├── go.mod
├── go.sum
└── README.md
```

## Endpoints

- `GET /` - API documentation
- `GET /health` - Server health status
- `GET /api/agents` - List all agents
- `GET /api/agents/{id}` - Get specific agent by ID
- `GET /api/configs` - Get all agent configurations
- `GET /api/pipelines` - Get parsed pipeline information
- `GET /api/topology` - Get topology graph (JSON)
- `GET /api/topology/view` - Get topology visualization (HTML)
- `GET /api/health` - Get health status of all agents
- `GET /api/summary` - Get summary overview of all agents
- `GET /api/transfer` - Get data transfer metrics

## Configuration

### Environment Variables

- `PORT` - API server port (default: `9081`)
- `OPAMP_LISTEN_ENDPOINT` - OpAMP server endpoint (default: `0.0.0.0:4320`)
- `PROMETHEUS_URL` - Prometheus URL for metrics (default: `http://localhost:9090`)

## Building

```bash
cd internal/examples/mcp-server
go build -o topology-server ./cmd/server
```

## Running

```bash
# Using default settings
./topology-server

# With custom port
PORT=9081 ./topology-server

# With Prometheus integration
PROMETHEUS_URL=http://prometheus:9090 ./topology-server
```

Or using `go run`:

```bash
cd internal/examples/mcp-server
go run ./cmd/server
```

## Usage

1. **Start the server**:
   ```bash
   go run ./cmd/server
   ```

2. **Connect OpenTelemetry Collector agents** to the OpAMP server:
   - OpAMP endpoint: `ws://localhost:4320/v1/opamp`
   - Agents will appear in the API immediately upon connection

3. **Access the topology view**:
   - Open browser: http://localhost:9081/api/topology/view
   - Or get JSON: http://localhost:9081/api/topology
   - Or check agents: http://localhost:9081/api/agents

4. **The topology view shows**:
   - Agents (purple nodes)
   - Receivers (blue nodes)
   - Processors (orange nodes)
   - Exporters (purple nodes)
   - Endpoints (red nodes)
   - Data flow edges with real-time rates


## Deployment

This server can be deployed alongside OpenTelemetry Collector agents and a Prometheus instance. See the [deploy/README.md](deploy/README.md) for complete deployment instructions.

### Quick Start Deployment

1. **Deploy OpenTelemetry Gateway**:
   ```bash
   helm upgrade --install otel-gateway open-telemetry/opentelemetry-collector \
     --namespace otel-gateway \
     --create-namespace \
     -f deploy/otel-gateway-values.yaml
   kubectl apply -f deploy/otel-gateway-metrics-svc.yaml
   ```

2. **Deploy OpenTelemetry Agent**:
   ```bash
   helm upgrade --install otel-agent open-telemetry/opentelemetry-collector \
     --namespace otelagent \
     --create-namespace \
     -f deploy/otel-agent-values.yaml
   kubectl apply -f deploy/otel-agent-metrics-svc.yaml
   ```

3. **Deploy Prometheus**:
   ```bash
   helm upgrade --install prometheus prometheus-community/prometheus \
     --namespace prometheus \
     --create-namespace \
     -f deploy/prometheus-values.yaml
   ```

4. **Port-Forward Prometheus**:
   ```bash
   kubectl port-forward -n prometheus svc/prometheus-server 9090:9090
   ```

5. **Start OpAMP Server**:
   ```bash
   cd internal/examples/mcp-server
   export PORT=9081
   export OPAMP_LISTEN_ENDPOINT=0.0.0.0:4320
   export PROMETHEUS_URL=http://localhost:9090
   go run ./cmd/server
   ```

6. **Access the Topology View**:
   - Open browser: http://localhost:9081/api/topology/view
   - API Documentation: http://localhost:9081/
   - Prometheus UI: http://localhost:9090

For detailed deployment instructions, troubleshooting, and configuration options, see [deploy/README.md](deploy/README.md).

## Notes

- The server reads agent data directly from the live OpAMP server in real-time
- No persistence layer is needed since agents are managed by the OpAMP server
- Prometheus integration is optional but recommended for real-time metrics
- All packages are in `internal/` to prevent external imports
- The entry point is in `cmd/server/main.go` following Go conventions
