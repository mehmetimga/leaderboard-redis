package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	id     string
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	logger *slog.Logger
}

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type          string `json:"type"`
	LeaderboardID string `json:"leaderboard_id,omitempty"`
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, logger *slog.Logger) *Client {
	return &Client{
		id:     uuid.New().String(),
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		logger: logger,
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
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
				c.logger.Error("websocket error", "error", err)
			}
			break
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			c.logger.Warn("invalid message format", "error", err)
			c.sendError("invalid message format")
			continue
		}

		c.handleMessage(&clientMsg)
	}
}

// handleMessage processes incoming client messages
func (c *Client) handleMessage(msg *ClientMessage) {
	switch msg.Type {
	case MessageTypeSubscribe:
		if msg.LeaderboardID != "" {
			c.hub.Subscribe(c, msg.LeaderboardID)
			c.sendAck("subscribed", msg.LeaderboardID)
		} else {
			c.sendError("leaderboard_id required for subscribe")
		}

	case MessageTypeUnsubscribe:
		if msg.LeaderboardID != "" {
			c.hub.Unsubscribe(c, msg.LeaderboardID)
			c.sendAck("unsubscribed", msg.LeaderboardID)
		}

	case MessageTypePing:
		c.sendPong()

	default:
		c.logger.Debug("unknown message type", "type", msg.Type)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
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
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
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

// sendError sends an error message to the client
func (c *Client) sendError(errMsg string) {
	msg := Message{
		Type:      MessageTypeError,
		Data:      map[string]string{"error": errMsg},
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// sendAck sends an acknowledgment message to the client
func (c *Client) sendAck(action, leaderboardID string) {
	msg := Message{
		Type:          action,
		LeaderboardID: leaderboardID,
		Data:          map[string]string{"status": "ok"},
		Timestamp:     time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// sendPong sends a pong response
func (c *Client) sendPong() {
	msg := Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// ServeWs handles WebSocket requests from peers
func ServeWs(hub *Hub, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", "error", err)
		return
	}

	client := NewClient(hub, conn, logger)
	hub.Register(client)

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	logger.Debug("new websocket connection", "client_id", client.id)
}

