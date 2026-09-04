# Fund — Technical Domain Specification

*Core financial aggregate. The product's heart.*

Status: Draft
Domain owner: Finance
Related ADRs: ADR-001 (payment orchestration, not custody), ADR-003 (ledger discipline)
Payment provider: **Polar.sh** (Merchant of Record) — see §9

---

## 1. Scope

The **Fund** is the group's shared financial goal plus the machinery to reach it:
contributions, contribution plans, and the audit ledger that records every movement.

**In scope:**
- Fund goal, participants, per-person target
- Contribution plans (single / monthly / biweekly / weekly) + strategy recommendation
- Contribution lifecycle (authorize, charge, succeed, fail, refund)
- Fund Ledger (append-only, EXPECTED / COLLECTED / PENDING / FAILED)
- Fund progress and Trip progress propagation (event-driven)

**Out of scope (per ADR-001 / ADR-003):**
- The platform does NOT hold money. Money is moved via a payment provider.
- Expense recording and settlement live in the Expense domain (separate).
- Reservation/activity payment *from* the fund is a downstream consumer of the ledger,
  coordinated via events, not owned here.

---

## 2. Aggregate & Entities

### Fund (aggregate root)
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | |
| trip_id | UUID | one fund per funding trip |
| goal_amount | Money | > 0 (invariant I-1) |
| status | FundStatus | OPEN / ACTIVE / FUNDED (see §6) |
| created_at / updated_at | timestamps | |

### FundMember
| Field | Type | Notes |
|-------|------|-------|
| fund_id | UUID | |
| user_id | UUID | |
| per_person_target | Money | derived = goal / participant_count |

### ContributionPlan
| Field | Type | Notes |
|-------|------|------|
| id | UUID | |
| fund_id | UUID | |
| user_id | UUID | |
| frequency | Frequency | SINGLE / MONTHLY / BIWEEKLY / WEEKLY |
| amount | Money | per occurrence |
| total_expected | Money | full plan value (amount × occurrences) |
| next_charge_at | timestamp | scheduled next occurrence |
| status | PlanStatus | ACTIVE / PAUSED / COMPLETED / FAILED (see §6) |
| payment_method_ref | ref | tokenized provider reference (ADR-001) |

### Contribution (each attempted charge)
| Field | Type | Notes |
|-------|------|------|
| id | UUID | |
| plan_id | UUID | nullable for direct one-off |
| fund_id | UUID | |
| amount | Money | |
| status | ContrStatus | see §6 |
| idempotency_key | string | unique per (plan, cycle) — invariant I-5 |
| payment_attempt_id | ref | link to Payment domain |

### FundLedgerEntry (append-only)
| Field | Type | Notes |
|-------|------|------|
| id | UUID | |
| fund_id | UUID | |
| type | LedgerType | EXPECTED / COLLECTED / PENDING / FAILED |
| delta | Money | signed amount |
| contribution_id | ref | link source |
| occurred_at | timestamp | |

Ledger is **append-only** (ADR-003). Balance is a derived read, never a stored mutable total.

---

## 3. Business Invariants

Never violate these:

- **I-1** `goal_amount > 0` — a fund cannot exist with a zero or negative goal.
- **I-2** `per_person_target = goal / participant_count` — participant_count ≥ 1 at fund activation.
- **I-3** `registered ≠ available` — a Contribution becomes COLLECTED only after a provider-confirmed
  success; registering a plan never counts as money received (ADR-003).
- **I-4** Contributions require **explicit user authorization** of a payment method (ADR-001).
  No charge without an AUTHORIZED plan/method backing it.
- **I-5** Idempotency — the same (plan, cycle) never charges twice. Each charge carries an
  idempotency key enforced by the provider and the ledger.
- **I-6** Ledger is append-only; balances are projections, never mutated in place.
- **I-7** Plan totals reconcile: `sum(amount × occurrences per plan) == fund goal` at funding
  completion, unless a deliberate goal adjustment (audited via ADR amendment-style event).

---

## 4. Contribution Strategy Recommendation

The engine recommends a plan automatically. Inputs:
- trip date (t)
- goal_amount (G)
- participant_count (P)
- money_already_collected (C) (COLLECTED ledger projection)
- now (today)

Derived:
- remaining = G − C
- per_person_remaining = remaining / P
- months_until_trip = ceil((t − now) / 30d)

Recommendation logic (priority order):
1. If `months_until_trip <= 1` and per_person_remaining is a single affordable chunk → **SINGLE**.
2. Else if trip is within the current month boundary and participants prefer spread → **WEEKLY**.
3. Else if `months_until_trip <= 3` → **MONTHLY** (default middle ground).
4. Else → **BIWEEKLY** or **MONTHLY** depending on per-payment affordability.

