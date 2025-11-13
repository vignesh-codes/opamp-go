# OpAMP protocol implementation in Go

Extension of https://github.com/open-telemetry/opamp-go/ 

Added a new example here: https://github.com/vignesh-codes/opamp-go/tree/feat/v0.1/internal/examples/opamp-api-and-mcp-server

NOTE: Code cleanings in progress

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
OpenTelemetry Agent Configuration Report
Production-Grade Assessment & Recommendations
Report Date: November 12, 2025
Agent ID: 6595e281-9a7b-4dea-9243-c4b50ffd9136
Agent Version: 0.139.0
Environment: Kubernetes (Linux, amd64)
Service: otel-agent-opentelemetry-collector-agent-xz2pc

Executive Summary
The OpenTelemetry collector agent demonstrates a well-structured configuration with multi-signal observability support (logs, metrics, traces). However, the current setup exhibits several production readiness gaps, primarily around instrumentation coverage, security hardening, and resource optimization. The configuration is suitable for development/testing environments but requires critical modifications before deployment to production.
Overall Readiness Status: ⚠️ CONDITIONAL - Requires addressing critical findings before production deployment.

1. Observability Signals Assessment
1.1 Traces Collection ✓
Status: Well-configured with multiple protocol support
The traces pipeline is properly instrumented with three receiver types:

OTLP gRPC (172.18.0.5:4317) - Native OpenTelemetry protocol with optimized binary encoding
Jaeger gRPC (172.18.0.5:14250) - Enterprise-grade distributed tracing protocol
Jaeger Compact (172.18.0.5:6831) - Low-bandwidth alternative
Jaeger HTTP (172.18.0.5:14268) - Firewall-friendly protocol
Zipkin HTTP (172.18.0.5:9411) - Cross-platform compatibility

Recommendation: All three trace protocols are excessive for production. Consolidate to OTLP gRPC (primary) with HTTP fallback for environments with firewall restrictions. Remove unused Jaeger and Zipkin protocols to reduce attack surface.
1.2 Metrics Collection ✓
Status: Comprehensive Kubernetes-native metrics
The metrics pipeline captures system-level observability across two dimensions:
Host-Level Metrics: CPU, memory, disk, and network metrics with 30-second collection intervals
Kubernetes-Level Metrics (kubeletstats receiver):

Pod metrics (CPU, memory, filesystem, network)
Container metrics (CPU time, memory RSS, working set)
Node metrics (comprehensive system state)
Volume metrics (storage utilization)

Issue Identified: CPU and memory utilization relative metrics (e.g., k8s.container.cpu_limit_utilization, k8s.container.memory_limit_utilization) are disabled, limiting cost/performance analysis capabilities.
Recommendation: Enable utilization metrics for container-level cost attribution and performance anomaly detection. These are essential for SLO/SLA monitoring.
1.3 Logs Collection ✓
Status: Container-wide collection with intelligent filtering
File-based log collection covers:

Container logs in /var/log/containers/
Pod logs in /var/log/pods/
Automatic container parser with metadata extraction
Exclusions for self-referential logs

Issue Identified: The filter/logs processor applies multiple exclusions (kube-system namespace, test pods, debug severity, istio-proxy, marked-drop attributes) but lacks explicit inclusion criteria. This creates implicit data loss.
Recommendation: Define explicit inclusion filters first, then apply exclusions. Add structured logging to logs pipeline for better querying and alerting.

2. Data Processing & Quality
2.1 Batch Processing
Configuration:

Batch size: 8,192 spans/logs/metrics
Timeout: 200ms
Queue size: 1,000 items

Assessment: The batch processor is appropriately configured for typical workloads. The 200ms timeout ensures reasonable latency while enabling efficient batching for throughput.
2.2 Filtering & Data Governance
Current Filters:
Logs are filtered by:

Kubernetes namespace (excluding kube-system)
Pod name patterns (excluding test pods)
Severity level (excluding DEBUG)
Container name (excluding istio-proxy)
Custom drop attributes

Metrics are filtered by:

Kubernetes namespace
Metric name (container.cpu.time)
Pod name patterns
Container identity

Production Concern: The filtering strategy relies on implicit exclusions rather than explicit, documented policies. This creates maintenance burden and potential data inconsistency.
Recommendation:

Document all filtering policies in code comments
Implement attribute whitelisting instead of blacklisting
Move filters to the backend (collector gateway) to reduce agent load
Add sampling strategies for high-volume signals (e.g., INFO-level logs)

2.3 Resource Enrichment
Current Implementation: Resource processor adds six attributes:

