-- name: GetLegalEntityByID :one
SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
    fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
    nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
    supported_currencies, supported_rails, updated_at
FROM legal_entities
WHERE id = $1;

-- name: GetLegalEntityByCode :one
SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
    fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
    nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
    supported_currencies, supported_rails, updated_at
FROM legal_entities
WHERE code = $1;

-- name: ListLegalEntitiesByJurisdiction :many
SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
    fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
    nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
    supported_currencies, supported_rails, updated_at
FROM legal_entities
WHERE jurisdiction = $1
ORDER BY code;

-- name: ListLegalEntities :many
SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
    fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
    nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
    supported_currencies, supported_rails, updated_at
FROM legal_entities
ORDER BY code;
