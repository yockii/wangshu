package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
	"github.com/yockii/wangshu/pkg/tools"
)

func (a *Agent) executionJob(job *types.BasicJobInfo) {
	msgs := []llm.Message{
		{
			Role:    constant.RoleSystem,
			Content: constant.CronJobExecutionPrompt,
		},
		{
			Role: constant.RoleUser,
			Content: fmt.Sprintf(`## 当前任务信息
任务ID: %s
任务描述: %s
`,
				job.ID,
				job.Description,
			),
		},
	}
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		availableTools := tools.GetDefaultToolRegistry().GetProviderDefs()
		resp, err := a.provider.Chat(ctx, a.model, msgs, availableTools, nil)
		if err != nil {
			slog.Error("LLM call failed", "jobId", job.ID, "error", err)
			return
		}
		if len(resp.Message.ToolCalls) == 0 {
			return
		}
		msgs = append(msgs, llm.Message{
			Role:      constant.RoleAssistant,
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		})
		// 执行所有的工具调用
		for _, tc := range resp.Message.ToolCalls {
			// EmitToolStart(sess.ID, tc.Name, tc.ID, args)
			toolResult, err := a.executeToolCall(ctx, tc, job.Channel, job.ChatID)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
				// EmitToolEnd(sess.ID, tc.Name, tc.ID, toolResult, true)
			} else {
				// EmitToolEnd(sess.ID, tc.Name, tc.ID, toolResult, false)
			}

			msgs = append(msgs, llm.Message{
				Role:      constant.RoleTool,
				Content:   toolResult,
				ToolCalls: []llm.ToolCall{tc},
			})
		}
	}
}
