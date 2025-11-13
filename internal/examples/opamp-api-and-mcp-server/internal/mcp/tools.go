package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// handleAnalyzeConfig analyzes an OpenTelemetry Collector configuration
func (s *Server) handleAnalyzeConfig(args map[string]interface{}) (interface{}, error) {
	configYAML, ok := args["config_yaml"].(string)
	if !ok {
		return nil, fmt.Errorf("config_yaml must be a string")
	}

	analyzer := NewConfigAnalyzer()
	analysis := analyzer.Analyze(configYAML)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": analysis,
			},
		},
	}, nil
}

// handleValidateConfig validates an OpenTelemetry Collector configuration
func (s *Server) handleValidateConfig(args map[string]interface{}) (interface{}, error) {
	configYAML, ok := args["config_yaml"].(string)
	if !ok {
		return nil, fmt.Errorf("config_yaml must be a string")
	}

	validator := NewConfigValidator()
	validation := validator.Validate(configYAML)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": validation,
			},
		},
	}, nil
}

// handleGetConfigInsights provides insights for an OpenTelemetry Collector configuration
func (s *Server) handleGetConfigInsights(args map[string]interface{}) (interface{}, error) {
	configYAML, ok := args["config_yaml"].(string)
	if !ok {
		return nil, fmt.Errorf("config_yaml must be a string")
	}

	insights := NewConfigInsights()
	insightsText := insights.GetInsights(configYAML)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": insightsText,
			},
		},
	}, nil
}

