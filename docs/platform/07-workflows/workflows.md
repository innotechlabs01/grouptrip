# Workflows

## Trip Creation
1. Register / login
2. Create trip (name, dates, destination draft)
3. Invite members
4. Members accept invitation → trip_member + role

## Funding Workflow
1. Define fund goal (from budget target or manual)
2. Set participants
3. System computes per-person target
4. Owner/admin configures contribution plan / members authorize
5. Auto-recommend strategy (single / monthly / biweekly / weekly)
6. Contributions charged via provider
7. Ledger updates: expected / collected / pending / failed
8. Trip progress recomputed (event-driven)

## Payment Workflow
1. User authorizes payment method (tokenized at provider)
2. PaymentIntent created
3. Attempt → PROCESSING
4. Provider webhook (signed, idempotent) → SUCCEEDED / FAILED
5. Ledger + progress updated
6. Refund path from SUCCEEDED

## Expense & Settlement Workflow
1. Member records expense (category, amount, split mode)
2. Split: individual / equal / percentage / fund
3. Assign participants + weights
4. At close, compute min-transfer settlement
