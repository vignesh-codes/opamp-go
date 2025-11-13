package types

// AgentState represents the persisted state of an agent from JSON
type AgentState struct {
	Labels           map[string]string `json:"labels"`
	CurrentConfig    string            `json:"current_config"`
	RemoteConfigHash string            `json:"remote_config_hash"`
}

// AgentsState represents the persisted state of all agents
type AgentsState struct {
	Agents map[string]*AgentState `json:"agents"`
}

// JSONAgent represents an agent from JSON (simplified version of data.Agent)
type JSONAgent struct {
	ID            string
	Labels        map[string]string
	CurrentConfig string
}

// PipelineConfig represents a parsed OpenTelemetry pipeline configuration
type PipelineConfig struct {
	Receivers  []string          `json:"receivers"`
	Processors []string          `json:"processors"`
	Exporters  []string          `json:"exporters"`
	Endpoints  map[string]string `json:"endpoints"` // exporter name -> endpoint URL
}

// AgentPipelineInfo contains pipeline information for an agent
type AgentPipelineInfo struct {
	AgentID    string                     `json:"agent_id"`
	AgentName  string                     `json:"agent_name"`
	Pipelines  map[string]*PipelineConfig `json:"pipelines"` // pipeline type (logs/metrics/traces) -> config
	Receivers  map[string]interface{}     `json:"receivers"`
	Processors map[string]interface{}     `json:"processors"`
	Exporters  map[string]interface{}     `json:"exporters"`
	ConfigYAML string                     `json:"config_yaml,omitempty"`
}

// TopologyNode represents a node in the topology graph
type TopologyNode struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"` // "agent", "receiver", "processor", "exporter", "endpoint"
	Name          string            `json:"name"`
	AgentID       string            `json:"agent_id,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty"`
	PipelineType  string            `json:"pipeline_type,omitempty"`  // logs, metrics, traces
	ComponentType string            `json:"component_type,omitempty"` // receiver, processor, exporter
	DataTransfer  *NodeDataTransfer `json:"data_transfer,omitempty"`  // Data transfer information
}

// NodeDataTransfer represents data transfer metrics for a node
type NodeDataTransfer struct {
	SpansReceived   float64 `json:"spans_received,omitempty"`
	SpansSent       float64 `json:"spans_sent,omitempty"`
	LogsReceived    float64 `json:"logs_received,omitempty"`
	LogsSent        float64 `json:"logs_sent,omitempty"`
	MetricsReceived float64 `json:"metrics_received,omitempty"`
	MetricsSent     float64 `json:"metrics_sent,omitempty"`
	// Rate fields (items per second)
	SpansReceivedRate   float64 `json:"spans_received_rate,omitempty"`
	SpansSentRate       float64 `json:"spans_sent_rate,omitempty"`
	LogsReceivedRate    float64 `json:"logs_received_rate,omitempty"`
	LogsSentRate        float64 `json:"logs_sent_rate,omitempty"`
	MetricsReceivedRate float64 `json:"metrics_received_rate,omitempty"`
	MetricsSentRate     float64 `json:"metrics_sent_rate,omitempty"`
}

// TopologyEdge represents a connection in the topology graph
type TopologyEdge struct {
	From         string            `json:"from"`
	To           string            `json:"to"`
	PipelineType string            `json:"pipeline_type"` // logs, metrics, traces
	Label        string            `json:"label,omitempty"`
	DataTransfer *EdgeDataTransfer `json:"data_transfer,omitempty"` // Data transfer information
}

// EdgeDataTransfer represents data transfer metrics for an edge
type EdgeDataTransfer struct {
	SpansTransferred   float64 `json:"spans_transferred,omitempty"`
	LogsTransferred    float64 `json:"logs_transferred,omitempty"`
	MetricsTransferred float64 `json:"metrics_transferred,omitempty"`
	// Rate fields (items per second) - output rates (what flows through the edge)
	SpansTransferredRate   float64 `json:"spans_transferred_rate,omitempty"`
	LogsTransferredRate    float64 `json:"logs_transferred_rate,omitempty"`
	MetricsTransferredRate float64 `json:"metrics_transferred_rate,omitempty"`
	// Input rate fields (items per second) - input to the component (before filtering/dropping)
	SpansInputRate   float64 `json:"spans_input_rate,omitempty"`
	LogsInputRate    float64 `json:"logs_input_rate,omitempty"`
	MetricsInputRate float64 `json:"metrics_input_rate,omitempty"`
}

