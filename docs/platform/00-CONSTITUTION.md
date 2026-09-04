# GroupTrip Constitution

Version: 0.1
Status: Draft

Immutables. Live here forever. Any change = constitutional amendment with full audit.

## §1 Identity

GroupTrip is a **Group Travel Operating System** — the platform where any group moves from
"we should do a trip" to "the trip is organized, funded, and ready" without coordinating
money, decisions, and bookings across multiple apps.

Tagline: **"From the trip idea to the paid and organized trip."**

## §2 North Star

**Funded Trips** — trips with participants, budget, financial goal, and at least one
contribution received.

A trip created but never funded demonstrates no value. Funding is the signal.

## §3 Core Principle — Coordination, not Custody (MVP)

The platform **orchestrates payments**; it does not hold a wallet by default.

- **A — Financial coordination**: app says "Juan authorized paying $100,000." Simpler to ship.
- **B — Money custody**: app actually receives, stores, holds, moves money. Regulatory burden.

MVP = **A**. Payment orchestration via payment provider, never a self-owned wallet.

## §4 Ledger Discipline

**Registered ≠ available.**

The system must always separate:
- **EXPECTED** — what the group agreed to contribute
- **COLLECTED** — what was actually received
- **PENDING** — authorized but not yet collected
- **FAILED** — attempts that failed

Never assume logical balance equals real custodial money.

## §5 Payment Security

- Never store card numbers. Payment processor owns tokenization, security, compliance.
- Platform stores only secure references to authorized methods + authorization state.
- Signed webhooks. Idempotency keys. Audit logs. RBAC. Rate limiting. Encryption.
- Secrets management. Provider fraud/risk checks. Payment reconciliation.

## §6 Auditability

In money, auditing is a design requirement, not a feature.

Every important movement generates an `AuditLog`:
- Actor, Action, Entity, EntityId, PreviousState, NewState, Timestamp, Metadata

## §7 Financial Ledger Separation

Split **logical balance** from **actually-custodied money**. This is architecturally mandatory.

## §8 Event-Driven

The product is fundamentally event-driven. Domain events propagate state:
- ContributionSucceeded → FundUpdated → BudgetProgressUpdated → TripProgressUpdated → NotificationTriggered
- PaymentFailed → ContributionFailed → FundRiskDetected → NotifyParticipant

## §9 Strategic Models

- Auth: user-owned. `Contribution Plan` authorizes future charges via payment provider.
- Decisions become Trip Configuration (Poll → Result → Decision → Configuration), not dead surveys.
- MVP scope: organization + money first. Social layer deferred to post-MVP.

## §10 Tech Foundations

- Backend: Go, Clean Architecture, DDD, Domain Events
- Frontend: **PWA-first** — single Web/PWA (Next.js), installable on mobile, with push
  notifications and Web Payments API (Apple Pay / Google Pay) for in-flow contributions
- Native app = post-MVP evolution only, never a blocker (frontend interchangeable; no backend rewrite)
- Database: Turso (libSQL / SQLite-compatible), embedded + distributed replicas
- Event-driven architecture
- Monorepo: `backend/` (Go API) + `pwa/` (Next.js PWA) + `docs/` (specs), each self-contained

## §11 MVP Boundary

Phase 1 — Core: Trip, Members, Decisions, Budget, Fund, Expenses, Payments, Dashboard.
Reservations, activities, itinerary, AI, social, marketplace, B2B = later phases.

## §12 No Silent Assumptions

Never assume "registered = money available". Never assume "voted = decided".
Never assume "decided = paid". State must be explicit and observable.

---

## Amendment Process

1. Propose CHANGE in `05-decisions/`
2. Full audit trail of why
3. Version bump in header
4. Old version archived in `archive/`
