package task

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/yockii/wangshu/pkg/llm"
)

type TaskManager struct {
	agentName string
	workspace string
	model     string
	provider  llm.Provider
	mu        sync.RWMutex
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewTaskManager(agentName, workspace, model string, provider llm.Provider) *TaskManager {
	ctx, cancel := context.WithCancel(context.Background())
	tm := &TaskManager{
		agentName: agentName,
		workspace: workspace,
		model:     model,
		provider:  provider,
		mu:        sync.RWMutex{},
		interval:  10 * time.Second,
		ctx:       ctx,
		cancel:    cancel,
	}
	go tm.start()
	return tm
}

func (tm *TaskManager) Stop() {
	if tm.cancel == nil {
		return
	}
	tm.cancel()
}

func (tm *TaskManager) start() {
	for {
		select {
		case <-tm.ctx.Done():
			return
		default:
		}

		startTime := time.Now()

		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("task manager run panic", "panic", r)
				}
			}()
			tm.run()
		}()

		elapsed := time.Since(startTime)
		if tm.interval > elapsed {
			select {
			case <-tm.ctx.Done():
				return
			case <-time.After(tm.interval - elapsed):
			}
		}
	}
}
