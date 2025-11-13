package prometheus

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
)

// Service handles queries to Prometheus API
type Service struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new Prometheus service
func New(baseURL string) *Service {
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}
	return &Service{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Query executes a PromQL query against Prometheus
func (s *Service) Query(query string) (*types.PrometheusQueryResponse, error) {
	url := s.baseURL + "/api/v1/query"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus query failed: status %d", resp.StatusCode)
	}

	var result types.PrometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: status %s", result.Status)
	}

	return &result, nil
}

// FetchTelemetryRates queries Prometheus for real-time rates
// If jobLabel is provided, rates are filtered by that job/component label
func (s *Service) FetchTelemetryRates(jobLabel string) (*types.TelemetryRates, error) {
	rates := &types.TelemetryRates{
		Receivers:  make(map[string]types.ReceiverRates),
		Processors: make(map[string]types.ProcessorRates),
		Exporters:  make(map[string]types.ExporterRates),
	}

	// Query receiver rates
	if err := s.fetchReceiverRates(rates, jobLabel); err != nil {
		log.Printf("[DEBUG] Error fetching receiver rates: %v", err)
	}

	// Query processor rates
	if err := s.fetchProcessorRates(rates, jobLabel); err != nil {
		log.Printf("[DEBUG] Error fetching processor rates: %v", err)
	}

	// Query exporter rates
	if err := s.fetchExporterRates(rates, jobLabel); err != nil {
		log.Printf("[DEBUG] Error fetching exporter rates: %v", err)
	}

	return rates, nil
}

// fetchReceiverRates queries Prometheus for receiver rates
// If jobLabel is provided, rates are filtered by that job/component label
// Note: 1m window may return empty results, so we try 5m as fallback for better reliability
func (s *Service) fetchReceiverRates(rates *types.TelemetryRates, jobLabel string) error {
	// Build job filter for PromQL query
	jobFilter := ""
	if jobLabel != "" {
		jobFilter = fmt.Sprintf(`{job="%s"}`, jobLabel)
	}

	// Try 1m window first (primary queries with correct metric names)
	primaryQueries := map[string]func(string, float64){
		fmt.Sprintf(`sum by(receiver) (rate(otelcol_receiver_accepted_log_records_total%s[1m]))`, jobFilter): func(receiver string, value float64) {
			r := rates.Receivers[receiver]
			r.LogsReceivedRate = value
			rates.Receivers[receiver] = r
		},
		fmt.Sprintf(`sum by(receiver) (rate(otelcol_receiver_accepted_metric_points_total%s[1m]))`, jobFilter): func(receiver string, value float64) {
			r := rates.Receivers[receiver]
			r.MetricsReceivedRate = value
			rates.Receivers[receiver] = r
		},
		fmt.Sprintf(`sum by(receiver) (rate(otelcol_receiver_accepted_spans_total%s[1m]))`, jobFilter): func(receiver string, value float64) {
			r := rates.Receivers[receiver]
			r.SpansReceivedRate = value
			rates.Receivers[receiver] = r
		},
	}

	// Execute 1m queries first
	result1m, err1m := s.executePrometheusQueriesWithResultCount(primaryQueries)
	if err1m != nil {
		log.Printf("[DEBUG] Error executing 1m receiver rate queries: %v", err1m)
	}

	// If 1m window returned no results, try 5m window as fallback
	// Note: 1m window should work fine for rates, but might fail if:
	// - Metrics don't exist yet (new collector)
	// - Scrape interval is longer than 1m
	// - Not enough data points in the window
	if result1m == 0 {
		log.Printf("[DEBUG] 1m window returned no results for receiver rates (result count: %d), trying 5m window as fallback", result1m)
		fallbackQueries := map[string]func(string, float64){
			fmt.Sprintf(`sum by(receiver) (rate(otelcol_receiver_accepted_log_records_total%s[5m]))`, jobFilter): func(receiver string, value float64) {
				r := rates.Receivers[receiver]
				// Only set if not already set by 1m query
				if r.LogsReceivedRate == 0 {
					r.LogsReceivedRate = value
				}
				rates.Receivers[receiver] = r
			},
			fmt.Sprintf(`sum by(receiver) (rate(otelcol_receiver_accepted_metric_points_total%s[5m]))`, jobFilter): func(receiver string, value float64) {
				r := rates.Receivers[receiver]
				// Only set if not already set by 1m query
				if r.MetricsReceivedRate == 0 {
					r.MetricsReceivedRate = value
				}
				rates.Receivers[receiver] = r
			},
			fmt.Sprintf(`sum by(receiver) (rate(otelcol_receiver_accepted_spans_total%s[5m]))`, jobFilter): func(receiver string, value float64) {
				r := rates.Receivers[receiver]
				// Only set if not already set by 1m query
				if r.SpansReceivedRate == 0 {
					r.SpansReceivedRate = value
				}
				rates.Receivers[receiver] = r
			},
		}
		if err := s.executePrometheusQueries(fallbackQueries); err != nil {
			log.Printf("[DEBUG] Error executing 5m fallback receiver rate queries: %v", err)
		}
	}

	return nil
}

