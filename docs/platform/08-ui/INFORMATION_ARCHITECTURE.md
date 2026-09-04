# UI / UX Information Architecture

## Apps
A single Web/PWA (Next.js) covering Traveler flows + owner/admin + platform admin.

## Navigational model
- **Trip-centric**: everything hangs off the active Trip.
- Top-level modules within a Trip: Overview, Decisions, Budget, Fund, Expenses, Itinerary.
- Money flows (Fund, Payments, Expenses) are the authoritative source for financial state.

## Main screens (MVP, Phase 1)
1. **Auth** — register / login
2. **Trips list** — my trips, invitations, status
3. **Trip detail (Overview/Dashboard)** — progress, budget, fund, pending members
4. **Trip setup** — dates, destination, team, roles
5. **Decisions** — create poll, vote, results, promote to config
6. **Budget** — categories, items, amounts, progress
7. **Fund** — goal, per-person, contribution plans, ledger
8. **Payments** — authorize method, contributions, history
9. **Expenses** — record, split, participants, settlement

## Information hierarchy within a Trip

```
Trip
├── People
├── Decisions
├── Budget
├── Fund        ← financial heart
├── Payments
├── Expenses
└── Itinerary
```

## UX principles
- Minimal cognitive load — show status not just state (e.g. "82% funded", "3 pending").
- Money is explicit: expected vs collected vs pending always distinguishable.
- Irreversible financial actions require confirmation + audit.
- Mobile-first (it's how trips are coordinated, on the phone).
