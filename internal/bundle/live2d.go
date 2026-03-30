package bundle

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/internal/config"
)

type Live2dBundle struct{}

func (*Live2dBundle) GetModelFile() string {
	if config.DefaultCfg.Live2D.Enabled {
		dir := filepath.Join(config.DefaultCfg.Live2D.ModelDir, config.DefaultCfg.Live2D.ModelName)
		// 找到*.model3.json
		entries, err := os.ReadDir(dir)
		if err != nil {
			return ""
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".model3.json") {
				// 组装成 /live2d/{modelName}/{modelJson}
				return "/live2d/" + config.DefaultCfg.Live2D.ModelName + "/" + entry.Name()
			}
		}
	}
	return ""
}
