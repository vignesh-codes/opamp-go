package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/config"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/data"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/metrics"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/prometheus"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/topology"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/types"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/view"
)

// Handler handles HTTP API requests
type Handler struct {
	agents            *data.Agents
	topologyService   *topology.Service
	metricsService    *metrics.Service
	configService     *config.Service
	prometheusService *prometheus.Service
}

// New creates a new API handler
func New(agents *data.Agents) *Handler {
	// Initialize services
	metricsService := metrics.New()
	prometheusURL := getPrometheusURL()
	prometheusService := prometheus.New(prometheusURL)
	metricsService.SetPrometheusService(prometheusService)
	configService := config.New()
	topologyService := topology.New(metricsService, prometheusService, configService)

	return &Handler{
		agents:            agents,
		topologyService:   topologyService,
		metricsService:    metricsService,
		configService:     configService,
		prometheusService: prometheusService,
	}
}

func getPrometheusURL() string {
	if url := os.Getenv("PROMETHEUS_URL"); url != "" {
		return url
	}
	return "http://localhost:9090"
}

// getLiveAgents gets all agents from the live OpAMP server
func (h *Handler) getLiveAgents() map[string]*types.JSONAgent {
	if h.agents == nil {
		return make(map[string]*types.JSONAgent)
	}

	// Get live agents from OpAMP server
	liveAgents := h.agents.GetAllAgentsReadonlyClone()
	if len(liveAgents) == 0 {
		return make(map[string]*types.JSONAgent)
	}

	return h.convertLiveAgentsToMap(liveAgents)
}

// convertLiveAgentsToMap converts live data.Agent to JSONAgent format
func (h *Handler) convertLiveAgentsToMap(agents map[data.InstanceId]*data.Agent) map[string]*types.JSONAgent {
	result := make(map[string]*types.JSONAgent)
	for id, agent := range agents {
		agentID := uuid.UUID(id).String()
		result[agentID] = &types.JSONAgent{
			ID:            agentID,
			Labels:        h.extractLabelsFromAgent(agent),
			CurrentConfig: agent.EffectiveConfig,
		}
	}
	return result
}

// extractLabelsFromAgent extracts labels from a live agent
func (h *Handler) extractLabelsFromAgent(agent *data.Agent) map[string]string {
	labels := make(map[string]string)
	if agent.Status == nil || agent.Status.AgentDescription == nil {
		return labels
	}

	// Extract from IdentifyingAttributes
	if agent.Status.AgentDescription.IdentifyingAttributes != nil {
		for _, attr := range agent.Status.AgentDescription.IdentifyingAttributes {
			if attr.Value != nil {
				strVal := attr.Value.GetStringValue()
				if strVal != "" {
					labels[attr.Key] = strVal
				}
			}
		}
	}

	// Extract from NonIdentifyingAttributes
	if agent.Status.AgentDescription.NonIdentifyingAttributes != nil {
		for _, attr := range agent.Status.AgentDescription.NonIdentifyingAttributes {
			if attr.Value != nil {
				strVal := attr.Value.GetStringValue()
				if strVal != "" {
					labels[attr.Key] = strVal
				}
			}
		}
	}

	return labels
}

// HandleGetTopology handles GET /api/topology
func (h *Handler) HandleGetTopology(w http.ResponseWriter, r *http.Request) {
	// Get agents from live OpAMP server
	agentsMap := h.getLiveAgents()

	// Check if we have any agents
	if len(agentsMap) == 0 {
		log.Printf("[WARN] No agents found. Make sure agents are connected to the OpAMP server.")
		// Return empty topology instead of error
		emptyTopology := &types.TopologyResponse{
			Nodes: []types.TopologyNode{},
			Edges: []types.TopologyEdge{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(emptyTopology)
		return
	}

	topology, err := h.topologyService.BuildTopology(agentsMap)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build topology: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topology)
}

// HandleTopologyView handles GET /api/topology/view
func (h *Handler) HandleTopologyView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(view.GetTopologyViewHTML()))
}

