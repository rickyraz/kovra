# Security Architecture

Security controls for B2B cross-border payment rails platform targeting Indonesian exporters and e-commerce platforms with EU/UK markets.

## Security Principles

```
1. DEFENSE IN DEPTH    - Multiple layers, no single point of failure
2. LEAST PRIVILEGE     - Minimum access required for each operation
3. ZERO TRUST          - Verify everything, trust nothing
4. AUDIT EVERYTHING    - Immutable logs for all financial operations
5. FAIL SECURE         - Deny by default, explicit allow
```

## Security Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                    TRANSPORT SECURITY                           │
│         TLS 1.3 | mTLS (B2B) | Certificate Pinning              │
├─────────────────────────────────────────────────────────────────┤
│                    AUTHENTICATION                               │
│  API Key (B2B) | FAPI 2.0 (EU) | SNAP Auth (ID) | UK FAPI       │
├─────────────────────────────────────────────────────────────────┤
│                    MESSAGE INTEGRITY                            │
│  JWS Signing | X-SIGNATURE (SNAP) | ISO 20022 Signing           │
├─────────────────────────────────────────────────────────────────┤
│                    REQUEST PROTECTION                           │
│  Idempotency Key | Rate Limiting | Input Validation             │
├─────────────────────────────────────────────────────────────────┤
│                    COMPLIANCE                                   │
│  AML/KYC | OFAC | EU Sanctions | PEP | Transaction Monitoring   │
└─────────────────────────────────────────────────────────────────┘
```

## Transport Security

### TLS Configuration

```yaml
Minimum Version: TLS 1.2 (TLS 1.3 preferred)

Cipher Suites:
  - TLS_AES_256_GCM_SHA384
  - TLS_CHACHA20_POLY1305_SHA256
  - TLS_AES_128_GCM_SHA256

Certificate:
  - RSA 4096-bit or ECDSA P-384
  - Validity: 1 year max
  - OCSP Stapling: Required
```

### mTLS for B2B Partners

```go
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    ClientAuth: tls.RequireAndVerifyClientCert,
    ClientCAs:  loadTrustedCAs(),
    Certificates: []tls.Certificate{serverCert},
}
```

## Authentication by Region

### B2B API Key Authentication

```
Format: sk_live_{tenant_id}_{random_32_bytes}
Storage: SHA-256 hash in PostgreSQL, cached in Redis (5min TTL)

Lookup flow:
1. Hash incoming API key
2. Check Redis cache
3. If miss, query PostgreSQL
4. Verify not expired/revoked
5. Inject tenant_id into context
```

### EU - FAPI 2.0 (PSD2)

| Component | Purpose |
|-----------|---------|
| PAR | Pushed Authorization Request |
| PKCE (S256) | Code challenge |
| DPoP | Token binding to client key |
| private_key_jwt | Client authentication |
| JARM | Signed authorization response |

Token lifetime: 5 min access, 24h refresh

### Indonesia - SNAP

```
Layer 1: OAuth 2.0 Access Token (15 min lifetime)
Layer 2: X-SIGNATURE per request

Signature = HMAC-SHA512(
  HTTPMethod + ":" + 
  RelativeURL + ":" + 
  SHA256(RequestBody) + ":" + 
  Timestamp,
  ClientSecret
)

Required Headers:
- Authorization: Bearer {token}
- X-TIMESTAMP: {ISO8601}
- X-SIGNATURE: {signature}
- X-PARTNER-ID: {partner_id}
- X-EXTERNAL-ID: {unique_request_id}
```

### UK - FCA FAPI

Based on FAPI 1.0 Advanced:
- OAuth 2.0 + PAR
- PKCE required
- mTLS or private_key_jwt
- OBDirectory certificates

## Message Integrity

### JWS Webhook Signatures

```
Algorithm: RS256
Key rotation: 90 days (14 day overlap)

Header: {"alg": "RS256", "kid": "webhook-key-2025-01"}
Payload: {event_id, event_type, data, created_at}

JWKS endpoint: GET /v1/.well-known/jwks.json
```

### ISO 20022 Signing

XMLDSig with RSA-SHA256 for SEPA/SWIFT messages.

## Request Protection

### Idempotency

```
Header: X-Idempotency-Key: {uuid}
Storage: Redis (24h TTL)
Key: idempotency:{tenant_id}:{key}

Behavior:
- New key → Process request
- Existing + same payload → Return cached response
- Existing + diff payload → 422 Unprocessable
```

### Rate Limiting

| Tier | Requests/min | Daily Volume |
|------|--------------|--------------|
| Starter | 100 | $10,000 |
| Growth | 500 | $100,000 |
| Enterprise | 2,000 | $1,000,000+ |

Per-endpoint limits:
- POST /transfers: 100/min
- POST /batches: 10/min
- GET /quotes: 500/min

Response headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### Input Validation

```go
type TransferRequest struct {
    QuoteID     uuid.UUID       `validate:"required,uuid4"`
    RecipientID uuid.UUID       `validate:"required,uuid4"`
    Amount      decimal.Decimal `validate:"required,gt=0,max=1000000"`
    Currency    string          `validate:"required,iso4217"`
    Reference   string          `validate:"max=140,printascii"`
}
```

Validation includes:
- Type coercion
- Range checks
- Format validation (ISO 4217, IBAN)
- SQL injection prevention (parameterized queries)
- XSS prevention (escape on output)

## Data Security

### Encryption at Rest

| Data | Method | Key Management |
|------|--------|----------------|
| PostgreSQL | TDE (AES-256) | AWS KMS / Vault |
| TigerBeetle | Built-in AES-256 | Dedicated keys |
| Redis | TLS only | Ephemeral data |
| Backups | AES-256-GCM | Separate keys |

### Data Classification

```
TIER 1 - CRITICAL (App-level encryption):
  Bank accounts, National IDs, API keys, Private keys

