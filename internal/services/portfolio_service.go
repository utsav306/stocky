package services

import (
	"context"
	"fmt"
	"stockBackend/internal/models"
	"stockBackend/internal/repository"
	"time"

	"github.com/sirupsen/logrus"
)

// PortfolioService handles portfolio and analytics operations
type PortfolioService struct {
	portfolioRepo repository.PortfolioRepository
	rewardRepo    repository.RewardRepository
	log           *logrus.Logger
}

// NewPortfolioService creates a new portfolio service
func NewPortfolioService(
	portfolioRepo repository.PortfolioRepository,
	rewardRepo repository.RewardRepository,
	log *logrus.Logger,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		rewardRepo:    rewardRepo,
		log:           log,
	}
}

// GetTodayStocks retrieves today's stock rewards for a user
func (ps *PortfolioService) GetTodayStocks(ctx context.Context, userID string) ([]*models.Reward, error) {
	ps.log.Infof("Getting today's stocks for user %s", userID)
	
	rewards, err := ps.rewardRepo.GetTodayRewards(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's rewards: %w", err)
	}

	return rewards, nil
}

// GetHistoricalINR retrieves historical INR values for a user
func (ps *PortfolioService) GetHistoricalINR(ctx context.Context, userID string, startDate, endDate string) ([]*models.Reward, error) {
	ps.log.Infof("Getting historical INR for user %s from %s to %s", userID, startDate, endDate)

	// Validate dates
	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02") // Default to 1 month ago
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02") // Default to today
	}

	rewards, err := ps.rewardRepo.GetHistoricalINR(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	return rewards, nil
}

// GetUserStats retrieves aggregated statistics for a user
func (ps *PortfolioService) GetUserStats(ctx context.Context, userID string) (*models.UserStats, error) {
	ps.log.Infof("Getting stats for user %s", userID)

	stats, err := ps.portfolioRepo.GetUserStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return stats, nil
}

// GetUserPortfolio retrieves complete portfolio for a user
func (ps *PortfolioService) GetUserPortfolio(ctx context.Context, userID string) ([]*models.Portfolio, error) {
	ps.log.Infof("Getting portfolio for user %s", userID)

	portfolio, err := ps.portfolioRepo.GetUserPortfolio(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return portfolio, nil
}

// GetDailyHoldings retrieves daily holdings for a user
func (ps *PortfolioService) GetDailyHoldings(ctx context.Context, userID string, date string) ([]*models.DailyHolding, error) {
	ps.log.Infof("Getting daily holdings for user %s on %s", userID, date)

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	holdings, err := ps.portfolioRepo.GetDailyHoldings(ctx, userID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily holdings: %w", err)
	}

	return holdings, nil
}
