package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

type securityEmitter struct {
	db    *DB
	inner Emitter
}

// NewSecurityEmitter adds sanitized security events in the mutation's transaction.
// The inner emitter is used for resulting notifications to prevent recursion.
func NewSecurityEmitter(db *DB, inner Emitter) Emitter { return &securityEmitter{db: db, inner: inner} }

func (e *securityEmitter) Emit(ctx context.Context, event domain.Event) error {
	if !strings.HasPrefix(event.Type, "security.") {
		if err := NewPgSecurity(e.db, e.inner).observe(ctx, event); err != nil {
			return securityStoreError(err)
		}
	}

	return e.inner.Emit(ctx, event)
}

type securityOldContactKey struct{}

//nolint:funlen,cyclop,gocyclo,gocognit // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) observe(ctx context.Context, event domain.Event) error {
	switch event.Type {
	case securitySessionCreated,
		securityTokenReuseDetected,
		securitySessionDeviceMismatch,
		"password.changed",
		"password.reset",
		"email.changed",
		"phone.changed",
		"mfa.factor.activated",
		"mfa.factor.removed",
		"mfa.recovery_codes.generated",
		"webauthn.credential.registered",
		"webauthn.credential.removed",
		"identity.linked",
		"identity.unlinked":
	default:
		return nil
	}

	scope := domain.SecurityScope{
		ProjectID:   event.ProjectID,
		Environment: api.EnvironmentFromContext(ctx),
	}
	if scope.Environment == "" {
		scope.Environment = event.Environment
	}

	scope, err := s.scope(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	policy, err := s.Policy(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	if policy.Mode == securityDisabled && (event.Type == securitySessionCreated ||
		event.Type == securityTokenReuseDetected || event.Type == securitySessionDeviceMismatch) {
		return nil
	}

	raw, err := json.Marshal(event.Payload)
	if err != nil {
		return securityStoreError(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return securityStoreError(err)
	}

	scope.AccountID = securityPayloadString(payload, securityAccountID, "user_id", "AccountID")
	sessionID := securityPayloadString(payload, "session_id")
	deviceID := ""
	reasons := []string{event.Type}
	severity := "notice"
	outcome := securitySucceeded

	if event.Type == securitySessionCreated {
		var session domain.Session
		//nolint:musttag // Preserve the existing persisted domain envelope.
		if err := json.Unmarshal(raw, &session); err != nil {
			return securityStoreError(err)
		}

		scope.AccountID = session.AccountID
		sessionID = session.ID

		reasons, deviceID, err = s.sessionReasons(ctx, scope, &session, policy)
		if err != nil {
			return securityStoreError(err)
		}

		if len(reasons) == 0 {
			return nil
		}

		severity = "warning"
	}

	if scope.AccountID == "" {
		if p, ok := api.PrincipalFrom(ctx); ok && p.AccountID != "" {
			scope.AccountID = p.AccountID
		} else if slices.Contains(
			[]string{
				securityTokenReuseDetected,
				"password.reset",
				"password.changed",
				"email.changed",
				"phone.changed",
				"identity.linked",
				"identity.unlinked",
				"mfa.recovery_codes.generated",
			},
			event.Type) {
			scope.AccountID = event.AggregateID
		}
	}

	if scope.AccountID == "" {
		return nil
	}

	if event.Type != securitySessionCreated &&
		event.Type != securityTokenReuseDetected &&
		event.Type != securitySessionDeviceMismatch {
		if err := securityAccountGuard(ctx, s.db, scope.ProjectID, scope.AccountID); err != nil {
			return securityStoreError(err)
		}
	}

	if policy.Mode == securityDisabled {
		return nil
	}

	if sessionID == "" {
		if p, ok := api.PrincipalFrom(ctx); ok {
			sessionID = p.SessionID
		}
	}

	if event.Type == securityTokenReuseDetected || event.Type == securitySessionDeviceMismatch {
		severity = "high"
		outcome = "blocked"
	}

	a, err := s.account(ctx, scope)
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil
	}

	if err != nil {
		return securityStoreError(err)
	}

	contacts := securityIncidentPrivate{}
	if a.EmailVerified {
		contacts.Email = a.PrimaryEmail
	}

	if a.PhoneVerified {
		contacts.Phone = a.PrimaryPhone
	}

	if old, ok := ctx.Value(securityOldContactKey{}).(securityIncidentPrivate); ok {
		contacts = old
	}

	var active int
	if err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT count(*) FROM iam_factors WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND status='active'`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID).Scan(&active); err != nil {
		return securityStoreError(err)
	}

	contacts.MFARequired = active > 0 || event.Type == "mfa.factor.removed"

	return s.recordIncident(
		ctx,
		scope,
		event.Type,
		sessionID,
		deviceID,
		severity,
		outcome,
		reasons,
		contacts,
		policy)
}

func (s *pgSecurity) recognizeDevice(
	ctx context.Context,
	scope domain.SecurityScope,
	sessionID string,
) (bool, string, error) {
	meta := domain.RequestMetaFromContext(ctx)
	if meta.DeviceToken == "" {
		return false, "", nil
	}

	var raw []byte

	err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_security_devices WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND token_hash=$4`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		accountHashToken(meta.DeviceToken)).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}

	if err != nil {
		return false, "", securityStoreError(err)
	}

	var device domain.SecurityDevice
	if err := json.Unmarshal(raw, &device); err != nil {
		return false, "", securityStoreError(err)
	}

	if device.Revoked {
		return false, "", nil
	}

	device.LastSeenAt = nowIn(ctx)
	if !slices.Contains(device.SessionIDs, sessionID) {
		device.SessionIDs = append(device.SessionIDs, sessionID)
	}

	if len(device.SessionIDs) > securitySessionHistoryLimit {
		device.SessionIDs = device.SessionIDs[len(device.SessionIDs)-securitySessionHistoryLimit:]
	}

	if err := s.save(ctx, scope, securityDevices, device.ID, "", device); err != nil {
		return false, "", securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_sessions SET data=jsonb_set(data,'{device_id}',to_jsonb($1::text)) WHERE
id=$2 AND project_id=$3 AND environment=$4 AND user_id=$5`,
		device.ID,
		sessionID,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID)

	return true, device.ID, securityStoreError(err)
}

//nolint:funlen // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) recordIncident(
	ctx context.Context,
	scope domain.SecurityScope,
	typ, sessionID, deviceID, severity, outcome string,
	reasons []string,
	contacts securityIncidentPrivate,
	policy *domain.SecurityPolicy,
) error {
	now := nowIn(ctx)
	meta := domain.RequestMetaFromContext(ctx)
	activityEvent := domain.SecurityEvent{
		ID:        newUUID(),
		Type:      typ,
		At:        now,
		Outcome:   outcome,
		SessionID: sessionID,
		DeviceID:  deviceID,
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
	}

	var raw []byte

	err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_security_incidents WHERE project_id=$1 AND environment=$2 AND
user_id=$3 AND status='open' AND created_at>$4 AND COALESCE(data->>'session_id','')=$5
AND data->'events'->0->>'type'=$6 ORDER BY created_at DESC LIMIT 1`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		now.Add(-time.Duration(policy.NotificationCooldownSeconds)*time.Second),
		sessionID,
		typ).Scan(&raw)
	if err == nil {
		var inc domain.SecurityIncident
		if err := json.Unmarshal(raw, &inc); err != nil {
			return securityStoreError(err)
		}

		inc.UpdatedAt = now
		if len(inc.Events) < securityEventHistoryLimit {
			inc.Events = append(inc.Events, activityEvent)
		}

		return s.save(ctx, scope, securityIncidents, inc.ID, inc.Status, inc)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return securityStoreError(err)
	}

	inc := domain.SecurityIncident{
		ID:                newUUID(),
		AccountID:         scope.AccountID,
		Status:            "open",
		Severity:          severity,
		Reasons:           reasons,
		Events:            []domain.SecurityEvent{activityEvent},
		SessionID:         sessionID,
		DeviceID:          deviceID,
		CreatedAt:         now,
		UpdatedAt:         now,
		PolicyVersion:     policy.Version,
		RevokedSessionIDs: []string{},
	}

	raw, err = json.Marshal(inc)
	if err != nil {
		return securityStoreError(err)
	}

	private, err := s.encrypt(contacts)
	if err != nil {
		return securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`INSERT INTO iam_security_incidents(id,project_id,environment,user_id,status,created_at,data,private_data)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		inc.ID,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		inc.Status,
		now,
		raw,
		private)
	if err != nil {
		return securityStoreError(err)
	}

	if err := s.emit(
		ctx,
		scope,
		"security.incident.created",
		inc.ID,
		map[string]any{
			"incident_id":     inc.ID,
			securityAccountID: scope.AccountID,
			"severity":        severity,
			"reasons":         reasons,
		}); err != nil {
		return securityStoreError(err)
	}

	if policy.Notify && policy.Mode == securityEnforce && (contacts.Email != "" || contacts.Phone != "") {
		token, err := s.continuation(ctx, scope, inc.ID, "", securityReview)
		if err != nil {
			return securityStoreError(err)
		}

		recipient := contacts.Email
		if recipient == "" {
			recipient = contacts.Phone
		}

		return s.queueDelivery(
			ctx,
			scope,
			securityDeliveryPayload{
				To:           recipient,
				Template:     "security_alert",
				Continuation: token,
				Summary:      typ + " · " + outcome + " · " + now.Format(time.RFC3339) + " · " + meta.UserAgent + " · " + meta.IP,
			},
			inc.ID,
			"",
			"incident:"+inc.ID)
	}

	return nil
}

//nolint:funlen // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) sessionReasons(
	ctx context.Context,
	scope domain.SecurityScope,
	sess *domain.Session,
	policy *domain.SecurityPolicy,
) ([]string, string, error) {
	sessionID := sess.ID

	var (
		deviceID string
		reasons  []string
	)

	known, id, err := s.recognizeDevice(ctx, scope, sessionID)
	if err != nil {
		return nil, "", securityStoreError(err)
	}

	deviceID = id

	var prior bool
	if err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE project_id=$1 AND environment=$2 AND
user_id=$3 AND id<>$4)`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		sessionID).Scan(&prior); err != nil {
		return nil, "", securityStoreError(err)
	}

	if !prior && nowIn(ctx).Sub(sess.CreatedAt) < time.Minute {
		var old bool
		if err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT created_at<$2 FROM iam_users WHERE id=$1`,
			scope.AccountID,
			nowIn(ctx).Add(-time.Minute)).Scan(&old); err != nil {
			return nil, "", securityStoreError(err)
		}

		if !old {
			return nil, "", nil
		}
	}

	reasons = []string{}
	if !known {
		reasons = append(reasons, "new_device")
	}

	meta := domain.RequestMetaFromContext(ctx)
	if meta.IP != "" && prior {
		var seen bool
		if err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE project_id=$1 AND environment=$2 AND
user_id=$3 AND id<>$4 AND data->>'ip'=$5)`,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID,
			sessionID,
			meta.IP).Scan(&seen); err != nil {
			return nil, "", securityStoreError(err)
		}

		if !seen {
			reasons = append(reasons, "new_ip")
		}
	}

	var failures bool
	if err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_security_attempts WHERE project_id=$1 AND environment=$2
AND subject_hash=$3 AND kind IN ('password_failures','mfa_failures') AND window_start>$4
AND attempts>=$5)`,
		scope.ProjectID,
		scope.Environment,
		accountHashToken(scope.AccountID),
		nowIn(ctx).Add(-time.Duration(policy.FailureWindowSeconds)*time.Second),
		policy.FailureThreshold).Scan(&failures); err != nil {
		return nil, "", securityStoreError(err)
	}

	if failures {
		reasons = append(reasons, "recent_failures")
	}

	if len(reasons) == 0 {
		return nil, "", nil
	}

	return reasons, deviceID, nil
}

func securityPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && value != "" {
			return value
		}
	}

	return ""
}
