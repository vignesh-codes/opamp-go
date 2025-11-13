package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// MCPRequest represents a JSON-RPC request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeParams represents the initialize request parameters
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// InitializeResult represents the initialize response
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolCallParams represents tool call parameters
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Server handles MCP protocol communication
type Server struct {
	tools     map[string]ToolHandler
	apiClient *APIClient
}

// ToolHandler is a function that handles a tool call
type ToolHandler func(args map[string]interface{}) (interface{}, error)

// NewServer creates a new MCP server
func NewServer() *Server {
	apiURL := os.Getenv("OPAMP_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:9081"
	}

	s := &Server{
		tools:     make(map[string]ToolHandler),
		apiClient: NewAPIClient(apiURL),
	}

	// Register default tools
	s.RegisterTool("analyze_config", Tool{
		Name:        "analyze_config",
		Description: "Analyzes an OpenTelemetry Collector configuration YAML and provides a detailed breakdown including: receivers (what data sources are configured), processors (what transformations are applied), exporters (where data is sent), and pipelines (how data flows through the system). Returns a structured markdown report with counts and details for each component type.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config_yaml": map[string]interface{}{
					"type":        "string",
					"description": "The complete OpenTelemetry Collector configuration in YAML format. Should include receivers, processors, exporters, and service sections.",
				},
			},
			"required": []string{"config_yaml"},
		},
	}, s.handleAnalyzeConfig)

	s.RegisterTool("validate_config", Tool{
		Name:        "validate_config",
		Description: "Validates an OpenTelemetry Collector configuration YAML for common issues and missing required sections. Checks for: required 'service' section, pipeline definitions, receivers and exporters in pipelines, and provides a detailed report with ❌ issues (must fix) and ⚠️ warnings (should fix). Returns validation status and specific recommendations.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config_yaml": map[string]interface{}{
					"type":        "string",
					"description": "The OpenTelemetry Collector configuration in YAML format to validate",
				},
			},
			"required": []string{"config_yaml"},
		},
	}, s.handleValidateConfig)

	s.RegisterTool("get_config_insights", Tool{
		Name:        "get_config_insights",
		Description: "Provides detailed insights and optimization recommendations for an OpenTelemetry Collector configuration. Analyzes processor usage (batch, memory_limiter, resource), pipeline complexity, and identifies potential performance issues. Returns 💡 recommendations for improvements and ⚠️ warnings about problematic patterns (e.g., debug exporters in production, too many processors).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config_yaml": map[string]interface{}{
					"type":        "string",
					"description": "The OpenTelemetry Collector configuration in YAML format to analyze",
				},
			},
			"required": []string{"config_yaml"},
		},
	}, s.handleGetConfigInsights)

	s.RegisterTool("list_agents", Tool{
		Name:        "list_agents",
		Description: "Lists all connected OpenTelemetry Collector agents from the live OpAMP API server (default: http://localhost:9081). Returns a formatted list showing agent IDs and names. Useful for discovering which agents are currently connected before querying specific agent details. Returns 'No agents found' message if none are connected.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, s.handleListAgents)

	s.RegisterTool("get_agent_config", Tool{
		Name:        "get_agent_config",
		Description: "Retrieves the complete configuration for a specific agent by ID from the live OpAMP API server. Returns agent name, labels/metadata, and the full current configuration YAML. Use this to inspect what configuration a specific agent is running. The configuration is returned as formatted YAML in a code block for easy reading.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "The unique agent ID to retrieve configuration for. Use list_agents tool first to get available agent IDs.",
				},
			},
			"required": []string{"agent_id"},
		},
	}, s.handleGetAgentConfig)

	s.RegisterTool("get_topology", Tool{
		Name:        "get_topology",
		Description: "Retrieves the complete topology graph from the live OpAMP API server showing all agents, receivers, processors, exporters, and their data flow connections. Returns a structured overview grouped by node type (agent, receiver, processor, exporter) with counts, and shows edge connections between components including pipeline types (traces, logs, metrics). Shows first 10 edges by default with total count. Use this to understand the complete data flow architecture across all connected agents.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, s.handleGetTopology)

	s.RegisterTool("get_agent_summary", Tool{
		Name:        "get_agent_summary",
		Description: "Retrieves a comprehensive summary overview of all agents from the live OpAMP API server including health status, connection status, metrics (spans, logs, metrics received/sent), and agent metadata. Returns formatted JSON with detailed information about each agent's current state. Use this to get a high-level view of your entire OpAMP fleet's health and activity.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, s.handleGetAgentSummary)

	s.RegisterTool("get_pipelines", Tool{
		Name:        "get_pipelines",
		Description: "Retrieves parsed pipeline information for all agents from the live OpAMP API server. Returns detailed information about each agent's pipelines including: pipeline types (traces, logs, metrics), the ordered list of receivers, processors, and exporters in each pipeline, and endpoint information. Returns formatted JSON. Use this to understand the data processing flow for each agent without needing to parse raw configuration YAML.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, s.handleGetPipelines)

	return s
}

// RegisterTool registers a tool with the server
func (s *Server) RegisterTool(name string, tool Tool, handler ToolHandler) {
	s.tools[name] = handler
}

// Run starts the MCP server and handles requests over stdio
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Error parsing request: %v", err)
			continue
		}

		resp := s.handleRequest(&req)

		respJSON, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshaling response: %v", err)
			continue
		}

		fmt.Println(string(respJSON))
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

// handleRequest processes an MCP request
func (s *Server) handleRequest(req *MCPRequest) *MCPResponse {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = s.handleInitialize(req.Params)
	case "tools/list":
		resp.Result = s.handleListTools()
	case "tools/call":
		result, err := s.handleToolCall(req.Params)
		if err != nil {
			resp.Error = &MCPError{
				Code:    -32603,
				Message: err.Error(),
			}
		} else {
			resp.Result = result
		}
	default:
		resp.Error = &MCPError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	return resp
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(params json.RawMessage) *InitializeResult {
	var initParams InitializeParams
	if err := json.Unmarshal(params, &initParams); err != nil {
		// Use defaults if unmarshal fails
	}

	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "opamp-fleet-mcp",
			Version: "1.0.0",
		},
	}
}

// handleListTools returns the list of available tools
func (s *Server) handleListTools() map[string]interface{} {
	// Reuse the same tool definitions from RegisterTool calls for consistency
	tools := []Tool{}

	// Build tools list from registered tools
	for name := range s.tools {
		// Get tool definition - we'll need to store these separately or reconstruct
		// For now, we'll create a minimal version that matches what's registered
		tool := Tool{
			Name: name,
		}

		// Set description and schema based on tool name
		switch name {
		case "analyze_config":
			tool.Description = "Analyzes an OpenTelemetry Collector configuration YAML and provides a detailed breakdown including: receivers (what data sources are configured), processors (what transformations are applied), exporters (where data is sent), and pipelines (how data flows through the system). Returns a structured markdown report with counts and details for each component type."
			tool.InputSchema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"config_yaml": map[string]interface{}{
						"type":        "string",
						"description": "The complete OpenTelemetry Collector configuration in YAML format. Should include receivers, processors, exporters, and service sections.",
					},
				},
				"required": []string{"config_yaml"},
			}
		case "validate_config":
			tool.Description = "Validates an OpenTelemetry Collector configuration YAML for common issues and missing required sections. Checks for: required 'service' section, pipeline definitions, receivers and exporters in pipelines, and provides a detailed report with ❌ issues (must fix) and ⚠️ warnings (should fix). Returns validation status and specific recommendations."
			tool.InputSchema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"config_yaml": map[string]interface{}{
						"type":        "string",
						"description": "The OpenTelemetry Collector configuration in YAML format to validate",
					},
				},
				"required": []string{"config_yaml"},
			}
		case "get_config_insights":
			tool.Description = "Provides detailed insights and optimization recommendations for an OpenTelemetry Collector configuration. Analyzes processor usage (batch, memory_limiter, resource), pipeline complexity, and identifies potential performance issues. Returns 💡 recommendations for improvements and ⚠️ warnings about problematic patterns (e.g., debug exporters in production, too many processors)."
			tool.InputSchema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"config_yaml": map[string]interface{}{
						"type":        "string",
						"description": "The OpenTelemetry Collector configuration in YAML format to analyze",
					},
				},
				"required": []string{"config_yaml"},
			}
		case "list_agents":
			tool.Description = "Lists all connected OpenTelemetry Collector agents from the live OpAMP API server (default: http://localhost:9081). Returns a formatted list showing agent IDs and names. Useful for discovering which agents are currently connected before querying specific agent details. Returns 'No agents found' message if none are connected."
			tool.InputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		case "get_agent_config":
			tool.Description = "Retrieves the complete configuration for a specific agent by ID from the live OpAMP API server. Returns agent name, labels/metadata, and the full current configuration YAML. Use this to inspect what configuration a specific agent is running. The configuration is returned as formatted YAML in a code block for easy reading."
			tool.InputSchema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "The unique agent ID to retrieve configuration for. Use list_agents tool first to get available agent IDs.",
					},
				},
				"required": []string{"agent_id"},
			}
		case "get_topology":
			tool.Description = "Retrieves the complete topology graph from the live OpAMP API server showing all agents, receivers, processors, exporters, and their data flow connections. Returns a structured overview grouped by node type (agent, receiver, processor, exporter) with counts, and shows edge connections between components including pipeline types (traces, logs, metrics). Shows first 10 edges by default with total count. Use this to understand the complete data flow architecture across all connected agents."
			tool.InputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		case "get_agent_summary":
			tool.Description = "Retrieves a comprehensive summary overview of all agents from the live OpAMP API server including health status, connection status, metrics (spans, logs, metrics received/sent), and agent metadata. Returns formatted JSON with detailed information about each agent's current state. Use this to get a high-level view of your entire OpAMP fleet's health and activity."
			tool.InputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		case "get_pipelines":
			tool.Description = "Retrieves parsed pipeline information for all agents from the live OpAMP API server. Returns detailed information about each agent's pipelines including: pipeline types (traces, logs, metrics), the ordered list of receivers, processors, and exporters in each pipeline, and endpoint information. Returns formatted JSON. Use this to understand the data processing flow for each agent without needing to parse raw configuration YAML."
			tool.InputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		tools = append(tools, tool)
	}

	return map[string]interface{}{
		"tools": tools,
	}
}

// handleToolCall handles a tool call request
func (s *Server) handleToolCall(params json.RawMessage) (interface{}, error) {
	var toolParams ToolCallParams
	if err := json.Unmarshal(params, &toolParams); err != nil {
		return nil, fmt.Errorf("invalid tool call parameters: %w", err)
	}

	handler, ok := s.tools[toolParams.Name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", toolParams.Name)
	}

	return handler(toolParams.Arguments)
}
