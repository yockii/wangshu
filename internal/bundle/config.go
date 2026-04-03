package bundle

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/runner"
)

type ConfigBundle struct{}

func (*ConfigBundle) GetConfig() *config.Config {
	return config.DefaultCfg
}

func (*ConfigBundle) ValidateConfig() error {
	return config.DefaultCfg.Validate()
}

func (*ConfigBundle) SaveConfig(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	config.SaveConfig(cfg)
	if _defaultAgent, err := runner.Reload(); err != nil {
		return err
	} else {
		DefaultChatBundle.SetAgent(_defaultAgent)
	}
	return nil
}

func (*ConfigBundle) SelectFolder(title string, defaultPath string) (string, error) {
	result, err := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(title).
		SetDirectory(defaultPath).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return result, nil
}

func (*ConfigBundle) GetModelList(modelDir string) []string {
	if modelDir == "" {
		return nil
	}
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return nil
	}
	var models []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			models = append(models, entry.Name())
		}
	}
	return models
}

func (*ConfigBundle) GetModelPath(modelDir string, modelName string) string {
	if modelDir == "" || modelName == "" {
		return ""
	}
	modelPath := filepath.Join(modelDir, modelName)
	entries, err := os.ReadDir(modelPath)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if !entry.IsDir() && (strings.HasSuffix(name, ".model.json") || strings.HasSuffix(name, ".model3.json")) {
			return filepath.Join(modelPath, entry.Name())
		}
	}
	return ""
}
