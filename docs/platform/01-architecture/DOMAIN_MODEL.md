# Domain Model

The system is organized around the `Trip` aggregate and its supporting domains.

## Trip — Central Aggregate

```
TRIP
│
├── People          (members, roles, invitations)
├── Decisions       (polls, votes, results → trip configuration)
├── Budget          (categories, items, targets)
├── Fund            (goal, members, contribution plans, ledger)
├── Payments        (payment intents, attempts, methods)
├── Expenses        (individual / split / percentage / fund)
├── Reservations    (hotel, transport, tours)
├── Activities      (linked to fund → payment)
├── Itinerary       (timeline of everything)
├── Tasks           (reminders, to-dos)
├── Notifications   (push, email, whatsapp)
└── Memories        (social layer — post-MVP)
```

## Domains

### Identity (01)
- User, Profile, Contact, PaymentProfile, Preferences

### Trips (02)
- Trip, TripMember, TripRole, TripInvitation, TripSettings, TripStatus
- States: DRAFT, PLANNING, FUNDING, READY, ACTIVE, COMPLETED, CANCELLED

### Decisions (07)
- Decision, DecisionOption, DecisionVote
- Lifecycle: POLL → RESULT → DECISION → TRIP CONFIGURATION

### Budget (08)
- Budget, BudgetCategory, BudgetItem
- Categories: transport, accommodation, food, transfers, activities, entertainment,
  shopping, insurance, other

### Fund (09) — the financial heart
- Fund, FundMember, ContributionPlan, Contribution, FundLedger
- Per-person: goal / participants = per-person target
- Strategies: single, monthly, biweekly, weekly (auto-recommended)

### Payments (10)
- Payment, PaymentIntent, PaymentMethodReference, PaymentAttempt, Refund
- States: PENDING, AUTHORIZED, PROCESSING, SUCCEEDED, FAILED, REFUNDED, CANCELLED

### Expenses (13)
- Expense, ExpenseParticipant, Settlement
- Split modes: individual, equal, percentage, fund
- Settlement computes minimal number of transfers

### Reservations (15)
- Reservation (hotel, tours...)
- States: PLANNED, QUOTED, RESERVED, PAID, CANCELLED, COMPLETED

### Activities (16)
- Activity, linked to fund → payment

### Notifications (18)
- Notification, channels: push, email, whatsapp
- Events: contribution due, payment success/fail, poll ending, confirmation, budget risk

## Key Financial Separation

The **Fund Ledger** separates:
- EXPECTED — agreed total
- COLLECTED — real money received
- PENDING — authorized, not collected
- FAILED — failed attempts

Logical balance ≠ custodial money. Never conflate.

## Contribution Strategy Recommendation

Engine computes recommended plan from:
- trip date
- goal
- participants
- money already accumulated
