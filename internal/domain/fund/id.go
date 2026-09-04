package fund

import "crypto/rand"
import "encoding/hex"

// newID returns a random hex identifier (for ledger entries, contributions, etc.).
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Deterministic fallback is unsafe; a random source failure is treated as fatal
		// only in tests of adversarial paths. In practice rand.Read never fails on the
		// supported platforms. We panic to avoid silently using an unsafe ID.
		panic("fund: cannot generate id: " + err.Error())
	}
	return hex.EncodeToString(b)
}
