Target EU(Sweden, Denmark)/UK, Indonesia, Global

```
┌─────────────────────────────────────────────────────────────┐
│                    SECURITY LAYER                           │
├─────────────────────────────────────────────────────────────┤
│  UNIVERSAL (All Regions):                                   │
│  ├── mTLS (Mutual TLS)                                      │
│  ├── JWS (Message Signing)                                  │
│  ├── Idempotency Key                                        │
│  ├── Request ID (tracing/debugging)                         │
│  └── TLS 1.2+ (minimum)                                     │
│                                                             │
│  FAPI 2.0 (EU - PSD2 Compliance):                           │
│  ├── OAuth 2.0 + PAR                                        │
│  ├── DPoP (mobile/SPA)                                      │
│  ├── private_key_jwt                                        │
│  ├── PKCE (S256)                                            │
│  ├── State parameter (CSRF)                                 │
│  └── JARM (response signing)                                │
│                                                             │
│  UK (Post-Brexit):                                          │
│  ├── UK FAPI (FCA compliance)                               │
│  ├── OAuth 2.0 + enhanced security                          │
│  └── Faster Payments/CHAPS auth                             │
│                                                             │
│  FedNow/US (Certificate-based):                             │
│  ├── PKI Certificate Auth                                   │
│  ├── FedLine VPN/WAN                                        │
│  └── ISO 20022 Message Signing                              │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    PAYLOAD LAYER                            │
├─────────────────────────────────────────────────────────────┤
│  Transaction Flow (Wise-style):                             │
│  ├── Quote API (lock FX rate)                               │
│  ├── Recipient API (validate destination)                   │
│  ├── Compliance Check (AML/sanctions)                       │
│  └── Transfer API (execute)                                 │
│                                                             │
│  Message Format:                                            │
│  ├── ISO 20022 (FedNow, SEPA, BI-FAST)                      │
│  └── Custom JSON (Wise, most APIs)                          │
│                                                             │
│  Webhook/Callback:                                          │
│  ├── Transaction status updates                             │
│  ├── JWS signature verification                             │
│  └── Retry logic with exponential backoff                   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                  REGIONAL ADAPTER                           │
├─────────────────────────────────────────────────────────────┤
│  Indonesia (SNAP):                                          │
│  ├── OAuth 2.0 + X-SIGNATURE                                │
│  ├── BI-FAST for instant transfers                          │
│  ├── RTGS for large amounts                                 │
│  └── SNAP payload format                                    │
│                                                             │
│  EU (PSD2/SCA):                                             │
│  ├── FAPI 2.0 Security Profile                              │
│  ├── SCA (MFA) for high-risk ops                            │
│  ├── SEPA Instant (SCT Inst)                                │
│  └── SEPA/ISO 20022                                         │
│                                                             │
│  Sweden/Denmark Specifics:                                  │
│  ├── BankID integration (Sweden)                            │
│  ├── MitID integration (Denmark)                            │
│  ├── Bankgiro/Plusgiro (Sweden domestic)                    │
│  └── Local clearing systems                                 │
│                                                             │
│  UK (Post-Brexit):                                          │
│  ├── Faster Payments (domestic)                             │
│  ├── CHAPS (high-value)                                     │
│  ├── SWIFT for international                                │
│  └── No SEPA (use alternative routes)                       │
│                                                             │
│  US (FedNow/ACH):                                           │
│  ├── Certificate-based auth                                 │
│  ├── FedLine connectivity                                   │
│  └── ISO 20022 native                                       │
│                                                             │
│  COMPLIANCE LAYER:                                          │
│  ├── AML/KYC screening                                      │
│  ├── Sanctions list checking (OFAC, EU)                     │
│  ├── Transaction monitoring                                 │
│  ├── PEP screening                                          │
│  └── Country-specific limits                                │
└─────────────────────────────────────────────────────────────┘
```