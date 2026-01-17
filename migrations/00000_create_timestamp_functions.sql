-- +goose Up
-- +goose StatementBegin

-- NO LONGER NEEDED: uuidv7_to_timestamp
-- UUIDv7 timestamp extraction is now native → use uuid_extract_timestamp()

-- DROP FUNCTION IF EXISTS uuid_to_ts(UUID);
-- DROP FUNCTION IF EXISTS uuidv7_to_timestamp(UUID);

-- STILL NEEDED:
-- Required for index-friendly time-range queries

CREATE OR REPLACE FUNCTION ts_to_uuid_min(ts TIMESTAMPTZ)
RETURNS UUID AS $$
DECLARE
    ms BIGINT;
    hex TEXT;
BEGIN
    ms := (EXTRACT(EPOCH FROM ts) * 1000)::BIGINT;
    hex := lpad(to_hex(ms), 12, '0');
    RETURN (
        substring(hex, 1, 8) || '-' || 
        substring(hex, 9, 4) || '-7000-8000-000000000000'
    )::UUID;
END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION ts_to_uuid_max(ts TIMESTAMPTZ)
RETURNS UUID AS $$
DECLARE
    ms BIGINT;
    hex TEXT;
BEGIN
    ms := (EXTRACT(EPOCH FROM ts) * 1000)::BIGINT;
    hex := lpad(to_hex(ms), 12, '0');
    RETURN (
        substring(hex, 1, 8) || '-' || 
        substring(hex, 9, 4) || '-7fff-bfff-ffffffffffff'
    )::UUID;
END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;

-- Extract timestamp (NATIVE)
-- SELECT uuid_extract_timestamp(id) AS created_at FROM orders;

-- -- Query by date range (CUSTOM - index friendly)
-- SELECT * FROM orders
-- WHERE id >= ts_to_uuid_min('2025-01-01'::TIMESTAMPTZ)
--   AND id < ts_to_uuid_min('2025-02-01'::TIMESTAMPTZ);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS ts_to_uuid_min(TIMESTAMPTZ);
DROP FUNCTION IF EXISTS ts_to_uuid_max(TIMESTAMPTZ);
-- +goose StatementEnd