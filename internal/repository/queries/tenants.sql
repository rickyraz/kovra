-- name: CreateTenant :one
INSERT INTO tenants (display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
    tenant_status, kyc_level, netting_enabled, netting_window_minutes,
    api_key_hash, webhook_url, webhook_secret_hash, metadata, updated_at;

-- name: GetTenantByID :one
SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
    tenant_status, kyc_level, netting_enabled, netting_window_minutes,
    api_key_hash, webhook_url, webhook_secret_hash, metadata, updated_at
FROM tenants
WHERE id = $1;

-- name: UpdateTenant :one
UPDATE tenants SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    legal_name = COALESCE(sqlc.narg('legal_name'), legal_name),
    tenant_status = COALESCE(sqlc.narg('tenant_status'), tenant_status),
    kyc_level = COALESCE(sqlc.narg('kyc_level'), kyc_level),
    netting_enabled = COALESCE(sqlc.narg('netting_enabled'), netting_enabled),
    netting_window_minutes = COALESCE(sqlc.narg('netting_window_minutes'), netting_window_minutes),
    webhook_url = COALESCE(sqlc.narg('webhook_url'), webhook_url),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
    tenant_status, kyc_level, netting_enabled, netting_window_minutes,
    api_key_hash, webhook_url, webhook_secret_hash, metadata, updated_at;

-- name: ListTenantsByLegalEntity :many
SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
    tenant_status, kyc_level, netting_enabled, netting_window_minutes,
    api_key_hash, webhook_url, webhook_secret_hash, metadata, updated_at
FROM tenants
WHERE legal_entity_id = $1
ORDER BY id DESC;

-- name: ListTenantsByParent :many
SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
    tenant_status, kyc_level, netting_enabled, netting_window_minutes,
    api_key_hash, webhook_url, webhook_secret_hash, metadata, updated_at
FROM tenants
WHERE parent_tenant_id = $1
ORDER BY id DESC;

-- name: ListActiveTenants :many
SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
    tenant_status, kyc_level, netting_enabled, netting_window_minutes,
    api_key_hash, webhook_url, webhook_secret_hash, metadata, updated_at
FROM tenants
WHERE tenant_status = 'active'
ORDER BY id DESC
LIMIT $1 OFFSET $2;
