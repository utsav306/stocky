package repository

import (
	"context"
	"stockBackend/internal/models"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByUserID(ctx context.Context, userID string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id int) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, userID string) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	Exists(ctx context.Context, userID string) (bool, error)
}

// StockPriceRepository defines the interface for stock price operations
type StockPriceRepository interface {
	Create(ctx context.Context, price *models.StockPrice) error
	GetLatest(ctx context.Context, stockSymbol string) (*models.StockPrice, error)
	GetLatestBatch(ctx context.Context, stockSymbols []string) (map[string]*models.StockPrice, error)
	GetHistory(ctx context.Context, stockSymbol string, limit int) ([]*models.StockPrice, error)
	GetByTimeRange(ctx context.Context, stockSymbol string, start, end string) ([]*models.StockPrice, error)
	BulkCreate(ctx context.Context, prices []*models.StockPrice) error
}

// RewardRepository defines the interface for reward operations
type RewardRepository interface {
	Create(ctx context.Context, reward *models.Reward) (*models.Reward, error)
	GetByID(ctx context.Context, id int) (*models.Reward, error)
	GetByEventID(ctx context.Context, eventID string) (*models.Reward, error)
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Reward, error)
	GetTodayRewards(ctx context.Context, userID string) ([]*models.Reward, error)
	GetHistoricalINR(ctx context.Context, userID string, startDate, endDate string) ([]*models.Reward, error)
	Update(ctx context.Context, reward *models.Reward) error
	Delete(ctx context.Context, id int) error
}

// LedgerRepository defines the interface for ledger operations
type LedgerRepository interface {
	Create(ctx context.Context, entry *models.LedgerEntry) error
	BulkCreate(ctx context.Context, entries []*models.LedgerEntry) error
	GetByRewardID(ctx context.Context, rewardID int) ([]*models.LedgerEntry, error)
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.LedgerEntry, error)
	ValidateBalance(ctx context.Context, rewardID int) (bool, error)
}

// RewardRequestRepository defines the interface for idempotency operations
type RewardRequestRepository interface {
	Create(ctx context.Context, request *models.RewardRequest) error
	GetByEventID(ctx context.Context, eventID string) (*models.RewardRequest, error)
	Update(ctx context.Context, request *models.RewardRequest) error
	MarkProcessed(ctx context.Context, eventID string, responsePayload string) error
	GetPending(ctx context.Context, limit int) ([]*models.RewardRequest, error)
}

// CorporateActionRepository defines the interface for corporate action operations
type CorporateActionRepository interface {
	Create(ctx context.Context, action *models.CorporateAction) error
	GetByID(ctx context.Context, id int) (*models.CorporateAction, error)
	GetByStockSymbol(ctx context.Context, stockSymbol string) ([]*models.CorporateAction, error)
	GetPendingActions(ctx context.Context) ([]*models.CorporateAction, error)
	MarkApplied(ctx context.Context, id int) error
	Update(ctx context.Context, action *models.CorporateAction) error
}

// PortfolioRepository defines the interface for portfolio operations
type PortfolioRepository interface {
	GetUserPortfolio(ctx context.Context, userID string) ([]*models.Portfolio, error)
	GetDailyHoldings(ctx context.Context, userID string, date string) ([]*models.DailyHolding, error)
	GetUserStats(ctx context.Context, userID string) (*models.UserStats, error)
}
