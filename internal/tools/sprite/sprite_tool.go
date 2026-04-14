package sprite

import (
	"context"
	"fmt"
	"strings"

	"github.com/yockii/wangshu/pkg/bus"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools/basic"
	"github.com/yockii/wangshu/pkg/tools/types"
)

type SpriteTool struct {
	basic.SimpleTool
}

func NewSpriteTool() *SpriteTool {
	tool := new(SpriteTool)
	tool.Name_ = constant.ToolNameSprite
	tool.Desc_ = "Control the sprite emotion. The sprite is a character as YOU."
	tool.Params_ = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"emotion": map[string]any{
				"type":        "string",
				"description": "The emotion to set for sprite",
				"enum":        constant.SpriteEmotions,
			},
		},
		"required": []string{"emotion"},
	}
	tool.ExecFunc = tool.execute
	return tool
}

func (*SpriteTool) execute(ctx context.Context, params map[string]string) *types.ToolResult {
	emotion, ok := params["emotion"]
	if !ok || emotion == "" {
		return types.NewToolResult().WithError(fmt.Errorf("emotion parameter is required"))
	}
	// emotion必须在几个枚举值中，校验一下
	switch emotion {
	case constant.SpriteEmotionHappy, constant.SpriteEmotionSad, constant.SpriteEmotionAngry, constant.SpriteEmotionNeutral, constant.SpriteEmotionExcited:
	default:
		return types.NewToolResult().WithError(
			fmt.Errorf("emotion must be one of %s",
				strings.Join(constant.SpriteEmotions, ",")))
	}

	bus.Default().PublishEmotion(emotion)
	return types.NewToolResult().WithRaw("ok")
}