Amount per occurrence (per person):
- SINGLE: `per_person_remaining`
- MONTHLY: `per_person_remaining / months_until_trip`
- BIWEEKLY: `per_person_remaining / (ceil(months_until_trip × 2))`
- WEEKLY: `per_person_remaining / (ceil(months_until_trip × 4))`

All results rounded to a clean monetary unit (e.g. per configurable rounding; minimum
currency unit). Recommendation is a **suggestion** — the owner/admin may override. Output
is a `RecommendStrategy` query result, not an automatic mutation (owner approves).

---

## 5. Domain Events

### Funding success chain
```
ContributionSucceeded
        ↓
FundUpdated
        ↓
BudgetProgressUpdated
        ↓
TripProgressUpdated
        ↓
NotificationTriggered
```

### Failure / risk chain
```
PaymentFailed
       ↓
ContributionFailed
       ↓
FundRiskDetected
       ↓
NotifyParticipant
```

### Event payloads (minimal)
- `ContributionSucceeded { contribution_id, fund_id, amount, at }`
- `ContributionFailed { contribution_id, fund_id, amount, plan_id?, reason }`
- `FundUpdated { fund_id, collected, pending, failed, expected }` (ledger projection snapshot)
- `FundRiskDetected { fund_id, type: FAILED_RATE | BEHIND_SCHEDULE | GOAL_CHANGE, detail }`

Handlers must be **idempotent** (event-driven architecture requirement).

---

## 6. State Machines

### Fund
```
OPEN        (goal set, participants defined)
  │ activate
  ▼
ACTIVE      (plans being charged; funding in progress)
  │ collected >= goal
  ▼
FUNDED
```
Transitions are guarded: `OPEN→ACTIVE` requires participant_count ≥ 1; `ACTIVE→FUNDED`
requires COLLECTED ≥ goal_amount (I-2, I-1). Deliberate goal changes are distinct,
audited operations.

### ContributionPlan
```
ACTIVE
  │ charge dispatched
  ▼
PROCESSING
  ├── provider success → COMPLETED  (all occurrences done) / stays ACTIVE (recurring)
  └── provider fail
              ├── retryable → PENDING (scheduled retry)
              └── terminal  → FAILED
```
`PAUSED` is owner/user-managed (stop future charges, existing ones unaffected).

### Contribution
```
PENDING → AUTHORIZED → PROCESSING → SUCCEEDED / FAILED / REFUNDED / CANCELLED
```
Maps onto the Payment domain (`PaymentAttempt`) states; the Fund projects them into the ledger.

---

## 7. Commands & Queries

### Commands
- `CreateFund(trip_id, goal_amount)`
- `SetGoal(fund_id, amount)` — guarded, audited (I-1)
- `AddFundMember(fund_id, user_id)` — recompute per-person target (I-2)
- `ConfigureContributionPlan(fund_id, user_id, frequency, amount)` — requires authorized method (I-4)
- `AuthorizeContribution(user_id, plan_id)` — user authorizes method with provider
- `ChargeContribution(plan_id)` — create PaymentIntent; enforce idempotency (I-5)
- `HandleProviderWebhook(payload)` — signed; maps SUCCEEDED/FAILED to ledger (I-3)
- `RefundContribution(contribution_id)` — from SUCCEEDED
- `RecommendStrategy(fund_id)` → suggestion (see §4)

### Queries
- `GetFundProgress(fund_id)` → { goal, collected, pending, failed, per_person_target, pct }
- `GetFundLedger(fund_id)` → append-only entries
- `GetMemberContributionStatus(fund_id, user_id)` → each plan + latest contribution
- `GetStrategyRecommendation(fund_id)` → §4 result

---

## 8. Fund Ledger — detail

- **Append-only.** Entries are never updated or deleted (I-6).
- Projections (`collected`, `pending`, `failed`, `expected`) are computed by summing deltas
  per `LedgerType` — this is what the dashboard and events read.
- A `COLLECTED` entry is added **only** on a provider-confirmed success (I-3); a `PENDING`
  entry is added on authorization; a `FAILED` entry on terminal failure; `EXPECTED` on plan
  configuration.
- Reconciliation: `collected + pending + failed ≤ expected` always holds; drift is a
  detectable anomaly, not a normal state.
- Every projection is foreign-keyed to its originating `contribution_id` for full traceability.

---

## 9. Payment Provider — Polar.sh

**Selected provider: Polar.sh** (Merchant of Record). The Fund depends on a Polar *adapter*
behind a provider interface; the interface stays provider-agnostic, but Polar is the concrete
implementation for the MVP (option B of the earlier decision).

### Why Polar
- **Merchant of Record (MoR)** — Polar owns tax compliance, invoicing, and card-handling
  regulatory surface. This strongly supports the "coordination, not custody" doctrine (ADR-001).
- **Saved payment methods + off-session charges** — the exact capability the Fund needs for
  authorized recurring contributions.
- Lower fees; global payments.

