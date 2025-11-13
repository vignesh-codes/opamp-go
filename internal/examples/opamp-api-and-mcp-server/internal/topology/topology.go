package topology

import (
	"fmt"
	"log"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/config"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/metrics"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/prometheus"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
)

// Service handles topology graph building
type Service struct {
	metricsService    *metrics.Service
	prometheusService *prometheus.Service
	configService     *config.Service
}

func hasNonZeroData(dt *types.NodeDataTransfer) bool {
	if dt == nil {
		return false
	}
	return dt.SpansReceived > 0 || dt.SpansSent > 0 ||
		dt.LogsReceived > 0 || dt.LogsSent > 0 ||
		dt.MetricsReceived > 0 || dt.MetricsSent > 0 ||
		dt.SpansReceivedRate > 0 || dt.SpansSentRate > 0 ||
		dt.LogsReceivedRate > 0 || dt.LogsSentRate > 0 ||
		dt.MetricsReceivedRate > 0 || dt.MetricsSentRate > 0
}

// New creates a new topology service
func New(metricsService *metrics.Service, prometheusService *prometheus.Service, configService *config.Service) *Service {
	return &Service{
		metricsService:    metricsService,
		prometheusService: prometheusService,
		configService:     configService,
	}
}

// BuildTopology builds the topology graph for all agents
func (s *Service) BuildTopology(agents map[string]*types.JSONAgent) (*types.TopologyResponse, error) {
	topology := &types.TopologyResponse{
		Nodes: []types.TopologyNode{},
		Edges: []types.TopologyEdge{},
	}

	nodeMap := make(map[string]bool)
	componentNodeMap := make(map[string]string)
	endpointToAgents := make(map[string][]string)
	agentEndpoints := make(map[string]map[string][]string)

	// Build component data maps for all agents
	componentDataMap := make(map[string]map[string]map[string]*types.NodeDataTransfer)
	pipelineDataMap := make(map[string]map[string]*types.EdgeDataTransfer)
	exporterEdgeDataMap := make(map[string]map[string]map[string]*types.EdgeDataTransfer)
	agentDataMap := make(map[string]*types.NodeDataTransfer)
	agentMetricsMap := make(map[string]*types.TelemetryMetrics)
	agentRatesMap := make(map[string]*types.TelemetryRates)

	for id, agent := range agents {
		agentID := id
		agentDataMap[agentID] = &types.NodeDataTransfer{}

		pipelineInfo := s.configService.ParsePipelineConfig(agent)
		metrics := s.metricsService.ScrapeCollectorMetrics(agent, pipelineInfo)
		agentMetricsMap[agentID] = metrics

		componentDataMap[agentID] = make(map[string]map[string]*types.NodeDataTransfer)
		pipelineDataMap[agentID] = make(map[string]*types.EdgeDataTransfer)

		// Fetch real-time rates from Prometheus for this specific agent
		// Filter rates by job/component label to avoid aggregating across all agents
		jobLabel := s.configService.GetAgentJobLabel(agent)
		rates, err := s.prometheusService.FetchTelemetryRates(jobLabel)
		if err != nil {
			log.Printf("[DEBUG] Failed to fetch rates from Prometheus for agent %s (job=%s): %v", agentID, jobLabel, err)
			rates = &types.TelemetryRates{
				Receivers:  make(map[string]types.ReceiverRates),
				Processors: make(map[string]types.ProcessorRates),
				Exporters:  make(map[string]types.ExporterRates),
			}
		}
		agentRatesMap[agentID] = rates

		// Build component data
		s.buildComponentData(agentID, pipelineInfo, metrics, rates, componentDataMap, pipelineDataMap, exporterEdgeDataMap)
	}

	// Build topology nodes and edges
	for id, agent := range agents {
		agentID := id
		agentName := s.configService.GetAgentName(agent)

		// Add agent node
		agentNodeID := fmt.Sprintf("agent-%s", agentID)
		if !nodeMap[agentNodeID] {
			agentNode := types.TopologyNode{
				ID:      agentNodeID,
				Type:    "agent",
				Name:    agentName,
				AgentID: agentID,
			}
			// Get agent data from componentDataMap
			if agentCompData, ok := componentDataMap[agentID]["agent"]; ok {
				if agentData, ok := agentCompData["all"]; ok {
					agentNode.DataTransfer = agentData
				}
			} else if agentData, ok := agentDataMap[agentID]; ok {
				agentNode.DataTransfer = agentData
			}
			topology.Nodes = append(topology.Nodes, agentNode)
			nodeMap[agentNodeID] = true
		}

		// Parse pipelines and build component nodes
		pipelineInfo := s.configService.ParsePipelineConfig(agent)
		agentEndpoints[agentNodeID] = make(map[string][]string)
		metrics := agentMetricsMap[agentID]
		rates := agentRatesMap[agentID]
		if rates == nil {
			rates = &types.TelemetryRates{
				Receivers:  make(map[string]types.ReceiverRates),
				Processors: make(map[string]types.ProcessorRates),
				Exporters:  make(map[string]types.ExporterRates),
			}
		}

		// Process each pipeline
		for pipelineType, pipeline := range pipelineInfo.Pipelines {
			// Skip traces pipeline if there's no trace data/activity
			if pipelineType == "traces" {
				hasTraceData := false
				// Check if any receiver has trace rates
				for _, receiverName := range pipeline.Receivers {
					if rates != nil {
						if receiverRates, ok := rates.Receivers[receiverName]; ok {
							if receiverRates.SpansReceivedRate > 0 {
								hasTraceData = true
								break
							}
						}
					}
					// Check if metrics have trace data
					if metrics != nil {
						if receiverData, ok := metrics.Receivers[receiverName]; ok {
							if spansReceived, ok := receiverData["spans_received"].(float64); ok && spansReceived > 0 {
								hasTraceData = true
								break
							}
						}
					}
				}
				// Check if any processor has trace rates
				if !hasTraceData && rates != nil {
					for _, processorName := range pipeline.Processors {
						if processorRates, ok := rates.Processors[processorName]; ok {
							if processorRates.SpansIncomingRate > 0 || processorRates.SpansOutgoingRate > 0 {
								hasTraceData = true
								break
							}
						}
					}
				}
				// Check if any exporter has trace rates
				if !hasTraceData && rates != nil {
					for _, exporterName := range pipeline.Exporters {
						if exporterName != "debug" {
							if exporterRates, ok := rates.Exporters[exporterName]; ok {
								if exporterRates.SpansSentRate > 0 {
									hasTraceData = true
									break
								}
							}
						}
					}
				}
				// Skip traces pipeline if no data
				if !hasTraceData {
					continue
				}
			}

			// Build receiver nodes (always show receivers, even if exporters are debug-only)
			receiverNodeIDs := s.buildReceiverNodes(agentNodeID, agentID, pipelineType, pipeline.Receivers, componentDataMap, topology, nodeMap, componentNodeMap, pipelineDataMap, pipeline, metrics, rates)

			// Build processor nodes (always show processors, even if exporters are debug-only)
			processorNodeIDs := s.buildProcessorNodes(agentNodeID, agentID, pipelineType, pipeline.Processors, pipeline.Receivers, receiverNodeIDs, componentDataMap, topology, nodeMap, componentNodeMap, pipelineDataMap, pipeline, metrics, rates)

			// Build exporter nodes
			// Note: buildExporterNodes will skip debug exporters, but we still show receivers and processors
			// Pass the last processor name so we can use its output rate for edges
			lastProcessorName := ""
			if len(pipeline.Processors) > 0 {
				for i := len(pipeline.Processors) - 1; i >= 0; i-- {
					candidate := pipeline.Processors[i]
					if compData, ok := componentDataMap[agentID][candidate]; ok {
						if dt, ok := compData[pipelineType]; ok {
							if hasNonZeroData(dt) {
								lastProcessorName = candidate
								break
							}
						}
					}
					// Fall back to the first processor with rate data even if componentDataMap had nothing
					if lastProcessorName == "" && rates != nil {
						if processorRates, ok := rates.Processors[candidate]; ok {
							switch pipelineType {
							case "traces":
								if processorRates.SpansOutgoingRate > 0 {
									lastProcessorName = candidate
								}
							case "logs":
								if processorRates.LogsOutgoingRate > 0 {
									lastProcessorName = candidate
								}
							case "metrics":
								if processorRates.MetricsOutgoingRate > 0 {
									lastProcessorName = candidate
								}
							}
						}
					}
					if lastProcessorName != "" {
						break
					}
				}
			}
			lastComponents := receiverNodeIDs
			if len(processorNodeIDs) > 0 {
				lastComponents = []string{processorNodeIDs[len(processorNodeIDs)-1]}
			}
			s.buildExporterNodes(agentNodeID, agentID, pipelineType, pipeline.Exporters, lastComponents, lastProcessorName, componentDataMap, exporterEdgeDataMap, topology, nodeMap, componentNodeMap, pipeline, metrics, rates, endpointToAgents, agentEndpoints)
		}

		// Recalculate agent-level summary from actual exporter nodes (after buildExporterNodes updates them)
		// This ensures the agent-level LogsSentRate matches what's shown in exporter nodes
		if agentCompData, ok := componentDataMap[agentID]["agent"]; ok {
			if agentData, ok := agentCompData["all"]; ok {
				// Reset sent rates and recalculate from exporter nodes
				agentData.SpansSentRate = 0
				agentData.LogsSentRate = 0
				agentData.MetricsSentRate = 0

				// Sum up all exporter nodes' sent rates
				for pipelineType, pipeline := range pipelineInfo.Pipelines {
					for _, exporterName := range pipeline.Exporters {
						if exporterName == "debug" {
							continue
						}
						if exporterData, ok := componentDataMap[agentID][exporterName]; ok {
							if exporterNodeData, ok := exporterData[pipelineType]; ok {
								switch pipelineType {
								case "traces":
									agentData.SpansSentRate += exporterNodeData.SpansSentRate
								case "logs":
									agentData.LogsSentRate += exporterNodeData.LogsSentRate
								case "metrics":
									agentData.MetricsSentRate += exporterNodeData.MetricsSentRate
								}
							}
						}
					}
				}
			}
		}
	}

	return topology, nil
}

