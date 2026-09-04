# Backlog — Phase 1 (Core)

Ordered by dependency. Financial domain is the core differentiator.

## 1. Identity (0-bootstrap)
- [ ] User registration + auth
- [ ] Profile
- [ ] Session management

## 2. Trips (nucleus)
- [ ] Trip creation
- [ ] Trip settings + status machine
- [ ] Trip members + roles
- [ ] Invitations (join via link)

## 3. Decisions
- [ ] Create poll (options, deadline, weights)
- [ ] Vote
- [ ] Resolve poll → result
- [ ] Promote result → trip configuration (destination, dates)

## 4. Budget
- [ ] Budget aggregate with categories
- [ ] Budget items + amounts
- [ ] Budget progress tracking

## 5. Fund (heart)
- [ ] Fund goal + participants + per-person target
- [ ] Contribution plans (single / monthly / biweekly / weekly)
- [ ] Strategy auto-recommendation
- [ ] Fund ledger (expected / collected / pending / failed)
- [ ] Fund progress

## 6. Payments
- [ ] Payment method authorization (via provider, tokenized)
- [ ] Contribution charge (one-time + scheduled)
- [ ] Payment attempt lifecycle
- [ ] Refund
- [ ] Webhooks (signed, idempotent)

## 7. Expenses
- [ ] Expense record (individual / equal / percentage / fund)
- [ ] Expense participants
- [ ] Settlement (minimal transfers)

## 8. Dashboard
- [ ] Trip overview: progress, budget, fund, pending
- [ ] Per-member contribution status

## Cross-cutting
- [ ] Notifications (channels, events)
- [ ] Audit logs
- [ ] Events plumbing + projections
