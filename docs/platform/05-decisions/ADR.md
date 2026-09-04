# Architecture Decision Records

Active decisions.

## ADR-001 — Payment orchestration, not wallet ownership (MVP)
**Status: Accepted**

**Context:** MVP needs to move money without heavy regulatory burden.

**Decision:** Coordinate payments via a payment provider (tokenized method references).
Do not build a self-owned wallet in the MVP.

**Consequences:**
- Lower regulatory surface at launch
- Provider owns card data, security, compliance
- Custody model (B) is a future, deliberate, regulated move

## ADR-002 — Go + Clean Architecture + DDD, PostgreSQL, Event-driven
**Status: Accepted**

**Context:** Domain complexity (esp. finance) needs strict boundaries; events fit money flows.

**Decision:** Backend Go (Clean Architecture + DDD + domain events); PostgreSQL; event-driven.

**Consequences:**
- Clear domain/application/infrastructure layering
- Event projections enable progress/notification side effects
- Financial ledger discipline is first-class

## ADR-003 — Ledger discipline: logical balance ≠ custodial money
**Status: Accepted**

**Context:** Must always distinguish registered vs available.

**Decision:** Fund Ledger keeps EXPECTED / COLLECTED / PENDING / FAILED separate.

**Consequences:**
- Dashboard + logic never confuse intended with real money
- Reconciliation is possible
