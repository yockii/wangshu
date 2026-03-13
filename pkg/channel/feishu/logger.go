package feishu

import (
	"context"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/yockii/wangshu/pkg/logger"
)

type feishuLogger struct {
	l logger.BufferLogger
}

func (l feishuLogger) Debug(ctx context.Context, args ...interface{}) {
	l.l.Debug(ctx, args...)
}

func (l feishuLogger) Info(ctx context.Context, args ...interface{}) {
	l.l.Info(ctx, args...)
}

func (l feishuLogger) Warn(ctx context.Context, args ...interface{}) {
	l.l.Warn(ctx, args...)
}

func (l feishuLogger) Error(ctx context.Context, args ...interface{}) {
	l.l.Error(ctx, args...)
}

func newFeishuLogger() larkcore.Logger {
	return feishuLogger{l: logger.NewBufferLogger()}
}
