package bus

import (
	"context"
	"sync"

	"github.com/yockii/wangshu/pkg/constant"
)

type InboundHandler func(ctx context.Context, msg InboundMessage)
type OutboundHandler func(ctx context.Context, msg Message)

// MessageBus 处理消息传递，包括从通道到智能体的入站消息和从智能体到通道的出站消息
type MessageBus struct {
	inbound          chan InboundMessage
	outbound         chan Message
	inboundHandlers  map[string]InboundHandler
	outboundHandlers []OutboundHandler
	mu               sync.RWMutex
	closed           bool
}

// NewMessageBus 创建一个新的消息总线，bufferSize 为缓冲区大小
func NewMessageBus(bufferSize int) *MessageBus {
	return &MessageBus{
		inbound:          make(chan InboundMessage, bufferSize),
		outbound:         make(chan Message, bufferSize),
		inboundHandlers:  make(map[string]InboundHandler),
		outboundHandlers: make([]OutboundHandler, 0),
	}
}

func (b *MessageBus) Start(ctx context.Context) {
	go b.processInboundMessages(ctx)
	go b.processOutboundMessages(ctx)
}
func (b *MessageBus) RegisterInboundHandler(channel string, handler InboundHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inboundHandlers[channel] = handler
}
func (b *MessageBus) RegisterOutboundHandler(handler OutboundHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.outboundHandlers = append(b.outboundHandlers, handler)
}
func (b *MessageBus) processInboundMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-b.inbound:
			if !ok {
				return
			}
			// 防止动态新增
			b.mu.RLock()
			handler, ok := b.inboundHandlers[msg.Metadata.Channel]
			b.mu.RUnlock()
			if ok {
				go handler(ctx, msg)
			}
		}
	}
}
func (b *MessageBus) processOutboundMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-b.outbound:
			if !ok {
				return
			}
			b.mu.RLock()
			handlers := make([]OutboundHandler, len(b.outboundHandlers))
			copy(handlers, b.outboundHandlers)
			b.mu.RUnlock()
			for _, handler := range handlers {
				go handler(ctx, msg)
			}
		}
	}
}

// PublishInbound 发送一个入站消息到总线；通常为渠道收到用户消息后调用
func (b *MessageBus) PublishInbound(msg InboundMessage) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return nil
	}
	b.inbound <- msg
	return nil
}

func (b *MessageBus) PublishOutbound(msg Message) error {
	// 如果仅仅是HEARTBEAT_OK, 则不发送
	if msg.Content == constant.HEARTBEAT_OK || (msg.Content == "" && msg.Media == nil) {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return nil
	}
	b.outbound <- msg
	return nil
}

// Close 关闭消息总线，释放资源；在系统关闭时调用
func (b *MessageBus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true

	close(b.inbound)
	close(b.outbound)
	b.mu.Unlock()
}
