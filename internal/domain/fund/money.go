package fund

import (
	"errors"
	"fmt"
)

// Money represents an amount of money in the smallest currency unit (e.g. cents/pesos).
// Using an int64 avoids floating-point errors entirely.
type Money struct {
	amount   int64
	currency string
}

// NewMoney creates a Money value, rejecting non-positive amounts and empty currency.
func NewMoney(amount int64, currency string) (Money, error) {
	if amount <= 0 {
		return Money{}, errors.New("fund: amount must be positive")
	}
	if currency == "" {
		return Money{}, errors.New("fund: currency must not be empty")
	}
	return Money{amount: amount, currency: currency}, nil
}

// ZeroMoney returns a zero-valued Money with the given currency.
func ZeroMoney(currency string) Money {
	return Money{amount: 0, currency: currency}
}

// Amount returns the amount in smallest currency units.
func (m Money) Amount() int64 { return m.amount }

// Currency returns the ISO 4217 currency code (lowercase).
func (m Money) Currency() string { return m.currency }

// Add returns a new Money representing the sum, requiring same currency.
func (m Money) Add(o Money) (Money, error) {
	if m.currency != o.currency {
		return Money{}, fmt.Errorf("fund: currency mismatch %s != %s", m.currency, o.currency)
	}
	return Money{amount: m.amount + o.amount, currency: m.currency}, nil
}

// Gte reports whether m >= o, requiring same currency.
func (m Money) Gte(o Money) (bool, error) {
	if m.currency != o.currency {
		return false, fmt.Errorf("fund: currency mismatch %s != %s", m.currency, o.currency)
	}
	return m.amount >= o.amount, nil
}

// Div returns m divided by n (per-person target), same currency. n must be > 0.
func (m Money) Div(n int) (Money, error) {
	if n <= 0 {
		return Money{}, errors.New("fund: division by non-positive participant count")
	}
	return Money{amount: m.amount / int64(n), currency: m.currency}, nil
}

// String renders for display/logging.
func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}
