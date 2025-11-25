package repository

import (
	"context"
	"fmt"
	"stockBackend/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type corporateActionRepository struct {
	db *pgxpool.Pool
}

// NewCorporateActionRepository creates a new corporate action repository
func NewCorporateActionRepository(db *pgxpool.Pool) CorporateActionRepository {
	return &corporateActionRepository{db: db}
}

func (r *corporateActionRepository) Create(ctx context.Context, action *models.CorporateAction) error {
	query := `
		INSERT INTO corporate_actions (
			stock_symbol, action_type, action_date, ratio_from, ratio_to,
			new_symbol, description, applied
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		action.StockSymbol, action.ActionType, action.ActionDate,
		action.RatioFrom, action.RatioTo, action.NewSymbol,
		action.Description, action.Applied,
	).Scan(&action.ID, &action.CreatedAt, &action.UpdatedAt)
}

func (r *corporateActionRepository) GetByID(ctx context.Context, id int) (*models.CorporateAction, error) {
	query := `
		SELECT id, stock_symbol, action_type, action_date, ratio_from, ratio_to,
			new_symbol, description, applied, applied_at, created_at, updated_at
		FROM corporate_actions
		WHERE id = $1
	`
	action := &models.CorporateAction{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&action.ID, &action.StockSymbol, &action.ActionType, &action.ActionDate,
		&action.RatioFrom, &action.RatioTo, &action.NewSymbol, &action.Description,
		&action.Applied, &action.AppliedAt, &action.CreatedAt, &action.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("corporate action not found: %w", err)
	}
	return action, nil
}

func (r *corporateActionRepository) GetByStockSymbol(ctx context.Context, stockSymbol string) ([]*models.CorporateAction, error) {
	query := `
		SELECT id, stock_symbol, action_type, action_date, ratio_from, ratio_to,
			new_symbol, description, applied, applied_at, created_at, updated_at
		FROM corporate_actions
		WHERE stock_symbol = $1
		ORDER BY action_date DESC
	`
	rows, err := r.db.Query(ctx, query, stockSymbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanActions(rows)
}

func (r *corporateActionRepository) GetPendingActions(ctx context.Context) ([]*models.CorporateAction, error) {
	query := `
		SELECT id, stock_symbol, action_type, action_date, ratio_from, ratio_to,
			new_symbol, description, applied, applied_at, created_at, updated_at
		FROM corporate_actions
		WHERE applied = FALSE AND action_date <= CURRENT_DATE
		ORDER BY action_date ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanActions(rows)
}

func (r *corporateActionRepository) MarkApplied(ctx context.Context, id int) error {
	query := `
		UPDATE corporate_actions
		SET applied = TRUE, applied_at = $1
		WHERE id = $2
	`
	now := time.Now()
	_, err := r.db.Exec(ctx, query, now, id)
	return err
}

func (r *corporateActionRepository) Update(ctx context.Context, action *models.CorporateAction) error {
	query := `
		UPDATE corporate_actions
		SET stock_symbol = $1, action_type = $2, action_date = $3,
			ratio_from = $4, ratio_to = $5, new_symbol = $6, description = $7
		WHERE id = $8
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query,
		action.StockSymbol, action.ActionType, action.ActionDate,
		action.RatioFrom, action.RatioTo, action.NewSymbol,
		action.Description, action.ID,
	).Scan(&action.UpdatedAt)
}

func (r *corporateActionRepository) scanActions(rows pgx.Rows) ([]*models.CorporateAction, error) {
	var actions []*models.CorporateAction
	for rows.Next() {
		action := &models.CorporateAction{}
		if err := rows.Scan(
			&action.ID, &action.StockSymbol, &action.ActionType, &action.ActionDate,
			&action.RatioFrom, &action.RatioTo, &action.NewSymbol, &action.Description,
			&action.Applied, &action.AppliedAt, &action.CreatedAt, &action.UpdatedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}
