-- Initial database schema for stock reward backend
-- This sets up all the tables we need for tracking users, stock prices, rewards, and accounting
-- Run this first before the views/functions migration

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_user_id ON users(user_id);
CREATE INDEX idx_users_email ON users(email);

COMMENT ON TABLE users IS 'Stores user information';
COMMENT ON COLUMN users.user_id IS 'External user identifier';


CREATE TABLE IF NOT EXISTS stock_prices (
    id SERIAL PRIMARY KEY,
    stock_symbol VARCHAR(20) NOT NULL,
    price DECIMAL(15, 4) NOT NULL CHECK (price >= 0),
    currency VARCHAR(3) DEFAULT 'INR',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(50) DEFAULT 'MOCK_SERVICE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stock_prices_symbol_timestamp ON stock_prices(stock_symbol, timestamp DESC);
CREATE INDEX idx_stock_prices_timestamp ON stock_prices(timestamp DESC);

COMMENT ON TABLE stock_prices IS 'Stores historical stock prices';
COMMENT ON COLUMN stock_prices.price IS 'Stock price in specified currency';


CREATE TABLE IF NOT EXISTS rewards (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    stock_symbol VARCHAR(20) NOT NULL,
    quantity DECIMAL(15, 6) NOT NULL CHECK (quantity != 0),
    event_type VARCHAR(50) NOT NULL DEFAULT 'REWARD',
    event_id VARCHAR(100) UNIQUE NOT NULL,
    event_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    stock_price DECIMAL(15, 4) NOT NULL CHECK (stock_price >= 0),
    total_value_inr DECIMAL(15, 2) NOT NULL,
    brokerage_fee DECIMAL(15, 2) DEFAULT 0,
    transaction_fee DECIMAL(15, 2) DEFAULT 0,
    net_value_inr DECIMAL(15, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'COMPLETED',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_rewards_user_id ON rewards(user_id);
CREATE INDEX idx_rewards_event_id ON rewards(event_id);
CREATE INDEX idx_rewards_user_timestamp ON rewards(user_id, event_timestamp DESC);
CREATE INDEX idx_rewards_stock_symbol ON rewards(stock_symbol);

COMMENT ON TABLE rewards IS 'Stores all reward events (positive and negative)';
COMMENT ON COLUMN rewards.quantity IS 'Number of stocks rewarded (can be negative for adjustments)';
COMMENT ON COLUMN rewards.event_id IS 'Unique identifier for idempotency';

CREATE TABLE IF NOT EXISTS ledger_entries (
    id SERIAL PRIMARY KEY,
    reward_id INTEGER NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    entry_type VARCHAR(20) NOT NULL CHECK (entry_type IN ('DEBIT', 'CREDIT')),
    account_type VARCHAR(50) NOT NULL,
    amount DECIMAL(15, 2) NOT NULL CHECK (amount >= 0),
    currency VARCHAR(3) DEFAULT 'INR',
    description TEXT,
    reference_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reward_id) REFERENCES rewards(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_ledger_reward_id ON ledger_entries(reward_id);
CREATE INDEX idx_ledger_user_id ON ledger_entries(user_id);
CREATE INDEX idx_ledger_account_type ON ledger_entries(account_type);
CREATE INDEX idx_ledger_created_at ON ledger_entries(created_at DESC);

COMMENT ON TABLE ledger_entries IS 'Double-entry ledger for all financial transactions';
COMMENT ON COLUMN ledger_entries.entry_type IS 'DEBIT or CREDIT';
COMMENT ON COLUMN ledger_entries.account_type IS 'e.g., STOCK_ASSET, CASH, BROKERAGE_EXPENSE, FEE_EXPENSE';

CREATE TABLE IF NOT EXISTS reward_requests (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) UNIQUE NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    stock_symbol VARCHAR(20) NOT NULL,
    quantity DECIMAL(15, 6) NOT NULL,
    request_payload JSONB NOT NULL,
    response_payload JSONB,
    status VARCHAR(20) DEFAULT 'PROCESSING',
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_reward_requests_event_id ON reward_requests(event_id);
CREATE INDEX idx_reward_requests_user_id ON reward_requests(user_id);
CREATE INDEX idx_reward_requests_status ON reward_requests(status);

COMMENT ON TABLE reward_requests IS 'Tracks all reward requests for idempotency';
COMMENT ON COLUMN reward_requests.event_id IS 'Unique event identifier to prevent duplicates';

CREATE TABLE IF NOT EXISTS corporate_actions (
    id SERIAL PRIMARY KEY,
    stock_symbol VARCHAR(20) NOT NULL,
    action_type VARCHAR(50) NOT NULL CHECK (action_type IN ('SPLIT', 'REVERSE_SPLIT', 'MERGER', 'DIVIDEND', 'BONUS')),
    action_date DATE NOT NULL,
    ratio_from INTEGER NOT NULL DEFAULT 1,
    ratio_to INTEGER NOT NULL DEFAULT 1,
    new_symbol VARCHAR(20),
    description TEXT,
    applied BOOLEAN DEFAULT FALSE,
    applied_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_corporate_actions_symbol ON corporate_actions(stock_symbol);
CREATE INDEX idx_corporate_actions_date ON corporate_actions(action_date DESC);
CREATE INDEX idx_corporate_actions_applied ON corporate_actions(applied);

COMMENT ON TABLE corporate_actions IS 'Tracks stock splits, mergers, and other corporate actions';
COMMENT ON COLUMN corporate_actions.ratio_from IS 'Original ratio (e.g., 1 for 1:2 split)';
COMMENT ON COLUMN corporate_actions.ratio_to IS 'New ratio (e.g., 2 for 1:2 split)';

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rewards_updated_at BEFORE UPDATE ON rewards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reward_requests_updated_at BEFORE UPDATE ON reward_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_corporate_actions_updated_at BEFORE UPDATE ON corporate_actions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO users (user_id, name, email) VALUES
    ('USR001', 'Utsav Tiwari', 'utsav.tiwari@example.com'),
    ('USR002', 'Jane Smith', 'jane.smith@example.com'),
    ('USR003', 'Bob Johnson', 'bob.johnson@example.com')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO stock_prices (stock_symbol, price, currency) VALUES
    ('AAPL', 175.50, 'INR'),
    ('GOOGL', 142.30, 'INR'),
    ('MSFT', 380.75, 'INR'),
    ('TSLA', 245.60, 'INR'),
    ('AMZN', 155.20, 'INR')
ON CONFLICT DO NOTHING;
