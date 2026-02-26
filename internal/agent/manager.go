package agent

import (
	"sync"
)

var (
	globalAgents map[string]*Agent
	agentsMutex  sync.RWMutex
)

// InitializeAgentManager initializes the global agent manager with all agents
func InitializeAgentManager(agents map[string]*Agent) {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()
	globalAgents = agents
}

// GetAgent retrieves an agent by name
func GetAgent(name string) (*Agent, bool) {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()
	ag, ok := globalAgents[name]
	return ag, ok
}

// GetAllAgents returns all registered agents
func GetAllAgents() map[string]*Agent {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*Agent, len(globalAgents))
	for k, v := range globalAgents {
		result[k] = v
	}
	return result
}

// GetDefaultAgent returns the default agent
func GetDefaultAgent() (*Agent, bool) {
	return GetAgent("default")
}

// GetAnyAgent returns any available agent (tries default first, then first available)
func GetAnyAgent() (*Agent, bool) {
	// Try default first
	if ag, ok := GetDefaultAgent(); ok {
		return ag, true
	}

	// Get any available agent
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()

	for _, ag := range globalAgents {
		return ag, true
	}

	return nil, false
}

// GetAgentCount returns the number of registered agents
func GetAgentCount() int {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()
	return len(globalAgents)
}

// GetAgentNames returns a list of all agent names
func GetAgentNames() []string {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()

	names := make([]string, 0, len(globalAgents))
	for name := range globalAgents {
		names = append(names, name)
	}
	return names
}
