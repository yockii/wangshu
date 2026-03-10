package agent

import (
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

var (
	globalAgents = make(map[string]*Agent)
	agentsMutex  sync.RWMutex
)

// InitializeAgentManager initializes the global agent manager with all agents
func InitializeAgentManager() (defaultAgent *Agent) {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	// 只有启用的agent才需要初始化
	enabledAgentNames := make(map[string]struct{})
	for _, ch := range config.DefaultCfg.Channels {
		if ch.Enabled {
			enabledAgentNames[ch.Agent] = struct{}{}
		}
	}

	for name, ac := range config.DefaultCfg.Agents {
		// 只有启用的agent才需要初始化
		if _, ok := enabledAgentNames[name]; !ok {
			continue
		}

		agent, err := NewAgent(
			llm.GetProvider(ac.Provider),
			name,
			ac.Model,
			24*time.Hour,
			10,
			ac.Workspace,
			ac.EnableImageRecognition,
		)
		if err != nil {
			slog.Error("Failed to initialize agent, this agent will not be available", "agent", name, "error", err)
			continue
		}
		// Set agent name and start cron manager
		globalAgents[name] = agent
		if name == constant.Default || defaultAgent == nil {
			defaultAgent = globalAgents[name]
		}
	}
	return
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
	maps.Copy(result, globalAgents)
	return result
}

// GetDefaultAgent returns the default agent
func GetDefaultAgent() (*Agent, bool) {
	return GetAgent(constant.Default)
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

func StopAllAgents() {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	for _, ag := range globalAgents {
		ag.Stop()
	}
}
