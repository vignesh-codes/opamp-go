package metrics

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/prometheus"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
)

// Service handles scraping and parsing of OpenTelemetry Collector metrics
type Service struct {
	httpClient        *http.Client
	prometheusService *prometheus.Service
}

// New creates a new metrics service
func New() *Service {
	return &Service{
		httpClient:        &http.Client{Timeout: 5 * time.Second},
		prometheusService: nil, // Will be set by SetPrometheusService if needed
	}
}

// SetPrometheusService sets the Prometheus service for fetching metrics from Prometheus
func (s *Service) SetPrometheusService(promService *prometheus.Service) {
	s.prometheusService = promService
}

// ScrapeCollectorMetrics scrapes metrics from Prometheus (preferred) or returns empty metrics
func (s *Service) ScrapeCollectorMetrics(agent *types.JSONAgent, pipelineInfo *types.AgentPipelineInfo) *types.TelemetryMetrics {
	metrics := &types.TelemetryMetrics{
		Receivers:  make(map[string]map[string]interface{}),
		Exporters:  make(map[string]map[string]interface{}),
		Processors: make(map[string]map[string]interface{}),
		ByPipeline: make(map[string]map[string]interface{}),
		Totals:     make(map[string]interface{}),
	}

	// Try to fetch from Prometheus first (preferred method)
	if s.prometheusService != nil {
		err := s.fetchMetricsFromPrometheus(metrics, pipelineInfo)
		if err == nil {
			// Successfully fetched from Prometheus
			return metrics
		}
		log.Printf("[DEBUG] Failed to fetch metrics from Prometheus: %v", err)
	}

	// Return empty metrics if Prometheus is not available
	return metrics
}

