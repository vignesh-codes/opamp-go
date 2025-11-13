# OpAMP Fleet MCP Server

An MCP (Model Context Protocol) server for Claude Anthropic that provides tools to analyze and understand OpenTelemetry Collector configurations from your OpAMP fleet.

## Features

The MCP server provides the following tools that connect to your live OpAMP API server:

### Configuration Analysis Tools

1. **analyze_config** - Analyzes an OpenTelemetry Collector configuration and provides a detailed breakdown of receivers, processors, exporters, and pipelines.

2. **validate_config** - Validates an OpenTelemetry Collector configuration for common issues and missing required sections.

3. **get_config_insights** - Provides insights and recommendations for optimizing your OpenTelemetry Collector configuration.

### Live API Tools (Connects to OpAMP Server at port 9081)

4. **list_agents** - Lists all connected OpenTelemetry Collector agents from the live API server.

5. **get_agent_config** - Retrieves the configuration for a specific agent by ID from the live API server.

6. **get_topology** - Gets the complete topology graph showing all agents, receivers, processors, exporters, and their connections.

7. **get_agent_summary** - Gets a summary overview of all agents including health status and metrics.

8. **get_pipelines** - Gets parsed pipeline information for all agents showing receivers, processors, and exporters.

## Building

To build the MCP server binary:

```bash
cd internal/examples/mcp-server/internal/mcp
go build -o mcp-server ./cmd/mcp-server
```

Or from the mcp-server root:

```bash
cd internal/examples/mcp-server/internal/mcp
go build -o mcp-server ./cmd/mcp-server
```

## Configuration

The MCP server uses the following environment variables:

- **`OPAMP_API_URL`** - The base URL of the OpAMP API server (default: `http://localhost:9081`)

**Note:** The MCP server uses the live API endpoints to get agent data directly from the OpAMP server. Make sure your OpAMP server is running on port 9081 (or configure the URL via `OPAMP_API_URL`).

## Usage with Claude Desktop

Add the following to your Claude Desktop configuration file (typically `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "opamp-fleet": {
      "command": "/home/vws/Desktop/vws-codes/projects/PaaS-On-Kubernetes/opamp-go/internal/examples/mcp-server/internal/mcp/mcp-server",
      "args": [],
      "env": {
        "OPAMP_API_URL": "http://localhost:9081"
      }
    }
  }
}
```

### Example Configuration

```json
{
  "mcpServers": {
    "opamp-fleet": {
      "command": "/home/vws/Desktop/vws-codes/projects/PaaS-On-Kubernetes/opamp-go/internal/examples/mcp-server/internal/mcp/mcp-server",
      "args": [],
      "env": {
        "OPAMP_API_URL": "http://localhost:9081"
      }
    }
  }
}
```

## Protocol

The MCP server communicates using JSON-RPC 2.0 over stdio. It implements the following methods:

- `initialize` - Initializes the MCP connection
- `tools/list` - Lists all available tools
- `tools/call` - Calls a specific tool with arguments

## Tools

### analyze_config

Analyzes an OpenTelemetry Collector configuration YAML and provides a structured breakdown.

**Arguments:**
- `config_yaml` (string, required): The OpenTelemetry Collector configuration in YAML format

**Example:**
```json
{
  "name": "analyze_config",
  "arguments": {
    "config_yaml": "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 0.0.0.0:4317\n..."
  }
}
```

### validate_config

Validates an OpenTelemetry Collector configuration for common issues.

**Arguments:**
- `config_yaml` (string, required): The OpenTelemetry Collector configuration in YAML format

### get_config_insights

Provides insights and recommendations for optimizing your configuration.

**Arguments:**
- `config_yaml` (string, required): The OpenTelemetry Collector configuration in YAML format

### list_agents

Lists all connected agents from the live API server.

**Arguments:** None

### get_agent_config

Gets the configuration for a specific agent from the live API server.

**Arguments:**
- `agent_id` (string, required): The agent ID to retrieve configuration for

### get_topology

Gets the complete topology graph from the live API server.

**Arguments:** None

### get_agent_summary

Gets a summary overview of all agents from the live API server.

**Arguments:** None

### get_pipelines

Gets parsed pipeline information for all agents from the live API server.

**Arguments:** None

## Development

The MCP server is implemented in Go and uses:
- `gopkg.in/yaml.v3` for YAML parsing
- Standard library `encoding/json` for JSON-RPC communication

## License

Same as the parent opamp-go project.

