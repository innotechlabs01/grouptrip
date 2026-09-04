package fundrepo

import (
	"errors"
	"sync"

	"github.com/frg/grouptrip/internal/domain/fund"
)

// MemoryRepo is an in-memory FundRepository for tests and early scaffolding.
// NOT for production — swap for the Turso-backed repository (sqlite.go).
type MemoryRepo struct {
	mu    sync.RWMutex
	funds map[string]*fund.Fund
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{funds: make(map[string]*fund.Fund)}
}

func (r *MemoryRepo) Save(f *fund.Fund) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funds[f.ID] = f
	return nil
}

func (r *MemoryRepo) Load(id string) (*fund.Fund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.funds[id]
	if !ok {
		return nil, errors.New("fundrepo: fund not found")
	}
	return f, nil
}