// executePrometheusQueriesWithResultCount executes queries and returns the total result count
func (s *Service) executePrometheusQueriesWithResultCount(queries map[string]func(string, float64)) (int, error) {
	totalResults := 0
	for query, handler := range queries {
		result, err := s.Query(query)
		if err != nil {
			log.Printf("[DEBUG] Prometheus query failed for %s: %v", query, err)
			continue
		}

		resultCount := len(result.Data.Result)
		if resultCount == 0 {
			log.Printf("[DEBUG] Prometheus query returned 0 results: %s", query)
		} else {
			log.Printf("[DEBUG] Prometheus query returned %d results: %s", resultCount, query)
		}
		totalResults += resultCount

		for _, res := range result.Data.Result {
			var componentName string
			if receiver, ok := res.Metric["receiver"]; ok {
				componentName = receiver
			} else if processor, ok := res.Metric["processor"]; ok {
				componentName = processor
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

	return totalResults, nil
}

// fetchProcessorRates queries Prometheus for processor rates
// If jobLabel is provided, rates are filtered by that job/component label
// Note: 1m window may return empty results, so we try 5m as fallback for better reliability
func (s *Service) fetchProcessorRates(rates *types.TelemetryRates, jobLabel string) error {
	// Build job filter for PromQL query - must be inside the selector braces
	// Format: ,job="otel-agent" (comma prefix to add after first label)
	jobFilter := ""
	if jobLabel != "" {
		jobFilter = fmt.Sprintf(`,job="%s"`, jobLabel)
	}

	// Try 1m window first (primary queries)
	// NOTE: Label name is "otel.signal" (with dot) - must use quotes around label name in PromQL
	// Syntax: {"otel.signal"="metrics",job="otel-agent"} - job filter goes BEFORE closing brace
	primaryQueries := map[string]func(string, float64){
		fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_incoming_items_total{"otel.signal"="traces"%s}[1m]))`, jobFilter): func(processor string, value float64) {
			if processor == "" {
				return
			}
			p := rates.Processors[processor]
			p.SpansIncomingRate = value
			p.SpansAcceptedRate = value // Legacy compatibility
			rates.Processors[processor] = p
		},
		fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_incoming_items_total{"otel.signal"="logs"%s}[1m]))`, jobFilter): func(processor string, value float64) {
			if processor == "" {
				return
			}
			p := rates.Processors[processor]
			p.LogsIncomingRate = value
			p.LogsAcceptedRate = value // Legacy compatibility
			rates.Processors[processor] = p
		},
		fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_incoming_items_total{"otel.signal"="metrics"%s}[1m]))`, jobFilter): func(processor string, value float64) {
			if processor == "" {
				return
			}
			p := rates.Processors[processor]
			p.MetricsIncomingRate = value
			p.MetricsAcceptedRate = value // Legacy compatibility
			rates.Processors[processor] = p
		},
		fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_outgoing_items_total{"otel.signal"="traces"%s}[1m]))`, jobFilter): func(processor string, value float64) {
			if processor == "" {
				return
			}
			p := rates.Processors[processor]
			p.SpansOutgoingRate = value
			rates.Processors[processor] = p
		},
		fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_outgoing_items_total{"otel.signal"="logs"%s}[1m]))`, jobFilter): func(processor string, value float64) {
			if processor == "" {
				return
			}
			p := rates.Processors[processor]
			p.LogsOutgoingRate = value
			rates.Processors[processor] = p
		},
		fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_outgoing_items_total{"otel.signal"="metrics"%s}[1m]))`, jobFilter): func(processor string, value float64) {
			if processor == "" {
				return
			}
			p := rates.Processors[processor]
			p.MetricsOutgoingRate = value
			rates.Processors[processor] = p
		},
	}

	// Execute 1m queries first
	result1m, err1m := s.executePrometheusQueriesWithResultCount(primaryQueries)
	if err1m != nil {
		log.Printf("[DEBUG] Error executing 1m processor rate queries: %v", err1m)
	}

	// If 1m window returned no results, try 5m window as fallback
	// Note: 1m window should work fine for rates, but might fail if:
	// - Metrics don't exist yet (new collector)
	// - Scrape interval is longer than 1m
	// - Not enough data points in the window
	if result1m == 0 {
		log.Printf("[DEBUG] 1m window returned no results for processor rates (result count: %d), trying 5m window as fallback", result1m)
		fallbackQueries := map[string]func(string, float64){
			fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_incoming_items_total{"otel.signal"="traces"%s}[5m]))`, jobFilter): func(processor string, value float64) {
				if processor == "" {
					return
				}
				p := rates.Processors[processor]
				// Only set if not already set by 1m query
				if p.SpansIncomingRate == 0 {
					p.SpansIncomingRate = value
					p.SpansAcceptedRate = value // Legacy compatibility
				}
				rates.Processors[processor] = p
			},
			fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_incoming_items_total{"otel.signal"="logs"%s}[5m]))`, jobFilter): func(processor string, value float64) {
				if processor == "" {
					return
				}
				p := rates.Processors[processor]
				// Only set if not already set by 1m query
				if p.LogsIncomingRate == 0 {
					p.LogsIncomingRate = value
					p.LogsAcceptedRate = value // Legacy compatibility
				}
				rates.Processors[processor] = p
			},
			fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_incoming_items_total{"otel.signal"="metrics"%s}[5m]))`, jobFilter): func(processor string, value float64) {
				if processor == "" {
					return
				}
				p := rates.Processors[processor]
				// Only set if not already set by 1m query
				if p.MetricsIncomingRate == 0 {
					p.MetricsIncomingRate = value
					p.MetricsAcceptedRate = value // Legacy compatibility
				}
				rates.Processors[processor] = p
			},
			fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_outgoing_items_total{"otel.signal"="traces"%s}[5m]))`, jobFilter): func(processor string, value float64) {
				if processor == "" {
					return
				}
				p := rates.Processors[processor]
				// Only set if not already set by 1m query
				if p.SpansOutgoingRate == 0 {
					p.SpansOutgoingRate = value
				}
				rates.Processors[processor] = p
			},
			fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_outgoing_items_total{"otel.signal"="logs"%s}[5m]))`, jobFilter): func(processor string, value float64) {
				if processor == "" {
					return
				}
				p := rates.Processors[processor]
				// Only set if not already set by 1m query
				if p.LogsOutgoingRate == 0 {
					p.LogsOutgoingRate = value
				}
				rates.Processors[processor] = p
			},
			fmt.Sprintf(`sum by(processor) (rate(otelcol_processor_outgoing_items_total{"otel.signal"="metrics"%s}[5m]))`, jobFilter): func(processor string, value float64) {
				if processor == "" {
					return
				}
				p := rates.Processors[processor]
				// Only set if not already set by 1m query
				if p.MetricsOutgoingRate == 0 {
					p.MetricsOutgoingRate = value
				}
				rates.Processors[processor] = p
			},
		}
		if err := s.executePrometheusQueries(fallbackQueries); err != nil {
			log.Printf("[DEBUG] Error executing 5m fallback processor rate queries: %v", err)
		}
	}

	return nil
}

// executePrometheusQueriesWithLabelsAndResultCount executes queries with labels and returns the total result count
func (s *Service) executePrometheusQueriesWithLabelsAndResultCount(queries map[string]func(map[string]string, float64)) (int, error) {
	totalResults := 0
	for query, handler := range queries {
		result, err := s.Query(query)
		if err != nil {
			log.Printf("[DEBUG] Prometheus query failed for %s: %v", query, err)
			continue
		}

		resultCount := len(result.Data.Result)
		totalResults += resultCount

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

	return totalResults, nil
}

// fetchExporterRates queries Prometheus for exporter rates
// If jobLabel is provided, rates are filtered by that job/component label
// Note: 1m window may return empty results, so we try 5m as fallback for better reliability
func (s *Service) fetchExporterRates(rates *types.TelemetryRates, jobLabel string) error {
	// Build job filter for PromQL query
	jobFilter := ""
	if jobLabel != "" {
		jobFilter = fmt.Sprintf(`{job="%s"}`, jobLabel)
	}

	// Try 1m window first (primary queries with correct metric names)
	primaryQueries := map[string]func(string, float64){
		fmt.Sprintf(`sum by(exporter) (rate(otelcol_exporter_sent_log_records_total%s[1m]))`, jobFilter): func(exporter string, value float64) {
			e := rates.Exporters[exporter]
			e.LogsSentRate = value
			rates.Exporters[exporter] = e
		},
		fmt.Sprintf(`sum by(exporter) (rate(otelcol_exporter_sent_metric_points_total%s[1m]))`, jobFilter): func(exporter string, value float64) {
			e := rates.Exporters[exporter]
			e.MetricsSentRate = value
			rates.Exporters[exporter] = e
		},
		fmt.Sprintf(`sum by(exporter) (rate(otelcol_exporter_sent_spans_total%s[1m]))`, jobFilter): func(exporter string, value float64) {
			e := rates.Exporters[exporter]
			e.SpansSentRate = value
			rates.Exporters[exporter] = e
		},
	}

	// Execute 1m queries first
	result1m, err1m := s.executePrometheusQueriesWithResultCount(primaryQueries)
	if err1m != nil {
		log.Printf("[DEBUG] Error executing 1m exporter rate queries: %v", err1m)
	}

	// If 1m window returned no results, try 5m window as fallback
	// Note: 1m window should work fine for rates, but might fail if:
	// - Metrics don't exist yet (new collector)
	// - Scrape interval is longer than 1m
	// - Not enough data points in the window
	if result1m == 0 {
		log.Printf("[DEBUG] 1m window returned no results for exporter rates (result count: %d), trying 5m window as fallback", result1m)
		fallbackQueries := map[string]func(string, float64){
			fmt.Sprintf(`sum by(exporter) (rate(otelcol_exporter_sent_log_records_total%s[5m]))`, jobFilter): func(exporter string, value float64) {
				e := rates.Exporters[exporter]
				// Only set if not already set by 1m query
				if e.LogsSentRate == 0 {
					e.LogsSentRate = value
				}
				rates.Exporters[exporter] = e
			},
			fmt.Sprintf(`sum by(exporter) (rate(otelcol_exporter_sent_metric_points_total%s[5m]))`, jobFilter): func(exporter string, value float64) {
				e := rates.Exporters[exporter]
				// Only set if not already set by 1m query
				if e.MetricsSentRate == 0 {
					e.MetricsSentRate = value
				}
				rates.Exporters[exporter] = e
			},
			fmt.Sprintf(`sum by(exporter) (rate(otelcol_exporter_sent_spans_total%s[5m]))`, jobFilter): func(exporter string, value float64) {
				e := rates.Exporters[exporter]
				// Only set if not already set by 1m query
				if e.SpansSentRate == 0 {
					e.SpansSentRate = value
				}
				rates.Exporters[exporter] = e
			},
		}
		if err := s.executePrometheusQueries(fallbackQueries); err != nil {
			log.Printf("[DEBUG] Error executing 5m fallback exporter rate queries: %v", err)
		}
	}

	return nil
}

// executePrometheusQueries executes multiple Prometheus queries
func (s *Service) executePrometheusQueries(queries map[string]func(string, float64)) error {
	for query, handler := range queries {
		result, err := s.Query(query)
		if err != nil {
			log.Printf("[DEBUG] Prometheus query failed for %s: %v", query, err)
			continue
		}

		for _, res := range result.Data.Result {
			var componentName string
			if receiver, ok := res.Metric["receiver"]; ok {
				componentName = receiver
			} else if processor, ok := res.Metric["processor"]; ok {
				componentName = processor
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

// executePrometheusQueriesWithLabels executes queries that need full label maps
func (s *Service) executePrometheusQueriesWithLabels(queries map[string]func(map[string]string, float64)) error {
	for query, handler := range queries {
		result, err := s.Query(query)
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
