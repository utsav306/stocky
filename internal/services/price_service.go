package services

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"stockBackend/internal/models"
	"stockBackend/internal/repository"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// PriceService handles stock price updates
type PriceService struct {
	priceRepo repository.StockPriceRepository
	log       *logrus.Logger
	cron      *cron.Cron
	minPrice  float64
	maxPrice  float64
	stocks    []string
}

// NewPriceService creates a new price service
func NewPriceService(priceRepo repository.StockPriceRepository, log *logrus.Logger) *PriceService {
	minPrice := 100.0
	maxPrice := 5000.0

	if min := os.Getenv("MOCK_PRICE_MIN"); min != "" {
		if val, err := strconv.ParseFloat(min, 64); err == nil {
			minPrice = val
		}
	}
	if max := os.Getenv("MOCK_PRICE_MAX"); max != "" {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			maxPrice = val
		}
	}

	return &PriceService{
		priceRepo: priceRepo,
		log:       log,
		cron:      cron.New(),
		minPrice:  minPrice,
		maxPrice:  maxPrice,
		stocks: []string{
			"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN",
			"META", "NVDA", "NFLX", "AMD", "INTC",
		},
	}
}

// Start begins the scheduled price updates
func (s *PriceService) Start() error {
	// Get interval from environment (default 1 hour)
	interval := "1h"
	if envInterval := os.Getenv("PRICE_UPDATE_INTERVAL_HOURS"); envInterval != "" {
		interval = envInterval + "h"
	}

	// Schedule hourly updates
	cronExpr := "@hourly"
	if interval == "1h" {
		cronExpr = "@hourly"
	} else {
		cronExpr = fmt.Sprintf("@every %s", interval)
	}

	_, err := s.cron.AddFunc(cronExpr, func() {
		ctx := context.Background()
		if err := s.UpdatePrices(ctx); err != nil {
			s.log.Errorf("Failed to update prices: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule price updates: %w", err)
	}

	s.cron.Start()
	s.log.Infof("Price service started with interval: %s", interval)

	// Run initial update
	go func() {
		ctx := context.Background()
		if err := s.UpdatePrices(ctx); err != nil {
			s.log.Errorf("Failed initial price update: %v", err)
		}
	}()

	return nil
}

// Stop stops the price service
func (s *PriceService) Stop() {
	if s.cron != nil {
		s.cron.Stop()
		s.log.Info("Price service stopped")
	}
}

// UpdatePrices updates prices for all stocks
func (s *PriceService) UpdatePrices(ctx context.Context) error {
	s.log.Info("Starting price update for all stocks")
	startTime := time.Now()

	prices := make([]*models.StockPrice, 0, len(s.stocks))
	for _, symbol := range s.stocks {
		price := s.generateMockPrice(symbol)
		prices = append(prices, &models.StockPrice{
			StockSymbol: symbol,
			Price:       price,
			Currency:    "INR",
			Source:      "MOCK_SERVICE",
			Timestamp:   time.Now(),
		})
	}

	// Bulk insert prices
	if err := s.priceRepo.BulkCreate(ctx, prices); err != nil {
		s.log.Errorf("Failed to save prices: %v", err)
		return err
	}

	duration := time.Since(startTime)
	s.log.Infof("Successfully updated %d stock prices in %v", len(prices), duration)
	
	return nil
}

// UpdateSinglePrice updates price for a single stock
func (s *PriceService) UpdateSinglePrice(ctx context.Context, symbol string) (*models.StockPrice, error) {
	s.log.Infof("Updating price for stock: %s", symbol)

	price := &models.StockPrice{
		StockSymbol: symbol,
		Price:       s.generateMockPrice(symbol),
		Currency:    "INR",
		Source:      "MOCK_SERVICE",
		Timestamp:   time.Now(),
	}

	if err := s.priceRepo.Create(ctx, price); err != nil {
		return nil, fmt.Errorf("failed to save price: %w", err)
	}

	s.log.Infof("Updated price for %s: %.2f INR", symbol, price.Price)
	return price, nil
}

// GetLatestPrice retrieves the latest price for a stock
func (s *PriceService) GetLatestPrice(ctx context.Context, symbol string) (*models.StockPrice, error) {
	price, err := s.priceRepo.GetLatest(ctx, symbol)
	if err != nil {
		s.log.Warnf("No price found for %s, generating new price", symbol)
		// If no price exists, generate and save one
		return s.UpdateSinglePrice(ctx, symbol)
	}
	return price, nil
}

// GetLatestPrices retrieves latest prices for multiple stocks
func (s *PriceService) GetLatestPrices(ctx context.Context, symbols []string) (map[string]*models.StockPrice, error) {
	prices, err := s.priceRepo.GetLatestBatch(ctx, symbols)
	if err != nil {
		return nil, err
	}

	// For any missing symbols, generate prices
	for _, symbol := range symbols {
		if _, exists := prices[symbol]; !exists {
			if price, err := s.UpdateSinglePrice(ctx, symbol); err == nil {
				prices[symbol] = price
			}
		}
	}

	return prices, nil
}

// GetPriceHistory retrieves price history for a stock
func (s *PriceService) GetPriceHistory(ctx context.Context, symbol string, limit int) ([]*models.StockPrice, error) {
	return s.priceRepo.GetHistory(ctx, symbol, limit)
}

// generateMockPrice generates a random price with some volatility
func (s *PriceService) generateMockPrice(symbol string) float64 {
	// Use symbol as seed for some consistency
	seed := int64(0)
	for _, c := range symbol {
		seed += int64(c)
	}
	
	// Add time component for variation
	seed += time.Now().Unix()
	
	r := rand.New(rand.NewSource(seed))
	
	// Generate price in range with 2 decimal precision
	price := s.minPrice + r.Float64()*(s.maxPrice-s.minPrice)
	
	// Round to 2 decimal places
	price = float64(int(price*100)) / 100
	
	return price
}

// GetSupportedStocks returns list of supported stock symbols
func (s *PriceService) GetSupportedStocks() []string {
	return s.stocks
}

// AddStock adds a new stock symbol to track
func (s *PriceService) AddStock(symbol string) {
	for _, existing := range s.stocks {
		if existing == symbol {
			return
		}
	}
	s.stocks = append(s.stocks, symbol)
	s.log.Infof("Added new stock symbol: %s", symbol)
}
