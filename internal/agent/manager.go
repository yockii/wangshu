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
func InitializeAgentManager(isTUIMode bool) (defaultAgent *Agent) {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	// 只有启用的agent才需要初始化
	enabledAgentNames := make(map[string]struct{})
	for _, ch := range config.DefaultCfg.Channels {
		if ch.Enabled {
			enabledAgentNames[ch.Agent] = struct{}{}
		}
	}

	workspaceCheckMap := make(map[string]struct{})

	defaultAgentName := ""
	var defaultAgentCfg *config.AgentConfig

	for name, ac := range config.DefaultCfg.Agents {
		if defaultAgentName == "" || name == constant.Default {
			defaultAgentName = name
			defaultAgentCfg = ac
		}

		// 只有启用的agent才需要初始化
		if _, ok := enabledAgentNames[name]; !ok {
			continue
		}

		// // 检查workspace是否存在
		// if _, ok := workspaceCheckMap[ac.Workspace]; ok {
		// 	slog.Warn("DUPLICATE workspace FOUND for agent, this MAY CAUSE UNEXPECTED BEHAVIOR", "agent", name, "workspace", ac.Workspace)
		// }
		// workspaceCheckMap[ac.Workspace] = struct{}{}

		// agent, err := NewAgent(
		// 	llm.GetProvider(ac.Provider),
		// 	name,
		// 	ac.Model,
		// 	ac.MemoryOrganizeTime,
		// 	24*time.Hour,
		// 	10,
		// 	ac.Workspace,
		// 	ac.EnableImageRecognition,
		// )
		agent, err := initialAgent(name, ac, workspaceCheckMap)
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

	if defaultAgent == nil && isTUIMode && defaultAgentName != "" && defaultAgentCfg != nil {
		// 没有任何channel启动，但是在tui mode下
		var err error
		defaultAgent, err = initialAgent(defaultAgentName, defaultAgentCfg, workspaceCheckMap)
		if err != nil {
			slog.Error("Failed to initialize default agent, this agent will not be available", "agent", defaultAgentName, "error", err)
			defaultAgent = nil
		}
	}

	return
}

func initialAgent(name string, ac *config.AgentConfig, workspaceCheckMap map[string]struct{}) (*Agent, error) {
	// 检查workspace是否存在
	if _, ok := workspaceCheckMap[ac.Workspace]; ok {
		slog.Warn("DUPLICATE workspace FOUND for agent, this MAY CAUSE UNEXPECTED BEHAVIOR", "agent", name, "workspace", ac.Workspace)
	}
	workspaceCheckMap[ac.Workspace] = struct{}{}

	agent, err := NewAgent(
		llm.GetProvider(ac.Provider),
		name,
		ac.Model,
		ac.MemoryOrganizeTime,
		24*time.Hour,
		10,
		ac.Workspace,
		ac.EnableImageRecognition,
	)
	if err != nil {
		return nil, err
	}
	return agent, nil
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

func ClearAgents() {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	for _, ag := range globalAgents {
		ag.Stop()
	}
	globalAgents = make(map[string]*Agent)
}