// fetchMetricsFromPrometheus queries Prometheus for cumulative counter values
func (s *Service) fetchMetricsFromPrometheus(metrics *types.TelemetryMetrics, pipelineInfo *types.AgentPipelineInfo) error {
	// Query receiver metrics (cumulative counters)
	receiverQueries := map[string]func(string, float64){
		`sum by(receiver) (otelcol_receiver_accepted_spans_total)`: func(receiver string, value float64) {
			if metrics.Receivers[receiver] == nil {
				metrics.Receivers[receiver] = make(map[string]interface{})
			}
			metrics.Receivers[receiver]["spans_received"] = value
		},
		`sum by(receiver) (otelcol_receiver_accepted_log_records_total)`: func(receiver string, value float64) {
			if metrics.Receivers[receiver] == nil {
				metrics.Receivers[receiver] = make(map[string]interface{})
			}
			metrics.Receivers[receiver]["logs_received"] = value
		},
		`sum by(receiver) (otelcol_receiver_accepted_metric_points_total)`: func(receiver string, value float64) {
			if metrics.Receivers[receiver] == nil {
				metrics.Receivers[receiver] = make(map[string]interface{})
			}
			metrics.Receivers[receiver]["metrics_received"] = value
		},
	}

	// Query exporter metrics
	exporterQueries := map[string]func(string, float64){
		`sum by(exporter) (otelcol_exporter_sent_spans_total)`: func(exporter string, value float64) {
			if metrics.Exporters[exporter] == nil {
				metrics.Exporters[exporter] = make(map[string]interface{})
			}
			metrics.Exporters[exporter]["spans_sent"] = value
		},
		`sum by(exporter) (otelcol_exporter_sent_log_records_total)`: func(exporter string, value float64) {
			if metrics.Exporters[exporter] == nil {
				metrics.Exporters[exporter] = make(map[string]interface{})
			}
			metrics.Exporters[exporter]["logs_sent"] = value
		},
		`sum by(exporter) (otelcol_exporter_sent_metric_points_total)`: func(exporter string, value float64) {
			if metrics.Exporters[exporter] == nil {
				metrics.Exporters[exporter] = make(map[string]interface{})
			}
			metrics.Exporters[exporter]["metrics_sent"] = value
		},
	}

	// Query processor metrics
	// NOTE: Label name is "otel.signal" (with dot) - must use quotes around label name in PromQL
	// Syntax: {"otel.signal"="metrics"} - quotes around label name are required for labels with dots
	processorQueries := map[string]func(string, float64){
		// Traces processors
		`sum by(processor) (otelcol_processor_incoming_items_total{"otel.signal"="traces"})`: func(processor string, value float64) {
			if processor == "" {
				return
			}
			if metrics.Processors[processor] == nil {
				metrics.Processors[processor] = make(map[string]interface{})
			}
			metrics.Processors[processor]["spans_accepted"] = value
		},
		`sum by(processor) (otelcol_processor_outgoing_items_total{"otel.signal"="traces"})`: func(processor string, value float64) {
			if processor == "" {
				return
			}
			if metrics.Processors[processor] == nil {
				metrics.Processors[processor] = make(map[string]interface{})
			}
			metrics.Processors[processor]["spans_sent"] = value
		},
		// Logs processors
		`sum by(processor) (otelcol_processor_incoming_items_total{"otel.signal"="logs"})`: func(processor string, value float64) {
			if processor == "" {
				return
			}
			if metrics.Processors[processor] == nil {
				metrics.Processors[processor] = make(map[string]interface{})
			}
			metrics.Processors[processor]["logs_accepted"] = value
		},
		`sum by(processor) (otelcol_processor_outgoing_items_total{"otel.signal"="logs"})`: func(processor string, value float64) {
			if processor == "" {
				return
			}
			if metrics.Processors[processor] == nil {
				metrics.Processors[processor] = make(map[string]interface{})
			}
			metrics.Processors[processor]["logs_sent"] = value
		},
		// Metrics processors
		`sum by(processor) (otelcol_processor_incoming_items_total{"otel.signal"="metrics"})`: func(processor string, value float64) {
			if processor == "" {
				return
			}
			if metrics.Processors[processor] == nil {
				metrics.Processors[processor] = make(map[string]interface{})
			}
			metrics.Processors[processor]["metrics_accepted"] = value
		},
		`sum by(processor) (otelcol_processor_outgoing_items_total{"otel.signal"="metrics"})`: func(processor string, value float64) {
			if processor == "" {
				return
			}
			if metrics.Processors[processor] == nil {
				metrics.Processors[processor] = make(map[string]interface{})
			}
			metrics.Processors[processor]["metrics_sent"] = value
		},
	}

	// Execute queries (continue on error to get as much data as possible)
	_ = s.executePrometheusQueriesForMetrics(receiverQueries)
	_ = s.executePrometheusQueriesForMetrics(exporterQueries)
	_ = s.executePrometheusQueriesForMetrics(processorQueries)

	// Calculate totals
	var totalSpansReceived, totalSpansSent float64
	var totalLogsReceived, totalLogsSent float64
	var totalMetricsReceived, totalMetricsSent float64

	for _, receiverData := range metrics.Receivers {
		if v, ok := receiverData["spans_received"].(float64); ok {
			totalSpansReceived += v
		}
		if v, ok := receiverData["logs_received"].(float64); ok {
			totalLogsReceived += v
		}
		if v, ok := receiverData["metrics_received"].(float64); ok {
			totalMetricsReceived += v
		}
	}

	for _, exporterData := range metrics.Exporters {
		if v, ok := exporterData["spans_sent"].(float64); ok {
			totalSpansSent += v
		}
		if v, ok := exporterData["logs_sent"].(float64); ok {
			totalLogsSent += v
		}
		if v, ok := exporterData["metrics_sent"].(float64); ok {
			totalMetricsSent += v
		}
	}

	metrics.Totals["spans_received"] = totalSpansReceived
	metrics.Totals["spans_sent"] = totalSpansSent
	metrics.Totals["logs_received"] = totalLogsReceived
	metrics.Totals["logs_sent"] = totalLogsSent
	metrics.Totals["metrics_received"] = totalMetricsReceived
	metrics.Totals["metrics_sent"] = totalMetricsSent

	// Build pipeline data
	s.buildPipelineData(pipelineInfo, metrics)

	// Return error if no data was retrieved to trigger fallback
	if len(metrics.Receivers) == 0 && len(metrics.Exporters) == 0 && len(metrics.Processors) == 0 {
		return log.Output(2, "no metrics data retrieved from Prometheus")
	}

	return nil
}

