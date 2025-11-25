# Stock Reward Backend

A production-ready Go backend service for managing stock rewards with double-entry ledger accounting, idempotency, and comprehensive analytics.

## 🚀 Features

- **Stock Reward Management**: Process stock rewards with automatic price calculation
- **Idempotency**: Prevent duplicate reward processing using event IDs
- **Double-Entry Ledger**: Complete accounting system for all transactions
- **Mock Price Service**: Automated stock price updates with configurable intervals
- **Portfolio Analytics**: Real-time portfolio valuation and profit/loss tracking
- **Historical Data**: Track rewards and INR values over time
- **Corporate Actions**: Support for stock splits, mergers, and other events
- **Production-Ready**: Structured logging, graceful shutdown, connection pooling

## 📋 Prerequisites

- Go 1.23.3 or higher
- PostgreSQL 14+ (or Supabase account)
- Git

## 🛠️ Installation

### 1. Clone the repository

```bash
git clone <repository-url>
cd stockBackend
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Set up your database

#### Option A: Using Supabase (Recommended)

1. Create a free account at [supabase.com](https://supabase.com)
2. Create a new project
3. Go to **Settings** → **Database** → **Connection Pooling**
4. Select **Session Mode** and copy the connection string
5. Update your `.env` file:

```bash
cp .env.example .env
# Edit .env and add your Supabase connection string:
DATABASE_URL=postgresql://postgres.xxxxx:your_password@aws-0-region.pooler.supabase.com:6543/postgres
```

6. Run migrations in Supabase SQL Editor:
   - Open **SQL Editor** → **New Query**
   - Copy and paste contents of `migrations/001_create_initial_schema.sql`
   - Click **Run**
   - Repeat for `migrations/002_create_views_and_functions.sql`

#### Option B: Using Local PostgreSQL

```bash
# Create database
createdb assignment

# Run migrations
psql -d assignment -f migrations/001_create_initial_schema.sql
psql -d assignment -f migrations/002_create_views_and_functions.sql

# Configure .env with local settings
cp .env.example .env
# Edit .env and set DB_HOST=localhost, DB_PASSWORD=your_password, etc.
```

### 4. Run the application

```bash
go run cmd/main.go
```

The server will start on `http://localhost:8080`

## 🗄️ Database Schema

### Tables

1. **users** - User information
2. **stock_prices** - Historical stock prices
3. **rewards** - Stock reward transactions
4. **ledger_entries** - Double-entry ledger records
5. **reward_requests** - Idempotency tracking
6. **corporate_actions** - Stock splits, mergers, etc.

### Entity Relationship Diagram

```
┌─────────────┐         ┌──────────────┐         ┌─────────────────┐
│   users     │────────<│   rewards    │>────────│ ledger_entries  │
└─────────────┘         └──────────────┘         └─────────────────┘
                               │
                               │
                        ┌──────┴───────┐
                        │              │
                ┌───────▼──────┐ ┌────▼──────────┐
                │stock_prices  │ │reward_requests│
                └──────────────┘ └───────────────┘
```

## 📡 API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Endpoints

#### Health Check
```http
GET /health
```

#### Price Management

**Trigger Price Update (All Stocks)**
```http
POST /api/v1/prices/update
```

**Update Single Stock Price**
```http
POST /api/v1/prices/update/:symbol
```

**Get Latest Price**
```http
GET /api/v1/prices/:symbol
```

**Get Price History**
```http
GET /api/v1/prices/:symbol/history?limit=10
```

**Get Supported Stocks**
```http
GET /api/v1/prices/stocks
```

#### Reward Management

**Create Reward**
```http
POST /api/v1/reward
Content-Type: application/json

{
  "user_id": "USR001",
  "stock_symbol": "AAPL",
  "quantity": 10.5,
  "event_id": "EVT-2024-001",
  "event_timestamp": "2024-01-15T10:30:00Z",
  "event_type": "REWARD",
  "notes": "Performance bonus"
}
```

**Get Reward by Event ID**
```http
GET /api/v1/reward/:eventId
```

**Get User Rewards**
```http
GET /api/v1/rewards/:userId?limit=10&offset=0
```

#### Analytics & Portfolio

**Get Today's Stocks**
```http
GET /api/v1/today-stocks/:userId
```

**Get Historical INR Values**
```http
GET /api/v1/historical-inr/:userId?start_date=2024-01-01&end_date=2024-12-31
```

**Get User Statistics**
```http
GET /api/v1/stats/:userId
```

**Get User Portfolio**
```http
GET /api/v1/portfolio/:userId
```

**Get Daily Holdings**
```http
GET /api/v1/holdings/:userId?date=2024-01-15
```

## 🔧 Configuration

### Environment Variables

#### Database Configuration

The application supports two ways to configure the database connection:

**Option 1: Connection String (Recommended for Supabase)**

| Variable | Description | Example |
|----------|-------------|----------|
| `DATABASE_URL` | Full PostgreSQL connection string | `postgresql://user:pass@host:port/db` |

