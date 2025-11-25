package repository

import (
	"context"
	"fmt"
	"stockBackend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ledgerRepository struct {
	db *pgxpool.Pool
}

// NewLedgerRepository creates a new ledger repository
func NewLedgerRepository(db *pgxpool.Pool) LedgerRepository {
	return &ledgerRepository{db: db}
}

func (r *ledgerRepository) Create(ctx context.Context, entry *models.LedgerEntry) error {
	query := `
		INSERT INTO ledger_entries (
			reward_id, user_id, entry_type, account_type, amount, currency, description, reference_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		entry.RewardID, entry.UserID, entry.EntryType, entry.AccountType,
		entry.Amount, entry.Currency, entry.Description, entry.ReferenceID,
	).Scan(&entry.ID, &entry.CreatedAt)
}

func (r *ledgerRepository) BulkCreate(ctx context.Context, entries []*models.LedgerEntry) error {
	if len(entries) == 0 {
		return nil
	}

	query := `
		INSERT INTO ledger_entries (
			reward_id, user_id, entry_type, account_type, amount, currency, description, reference_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	batch := &pgx.Batch{}
	for _, entry := range entries {
		batch.Queue(query,
			entry.RewardID, entry.UserID, entry.EntryType, entry.AccountType,
			entry.Amount, entry.Currency, entry.Description, entry.ReferenceID,
		)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range entries {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert ledger entry: %w", err)
		}
	}

	return nil
}

func (r *ledgerRepository) GetByRewardID(ctx context.Context, rewardID int) ([]*models.LedgerEntry, error) {
	query := `
		SELECT id, reward_id, user_id, entry_type, account_type, amount, currency,
			description, reference_id, created_at
		FROM ledger_entries
		WHERE reward_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, rewardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEntries(rows)
}

func (r *ledgerRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.LedgerEntry, error) {
	query := `
		SELECT id, reward_id, user_id, entry_type, account_type, amount, currency,
			description, reference_id, created_at
		FROM ledger_entries
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEntries(rows)
}

func (r *ledgerRepository) ValidateBalance(ctx context.Context, rewardID int) (bool, error) {
	query := `SELECT validate_ledger_balance($1)`
	var isBalanced bool
	err := r.db.QueryRow(ctx, query, rewardID).Scan(&isBalanced)
	return isBalanced, err
}

func (r *ledgerRepository) scanEntries(rows pgx.Rows) ([]*models.LedgerEntry, error) {
	var entries []*models.LedgerEntry
	for rows.Next() {
		entry := &models.LedgerEntry{}
		if err := rows.Scan(
			&entry.ID, &entry.RewardID, &entry.UserID, &entry.EntryType,
			&entry.AccountType, &entry.Amount, &entry.Currency,
			&entry.Description, &entry.ReferenceID, &entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
