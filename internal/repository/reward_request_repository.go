package repository

import (
	"context"
	"fmt"
	"stockBackend/internal/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type rewardRequestRepository struct {
	db *pgxpool.Pool
}

// NewRewardRequestRepository creates a new reward request repository
func NewRewardRequestRepository(db *pgxpool.Pool) RewardRequestRepository {
	return &rewardRequestRepository{db: db}
}

func (r *rewardRequestRepository) Create(ctx context.Context, request *models.RewardRequest) error {
	query := `
		INSERT INTO reward_requests (
			event_id, user_id, stock_symbol, quantity, request_payload, status
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		request.EventID, request.UserID, request.StockSymbol,
		request.Quantity, request.RequestPayload, request.Status,
	).Scan(&request.ID, &request.CreatedAt, &request.UpdatedAt)
}

func (r *rewardRequestRepository) GetByEventID(ctx context.Context, eventID string) (*models.RewardRequest, error) {
	query := `
		SELECT id, event_id, user_id, stock_symbol, quantity, request_payload,
			response_payload, status, processed_at, created_at, updated_at
		FROM reward_requests
		WHERE event_id = $1
	`
	request := &models.RewardRequest{}
	err := r.db.QueryRow(ctx, query, eventID).Scan(
		&request.ID, &request.EventID, &request.UserID, &request.StockSymbol,
		&request.Quantity, &request.RequestPayload, &request.ResponsePayload,
		&request.Status, &request.ProcessedAt, &request.CreatedAt, &request.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("reward request not found: %w", err)
	}
	return request, nil
}

func (r *rewardRequestRepository) Update(ctx context.Context, request *models.RewardRequest) error {
	query := `
		UPDATE reward_requests
		SET response_payload = $1, status = $2, processed_at = $3
		WHERE event_id = $4
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query,
		request.ResponsePayload, request.Status, request.ProcessedAt, request.EventID,
	).Scan(&request.UpdatedAt)
}

func (r *rewardRequestRepository) MarkProcessed(ctx context.Context, eventID string, responsePayload string) error {
	query := `
		UPDATE reward_requests
		SET response_payload = $1, status = 'COMPLETED', processed_at = $2
		WHERE event_id = $3
	`
	now := time.Now()
	_, err := r.db.Exec(ctx, query, responsePayload, now, eventID)
	return err
}

func (r *rewardRequestRepository) GetPending(ctx context.Context, limit int) ([]*models.RewardRequest, error) {
	query := `
		SELECT id, event_id, user_id, stock_symbol, quantity, request_payload,
			response_payload, status, processed_at, created_at, updated_at
		FROM reward_requests
		WHERE status = 'PROCESSING'
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.RewardRequest
	for rows.Next() {
		request := &models.RewardRequest{}
		if err := rows.Scan(
			&request.ID, &request.EventID, &request.UserID, &request.StockSymbol,
			&request.Quantity, &request.RequestPayload, &request.ResponsePayload,
			&request.Status, &request.ProcessedAt, &request.CreatedAt, &request.UpdatedAt,
		); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}