// TopologyResponse represents the complete topology visualization
type TopologyResponse struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// TelemetryMetrics represents scraped metrics from collectors
type TelemetryMetrics struct {
	Receivers  map[string]map[string]interface{} `json:"receivers"`
	Exporters  map[string]map[string]interface{} `json:"exporters"`
	Processors map[string]map[string]interface{} `json:"processors"`
	ByPipeline map[string]map[string]interface{} `json:"by_pipeline"`
	Totals     map[string]interface{}            `json:"totals"`
}

// TelemetryRates stores rate metrics for receivers, processors, and exporters
type TelemetryRates struct {
	Receivers  map[string]ReceiverRates  `json:"receivers"`
	Processors map[string]ProcessorRates `json:"processors"`
	Exporters  map[string]ExporterRates  `json:"exporters"`
}

// ReceiverRates stores rate metrics for a receiver
type ReceiverRates struct {
	SpansReceivedRate   float64 `json:"spans_received_rate,omitempty"`
	LogsReceivedRate    float64 `json:"logs_received_rate,omitempty"`
	MetricsReceivedRate float64 `json:"metrics_received_rate,omitempty"`
	SpansRefusedRate    float64 `json:"spans_refused_rate,omitempty"`
	LogsRefusedRate     float64 `json:"logs_refused_rate,omitempty"`
	MetricsRefusedRate  float64 `json:"metrics_refused_rate,omitempty"`
}

// ProcessorRates stores rate metrics for a processor
type ProcessorRates struct {
	// Incoming rates (input to processor)
	SpansIncomingRate   float64 `json:"spans_incoming_rate,omitempty"`
	LogsIncomingRate    float64 `json:"logs_incoming_rate,omitempty"`
	MetricsIncomingRate float64 `json:"metrics_incoming_rate,omitempty"`
	// Outgoing rates (output from processor)
	SpansOutgoingRate   float64 `json:"spans_outgoing_rate,omitempty"`
	LogsOutgoingRate    float64 `json:"logs_outgoing_rate,omitempty"`
	MetricsOutgoingRate float64 `json:"metrics_outgoing_rate,omitempty"`
	// Legacy fields for backward compatibility
	SpansAcceptedRate   float64 `json:"spans_accepted_rate,omitempty"`
	SpansRefusedRate    float64 `json:"spans_refused_rate,omitempty"`
	LogsAcceptedRate    float64 `json:"logs_accepted_rate,omitempty"`
	LogsRefusedRate     float64 `json:"logs_refused_rate,omitempty"`
	MetricsAcceptedRate float64 `json:"metrics_accepted_rate,omitempty"`
	MetricsRefusedRate  float64 `json:"metrics_refused_rate,omitempty"`
}

// ExporterRates stores rate metrics for an exporter
type ExporterRates struct {
	SpansSentRate     float64 `json:"spans_sent_rate,omitempty"`
	SpansFailedRate   float64 `json:"spans_failed_rate,omitempty"`
	LogsSentRate      float64 `json:"logs_sent_rate,omitempty"`
	LogsFailedRate    float64 `json:"logs_failed_rate,omitempty"`
	MetricsSentRate   float64 `json:"metrics_sent_rate,omitempty"`
	MetricsFailedRate float64 `json:"metrics_failed_rate,omitempty"`
}

// PrometheusQueryResult represents a single metric result from Prometheus
type PrometheusQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"` // [timestamp, value]
}

// PrometheusQueryResponse represents the response from Prometheus API
type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string                  `json:"resultType"`
		Result     []PrometheusQueryResult `json:"result"`
	} `json:"data"`
}
