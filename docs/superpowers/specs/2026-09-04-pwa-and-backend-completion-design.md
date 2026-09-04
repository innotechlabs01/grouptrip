# grouptrip — PWA Build + Backend Completion — Design

Date: 2026-09-04
Scope: New Next.js PWA (all MVP screens) + completion of missing backend domains (auth, trips, budget, decisions, expenses), ending with a fully connected end-to-end product.

## 1. Vision & Goal

Build the complete GroupTrip **PWA** (Single Next.js build, mobile-installable) covering every MVP screen, driven by the **fund as the financial heart**, and then complete the backend domains the PWA depends on so every screen is wired to a real HTTP endpoint — no permanent mocks.

North Star: **Funded Trips** — trips with participants, budget, a financial goal, and at least one real contribution (Polar).

## 2. Decisions (already approved by the user)

| Decision | Choice |
|---|---|
| Scope | Build the whole PWA first; then complete the missing backend endpoints so it all functions end-to-end. |
| Stack (UI) | Next.js App Router + TypeScript + Tailwind CSS + shadcn/ui + TanStack Query + React Hook Form + Zod. |
| Auth | Real from day one. Backend auth domain (register/login/JWT) is built early and the PWA consumes it. |
| Auth security | argon2id hashes; JWT access token (15 min) + rotating refresh token in a cookie (httpOnly + SameSite); refresh tokens hashed in DB for real revocation; rate limiting on login/register. |
| Payments | Polar embed client-side (customerSession + PaymentMethod.createInline). PWA gets a `payment_method_id` and sends it to `POST /funds/{id}/contributions`. Web Payments (Apple/Google Pay) via Polar. |
| DB | Turso (libSQL / SQLite-compatible), already the backend store. |
| Architecture | Backend stays Clean/DDD Go. PWA is a separate Next.js app in `pwa/`. |

## 3. Repo layout

```
grouptrip/
├── backend/          # Go Clean/DDD API (existing)
│   └── internal/
│       ├── domain/   # fund (exists); + auth, trip, member, budget, decision, expense (new)
│       ├── application/ (commands/queries)
│       ├── infrastructure/ (turso, payments, + session/key store)
│       └── interfaces/http/
├── pwa/              # NEW Next.js App Router + TS + Tailwind + shadcn + React Query
└── docs/
    ├── platform/     # product/architecture specs (existing)
    └── superpowers/specs/  # this design + implementation plan
```

## 4. Backend completion (phases)

### 4.1 Auth domain (Fase 0 — required early)
Endpoints:
- `POST /auth/register` — { email, password, name? } → 201, sets session
- `POST /auth/login` — { email, password } → set session, return user
- `POST /auth/logout` — revoke refresh token, clear cookie
- `GET /auth/me` — current user from access token
- `POST /auth/refresh` — rotate refresh token (cookie), return new access

Security:
- Passwords: **argon2id** (golang.org/x/crypto/argon2 via a small wrapper, or `github.com/alexedwards/argon2id`).
- Access: JWT HS256/EdDSA, 15 min, in-memory on the client.
- Refresh: long-lived random token, stored **hashed** in Turso `refresh_tokens` table; rotation on each use; replay of a used token revokes the whole user's session chain.
- Cookie: `httpOnly`, `Secure`, `SameSite=Lax`, path `/auth`.
- Rate limit on `/auth/login` and `/auth/register` (per IP) — simple in-process limiter for MVP.

Storage (Turso): `users` (id, email unique, password_hash, name, created_at), `refresh_tokens` (id, user_id, token_hash, expires_at, revoked_at, created_at).

### 4.2 Trip + Member domain
- `POST /trips` create, `GET /trips` list mine, `GET /trips/{id}` detail, `PATCH /trips/{id}`, `POST /trips/{id}/members` invite, list/update members/roles.
- Auth-scoped: trips belong to a user; membership gates access.

### 4.3 Budget domain
- `GET/POST/PATCH/DELETE /trips/{id}/budget/...` categories + items + totals.

### 4.4 Decisions domain
- `POST /trips/{id}/decisions` create poll, `POST .../vote`, `GET .../results`, promote to config.

### 4.5 Expense domain
- Record, split, participants, settlement (later slice).

