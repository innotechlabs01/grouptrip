# Database

## Engine
PostgreSQL

## Entities

### Identity
```
users
user_profiles
contacts
payment_profiles
preferences
```

### Trips
```
trips
trip_members
trip_roles
trip_invitations
trip_settings
```

### Decisions
```
decisions
decision_options
decision_votes
```

### Budget
```
budgets
budget_categories
budget_items
```

### Fund (financial heart) — never fold complexity into one table
```
funds
fund_members
contribution_plans
contributions
fund_ledger
```

### Payments
```
payment_customers
payment_methods
payment_intents
payment_attempts
refunds
```

### Expenses
```
expenses
expense_participants
settlements
```

### Travel
```
reservations
activities
itinerary_entries
tasks
```

### Notifications & Audit
```
notifications
audit_logs
```

## Rule
Never cram the full financial complexity into a single table.
The Fund Ledger is a separate, append-only, auditable record.
