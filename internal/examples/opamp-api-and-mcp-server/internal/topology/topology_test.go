package topology

import (
	"testing"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/config"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/metrics"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/prometheus"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
)

func TestNew(t *testing.T) {
	metricsService := metrics.New()
	prometheusService := prometheus.New("")
	configService := config.New()

	s := New(metricsService, prometheusService, configService)
	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func TestBuildTopology_EmptyAgents(t *testing.T) {
	metricsService := metrics.New()
	prometheusService := prometheus.New("")
	configService := config.New()

	s := New(metricsService, prometheusService, configService)

	agents := make(map[string]*types.JSONAgent)
	topology, err := s.BuildTopology(agents)

	if err != nil {
		t.Fatalf("BuildTopology failed: %v", err)
	}

	if len(topology.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for empty agents, got %d", len(topology.Nodes))
	}

	if len(topology.Edges) != 0 {
		t.Errorf("Expected 0 edges for empty agents, got %d", len(topology.Edges))
	}
}

func TestBuildTopology_WithAgent(t *testing.T) {
	metricsService := metrics.New()
	prometheusService := prometheus.New("")
	configService := config.New()

	s := New(metricsService, prometheusService, configService)

	agents := map[string]*types.JSONAgent{
		"test-agent-1": {
			ID:            "test-agent-1",
			Labels:        map[string]string{"service.name": "test-service"},
			CurrentConfig: "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 0.0.0.0:4317\nprocessors:\n  batch:\nexporters:\n  otlp:\n    endpoint: http://collector:4318\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      processors: [batch]\n      exporters: [otlp]\n",
		},
	}

	topology, err := s.BuildTopology(agents)

	if err != nil {
		t.Fatalf("BuildTopology failed: %v", err)
	}

	if len(topology.Nodes) == 0 {
		t.Error("Expected at least one node (the agent), got 0")
	}

	// Check that we have an agent node
	hasAgentNode := false
	for _, node := range topology.Nodes {
		if node.Type == "agent" {
			hasAgentNode = true
			break
		}
	}

	if !hasAgentNode {
		t.Error("Expected to find an agent node in the topology")
	}
}