// buildComponentData builds component data maps from metrics and rates
func (s *Service) buildComponentData(agentID string, pipelineInfo *types.AgentPipelineInfo, metrics *types.TelemetryMetrics, rates *types.TelemetryRates, componentDataMap map[string]map[string]map[string]*types.NodeDataTransfer, pipelineDataMap map[string]map[string]*types.EdgeDataTransfer, exporterEdgeDataMap map[string]map[string]map[string]*types.EdgeDataTransfer) {
	// Build receiver data
	for receiverName, receiverData := range metrics.Receivers {
		if componentDataMap[agentID][receiverName] == nil {
			componentDataMap[agentID][receiverName] = make(map[string]*types.NodeDataTransfer)
		}

		// Create base node data for this receiver
		baseNodeData := &types.NodeDataTransfer{}

		// Set cumulative values
		if v, ok := receiverData["spans_received"].(float64); ok {
			baseNodeData.SpansReceived = v
		}
		if v, ok := receiverData["logs_received"].(float64); ok {
			baseNodeData.LogsReceived = v
		}
		if v, ok := receiverData["metrics_received"].(float64); ok {
			baseNodeData.MetricsReceived = v
		}

		// Set rates if available
		if rates != nil {
			if receiverRates, ok := rates.Receivers[receiverName]; ok {
				baseNodeData.SpansReceivedRate = receiverRates.SpansReceivedRate
				baseNodeData.LogsReceivedRate = receiverRates.LogsReceivedRate
				baseNodeData.MetricsReceivedRate = receiverRates.MetricsReceivedRate
			}
		}

		// Assign to all pipelines that use this receiver
		// IMPORTANT: Filter rates by pipeline type - only show relevant rates for each pipeline
		for pipelineType, pipeline := range pipelineInfo.Pipelines {
			for _, pipelineReceiver := range pipeline.Receivers {
				if pipelineReceiver == receiverName {
					// Create pipeline-specific node data with filtered cumulative values and rates
					nodeData := &types.NodeDataTransfer{}
					// Only set cumulative values and rates for the specific pipeline type
					switch pipelineType {
					case "traces":
						nodeData.SpansReceived = baseNodeData.SpansReceived
						// Receivers pass data through, so sent equals received
						nodeData.SpansSent = baseNodeData.SpansReceived
						nodeData.SpansReceivedRate = baseNodeData.SpansReceivedRate
						// Receivers pass data through, so sent rate equals received rate
						nodeData.SpansSentRate = baseNodeData.SpansReceivedRate
					case "logs":
						nodeData.LogsReceived = baseNodeData.LogsReceived
						// Receivers pass data through, so sent equals received
						nodeData.LogsSent = baseNodeData.LogsReceived
						nodeData.LogsReceivedRate = baseNodeData.LogsReceivedRate
						// Receivers pass data through, so sent rate equals received rate
						nodeData.LogsSentRate = baseNodeData.LogsReceivedRate
					case "metrics":
						nodeData.MetricsReceived = baseNodeData.MetricsReceived
						// Receivers pass data through, so sent equals received
						nodeData.MetricsSent = baseNodeData.MetricsReceived
						nodeData.MetricsReceivedRate = baseNodeData.MetricsReceivedRate
						// Receivers pass data through, so sent rate equals received rate
						nodeData.MetricsSentRate = baseNodeData.MetricsReceivedRate
					}
					componentDataMap[agentID][receiverName][pipelineType] = nodeData
					break
				}
			}
		}
	}

	// Build processor data
	for processorName, processorData := range metrics.Processors {
		if componentDataMap[agentID][processorName] == nil {
			componentDataMap[agentID][processorName] = make(map[string]*types.NodeDataTransfer)
		}

		baseNodeData := &types.NodeDataTransfer{}

		// Set cumulative values
		if v, ok := processorData["spans_accepted"].(float64); ok {
			baseNodeData.SpansReceived = v
		}
		if v, ok := processorData["logs_accepted"].(float64); ok {
			baseNodeData.LogsReceived = v
		}
		if v, ok := processorData["metrics_accepted"].(float64); ok {
			baseNodeData.MetricsReceived = v
		}
		if v, ok := processorData["spans_sent"].(float64); ok {
			baseNodeData.SpansSent = v
		}
		if v, ok := processorData["logs_sent"].(float64); ok {
			baseNodeData.LogsSent = v
		}
		if v, ok := processorData["metrics_sent"].(float64); ok {
			baseNodeData.MetricsSent = v
		}

		// Set rates if available
		if rates != nil {
			if processorRates, ok := rates.Processors[processorName]; ok {
				baseNodeData.SpansReceivedRate = processorRates.SpansIncomingRate
				baseNodeData.SpansSentRate = processorRates.SpansOutgoingRate
				baseNodeData.LogsReceivedRate = processorRates.LogsIncomingRate
				baseNodeData.LogsSentRate = processorRates.LogsOutgoingRate
				baseNodeData.MetricsReceivedRate = processorRates.MetricsIncomingRate
				baseNodeData.MetricsSentRate = processorRates.MetricsOutgoingRate
			}
		}

		// Assign to all pipelines that use this processor
		// IMPORTANT: Filter both cumulative values and rates by pipeline type
		for pipelineType, pipeline := range pipelineInfo.Pipelines {
			for _, pipelineProcessor := range pipeline.Processors {
				if pipelineProcessor == processorName {
					// Create pipeline-specific node data with filtered cumulative values and rates
					nodeData := &types.NodeDataTransfer{}
					// Only set cumulative values and rates for the specific pipeline type
					switch pipelineType {
					case "traces":
						nodeData.SpansReceived = baseNodeData.SpansReceived
						nodeData.SpansSent = baseNodeData.SpansSent
						nodeData.SpansReceivedRate = baseNodeData.SpansReceivedRate
						nodeData.SpansSentRate = baseNodeData.SpansSentRate
					case "logs":
						nodeData.LogsReceived = baseNodeData.LogsReceived
						nodeData.LogsSent = baseNodeData.LogsSent
						nodeData.LogsReceivedRate = baseNodeData.LogsReceivedRate
						nodeData.LogsSentRate = baseNodeData.LogsSentRate
					case "metrics":
						nodeData.MetricsReceived = baseNodeData.MetricsReceived
						nodeData.MetricsSent = baseNodeData.MetricsSent
						nodeData.MetricsReceivedRate = baseNodeData.MetricsReceivedRate
						nodeData.MetricsSentRate = baseNodeData.MetricsSentRate
					}
					componentDataMap[agentID][processorName][pipelineType] = nodeData
					break
				}
			}
		}
	}

	// Build exporter data
	for exporterName, exporterData := range metrics.Exporters {
		if componentDataMap[agentID][exporterName] == nil {
			componentDataMap[agentID][exporterName] = make(map[string]*types.NodeDataTransfer)
		}

		baseNodeData := &types.NodeDataTransfer{}

		// Set cumulative values
		if v, ok := exporterData["spans_sent"].(float64); ok {
			baseNodeData.SpansSent = v
		}
		if v, ok := exporterData["logs_sent"].(float64); ok {
			baseNodeData.LogsSent = v
		}
		if v, ok := exporterData["metrics_sent"].(float64); ok {
			baseNodeData.MetricsSent = v
		}

		// Set rates if available
		if rates != nil {
			if exporterRates, ok := rates.Exporters[exporterName]; ok {
				baseNodeData.SpansSentRate = exporterRates.SpansSentRate
				baseNodeData.LogsSentRate = exporterRates.LogsSentRate
				baseNodeData.MetricsSentRate = exporterRates.MetricsSentRate
			}
		}

		// Assign to all pipelines that use this exporter
		// IMPORTANT: Filter both cumulative values and rates by pipeline type
		for pipelineType, pipeline := range pipelineInfo.Pipelines {
			for _, pipelineExporter := range pipeline.Exporters {
				if pipelineExporter == exporterName {
					// Create pipeline-specific node data with filtered cumulative values and rates
					nodeData := &types.NodeDataTransfer{}

					// Set exporter's received values from the last processor's outgoing values
					// If no processors, use receiver's sent values
					// This ensures the flow of data matches correctly (filter output -> exporter input)
					// NOTE: Some processors like "batch" don't report outgoing_items_total metrics,
					// so we need to find the last processor that actually has outgoing rate data
					lastProcessorName := ""
					var lastProcessorData *types.NodeDataTransfer
					if len(pipeline.Processors) > 0 {
						// First, try to find the last processor with data in componentDataMap
						for i := len(pipeline.Processors) - 1; i >= 0; i-- {
							candidate := pipeline.Processors[i]
							if processorData, ok := componentDataMap[agentID][candidate]; ok {
								if processorNodeData, ok := processorData[pipelineType]; ok {
									// Check if this processor has outgoing rate data
									hasOutgoingRate := false
									switch pipelineType {
									case "traces":
										hasOutgoingRate = processorNodeData.SpansSentRate > 0
									case "logs":
										hasOutgoingRate = processorNodeData.LogsSentRate > 0
									case "metrics":
										hasOutgoingRate = processorNodeData.MetricsSentRate > 0
									}
									if hasOutgoingRate {
										lastProcessorName = candidate
										lastProcessorData = processorNodeData
										break
									}
								}
							}
						}
					}

					if lastProcessorData != nil {
						// Use the last processor with outgoing rate data
						switch pipelineType {
						case "traces":
							nodeData.SpansReceived = lastProcessorData.SpansSent
							nodeData.SpansReceivedRate = lastProcessorData.SpansSentRate
						case "logs":
							nodeData.LogsReceived = lastProcessorData.LogsSent
							nodeData.LogsReceivedRate = lastProcessorData.LogsSentRate
						case "metrics":
							nodeData.MetricsReceived = lastProcessorData.MetricsSent
							nodeData.MetricsReceivedRate = lastProcessorData.MetricsSentRate
						}
					} else if len(pipeline.Processors) > 0 && rates != nil {
						// Fallback: use rates from the last processor that reports outgoing data
						// This handles cases where processors like "batch" don't have outgoing rate metrics
						// IMPORTANT: We iterate from the end to find the LAST processor with outgoing data
						// This ensures we get filter/logs outgoing rate, not batch (which doesn't report it)
						for i := len(pipeline.Processors) - 1; i >= 0; i-- {
							candidate := pipeline.Processors[i]
							if processorRates, ok := rates.Processors[candidate]; ok {
								switch pipelineType {
								case "traces":
									if processorRates.SpansOutgoingRate > 0 {
										nodeData.SpansReceivedRate = processorRates.SpansOutgoingRate
										lastProcessorName = candidate
										break
									}
								case "logs":
									if processorRates.LogsOutgoingRate > 0 {
										nodeData.LogsReceivedRate = processorRates.LogsOutgoingRate
										lastProcessorName = candidate
										break
									}
								case "metrics":
									if processorRates.MetricsOutgoingRate > 0 {
										nodeData.MetricsReceivedRate = processorRates.MetricsOutgoingRate
										lastProcessorName = candidate
										break
									}
								}
							}
							if lastProcessorName != "" {
								break
							}
						}
					}

					// If still no processor data found, try to use the last processor's incoming rate
					// This handles cases where a processor filters everything (outgoing = 0 but incoming > 0)
					// But we should still show the drop at the processor, not at the exporter
					if lastProcessorName == "" && len(pipeline.Processors) > 0 && rates != nil {
						// Use the last processor in the list (before batch which doesn't report outgoing)
						lastProcInList := pipeline.Processors[len(pipeline.Processors)-1]
						// If it's batch, use the one before it
						if lastProcInList == "batch" && len(pipeline.Processors) > 1 {
							lastProcInList = pipeline.Processors[len(pipeline.Processors)-2]
						}
						if processorRates, ok := rates.Processors[lastProcInList]; ok {
							switch pipelineType {
							case "logs":
								// Use outgoing rate if available, otherwise incoming rate (to show the drop)
								if processorRates.LogsOutgoingRate > 0 {
									nodeData.LogsReceivedRate = processorRates.LogsOutgoingRate
									lastProcessorName = lastProcInList
								} else if processorRates.LogsIncomingRate > 0 {
									// Even if outgoing is 0, use incoming to show where the drop happens
									nodeData.LogsReceivedRate = processorRates.LogsIncomingRate
									lastProcessorName = lastProcInList
								}
							case "metrics":
								if processorRates.MetricsOutgoingRate > 0 {
									nodeData.MetricsReceivedRate = processorRates.MetricsOutgoingRate
									lastProcessorName = lastProcInList
								} else if processorRates.MetricsIncomingRate > 0 {
									nodeData.MetricsReceivedRate = processorRates.MetricsIncomingRate
									lastProcessorName = lastProcInList
								}
							case "traces":
								if processorRates.SpansOutgoingRate > 0 {
									nodeData.SpansReceivedRate = processorRates.SpansOutgoingRate
									lastProcessorName = lastProcInList
								} else if processorRates.SpansIncomingRate > 0 {
									nodeData.SpansReceivedRate = processorRates.SpansIncomingRate
									lastProcessorName = lastProcInList
								}
							}
						}
					}

					if lastProcessorName == "" && len(pipeline.Receivers) > 0 {
						// No processors with data, use first receiver's sent values
						firstReceiverName := pipeline.Receivers[0]
						if receiverData, ok := componentDataMap[agentID][firstReceiverName]; ok {
							if receiverNodeData, ok := receiverData[pipelineType]; ok {
								switch pipelineType {
								case "traces":
									nodeData.SpansReceived = receiverNodeData.SpansSent
									nodeData.SpansReceivedRate = receiverNodeData.SpansSentRate
								case "logs":
									nodeData.LogsReceived = receiverNodeData.LogsSent
									nodeData.LogsReceivedRate = receiverNodeData.LogsSentRate
								case "metrics":
									nodeData.MetricsReceived = receiverNodeData.MetricsSent
									nodeData.MetricsReceivedRate = receiverNodeData.MetricsSentRate
								}
							}
						}
					}

					// Set sent values from exporter metrics/rates
					// IMPORTANT: Exporters don't filter, so sent should match received
					// If received rate is set, use it for sent rate (since exporters just forward)
					// Only use exporter sent rate if received rate is not available
					switch pipelineType {
					case "traces":
						nodeData.SpansSent = baseNodeData.SpansSent
						// If we have received rate, use it for sent rate (exporters don't filter)
						if nodeData.SpansReceivedRate > 0 {
							nodeData.SpansSentRate = nodeData.SpansReceivedRate
						} else {
							nodeData.SpansSentRate = baseNodeData.SpansSentRate
						}
					case "logs":
						nodeData.LogsSent = baseNodeData.LogsSent
						// If we have received rate, use it for sent rate (exporters don't filter)
						if nodeData.LogsReceivedRate > 0 {
							nodeData.LogsSentRate = nodeData.LogsReceivedRate
						} else {
							nodeData.LogsSentRate = baseNodeData.LogsSentRate
						}
					case "metrics":
						nodeData.MetricsSent = baseNodeData.MetricsSent
						// If we have received rate, use it for sent rate (exporters don't filter)
						if nodeData.MetricsReceivedRate > 0 {
							nodeData.MetricsSentRate = nodeData.MetricsReceivedRate
						} else {
							nodeData.MetricsSentRate = baseNodeData.MetricsSentRate
						}
					}
					componentDataMap[agentID][exporterName][pipelineType] = nodeData
					break
				}
			}
		}
	}

	// Build pipeline edge data with rates
	for pipelineType, pipeline := range pipelineInfo.Pipelines {
		edgeData := &types.EdgeDataTransfer{}
		pipelineData := metrics.ByPipeline[pipelineType]

		// Set cumulative values
		if pipelineData != nil {
			if v, ok := pipelineData["spans_received"].(float64); ok {
				edgeData.SpansTransferred = v
			}
			if v, ok := pipelineData["logs_received"].(float64); ok {
				edgeData.LogsTransferred = v
			}
			if v, ok := pipelineData["metrics_received"].(float64); ok {
				edgeData.MetricsTransferred = v
			}

			// Also set sent values if available
			if v, ok := pipelineData["spans_sent"].(float64); ok {
				edgeData.SpansTransferred = v
			}
			if v, ok := pipelineData["logs_sent"].(float64); ok {
				edgeData.LogsTransferred = v
			}
			if v, ok := pipelineData["metrics_sent"].(float64); ok {
				edgeData.MetricsTransferred = v
			}
		}

		// Calculate rates for this pipeline based on component rates
		if rates != nil {
			var spansInputRate, spansOutputRate float64
			var logsInputRate, logsOutputRate float64
			var metricsInputRate, metricsOutputRate float64

			// Sum receiver rates (input to pipeline)
			for _, receiverName := range pipeline.Receivers {
				if receiverRates, ok := rates.Receivers[receiverName]; ok {
					switch pipelineType {
					case "traces":
						spansInputRate += receiverRates.SpansReceivedRate
					case "logs":
						logsInputRate += receiverRates.LogsReceivedRate
					case "metrics":
						metricsInputRate += receiverRates.MetricsReceivedRate
					}
				}
			}

			// Use processor output rates or exporter rates (output from pipeline)
			for i := len(pipeline.Processors) - 1; i >= 0; i-- {
				processorName := pipeline.Processors[i]
				if processorRates, ok := rates.Processors[processorName]; ok {
					switch pipelineType {
					case "traces":
						if processorRates.SpansOutgoingRate > 0 {
							spansOutputRate = processorRates.SpansOutgoingRate
							break
						}
					case "logs":
						if processorRates.LogsOutgoingRate > 0 {
							logsOutputRate = processorRates.LogsOutgoingRate
							break
						}
					case "metrics":
						if processorRates.MetricsOutgoingRate > 0 {
							metricsOutputRate = processorRates.MetricsOutgoingRate
							break
						}
					}
				}
			}

			// If no processor output, use exporter rates
			if spansOutputRate == 0 && logsOutputRate == 0 && metricsOutputRate == 0 {
				for _, exporterName := range pipeline.Exporters {
					if exporterRates, ok := rates.Exporters[exporterName]; ok {
						switch pipelineType {
						case "traces":
							if exporterRates.SpansSentRate > 0 {
								spansOutputRate = exporterRates.SpansSentRate
							}
						case "logs":
							if exporterRates.LogsSentRate > 0 {
								logsOutputRate = exporterRates.LogsSentRate
							}
						case "metrics":
							if exporterRates.MetricsSentRate > 0 {
								metricsOutputRate = exporterRates.MetricsSentRate
							}
						}
					}
				}
			}

			// Set edge rates
			switch pipelineType {
			case "traces":
				edgeData.SpansInputRate = spansInputRate
				edgeData.SpansTransferredRate = spansOutputRate
				if spansOutputRate == 0 {
					edgeData.SpansTransferredRate = spansInputRate
				}
			case "logs":
				edgeData.LogsInputRate = logsInputRate
				edgeData.LogsTransferredRate = logsOutputRate
				if logsOutputRate == 0 {
					edgeData.LogsTransferredRate = logsInputRate
				}
			case "metrics":
				edgeData.MetricsInputRate = metricsInputRate
				edgeData.MetricsTransferredRate = metricsOutputRate
				if metricsOutputRate == 0 {
					edgeData.MetricsTransferredRate = metricsInputRate
				}
			}
		}

		pipelineDataMap[agentID][pipelineType] = edgeData
	}

	// Build agent-level data transfer
	agentData := &types.NodeDataTransfer{}
	if metrics.Totals != nil {
		if v, ok := metrics.Totals["spans_received"].(float64); ok {
			agentData.SpansReceived = v
		}
		if v, ok := metrics.Totals["spans_sent"].(float64); ok {
			agentData.SpansSent = v
		}
		if v, ok := metrics.Totals["logs_received"].(float64); ok {
			agentData.LogsReceived = v
		}
		if v, ok := metrics.Totals["logs_sent"].(float64); ok {
			agentData.LogsSent = v
		}
		if v, ok := metrics.Totals["metrics_received"].(float64); ok {
			agentData.MetricsReceived = v
		}
		if v, ok := metrics.Totals["metrics_sent"].(float64); ok {
			agentData.MetricsSent = v
		}
	}

	// Aggregate rates from all components for this agent, filtered by pipeline type
	if rates != nil {
		// Aggregate rates by pipeline type to avoid mixing data from different pipelines
		for pipelineType, pipeline := range pipelineInfo.Pipelines {
			// Skip traces pipeline if no data (same logic as in BuildTopology)
			if pipelineType == "traces" {
				hasTraceData := false
				for _, receiverName := range pipeline.Receivers {
					if receiverRates, ok := rates.Receivers[receiverName]; ok {
						if receiverRates.SpansReceivedRate > 0 {
							hasTraceData = true
							break
						}
					}
				}
				if !hasTraceData {
					continue
				}
			}

			// Aggregate receiver rates for this pipeline type
			// Receivers are the primary source for received rates
			for _, receiverName := range pipeline.Receivers {
				if receiverRates, ok := rates.Receivers[receiverName]; ok {
					switch pipelineType {
					case "traces":
						agentData.SpansReceivedRate += receiverRates.SpansReceivedRate
					case "logs":
						agentData.LogsReceivedRate += receiverRates.LogsReceivedRate
					case "metrics":
						agentData.MetricsReceivedRate += receiverRates.MetricsReceivedRate
					}
				}
			}

			// Aggregate processor rates for this pipeline type
			// Processors process data, so their rates can be used as fallback or validation
			// For logs/metrics: use first processor incoming rate as fallback for received if receiver rates are 0
			// Use last processor outgoing rate as fallback for sent if exporter rates are 0 (or only debug exporters)
			hasNonDebugExporter := false
			for _, exporterName := range pipeline.Exporters {
				if exporterName != "debug" {
					hasNonDebugExporter = true
					break
				}
			}

			if len(pipeline.Processors) > 0 {
				switch pipelineType {
				case "logs":
					// Use first processor incoming rate as fallback for received rate if receiver rates are 0
					firstProcessor := pipeline.Processors[0]
					if processorRates, ok := rates.Processors[firstProcessor]; ok {
						if agentData.LogsReceivedRate == 0 && processorRates.LogsIncomingRate > 0 {
							agentData.LogsReceivedRate = processorRates.LogsIncomingRate
						}
					}
					// Note: For logs sent rate, we'll use exporter rates as primary source (see below)
					// Only use processor rates as fallback if no exporter rates are available
				case "metrics":
					// Use first processor incoming rate as fallback for received rate if receiver rates are 0
					firstProcessor := pipeline.Processors[0]
					if processorRates, ok := rates.Processors[firstProcessor]; ok {
						if agentData.MetricsReceivedRate == 0 && processorRates.MetricsIncomingRate > 0 {
							agentData.MetricsReceivedRate = processorRates.MetricsIncomingRate
						}
					}
					// Use last processor outgoing rate as fallback for sent rate if no exporter rates or only debug exporters
					if !hasNonDebugExporter {
						lastProcessor := pipeline.Processors[len(pipeline.Processors)-1]
						if processorRates, ok := rates.Processors[lastProcessor]; ok {
							if processorRates.MetricsOutgoingRate > 0 {
								agentData.MetricsSentRate = processorRates.MetricsOutgoingRate
							}
						}
					} else {
						// If we have non-debug exporters, use processor rates as fallback only if exporter rates are 0
						lastProcessor := pipeline.Processors[len(pipeline.Processors)-1]
						if processorRates, ok := rates.Processors[lastProcessor]; ok {
							if agentData.MetricsSentRate == 0 && processorRates.MetricsOutgoingRate > 0 {
								agentData.MetricsSentRate = processorRates.MetricsOutgoingRate
							}
						}
					}
				case "traces":
					// Use last processor outgoing rate as fallback for sent rate if exporter rates are 0
					lastProcessor := pipeline.Processors[len(pipeline.Processors)-1]
					if processorRates, ok := rates.Processors[lastProcessor]; ok {
						if agentData.SpansSentRate == 0 && processorRates.SpansOutgoingRate > 0 {
							agentData.SpansSentRate += processorRates.SpansOutgoingRate
						}
					}
				}
			}

			// Aggregate exporter rates for this pipeline type (skip debug exporters)
			// Use the actual exporter node data from componentDataMap to ensure consistency
			// This matches what's displayed in the exporter nodes
			for _, exporterName := range pipeline.Exporters {
				if exporterName == "debug" {
					continue
				}
				// First try to get from componentDataMap (this is what's shown in the UI)
				if exporterData, ok := componentDataMap[agentID][exporterName]; ok {
					if exporterNodeData, ok := exporterData[pipelineType]; ok {
						switch pipelineType {
						case "traces":
							if exporterNodeData.SpansSentRate > 0 {
								agentData.SpansSentRate += exporterNodeData.SpansSentRate
							}
						case "logs":
							// Use the exporter node's LogsSentRate (this matches what's shown in the UI)
							if exporterNodeData.LogsSentRate > 0 {
								agentData.LogsSentRate += exporterNodeData.LogsSentRate
							}
						case "metrics":
							if exporterNodeData.MetricsSentRate > 0 {
								agentData.MetricsSentRate += exporterNodeData.MetricsSentRate
							}
						}
						continue // Skip fallback to rates if we found data in componentDataMap
					}
				}
				// Fallback to rates if componentDataMap doesn't have the data yet
				if exporterRates, ok := rates.Exporters[exporterName]; ok {
					switch pipelineType {
					case "traces":
						agentData.SpansSentRate += exporterRates.SpansSentRate
					case "logs":
						agentData.LogsSentRate += exporterRates.LogsSentRate
					case "metrics":
						agentData.MetricsSentRate += exporterRates.MetricsSentRate
					}
				}
			}

			// Use processor rates as fallback only if exporter rates are 0
			// This ensures we show the actual output from exporters, not intermediate processor values
			if len(pipeline.Processors) > 0 {
				switch pipelineType {
				case "logs":
					// Only use processor rates as fallback if no exporter rates were found
					if agentData.LogsSentRate == 0 && hasNonDebugExporter {
						lastProcessor := pipeline.Processors[len(pipeline.Processors)-1]
						if processorRates, ok := rates.Processors[lastProcessor]; ok {
							if processorRates.LogsOutgoingRate > 0 {
								agentData.LogsSentRate = processorRates.LogsOutgoingRate
							}
						}
					} else if !hasNonDebugExporter {
						// If only debug exporters, use processor rates
						lastProcessor := pipeline.Processors[len(pipeline.Processors)-1]
						if processorRates, ok := rates.Processors[lastProcessor]; ok {
							if processorRates.LogsOutgoingRate > 0 {
								agentData.LogsSentRate = processorRates.LogsOutgoingRate
							}
						}
					}
				}
			}
		}
	}

	componentDataMap[agentID]["agent"] = make(map[string]*types.NodeDataTransfer)
	componentDataMap[agentID]["agent"]["all"] = agentData
}

