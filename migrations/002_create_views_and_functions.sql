-- Views and helper functions for analytics and reporting
-- Run this after the initial schema migration

CREATE OR REPLACE VIEW v_user_portfolio AS
SELECT 
    r.user_id,
    r.stock_symbol,
    SUM(r.quantity) as total_quantity,
    AVG(r.stock_price) as avg_purchase_price,
    SUM(r.total_value_inr) as total_invested_inr,
    SUM(r.brokerage_fee + r.transaction_fee) as total_fees,
    COUNT(*) as transaction_count,
    MIN(r.event_timestamp) as first_reward_date,
    MAX(r.event_timestamp) as last_reward_date
FROM rewards r
WHERE r.status = 'COMPLETED'
GROUP BY r.user_id, r.stock_symbol
HAVING SUM(r.quantity) > 0;

COMMENT ON VIEW v_user_portfolio IS 'Aggregated portfolio view per user and stock';


CREATE OR REPLACE VIEW v_daily_holdings AS
SELECT 
    r.user_id,
    r.stock_symbol,
    DATE(r.event_timestamp) as holding_date,
    SUM(r.quantity) as daily_quantity,
    SUM(r.total_value_inr) as daily_value_inr
FROM rewards r
WHERE r.status = 'COMPLETED'
GROUP BY r.user_id, r.stock_symbol, DATE(r.event_timestamp)
ORDER BY holding_date DESC;

COMMENT ON VIEW v_daily_holdings IS 'Daily stock holdings per user';


CREATE OR REPLACE FUNCTION get_latest_stock_price(p_stock_symbol VARCHAR)
RETURNS DECIMAL(15, 4) AS $$
DECLARE
    v_price DECIMAL(15, 4);
BEGIN
    SELECT price INTO v_price
    FROM stock_prices
    WHERE stock_symbol = p_stock_symbol
    ORDER BY timestamp DESC
    LIMIT 1;
    
    RETURN COALESCE(v_price, 0);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_latest_stock_price IS 'Returns the most recent price for a given stock symbol';


CREATE OR REPLACE FUNCTION get_user_portfolio_value(p_user_id VARCHAR)
RETURNS DECIMAL(15, 2) AS $$
DECLARE
    v_total_value DECIMAL(15, 2);
BEGIN
    SELECT COALESCE(SUM(
        vp.total_quantity * get_latest_stock_price(vp.stock_symbol)
    ), 0) INTO v_total_value
    FROM v_user_portfolio vp
    WHERE vp.user_id = p_user_id;
    
    RETURN v_total_value;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_user_portfolio_value IS 'Calculates total portfolio value at current prices';


CREATE OR REPLACE FUNCTION validate_ledger_balance(p_reward_id INTEGER)
RETURNS BOOLEAN AS $$
DECLARE
    v_debit_total DECIMAL(15, 2);
    v_credit_total DECIMAL(15, 2);
BEGIN
    SELECT COALESCE(SUM(amount), 0) INTO v_debit_total
    FROM ledger_entries
    WHERE reward_id = p_reward_id AND entry_type = 'DEBIT';
    
    SELECT COALESCE(SUM(amount), 0) INTO v_credit_total
    FROM ledger_entries
    WHERE reward_id = p_reward_id AND entry_type = 'CREDIT';
    
    RETURN v_debit_total = v_credit_total;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION validate_ledger_balance IS 'Validates that debits equal credits for a reward transaction';
