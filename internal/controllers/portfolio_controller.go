package controllers

import (
	"net/http"
	"stockBackend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PortfolioController handles portfolio and analytics endpoints
type PortfolioController struct {
	portfolioService *services.PortfolioService
	log              *logrus.Logger
}

// NewPortfolioController creates a new portfolio controller
func NewPortfolioController(portfolioService *services.PortfolioService, log *logrus.Logger) *PortfolioController {
	return &PortfolioController{
		portfolioService: portfolioService,
		log:              log,
	}
}

// GetTodayStocks retrieves today's stock rewards for a user
// GET /api/v1/today-stocks/:userId
func (pc *PortfolioController) GetTodayStocks(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	rewards, err := pc.portfolioService.GetTodayStocks(c.Request.Context(), userID)
	if err != nil {
		pc.log.Errorf("Failed to get today's stocks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get today's stocks",
			"message": err.Error(),
		})
		return
	}

	// Calculate total
	totalINR := 0.0
	totalQuantity := 0.0
	for _, reward := range rewards {
		totalINR += reward.TotalValueINR
		totalQuantity += reward.Quantity
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":        userID,
		"date":           "today",
		"rewards":        rewards,
		"count":          len(rewards),
		"total_quantity": totalQuantity,
		"total_inr":      totalINR,
	})
}

// GetHistoricalINR retrieves historical INR values for a user
// GET /api/v1/historical-inr/:userId?start_date=2024-01-01&end_date=2024-12-31
func (pc *PortfolioController) GetHistoricalINR(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	rewards, err := pc.portfolioService.GetHistoricalINR(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		pc.log.Errorf("Failed to get historical INR: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get historical data",
			"message": err.Error(),
		})
		return
	}

	// Calculate totals
	totalINR := 0.0
	totalQuantity := 0.0
	totalFees := 0.0
	for _, reward := range rewards {
		totalINR += reward.TotalValueINR
		totalQuantity += reward.Quantity
		totalFees += reward.BrokerageFee + reward.TransactionFee
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":        userID,
		"start_date":     startDate,
		"end_date":       endDate,
		"rewards":        rewards,
		"count":          len(rewards),
		"total_quantity": totalQuantity,
		"total_inr":      totalINR,
		"total_fees":     totalFees,
		"net_inr":        totalINR - totalFees,
	})
}

// GetUserStats retrieves aggregated statistics for a user
// GET /api/v1/stats/:userId
func (pc *PortfolioController) GetUserStats(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	stats, err := pc.portfolioService.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		pc.log.Errorf("Failed to get user stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get stats",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetUserPortfolio retrieves complete portfolio for a user
// GET /api/v1/portfolio/:userId
func (pc *PortfolioController) GetUserPortfolio(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	portfolio, err := pc.portfolioService.GetUserPortfolio(c.Request.Context(), userID)
	if err != nil {
		pc.log.Errorf("Failed to get portfolio: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get portfolio",
			"message": err.Error(),
		})
		return
	}

	// Calculate totals
	totalInvested := 0.0
	totalCurrentValue := 0.0
	totalProfitLoss := 0.0
	for _, item := range portfolio {
		totalInvested += item.TotalInvestedINR
		totalCurrentValue += item.CurrentValueINR
		totalProfitLoss += item.ProfitLossINR
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":             userID,
		"portfolio":           portfolio,
		"holdings_count":      len(portfolio),
		"total_invested_inr":  totalInvested,
		"total_current_value": totalCurrentValue,
		"total_profit_loss":   totalProfitLoss,
		"profit_loss_percent": func() float64 {
			if totalInvested > 0 {
				return (totalProfitLoss / totalInvested) * 100
			}
			return 0
		}(),
	})
}

// GetDailyHoldings retrieves daily holdings for a user
// GET /api/v1/holdings/:userId?date=2024-01-01
func (pc *PortfolioController) GetDailyHoldings(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	date := c.Query("date")

	holdings, err := pc.portfolioService.GetDailyHoldings(c.Request.Context(), userID, date)
	if err != nil {
		pc.log.Errorf("Failed to get daily holdings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get holdings",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"date":     date,
		"holdings": holdings,
		"count":    len(holdings),
	})
}
