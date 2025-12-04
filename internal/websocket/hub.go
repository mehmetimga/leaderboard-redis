package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/leaderboard-redis/internal/domain"
)

// Message types
const (
	MessageTypeLeaderboardUpdate = "leaderboard_update"
	MessageTypePlayerUpdate      = "player_update"
	MessageTypeSubscribe         = "subscribe"
	MessageTypeUnsubscribe       = "unsubscribe"
	MessageTypePing              = "ping"
	MessageTypePong              = "pong"
	MessageTypeError             = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type          string      `json:"type"`
	LeaderboardID string      `json:"leaderboard_id,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
}

// LeaderboardUpdate contains leaderboard data for broadcast
type LeaderboardUpdate struct {
	LeaderboardID string                   `json:"leaderboard_id"`
	Entries       []domain.LeaderboardEntry `json:"entries"`
	TotalPlayers  int64                    `json:"total_players"`
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by leaderboard ID
	clients map[string]map[*Client]bool

	// All connected clients
	allClients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from clients
	broadcast chan *Message

	// Subscription requests
	subscribe chan *subscriptionRequest

	// Unsubscription requests
	unsubscribe chan *subscriptionRequest

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger
	logger *slog.Logger

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

type subscriptionRequest struct {
	client        *Client
	leaderboardID string
}

// NewHub creates a new Hub
func NewHub(logger *slog.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:     make(map[string]map[*Client]bool),
		allClients:  make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *Message, 256),
		subscribe:   make(chan *subscriptionRequest, 64),
		unsubscribe: make(chan *subscriptionRequest, 64),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	h.logger.Info("WebSocket hub started")
	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("WebSocket hub stopping")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.allClients[client] = true
			h.mu.Unlock()
			h.logger.Debug("client registered", "client_id", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.allClients[client]; ok {
				delete(h.allClients, client)
				// Remove from all leaderboard subscriptions
				for lbID, clients := range h.clients {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.clients, lbID)
						}
					}
				}
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Debug("client unregistered", "client_id", client.id)

		case req := <-h.subscribe:
			h.mu.Lock()
			if _, ok := h.clients[req.leaderboardID]; !ok {
				h.clients[req.leaderboardID] = make(map[*Client]bool)
			}
			h.clients[req.leaderboardID][req.client] = true
			h.mu.Unlock()
			h.logger.Debug("client subscribed", "client_id", req.client.id, "leaderboard_id", req.leaderboardID)

		case req := <-h.unsubscribe:
			h.mu.Lock()
			if clients, ok := h.clients[req.leaderboardID]; ok {
				delete(clients, req.client)
				if len(clients) == 0 {
					delete(h.clients, req.leaderboardID)
				}
			}
			h.mu.Unlock()
			h.logger.Debug("client unsubscribed", "client_id", req.client.id, "leaderboard_id", req.leaderboardID)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Stop stops the hub
func (h *Hub) Stop() {
	h.cancel()
}

// broadcastMessage sends a message to all subscribed clients
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("failed to marshal message", "error", err)
		return
	}

	// If message has a leaderboard ID, only send to subscribed clients
	if message.LeaderboardID != "" {
		if clients, ok := h.clients[message.LeaderboardID]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
					// Client's buffer is full, skip
					h.logger.Warn("client buffer full, skipping", "client_id", client.id)
				}
			}
		}
	} else {
		// Broadcast to all clients
		for client := range h.allClients {
			select {
			case client.send <- data:
			default:
				h.logger.Warn("client buffer full, skipping", "client_id", client.id)
			}
		}
	}
}

// BroadcastLeaderboardUpdate sends a leaderboard update to all subscribed clients
func (h *Hub) BroadcastLeaderboardUpdate(leaderboardID string, entries []domain.LeaderboardEntry, totalPlayers int64) {
	message := &Message{
		Type:          MessageTypeLeaderboardUpdate,
		LeaderboardID: leaderboardID,
		Data: LeaderboardUpdate{
			LeaderboardID: leaderboardID,
			Entries:       entries,
			TotalPlayers:  totalPlayers,
		},
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("broadcast channel full, dropping message")
	}
}

// BroadcastPlayerUpdate sends a player update notification
func (h *Hub) BroadcastPlayerUpdate(leaderboardID string, entry domain.LeaderboardEntry) {
	message := &Message{
		Type:          MessageTypePlayerUpdate,
		LeaderboardID: leaderboardID,
		Data:          entry,
		Timestamp:     time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("broadcast channel full, dropping message")
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Subscribe adds a client to a leaderboard subscription
func (h *Hub) Subscribe(client *Client, leaderboardID string) {
	h.subscribe <- &subscriptionRequest{
		client:        client,
		leaderboardID: leaderboardID,
	}
}

// Unsubscribe removes a client from a leaderboard subscription
func (h *Hub) Unsubscribe(client *Client, leaderboardID string) {
	h.unsubscribe <- &subscriptionRequest{
		client:        client,
		leaderboardID: leaderboardID,
	}
}

// GetSubscriberCount returns the number of subscribers for a leaderboard
func (h *Hub) GetSubscriberCount(leaderboardID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.clients[leaderboardID]; ok {
		return len(clients)
	}
	return 0
}

// GetTotalConnections returns the total number of connected clients
func (h *Hub) GetTotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.allClients)
}

