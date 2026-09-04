package fund

import (
	"testing"
	"time"
)

func TestRecommendSingleForShortHorizon(t *testing.T) {
	goal, _ := NewMoney(1000000, "cop")
	// trip in ~2 weeks
	trip := time.Now().AddDate(0, 0, 14)
	s, err := Recommend(Input{
		TripDate:         trip,
		Goal:             goal,
		ParticipantCount: 4,
		Today:            time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Frequency != FrequencySingle {
		t.Fatalf("expected SINGLE for short horizon, got %s", s.Frequency)
	}
	// per-person = 250000
	if s.Amount.Amount() != 250000 {
		t.Fatalf("expected amount 250000, got %d", s.Amount.Amount())
	}
}

func TestRecommendMonthlyForMediumHorizon(t *testing.T) {
	goal, _ := NewMoney(1200000, "cop")
	trip := time.Now().AddDate(0, 2, 0)
	s, err := Recommend(Input{
		TripDate:         trip,
		Goal:             goal,
		ParticipantCount: 4,
		Today:            time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Frequency != FrequencyMonthly {
		t.Fatalf("expected MONTHLY for medium horizon, got %s", s.Frequency)
	}
	// per-person 300000 / 2 months = 150000
	if s.Amount.Amount() != 150000 {
		t.Fatalf("expected amount 150000, got %d", s.Amount.Amount())
	}
}

func TestRecommendAccountsForCollected(t *testing.T) {
	goal, _ := NewMoney(1000000, "cop")
	trip := time.Now().AddDate(0, 2, 0)
	collected, _ := NewMoney(600000, "cop")
	s, err := Recommend(Input{
		TripDate:         trip,
		Goal:             goal,
		ParticipantCount: 4,
		AlreadyCollected: collected,
		Today:            time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// remaining 400000 / 4 = 100000 per person
	if s.PerPersonTotal.Amount() != 100000 {
		t.Fatalf("expected per-person total 100000, got %d", s.PerPersonTotal.Amount())
	}
}
