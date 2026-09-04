# Architecture

## System Overview

```
                    ┌─────────────────┐
                    │   Web / PWA     │
                    │ Next.js         │
                    └────────┬────────┘
                             │
                      API / Application
                             │
        ┌────────────────────┼───────────────────┐
        │                    │                   │
     Identity             Trips              Finance
        │                    │                   │
        ├──────────────┬─────┴───────┬───────────┤
        │              │             │
    Decisions        Budget         Fund
                                      │
                                  Payments
                                      │
                              Payment Provider
                                      │
                              Bank / Card Rails
```

## Backend Layering (Go)

```
/internal

  /domain
      /trip
      /member
      /decision
      /budget
      /fund
      /payment
      /expense
      /reservation

  /application
      /commands
      /queries

  /infrastructure
      /database
      /payments
      /notifications

  /interfaces
      /http
      /webhooks
```

## Principles
- Clean Architecture
- DDD
- Domain Events
- Event-driven (see `event-driven.md`)
- PostgreSQL persistence (see `database.md`)
- Payment orchestration, not wallet ownership (MVP)

## Event Examples

ContributionSucceeded → FundUpdated → BudgetProgressUpdated → TripProgressUpdated → NotificationTriggered

PaymentFailed → ContributionFailed → FundRiskDetected → NotifyParticipant
