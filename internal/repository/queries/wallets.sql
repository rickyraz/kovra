-- name: CreateWallet :one
INSERT INTO wallets (tenant_id, currency, tb_account_id)
VALUES ($1, $2, $3)
RETURNING id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, updated_at;

-- name: GetWalletByID :one
SELECT id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, updated_at
FROM wallets
WHERE id = $1;

-- name: GetWalletByTenantAndCurrency :one
SELECT id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, updated_at
FROM wallets
WHERE tenant_id = $1 AND currency = $2;

-- name: ListWalletsByTenant :many
SELECT id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, updated_at
FROM wallets
WHERE tenant_id = $1
ORDER BY currency;

-- name: UpdateWalletCachedBalance :exec
UPDATE wallets
SET cached_balance = $2, cached_pending = $3, cached_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: UpdateWalletStatus :exec
UPDATE wallets
SET status = $2, updated_at = NOW()
WHERE id = $1;