// HandleHealth returns server health status
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()
	health := map[string]interface{}{
		"status":       "healthy",
		"agents_count": len(agentsMap),
		"source":       "live_opamp",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// HandleRoot returns API documentation
func (h *Handler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"name":        "OpAMP Topology Server",
		"description": "Topology visualization server for OpenTelemetry Collector fleet",
		"version":     "1.0.0",
		"endpoints": map[string]string{
			"GET /health":            "Server health status",
			"GET /api/agents":        "List all agents",
			"GET /api/agents/{id}":   "Get specific agent by ID",
			"GET /api/configs":       "Get all agent configurations",
			"GET /api/pipelines":     "Get parsed pipeline information",
			"GET /api/topology":      "Get topology graph (JSON)",
			"GET /api/topology/view": "Get topology visualization (HTML)",
			"GET /api/health":        "Get health status of all agents",
			"GET /api/summary":       "Get summary overview of all agents",
			"GET /api/transfer":      "Get data transfer metrics",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

// HandleListAgents handles GET /api/agents
func (h *Handler) HandleListAgents(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()

	response := make(map[string]map[string]interface{})
	for id, agent := range agentsMap {
		response[id] = map[string]interface{}{
			"id":             id,
			"labels":         agent.Labels,
			"current_config": agent.CurrentConfig,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetAgent handles GET /api/agents/{id}
func (h *Handler) HandleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	agentsMap := h.getLiveAgents()
	agent, ok := agentsMap[agentID]
	if !ok {
		http.Error(w, fmt.Sprintf("agent %s not found", agentID), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":             agentID,
		"labels":         agent.Labels,
		"current_config": agent.CurrentConfig,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetConfigs handles GET /api/configs
func (h *Handler) HandleGetConfigs(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()

	configs := make(map[string]map[string]string)
	for id, agent := range agentsMap {
		configs[id] = map[string]string{
			"default": agent.CurrentConfig,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// HandleGetPipelines handles GET /api/pipelines
func (h *Handler) HandleGetPipelines(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()

	pipelines := make(map[string]*types.AgentPipelineInfo)
	for id, agent := range agentsMap {
		pipelines[id] = h.configService.ParsePipelineConfig(agent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipelines)
}

// HandleGetHealth handles GET /api/health
func (h *Handler) HandleGetHealth(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()

	health := make(map[string]map[string]interface{})
	for id := range agentsMap {
		health[id] = map[string]interface{}{
			"healthy": true, // Agents connected to OpAMP are considered healthy
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// HandleGetSummary handles GET /api/summary
func (h *Handler) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()

	summary := map[string]interface{}{
		"total_agents": len(agentsMap),
		"agents":       []map[string]interface{}{},
	}

	for id, agent := range agentsMap {
		agentName := h.configService.GetAgentName(agent)
		pipelineInfo := h.configService.ParsePipelineConfig(agent)

		agentSummary := map[string]interface{}{
			"instance_id":     id,
			"instance_id_str": id,
			"healthy":         true,
			"name":            agentName,
			"pipeline_count":  len(pipelineInfo.Pipelines),
		}

		summary["agents"] = append(summary["agents"].([]map[string]interface{}), agentSummary)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// HandleGetDataTransfer handles GET /api/transfer
func (h *Handler) HandleGetDataTransfer(w http.ResponseWriter, r *http.Request) {
	agentsMap := h.getLiveAgents()

	transfer := make(map[string]map[string]interface{})

	for id, agent := range agentsMap {
		pipelineInfo := h.configService.ParsePipelineConfig(agent)
		metrics := h.metricsService.ScrapeCollectorMetrics(agent, pipelineInfo)

		agentName := h.configService.GetAgentName(agent)

		agentTransfer := map[string]interface{}{
			"instance_id":     id,
			"instance_id_str": id,
			"agent_name":      agentName,
			"receivers":       metrics.Receivers,
			"exporters":       metrics.Exporters,
			"processors":      metrics.Processors,
			"by_pipeline":     metrics.ByPipeline,
			"totals":          metrics.Totals,
		}

		transfer[id] = agentTransfer
	}

	// Calculate cluster-wide totals
	clusterTotals := map[string]interface{}{
		"total_agents":           len(agentsMap),
		"total_spans_received":   0.0,
		"total_spans_sent":       0.0,
		"total_logs_received":    0.0,
		"total_logs_sent":        0.0,
		"total_metrics_received": 0.0,
		"total_metrics_sent":     0.0,
	}

	for _, agentData := range transfer {
		if totals, ok := agentData["totals"].(map[string]interface{}); ok {
			if v, ok := totals["spans_received"].(float64); ok {
				clusterTotals["total_spans_received"] = clusterTotals["total_spans_received"].(float64) + v
			}
			if v, ok := totals["spans_sent"].(float64); ok {
				clusterTotals["total_spans_sent"] = clusterTotals["total_spans_sent"].(float64) + v
			}
			if v, ok := totals["logs_received"].(float64); ok {
				clusterTotals["total_logs_received"] = clusterTotals["total_logs_received"].(float64) + v
			}
			if v, ok := totals["logs_sent"].(float64); ok {
				clusterTotals["total_logs_sent"] = clusterTotals["total_logs_sent"].(float64) + v
			}
			if v, ok := totals["metrics_received"].(float64); ok {
				clusterTotals["total_metrics_received"] = clusterTotals["total_metrics_received"].(float64) + v
			}
			if v, ok := totals["metrics_sent"].(float64); ok {
				clusterTotals["total_metrics_sent"] = clusterTotals["total_metrics_sent"].(float64) + v
			}
		}
	}

	result := map[string]interface{}{
		"agents":         transfer,
		"cluster_totals": clusterTotals,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SetupRoutes configures all HTTP routes
func (h *Handler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/health", h.HandleHealth).Methods("GET")
	router.HandleFunc("/api/agents", h.HandleListAgents).Methods("GET")
	router.HandleFunc("/api/agents/{id}", h.HandleGetAgent).Methods("GET")
	router.HandleFunc("/api/configs", h.HandleGetConfigs).Methods("GET")
	router.HandleFunc("/api/pipelines", h.HandleGetPipelines).Methods("GET")
	router.HandleFunc("/api/topology", h.HandleGetTopology).Methods("GET")
	router.HandleFunc("/api/topology/view", h.HandleTopologyView).Methods("GET")
	router.HandleFunc("/api/health", h.HandleGetHealth).Methods("GET")
	router.HandleFunc("/api/summary", h.HandleGetSummary).Methods("GET")
	router.HandleFunc("/api/transfer", h.HandleGetDataTransfer).Methods("GET")

	// Root endpoint with API documentation
	router.HandleFunc("/", h.HandleRoot).Methods("GET")

	return router
}
