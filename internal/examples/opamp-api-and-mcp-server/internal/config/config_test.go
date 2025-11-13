package config

import (
	"testing"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetAgentName(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		agent    *types.JSONAgent
		expected string
	}{
		{
			name: "with service.name",
			agent: &types.JSONAgent{
				ID: "test-id",
				Labels: map[string]string{
					"service.name": "my-service",
				},
			},
			expected: "my-service",
		},
		{
			name: "with name label",
			agent: &types.JSONAgent{
				ID: "test-id",
				Labels: map[string]string{
					"name": "my-name",
				},
			},
			expected: "my-name",
		},
		{
			name: "fallback to ID",
			agent: &types.JSONAgent{
				ID:     "test-id",
				Labels: map[string]string{},
			},
			expected: "test-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.GetAgentName(tt.agent)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "http endpoint",
			input:    "http://example.com:8080",
			expected: "example-com-8080",
		},
		{
			name:     "https endpoint",
			input:    "https://example.com:8080",
			expected: "example-com-8080",
		},
		{
			name:     "endpoint with path",
			input:    "http://example.com:8080/api/v1",
			expected: "example-com-8080-api-v1",
		},
		{
			name:     "simple endpoint",
			input:    "example.com:8080",
			expected: "example-com-8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.NormalizeEndpoint(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParsePipelineConfig(t *testing.T) {
	s := New()

	agent := &types.JSONAgent{
		ID:     "test-agent",
		Labels: map[string]string{"service.name": "test-service"},
		CurrentConfig: `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  batch:
exporters:
  otlp:
    endpoint: http://collector:4318
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
`,
	}

	info := s.ParsePipelineConfig(agent)

	if info.AgentID != "test-agent" {
		t.Errorf("Expected AgentID 'test-agent', got '%s'", info.AgentID)
	}

	if info.AgentName != "test-service" {
		t.Errorf("Expected AgentName 'test-service', got '%s'", info.AgentName)
	}

	if len(info.Receivers) == 0 {
		t.Error("Expected receivers to be parsed")
	}

	if len(info.Exporters) == 0 {
		t.Error("Expected exporters to be parsed")
	}

	if len(info.Pipelines) == 0 {
		t.Error("Expected pipelines to be parsed")
	}

	tracesPipeline, ok := info.Pipelines["traces"]
	if !ok {
		t.Fatal("Expected 'traces' pipeline to be parsed")
	}

	if len(tracesPipeline.Receivers) == 0 {
		t.Error("Expected traces pipeline to have receivers")
	}

	if len(tracesPipeline.Exporters) == 0 {
		t.Error("Expected traces pipeline to have exporters")
	}
}
