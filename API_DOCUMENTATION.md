# API Documentation

## Overview

This document provides detailed information about all API endpoints, request/response formats, and example usage.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently, the API does not require authentication. In production, implement JWT or API key authentication.

## Response Format

All responses follow this structure:

### Success Response
```json
{
  "success": true,
  "data": { ... },
  "message": "Optional message"
}
```

### Error Response
```json
{
  "error": "Error type",
  "message": "Detailed error message"
}
```

## Endpoints

### 1. Health Check

**GET** `/health`

Check service health and database connectivity.

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z",
  "database": "healthy",
  "service": "stock-reward-backend",
  "version": "1.0.0"
}
```

---

### 2. Create Reward

**POST** `/api/v1/reward`

Process a new stock reward with automatic price calculation and ledger entries.

**Request Body:**
```json
{
  "user_id": "USR001",
  "stock_symbol": "AAPL",
  "quantity": 10.5,
  "event_id": "EVT-2024-001",
  "event_timestamp": "2024-01-15T10:30:00Z",
  "event_type": "REWARD",
  "notes": "Performance bonus Q1 2024"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "reward_id": 123,
    "user_id": "USR001",
    "stock_symbol": "AAPL",
    "quantity": 10.5,
    "stock_price": 175.50,
    "total_value_inr": 1842.75,
    "brokerage_fee": 1.84,
    "transaction_fee": 0.92,
    "net_value_inr": 1839.99,
    "event_id": "EVT-2024-001",
    "status": "SUCCESS",
    "message": "Reward processed successfully",
    "timestamp": "2024-01-15T10:30:05Z"
  }
}
```

**Idempotency:**
- Same `event_id` returns cached response
- Prevents duplicate processing

**Negative Rewards:**
```json
{
  "user_id": "USR001",
  "stock_symbol": "AAPL",
  "quantity": -2.5,
  "event_id": "EVT-ADJ-001",
  "event_type": "ADJUSTMENT"
}
```

---

### 3. Get Today's Stocks

**GET** `/api/v1/today-stocks/:userId`

Retrieve all stock rewards received today.

**Response:**
```json
{
  "user_id": "USR001",
  "date": "today",
  "rewards": [
    {
      "id": 123,
      "stock_symbol": "AAPL",
      "quantity": 10.5,
      "stock_price": 175.50,
      "total_value_inr": 1842.75,
      "event_timestamp": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "total_quantity": 10.5,
  "total_inr": 1842.75
}
```

---

### 4. Get Historical INR

**GET** `/api/v1/historical-inr/:userId?start_date=2024-01-01&end_date=2024-12-31`

Retrieve historical INR values for a date range.

**Query Parameters:**
- `start_date` (optional): Start date (YYYY-MM-DD), default: 1 month ago
- `end_date` (optional): End date (YYYY-MM-DD), default: today

**Response:**
```json
{
  "user_id": "USR001",
  "start_date": "2024-01-01",
  "end_date": "2024-12-31",
  "rewards": [ ... ],
  "count": 45,
  "total_quantity": 250.5,
  "total_inr": 125000.50,
  "total_fees": 125.00,
  "net_inr": 124875.50
}
```

---

### 5. Get User Statistics

**GET** `/api/v1/stats/:userId`

Get aggregated statistics for a user.

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": "USR001",
    "total_rewards": 45,
    "total_stocks_quantity": 250.5,
    "total_invested_inr": 125000.50,
    "total_fees_inr": 125.00,
    "current_portfolio_value": 135000.75,
    "total_profit_loss_inr": 10000.25,
    "total_profit_loss_percent": 8.0,
    "unique_stocks": 5
  }
}
```

---

### 6. Get User Portfolio

**GET** `/api/v1/portfolio/:userId`

Get complete portfolio with current valuations and profit/loss.

**Response:**
```json
{
  "user_id": "USR001",
  "portfolio": [
    {
      "stock_symbol": "AAPL",
      "total_quantity": 50.5,
      "avg_purchase_price": 170.25,
      "total_invested_inr": 8597.63,
      "total_fees": 8.60,
      "transaction_count": 5,
      "current_price": 175.50,
      "current_value_inr": 8862.75,
      "profit_loss_inr": 265.12,
      "profit_loss_percent": 3.08,
      "first_reward_date": "2024-01-01T00:00:00Z",
      "last_reward_date": "2024-01-15T10:30:00Z"
    }
  ],
  "holdings_count": 5,
  "total_invested_inr": 125000.50,
  "total_current_value": 135000.75,
  "total_profit_loss": 10000.25,
  "profit_loss_percent": 8.0
}
```

---

### 7. Price Management

#### Trigger Price Update (All Stocks)

**POST** `/api/v1/prices/update`

Manually trigger price update for all stocks.

**Response:**
```json
{
  "message": "Prices updated successfully",
  "stocks": ["AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", ...]
}
```

#### Get Latest Price

**GET** `/api/v1/prices/:symbol`

Get the latest price for a stock.

**Response:**
```json
{
  "data": {
    "id": 456,
    "stock_symbol": "AAPL",
    "price": 175.50,
    "currency": "INR",
    "timestamp": "2024-01-15T10:00:00Z",
    "source": "MOCK_SERVICE"
  }
}
```

#### Get Price History

**GET** `/api/v1/prices/:symbol/history?limit=10`

Get historical prices for a stock.

**Response:**
```json
{
  "data": [
    {
      "stock_symbol": "AAPL",
      "price": 175.50,
      "timestamp": "2024-01-15T10:00:00Z"
    },
    ...
  ],
  "count": 10
}
```

---

## Error Codes

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request - Invalid input |
| 404 | Not Found |
| 500 | Internal Server Error |
| 503 | Service Unavailable - Database down |

## Rate Limiting

Currently no rate limiting. In production, implement:
- 100 requests per minute per IP
- 1000 requests per hour per user

## Pagination

Endpoints supporting pagination use:
- `limit`: Number of results (default: 10, max: 100)
- `offset`: Number of results to skip (default: 0)

Example:
```
GET /api/v1/rewards/USR001?limit=20&offset=40
```

## Date Formats

All dates use ISO 8601 format:
- `2024-01-15T10:30:00Z` (with time)
- `2024-01-15` (date only)

## Decimal Precision

- Quantities: 6 decimal places
- Prices: 4 decimal places
- INR values: 2 decimal places
- Percentages: 2 decimal places