// handleListAgents lists all agents from the live API
func (s *Server) handleListAgents(args map[string]interface{}) (interface{}, error) {
	agents, err := s.apiClient.GetList("/api/agents")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agents from API: %w", err)
	}

	if len(agents) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "No agents found. Make sure agents are connected to the OpAMP server.",
				},
			},
		}, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d agent(s):\n\n", len(agents)))

	for _, agentRaw := range agents {
		if agent, ok := agentRaw.(map[string]interface{}); ok {
			agentID, _ := agent["id"].(string)
			agentName, _ := agent["name"].(string)
			if agentID != "" {
				result.WriteString(fmt.Sprintf("- **%s**", agentID))
				if agentName != "" {
					result.WriteString(fmt.Sprintf(" (%s)", agentName))
				}
				result.WriteString("\n")
			}
		}
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

// handleGetAgentConfig gets the configuration for a specific agent from the live API
func (s *Server) handleGetAgentConfig(args map[string]interface{}) (interface{}, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok {
		return nil, fmt.Errorf("agent_id must be a string")
	}

	agent, err := s.apiClient.Get(fmt.Sprintf("/api/agents/%s", agentID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent from API: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("## Agent: %s\n\n", agentID))

	if name, ok := agent["name"].(string); ok && name != "" {
		result.WriteString(fmt.Sprintf("**Name:** %s\n\n", name))
	}

	if labels, ok := agent["labels"].(map[string]interface{}); ok && len(labels) > 0 {
		result.WriteString("### Labels\n\n")
		for key, value := range labels {
			result.WriteString(fmt.Sprintf("- `%s`: %v\n", key, value))
		}
		result.WriteString("\n")
	}

	if config, ok := agent["current_config"].(string); ok && config != "" {
		result.WriteString("### Configuration\n\n")
		result.WriteString("```yaml\n")
		result.WriteString(config)
		result.WriteString("\n```\n")
	} else {
		result.WriteString("### Configuration\n\n*No configuration available*\n")
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

// handleGetTopology gets the complete topology graph from the live API
func (s *Server) handleGetTopology(args map[string]interface{}) (interface{}, error) {
	topology, err := s.apiClient.Get("/api/topology")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch topology graph from OpAMP API server at %s/api/topology. Make sure the OpAMP server is running and accessible. Error: %w", s.apiClient.BaseURL(), err)
	}

	// Format topology information
	var result strings.Builder
	result.WriteString("# Topology Overview\n\n")

	if nodes, ok := topology["nodes"].([]interface{}); ok {
		result.WriteString(fmt.Sprintf("## Nodes (%d total)\n\n", len(nodes)))

		// Group by type
		byType := make(map[string][]interface{})
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				nodeType, _ := nodeMap["type"].(string)
				if nodeType == "" {
					nodeType = "unknown"
				}
				byType[nodeType] = append(byType[nodeType], node)
			}
		}

		for nodeType, typeNodes := range byType {
			result.WriteString(fmt.Sprintf("### %s (%d)\n\n", nodeType, len(typeNodes)))
			for _, node := range typeNodes {
				if nodeMap, ok := node.(map[string]interface{}); ok {
					name, _ := nodeMap["name"].(string)
					agentID, _ := nodeMap["agent_id"].(string)
					if name != "" {
						result.WriteString(fmt.Sprintf("- **%s**", name))
						if agentID != "" {
							result.WriteString(fmt.Sprintf(" (Agent: %s)", agentID))
						}
						result.WriteString("\n")
					}
				}
			}
			result.WriteString("\n")
		}
	}

	if edges, ok := topology["edges"].([]interface{}); ok {
		result.WriteString(fmt.Sprintf("## Edges (%d total)\n\n", len(edges)))
		result.WriteString("Connections between components:\n\n")
		for i, edge := range edges {
			if i < 10 { // Show first 10 edges
				if edgeMap, ok := edge.(map[string]interface{}); ok {
					from, _ := edgeMap["from"].(string)
					to, _ := edgeMap["to"].(string)
					pipelineType, _ := edgeMap["pipeline_type"].(string)
					if from != "" && to != "" {
						result.WriteString(fmt.Sprintf("- `%s` → `%s`", from, to))
						if pipelineType != "" {
							result.WriteString(fmt.Sprintf(" (%s)", pipelineType))
						}
						result.WriteString("\n")
					}
				}
			}
		}
		if len(edges) > 10 {
			result.WriteString(fmt.Sprintf("\n... and %d more edges\n", len(edges)-10))
		}
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

// handleGetAgentSummary gets a summary overview of all agents
func (s *Server) handleGetAgentSummary(args map[string]interface{}) (interface{}, error) {
	summary, err := s.apiClient.Get("/api/summary")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent summary from OpAMP API server at %s/api/summary. Make sure the OpAMP server is running and accessible. Error: %w", s.apiClient.BaseURL(), err)
	}

	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format summary JSON: %w", err)
	}

	var result strings.Builder
	result.WriteString("# Agent Summary Overview\n\n")
	result.WriteString("This summary provides a high-level view of all connected agents in your OpAMP fleet.\n\n")
	result.WriteString("## Summary Data\n\n")
	result.WriteString("```json\n")
	result.WriteString(string(summaryJSON))
	result.WriteString("\n```\n\n")
	result.WriteString("**Note:** This data includes health status, connection information, and metrics for each agent.\n")

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

// handleGetPipelines gets parsed pipeline information
func (s *Server) handleGetPipelines(args map[string]interface{}) (interface{}, error) {
	pipelines, err := s.apiClient.Get("/api/pipelines")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline information from OpAMP API server at %s/api/pipelines. Make sure the OpAMP server is running and accessible. Error: %w", s.apiClient.BaseURL(), err)
	}

	pipelinesJSON, err := json.MarshalIndent(pipelines, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format pipeline JSON: %w", err)
	}

	var result strings.Builder
	result.WriteString("# Pipeline Information\n\n")
	result.WriteString("This data shows the parsed pipeline configuration for all agents, including:\n\n")
	result.WriteString("- **Pipeline Types**: traces, logs, metrics\n")
	result.WriteString("- **Component Order**: The sequence of receivers → processors → exporters\n")
	result.WriteString("- **Endpoint Information**: Where data is being exported\n\n")
	result.WriteString("## Pipeline Data\n\n")
	result.WriteString("```json\n")
	result.WriteString(string(pipelinesJSON))
	result.WriteString("\n```\n\n")
	result.WriteString("**Note:** Use this information to understand the data flow architecture without parsing raw YAML configurations.\n")

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

// ConfigAnalyzer analyzes OpenTelemetry Collector configurations
type ConfigAnalyzer struct{}

// NewConfigAnalyzer creates a new config analyzer
func NewConfigAnalyzer() *ConfigAnalyzer {
	return &ConfigAnalyzer{}
}

// Analyze analyzes a configuration and returns insights
func (a *ConfigAnalyzer) Analyze(configYAML string) string {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return fmt.Sprintf("Error parsing YAML: %v", err)
	}

	var analysis strings.Builder
	analysis.WriteString("# OpenTelemetry Collector Configuration Analysis\n\n")

	// Analyze receivers
	if receivers, ok := config["receivers"].(map[string]interface{}); ok {
		analysis.WriteString(fmt.Sprintf("## Receivers (%d)\n\n", len(receivers)))
		for name := range receivers {
			analysis.WriteString(fmt.Sprintf("- %s\n", name))
		}
		analysis.WriteString("\n")
	}

	// Analyze processors
	if processors, ok := config["processors"].(map[string]interface{}); ok {
		analysis.WriteString(fmt.Sprintf("## Processors (%d)\n\n", len(processors)))
		for name := range processors {
			analysis.WriteString(fmt.Sprintf("- %s\n", name))
		}
		analysis.WriteString("\n")
	}

	// Analyze exporters
	if exporters, ok := config["exporters"].(map[string]interface{}); ok {
		analysis.WriteString(fmt.Sprintf("## Exporters (%d)\n\n", len(exporters)))
		for name := range exporters {
			analysis.WriteString(fmt.Sprintf("- %s\n", name))
		}
		analysis.WriteString("\n")
	}

	// Analyze pipelines
	if service, ok := config["service"].(map[string]interface{}); ok {
		if pipelines, ok := service["pipelines"].(map[string]interface{}); ok {
			analysis.WriteString(fmt.Sprintf("## Pipelines (%d)\n\n", len(pipelines)))
			for pipelineType, pipelineConfig := range pipelines {
				if pipelineMap, ok := pipelineConfig.(map[string]interface{}); ok {
					analysis.WriteString(fmt.Sprintf("### %s Pipeline\n", pipelineType))

					if receivers, ok := pipelineMap["receivers"].([]interface{}); ok {
						analysis.WriteString(fmt.Sprintf("Receivers: %v\n", receivers))
					}
					if processors, ok := pipelineMap["processors"].([]interface{}); ok {
						analysis.WriteString(fmt.Sprintf("Processors: %v\n", processors))
					}
					if exporters, ok := pipelineMap["exporters"].([]interface{}); ok {
						analysis.WriteString(fmt.Sprintf("Exporters: %v\n", exporters))
					}
					analysis.WriteString("\n")
				}
			}
		}
	}

	return analysis.String()
}

// ConfigValidator validates OpenTelemetry Collector configurations
type ConfigValidator struct{}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate validates a configuration and returns validation results
func (v *ConfigValidator) Validate(configYAML string) string {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return fmt.Sprintf("❌ Invalid YAML: %v", err)
	}

	var validation strings.Builder
	validation.WriteString("# Configuration Validation Results\n\n")

	issues := []string{}
	warnings := []string{}

	// Check for required sections
	if _, ok := config["receivers"]; !ok {
		warnings = append(warnings, "No receivers defined")
	}
	if _, ok := config["exporters"]; !ok {
		warnings = append(warnings, "No exporters defined")
	}
	if _, ok := config["service"]; !ok {
		issues = append(issues, "Missing required 'service' section")
	}

	// Check service pipelines
	if service, ok := config["service"].(map[string]interface{}); ok {
		if pipelines, ok := service["pipelines"].(map[string]interface{}); ok {
			if len(pipelines) == 0 {
				issues = append(issues, "No pipelines defined in service section")
			}

			// Validate each pipeline
			for pipelineType, pipelineConfig := range pipelines {
				if pipelineMap, ok := pipelineConfig.(map[string]interface{}); ok {
					receivers, _ := pipelineMap["receivers"].([]interface{})
					exporters, _ := pipelineMap["exporters"].([]interface{})

					if len(receivers) == 0 {
						issues = append(issues, fmt.Sprintf("Pipeline '%s' has no receivers", pipelineType))
					}
					if len(exporters) == 0 {
						issues = append(issues, fmt.Sprintf("Pipeline '%s' has no exporters", pipelineType))
					}
				}
			}
		} else {
			issues = append(issues, "No pipelines defined in service section")
		}
	}

	if len(issues) == 0 && len(warnings) == 0 {
		validation.WriteString("✅ Configuration is valid!\n\n")
	} else {
		if len(issues) > 0 {
			validation.WriteString("## ❌ Issues\n\n")
			for _, issue := range issues {
				validation.WriteString(fmt.Sprintf("- %s\n", issue))
			}
			validation.WriteString("\n")
		}

		if len(warnings) > 0 {
			validation.WriteString("## ⚠️ Warnings\n\n")
			for _, warning := range warnings {
				validation.WriteString(fmt.Sprintf("- %s\n", warning))
			}
			validation.WriteString("\n")
		}
	}

	return validation.String()
}

