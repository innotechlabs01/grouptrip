# Architecture

## System Overview

```
                    ┌─────────────────┐
                    │  Web / PWA      │
                    │  Next.js        │
                    │  (PWA-first)    │
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

## Frontend — PWA-first

A single **PWA** (Next.js) is the one frontend for MVP. It is:
- **Mobile-installable** — added to the home screen as an icon, runs on iOS, Android, and desktop
- **Push-capable** — notifications via service workers (no App Store review friction)
- **Payments in flow** — Web Payments API enables Apple Pay / Google Pay for contribution
  authorization and one-time payments through the payment provider

The label in the diagram above (`Web / PWA`) reflects this one codebase.

> Native iOS/Android apps are a post-MVP **evolution**, not a requirement. The clean Go/API
> layer means the frontend is interchangeable — moving to a native app later does not
> require a backend rewrite.

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
- Turso (libSQL/SQLite-compatible) persistence (see `database.md`)
- Payment orchestration, not wallet ownership (MVP)

## Event Examples

ContributionSucceeded → FundUpdated → BudgetProgressUpdated → TripProgressUpdated → NotificationTriggered

PaymentFailed → ContributionFailed → FundRiskDetected → NotifyParticipant
