# State Machines

## Trip
```
DRAFT
  │ create/invite
  ▼
PLANNING
  │ decisions resolved + budget set
  ▼
FUNDING
  │ fund reaches goal (or ready flag)
  ▼
READY
  │ departure
  ▼
ACTIVE
  │ return
  ▼
COMPLETED
        │
        └────────── CANCELLED (any stage)
```

## Decision
```
OPEN (poll)
  │ votes
  ▼
VOTING
  │ deadline / closing
  ▼
RESOLVED (result)
  │ promote (owner/admin)
  ▼
APPLIED → TRIP CONFIGURATION
```

## Contribution Plan
```
ACTIVE
  │ charge
  ▼
PROCESSING
  ├── success → PAID
  ├── fail (retry) → PENDING
  └── fail (final) → FAILED
```

## Payment
```
PENDING
  ▼
AUTHORIZED
  ▼
PROCESSING
  ├── SUCCEEDED
  ├── FAILED
  ├── REFUNDED (from succeeded)
  └── CANCELLED
```

## Reservation
```
PLANNED → QUOTED → RESERVED → PAID → COMPLETED
                              │
                              └── CANCELLED
```

## Fund
```
OPEN (goal set)
  │ contributions
  ▼
ACTIVE (funding in progress)
  │ goal reached
  ▼
FUNDED
```
