package repository

import (
	"context"
	"fmt"
	"stockBackend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rewardRepository struct {
	db *pgxpool.Pool
}

// NewRewardRepository creates a new reward repository
func NewRewardRepository(db *pgxpool.Pool) RewardRepository {
	return &rewardRepository{db: db}
}

func (r *rewardRepository) Create(ctx context.Context, reward *models.Reward) (*models.Reward, error) {
	query := `
		INSERT INTO rewards (
			user_id, stock_symbol, quantity, event_type, event_id, event_timestamp,
			stock_price, total_value_inr, brokerage_fee, transaction_fee, net_value_inr,
			status, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		reward.UserID, reward.StockSymbol, reward.Quantity, reward.EventType,
		reward.EventID, reward.EventTimestamp, reward.StockPrice, reward.TotalValueINR,
		reward.BrokerageFee, reward.TransactionFee, reward.NetValueINR,
		reward.Status, reward.Notes,
	).Scan(&reward.ID, &reward.CreatedAt, &reward.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create reward: %w", err)
	}
	return reward, nil
}

func (r *rewardRepository) GetByID(ctx context.Context, id int) (*models.Reward, error) {
	query := `
		SELECT id, user_id, stock_symbol, quantity, event_type, event_id, event_timestamp,
			stock_price, total_value_inr, brokerage_fee, transaction_fee, net_value_inr,
			status, notes, created_at, updated_at
		FROM rewards
		WHERE id = $1
	`
	reward := &models.Reward{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&reward.ID, &reward.UserID, &reward.StockSymbol, &reward.Quantity,
		&reward.EventType, &reward.EventID, &reward.EventTimestamp, &reward.StockPrice,
		&reward.TotalValueINR, &reward.BrokerageFee, &reward.TransactionFee,
		&reward.NetValueINR, &reward.Status, &reward.Notes,
		&reward.CreatedAt, &reward.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("reward not found: %w", err)
	}
	return reward, nil
}

func (r *rewardRepository) GetByEventID(ctx context.Context, eventID string) (*models.Reward, error) {
	query := `
		SELECT id, user_id, stock_symbol, quantity, event_type, event_id, event_timestamp,
			stock_price, total_value_inr, brokerage_fee, transaction_fee, net_value_inr,
			status, notes, created_at, updated_at
		FROM rewards
		WHERE event_id = $1
	`
	reward := &models.Reward{}
	err := r.db.QueryRow(ctx, query, eventID).Scan(
		&reward.ID, &reward.UserID, &reward.StockSymbol, &reward.Quantity,
		&reward.EventType, &reward.EventID, &reward.EventTimestamp, &reward.StockPrice,
		&reward.TotalValueINR, &reward.BrokerageFee, &reward.TransactionFee,
		&reward.NetValueINR, &reward.Status, &reward.Notes,
		&reward.CreatedAt, &reward.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("reward not found: %w", err)
	}
	return reward, nil
}

func (r *rewardRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Reward, error) {
	query := `
		SELECT id, user_id, stock_symbol, quantity, event_type, event_id, event_timestamp,
			stock_price, total_value_inr, brokerage_fee, transaction_fee, net_value_inr,
			status, notes, created_at, updated_at
		FROM rewards
		WHERE user_id = $1
		ORDER BY event_timestamp DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRewards(rows)
}

func (r *rewardRepository) GetTodayRewards(ctx context.Context, userID string) ([]*models.Reward, error) {
	query := `
		SELECT id, user_id, stock_symbol, quantity, event_type, event_id, event_timestamp,
			stock_price, total_value_inr, brokerage_fee, transaction_fee, net_value_inr,
			status, notes, created_at, updated_at
		FROM rewards
		WHERE user_id = $1 
			AND DATE(event_timestamp) = CURRENT_DATE
			AND status = 'COMPLETED'
		ORDER BY event_timestamp DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRewards(rows)
}

func (r *rewardRepository) GetHistoricalINR(ctx context.Context, userID string, startDate, endDate string) ([]*models.Reward, error) {
	query := `
		SELECT id, user_id, stock_symbol, quantity, event_type, event_id, event_timestamp,
			stock_price, total_value_inr, brokerage_fee, transaction_fee, net_value_inr,
			status, notes, created_at, updated_at
		FROM rewards
		WHERE user_id = $1 
			AND event_timestamp BETWEEN $2 AND $3
			AND status = 'COMPLETED'
		ORDER BY event_timestamp DESC
	`
	rows, err := r.db.Query(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRewards(rows)
}

func (r *rewardRepository) Update(ctx context.Context, reward *models.Reward) error {
	query := `
		UPDATE rewards
		SET status = $1, notes = $2
		WHERE id = $3
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, reward.Status, reward.Notes, reward.ID).
		Scan(&reward.UpdatedAt)
}

func (r *rewardRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM rewards WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *rewardRepository) scanRewards(rows pgx.Rows) ([]*models.Reward, error) {
	var rewards []*models.Reward
	for rows.Next() {
		reward := &models.Reward{}
		if err := rows.Scan(
			&reward.ID, &reward.UserID, &reward.StockSymbol, &reward.Quantity,
			&reward.EventType, &reward.EventID, &reward.EventTimestamp, &reward.StockPrice,
			&reward.TotalValueINR, &reward.BrokerageFee, &reward.TransactionFee,
			&reward.NetValueINR, &reward.Status, &reward.Notes,
			&reward.CreatedAt, &reward.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rewards = append(rewards, reward)
	}
	return rewards, rows.Err()
}
