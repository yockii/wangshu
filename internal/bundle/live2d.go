package bundle

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yockii/wangshu/internal/app"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/internal/store"
	"github.com/yockii/wangshu/internal/types"
	"github.com/yockii/wangshu/internal/variable"
	"github.com/yockii/wangshu/pkg/constant"
)

type Live2dBundle struct{}

func (*Live2dBundle) GetModelFile() string {
	if config.DefaultCfg.Live2D.Enabled {
		dir := filepath.Join(config.DefaultCfg.Live2D.ModelDir, config.DefaultCfg.Live2D.ModelName)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return ""
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".model.json") || strings.HasSuffix(name, ".model3.json") {
				return "/live2d/" + config.DefaultCfg.Live2D.ModelName + "/" + entry.Name()
			}
		}
	}
	return ""
}

func (*Live2dBundle) GetLive2DConfig() *types.Live2DConfig {
	return &config.DefaultCfg.Live2D
}

func (*Live2dBundle) IsEditMode() bool {
	return app.IsLive2DEditMode()
}

func (*Live2dBundle) ExitEditMode() {
	app.ExitLive2DEditMode()
}

func (*Live2dBundle) UpdateLive2DMotions(motions []types.Live2DMotion) {
	variable.Live2DMotions = motions
}

func (*Live2dBundle) UpdateLive2DExpressions(expressions []string) {
	variable.Live2DExpressions = expressions
}

func (*Live2dBundle) GetMotions() []types.Live2DMotion {
	return variable.Live2DMotions
}

func (*Live2dBundle) GetExpressions() []string {
	return variable.Live2DExpressions
}

func (*Live2dBundle) GetCurrentModelName() string {
	if config.DefaultCfg != nil {
		return config.DefaultCfg.Live2D.ModelName
	}
	return ""
}

func (*Live2dBundle) GetEmotions() []string {
	return constant.SpriteEmotions
}

func (*Live2dBundle) GetEmotionMapping(modelName string) *types.EmotionMapping {
	mapping, err := store.Get[*types.EmotionMapping](constant.StorePrefixEmotionMapping, modelName)
	if err != nil || mapping == nil {
		return &types.EmotionMapping{
			ID:       modelName,
			Mappings: make(map[string]*types.EmotionAction),
		}
	}
	return mapping
}

func (*Live2dBundle) SaveEmotionMapping(mapping *types.EmotionMapping) error {
	return store.Save(constant.StorePrefixEmotionMapping+config.DefaultCfg.Live2D.ModelName, mapping)
}

func (*Live2dBundle) PreviewMotion(group string, no int) {
	application.Get().Event.Emit(constant.EventLive2DDoMotion, map[string]any{
		"group": group,
		"no":    no,
	})
}

func (*Live2dBundle) PreviewExpression(id string) {
	application.Get().Event.Emit(constant.EventLive2DDoExpression, id)
}

func (*Live2dBundle) TriggerEmotion(emotion string) {
	modelName := ""
	if config.DefaultCfg != nil {
		modelName = config.DefaultCfg.Live2D.ModelName
	}
	if modelName == "" {
		return
	}
	mapping, err := store.Get[*types.EmotionMapping](constant.StorePrefixEmotionMapping, modelName)
	if err != nil || mapping == nil {
		return
	}
	action, ok := mapping.Mappings[emotion]
	if !ok || action == nil {
		return
	}
	if action.MotionGroup != "" {
		application.Get().Event.Emit(constant.EventLive2DDoMotion, map[string]interface{}{
			"group": action.MotionGroup,
			"no":    action.MotionNo,
		})
	}
	if action.ExpressionId != "" {
		application.Get().Event.Emit(constant.EventLive2DDoExpression, action.ExpressionId)
	}
}
