package cron

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	cron "github.com/netresearch/go-cron"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/llm"
)

type CronManager struct {
	agentName string
	workspace string
	model     string
	provider  llm.Provider
	mu        sync.RWMutex
	cronJobs  map[string]*types.BasicJobInfo
	ctx       context.Context
	cancel    context.CancelFunc
	c         *cron.Cron
}

func NewCronManager(agentName, workspace, model string, provider llm.Provider) *CronManager {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &CronManager{
		agentName: agentName,
		workspace: workspace,
		model:     model,
		provider:  provider,
		mu:        sync.RWMutex{},
		cronJobs:  make(map[string]*types.BasicJobInfo),
		ctx:       ctx,
		cancel:    cancel,
		c: cron.New(
			cron.WithParser(cron.MustNewParser(
				cron.SecondOptional | cron.Minute | cron.Hour |
					cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
			))),
	}

	mgr.c.Start()
	go mgr.start()

	return mgr
}

func (mgr *CronManager) start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mgr.ctx.Done():
			return
		case <-ticker.C:
			mgr.scanJobs()
		}
	}
}

func (mgr *CronManager) scanJobs() {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	cronDir := filepath.Join(mgr.workspace, constant.DirCron)
	os.MkdirAll(cronDir, 0755)
	entries, err := os.ReadDir(cronDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != constant.ExtJSON {
			continue
		}
		jobJsonPath := filepath.Join(cronDir, entry.Name())
		data, err := os.ReadFile(jobJsonPath)
		if err != nil {
			slog.Warn("Failed to read job", "jobFile", jobJsonPath)
			continue
		}
		job := types.BasicJobInfo{}
		if err := json.Unmarshal(data, &job); err != nil {
			slog.Warn("Failed to unmarshal job", "jobFile", jobJsonPath)
			continue
		}
		if job.ID == "" {
			slog.Warn("Job ID is empty", "jobFile", jobJsonPath)
			continue
		}
		if _, ok := mgr.cronJobs[job.ID]; !ok {
			if job.Status == constant.CronStatusEnabled {
				j := job
				mgr.cronJobs[j.ID] = &j
				mgr.c.AddFunc(j.Schedule, func() {
					mgr.executeJob(&j)
				}, cron.WithName(j.ID))
			}
		} else {
			entry := mgr.c.EntryByName(job.ID)
			if !entry.Valid() {
				continue
			}
			if job.Status == constant.CronStatusEnabled && entry.Paused {
				mgr.c.ResumeEntryByName(job.ID)
			} else if job.Status == constant.CronStatusPaused && !entry.Paused {
				mgr.c.PauseEntryByName(job.ID)
			} else if job.Status == constant.CronStatusDisabled {
				mgr.c.RemoveByName(job.ID)
				// 删除文件
				os.Remove(jobJsonPath)
			}
		}
	}
}

func (mgr *CronManager) executeJob(job *types.BasicJobInfo) {
	ctx := context.Background()
	if err := mgr.Execute(ctx, job); err != nil {
		slog.Error("执行定时任务失败", "jobId", job.ID, "error", err)
		return
	}

	entry := mgr.c.EntryByName(job.ID)
	if entry.Valid() {
		job.NextRun = &entry.Next
	}
	// 更新job文件
	now := time.Now()
	job.LastRun = &now
	job.UpdatedAt = now
	if job.Once {
		job.Status = constant.CronStatusDisabled
	}

	jobJsonPath := filepath.Join(mgr.workspace, constant.DirCron, job.ID+constant.ExtJSON)
	data, err := json.Marshal(job)
	if err != nil {
		slog.Warn("Failed to marshal job", "jobID", job.ID)
		return
	}
	if err := os.WriteFile(jobJsonPath, data, 0644); err != nil {
		slog.Warn("Failed to write job", "jobID", job.ID)
		return
	}
}

func (mgr *CronManager) Stop() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.cancel()
	for _, job := range mgr.cronJobs {
		mgr.c.RemoveByName(job.ID)
	}
}
