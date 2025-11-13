package metrics

import (
	"testing"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/prometheus"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func TestSetPrometheusService(t *testing.T) {
	s := New()
	promService := prometheus.New("http://localhost:9090")

	s.SetPrometheusService(promService)

	if s.prometheusService == nil {
		t.Error("Prometheus service was not set")
	}
}

func TestScrapeCollectorMetrics_NoPrometheus(t *testing.T) {
	s := New()

	agent := &types.JSONAgent{
		ID:            "test-agent",
		Labels:        map[string]string{},
		CurrentConfig: "",
	}

	pipelineInfo := &types.AgentPipelineInfo{
		AgentID:   "test-agent",
		AgentName: "test",
		Pipelines: make(map[string]*types.PipelineConfig),
	}

	metrics := s.ScrapeCollectorMetrics(agent, pipelineInfo)

	if metrics == nil {
		t.Fatal("ScrapeCollectorMetrics returned nil")
	}

	if metrics.Receivers == nil {
		t.Error("Expected Receivers map to be initialized")
	}

	if metrics.Exporters == nil {
		t.Error("Expected Exporters map to be initialized")
	}

	if metrics.Processors == nil {
		t.Error("Expected Processors map to be initialized")
	}
}