// buildReceiverNodes builds receiver nodes for a pipeline
func (s *Service) buildReceiverNodes(agentNodeID, agentID, pipelineType string, receivers []string, componentDataMap map[string]map[string]map[string]*types.NodeDataTransfer, topology *types.TopologyResponse, nodeMap map[string]bool, componentNodeMap map[string]string, pipelineDataMap map[string]map[string]*types.EdgeDataTransfer, pipeline *types.PipelineConfig, metrics *types.TelemetryMetrics, rates *types.TelemetryRates) []string {
	receiverNodeIDs := []string{}
	for _, receiverName := range receivers {
		receiverID := fmt.Sprintf("receiver/%s", receiverName)
		componentKey := fmt.Sprintf("%s-%s-%s", agentNodeID, receiverID, pipelineType)

		if _, exists := componentNodeMap[componentKey]; !exists {
			receiverNodeID := fmt.Sprintf("receiver-%s-%s-%s", agentNodeID, receiverName, pipelineType)
			componentNodeMap[componentKey] = receiverNodeID

			if !nodeMap[receiverNodeID] {
				receiverNode := types.TopologyNode{
					ID:            receiverNodeID,
					Type:          "receiver",
					Name:          receiverName,
					AgentID:       agentID,
					ComponentType: "receiver",
					PipelineType:  pipelineType,
				}
				// Get receiver data - try componentDataMap first, then create from rates if needed
				if compData, ok := componentDataMap[agentID][receiverName]; ok {
					if dt, ok := compData[pipelineType]; ok {
						receiverNode.DataTransfer = dt
					} else {
						// Data not found for this pipeline type, create it from rates if available
						nodeData := &types.NodeDataTransfer{}
						if rates != nil {
							if receiverRates, ok := rates.Receivers[receiverName]; ok {
								switch pipelineType {
								case "traces":
									nodeData.SpansReceivedRate = receiverRates.SpansReceivedRate
									// Receivers pass data through, so sent rate equals received rate
									nodeData.SpansSentRate = receiverRates.SpansReceivedRate
								case "logs":
									nodeData.LogsReceivedRate = receiverRates.LogsReceivedRate
									// Receivers pass data through, so sent rate equals received rate
									nodeData.LogsSentRate = receiverRates.LogsReceivedRate
								case "metrics":
									nodeData.MetricsReceivedRate = receiverRates.MetricsReceivedRate
									// Receivers pass data through, so sent rate equals received rate
									nodeData.MetricsSentRate = receiverRates.MetricsReceivedRate
								}
							}
						}
						// Always set DataTransfer, even if rates are 0 (so frontend knows pipeline exists)
						receiverNode.DataTransfer = nodeData
						// Store it for future use
						if compData == nil {
							componentDataMap[agentID][receiverName] = make(map[string]*types.NodeDataTransfer)
						}
						componentDataMap[agentID][receiverName][pipelineType] = nodeData
					}
				} else {
					// Receiver not in componentDataMap, create from rates if available
					nodeData := &types.NodeDataTransfer{}
					if rates != nil {
						if receiverRates, ok := rates.Receivers[receiverName]; ok {
							switch pipelineType {
							case "traces":
								nodeData.SpansReceivedRate = receiverRates.SpansReceivedRate
								// Receivers pass data through, so sent rate equals received rate
								nodeData.SpansSentRate = receiverRates.SpansReceivedRate
							case "logs":
								nodeData.LogsReceivedRate = receiverRates.LogsReceivedRate
								// Receivers pass data through, so sent rate equals received rate
								nodeData.LogsSentRate = receiverRates.LogsReceivedRate
							case "metrics":
								nodeData.MetricsReceivedRate = receiverRates.MetricsReceivedRate
								// Receivers pass data through, so sent rate equals received rate
								nodeData.MetricsSentRate = receiverRates.MetricsReceivedRate
							}
						}
					}
					// Always set DataTransfer, even if rates are 0 (so frontend knows pipeline exists)
					receiverNode.DataTransfer = nodeData
					// Store it for future use
					if componentDataMap[agentID][receiverName] == nil {
						componentDataMap[agentID][receiverName] = make(map[string]*types.NodeDataTransfer)
					}
					componentDataMap[agentID][receiverName][pipelineType] = nodeData
				}
				// Always create the node, even if data is not available
				topology.Nodes = append(topology.Nodes, receiverNode)
				nodeMap[receiverNodeID] = true
			}
		}

		receiverNodeID := componentNodeMap[componentKey]
		receiverNodeIDs = append(receiverNodeIDs, receiverNodeID)

		// Create edge from agent to receiver
		// NOTE: Values are shown in nodes (boxes), not on edges (arrows)
		// Edges are kept minimal - just connection information
		edge := types.TopologyEdge{
			From:         agentNodeID,
			To:           receiverNodeID,
			PipelineType: pipelineType,
			Label:        fmt.Sprintf("%s pipeline", pipelineType),
		}
		// Don't set edge.DataTransfer - values should be in nodes, not on edges
		topology.Edges = append(topology.Edges, edge)
	}
	return receiverNodeIDs
}

