# Glossary

## Terms

**Trip** — the central aggregate: a group trip with people, decisions, budget, fund, payments, expenses, reservations, itinerary, tasks, notifications.

**Trip Fund** — the group's shared financial goal + contributions + ledger. The product's heart.

**Contribution Plan** — a participant's authorized, recurring contribution schedule (single / monthly / biweekly / weekly).

**Fund Ledger** — append-only record separating EXPECTED / COLLECTED / PENDING / FAILED.

**PaymentIntent** — a payment order (one-time or contribution) to be executed via provider.

**PaymentMethodReference** — a tokenized reference to an authorized payment method; never the raw card.

**Decision** — a group poll (options + votes) that resolves into Trip Configuration (Poll → Result → Decision → Config).

**Budget** — trip cost plan by category (transport, accommodation, food, transfers, activities, entertainment, shopping, insurance, other).

**Expense** — a recorded trip cost, split individually / equally / by percentage / from fund.

**Settlement** — the minimal-transfer resolution of who owes whom at the end.

**Reservation** — a booked service (hotel, tour, transport...) with status machine.

**Itinerary** — the operational timeline turning decisions + reservations into a day-by-day schedule.

**Funded Trip** (North Star) — a trip with participants, budget, goal, and ≥1 real contribution.

## Naming conventions
- Aggregates: singular noun (Trip, Fund, Budget, Expense)
- Join/relation entities: descriptive (TripMember, FundMember, ExpenseParticipant, BudgetItem)
- Ledgers/logs: append-only records (FundLedger, AuditLog)
