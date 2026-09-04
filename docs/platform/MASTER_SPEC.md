# GroupTrip Master Specification

## Purpose

This document is the **entry point** for the entire GroupTrip platform documentation.

GroupTrip is a **Group Travel Operating System** — not an expense splitter, not a
booking app. It is a collaborative platform where groups decide, plan, fund, execute,
and control shared trips through a single source of truth.

> The immutable rules live in `00-CONSTITUTION.md`.
> The navigation map for humans and AIs is `PLATFORM_INDEX.md`.
> This file only points to specialized documents — it contains no rules of its own.

---

## Product Vision
See: `04-product/VISION.md`

## Core Value Proposition
See: `04-product/VALUE_PROPOSITION.md`

## Business Model
See: `00-business/MODEL.md`

## Actors & Roles
See: `00-business/ACTORS.md`

## Architecture
See: `01-architecture/ARCHITECTURE.md` → `01-architecture/ddd.md`, `01-architecture/event-driven.md`

## Business Domains
See: `00-business/domains/` — identity, trips, decisions, budget, fund, payments, expenses, reservations, activities, notifications

## Domain Model
See: `01-architecture/DOMAIN_MODEL.md`

## State Machines
See: `06-state-machines/` — trip, decision, budget, fund, payment, reservation

## Workflows
See: `07-workflows/` — trip-creation, funding, payment, settlement

## API & Events
See: `01-architecture/api.md`, `01-architecture/event-driven.md`

## Database
See: `01-architecture/database.md`

## Security
See: `02-engineering/security.md`

## Infrastructure
See: `03-operations/` — infrastructure, deployment, monitoring

## Engineering Standards
See: `02-engineering/` — coding, testing, review, quality-gates

## UI / Design System
See: `08-ui/` — information architecture, ux-flows

## AI Features
See: `09-ai/` — trip-planner, financial-assistant

## Product Management
See: `11-product-management/` — MVP, roadmap, backlog, features

## Quality
See: `12-quality/` — checklists, definition-of-done

## Reference
See: `10-reference/` — glossary, naming conventions

## Decisions
See: `05-decisions/` — ADRs

---

## How to start
1. Read `PLATFORM_INDEX.md`
2. Read `00-CONSTITUTION.md`
3. Follow the reading order defined in `PLATFORM_INDEX.md`
