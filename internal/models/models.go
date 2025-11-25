package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// StockPrice represents a stock price record
type StockPrice struct {
	ID          int       `json:"id" db:"id"`
	StockSymbol string    `json:"stock_symbol" db:"stock_symbol"`
	Price       float64   `json:"price" db:"price"`
	Currency    string    `json:"currency" db:"currency"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	Source      string    `json:"source" db:"source"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Reward represents a stock reward transaction
type Reward struct {
	ID             int       `json:"id" db:"id"`
	UserID         string    `json:"user_id" db:"user_id"`
	StockSymbol    string    `json:"stock_symbol" db:"stock_symbol"`
	Quantity       float64   `json:"quantity" db:"quantity"`
	EventType      string    `json:"event_type" db:"event_type"`
	EventID        string    `json:"event_id" db:"event_id"`
	EventTimestamp time.Time `json:"event_timestamp" db:"event_timestamp"`
	StockPrice     float64   `json:"stock_price" db:"stock_price"`
	TotalValueINR  float64   `json:"total_value_inr" db:"total_value_inr"`
	BrokerageFee   float64   `json:"brokerage_fee" db:"brokerage_fee"`
	TransactionFee float64   `json:"transaction_fee" db:"transaction_fee"`
	NetValueINR    float64   `json:"net_value_inr" db:"net_value_inr"`
	Status         string    `json:"status" db:"status"`
	Notes          *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// LedgerEntry represents a double-entry ledger record
type LedgerEntry struct {
	ID          int       `json:"id" db:"id"`
	RewardID    int       `json:"reward_id" db:"reward_id"`
	UserID      string    `json:"user_id" db:"user_id"`
	EntryType   string    `json:"entry_type" db:"entry_type"` // DEBIT or CREDIT
	AccountType string    `json:"account_type" db:"account_type"`
	Amount      float64   `json:"amount" db:"amount"`
	Currency    string    `json:"currency" db:"currency"`
	Description *string   `json:"description,omitempty" db:"description"`
	ReferenceID *string   `json:"reference_id,omitempty" db:"reference_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RewardRequest represents an idempotency record for reward requests
type RewardRequest struct {
	ID              int       `json:"id" db:"id"`
	EventID         string    `json:"event_id" db:"event_id"`
	UserID          string    `json:"user_id" db:"user_id"`
	StockSymbol     string    `json:"stock_symbol" db:"stock_symbol"`
	Quantity        float64   `json:"quantity" db:"quantity"`
	RequestPayload  string    `json:"request_payload" db:"request_payload"`   // JSONB
	ResponsePayload *string   `json:"response_payload,omitempty" db:"response_payload"` // JSONB
	Status          string    `json:"status" db:"status"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// CorporateAction represents stock splits, mergers, etc.
type CorporateAction struct {
	ID          int        `json:"id" db:"id"`
	StockSymbol string     `json:"stock_symbol" db:"stock_symbol"`
	ActionType  string     `json:"action_type" db:"action_type"` // SPLIT, REVERSE_SPLIT, MERGER, etc.
	ActionDate  time.Time  `json:"action_date" db:"action_date"`
	RatioFrom   int        `json:"ratio_from" db:"ratio_from"`
	RatioTo     int        `json:"ratio_to" db:"ratio_to"`
	NewSymbol   *string    `json:"new_symbol,omitempty" db:"new_symbol"`
	Description *string    `json:"description,omitempty" db:"description"`
	Applied     bool       `json:"applied" db:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty" db:"applied_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Portfolio represents aggregated user portfolio data
type Portfolio struct {
	UserID            string    `json:"user_id" db:"user_id"`
	StockSymbol       string    `json:"stock_symbol" db:"stock_symbol"`
	TotalQuantity     float64   `json:"total_quantity" db:"total_quantity"`
	AvgPurchasePrice  float64   `json:"avg_purchase_price" db:"avg_purchase_price"`
	TotalInvestedINR  float64   `json:"total_invested_inr" db:"total_invested_inr"`
	TotalFees         float64   `json:"total_fees" db:"total_fees"`
	TransactionCount  int       `json:"transaction_count" db:"transaction_count"`
	FirstRewardDate   time.Time `json:"first_reward_date" db:"first_reward_date"`
	LastRewardDate    time.Time `json:"last_reward_date" db:"last_reward_date"`
	CurrentPrice      float64   `json:"current_price,omitempty"`
	CurrentValueINR   float64   `json:"current_value_inr,omitempty"`
	ProfitLossINR     float64   `json:"profit_loss_inr,omitempty"`
	ProfitLossPercent float64   `json:"profit_loss_percent,omitempty"`
}

// DailyHolding represents daily stock holdings
type DailyHolding struct {
	UserID         string    `json:"user_id" db:"user_id"`
	StockSymbol    string    `json:"stock_symbol" db:"stock_symbol"`
	HoldingDate    time.Time `json:"holding_date" db:"holding_date"`
	DailyQuantity  float64   `json:"daily_quantity" db:"daily_quantity"`
	DailyValueINR  float64   `json:"daily_value_inr" db:"daily_value_inr"`
}

// UserStats represents aggregated user statistics
type UserStats struct {
	UserID               string  `json:"user_id"`
	TotalRewards         int     `json:"total_rewards"`
	TotalStocksQuantity  float64 `json:"total_stocks_quantity"`
	TotalInvestedINR     float64 `json:"total_invested_inr"`
	TotalFeesINR         float64 `json:"total_fees_inr"`
	CurrentPortfolioValue float64 `json:"current_portfolio_value"`
	TotalProfitLossINR   float64 `json:"total_profit_loss_inr"`
	TotalProfitLossPercent float64 `json:"total_profit_loss_percent"`
	UniqueStocks         int     `json:"unique_stocks"`
}
