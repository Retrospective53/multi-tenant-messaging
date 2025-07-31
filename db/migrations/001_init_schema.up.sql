-- Enable UUID generation extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Track all tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Partitioned messages table
CREATE TABLE messages (
    id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id) -- ✅ composite primary key required
) PARTITION BY LIST (tenant_id);

-- Example tenant partition
-- Replace the UUID below with the actual tenant ID
CREATE TABLE messages_tenant_1 PARTITION OF messages
    FOR VALUES IN ('123e4567-e89b-12d3-a456-426614174000');

-- Optional: add index for sorting and pagination in partition
CREATE INDEX idx_messages_tenant_1_created_at ON messages_tenant_1 (created_at DESC);

-- Dead-letter table for failed message delivery
CREATE TABLE dead_letters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    original_payload JSONB NOT NULL,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Link dead letters to tenants
ALTER TABLE dead_letters
ADD CONSTRAINT fk_dead_letter_tenant
FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Indexes for dead_letters
CREATE INDEX idx_dead_letters_tenant_created ON dead_letters (tenant_id, created_at DESC);
