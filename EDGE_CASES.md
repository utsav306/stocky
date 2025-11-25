# Edge Cases and Design Decisions

## 1. Duplicate Reward Events (Idempotency)

### Problem
Multiple requests with the same `event_id` could create duplicate rewards.

### Solution
- **reward_requests table**: Tracks all incoming requests
- **Unique constraint** on `event_id`
- **Status tracking**: PROCESSING → COMPLETED
- **Response caching**: Store response payload for duplicate requests

### Implementation
```go
// Check if event already processed
existingRequest, err := rewardRequestRepo.GetByEventID(ctx, req.EventID)
if err == nil && existingRequest.Status == "COMPLETED" {
    // Return cached response
    return previousResponse, nil
}
```

### Test Case
```bash
# First request - creates reward
curl -X POST /api/v1/reward -d '{"event_id": "EVT-001", ...}'

# Second request - returns cached result
curl -X POST /api/v1/reward -d '{"event_id": "EVT-001", ...}'
```

---

## 2. Stock Splits and Mergers

### Design
**corporate_actions table** tracks all corporate events:
- Stock splits (1:2, 2:1, etc.)
- Reverse splits
- Mergers
- Dividends
- Bonus issues

### Fields
- `action_type`: SPLIT, REVERSE_SPLIT, MERGER, etc.
- `ratio_from` / `ratio_to`: Split ratio
- `action_date`: When to apply
- `applied`: Boolean flag
- `new_symbol`: For mergers

### Processing Logic
```sql
-- Example: 1:2 stock split for AAPL
INSERT INTO corporate_actions (
    stock_symbol, action_type, action_date,
    ratio_from, ratio_to, description
) VALUES (
    'AAPL', 'SPLIT', '2024-06-01',
    1, 2, 'Stock split 1:2'
);

-- Update all existing rewards
UPDATE rewards
SET quantity = quantity * 2,
    stock_price = stock_price / 2
WHERE stock_symbol = 'AAPL'
  AND event_timestamp < '2024-06-01';
```

### Future Enhancement
Background job to automatically apply pending corporate actions.

---

## 3. Price Service Downtime

### Fallback Strategy

1. **Primary**: Fetch from database
2. **Fallback**: Generate mock price
3. **Stale Price**: Use last known price with warning

### Implementation
```go
func GetLatestPrice(ctx context.Context, symbol string) (*StockPrice, error) {
    // Try database first
    price, err := priceRepo.GetLatest(ctx, symbol)
    if err != nil {
        // Fallback: Generate new price
        return UpdateSinglePrice(ctx, symbol)
    }
    
    // Check if price is stale (> 24 hours)
    if time.Since(price.Timestamp) > 24*time.Hour {
        log.Warn("Using stale price")
    }
    
    return price, nil
}
```

### Stale Price Handling
- **Warning logged** if price > 24 hours old
- **Auto-refresh** attempted in background
- **Graceful degradation**: Use stale price rather than fail

---

## 4. Rounding Rules

### Precision Standards
- **Quantities**: 6 decimals (0.000001)
- **Prices**: 4 decimals (0.0001)
- **INR values**: 2 decimals (0.01)
- **Percentages**: 2 decimals (0.01%)

### Rounding Method
```go
func roundToTwoDecimals(value float64) float64 {
    return math.Round(value*100) / 100
}
```

### Application
- Brokerage fee: `Round(totalValue * 0.1 / 100, 2)`
- Transaction fee: `Round(totalValue * 0.05 / 100, 2)`
- Net value: `totalValue - brokerageFee - transactionFee`

### Edge Case: Rounding Errors
For very small values, ensure minimum fee:
```go
if brokerageFee < 0.01 && totalValue > 0 {
    brokerageFee = 0.01 // Minimum 1 paisa
}
```

---

## 5. Negative Rewards (Adjustments)

### Use Cases
- Correction of overpayment
- Reversal of erroneous reward
- Penalty deduction
- Stock buyback

