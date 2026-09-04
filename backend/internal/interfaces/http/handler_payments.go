package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/infrastructure/payments"
)

// handleContribute handles POST /funds/{id}/contributions.
// Body: {contribution_id, plan_id, product_id, customer_email, customer_id,
//
//	payment_method_id, amount, currency, description}
func (s *Server) handleContribute(w http.ResponseWriter, r *http.Request) {
	fundID := r.PathValue("id")
	if fundID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing fund id")
		return
	}

	var req struct {
		ContributionID  string `json:"contribution_id"`
		PlanID          string `json:"plan_id"`
		ProductID       string `json:"product_id"`
		CustomerEmail   string `json:"customer_email"`
		CustomerID      string `json:"customer_id"`
		PaymentMethodID string `json:"payment_method_id"`
		Amount          int64  `json:"amount"`
		Currency        string `json:"currency"`
		Description     string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields.
	if req.ContributionID == "" {
		writeJSONError(w, http.StatusBadRequest, "contribution_id is required")
		return
	}
	if req.ProductID == "" {
		writeJSONError(w, http.StatusBadRequest, "product_id is required")
		return
	}
	if req.CustomerEmail == "" {
		writeJSONError(w, http.StatusBadRequest, "customer_email is required")
		return
	}
	if req.Amount <= 0 {
		writeJSONError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if req.Currency == "" {
		writeJSONError(w, http.StatusBadRequest, "currency is required")
		return
	}

	in := commands.ContributeInput{
		ContributionID:  req.ContributionID,
		FundID:          fundID,
		PlanID:          req.PlanID,
		ProductID:       req.ProductID,
		CustomerEmail:   req.CustomerEmail,
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Description:     req.Description,
	}

	if err := s.contribute.Execute(in); err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "load fund"):
			writeJSONError(w, http.StatusNotFound, "fund not found")
		case errors.Is(err, payments.ErrCardDeclined):
			writeJSONError(w, http.StatusPaymentRequired, "card declined")
		case strings.Contains(msg, "commands:"):
			writeJSONError(w, http.StatusBadRequest, msg)
		default:
			writeJSONError(w, http.StatusInternalServerError, "failed to process contribution")
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"contribution_id": in.ContributionID,
		"status":          "processing",
	})
}

// fundProgress handles GET /funds/{id}/progress.
func (s *Server) fundProgress(w http.ResponseWriter, r *http.Request) {
	fundID := r.PathValue("id")
	if fundID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing fund id")
		return
	}

	prog, err := s.progress.Execute(fundID)
	if err != nil {
		if strings.Contains(err.Error(), "load fund") || strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, "fund not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load fund progress")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"fund_id":           fundID,
		"goal_amount":       prog.Goal.Amount(),
		"goal_currency":     prog.Goal.Currency(),
		"collected":         prog.Collected.Amount(),
		"pending":           prog.Pending.Amount(),
		"failed":            prog.Failed.Amount(),
		"per_person_target": prog.PerPersonTarget.Amount(),
		"percent":           prog.Percent,
		"status":            string(prog.Status),
	})
}