### 4.6 Fund (exists) — stays as the financial heart; already has
- `POST /funds`, `POST /funds/{id}/members`, `GET /funds/{id}`, `GET /funds/{id}/progress`, `POST /funds/{id}/contributions`, `POST /webhooks/polar`.

## 5. PWA design (Fase 1)

### 5.1 Stack
Next.js (App Router) + TypeScript + Tailwind CSS + shadcn/ui + TanStack Query + React Hook Form + Zod. PWA installable via `next-pwa` (or manual manifest+SW if `next-pwa` is deprecated).

### 5.2 Routes
```
/                     → redirect (to /login or /trips)
/login  /register     → auth screens (real backend from day one)
/trips                → my trips list
/trips/new            → create trip
/trips/[id]           → trip dashboard / overview
/trips/[id]/decisions → polls, votes, results
/trips/[id]/budget    → categories, items, progress
/trips/[id]/fund      → financial heart: goal, per-person, ledger, contribute
/trips/[id]/payments  → authorize method, contributions, history
/trips/[id]/expenses  → record, split, settlement
/trips/[id]/itinerary → placeholder (later)
```
Protected layout: validates session; redirects to `/login` when unauthenticated.

### 5.3 Data layer
- `pwa/lib/api.ts` — typed fetch client, base URL from `NEXT_PUBLIC_API_URL`, error normalization.
- `pwa/lib/api/auth.ts|trip.ts|budget.ts|decision.ts|fund.ts|expense.ts` — one module per domain.
- TanStack Query for caching/refetch; server components vs client components chosen pragmatically (data fetching in client components via React Query for the interactive app).
- Where a backend endpoint does not yet exist, the domain module stays behind the same typed interface but the underlying call is marked pending until Fase 2 wires it; UI degrades gracefully (shows "not available yet" with the interface fully typed), NOT a fake permanent mock.

### 5.4 Auth wiring
- `pwa/lib/auth.ts` — session context + guard; calls real backend auth endpoints.
- Route guard wraps trip routes.
- On 401 from any API call, trigger refresh → retry → else redirect to login.

### 5.5 Payments (Polar embed)
- On `/trips/[id]/fund` and `/trips/[id]/payments`: tokenize card with **Polar customerSession + PaymentMethod.createInline** (client-side).
- Result `payment_method_id` is sent to `POST /funds/{id}/contributions`.
- Shows `processing` until the `order.paid` webhook flips the contribution to SUCCEEDED (server drives state; PWA refetches progress).

### 5.6 PWA installable
- `manifest.webmanifest`, icons, theme color, `apple-touch-icon`.
- Service worker for basic installability + offline shell (push later).

## 6. UX principles (from IA spec)
- Minimal cognitive load: show status not just state ("82% funded", "3 pending").
- Money is explicit: expected vs collected vs pending always distinguishable.
- Irreversible financial actions require confirmation + audit.
- Mobile-first (home-screen installable), runs iOS/Android/desktop.

## 7. Error handling & security
- Auth errors surfaced inline (invalid credentials, email taken).
- API 401 → refresh flow → logout on failure.
- Financial actions: confirm dialog + loading state + result.
- No secrets in the client; backend holds secrets/env.

## 8. Testing
- Phase 0: Go tests for auth (register/login/refresh rotation/revocation/argon2id).
- Fase 1: component + integration tests for auth flow; React Query + Playwright for critical journeys (register → create trip → create fund → contribute).
- Continuous: `go test ./...` + `pnpm test` + `pnpm build` green before merge.

## 9. Non-goals (this design)
- Native iOS/Android apps (post-MVP evolution).
- Social layer, chat, notifications push beyond installability.
- Off-session charging enablement (blocked on Polar support opt-in; surfaced separately).
- Production deployment of the new domains — endpoints are built and tested, not deployed here.

## 10. Delivery order
1. **Fase 0**: backend auth (domain+repo+handlers+tests) — unblocks PWA auth.
2. **Fase 1**: PWA scaffold + auth + trips + fund+payments (real) first; then budget, decisions, expenses screens against typed-but-pending integrations.
3. **Fase 2**: backend completion for trips, budget, decisions, expenses; flip pending integrations to real; end-to-end verify.