### Implementation
```go
if reward.Quantity < 0 {
    // Reverse ledger entries
    // CREDIT: Stock Asset (decrease)
    // DEBIT: Adjustment Expense
}
```

### Validation
- **Allow negative quantities**
- **Prevent negative balance**: Check user has enough stocks
- **Separate event type**: "ADJUSTMENT" vs "REWARD"

### Example
```json
{
  "user_id": "USR001",
  "stock_symbol": "AAPL",
  "quantity": -5.0,
  "event_id": "EVT-ADJ-001",
  "event_type": "ADJUSTMENT",
  "notes": "Correction for duplicate reward"
}
```

---

## 6. Concurrent Requests

### Problem
Multiple simultaneous requests for same user/stock.

### Solution
- **Database transactions**: ACID compliance
- **Row-level locking**: PostgreSQL handles automatically
- **Unique constraints**: Prevent duplicate event_ids

### Transaction Example
```go
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

// All operations in transaction
createReward(tx, ...)
createLedgerEntries(tx, ...)
updateIdempotencyRecord(tx, ...)

tx.Commit(ctx)
```

---

## 7. Large Data Volumes

### Optimization Strategies

#### Indexes
```sql
CREATE INDEX idx_rewards_user_timestamp 
    ON rewards(user_id, event_timestamp DESC);
CREATE INDEX idx_stock_prices_symbol_timestamp 
    ON stock_prices(stock_symbol, timestamp DESC);
```

#### Partitioning
```sql
-- Partition rewards by month
CREATE TABLE rewards_2024_01 PARTITION OF rewards
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

#### Batch Operations
```go
// Bulk insert prices
priceRepo.BulkCreate(ctx, prices)

// Bulk create ledger entries
ledgerRepo.BulkCreate(ctx, entries)
```

---

## 8. Missing User

### Validation
```go
userExists, err := userRepo.Exists(ctx, req.UserID)
if !userExists {
    return fmt.Errorf("user %s does not exist", req.UserID)
}
```

### Auto-Creation (Optional)
```go
if !userExists {
    // Create user automatically
    user := &User{
        UserID: req.UserID,
        Name:   "Auto-created",
        Email:  fmt.Sprintf("%s@example.com", req.UserID),
    }
    userRepo.Create(ctx, user)
}
```

---

## 9. Invalid Stock Symbol

### Validation
```go
supportedStocks := priceService.GetSupportedStocks()
if !contains(supportedStocks, req.StockSymbol) {
    // Auto-add new symbol
    priceService.AddStock(req.StockSymbol)
}
```

### Dynamic Stock Addition
- New symbols automatically added to tracking
- Price generated on first use
- Logged for admin review

---

## 10. Database Connection Loss

### Resilience
- **Connection pooling**: Min 2, Max 10 connections
- **Health checks**: Periodic ping
- **Retry logic**: Exponential backoff
- **Circuit breaker**: Stop requests if DB down

### Implementation
```go
config.MaxConns = 10
config.MinConns = 2
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
```

---

## 11. Time Zone Handling

### Standard
- All timestamps in **UTC**
- ISO 8601 format
- Database stores `TIMESTAMP WITH TIME ZONE`

### Conversion
```go
eventTimestamp := req.EventTimestamp
if eventTimestamp.IsZero() {
    eventTimestamp = time.Now().UTC()
}
```

---

## 12. Decimal Precision in Database

### PostgreSQL Types
```sql
quantity DECIMAL(15, 6)  -- 999999999.999999
price DECIMAL(15, 4)     -- 99999999999.9999
amount DECIMAL(15, 2)    -- 9999999999999.99
```

### Go Handling
Use `float64` with rounding for display:
```go
type Reward struct {
    Quantity float64 `json:"quantity"`
    Price    float64 `json:"stock_price"`
}
```

---

## Summary

All edge cases are handled with:
1. **Validation** at API layer
2. **Database constraints** for data integrity
3. **Graceful fallbacks** for service failures
4. **Comprehensive logging** for debugging
5. **Transaction support** for consistency
