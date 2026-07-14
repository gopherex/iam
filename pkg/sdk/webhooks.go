package sdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ErrWebhookMalformedHeaders = errors.New("iam sdk: malformed webhook headers")
	ErrWebhookStaleTimestamp   = errors.New("iam sdk: webhook timestamp outside allowed skew")
	ErrWebhookInvalidSignature = errors.New("iam sdk: invalid webhook signature")
	ErrWebhookMalformedBody    = errors.New("iam sdk: malformed webhook body")
)

// WebhookVerifierConfig configures Standard Webhooks verification. Replay
// storage intentionally belongs to the consumer: persist WebhookEvent.ID after
// Verify succeeds when a consumer needs one-time processing semantics.
type WebhookVerifierConfig struct {
	SigningSecret string
	MaxSkew       time.Duration
	Now           func() time.Time
}

// WebhookVerifier verifies IAM's Standard Webhooks request signatures.
type WebhookVerifier struct {
	secret  []byte
	maxSkew time.Duration
	now     func() time.Time
}

// WebhookEvent is IAM's public webhook envelope. Data remains raw so callers
// can decode it into the type appropriate for Event.Type.
type WebhookEvent struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	OccurredAt  time.Time       `json:"occurred_at"`
	ProjectID   string          `json:"project_id"`
	Environment string          `json:"environment"`
	Data        json.RawMessage `json:"data"`
}

// SessionRevokedData is the stable data schema for a session.revoked event.
type SessionRevokedData struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

// DecodeData decodes the event data into dst after Verify has authenticated the
// envelope. dst is normally a pointer to an event-specific type.
func (e *WebhookEvent) DecodeData(dst any) error {
	if err := json.Unmarshal(e.Data, dst); err != nil {
		return fmt.Errorf("%w: data: %w", ErrWebhookMalformedBody, err)
	}
	return nil
}

// SessionRevokedData decodes a session.revoked event and validates its required
// fields and project consistency with the envelope.
func (e *WebhookEvent) SessionRevokedData() (SessionRevokedData, error) {
	if e.Type != "session.revoked" {
		return SessionRevokedData{}, fmt.Errorf("%w: expected session.revoked, got %q", ErrWebhookMalformedBody, e.Type)
	}
	var data SessionRevokedData
	if err := e.DecodeData(&data); err != nil {
		return SessionRevokedData{}, err
	}
	if data.SessionID == "" || data.UserID == "" || data.ProjectID == "" || data.ProjectID != e.ProjectID {
		return SessionRevokedData{}, fmt.Errorf("%w: incomplete session.revoked data", ErrWebhookMalformedBody)
	}
	return data, nil
}

// NewWebhookVerifier constructs a verifier for the signing secret returned by
// IAM. IAM emits Standard Webhooks whsec_ secrets; raw legacy secrets are also
// accepted so an existing subscription can migrate without a delivery gap.
func NewWebhookVerifier(config WebhookVerifierConfig) (*WebhookVerifier, error) {
	secret, err := decodeWebhookSecret(config.SigningSecret)
	if err != nil {
		return nil, err
	}
	maxSkew := config.MaxSkew
	if maxSkew == 0 {
		maxSkew = 5 * time.Minute
	}
	if maxSkew < 0 {
		return nil, fmt.Errorf("iam sdk: webhook max skew must not be negative")
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &WebhookVerifier{secret: secret, maxSkew: maxSkew, now: now}, nil
}

// Verify authenticates the Standard Webhooks headers before decoding body.
func (v *WebhookVerifier) Verify(headers http.Header, body []byte) (*WebhookEvent, error) {
	id, timestamp, signatures, err := parseWebhookHeaders(headers)
	if err != nil {
		return nil, err
	}
	if delta := v.now().Sub(time.Unix(timestamp, 0)); delta > v.maxSkew || delta < -v.maxSkew {
		return nil, ErrWebhookStaleTimestamp
	}
	mac := hmac.New(sha256.New, v.secret)
	_, _ = fmt.Fprintf(mac, "%s.%d.", id, timestamp)
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	valid := false
	for _, signature := range signatures {
		decoded, err := base64.StdEncoding.DecodeString(signature)
		if err == nil && hmac.Equal(expected, decoded) {
			valid = true
		}
	}
	if !valid {
		return nil, ErrWebhookInvalidSignature
	}
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWebhookMalformedBody, err)
	}
	if event.ID == "" || event.ID != id || event.Type == "" || event.ProjectID == "" {
		return nil, fmt.Errorf("%w: envelope does not match webhook-id", ErrWebhookMalformedBody)
	}
	return &event, nil
}

func decodeWebhookSecret(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("iam sdk: webhook signing secret is required")
	}
	if encoded, ok := strings.CutPrefix(value, "whsec_"); ok {
		secret, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(secret) == 0 {
			return nil, fmt.Errorf("iam sdk: invalid webhook signing secret")
		}
		return secret, nil
	}
	return []byte(value), nil
}

func parseWebhookHeaders(headers http.Header) (string, int64, []string, error) {
	id := strings.TrimSpace(headers.Get("webhook-id"))
	timestampText := strings.TrimSpace(headers.Get("webhook-timestamp"))
	if id == "" || timestampText == "" {
		return "", 0, nil, ErrWebhookMalformedHeaders
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil || timestamp < 0 {
		return "", 0, nil, ErrWebhookMalformedHeaders
	}
	var signatures []string
	for _, item := range strings.Fields(headers.Get("webhook-signature")) {
		version, signature, ok := strings.Cut(item, ",")
		if ok && version == "v1" && signature != "" {
			signatures = append(signatures, signature)
		}
	}
	if len(signatures) == 0 {
		return "", 0, nil, ErrWebhookMalformedHeaders
	}
	return id, timestamp, signatures, nil
}