// buildProcessorNodes builds processor nodes for a pipeline
func (s *Service) buildProcessorNodes(agentNodeID, agentID, pipelineType string, processors []string, receiverNames []string, receiverNodeIDs []string, componentDataMap map[string]map[string]map[string]*types.NodeDataTransfer, topology *types.TopologyResponse, nodeMap map[string]bool, componentNodeMap map[string]string, pipelineDataMap map[string]map[string]*types.EdgeDataTransfer, pipeline *types.PipelineConfig, metrics *types.TelemetryMetrics, rates *types.TelemetryRates) []string {
	processorNodeIDs := []string{}
	lastComponents := receiverNodeIDs
	if len(lastComponents) == 0 {
		lastComponents = []string{agentNodeID}
	}

	for _, processorName := range processors {
		processorID := fmt.Sprintf("processor/%s", processorName)
		componentKey := fmt.Sprintf("%s-%s-%s", agentNodeID, processorID, pipelineType)

		processorNodeID := ""
		if existingID, exists := componentNodeMap[componentKey]; exists {
			processorNodeID = existingID
		} else {
			processorNodeID = fmt.Sprintf("processor-%s-%s-%s", agentNodeID, processorName, pipelineType)
			componentNodeMap[componentKey] = processorNodeID
		}

		if !nodeMap[processorNodeID] {
			processorNode := types.TopologyNode{
				ID:            processorNodeID,
				Type:          "processor",
				Name:          processorName,
				AgentID:       agentID,
				ComponentType: "processor",
				PipelineType:  pipelineType,
			}
			// Get processor data - try componentDataMap first, then create from rates if needed
			if compData, ok := componentDataMap[agentID][processorName]; ok {
				if dt, ok := compData[pipelineType]; ok {
					processorNode.DataTransfer = dt
				} else {
					// Data not found for this pipeline type, create it from rates if available
					nodeData := &types.NodeDataTransfer{}
					if rates != nil {
						if processorRates, ok := rates.Processors[processorName]; ok {
							switch pipelineType {
							case "traces":
								nodeData.SpansReceivedRate = processorRates.SpansIncomingRate
								nodeData.SpansSentRate = processorRates.SpansOutgoingRate
							case "logs":
								nodeData.LogsReceivedRate = processorRates.LogsIncomingRate
								nodeData.LogsSentRate = processorRates.LogsOutgoingRate
							case "metrics":
								nodeData.MetricsReceivedRate = processorRates.MetricsIncomingRate
								nodeData.MetricsSentRate = processorRates.MetricsOutgoingRate
							}
						}
					}
					processorNode.DataTransfer = nodeData
					// Store it for future use
					if compData == nil {
						componentDataMap[agentID][processorName] = make(map[string]*types.NodeDataTransfer)
					}
					componentDataMap[agentID][processorName][pipelineType] = nodeData
				}
			} else {
				// Processor not in componentDataMap, create from rates if available
				nodeData := &types.NodeDataTransfer{}
				if rates != nil {
					if processorRates, ok := rates.Processors[processorName]; ok {
						switch pipelineType {
						case "traces":
							nodeData.SpansReceivedRate = processorRates.SpansIncomingRate
							nodeData.SpansSentRate = processorRates.SpansOutgoingRate
						case "logs":
							nodeData.LogsReceivedRate = processorRates.LogsIncomingRate
							nodeData.LogsSentRate = processorRates.LogsOutgoingRate
						case "metrics":
							nodeData.MetricsReceivedRate = processorRates.MetricsIncomingRate
							nodeData.MetricsSentRate = processorRates.MetricsOutgoingRate
						}
					}
				}
				processorNode.DataTransfer = nodeData
				// Store it for future use
				if componentDataMap[agentID][processorName] == nil {
					componentDataMap[agentID][processorName] = make(map[string]*types.NodeDataTransfer)
				}
				componentDataMap[agentID][processorName][pipelineType] = nodeData
			}

			// Check if all data transfer values are 0 (e.g., batch processor)
			// Skip rendering processor node if all values are 0
			shouldSkipNode := false
			if processorNode.DataTransfer != nil {
				dt := processorNode.DataTransfer
				hasNonZeroValue := dt.SpansReceivedRate > 0 || dt.SpansSentRate > 0 ||
					dt.LogsReceivedRate > 0 || dt.LogsSentRate > 0 ||
					dt.MetricsReceivedRate > 0 || dt.MetricsSentRate > 0 ||
					dt.SpansReceived > 0 || dt.SpansSent > 0 ||
					dt.LogsReceived > 0 || dt.LogsSent > 0 ||
					dt.MetricsReceived > 0 || dt.MetricsSent > 0

				// Skip if all values are 0 (e.g., batch processor)
				if !hasNonZeroValue {
					shouldSkipNode = true
				}
			} else {
				// No DataTransfer means no data, skip it
				shouldSkipNode = true
			}

			if shouldSkipNode {
				// Skip adding the node to topology, but mark it as processed to avoid duplicate work
				nodeMap[processorNodeID] = true
				// Continue to next processor without adding node or edges
				continue
			}

			// Add the node only if it has non-zero values
			topology.Nodes = append(topology.Nodes, processorNode)
			nodeMap[processorNodeID] = true

			// Add to processorNodeIDs for edge creation
			processorNodeIDs = append(processorNodeIDs, processorNodeID)

			// Connect from previous component
			// For the first visible processor, connect from ALL receivers (or previous visible processors)
			// For subsequent visible processors, connect from the previous visible processor
			if len(processorNodeIDs) == 1 {
				// First visible processor: connect from all receivers (or agent if no receivers)
				for _, receiverNodeID := range lastComponents {
					edge := types.TopologyEdge{
						From:         receiverNodeID,
						To:           processorNodeID,
						PipelineType: pipelineType,
						Label:        pipelineType,
					}
					// NOTE: Values are shown in nodes (boxes), not on edges (arrows)
					topology.Edges = append(topology.Edges, edge)
				}
			} else {
				// Subsequent visible processor: connect from previous visible processor
				prevID := processorNodeIDs[len(processorNodeIDs)-2]
				edge := types.TopologyEdge{
					From:         prevID,
					To:           processorNodeID,
					PipelineType: pipelineType,
					Label:        pipelineType,
				}
				// NOTE: Values are shown in nodes (boxes), not on edges (arrows)
				topology.Edges = append(topology.Edges, edge)
			}
		}
	}

	return processorNodeIDs
}

