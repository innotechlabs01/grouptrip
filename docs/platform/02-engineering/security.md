# Security Model

Applies especially to the financial parts.

## Payment security
- **Never store card numbers.** Payment processor owns tokenization, security, compliance.
- Platform stores only **secure references** to authorized methods + authorization state.
- Provider fraud/risk checks.
- Payment reconciliation.

## Webhooks
- Signed webhooks.
- Idempotency keys.

## Application security
- Audit logs (every important movement).
- RBAC (trip + platform roles).
- Rate limiting.
- Encryption (at rest and in transit).
- Secrets management.

## Financial model (MVP)
- Coordination, not custody.
- Payment orchestration via provider — the processor is responsible for sensitive data.
