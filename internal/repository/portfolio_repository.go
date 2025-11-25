package repository

import (
	"context"
	"fmt"
	"stockBackend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type portfolioRepository struct {
	db *pgxpool.Pool
}

// NewPortfolioRepository creates a new portfolio repository
func NewPortfolioRepository(db *pgxpool.Pool) PortfolioRepository {
	return &portfolioRepository{db: db}
}

func (r *portfolioRepository) GetUserPortfolio(ctx context.Context, userID string) ([]*models.Portfolio, error) {
	query := `
		SELECT 
			user_id, stock_symbol, total_quantity, avg_purchase_price,
			total_invested_inr, total_fees, transaction_count,
			first_reward_date, last_reward_date
		FROM v_user_portfolio
		WHERE user_id = $1
		ORDER BY total_invested_inr DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var portfolios []*models.Portfolio
	for rows.Next() {
		portfolio := &models.Portfolio{}
		if err := rows.Scan(
			&portfolio.UserID, &portfolio.StockSymbol, &portfolio.TotalQuantity,
			&portfolio.AvgPurchasePrice, &portfolio.TotalInvestedINR, &portfolio.TotalFees,
			&portfolio.TransactionCount, &portfolio.FirstRewardDate, &portfolio.LastRewardDate,
		); err != nil {
			return nil, err
		}
		
		// Get current price for this stock
		currentPrice, err := r.getCurrentPrice(ctx, portfolio.StockSymbol)
		if err == nil && currentPrice > 0 {
			portfolio.CurrentPrice = currentPrice
			portfolio.CurrentValueINR = portfolio.TotalQuantity * currentPrice
			portfolio.ProfitLossINR = portfolio.CurrentValueINR - portfolio.TotalInvestedINR
			if portfolio.TotalInvestedINR > 0 {
				portfolio.ProfitLossPercent = (portfolio.ProfitLossINR / portfolio.TotalInvestedINR) * 100
			}
		}
		
		portfolios = append(portfolios, portfolio)
	}
	return portfolios, rows.Err()
}

func (r *portfolioRepository) GetDailyHoldings(ctx context.Context, userID string, date string) ([]*models.DailyHolding, error) {
	query := `
		SELECT user_id, stock_symbol, holding_date, daily_quantity, daily_value_inr
		FROM v_daily_holdings
		WHERE user_id = $1 AND holding_date = $2
		ORDER BY daily_value_inr DESC
	`
	rows, err := r.db.Query(ctx, query, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []*models.DailyHolding
	for rows.Next() {
		holding := &models.DailyHolding{}
		if err := rows.Scan(
			&holding.UserID, &holding.StockSymbol, &holding.HoldingDate,
			&holding.DailyQuantity, &holding.DailyValueINR,
		); err != nil {
			return nil, err
		}
		holdings = append(holdings, holding)
	}
	return holdings, rows.Err()
}

func (r *portfolioRepository) GetUserStats(ctx context.Context, userID string) (*models.UserStats, error) {
	// Get basic stats from rewards
	statsQuery := `
		SELECT 
			COUNT(*) as total_rewards,
			SUM(quantity) as total_stocks_quantity,
			SUM(total_value_inr) as total_invested_inr,
			SUM(brokerage_fee + transaction_fee) as total_fees_inr,
			COUNT(DISTINCT stock_symbol) as unique_stocks
		FROM rewards
		WHERE user_id = $1 AND status = 'COMPLETED'
	`
	
	stats := &models.UserStats{UserID: userID}
	err := r.db.QueryRow(ctx, statsQuery, userID).Scan(
		&stats.TotalRewards,
		&stats.TotalStocksQuantity,
		&stats.TotalInvestedINR,
		&stats.TotalFeesINR,
		&stats.UniqueStocks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Get current portfolio value
	portfolioValueQuery := `SELECT get_user_portfolio_value($1)`
	err = r.db.QueryRow(ctx, portfolioValueQuery, userID).Scan(&stats.CurrentPortfolioValue)
	if err != nil {
		// If function doesn't exist or fails, calculate manually
		stats.CurrentPortfolioValue = 0
	}

	// Calculate profit/loss
	stats.TotalProfitLossINR = stats.CurrentPortfolioValue - stats.TotalInvestedINR
	if stats.TotalInvestedINR > 0 {
		stats.TotalProfitLossPercent = (stats.TotalProfitLossINR / stats.TotalInvestedINR) * 100
	}

	return stats, nil
}

func (r *portfolioRepository) getCurrentPrice(ctx context.Context, stockSymbol string) (float64, error) {
	query := `SELECT get_latest_stock_price($1)`
	var price float64
	err := r.db.QueryRow(ctx, query, stockSymbol).Scan(&price)
	if err != nil {
		// Fallback to direct query if function doesn't exist
		fallbackQuery := `
			SELECT price FROM stock_prices
			WHERE stock_symbol = $1
			ORDER BY timestamp DESC
			LIMIT 1
		`
		err = r.db.QueryRow(ctx, fallbackQuery, stockSymbol).Scan(&price)
	}
	return price, err
}