### Mapping of required capabilities
| Fund capability | Polar mechanism |
|---|---|
| Tokenization (no card data) | `customerSessions.create` → `PaymentMethod.createInline` embed; Polar stores the card. Platform holds only a `payment_method_id` reference (ADR-001). |
| Autorizar método | Customer + embed flow; `setAsDefault` optional |
| Cobro off-session (aporte recurrente) | **Draft order** pattern: `POST /v1/orders/` (no charge) → `POST /v1/orders/{id}/finalize` (charge saved payment method) |
| Confirmación (I-3) | Webhook `order.paid` → mark Contribution SUCCEEDED → ledger COLLECTED |
| Refunds | Order refund APIs (from a SUCCEEDED order) |
| Reconciliación | Polar orders/statements vs Fund ledger (ADR-003) |

### Draft-order charging flow (contribution)
1. `POST /v1/orders/` — create draft for the customer with `customer_id`, `product_id`
   (a one-time Polar product), and optional `amount` (contribution, smallest currency unit),
   `currency` (ISO 4217, lowercase), `description` (e.g. trip + plan label). Does **not** charge.
2. `POST /v1/orders/{id}/finalize` — charge the customer's default payment method, or a
   specific `payment_method_id` (our adapter exposes this as an explicit parameter).
3. Webhook `order.paid` → the platform maps success → ledger `COLLECTED` (I-3).

> The adapter requires a **`product_id`** (one-time product) at draft-order creation — the
> Fund's contribution "product" must exist in Polar. A `Refund` must carry the exact amount
> from the Contribution/ledger record (Polar requires an explicit refund amount; the platform
> never hardcodes it).

> **Prerequisite:** off-session charges ("arbitrary charges" / draft-order finalize) are a
> **preview feature requiring a paid Polar plan**, the `orders:write` scope, and
> sales-management permissions. This must be provisioned before the Fund's automated
> contribution charging is enabled.

### Idempotency (I-5)
Polar does not expose a client-supplied idempotency key on draft-order create/finalize.
The platform enforces idempotency **at the model level**:
- **Persist `Contribution` as a first-class entity** (SQLite `contributions` table). The
  `ContributeCommand` writes the Contribution **PENDING before any provider call** (the
  idempotency anchor), stores the draft `order_id` as `external_ref`, and never re-charges:
  a non-terminal Contribution with the same ID is a no-op (retry path), a terminal one is refused.
- **Webhook `order.paid` is idempotent**: `HandleOrderPaid` looks the Contribution up by
  `external_ref`, advances it to SUCCEEDED, and records COLLECTED exactly once. A redelivered
  webhook on an already-terminal Contribution is a no-op; a duplicate COLLECTED is rejected by
  both the domain (`ErrContributionAlreadyRecorded`) and a partial UNIQUE index on
  `ledger.contribution_id WHERE type='COLLECTED'`.
- The ledger append is single-write per contribution (I-6).

### Security notes (per `02-engineering/security.md`)
- Signed webhooks; validate Polar's signature before trusting `order.paid`.
- Payment method embeds use short-lived server-issued `customerSession` tokens (expire 1h),
  scoped to the customer; never favor client-trusted session creation.
- Never store card data; persist only `payment_method_id` references.

---

## 10. Open Questions / Next Steps
- [ ] Provision Polar paid plan + verify off-session arbitrary-charge access (preview feature).
      **Status (probed 2026-09-04):** the LocalPlug sandbox org (`28dc9312-991b-4cb5-8022-192896135af4`)
      has `orders:write` but returns `403 OffSessionChargesNotEnabled` on `POST /v1/orders/`. Off-session
      is a preview feature that must be enabled by Polar support (paid plan) — **cannot be enabled via API**.
      Action: contact Polar support to opt the org in; until then automated draft→finalize charging is blocked.
- [ ] Decide refund policy (full vs partial) and link to `RefundContribution`.
- [x] **Webhook signature verification** (Standard Webhooks HMAC-SHA256: `webhook-id`/`webhook-timestamp`/
      `webhook-signature`, raw-body sign, 401 on invalid, fail-closed without secret) — commit `c1f8e87`.
- [x] Concrete `infrastructure/payments` Polar adapter (Go), tested against this contract.
- [x] Idempotent contribution charge command (`ContributeCommand`) + `order.paid` webhook handler + `Contribution` persistence (SQLite).

### Follow-ups (implementation note, not blocking v0)
- `ContributeCommand` marks a Contribution **FAILED on any finalize error** (incl. transient). A
  charge that Polar actually took but whose response was lost would be un-attributable; the
  finalize step should eventually leave transient errors in a retryable state and reconcile via
  Polar orders on a later pass.
- `webhookPolar` currently swallows events into `events.NoopSink{}` (no event infra yet); the
  `ContributionSucceeded`/`FundUpdated` events are emitted but not yet consumed.
- Not-found detection uses a small string match (`contribrepo: not found`) in a couple of spots;
  converge on a single shared sentinel if it grows.

