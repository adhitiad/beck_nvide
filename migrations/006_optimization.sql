-- Phase 5: Scaling & Optimization

-- Composite Indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_streams_status_started ON streams(status, started_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_gift_tx_stream_created ON gift_transactions(stream_id, created_at DESC);

-- Partitioning Example (Since we cannot easily convert an existing table to a partitioned table in postgres without recreating it, 
-- we will just note this as part of the schema design for future fresh deployments, or create a partitioned history table)

-- Create a partitioned table for transaction history to archive old transactions
CREATE TABLE IF NOT EXISTS transaction_history (
    id UUID,
    user_id UUID,
    type VARCHAR(50),
    amount BIGINT,
    currency VARCHAR(10),
    status VARCHAR(20),
    reference_id VARCHAR(255),
    payment_method VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE
) PARTITION BY RANGE (created_at);

-- Create partitions for the next few months
CREATE TABLE IF NOT EXISTS transaction_history_y2026m05 PARTITION OF transaction_history FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS transaction_history_y2026m06 PARTITION OF transaction_history FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Add indexes to the partitioned table
CREATE INDEX IF NOT EXISTS idx_tx_history_user_created ON transaction_history(user_id, created_at DESC);

