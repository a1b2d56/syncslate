package ws

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 90 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	rateLimit *rate.Limiter
	mu        sync.Mutex
	closed    bool
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256), // Bounded send queue (256 cap)
		userID:    userID,
		rateLimit: rate.NewLimiter(10, 10), // Max 10 msg/s
	}
}

func (c *Client) ReadPump(router *Router) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket read error", "user_id", c.userID, "error", err)
			}
			break
		}

		if !c.rateLimit.Allow() {
			c.SendJSON("error", map[string]string{"message": "rate limit exceeded"})
			continue
		}

		router.HandleEvent(c, message)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current frame
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendJSON(eventType string, payload interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}

	data, err := marshalEvent(eventType, payload)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
		// Send queue full -> slow consumer eviction
		slog.Warn("client send queue full, evicting", "user_id", c.userID)
		c.closed = true
		c.conn.Close()
	}
}
