package controllers

import (
	"net/http"
	"stockBackend/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RewardController handles reward-related endpoints
type RewardController struct {
	rewardService *services.RewardService
	log           *logrus.Logger
}

// NewRewardController creates a new reward controller
func NewRewardController(rewardService *services.RewardService, log *logrus.Logger) *RewardController {
	return &RewardController{
		rewardService: rewardService,
		log:           log,
	}
}

// CreateReward processes a new reward request
// POST /api/v1/reward
func (rc *RewardController) CreateReward(c *gin.Context) {
	var req services.RewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rc.log.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	response, err := rc.rewardService.ProcessReward(c.Request.Context(), &req)
	if err != nil {
		rc.log.Errorf("Failed to process reward: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process reward",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetRewardByEventID retrieves a reward by event ID
// GET /api/v1/reward/:eventId
func (rc *RewardController) GetRewardByEventID(c *gin.Context) {
	eventID := c.Param("eventId")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Event ID is required",
		})
		return
	}

	reward, err := rc.rewardService.GetRewardByEventID(c.Request.Context(), eventID)
	if err != nil {
		rc.log.Errorf("Failed to get reward: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Reward not found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": reward,
	})
}

// GetUserRewards retrieves rewards for a user
// GET /api/v1/rewards/:userId?limit=10&offset=0
func (rc *RewardController) GetUserRewards(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	rewards, err := rc.rewardService.GetUserRewards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		rc.log.Errorf("Failed to get user rewards: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get rewards",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   rewards,
		"count":  len(rewards),
		"limit":  limit,
		"offset": offset,
	})
}
