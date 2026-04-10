package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/pkg/llm"
)

var DefaultManager = &McpManager{
	sessions: sync.Map{},
}

type McpManager struct {
	sessions sync.Map
}

func (m *McpManager) ReLoadMcpClients() error {
	m.CloseAll()

	var err error
	for name, mcpCfg := range config.DefaultCfg.McpServers {
		client := mcp.NewClient(&mcp.Implementation{
			Name: name,
		}, &mcp.ClientOptions{})
		if mcpCfg.TransportType == "" || mcpCfg.TransportType == "stdio" {
			cmd := exec.Command(mcpCfg.Command, mcpCfg.Args...)
			if len(mcpCfg.Env) > 0 {
				var envs []string
				for k, v := range mcpCfg.Env {
					envs = append(envs, k+"="+v)
				}
				cmd.Env = append(cmd.Environ(), envs...)
			}
			if mcpCfg.Cwd != "" {
				cmd.Dir = mcpCfg.Cwd
			}

			transport := &mcp.CommandTransport{Command: cmd}
			sess, connectErr := client.Connect(context.Background(), transport, nil)
			if connectErr != nil {
				err = multierror.Append(err, connectErr)
				continue
			}
			DefaultManager.sessions.Store(name, sess)
		}
	}
	return err
}

func (m *McpManager) CallMcpTool(ctx context.Context, mcpName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	sess, ok := m.sessions.Load(mcpName)
	if !ok {
		return nil, fmt.Errorf("mcp client not found: %s", mcpName)
	}
	session := sess.(*mcp.ClientSession)
	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}
	res, err := session.CallTool(ctx, params)
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return nil, fmt.Errorf("mcp tool error: %s", res.GetError().Error())
	}
	return res, nil
}

func (m *McpManager) GetMcpTools() ([]llm.ToolDefinition, error) {
	var res []llm.ToolDefinition
	m.sessions.Range(func(key, value any) bool {
		mcpName := key.(string)
		sess := value.(*mcp.ClientSession)
		tools, err := sess.ListTools(context.Background(), &mcp.ListToolsParams{})
		if err != nil {
			return true
		}
		for _, tool := range tools.Tools {
			params, ok := tool.InputSchema.(map[string]any)
			if !ok {
				continue
			}
			res = append(res, llm.ToolDefinition{
				Type: "function",
				Function: llm.ToolFunctionDefinition{
					Name:        McpToolPrefix + mcpName + ":" + tool.Name,
					Description: tool.Description,
					Parameters:  params,
				},
			})
		}
		return true
	})
	return res, nil
}

func (m *McpManager) CloseAll() error {
	m.sessions.Range(func(key, value any) bool {
		sess := value.(*mcp.ClientSession)
		sess.Close()
		m.sessions.Delete(key)
		return true
	})
	return nil
}