environment: dev
cluster: local-k8s
Multiple "noise" attributes (xk3h82ls9p0q, 3984jf9sd93jfw9sd2j3f9jsdf, etc.)

Critical Issue: Noise attributes serve no operational purpose and significantly increase cardinality, leading to higher storage costs and slower queries. These appear to be test/development artifacts.
Recommendation: Remove all noise attributes immediately. Retain only operational attributes:

environment (dev/staging/prod)
cluster (logical cluster identifier)
Add: region, datacenter, team, service.version for full context

2.4 Kubernetes Attributes
The k8sattributes processor enriches telemetry with Kubernetes context:
Enabled Metadata:

Namespace name
Pod name
Container name

Disabled Metadata:

Pod annotations
Pod labels
Deployment name association
Node association

Assessment: This is minimally configured. Essential Kubernetes labels (app, version, owner) are not extracted.
Recommendation: Enable comprehensive label extraction with regex patterns for selective capture of business-critical labels. This enables sophisticated querying and attribution.

3. Stability & Resilience
3.1 Memory Management
Critical Gap: Memory limiter processor is configured but not active in any pipeline.
yamlmemory_limiter:
    check_interval: 5s
    limit_percentage: 80
    spike_limit_percentage: 25
This configuration exists but is unused. Without active memory management, the agent is vulnerable to OOM kills under load spikes.
Recommendation: Immediately add memory_limiter to all three pipelines (logs, metrics, traces) at the first position. This is non-negotiable for production.
3.2 Queue Management
Exporter Queue Configuration:
OTLP exporter uses:

Queue size: 1,000 items
num_consumers: 10
block_on_overflow: false (drops data on queue saturation)
wait_for_result: false (fire-and-forget mode)

Debug exporter uses:

Queue size: 1 (extremely small)
Potential for immediate queue overflow

Production Concern: block_on_overflow=false means telemetry loss during backend latency. Combined with wait_for_result=false, this creates silent data loss.
Recommendation: For production, configure persistent storage or at least increase queue sizes significantly. Consider adding persistent queue storage using local disk for durability.
3.3 Retry Configuration
OTLP Exporter Retry Policy:

Initial interval: 5s
Max interval: 30s
Max elapsed time: 5 minutes
Multiplier: 1.5
Randomization: 0.5

Assessment: Good exponential backoff strategy with jitter to prevent thundering herd. Five-minute retry window is reasonable for temporary backend outages.
Recommendation: Add retry storage to persist failed batches to disk for extended outages exceeding 5 minutes.

4. Security Assessment
4.1 Critical Security Issues
1. TLS Disabled for Backend Communication:
yamltls:
    insecure: true
All backend communication to the gateway collector occurs unencrypted. This violates production security requirements.
2. TLS Skip Verification (OpAMP):
The OpAMP connection to wss://host.docker.internal:4320 disables certificate verification:
yamltls:
    insecure_skip_verify: true
This creates vulnerability to man-in-the-middle attacks.
3. Kubelet API: Kubelet stats collection skips verification:
yamlinsecure_skip_verify: true
While this may be acceptable in dev, production clusters should enforce proper certificate validation.
Recommendations:

Enable TLS with proper certificate management for all external connections
Implement certificate rotation policies
Use service mesh mTLS if available (Istio, Linkerd)
Enable certificate verification for kubelet API

4.2 Authentication & Authorization
Assessment: No explicit authentication configured between agent and backend. Authentication relies on network isolation.
Recommendation: Implement service-to-service authentication using:

TLS client certificates
OAuth 2.0 / JWT tokens
Service mesh mutual TLS


5. Backend Integration
5.1 Gateway Endpoint
Configuration: otel-gateway-opentelemetry-collector.otel-gateway.svc.cluster.local:4317
Assessment: Proper Kubernetes DNS naming indicates stable in-cluster deployment. Service discovery is appropriately configured.
Potential Issue: Single gateway endpoint with no failover. Agent depends on single point of availability.
Recommendation: Configure multiple gateway endpoints with load balancing to ensure high availability.
5.2 Data Compression
OTLP Exporter: gzip compression enabled
Assessment: Reduces bandwidth and latency for gateway communication. Appropriate for production.

6. Collection Efficiency
6.1 Telemetry Overhead
Current Overhead Sources:

1,800 kubeletstats metrics per collection interval (10s)
600+ hostmetrics per interval (30s)
Estimated log volume: ~10-100 logs/second from all containers

Issue: Kubelet collection interval is 10 seconds, producing 180 metrics/second in steady state. The 30-second hostmetrics interval compounds this.
Recommendation: Analyze actual required metric density. Most production systems function well with 30-second collection intervals for both host and kubelet metrics. Reducing frequency by 3x would dramatically lower resource consumption.
6.2 CPU & Memory Profile
The agent includes pprof profiling enabled:
yamlpprof:
    endpoint: 0.0.0.0:1777
