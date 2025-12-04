package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/leaderboard-redis/internal/domain"
	"github.com/leaderboard-redis/internal/service"
	"github.com/leaderboard-redis/internal/websocket"
)

// Handler provides HTTP handlers for the leaderboard API
type Handler struct {
	service *service.LeaderboardService
	hub     *websocket.Hub
	logger  *slog.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(service *service.LeaderboardService, hub *websocket.Hub, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		hub:     hub,
		logger:  logger,
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Router creates and configures the HTTP router
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", h.HealthCheck)
	r.Get("/ready", h.ReadyCheck)

	// WebSocket endpoint
	r.Get("/ws", h.HandleWebSocket)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Score operations
		r.Post("/scores", h.SubmitScore)
		r.Post("/scores/batch", h.SubmitScoreBatch)

		// Leaderboard operations
		r.Route("/leaderboards", func(r chi.Router) {
			r.Post("/", h.CreateLeaderboard)
			r.Get("/", h.ListLeaderboards)

			r.Route("/{leaderboardID}", func(r chi.Router) {
				r.Get("/", h.GetLeaderboard)
				r.Delete("/", h.DeleteLeaderboard)
				r.Post("/reset", h.ResetLeaderboard)
				r.Get("/stats", h.GetStats)

				// Rankings
				r.Get("/top", h.GetTop)
				r.Get("/range", h.GetRange)
				r.Get("/around/{playerID}", h.GetAroundPlayer)
				r.Get("/player/{playerID}", h.GetPlayerRank)
				r.Delete("/player/{playerID}", h.RemovePlayer)
			})
		})

		// WebSocket info endpoint
		r.Get("/ws/stats", h.GetWebSocketStats)
	})

	return r
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeSuccess writes a successful JSON response
func (h *Handler) writeSuccess(w http.ResponseWriter, data interface{}) {
	h.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// writeError writes an error JSON response
func (h *Handler) writeError(w http.ResponseWriter, status int, err error) {
	h.writeJSON(w, status, APIResponse{
		Success: false,
		Error:   err.Error(),
	})
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	websocket.ServeWs(h.hub, h.logger, w, r)
}

// GetWebSocketStats returns WebSocket connection statistics
func (h *Handler) GetWebSocketStats(w http.ResponseWriter, r *http.Request) {
	h.writeSuccess(w, map[string]interface{}{
		"total_connections": h.hub.GetTotalConnections(),
	})
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeSuccess(w, map[string]string{"status": "healthy"})
}

// ReadyCheck returns service readiness status
func (h *Handler) ReadyCheck(w http.ResponseWriter, r *http.Request) {
	h.writeSuccess(w, map[string]string{"status": "ready"})
}

// SubmitScore handles score submission
func (h *Handler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	var submission domain.ScoreSubmission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if submission.PlayerID == "" || submission.LeaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if err := h.service.SubmitScore(r.Context(), submission); err != nil {
		if domain.IsNotFoundError(err) {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to submit score", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, map[string]string{"status": "accepted"})
}

// SubmitScoreBatch handles batch score submission
func (h *Handler) SubmitScoreBatch(w http.ResponseWriter, r *http.Request) {
	var batch domain.BatchScoreSubmission
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if len(batch.Scores) == 0 {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if err := h.service.SubmitScoreBatch(r.Context(), batch); err != nil {
		h.logger.Error("failed to submit score batch", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":   "accepted",
		"received": len(batch.Scores),
	})
}

// CreateLeaderboard handles leaderboard creation
func (h *Handler) CreateLeaderboard(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateLeaderboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	config, err := h.service.CreateLeaderboard(r.Context(), req)
	if err != nil {
		if err == domain.ErrLeaderboardExists {
			h.writeError(w, http.StatusConflict, err)
			return
		}
		if err == domain.ErrInvalidLeaderboard {
			h.writeError(w, http.StatusBadRequest, err)
			return
		}
		h.logger.Error("failed to create leaderboard", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    config,
	})
}

// ListLeaderboards returns all leaderboards
func (h *Handler) ListLeaderboards(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.ListLeaderboards(r.Context())
	if err != nil {
		h.logger.Error("failed to list leaderboards", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, configs)
}

// GetLeaderboard returns a leaderboard by ID
func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	config, err := h.service.GetLeaderboard(r.Context(), leaderboardID)
	if err != nil {
		if err == domain.ErrLeaderboardNotFound {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to get leaderboard", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, config)
}

// DeleteLeaderboard deletes a leaderboard
func (h *Handler) DeleteLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if err := h.service.DeleteLeaderboard(r.Context(), leaderboardID); err != nil {
		if err == domain.ErrLeaderboardNotFound {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to delete leaderboard", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, map[string]string{"status": "deleted"})
}

// ResetLeaderboard clears all scores from a leaderboard
func (h *Handler) ResetLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if err := h.service.ResetLeaderboard(r.Context(), leaderboardID); err != nil {
		if err == domain.ErrLeaderboardNotFound {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to reset leaderboard", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, map[string]string{"status": "reset"})
}

// GetStats returns statistics for a leaderboard
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	stats, err := h.service.GetStats(r.Context(), leaderboardID)
	if err != nil {
		h.logger.Error("failed to get stats", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, stats)
}

// GetTop returns top N players from a leaderboard
func (h *Handler) GetTop(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	entries, err := h.service.GetTopN(r.Context(), leaderboardID, limit)
	if err != nil {
		h.logger.Error("failed to get top", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, entries)
}

// GetRange returns players within a specific rank range
func (h *Handler) GetRange(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	start := 0
	end := 10
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if s, err := strconv.Atoi(startStr); err == nil && s >= 0 {
			start = s
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if e, err := strconv.Atoi(endStr); err == nil && e >= start {
			end = e
		}
	}

	entries, err := h.service.GetRange(r.Context(), leaderboardID, start, end)
	if err != nil {
		h.logger.Error("failed to get range", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, entries)
}

// GetAroundPlayer returns players around a specific player's rank
func (h *Handler) GetAroundPlayer(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	playerID := chi.URLParam(r, "playerID")
	if leaderboardID == "" || playerID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	count := 5
	if rangeStr := r.URL.Query().Get("range"); rangeStr != "" {
		if c, err := strconv.Atoi(rangeStr); err == nil && c > 0 {
			count = c
		}
	}

	entries, err := h.service.GetAroundPlayer(r.Context(), leaderboardID, playerID, count)
	if err != nil {
		if err == domain.ErrPlayerNotFound {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to get around player", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, entries)
}

// GetPlayerRank returns a player's rank and score
func (h *Handler) GetPlayerRank(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	playerID := chi.URLParam(r, "playerID")
	if leaderboardID == "" || playerID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	entry, err := h.service.GetPlayerRank(r.Context(), leaderboardID, playerID)
	if err != nil {
		if err == domain.ErrPlayerNotFound {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to get player rank", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, entry)
}

// RemovePlayer removes a player from a leaderboard
func (h *Handler) RemovePlayer(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "leaderboardID")
	playerID := chi.URLParam(r, "playerID")
	if leaderboardID == "" || playerID == "" {
		h.writeError(w, http.StatusBadRequest, domain.ErrInvalidRequest)
		return
	}

	if err := h.service.RemovePlayer(r.Context(), leaderboardID, playerID); err != nil {
		if err == domain.ErrPlayerNotFound {
			h.writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to remove player", "error", err)
		h.writeError(w, http.StatusInternalServerError, domain.ErrInternalError)
		return
	}

	h.writeSuccess(w, map[string]string{"status": "removed"})
}
