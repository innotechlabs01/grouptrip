# Definition of Done

A ticket / feature is done only when ALL apply.

## Functional
- [ ] Works end-to-end against the MVP core loop
- [ ] Handles the happy path + known edge cases
- [ ] State transitions valid via state machine

## Financial rigor (where money is involved)
- [ ] Ledger discipline: expected / collected / pending / failed never conflated
- [ ] Idempotency on all payment/event handlers
- [ ] Registered ≠ available honored everywhere
- [ ] AuditLog generated for every important movement
- [ ] Webhooks signed + validated
- [ ] No card number ever stored

## Quality
- [ ] Tests green (unit + integration)
- [ ] Event-driven flows verified (handlers idempotent)
- [ ] Code review passed
- [ ] No silent degradation of the quality bar

## Observability
- [ ] Key events logged
- [ ] Failure paths observable
- [ ] Metrics for fund/payment KPIs available

## Security
- [ ] RBAC enforced at trip + platform level
- [ ] Rate limiting on sensitive endpoints
- [ ] Secrets handled via secrets manager

## Docs
- [ ] Any schema/API/event change reflected in this spec
