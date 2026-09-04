package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/infrastructure/contribrepo"
)

// orderPaidWebhook is the flexible shape of a Polar order.paid event.
// It tolerates missing optional fields defensively.
type orderPaidWebhook struct {
	Type string `json:"type"`
	Data struct {
		Order struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
	} `json:"data"`
}

// webhookPolar handles POST /webhooks/polar.
// Expected payload: {"type":"order.paid","data":{"order":{"id":"<order-id>","status":"paid"}}}
// The Standard Webhooks signature is verified first (HTTPS transport + HMAC-SHA256);
// without a configured secret the endpoint fails closed (never processes unverified events).
func (s *Server) webhookPolar(w http.ResponseWriter, r *http.Request) {
	if s.contribs == nil || s.webhookSecret == "" {
		writeJSONError(w, http.StatusInternalServerError, "webhook not configured")
		return
	}

	// Read the raw body BEFORE parsing: the signature is over the exact bytes received.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := verifyPolarWebhook(s.webhookSecret, body, r.Header); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	var ev orderPaidWebhook
	if err := json.Unmarshal(body, &ev); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Non-order.paid events are ignored (200).
	if ev.Type != "order.paid" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ignored"})
		return
	}

	orderID := ev.Data.Order.ID
	if orderID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing order id")
		return
	}

	handler := commands.HandleOrderPaid{
		Funds:  s.repo,
		Contrs: s.contribs,
		Events: events.NoopSink{},
	}

	err = handler.Execute(commands.OrderPaidInput{
		OrderID:    orderID,
		OccurredAt: time.Now(),
	})
	if err != nil {
		// Unknown order → 404
		if errors.Is(err, contribrepo.ErrNotFound) || isNotFoundSentence(err) {
			writeJSONError(w, http.StatusNotFound, "order unknown")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to process webhook: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

// isNotFoundSentence detects wrapped not-found errors by message.
func isNotFoundSentence(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "contribrepo: not found"
}