// executePrometheusQueriesForMetrics executes Prometheus queries and calls handlers with component name and value
func (s *Service) executePrometheusQueriesForMetrics(queries map[string]func(string, float64)) error {
	for query, handler := range queries {
		result, err := s.prometheusService.Query(query)
		if err != nil {
			log.Printf("[DEBUG] Prometheus query failed for %s: %v", query, err)
			continue
		}

		for _, res := range result.Data.Result {
			var componentName string
			if receiver, ok := res.Metric["receiver"]; ok {
				componentName = receiver
			} else if exporter, ok := res.Metric["exporter"]; ok {
				componentName = exporter
			} else {
				continue
			}

			if len(res.Value) < 2 {
				continue
			}

			var value float64
			switch v := res.Value[1].(type) {
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				value = parsed
			case float64:
				value = v
			default:
				continue
			}

			handler(componentName, value)
		}
	}
	return nil
}

// executePrometheusQueriesForMetricsWithLabels executes queries that need full label maps
func (s *Service) executePrometheusQueriesForMetricsWithLabels(queries map[string]func(map[string]string, float64)) error {
	for query, handler := range queries {
		result, err := s.prometheusService.Query(query)
		if err != nil {
			log.Printf("[DEBUG] Prometheus query failed for %s: %v", query, err)
			continue
		}

		for _, res := range result.Data.Result {
			if len(res.Value) < 2 {
				continue
			}

			var value float64
			switch v := res.Value[1].(type) {
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				value = parsed
			case float64:
				value = v
			default:
				continue
			}

			handler(res.Metric, value)
		}
	}
	return nil
}

// buildPipelineData builds pipeline-level metrics from component metrics
func (s *Service) buildPipelineData(pipelineInfo *types.AgentPipelineInfo, metrics *types.TelemetryMetrics) {
	pipelineDataMap := make(map[string]map[string]interface{})
	for pipelineType, pipeline := range pipelineInfo.Pipelines {
		pipelineData := make(map[string]interface{})

		// Sum receiver data for this pipeline
		var pipelineSpansReceived, pipelineLogsReceived, pipelineMetricsReceived float64
		for _, receiverName := range pipeline.Receivers {
			if receiverData, ok := metrics.Receivers[receiverName]; ok {
				if v, ok := receiverData["spans_received"].(float64); ok {
					pipelineSpansReceived += v
				}
				if v, ok := receiverData["logs_received"].(float64); ok {
					pipelineLogsReceived += v
				}
				if v, ok := receiverData["metrics_received"].(float64); ok {
					pipelineMetricsReceived += v
				}
			}
		}

		pipelineData["spans_received"] = pipelineSpansReceived
		pipelineData["logs_received"] = pipelineLogsReceived
		pipelineData["metrics_received"] = pipelineMetricsReceived
		pipelineData["spans_sent"] = pipelineSpansReceived
		pipelineData["logs_sent"] = pipelineLogsReceived
		pipelineData["metrics_sent"] = pipelineMetricsReceived

		pipelineDataMap[pipelineType] = pipelineData
	}

	// Add non-empty pipelines to metrics
	for pipelineType, pipelineData := range pipelineDataMap {
		spans := pipelineData["spans_received"].(float64)
		logs := pipelineData["logs_received"].(float64)
		metricsCount := pipelineData["metrics_received"].(float64)

		if spans > 0 || logs > 0 || metricsCount > 0 {
			metrics.ByPipeline[pipelineType] = pipelineData
		}
	}
}
