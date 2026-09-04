# Event-Driven Architecture

The product is fundamentally event-driven. Domain events propagate state changes across
aggregates and trigger downstream side effects (notifications, risk detection, progress updates).

## Why
- Decouples aggregates (Trip, Fund, Payment, Budget)
- Natural fit for money flows
- Leaves the system ready to scale (async processing, projections)

## Core Event Chains

### Funding success
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

### Payment failure
```
PaymentFailed
       ↓
ContributionFailed
       ↓
FundRiskDetected
       ↓
NotifyParticipant
```

## Event Types
- Domain events (ContributionSucceeded, PaymentFailed, DecisionResolved)
- Integration events (when crossing service boundaries)
- Notifications derived from domain events (never mutated directly at source)

## Priorities
1. Domain events are the source of truth for derived state
2. Idempotency for all event handlers
3. At-least-once delivery; handlers must be idempotent
4. Audit log records every important event

## Webhooks
- Outbound: notify users (push/email/whatsapp)
- Inbound: payment provider webhooks, **signed**, with idempotency keys
