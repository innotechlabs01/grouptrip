package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// computeSignature reproduces the Polar (literal-secret) HMAC for test vectors.
func computeSignature(secret, id, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyPolarWebhook_ValidLiteralKey(t *testing.T) {
	const secret = "polar_whs_testsecret_123456"
	id := "evt_1"
	ts := time.Now().Unix()
	body := []byte(`{"type":"order.paid","data":{"order":{"id":"ord_1","status":"paid"}}}`)

	hdr := make(http.Header)
	hdr.Set(headerWebhookID, id)
	hdr.Set(headerWebhookTimestamp, strconvFormat(ts))
	hdr.Set(headerWebhookSignature, "v1,"+computeSignature(secret, id, strconvFormat(ts), body))

	if err := verifyPolarWebhook(secret, body, hdr); err != nil {
		t.Fatalf("expected valid signature, got: %v", err)
	}
}

func TestVerifyPolarWebhook_InvalidSignature(t *testing.T) {
	const secret = "polar_whs_testsecret_123456"
	id := "evt_1"
	ts := time.Now().Unix()
	body := []byte(`{"type":"order.paid"}`)

	hdr := make(http.Header)
	hdr.Set(headerWebhookID, id)
	hdr.Set(headerWebhookTimestamp, strconvFormat(ts))
	hdr.Set(headerWebhookSignature, "v1,AAAA") // wrong

	if err := verifyPolarWebhook(secret, body, hdr); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got: %v", err)
	}
}

func TestVerifyPolarWebhook_TamperedBody(t *testing.T) {
	const secret = "polar_whs_testsecret_123456"
	id := "evt_1"
	ts := time.Now().Unix()
	body := []byte(`{"type":"order.paid"}`)
	tampered := []byte(`{"type":"order.paid","extra":true}`)

	hdr := make(http.Header)
	hdr.Set(headerWebhookID, id)
	hdr.Set(headerWebhookTimestamp, strconvFormat(ts))
	hdr.Set(headerWebhookSignature, "v1,"+computeSignature(secret, id, strconvFormat(ts), body))

	// Signature computed over `body`, but we verify against `tampered` → must fail.
	if err := verifyPolarWebhook(secret, tampered, hdr); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature on tampered body, got: %v", err)
	}
}

func TestVerifyPolarWebhook_StaleTimestamp(t *testing.T) {
	const secret = "polar_whs_testsecret_123456"
	id := "evt_1"
	old := time.Now().Add(-10 * time.Minute).Unix()
	body := []byte(`{"type":"order.paid"}`)

	hdr := make(http.Header)
	hdr.Set(headerWebhookID, id)
	hdr.Set(headerWebhookTimestamp, strconvFormat(old))
	hdr.Set(headerWebhookSignature, "v1,"+computeSignature(secret, id, strconvFormat(old), body))

	if err := verifyPolarWebhook(secret, body, hdr); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature on stale timestamp, got: %v", err)
	}
}

func TestVerifyPolarWebhook_MissingHeaders(t *testing.T) {
	const secret = "polar_whs_testsecret_123456"
	if err := verifyPolarWebhook(secret, []byte(`{}`), make(http.Header)); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature with missing headers, got: %v", err)
	}
}

func TestVerifyPolarWebhook_MultipleSignatures_AnyMatch(t *testing.T) {
	const secret = "polar_whs_testsecret_123456"
	id := "evt_1"
	ts := time.Now().Unix()
	body := []byte(`{"type":"order.paid"}`)
	tsStr := strconvFormat(ts)

	hdr := make(http.Header)
	hdr.Set(headerWebhookID, id)
	hdr.Set(headerWebhookTimestamp, tsStr)
	// Carry an old signature plus the current one (rotation).
	sig := "v1," + computeSignature("polar_whs_old_0000000000000000", id, tsStr, body) +
		" v1," + computeSignature(secret, id, tsStr, body)
	hdr.Set(headerWebhookSignature, sig)

	if err := verifyPolarWebhook(secret, body, hdr); err != nil {
		t.Fatalf("expected one of multiple signatures to match, got: %v", err)
	}
}

func TestVerifyPolarWebhook_SpecDerivedKey(t *testing.T) {
	// Forward-compat: if Polar moves to the generic spec key (base64decode of the raw
	// key material after the prefix), the dual-try derivation must still validate.
	const specSecret = "whsec_dGhpc19pc19iYXNlNjRfcmF3LWtleQ==" // base64 of "this_is_base64_raw-key"
	id := "evt_1"
	ts := time.Now().Unix()
	body := []byte(`{"type":"order.paid"}`)
	tsStr := strconvFormat(ts)

	// Spec key = base64decode(rest after prefix).
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(specSecret, "whsec_"))
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, raw)
	mac.Write([]byte(id + "." + tsStr + "."))
	mac.Write(body)
	specSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	hdr := make(http.Header)
	hdr.Set(headerWebhookID, id)
	hdr.Set(headerWebhookTimestamp, tsStr)
	hdr.Set(headerWebhookSignature, "v1,"+specSig)

	if err := verifyPolarWebhook(specSecret, body, hdr); err != nil {
		t.Fatalf("expected spec-derived key to validate, got: %v", err)
	}
}

func strconvFormat(i int64) string {
	return strconv.FormatInt(i, 10)
}
