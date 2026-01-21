package handlers

import (
	"backend/internal/models"
	"backend/internal/service"
	"backend/internal/websocket"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
)

// LeaderboardHandler handles HTTP requests for the leaderboard
type LeaderboardHandler struct {
	service   *service.LeaderboardService
	validator *validator.Validate
	hub       *websocket.Hub
}

// NewLeaderboardHandler creates a new leaderboard handler
func NewLeaderboardHandler(service *service.LeaderboardService, hub *websocket.Hub) *LeaderboardHandler {
	return &LeaderboardHandler{
		service:   service,
		validator: validator.New(),
		hub:       hub,
	}
}

// UpdateScore handles POST /api/v1/scores
// @Summary Update user score
// @Description Creates or updates a user's rating
// @Accept json
// @Produce json
// @Param request body models.ScoreRequest true "Score update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/scores [post]
func (h *LeaderboardHandler) UpdateScore(c *fiber.Ctx) error {
	var req models.ScoreRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "Validation failed",
			Message: validationErrors.Error(),
		})
	}

	// Update score via service
	if err := h.service.UpdateScore(c.Context(), req.Username, req.Rating); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "Failed to update score",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Score updated successfully",
		"username": req.Username,
		"rating":   req.Rating,
	})
}

// GetLeaderboard handles GET /api/v1/leaderboard
// @Summary Get leaderboard
// @Description Retrieves the leaderboard with pagination
// @Accept json
// @Produce json
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(50)
// @Success 200 {object} models.LeaderboardResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/leaderboard [get]
func (h *LeaderboardHandler) GetLeaderboard(c *fiber.Ctx) error {
	// Parse query parameters
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100 // Max limit to prevent abuse
	}

	// Get leaderboard from service
	leaderboard, err := h.service.GetLeaderboard(c.Context(), offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "Failed to retrieve leaderboard",
			Message: err.Error(),
		})
	}

	// Set explicit no-cache headers to prevent any caching
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate, private, max-age=0")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	c.Set("X-Content-Type-Options", "nosniff")

	return c.Status(fiber.StatusOK).JSON(leaderboard)
}

// SearchUser handles GET /api/v1/search/:username
// @Summary Search for a user
// @Description Retrieves a user's global rank and rating
// @Accept json
// @Produce json
// @Param username path string true "Username to search"
// @Success 200 {object} models.SearchResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/search/{username} [get]
func (h *LeaderboardHandler) SearchUser(c *fiber.Ctx) error {
	username := c.Params("username")

	// Validate username
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "Invalid username",
			Message: "Username cannot be empty",
		})
	}

	// Search for user
	result, err := h.service.SearchUser(c.Context(), username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error:   "User not found",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// HealthCheck handles GET /api/v1/health
// @Summary Health check
// @Description Checks the health of the service and its dependencies
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} models.ErrorResponse
// @Router /api/v1/health [get]
func (h *LeaderboardHandler) HealthCheck(c *fiber.Ctx) error {
	if err := h.service.HealthCheck(c.Context()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error:   "Health check failed",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "healthy",
		"message": "All systems operational",
	})
}

// SimulateLoad handles POST /api/v1/debug/simulate
// @Summary Legacy simulation endpoint (DEPRECATED)
// @Description Simulation now runs automatically on server start
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/debug/simulate [post]
func (h *LeaderboardHandler) SimulateLoad(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "deprecated",
		"message": "This endpoint is deprecated. Simulation runs automatically on server start.",
		"note":    "Check server logs for simulation metrics (logged every 30 seconds).",
	})
}

// HandleWebSocket handles WebSocket connections at /ws
// @Summary WebSocket endpoint for real-time leaderboard updates
// @Description Upgrade HTTP connection to WebSocket for receiving real-time leaderboard updates
// @Router /ws [get]
func (h *LeaderboardHandler) HandleWebSocket(c *fiberws.Conn) {
	// Connection is already upgraded by Fiber WebSocket middleware
	// Serve the WebSocket connection through our hub
	websocket.ServeWS(h.hub, c)
}
