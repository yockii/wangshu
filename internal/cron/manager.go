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
)

type Executor func(job *CronJob)

type CronManager struct {
	workspace string
	mu        sync.RWMutex
	cronJobs  map[string]*CronJob
	executor  Executor
	ctx       context.Context
	cancel    context.CancelFunc
	c         *cron.Cron
}

func NewManager(workspace string, executor Executor) *CronManager {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &CronManager{
		workspace: workspace,
		mu:        sync.RWMutex{},
		cronJobs:  make(map[string]*CronJob),
		executor:  executor,
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

	cronDir := filepath.Join(mgr.workspace, "cron")
	os.MkdirAll(cronDir, 0755)
	entries, err := os.ReadDir(cronDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		jobJsonPath := filepath.Join(cronDir, entry.Name())
		data, err := os.ReadFile(jobJsonPath)
		if err != nil {
			slog.Warn("Failed to read job", "jobFile", jobJsonPath)
			continue
		}
		var job CronJob
		if err := json.Unmarshal(data, &job); err != nil {
			slog.Warn("Failed to unmarshal job", "jobFile", jobJsonPath)
			continue
		}
		if job.ID == "" {
			slog.Warn("Job ID is empty", "jobFile", jobJsonPath)
			continue
		}
		if _, ok := mgr.cronJobs[job.ID]; !ok {
			if job.Status == "enabled" {
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
			if job.Status == "enabled" && entry.Paused {
				mgr.c.ResumeEntryByName(job.ID)
			} else if job.Status == "paused" && !entry.Paused {
				mgr.c.PauseEntryByName(job.ID)
			} else if job.Status == "disabled" {
				mgr.c.RemoveByName(job.ID)
				// 删除文件
				os.Remove(jobJsonPath)
			}
		}
	}
}

func (mgr *CronManager) executeJob(job *CronJob) {
	mgr.executor(job)

	entry := mgr.c.EntryByName(job.ID)
	if entry.Valid() {
		job.NextRun = &entry.Next
	}
	// 更新job文件
	now := time.Now()
	job.LastRun = &now
	job.UpdatedAt = now
	if job.Once {
		job.Status = "disabled"
	}

	jobJsonPath := filepath.Join(mgr.workspace, "cron", job.ID+".json")
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
