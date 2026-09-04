package fund

import "errors"

// Frequency describes how often a ContributionPlan charges.
type Frequency string

const (
	FrequencySingle   Frequency = "SINGLE"
	FrequencyMonthly  Frequency = "MONTHLY"
	FrequencyBiweekly Frequency = "BIWEEKLY"
	FrequencyWeekly   Frequency = "WEEKLY"
)

// IsValid reports whether f is a known Frequency.
func (f Frequency) IsValid() bool {
	switch f {
	case FrequencySingle, FrequencyMonthly, FrequencyBiweekly, FrequencyWeekly:
		return true
	default:
		return false
	}
}

// Validate returns an error for an unknown Frequency.
func (f Frequency) Validate() error {
	if !f.IsValid() {
		return errors.New("fund: invalid frequency")
	}
	return nil
}
