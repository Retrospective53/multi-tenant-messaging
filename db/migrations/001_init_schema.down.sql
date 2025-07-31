-- Drop partition tables (must be dropped before parent)
DROP TABLE IF EXISTS messages_tenant_1;

-- Drop the partitioned parent table
DROP TABLE IF EXISTS messages;

-- Drop dead letter table and its constraints
ALTER TABLE IF EXISTS dead_letters
DROP CONSTRAINT IF EXISTS fk_dead_letter_tenant;

DROP INDEX IF EXISTS idx_dead_letters_tenant_created;
DROP TABLE IF EXISTS dead_letters;

-- Drop tenants table
DROP TABLE IF EXISTS tenants;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
