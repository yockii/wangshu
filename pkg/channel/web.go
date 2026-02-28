package channel

import (
	"context"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yockii/yoclaw/pkg/bus"
)

type WebChannel struct {
	name        string
	conn        *websocket.Conn
	connMu      sync.RWMutex
	hostAddress string
	token       string
	stopCh      chan struct{}
	reconnectCh chan struct{}
}

func NewWebChannel(name, hostAddress, token string) *WebChannel {
	return &WebChannel{
		name:        name,
		hostAddress: hostAddress,
		token:       token,
		stopCh:      make(chan struct{}, 1),
		reconnectCh: make(chan struct{}, 1),
	}
}

func (c *WebChannel) Start() error {
	slog.Info("Web channel starting", "name", c.name, "host", c.hostAddress)
	go c.connectToServer()
	go c.monitor()
	return nil
}

func (c *WebChannel) connectToServer() {
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	u := url.URL{Scheme: "ws", Host: c.hostAddress, Path: "/ws"}
	if c.token != "" {
		u.RawQuery = "token=" + c.token
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		slog.Error("Failed to connect to web server", "error", err)
		c.reconnectCh <- struct{}{}
		return
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	go c.readLoop()
}

func (c *WebChannel) Stop() error {
	slog.Info("Web channel stopping", "name", c.name)
	close(c.stopCh)

	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *WebChannel) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			slog.Debug("stop web channel")
			return
		case <-c.reconnectCh:
			slog.Debug("reconnect to web server")
			time.Sleep(5 * time.Second)
			c.connectToServer()
		case <-ticker.C:
			slog.Debug("keepalive web ws")
		}
	}
}

func (c *WebChannel) SubscribeOutbound(ctx context.Context, msg bus.OutboundMessage) {
	if msg.Channel == c.name {
		c.SendMessage(ctx, msg.ChatID, msg.Content)
	}
}

func (c *WebChannel) readLoop() {
	defer func() {
		c.connMu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
		c.reconnectCh <- struct{}{}
	}()

	for {
		select {
		case <-c.stopCh:
			return
		default:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			var msg struct {
				Type    string `json:"type"`
				Content string `json:"content"`
				Session string `json:"session,omitempty"`
			}
			if err := conn.ReadJSON(&msg); err != nil {
				slog.Error("Failed to read message from web server", "error", err)
				return
			}

			bus.Default().PublishInbound(bus.InboundMessage{
				Channel:  c.name,
				ChatID:   msg.Session,
				SenderID: "web",
				Content:  msg.Content,
			})
		}
	}
}

func (c *WebChannel) SendMessage(ctx context.Context, chatID, content string) error {
	c.connMu.RLock()
	defer c.connMu.RUnlock()

	if c.conn == nil {
		slog.Warn("Web channel not connected", "name", c.name)
		return nil
	}

	msg := map[string]interface{}{
		"type":    "message",
		"content": content,
		"chat_id": chatID,
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		slog.Error("Failed to send message to web server", "error", err)
		return err
	}

	return nil
}
