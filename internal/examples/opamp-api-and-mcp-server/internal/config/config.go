package config

import (
	"log"
	"strings"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
	"gopkg.in/yaml.v3"
)

// Service handles parsing of agent configurations
type Service struct{}

// New creates a new config service
func New() *Service {
	return &Service{}
}

// ParsePipelineConfig parses OpenTelemetry pipeline configuration from JSON agent
func (s *Service) ParsePipelineConfig(agent *types.JSONAgent) *types.AgentPipelineInfo {
	info := &types.AgentPipelineInfo{
		AgentID:    agent.ID,
		AgentName:  s.GetAgentName(agent),
		Pipelines:  make(map[string]*types.PipelineConfig),
		Receivers:  make(map[string]interface{}),
		Processors: make(map[string]interface{}),
		Exporters:  make(map[string]interface{}),
	}

	// Get config YAML
	configYAML := agent.CurrentConfig
	if configYAML == "" {
		return info
	}

	info.ConfigYAML = configYAML

	// Parse YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		log.Printf("Failed to parse YAML config for agent %s: %v", info.AgentID, err)
		return info
	}

	// Extract receivers
	if receivers, ok := config["receivers"].(map[string]interface{}); ok {
		info.Receivers = receivers
	}

	// Extract processors
	if processors, ok := config["processors"].(map[string]interface{}); ok {
		info.Processors = processors
	}

	// Extract exporters
	if exporters, ok := config["exporters"].(map[string]interface{}); ok {
		info.Exporters = exporters

		// Extract exporter endpoints
		for name, exporterConfig := range exporters {
			if exporterMap, ok := exporterConfig.(map[string]interface{}); ok {
				if endpoint, ok := exporterMap["endpoint"].(string); ok {
					if info.Pipelines["default"] == nil {
						info.Pipelines["default"] = &types.PipelineConfig{
							Endpoints: make(map[string]string),
						}
					}
					info.Pipelines["default"].Endpoints[name] = endpoint
				}
			}
		}
	}

	// Extract service pipelines
	if service, ok := config["service"].(map[string]interface{}); ok {
		if pipelines, ok := service["pipelines"].(map[string]interface{}); ok {
			for pipelineType, pipelineConfig := range pipelines {
				if pipelineMap, ok := pipelineConfig.(map[string]interface{}); ok {
					pipeline := &types.PipelineConfig{
						Receivers:  []string{},
						Processors: []string{},
						Exporters:  []string{},
						Endpoints:  make(map[string]string),
					}

					if receivers, ok := pipelineMap["receivers"].([]interface{}); ok {
						for _, r := range receivers {
							if receiverName, ok := r.(string); ok {
								pipeline.Receivers = append(pipeline.Receivers, receiverName)
							}
						}
					}

					if processors, ok := pipelineMap["processors"].([]interface{}); ok {
						for _, p := range processors {
							if processorName, ok := p.(string); ok {
								pipeline.Processors = append(pipeline.Processors, processorName)
							}
						}
					}

					if exporters, ok := pipelineMap["exporters"].([]interface{}); ok {
						for _, e := range exporters {
							if exporterName, ok := e.(string); ok {
								pipeline.Exporters = append(pipeline.Exporters, exporterName)

								// Get endpoint for this exporter
								if info.Exporters != nil {
									if exporterConfig, ok := info.Exporters[exporterName].(map[string]interface{}); ok {
										if endpoint, ok := exporterConfig["endpoint"].(string); ok {
											pipeline.Endpoints[exporterName] = endpoint
										}
									}
								}
							}
						}
					}

					info.Pipelines[pipelineType] = pipeline
				}
			}
		}
	}

	return info
}

// GetAgentName extracts agent name from labels
func (s *Service) GetAgentName(agent *types.JSONAgent) string {
	if agent.Labels != nil {
		if name, ok := agent.Labels["service.name"]; ok && name != "" {
			return name
		}
		if name, ok := agent.Labels["name"]; ok && name != "" {
			return name
		}
	}
	return agent.ID
}

// NormalizeEndpoint normalizes endpoint string for use as node ID
func (s *Service) NormalizeEndpoint(endpoint string) string {
	endpoint = strings.ReplaceAll(endpoint, "http://", "")
	endpoint = strings.ReplaceAll(endpoint, "https://", "")
	endpoint = strings.ReplaceAll(endpoint, "://", "")
	endpoint = strings.ReplaceAll(endpoint, "/", "-")
	endpoint = strings.ReplaceAll(endpoint, ":", "-")
	return endpoint
}

// GetAgentJobLabel determines the Prometheus job/component label for an agent
// This is used to filter metrics by agent when querying Prometheus
func (s *Service) GetAgentJobLabel(agent *types.JSONAgent) string {
	agentName := s.GetAgentName(agent)
	// Check if agent name contains "otel-gateway" or "gateway"
	if strings.Contains(strings.ToLower(agentName), "otel-gateway") || strings.Contains(strings.ToLower(agentName), "gateway") {
		return "otel-gateway"
	}
	// Check if agent name contains "otel-agent" or "agent"
	if strings.Contains(strings.ToLower(agentName), "otel-agent") || strings.Contains(strings.ToLower(agentName), "agent") {
		return "otel-agent"
	}
	// Default: try to extract from service.name label
	if agent.Labels != nil {
		if serviceName, ok := agent.Labels["service.name"]; ok {
			serviceNameLower := strings.ToLower(serviceName)
			if strings.Contains(serviceNameLower, "otel-gateway") || strings.Contains(serviceNameLower, "gateway") {
				return "otel-gateway"
			}
			if strings.Contains(serviceNameLower, "otel-agent") || strings.Contains(serviceNameLower, "agent") {
				return "otel-agent"
			}
		}
	}
	// Default fallback: assume otel-agent
	return "otel-agent"
}