// buildExporterNodes builds exporter nodes for a pipeline
func (s *Service) buildExporterNodes(agentNodeID, agentID, pipelineType string, exporters []string, lastComponents []string, lastProcessorName string, componentDataMap map[string]map[string]map[string]*types.NodeDataTransfer, exporterEdgeDataMap map[string]map[string]map[string]*types.EdgeDataTransfer, topology *types.TopologyResponse, nodeMap map[string]bool, componentNodeMap map[string]string, pipeline *types.PipelineConfig, metrics *types.TelemetryMetrics, rates *types.TelemetryRates, endpointToAgents map[string][]string, agentEndpoints map[string]map[string][]string) {
	for _, exporterName := range exporters {
		if exporterName == "debug" {
			continue
		}

		exporterID := fmt.Sprintf("exporter/%s", exporterName)
		componentKey := fmt.Sprintf("%s-%s-%s", agentNodeID, exporterID, pipelineType)

		if _, exists := componentNodeMap[componentKey]; !exists {
			exporterNodeID := fmt.Sprintf("exporter-%s-%s-%s", agentNodeID, exporterName, pipelineType)
			componentNodeMap[componentKey] = exporterNodeID

			if !nodeMap[exporterNodeID] {
				exporterNode := types.TopologyNode{
					ID:            exporterNodeID,
					Type:          "exporter",
					Name:          exporterName,
					AgentID:       agentID,
					ComponentType: "exporter",
					PipelineType:  pipelineType,
				}
				// Get exporter data - try componentDataMap first, then create from rates if needed
				nodeData := &types.NodeDataTransfer{}
				if compData, ok := componentDataMap[agentID][exporterName]; ok {
					if dt, ok := compData[pipelineType]; ok {
						exporterNode.DataTransfer = dt
						nodeData = dt
					} else {
						// Data not found for this pipeline type, create it from rates if available
						// First, set received values from last processor (if it has outgoing rate data)
						if lastProcessorName != "" {
							if processorData, ok := componentDataMap[agentID][lastProcessorName]; ok {
								if processorNodeData, ok := processorData[pipelineType]; ok {
									// Check if processor has outgoing rate data
									hasOutgoingRate := false
									switch pipelineType {
									case "traces":
										hasOutgoingRate = processorNodeData.SpansSentRate > 0
										if hasOutgoingRate {
											nodeData.SpansReceivedRate = processorNodeData.SpansSentRate
										}
									case "logs":
										hasOutgoingRate = processorNodeData.LogsSentRate > 0
										if hasOutgoingRate {
											nodeData.LogsReceivedRate = processorNodeData.LogsSentRate
										}
									case "metrics":
										hasOutgoingRate = processorNodeData.MetricsSentRate > 0
										if hasOutgoingRate {
											nodeData.MetricsReceivedRate = processorNodeData.MetricsSentRate
										}
									}
								}
							}
							// Fallback: get from rates if componentDataMap doesn't have it yet
							if nodeData.LogsReceivedRate == 0 && nodeData.MetricsReceivedRate == 0 && nodeData.SpansReceivedRate == 0 && rates != nil {
								if processorRates, ok := rates.Processors[lastProcessorName]; ok {
									switch pipelineType {
									case "traces":
										if processorRates.SpansOutgoingRate > 0 {
											nodeData.SpansReceivedRate = processorRates.SpansOutgoingRate
										}
									case "logs":
										if processorRates.LogsOutgoingRate > 0 {
											nodeData.LogsReceivedRate = processorRates.LogsOutgoingRate
										}
									case "metrics":
										if processorRates.MetricsOutgoingRate > 0 {
											nodeData.MetricsReceivedRate = processorRates.MetricsOutgoingRate
										}
									}
								}
							}
						} else if len(pipeline.Receivers) > 0 && rates != nil {
							// No processors, use receiver rates
							firstReceiverName := pipeline.Receivers[0]
							if receiverRates, ok := rates.Receivers[firstReceiverName]; ok {
								switch pipelineType {
								case "traces":
									nodeData.SpansReceivedRate = receiverRates.SpansReceivedRate
								case "logs":
									nodeData.LogsReceivedRate = receiverRates.LogsReceivedRate
								case "metrics":
									nodeData.MetricsReceivedRate = receiverRates.MetricsReceivedRate
								}
							}
						}

						// Set sent rates from exporter rates
						// IMPORTANT: Exporters don't filter, so sent should match received
						// If received rate is already set, use it for sent rate
						// Only use exporter sent rate if received rate is not available
						if rates != nil {
							if exporterRates, ok := rates.Exporters[exporterName]; ok {
								switch pipelineType {
								case "traces":
									// Use received rate if available, otherwise use exporter sent rate
									if nodeData.SpansReceivedRate > 0 {
										nodeData.SpansSentRate = nodeData.SpansReceivedRate
									} else {
										nodeData.SpansSentRate = exporterRates.SpansSentRate
									}
								case "logs":
									// Use received rate if available, otherwise use exporter sent rate
									if nodeData.LogsReceivedRate > 0 {
										nodeData.LogsSentRate = nodeData.LogsReceivedRate
									} else {
										nodeData.LogsSentRate = exporterRates.LogsSentRate
									}
								case "metrics":
									// Use received rate if available, otherwise use exporter sent rate
									if nodeData.MetricsReceivedRate > 0 {
										nodeData.MetricsSentRate = nodeData.MetricsReceivedRate
									} else {
										nodeData.MetricsSentRate = exporterRates.MetricsSentRate
									}
								}
							}
						}

						// If still no processor data found and we have exporter sent rate,
						// use exporter sent rate as received rate (for processors like batch that don't filter)
						if lastProcessorName == "" && nodeData.LogsReceivedRate == 0 && nodeData.MetricsReceivedRate == 0 && nodeData.SpansReceivedRate == 0 {
							if nodeData.LogsSentRate > 0 && pipelineType == "logs" {
								nodeData.LogsReceivedRate = nodeData.LogsSentRate
							} else if nodeData.MetricsSentRate > 0 && pipelineType == "metrics" {
								nodeData.MetricsReceivedRate = nodeData.MetricsSentRate
							} else if nodeData.SpansSentRate > 0 && pipelineType == "traces" {
								nodeData.SpansReceivedRate = nodeData.SpansSentRate
							}
						}
						// Always set DataTransfer, even if rates are 0 (so frontend knows pipeline exists)
						exporterNode.DataTransfer = nodeData
						// Store it for future use
						if compData == nil {
							componentDataMap[agentID][exporterName] = make(map[string]*types.NodeDataTransfer)
						}
						componentDataMap[agentID][exporterName][pipelineType] = nodeData
					}
				} else {
					// Exporter not in componentDataMap, create from rates if available
					// First, set received values from last processor (if it has outgoing rate data)
					if lastProcessorName != "" {
						if processorData, ok := componentDataMap[agentID][lastProcessorName]; ok {
							if processorNodeData, ok := processorData[pipelineType]; ok {
								// Check if processor has outgoing rate data
								switch pipelineType {
								case "traces":
									if processorNodeData.SpansSentRate > 0 {
										nodeData.SpansReceivedRate = processorNodeData.SpansSentRate
									}
								case "logs":
									if processorNodeData.LogsSentRate > 0 {
										nodeData.LogsReceivedRate = processorNodeData.LogsSentRate
									}
								case "metrics":
									if processorNodeData.MetricsSentRate > 0 {
										nodeData.MetricsReceivedRate = processorNodeData.MetricsSentRate
									}
								}
							}
						}
						// Fallback: get from rates if componentDataMap doesn't have it yet
						if nodeData.LogsReceivedRate == 0 && nodeData.MetricsReceivedRate == 0 && nodeData.SpansReceivedRate == 0 && rates != nil {
							if processorRates, ok := rates.Processors[lastProcessorName]; ok {
								switch pipelineType {
								case "traces":
									if processorRates.SpansOutgoingRate > 0 {
										nodeData.SpansReceivedRate = processorRates.SpansOutgoingRate
									}
								case "logs":
									if processorRates.LogsOutgoingRate > 0 {
										nodeData.LogsReceivedRate = processorRates.LogsOutgoingRate
									}
								case "metrics":
									if processorRates.MetricsOutgoingRate > 0 {
										nodeData.MetricsReceivedRate = processorRates.MetricsOutgoingRate
									}
								}
							}
						}
					} else if len(pipeline.Processors) > 0 && rates != nil {
						// No lastProcessorName found, but we have processors - find the last one with data
						// Use the last processor in the list (before batch which doesn't report outgoing)
						lastProcInList := pipeline.Processors[len(pipeline.Processors)-1]
						// If it's batch, use the one before it (should be filter/logs)
						if lastProcInList == "batch" && len(pipeline.Processors) > 1 {
							lastProcInList = pipeline.Processors[len(pipeline.Processors)-2]
						}
						if processorRates, ok := rates.Processors[lastProcInList]; ok {
							switch pipelineType {
							case "logs":
								// Use outgoing rate if available, otherwise incoming rate
								if processorRates.LogsOutgoingRate > 0 {
									nodeData.LogsReceivedRate = processorRates.LogsOutgoingRate
									lastProcessorName = lastProcInList
								} else if processorRates.LogsIncomingRate > 0 {
									nodeData.LogsReceivedRate = processorRates.LogsIncomingRate
									lastProcessorName = lastProcInList
								}
							case "metrics":
								if processorRates.MetricsOutgoingRate > 0 {
									nodeData.MetricsReceivedRate = processorRates.MetricsOutgoingRate
									lastProcessorName = lastProcInList
								} else if processorRates.MetricsIncomingRate > 0 {
									nodeData.MetricsReceivedRate = processorRates.MetricsIncomingRate
									lastProcessorName = lastProcInList
								}
							case "traces":
								if processorRates.SpansOutgoingRate > 0 {
									nodeData.SpansReceivedRate = processorRates.SpansOutgoingRate
									lastProcessorName = lastProcInList
								} else if processorRates.SpansIncomingRate > 0 {
									nodeData.SpansReceivedRate = processorRates.SpansIncomingRate
									lastProcessorName = lastProcInList
								}
							}
						}
					} else if len(pipeline.Receivers) > 0 && rates != nil {
						// No processors, use receiver rates
						firstReceiverName := pipeline.Receivers[0]
						if receiverRates, ok := rates.Receivers[firstReceiverName]; ok {
							switch pipelineType {
							case "traces":
								nodeData.SpansReceivedRate = receiverRates.SpansReceivedRate
							case "logs":
								nodeData.LogsReceivedRate = receiverRates.LogsReceivedRate
							case "metrics":
								nodeData.MetricsReceivedRate = receiverRates.MetricsReceivedRate
							}
						}
					}

					// Set sent rates from exporter rates
					// IMPORTANT: Exporters don't filter, so sent should match received
					// If received rate is already set, use it for sent rate
					// Only use exporter sent rate if received rate is not available
					if rates != nil {
						if exporterRates, ok := rates.Exporters[exporterName]; ok {
							switch pipelineType {
							case "traces":
								// Use received rate if available, otherwise use exporter sent rate
								if nodeData.SpansReceivedRate > 0 {
									nodeData.SpansSentRate = nodeData.SpansReceivedRate
								} else {
									nodeData.SpansSentRate = exporterRates.SpansSentRate
								}
							case "logs":
								// Use received rate if available, otherwise use exporter sent rate
								if nodeData.LogsReceivedRate > 0 {
									nodeData.LogsSentRate = nodeData.LogsReceivedRate
								} else {
									nodeData.LogsSentRate = exporterRates.LogsSentRate
								}
							case "metrics":
								// Use received rate if available, otherwise use exporter sent rate
								if nodeData.MetricsReceivedRate > 0 {
									nodeData.MetricsSentRate = nodeData.MetricsReceivedRate
								} else {
									nodeData.MetricsSentRate = exporterRates.MetricsSentRate
								}
							}
						}
					}
					// Always set DataTransfer, even if rates are 0 (so frontend knows pipeline exists)
					exporterNode.DataTransfer = nodeData
					// Store it for future use
					if componentDataMap[agentID][exporterName] == nil {
						componentDataMap[agentID][exporterName] = make(map[string]*types.NodeDataTransfer)
					}
					componentDataMap[agentID][exporterName][pipelineType] = nodeData
				}
				topology.Nodes = append(topology.Nodes, exporterNode)
				nodeMap[exporterNodeID] = true
			}
		}

		exporterNodeID := componentNodeMap[componentKey]

		// Create edges from last components to exporter
		// Use the last processor's output rate (filtered data) if available
		if len(lastComponents) > 0 {
			for _, prevID := range lastComponents {
				edge := types.TopologyEdge{
					From:         prevID,
					To:           exporterNodeID,
					PipelineType: pipelineType,
					Label:        pipelineType,
				}

				// NOTE: Values are shown in nodes (boxes), not on edges (arrows)
				// Edges are kept minimal - just connection information
				// Don't set edge.DataTransfer - values should be in nodes, not on edges
				topology.Edges = append(topology.Edges, edge)
			}
		}

		// Handle endpoint connection
		endpoint, hasEndpoint := pipeline.Endpoints[exporterName]
		if hasEndpoint {
			endpointNodeID := fmt.Sprintf("endpoint-%s", s.configService.NormalizeEndpoint(endpoint))
			if !nodeMap[endpointNodeID] {
				topology.Nodes = append(topology.Nodes, types.TopologyNode{
					ID:       endpointNodeID,
					Type:     "endpoint",
					Name:     endpoint,
					Endpoint: endpoint,
				})
				nodeMap[endpointNodeID] = true
			}

			edge := types.TopologyEdge{
				From:         exporterNodeID,
				To:           endpointNodeID,
				PipelineType: pipelineType,
				Label:        fmt.Sprintf("%s via %s", pipelineType, exporterName),
			}
			topology.Edges = append(topology.Edges, edge)

			endpointToAgents[endpointNodeID] = append(endpointToAgents[endpointNodeID], agentNodeID)
			if agentEndpoints[agentNodeID][endpointNodeID] == nil {
				agentEndpoints[agentNodeID][endpointNodeID] = []string{}
			}
			agentEndpoints[agentNodeID][endpointNodeID] = append(agentEndpoints[agentNodeID][endpointNodeID], pipelineType)
		}
	}
}
