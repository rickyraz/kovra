-- +goose Up
-- +goose StatementBegin

-- Enable Row Level Security on transfer partitions
-- This enforces data residency compliance at the database level

-- Create roles for regional access control
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kovra_id_region') THEN
        CREATE ROLE kovra_id_region;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kovra_eu_region') THEN
        CREATE ROLE kovra_eu_region;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kovra_uk_region') THEN
        CREATE ROLE kovra_uk_region;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kovra_global') THEN
        CREATE ROLE kovra_global;  -- Auditors with global access
    END IF;
END
$$;

-- Enable RLS on each partition
ALTER TABLE transfers_id ENABLE ROW LEVEL SECURITY;
ALTER TABLE transfers_eu ENABLE ROW LEVEL SECURITY;
ALTER TABLE transfers_uk ENABLE ROW LEVEL SECURITY;

-- Indonesia partition: OJK data residency compliance
-- Only users with kovra_id_region or kovra_global role can access
CREATE POLICY ojk_data_residency ON transfers_id
    FOR ALL
    TO PUBLIC
    USING (
        pg_has_role(current_user, 'kovra_id_region', 'MEMBER') OR
        pg_has_role(current_user, 'kovra_global', 'MEMBER') OR
        current_user = 'kovra'  -- App user has full access
    );

-- EU partition: GDPR data residency compliance
CREATE POLICY gdpr_data_residency ON transfers_eu
    FOR ALL
    TO PUBLIC
    USING (
        pg_has_role(current_user, 'kovra_eu_region', 'MEMBER') OR
        pg_has_role(current_user, 'kovra_global', 'MEMBER') OR
        current_user = 'kovra'
    );

-- UK partition: FCA data residency compliance
CREATE POLICY fca_data_residency ON transfers_uk
    FOR ALL
    TO PUBLIC
    USING (
        pg_has_role(current_user, 'kovra_uk_region', 'MEMBER') OR
        pg_has_role(current_user, 'kovra_global', 'MEMBER') OR
        current_user = 'kovra'
    );

-- Grant base permissions to app user
GRANT ALL ON transfers TO kovra;
GRANT ALL ON transfers_id TO kovra;
GRANT ALL ON transfers_eu TO kovra;
GRANT ALL ON transfers_uk TO kovra;
GRANT ALL ON transfers_unknown TO kovra;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop RLS policies
DROP POLICY IF EXISTS fca_data_residency ON transfers_uk;
DROP POLICY IF EXISTS gdpr_data_residency ON transfers_eu;
DROP POLICY IF EXISTS ojk_data_residency ON transfers_id;

-- Disable RLS
ALTER TABLE transfers_uk DISABLE ROW LEVEL SECURITY;
ALTER TABLE transfers_eu DISABLE ROW LEVEL SECURITY;
ALTER TABLE transfers_id DISABLE ROW LEVEL SECURITY;

-- Drop roles (only if they exist and have no dependencies)
DROP ROLE IF EXISTS kovra_global;
DROP ROLE IF EXISTS kovra_uk_region;
DROP ROLE IF EXISTS kovra_eu_region;
DROP ROLE IF EXISTS kovra_id_region;

-- +goose StatementEnd