// ConfigInsights provides insights for OpenTelemetry Collector configurations
type ConfigInsights struct{}

// NewConfigInsights creates a new config insights provider
func NewConfigInsights() *ConfigInsights {
	return &ConfigInsights{}
}

// GetInsights provides insights and recommendations
func (i *ConfigInsights) GetInsights(configYAML string) string {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return fmt.Sprintf("Error parsing YAML: %v", err)
	}

	var insights strings.Builder
	insights.WriteString("# Configuration Insights & Recommendations\n\n")

	// Check for batch processor
	if processors, ok := config["processors"].(map[string]interface{}); ok {
		if _, hasBatch := processors["batch"]; !hasBatch {
			insights.WriteString("## 💡 Recommendation\n\n")
			insights.WriteString("Consider adding a `batch` processor to improve performance by batching telemetry data before exporting.\n\n")
		}
	}

	// Check for memory limiter
	if processors, ok := config["processors"].(map[string]interface{}); ok {
		if _, hasMemoryLimiter := processors["memory_limiter"]; !hasMemoryLimiter {
			insights.WriteString("## 💡 Recommendation\n\n")
			insights.WriteString("Consider adding a `memory_limiter` processor to prevent out-of-memory issues.\n\n")
		}
	}

	// Check for resource processor
	if processors, ok := config["processors"].(map[string]interface{}); ok {
		if _, hasResource := processors["resource"]; !hasResource {
			insights.WriteString("## 💡 Recommendation\n\n")
			insights.WriteString("Consider adding a `resource` processor to enrich telemetry data with common attributes.\n\n")
		}
	}

	// Analyze pipeline complexity
	if service, ok := config["service"].(map[string]interface{}); ok {
		if pipelines, ok := service["pipelines"].(map[string]interface{}); ok {
			for pipelineType, pipelineConfig := range pipelines {
				if pipelineMap, ok := pipelineConfig.(map[string]interface{}); ok {
					if processors, ok := pipelineMap["processors"].([]interface{}); ok {
						if len(processors) > 5 {
							insights.WriteString("## ⚠️ Warning\n\n")
							insights.WriteString(fmt.Sprintf("The '%s' pipeline has %d processors, which may impact performance. Consider consolidating or optimizing the processor chain.\n\n", pipelineType, len(processors)))
						}
					}
				}
			}
		}
	}

	// Check for debug exporter
	if exporters, ok := config["exporters"].(map[string]interface{}); ok {
		if _, hasDebug := exporters["debug"]; hasDebug {
			insights.WriteString("## ⚠️ Warning\n\n")
			insights.WriteString("A `debug` exporter is configured. This should only be used for development/testing, not production.\n\n")
		}
	}

	if insights.Len() == len("# Configuration Insights & Recommendations\n\n") {
		insights.WriteString("✅ No specific recommendations at this time. Your configuration looks good!")
	}

	return insights.String()
}