Assessment: Good for development debugging, but exposes internal profiling data on port 1777.
Recommendation: Disable in production or restrict access to only internal network with proper authentication.

7. Operational Visibility
7.1 Health Checks
Configuration:
yamlhealth_check:
    endpoint: 0.0.0.0:13133
    check_collector_pipeline:
        enabled: false
Issue: Pipeline health checks disabled, only component health is monitored.
Recommendation: Enable pipeline health checks to detect when exporters fail or become unhealthy.
7.2 Diagnostics & Telemetry
Collector Self-Telemetry:

Metrics: Detailed level (all metrics exposed)
Logs: Debug level with sampling (10 initial, then 1 per 100)
Prometheus scrape: Enabled on port 8888

Assessment: Comprehensive self-monitoring. Debug logging ensures visibility into collector behavior. Prometheus scrape enables integration with monitoring systems.
Recommendation: For production, reduce log level to INFO (not DEBUG) to decrease log volume and I/O overhead.

8. Kubernetes Integration
8.1 Resource Configuration
Detected from Labels:

host.arch: amd64
host.name: desktop-control-plane
os.type: linux
Service: otel-agent-opentelemetry-collector-agent-xz2pc

Assessment: Appropriate for single-node development. Missing resource requests/limits specification.
Recommendation: Define Kubernetes resource requests and limits:

Requests: CPU 500m, Memory 512Mi (minimum for production)
Limits: CPU 2000m, Memory 2Gi (scale based on workload)

8.2 OpAMP Management
Status: Enabled with full capabilities

Reports available components
Reports effective configuration
Reports health status

Assessment: Good for remote management and fleet orchestration.

9. Production-Ready Checklist
ItemStatusPriorityTLS enabled for all connections❌CRITICALMemory limiter active in pipelines❌CRITICALPersistent queue storage❌HIGHRemove debug exporter❌HIGHRemove noise attributes❌HIGHEnable container utilization metrics❌MEDIUMDocument filtering policies❌MEDIUMConfigure health check pipelines❌MEDIUMReduce kubeletstats interval to 30s❌MEDIUMEnable certificate verification❌MEDIUMDefine Kubernetes resource limits❌HIGHConsolidate trace receivers⚠️MEDIUMSwitch logs to INFO level⚠️LOW

10. Recommendations Summary
Immediate Actions (Before Production)

Enable TLS for all backend connections with proper certificate management
Activate memory limiter in all pipelines to prevent OOM events
Remove noise attributes from resource processor (saves ~15% cardinality)
Remove debug exporter - use sampling in backend instead
Define Kubernetes resource limits in Pod manifests

Short-term Improvements (First Sprint)

Implement persistent queue storage for durability
Enable container utilization metrics for cost analysis
Consolidate trace receivers to OTLP + HTTP fallback
Enable pipeline health checks
Document all filtering policies

Medium-term Optimization (Second Sprint)

Implement service-to-service authentication
Reduce kubeletstats collection interval to 30s
Enable comprehensive Kubernetes label extraction
Configure multi-endpoint gateway with failover
Switch telemetry logging to INFO level
Implement sampling strategies for high-volume logs

Long-term Enhancements (Quarterly)

Implement distributed tracing context propagation
Add custom metrics/instrumentation for business logic
Optimize cardinality control per signal type
Integrate with cost attribution system


Conclusion
Your OpenTelemetry agent configuration demonstrates solid fundamentals with comprehensive multi-signal collection. The setup is production-capable but requires security hardening and stability improvements before handling mission-critical workloads. Prioritize the critical-severity items (TLS, memory management) immediately. The configuration provides excellent visibility once these foundational issues are resolved.
Estimated effort to production-ready: 2-3 days for critical fixes, additional 1-2 weeks for comprehensive hardening.
Next steps:

Schedule security review with DevOps/SRE team
Allocate resources for TLS certificate infrastructure
Implement configuration changes in development environment
Conduct load testing before production deployment
```


## Architecture

![architecture](https://github.com/vignesh-codes/opamp-go/blob/feat/v0.1/internal/examples/opamp-api-and-mcp-server/assets/architecture.png)

I added prometheus just for the POC on how to achieve getting the rate of data flow. Ideally we might wanna add a custom lightweight processor to aggregate these metrics exposed by otelcol /metrics endpoint and then capture at a centralized server for final aggregation. This provides a stronger insight on how the data is flowing and helps in identifying the config bottlenecks in otelcol components. 

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
