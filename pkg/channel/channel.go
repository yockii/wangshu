package channel

import (
	"context"

	"github.com/yockii/wangshu/pkg/bus"
)

type Channel interface {
	Start() error
	Stop() error
	SendMessage(ctx context.Context, msg bus.OutboundMessage) error
}

type MessageEvent struct {
	Content string
	From    string
	ChatID  string
}

const (
	ChannelCanSendText         = 1
	ChannelCanSendImage        = 2
	ChannelCanSendVideo        = 4
	ChannelCanSendAudio        = 8
	ChannelCanSendFile         = 16
	ChannelCanSendLocation     = 32
	ChannelCanSendSticker      = 64
	ChannelCanSendRichMedia    = 128
	ChannelCanReceiveText      = 256
	ChannelCanReceiveImage     = 512
	ChannelCanReceiveVideo     = 1024
	ChannelCanReceiveAudio     = 2048
	ChannelCanReceiveFile      = 4096
	ChannelCanReceiveLocation  = 8192
	ChannelCanReceiveSticker   = 16384
	ChannelCanReceiveRichMedia = 32768
	ChannelSupportStreaming    = 65536
	ChannelSupportWebhook      = 131072
	ChannelSupportPolling      = 262144
	ChannelSupportAction       = 524288
)
