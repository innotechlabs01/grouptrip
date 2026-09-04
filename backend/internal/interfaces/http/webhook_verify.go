package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Polar webhooks follow the Standard Webhooks spec (HMAC-SHA256), with a Polar-specific
// signing-key derivation. See docs/platform/01-architecture/fund-spec.md and the Polar
// delivery docs. This verifier is zero-dependency and tested against known vectors.

const (
	webhookTolerance = 5 * time.Minute

	headerWebhookID        = "webhook-id"
	headerWebhookTimestamp = "webhook-timestamp"
	headerWebhookSignature = "webhook-signature"
)

// ErrInvalidSignature reports a webhook that failed the signature/timestamp check.
var ErrInvalidSignature = errors.New("webhook: invalid signature")

// verifyPolarWebhook verifies a Polar webhook using the Standard Webhooks scheme.
//
// The signature is HMAC-SHA256 over the exact byte content:
//
//	webhook-id + "." + webhook-timestamp + "." + rawBody
//
// where rawBody is the request body bytes verbatim (never re-encoded). The signature
// header is a space-delimited list of "v1,<base64-sig>" entries; any match passes
// (supports zero-downtime secret rotation). The timestamp must fall within 5 minutes.
//
// Polar signs with the literal UTF-8 bytes of the full secret string (prefix included,
// e.g. "setup_..." / "polar_whs_..."), deviating from the generic spec's
// base64decode(secret-without-prefix) derivation. We therefore try BOTH derivations to
// stay correct today and forward-compatible if Polar moves to the spec key.
func verifyPolarWebhook(secret string, body []byte, headers http.Header) error {
	id := headers.Get(headerWebhookID)
	tsStr := headers.Get(headerWebhookTimestamp)
	sigHeader := headers.Get(headerWebhookSignature)
	if id == "" || tsStr == "" || sigHeader == "" {
		return ErrInvalidSignature
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return ErrInvalidSignature
	}
	if d := time.Since(time.Unix(ts, 0)); d < -webhookTolerance || d > webhookTolerance {
		return ErrInvalidSignature
	}

	// Content to sign: id . timestamp . rawBody (bytes verbatim).
	content := make([]byte, 0, len(id)+1+len(tsStr)+1+len(body))
	content = append(content, id...)
	content = append(content, '.')
	content = append(content, tsStr...)
	content = append(content, '.')
	content = append(content, body...)

	// Candidate signing keys; accept if any derivation yields a matching signature.
	keys := [][]byte{[]byte(secret)}
	if alt := specDerivedKey(secret); alt != nil {
		keys = append(keys, alt)
	}

	for _, key := range keys {
		mac := hmac.New(sha256.New, key)
		mac.Write(content)
		expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		for _, part := range strings.Fields(sigHeader) {
			v, sig, ok := strings.Cut(part, ",")
			if !ok || v != "v1" {
				continue
			}
			if hmac.Equal([]byte(sig), []byte(expected)) {
				return nil
			}
		}
	}
	return ErrInvalidSignature
}

// specDerivedKey implements the generic Standard Webhooks key derivation that Polar may
// adopt: base64-decode the secret after stripping the prefix. Returns nil when the secret
// has no recognizable prefix or cannot be decoded (in which case only the literal key applies).
func specDerivedKey(secret string) []byte {
	for _, prefix := range []string{"whsec_", "polar_whs_"} {
		if rest, ok := strings.CutPrefix(secret, prefix); ok {
			decoded, err := base64.StdEncoding.DecodeString(rest)
			if err != nil {
				return nil
			}
			return decoded
		}
	}
	return nil
}
