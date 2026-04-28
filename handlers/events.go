package handlers

import (
	"net/http"

	"fast-round-api/models"
	"fast-round-api/storage"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	Store  *storage.RedisStore
	APIKey string
}

func (h *EventHandler) HandleEvent(c *gin.Context) {
	var req models.EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	newState, err := h.Store.UpdateMatchEvent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Store.PublishUpdate(c.Request.Context(), "match_updates", newState); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish update failed"})
		return
	}

	c.JSON(http.StatusOK, newState)
}

func (h *EventHandler) HandleGetMatch(c *gin.Context) {
	matchID := c.Param("match_id")

	state, err := h.Store.GetMatchState(c.Request.Context(), matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get match failed"})
		return
	}

	c.JSON(http.StatusOK, state)
}

func (h *EventHandler) RequireAPIKey(c *gin.Context) {
	if h.APIKey == "" {
		c.Next()
		return
	}

	if c.GetHeader("X-API-Key") != h.APIKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.Next()
}
