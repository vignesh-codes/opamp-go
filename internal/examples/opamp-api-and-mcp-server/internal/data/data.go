package data

import (
	"sync"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
)

// InstanceId is a 16-byte identifier for an agent instance
type InstanceId [16]byte

// Agent represents a connected Agent
type Agent struct {
	InstanceId      InstanceId
	InstanceIdStr   string
	conn            types.Connection
	mux             sync.RWMutex
	Status          *protobufs.AgentToServer
	EffectiveConfig string
}

// Agents manages all connected agents
type Agents struct {
	mux         sync.RWMutex
	agentsById  map[InstanceId]*Agent
	connections map[types.Connection]map[InstanceId]bool
}

// AllAgents is the global instance
var AllAgents = &Agents{
	agentsById:  make(map[InstanceId]*Agent),
	connections: make(map[types.Connection]map[InstanceId]bool),
}

// GetAllAgentsReadonlyClone returns a read-only clone of all agents
func (agents *Agents) GetAllAgentsReadonlyClone() map[InstanceId]*Agent {
	agents.mux.RLock()
	defer agents.mux.RUnlock()

	result := make(map[InstanceId]*Agent, len(agents.agentsById))
	for id, agent := range agents.agentsById {
		result[id] = agent
	}
	return result
}

// RemoveConnection removes all agents associated with a connection
func (agents *Agents) RemoveConnection(conn types.Connection) {
	agents.mux.Lock()
	defer agents.mux.Unlock()

	for instanceId := range agents.connections[conn] {
		delete(agents.agentsById, instanceId)
	}
	delete(agents.connections, conn)
}

// FindAgent finds an agent by instance ID
func (agents *Agents) FindAgent(agentId InstanceId) *Agent {
	agents.mux.RLock()
	defer agents.mux.RUnlock()
	return agents.agentsById[agentId]
}

// FindOrCreateAgent finds or creates an agent
func (agents *Agents) FindOrCreateAgent(instanceId InstanceId, conn types.Connection) *Agent {
	agents.mux.Lock()
	defer agents.mux.Unlock()

	agent, exists := agents.agentsById[instanceId]
	if !exists {
		agent = &Agent{
			InstanceId: instanceId,
			conn:       conn,
		}
		agents.agentsById[instanceId] = agent
	} else {
		// Update connection if it changed
		agent.conn = conn
	}

	// Track connection
	if agents.connections[conn] == nil {
		agents.connections[conn] = make(map[InstanceId]bool)
	}
	agents.connections[conn][instanceId] = true

	return agent
}

// UpdateStatus updates the agent's status
func (a *Agent) UpdateStatus(msg *protobufs.AgentToServer, response *protobufs.ServerToAgent) {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.Status = msg
	if msg.EffectiveConfig != nil {
		a.EffectiveConfig = string(msg.EffectiveConfig.ConfigMap.ConfigMap[""].Body)
	}
}
