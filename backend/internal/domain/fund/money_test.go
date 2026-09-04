package fund

import "testing"

func TestMoneyRejectsNonPositive(t *testing.T) {
	if _, err := NewMoney(0, "cop"); err == nil {
		t.Fatal("expected error for zero amount")
	}
	if _, err := NewMoney(-5, "cop"); err == nil {
		t.Fatal("expected error for negative amount")
	}
	if _, err := NewMoney(100, ""); err == nil {
		t.Fatal("expected error for empty currency")
	}
}

func TestMoneyAddSameCurrency(t *testing.T) {
	a, _ := NewMoney(100, "cop")
	b, _ := NewMoney(250, "cop")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Amount() != 350 {
		t.Fatalf("expected 350, got %d", sum.Amount())
	}
}

func TestMoneyAddMismatch(t *testing.T) {
	a, _ := NewMoney(100, "cop")
	b, _ := NewMoney(100, "usd")
	if _, err := a.Add(b); err == nil {
		t.Fatal("expected currency mismatch error")
	}
}

func TestMoneyDivByPositive(t *testing.T) {
	a, _ := NewMoney(1000, "cop")
	d, err := a.Div(4)
	if err != nil {
		t.Fatal(err)
	}
	if d.Amount() != 250 {
		t.Fatalf("expected 250, got %d", d.Amount())
	}
}

func TestMoneyDivByZero(t *testing.T) {
	a, _ := NewMoney(1000, "cop")
	if _, err := a.Div(0); err == nil {
		t.Fatal("expected error for zero divisor")
	}
}

func TestMoneyGte(t *testing.T) {
	a, _ := NewMoney(100, "cop")
	b, _ := NewMoney(50, "cop")
	ok, err := a.Gte(b)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected 100 >= 50")
	}
}

func TestFrequencyValidation(t *testing.T) {
	if err := FrequencySingle.Validate(); err != nil {
		t.Fatal("SINGLE should be valid")
	}
	if err := Frequency("NOPE").Validate(); err == nil {
		t.Fatal("NOPE should be invalid")
	}
}
