package controllers

import (
	"net/http"
	"stockBackend/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PriceController handles price-related endpoints
type PriceController struct {
	priceService *services.PriceService
	log          *logrus.Logger
}

// NewPriceController creates a new price controller
func NewPriceController(priceService *services.PriceService, log *logrus.Logger) *PriceController {
	return &PriceController{
		priceService: priceService,
		log:          log,
	}
}

// TriggerPriceUpdate manually triggers a price update for all stocks
// POST /api/v1/prices/update
func (pc *PriceController) TriggerPriceUpdate(c *gin.Context) {
	pc.log.Info("Manual price update triggered")

	if err := pc.priceService.UpdatePrices(c.Request.Context()); err != nil {
		pc.log.Errorf("Failed to update prices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update prices",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Prices updated successfully",
		"stocks":  pc.priceService.GetSupportedStocks(),
	})
}

// UpdateSingleStockPrice updates price for a single stock
// POST /api/v1/prices/update/:symbol
func (pc *PriceController) UpdateSingleStockPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Stock symbol is required",
		})
		return
	}

	price, err := pc.priceService.UpdateSinglePrice(c.Request.Context(), symbol)
	if err != nil {
		pc.log.Errorf("Failed to update price for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update price",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Price updated successfully",
		"data":    price,
	})
}

// GetLatestPrice retrieves the latest price for a stock
// GET /api/v1/prices/:symbol
func (pc *PriceController) GetLatestPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Stock symbol is required",
		})
		return
	}

	price, err := pc.priceService.GetLatestPrice(c.Request.Context(), symbol)
	if err != nil {
		pc.log.Errorf("Failed to get price for %s: %v", symbol, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Price not found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": price,
	})
}

// GetPriceHistory retrieves price history for a stock
// GET /api/v1/prices/:symbol/history?limit=10
func (pc *PriceController) GetPriceHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Stock symbol is required",
		})
		return
	}

	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	prices, err := pc.priceService.GetPriceHistory(c.Request.Context(), symbol, limit)
	if err != nil {
		pc.log.Errorf("Failed to get price history for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get price history",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  prices,
		"count": len(prices),
	})
}

// GetSupportedStocks returns list of supported stock symbols
// GET /api/v1/prices/stocks
func (pc *PriceController) GetSupportedStocks(c *gin.Context) {
	stocks := pc.priceService.GetSupportedStocks()
	c.JSON(http.StatusOK, gin.H{
		"data":  stocks,
		"count": len(stocks),
	})
}
