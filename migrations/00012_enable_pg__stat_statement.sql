-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION pg_stat_statements;
- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP EXTENSION IF EXISTS pg_stat_statements;
-- +goose StatementEnd
