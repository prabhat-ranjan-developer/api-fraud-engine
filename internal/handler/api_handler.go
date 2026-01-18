package handler

import (
	"fraud-engine/internal/domain"
	"fraud-engine/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *service.FraudService
}

func NewHandler(s *service.FraudService) *Handler {
	return &Handler{Service: s}
}

// CheckFraudV1, CheckFraudV2, AddBlacklistRule ... (Keep existing methods) ...

func (h *Handler) CheckFraudV1(c *gin.Context) {
	var req domain.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	result := h.Service.CheckFraudV1(c.Request.Context(), req)
	httpCode := http.StatusOK
	if result.Status == "BLOCK" {
		httpCode = http.StatusForbidden
	}
	c.JSON(httpCode, result)
}

func (h *Handler) CheckFraudV2(c *gin.Context) {
	// ... existing implementation ...
	var req domain.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	result := h.Service.CheckFraudV2(c.Request.Context(), req)
	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) AddBlacklistRule(c *gin.Context) {
	// ... existing implementation ...
	var req domain.RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	err := h.Service.BlacklistEntity(c.Request.Context(), req.Type, req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Rule added", "banned": req.Value})
}

// StreamFraudAlerts listens to the Go Channel and pushes SSE events
func (h *Handler) StreamFraudAlerts(msgChan <-chan string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Set Headers for Server-Sent Events (SSE)
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")

		c.Writer.Flush()

		// 2. Capture the Request Context
		// This channel closes automatically when the client disconnects
		clientGone := c.Request.Context().Done()

		// 3. Loop forever until client leaves or message arrives
		for {
			select {
			case <-clientGone:
				// Client closed the browser tab -> Stop the loop
				return
			case msg := <-msgChan:
				// New message from Kafka -> Send to Browser
				c.SSEvent("alert", msg)
				c.Writer.Flush()
			}
		}
	}
}
