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

## ADR-004 — PWA-first, native as extension
**Status: Accepted**

**Context:** Group trips are coordinated on the phone, in motion, from the "friend who
always organizes" down to each participant checking "are we on track with the money?"
A mobile experience is mandatory, but multiple native apps at launch fragment the team
and add App Store / review friction.

**Decision:** The MVP frontend is a single **PWA** (Next.js), mobile-installable with push
notifications and Web Payments API (Apple Pay / Google Pay) for in-flow contributions.
Native iOS/Android apps are a post-MVP evolution, never a blocker.

**Consequences:**
- One Next.js codebase serves iOS, Android, and desktop
- Zero App Store submission/review friction; continuous deploy
- In-flow mobile payments via Web Payments API
- Clean Go/API layer keeps the frontend interchangeable — a future native app needs no backend rewrite

## ADR-005 — Payment provider: Polar.sh
**Status: Accepted**

**Context:** The Fund (financial heart) needs saved payment methods with **off-session
charges** for authorized recurring contributions, plus minimal regulatory surface per
ADR-001 (coordination, not custody).

**Decision:** Use **Polar.sh** as the payment provider. Polar is a Merchant of Record; the
Fund persists only `payment_method_id` references (never card data) and charges authorized
recurring contributions via the draft-order → finalize pattern. Kept behind a
provider-agnostic interface for future swap.

**Consequences:**
- MoR: Polar owns tax, invoicing, and card-handling compliance
- Contribution lifecycle: authorize (embed) → draft order → finalize → `order.paid` webhook → ledger
- Off-session charges require a paid plan + preview access + `orders:write` scope — must be provisioned
- Idempotency enforced at model level (Polar has no client idempotency key)
- Concrete Go adapter under `infrastructure/payments`, replacing the earlier "deferred" choice