**Option 2: Individual Parameters (For Local Development)**

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_USER` | Database user | postgres |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | assignment |
| `DB_SSLMODE` | SSL mode (disable/require) | disable |

> **Note:** If `DATABASE_URL` is set, it takes priority over individual DB_* variables.

#### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8080 |
| `GIN_MODE` | Gin mode (debug/release) | debug |
| `LOG_LEVEL` | Log level (info/debug/error) | info |
| `LOG_FORMAT` | Log format (json/text) | json |

#### Price Service Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PRICE_UPDATE_INTERVAL_HOURS` | Price update frequency | 1 |
| `MOCK_PRICE_MIN` | Minimum mock price | 100 |
| `MOCK_PRICE_MAX` | Maximum mock price | 5000 |

#### Fee Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BROKERAGE_PERCENT` | Brokerage fee % | 0.1 |
| `TRANSACTION_FEE_PERCENT` | Transaction fee % | 0.05 |

## 📝 Example Requests

### Create a Reward

```bash
curl -X POST http://localhost:8080/api/v1/reward \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "USR001",
    "stock_symbol": "AAPL",
    "quantity": 10,
    "event_id": "EVT-001",
    "event_type": "REWARD"
  }'
```

### Get Today's Stocks

```bash
curl http://localhost:8080/api/v1/today-stocks/USR001
```

### Get Portfolio

```bash
curl http://localhost:8080/api/v1/portfolio/USR001
```

### Trigger Price Update

```bash
curl -X POST http://localhost:8080/api/v1/prices/update
```

## 🎯 Key Features Explained

### Idempotency

The system prevents duplicate reward processing using the `event_id` field. If the same `event_id` is submitted multiple times, the system returns the original response without creating duplicate records.

### Double-Entry Ledger

Every reward transaction creates balanced ledger entries:
- **DEBIT**: Stock Asset (increase in holdings)
- **CREDIT**: Reward Income (source of asset)
- **DEBIT**: Brokerage Expense
- **CREDIT**: Cash (payment of fees)

### Fee Calculation

- **Brokerage Fee**: Configurable percentage of total value
- **Transaction Fee**: Configurable percentage of total value
- **Net Value**: Total value minus all fees

### Negative Rewards

The system supports negative quantities for adjustments or corrections, properly reversing ledger entries.

### Price Service

- Automatic hourly price updates (configurable)
- Manual trigger endpoints
- Stale price fallback (generates new price if none exists)
- Batch price retrieval for efficiency

## 🏗️ Project Structure

```
stockBackend/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── controllers/         # HTTP handlers
│   │   ├── price_controller.go
│   │   ├── reward_controller.go
│   │   └── portfolio_controller.go
│   ├── services/            # Business logic
│   │   ├── price_service.go
│   │   ├── reward_service.go
│   │   └── portfolio_service.go
│   ├── repository/          # Data access layer
│   │   ├── interfaces.go
│   │   ├── user_repository.go
│   │   ├── stock_price_repository.go
│   │   ├── reward_repository.go
│   │   ├── ledger_repository.go
│   │   ├── reward_request_repository.go
│   │   ├── corporate_action_repository.go
│   │   └── portfolio_repository.go
│   ├── models/              # Data models
│   │   └── models.go
│   └── db/                  # Database utilities
│       └── db.go
├── migrations/              # SQL migrations
│   ├── 001_create_initial_schema.sql
│   └── 002_create_views_and_functions.sql
├── postman/                 # Postman collection
│   └── stock-reward-backend.postman_collection.json
├── .env.example             # Environment template
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## 🔍 Edge Cases Handled

1. **Duplicate Requests**: Idempotency via `event_id`
2. **Missing Prices**: Auto-generate if not available
3. **Negative Rewards**: Support for adjustments/corrections
4. **Concurrent Requests**: Database transactions ensure consistency
5. **Price Service Downtime**: Fallback to price generation
6. **Rounding**: All monetary values rounded to 2 decimals
7. **Stock Splits/Mergers**: Corporate actions table for tracking

## 📈 Scaling Considerations

### Horizontal Scaling
- Stateless design allows multiple instances
- Connection pooling for efficient database usage
- Consider Redis for distributed caching

### Database Optimization
- Indexes on frequently queried columns
- Partitioning for large tables (rewards, ledger_entries)
- Read replicas for analytics queries

### Performance
- Batch operations for bulk price updates
- Materialized views for complex aggregations
- Background jobs for heavy computations

### Monitoring
- Structured logging (JSON format)
- Health check endpoint
- Database connection pool metrics

## 🧪 Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/services/...
```

## 📦 Deployment

### Docker (Optional)

```dockerfile
FROM golang:1.23.3-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
EXPOSE 8080
CMD ["./main"]
```

### Build for Production

```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📄 License

This project is licensed under the MIT License.

## 👥 Authors

- Backend Team

## 📞 Support

For issues and questions, please open an issue on GitHub.
