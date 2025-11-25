package repository

import (
	"context"
	"fmt"
	"stockBackend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type stockPriceRepository struct {
	db *pgxpool.Pool
}

// NewStockPriceRepository creates a new stock price repository
func NewStockPriceRepository(db *pgxpool.Pool) StockPriceRepository {
	return &stockPriceRepository{db: db}
}

func (r *stockPriceRepository) Create(ctx context.Context, price *models.StockPrice) error {
	query := `
		INSERT INTO stock_prices (stock_symbol, price, currency, source, timestamp)
		VALUES ($1, $2, $3, $4, COALESCE($5, CURRENT_TIMESTAMP))
		RETURNING id, timestamp, created_at
	`
	var timestamp *string
	if !price.Timestamp.IsZero() {
		ts := price.Timestamp.Format("2006-01-02 15:04:05")
		timestamp = &ts
	}
	
	return r.db.QueryRow(ctx, query,
		price.StockSymbol, price.Price, price.Currency, price.Source, timestamp,
	).Scan(&price.ID, &price.Timestamp, &price.CreatedAt)
}

func (r *stockPriceRepository) GetLatest(ctx context.Context, stockSymbol string) (*models.StockPrice, error) {
	query := `
		SELECT id, stock_symbol, price, currency, timestamp, source, created_at
		FROM stock_prices
		WHERE stock_symbol = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	price := &models.StockPrice{}
	err := r.db.QueryRow(ctx, query, stockSymbol).Scan(
		&price.ID, &price.StockSymbol, &price.Price, &price.Currency,
		&price.Timestamp, &price.Source, &price.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("stock price not found: %w", err)
	}
	return price, nil
}

func (r *stockPriceRepository) GetLatestBatch(ctx context.Context, stockSymbols []string) (map[string]*models.StockPrice, error) {
	query := `
		SELECT DISTINCT ON (stock_symbol) 
			id, stock_symbol, price, currency, timestamp, source, created_at
		FROM stock_prices
		WHERE stock_symbol = ANY($1)
		ORDER BY stock_symbol, timestamp DESC
	`
	rows, err := r.db.Query(ctx, query, stockSymbols)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make(map[string]*models.StockPrice)
	for rows.Next() {
		price := &models.StockPrice{}
		if err := rows.Scan(
			&price.ID, &price.StockSymbol, &price.Price, &price.Currency,
			&price.Timestamp, &price.Source, &price.CreatedAt,
		); err != nil {
			return nil, err
		}
		prices[price.StockSymbol] = price
	}
	return prices, rows.Err()
}

func (r *stockPriceRepository) GetHistory(ctx context.Context, stockSymbol string, limit int) ([]*models.StockPrice, error) {
	query := `
		SELECT id, stock_symbol, price, currency, timestamp, source, created_at
		FROM stock_prices
		WHERE stock_symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, stockSymbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*models.StockPrice
	for rows.Next() {
		price := &models.StockPrice{}
		if err := rows.Scan(
			&price.ID, &price.StockSymbol, &price.Price, &price.Currency,
			&price.Timestamp, &price.Source, &price.CreatedAt,
		); err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}
	return prices, rows.Err()
}

func (r *stockPriceRepository) GetByTimeRange(ctx context.Context, stockSymbol string, start, end string) ([]*models.StockPrice, error) {
	query := `
		SELECT id, stock_symbol, price, currency, timestamp, source, created_at
		FROM stock_prices
		WHERE stock_symbol = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
	`
	rows, err := r.db.Query(ctx, query, stockSymbol, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*models.StockPrice
	for rows.Next() {
		price := &models.StockPrice{}
		if err := rows.Scan(
			&price.ID, &price.StockSymbol, &price.Price, &price.Currency,
			&price.Timestamp, &price.Source, &price.CreatedAt,
		); err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}
	return prices, rows.Err()
}

func (r *stockPriceRepository) BulkCreate(ctx context.Context, prices []*models.StockPrice) error {
	if len(prices) == 0 {
		return nil
	}

	query := `
		INSERT INTO stock_prices (stock_symbol, price, currency, source, timestamp)
		VALUES ($1, $2, $3, $4, COALESCE($5, CURRENT_TIMESTAMP))
	`
	
	batch := &pgx.Batch{}
	for _, price := range prices {
		var timestamp *string
		if !price.Timestamp.IsZero() {
			ts := price.Timestamp.Format("2006-01-02 15:04:05")
			timestamp = &ts
		}
		batch.Queue(query, price.StockSymbol, price.Price, price.Currency, price.Source, timestamp)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range prices {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert price: %w", err)
		}
	}

	return nil
}
