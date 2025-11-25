package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"stockBackend/internal/models"
	"stockBackend/internal/repository"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// RewardService handles reward operations
type RewardService struct {
	rewardRepo        repository.RewardRepository
	ledgerRepo        repository.LedgerRepository
	rewardRequestRepo repository.RewardRequestRepository
	userRepo          repository.UserRepository
	priceService      *PriceService
	log               *logrus.Logger
	brokeragePercent  float64
	feePercent        float64
}

// RewardRequest represents an incoming reward request
type RewardRequest struct {
	UserID         string    `json:"user_id" binding:"required"`
	StockSymbol    string    `json:"stock_symbol" binding:"required"`
	Quantity       float64   `json:"quantity" binding:"required"`
	EventID        string    `json:"event_id" binding:"required"`
	EventTimestamp time.Time `json:"event_timestamp"`
	EventType      string    `json:"event_type"`
	Notes          string    `json:"notes"`
}

// RewardResponse represents the response after processing a reward
type RewardResponse struct {
	RewardID       int       `json:"reward_id"`
	UserID         string    `json:"user_id"`
	StockSymbol    string    `json:"stock_symbol"`
	Quantity       float64   `json:"quantity"`
	StockPrice     float64   `json:"stock_price"`
	TotalValueINR  float64   `json:"total_value_inr"`
	BrokerageFee   float64   `json:"brokerage_fee"`
	TransactionFee float64   `json:"transaction_fee"`
	NetValueINR    float64   `json:"net_value_inr"`
	EventID        string    `json:"event_id"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	Timestamp      time.Time `json:"timestamp"`
}

// NewRewardService creates a new reward service
func NewRewardService(
	rewardRepo repository.RewardRepository,
	ledgerRepo repository.LedgerRepository,
	rewardRequestRepo repository.RewardRequestRepository,
	userRepo repository.UserRepository,
	priceService *PriceService,
	log *logrus.Logger,
) *RewardService {
	brokeragePercent := 0.1 // Default 0.1%
	feePercent := 0.05      // Default 0.05%

	if bp := os.Getenv("BROKERAGE_PERCENT"); bp != "" {
		if val, err := strconv.ParseFloat(bp, 64); err == nil {
			brokeragePercent = val
		}
	}
	if fp := os.Getenv("TRANSACTION_FEE_PERCENT"); fp != "" {
		if val, err := strconv.ParseFloat(fp, 64); err == nil {
			feePercent = val
		}
	}

	return &RewardService{
		rewardRepo:        rewardRepo,
		ledgerRepo:        ledgerRepo,
		rewardRequestRepo: rewardRequestRepo,
		userRepo:          userRepo,
		priceService:      priceService,
		log:               log,
		brokeragePercent:  brokeragePercent,
		feePercent:        feePercent,
	}
}

// ProcessReward processes a reward request with idempotency
func (rs *RewardService) ProcessReward(ctx context.Context, req *RewardRequest) (*RewardResponse, error) {
	rs.log.Infof("Processing reward request for user %s, event %s", req.UserID, req.EventID)

	// Step 1: Validate request
	if err := rs.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 2: Check idempotency - has this event been processed before?
	existingRequest, err := rs.rewardRequestRepo.GetByEventID(ctx, req.EventID)
	if err == nil && existingRequest != nil {
		rs.log.Warnf("Duplicate request detected for event %s", req.EventID)
		
		// If already completed, return the previous response
		if existingRequest.Status == "COMPLETED" && existingRequest.ResponsePayload != nil {
			var response RewardResponse
			if err := json.Unmarshal([]byte(*existingRequest.ResponsePayload), &response); err == nil {
				response.Message = "Duplicate request - returning previous result"
				return &response, nil
			}
		}
		
		return nil, fmt.Errorf("request already processing or failed")
	}

	// Step 3: Ensure user exists
	userExists, err := rs.userRepo.Exists(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if !userExists {
		return nil, fmt.Errorf("user %s does not exist", req.UserID)
	}

	// Step 4: Create idempotency record
	requestPayload, _ := json.Marshal(req)
	rewardRequest := &models.RewardRequest{
		EventID:        req.EventID,
		UserID:         req.UserID,
		StockSymbol:    req.StockSymbol,
		Quantity:       req.Quantity,
		RequestPayload: string(requestPayload),
		Status:         "PROCESSING",
	}
	
	if err := rs.rewardRequestRepo.Create(ctx, rewardRequest); err != nil {
		return nil, fmt.Errorf("failed to create idempotency record: %w", err)
	}

	// Step 5: Get latest stock price
	stockPrice, err := rs.priceService.GetLatestPrice(ctx, req.StockSymbol)
	if err != nil {
		rs.log.Errorf("Failed to get price for %s: %v", req.StockSymbol, err)
		return nil, fmt.Errorf("failed to get stock price: %w", err)
	}

	// Step 6: Calculate values
	totalValueINR := req.Quantity * stockPrice.Price
	brokerageFee := rs.calculateBrokerage(totalValueINR)
	transactionFee := rs.calculateTransactionFee(totalValueINR)
	netValueINR := totalValueINR - brokerageFee - transactionFee

	// Handle negative rewards (adjustments)
	if req.Quantity < 0 {
		netValueINR = totalValueINR + brokerageFee + transactionFee
	}

	// Step 7: Create reward record
	eventTimestamp := req.EventTimestamp
	if eventTimestamp.IsZero() {
		eventTimestamp = time.Now()
	}

	eventType := req.EventType
	if eventType == "" {
		eventType = "REWARD"
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	reward := &models.Reward{
		UserID:         req.UserID,
		StockSymbol:    req.StockSymbol,
		Quantity:       req.Quantity,
		EventType:      eventType,
		EventID:        req.EventID,
		EventTimestamp: eventTimestamp,
		StockPrice:     stockPrice.Price,
		TotalValueINR:  totalValueINR,
		BrokerageFee:   brokerageFee,
		TransactionFee: transactionFee,
		NetValueINR:    netValueINR,
		Status:         "COMPLETED",
		Notes:          notes,
	}

	createdReward, err := rs.rewardRepo.Create(ctx, reward)
	if err != nil {
		return nil, fmt.Errorf("failed to create reward: %w", err)
	}

	// Step 8: Create ledger entries (double-entry bookkeeping)
	if err := rs.createLedgerEntries(ctx, createdReward); err != nil {
		rs.log.Errorf("Failed to create ledger entries: %v", err)
		// Don't fail the reward, but log the error
	}

	// Step 9: Mark request as completed
	response := &RewardResponse{
		RewardID:       createdReward.ID,
		UserID:         createdReward.UserID,
		StockSymbol:    createdReward.StockSymbol,
		Quantity:       createdReward.Quantity,
		StockPrice:     createdReward.StockPrice,
		TotalValueINR:  createdReward.TotalValueINR,
		BrokerageFee:   createdReward.BrokerageFee,
		TransactionFee: createdReward.TransactionFee,
		NetValueINR:    createdReward.NetValueINR,
		EventID:        createdReward.EventID,
		Status:         "SUCCESS",
		Message:        "Reward processed successfully",
		Timestamp:      time.Now(),
	}

	responsePayload, _ := json.Marshal(response)
	responseStr := string(responsePayload)
	if err := rs.rewardRequestRepo.MarkProcessed(ctx, req.EventID, responseStr); err != nil {
		rs.log.Errorf("Failed to mark request as processed: %v", err)
	}

	rs.log.Infof("Successfully processed reward %d for user %s", createdReward.ID, req.UserID)
	return response, nil
}

// validateRequest validates the reward request
func (rs *RewardService) validateRequest(req *RewardRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.StockSymbol == "" {
		return fmt.Errorf("stock_symbol is required")
	}
	if req.Quantity == 0 {
		return fmt.Errorf("quantity cannot be zero")
	}
	if req.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	return nil
}

// calculateBrokerage calculates brokerage fee
func (rs *RewardService) calculateBrokerage(totalValue float64) float64 {
	fee := math.Abs(totalValue) * (rs.brokeragePercent / 100.0)
	return rs.roundToTwoDecimals(fee)
}

// calculateTransactionFee calculates transaction fee
func (rs *RewardService) calculateTransactionFee(totalValue float64) float64 {
	fee := math.Abs(totalValue) * (rs.feePercent / 100.0)
	return rs.roundToTwoDecimals(fee)
}

// roundToTwoDecimals rounds a float to 2 decimal places
func (rs *RewardService) roundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// createLedgerEntries creates double-entry ledger entries for a reward
func (rs *RewardService) createLedgerEntries(ctx context.Context, reward *models.Reward) error {
	entries := make([]*models.LedgerEntry, 0)

	// For positive rewards (receiving stocks)
	if reward.Quantity > 0 {
		// DEBIT: Stock Asset Account (increase in assets)
		stockAssetDesc := fmt.Sprintf("Stock reward: %s x %.6f @ %.2f INR", 
			reward.StockSymbol, reward.Quantity, reward.StockPrice)
		entries = append(entries, &models.LedgerEntry{
			RewardID:    reward.ID,
			UserID:      reward.UserID,
			EntryType:   "DEBIT",
			AccountType: "STOCK_ASSET",
			Amount:      reward.TotalValueINR,
			Currency:    "INR",
			Description: &stockAssetDesc,
			ReferenceID: &reward.EventID,
		})

		// CREDIT: Reward Income Account (source of the asset)
		rewardIncomeDesc := fmt.Sprintf("Reward income for event %s", reward.EventID)
		entries = append(entries, &models.LedgerEntry{
			RewardID:    reward.ID,
			UserID:      reward.UserID,
			EntryType:   "CREDIT",
			AccountType: "REWARD_INCOME",
			Amount:      reward.TotalValueINR,
			Currency:    "INR",
			Description: &rewardIncomeDesc,
			ReferenceID: &reward.EventID,
		})

		// DEBIT: Brokerage Expense
		if reward.BrokerageFee > 0 {
			brokerageDesc := fmt.Sprintf("Brokerage fee for %s", reward.EventID)
			entries = append(entries, &models.LedgerEntry{
				RewardID:    reward.ID,
				UserID:      reward.UserID,
				EntryType:   "DEBIT",
				AccountType: "BROKERAGE_EXPENSE",
				Amount:      reward.BrokerageFee,
				Currency:    "INR",
				Description: &brokerageDesc,
				ReferenceID: &reward.EventID,
			})

			// CREDIT: Cash (payment of brokerage)
			entries = append(entries, &models.LedgerEntry{
				RewardID:    reward.ID,
				UserID:      reward.UserID,
				EntryType:   "CREDIT",
				AccountType: "CASH",
				Amount:      reward.BrokerageFee,
				Currency:    "INR",
				Description: &brokerageDesc,
				ReferenceID: &reward.EventID,
			})
		}

		// DEBIT: Transaction Fee Expense
		if reward.TransactionFee > 0 {
			feeDesc := fmt.Sprintf("Transaction fee for %s", reward.EventID)
			entries = append(entries, &models.LedgerEntry{
				RewardID:    reward.ID,
				UserID:      reward.UserID,
				EntryType:   "DEBIT",
				AccountType: "FEE_EXPENSE",
				Amount:      reward.TransactionFee,
				Currency:    "INR",
				Description: &feeDesc,
				ReferenceID: &reward.EventID,
			})

			// CREDIT: Cash (payment of fee)
			entries = append(entries, &models.LedgerEntry{
				RewardID:    reward.ID,
				UserID:      reward.UserID,
				EntryType:   "CREDIT",
				AccountType: "CASH",
				Amount:      reward.TransactionFee,
				Currency:    "INR",
				Description: &feeDesc,
				ReferenceID: &reward.EventID,
			})
		}
	} else {
		// For negative rewards (adjustments/deductions)
		// CREDIT: Stock Asset Account (decrease in assets)
		stockAssetDesc := fmt.Sprintf("Stock adjustment: %s x %.6f @ %.2f INR",
			reward.StockSymbol, reward.Quantity, reward.StockPrice)
		entries = append(entries, &models.LedgerEntry{
			RewardID:    reward.ID,
			UserID:      reward.UserID,
			EntryType:   "CREDIT",
			AccountType: "STOCK_ASSET",
			Amount:      math.Abs(reward.TotalValueINR),
			Currency:    "INR",
			Description: &stockAssetDesc,
			ReferenceID: &reward.EventID,
		})

		// DEBIT: Adjustment Expense Account
		adjustmentDesc := fmt.Sprintf("Stock adjustment for event %s", reward.EventID)
		entries = append(entries, &models.LedgerEntry{
			RewardID:    reward.ID,
			UserID:      reward.UserID,
			EntryType:   "DEBIT",
			AccountType: "ADJUSTMENT_EXPENSE",
			Amount:      math.Abs(reward.TotalValueINR),
			Currency:    "INR",
			Description: &adjustmentDesc,
			ReferenceID: &reward.EventID,
		})
	}

	// Bulk create all ledger entries
	return rs.ledgerRepo.BulkCreate(ctx, entries)
}

// GetRewardByEventID retrieves a reward by event ID
func (rs *RewardService) GetRewardByEventID(ctx context.Context, eventID string) (*models.Reward, error) {
	return rs.rewardRepo.GetByEventID(ctx, eventID)
}

// GetUserRewards retrieves rewards for a user
func (rs *RewardService) GetUserRewards(ctx context.Context, userID string, limit, offset int) ([]*models.Reward, error) {
	return rs.rewardRepo.GetByUserID(ctx, userID, limit, offset)
}
