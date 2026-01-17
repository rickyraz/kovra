-- name: CreateTransfer :one
INSERT INTO transfers (
    tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, recipient_id,
    idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee, rail
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
    idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
    status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
    tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
    updated_at, completed_at;

-- name: GetTransferByID :one
SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
    idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
    status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
    tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
    updated_at, completed_at
FROM transfers
WHERE id = $1;

-- name: GetTransferByIdempotencyKey :one
SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
    idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
    status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
    tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
    updated_at, completed_at
FROM transfers
WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: UpdateTransferStatus :exec
UPDATE transfers
SET status = $2, failure_reason = $3, updated_at = NOW(),
    completed_at = CASE WHEN $2 IN ('completed', 'rolled_back', 'cancelled', 'rejected') THEN NOW() ELSE completed_at END
WHERE id = $1;

-- name: UpdateTransferTBTransferIDs :exec
UPDATE transfers
SET tb_transfer_ids = $2, updated_at = NOW()
WHERE id = $1;

-- name: ListTransfersByTenant :many
SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
    idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
    status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
    tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
    updated_at, completed_at
FROM transfers
WHERE tenant_id = $1
    AND (sqlc.narg('status')::transfer_status_enum IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('from_currency')::text IS NULL OR from_currency = sqlc.narg('from_currency'))
    AND (sqlc.narg('to_currency')::text IS NULL OR to_currency = sqlc.narg('to_currency'))
    AND (sqlc.narg('compliance_region')::text IS NULL OR compliance_region = sqlc.narg('compliance_region'))
    AND (sqlc.narg('updated_after')::timestamptz IS NULL OR updated_at >= sqlc.narg('updated_after'))
    AND (sqlc.narg('updated_before')::timestamptz IS NULL OR updated_at < sqlc.narg('updated_before'))
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransfersByTenantAndStatus :many
SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
    idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
    status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
    tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
    updated_at, completed_at
FROM transfers
WHERE tenant_id = $1 AND status = $2
ORDER BY updated_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateTransferRailReference :exec
UPDATE transfers
SET rail_reference = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateTransferComplianceStatus :exec
UPDATE transfers
SET compliance_status = $2, risk_score = $3, screened_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: UpdateTransferNetting :exec
UPDATE transfers
SET netting_group_id = $2, is_netted = $3, updated_at = NOW()
WHERE id = $1;
