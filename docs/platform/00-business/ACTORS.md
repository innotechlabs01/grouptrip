# Actors & Roles

## Traveler
Regular user.
- Create trips
- Join trips
- Vote on decisions
- Contribute money
- Pay
- Record expenses
- View itinerary

## Trip Owner
Trip creator.
- Manage participants
- Define rules
- Create budget
- Configure fund
- Approve certain operations
- Manage reservations

## Trip Admin
Delegated person who helps administer the trip.

## Payment Participant
Participant with a configured payment method.
- Authorize contributions (one-time or recurring)
- Approve certain expenses

## Provider
External provider: hotel, tour, transport, restaurant, experience, agency.

## Platform Admin
Global administration: users, trips, payments, providers, disputes, metrics, configuration.

## Authorization Models
- Payment submission only with explicit user authorization
- RBAC for trip-level roles (Owner > Admin > Traveler)
- Every financial action auditable and attributed to an actor
