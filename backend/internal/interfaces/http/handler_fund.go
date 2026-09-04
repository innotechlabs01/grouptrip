package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/frg/grouptrip/internal/domain/fund"
)

// createFund handles POST /funds.
// Body: {"trip_id": "...", "goal_amount": 1000, "goal_currency": "usd"}
func (s *Server) createFund(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TripID       string `json:"trip_id"`
		GoalAmount   int64  `json:"goal_amount"`
		GoalCurrency string `json:"goal_currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	goal, err := fund.NewMoney(req.GoalAmount, req.GoalCurrency)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generate a simple ID (in production, use a UUID or nanoid).
	// For now, use a timestamp-based ID for uniqueness.
	id := generateID()
	// Construct via NewFund to enforce write-path invariants (I-1: goal > 0).
	f, err := fund.NewFund(id, req.TripID, goal)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.repo.Save(f); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save fund")
		return
	}

	writeJSON(w, http.StatusCreated, fundToJSON(f))
}

// addMember handles POST /funds/{id}/members.
// Body: {"user_id": "..."}
func (s *Server) addMember(w http.ResponseWriter, r *http.Request) {
	fundID := r.PathValue("id")
	if fundID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing fund id")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f, err := s.repo.Load(fundID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "fund not found")
		return
	}

	if _, err := f.AddMember(req.UserID); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.repo.Save(f); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save fund")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user_id":           req.UserID,
		"per_person_target": f.Members[len(f.Members)-1].PerPersonTarget.Amount(),
	})
}

// getFund handles GET /funds/{id}.
func (s *Server) getFund(w http.ResponseWriter, r *http.Request) {
	fundID := r.PathValue("id")
	if fundID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing fund id")
		return
	}

	f, err := s.repo.Load(fundID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "fund not found")
		return
	}

	writeJSON(w, http.StatusOK, fundToJSON(f))
}

// --- helpers ---

// fundToJSON converts a Fund aggregate to a JSON-friendly map.
func fundToJSON(f *fund.Fund) map[string]interface{} {
	members := make([]map[string]interface{}, len(f.Members))
	for i, m := range f.Members {
		members[i] = map[string]interface{}{
			"user_id":           m.UserID,
			"per_person_target": m.PerPersonTarget.Amount(),
		}
	}

	return map[string]interface{}{
		"id":               f.ID,
		"trip_id":          f.TripID,
		"goal_amount":      f.Goal.Amount(),
		"goal_currency":    f.Goal.Currency(),
		"status":           string(f.Status),
		"members":          members,
		"collected":        f.Collected().Amount(),
		"pending":          f.Pending().Amount(),
		"failed":           f.Failed().Amount(),
		"goal_adjustments": f.GoalAdjustments,
		"created_at":       f.CreatedAt.Format(time.RFC3339),
		"updated_at":       f.UpdatedAt.Format(time.RFC3339),
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{"error": message})
}

// generateID creates a simple unique ID based on timestamp and random component.
// In production, use a proper UUID library.
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}
