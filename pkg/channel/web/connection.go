package web

import (
	"log/slog"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// connectToServer 连接到WebSocket服务器
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

// monitor 监控WebSocket连接状态
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

// readLoop 读取消息循环
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

			c.handleIncomingMessage(msg)
		}
	}
}