TIER 2 - SENSITIVE (DB encryption + access control):
  Full name, DOB, Address, Phone, Email

TIER 3 - INTERNAL (Access control):
  Transfer amounts, Timestamps, Status history

TIER 4 - PUBLIC:
  Currency codes, FX rates, API docs
```

### Field-Level Encryption

```go
func EncryptPII(plaintext string, keyID string) (*EncryptedField, error) {
    key := keyStore.Get(keyID)
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
    return &EncryptedField{Ciphertext: ciphertext, Nonce: nonce, KeyID: keyID}, nil
}
```

## Compliance Controls

### KYC Tiers

| Tier | Volume | Requirements |
|------|--------|--------------|
| Basic | < $1K/mo | Email, phone, business registration |
| Standard | $1K-$50K/mo | + Gov ID, address proof, business license, UBO |
| Enhanced | > $50K/mo | + Source of funds, bank statements, on-site visit |

### Sanctions Screening

Lists checked:
- **Global**: UN Consolidated, Interpol
- **US (OFAC)**: SDN, Consolidated, Sectoral
- **EU**: Consolidated Financial Sanctions
- **UK**: HMT, UK Sanctions List
- **Indonesia**: PPATK, BI Blacklist

Frequency:
- Real-time: Every transfer
- Daily: Full customer base
- On-update: When lists change

Matching: Exact + Fuzzy (Levenshtein) + Phonetic + Transliteration

### Transaction Monitoring

```go
// Risk score thresholds
// > 80  → Auto-block + alert
// 50-80 → Manual review
// < 50  → Auto-approve

Rules:
- Velocity: > $50K in 24h → score 60
- Structuring: 3+ transfers near $10K threshold → score 85
- High-risk country → score 50
- First-time corridor → score 30
```

### Reporting Thresholds

| Region | Threshold | Report | Deadline |
|--------|-----------|--------|----------|
| Indonesia | IDR 100M | LTKM to PPATK | 3 days |
| EU | €15,000 | SAR | 24 hours |
| UK | £10,000 | SAR to NCA | 24 hours |

## Audit Logging

### Log Structure

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event_type": "transfer.initiated",
  "trace": {"request_id": "req_abc", "trace_id": "..."},
  "actor": {"tenant_id": "...", "ip": "203.0.113.50"},
  "resource": {"type": "transfer", "id": "tf_xyz"},
  "changes": {"status": {"from": null, "to": "INITIATED"}},
  "compliance": {"aml_check": "PASSED", "risk_score": 15}
}
```

### Retention

| Type | Retention | Reason |
|------|-----------|--------|
| Security events | 7 years | PCI-DSS, AML |
| Access logs | 2 years | Audit trail |
| App logs | 90 days | Debugging |

### Immutability

```sql
CREATE TRIGGER audit_trail_immutable
    BEFORE UPDATE OR DELETE ON audit_trail
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();
```

## Incident Response

### Severity Levels

| Level | Response | Example |
|-------|----------|---------|
| P1 Critical | 15 min | Data breach, funds at risk |
| P2 High | 1 hour | Auth bypass, compliance failure |
| P3 Medium | 4 hours | Rate limit abuse |
| P4 Low | 24 hours | Failed login attempts |

### Response Phases

1. **Detection**: Alerts, log analysis, external reports
2. **Triage**: Assess severity, identify affected tenants
3. **Containment**: Isolate, revoke credentials, block IPs
4. **Eradication**: Root cause, patch, remove access
5. **Recovery**: Restore service, verify integrity
6. **Lessons**: Post-mortem, update runbooks

## Key Rotation Schedule

| Key Type | Rotation | Method |
|----------|----------|--------|
| API signing keys | 90 days | Overlap + gradual rollout |
| Encryption keys (DEK) | 1 year | Re-encrypt on rotation |
| Master keys (KEK) | 2 years | KMS managed |
| TLS certificates | 1 year | Auto-renewal |
| OAuth client secrets | 6 months | Regenerate + notify |

## Security Headers

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
```

## Vulnerability Management

### Scanning

| Type | Frequency | Tool |
|------|-----------|------|
| SAST | Every PR | CodeQL, Semgrep |
| DAST | Weekly | OWASP ZAP |
| Dependencies | Daily | Dependabot, Snyk |
| Pentest | Annual | External firm |

### Patching SLA

| Severity | Deadline |
|----------|----------|
| Critical | 24 hours |
| High | 7 days |
| Medium | 30 days |
| Low | 90 days |