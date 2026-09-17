package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

// pgSecurity uses the ambient transaction for capability consumption, account
// mutations and delivery insertion. No token or contact comes from public events.
type pgSecurity struct {
	db      *DB
	emitter Emitter
}

func NewPgSecurity(db *DB, emitter Emitter) *pgSecurity { return &pgSecurity{db: db, emitter: emitter} }

var _ api.AccountSecurityStore = (*pgSecurity)(nil)

func (s *pgSecurity) scope(ctx context.Context, v domain.SecurityScope) (domain.SecurityScope, error) {
	if v.ProjectID == "" {
		return v, domain.ErrProjectNotFound
	}

	env, err := effectiveEnv(api.WithEnvironment(ctx, v.Environment), s.db, v.ProjectID)
	if err != nil {
		return v, securityStoreError(err)
	}

	v.Environment = env

	return v, nil
}

func securityDefaultPolicy() domain.SecurityPolicy {
	return domain.SecurityPolicy{
		Version:                     1,
		Mode:                        securityDisabled,
		ClientURLs:                  map[string]string{},
		FlowTTLSeconds:              securityDefaultFlowSeconds,
		TrustTTLSeconds:             securityDefaultTrustSeconds,
		RetentionDays:               securityDefaultRetentionDays,
		FailureThreshold:            securityDefaultFailureThreshold,
		FailureWindowSeconds:        securityDefaultFailureWindowSeconds,
		NotificationCooldownSeconds: securityDefaultNotificationCooldown,
	}
}

func (s *pgSecurity) Policy(ctx context.Context, scope domain.SecurityScope) (*domain.SecurityPolicy, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	p := securityDefaultPolicy()

	var raw []byte

	err = s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_security_policies WHERE project_id=$1 AND environment=$2`,
		scope.ProjectID,
		scope.Environment).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return &p, nil
	}

	if err != nil {
		return nil, securityStoreError(err)
	}

	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, securityStoreError(err)
	}

	return &p, nil
}

func securityURLValid(raw string) bool {
	continuationURL, err := url.Parse(raw)
	if err != nil || continuationURL.Host == "" || continuationURL.User != nil ||
		continuationURL.Fragment != "" || continuationURL.RawQuery != "" {
		return false
	}

	return continuationURL.Scheme == "https" ||
		(continuationURL.Scheme == "http" &&
			(continuationURL.Hostname() == "localhost" ||
				continuationURL.Hostname() == "127.0.0.1" ||
				continuationURL.Hostname() == "::1"))
}

func (s *pgSecurity) SetPolicy(
	ctx context.Context,
	scope domain.SecurityScope,
	p domain.SecurityPolicy,
) (*domain.SecurityPolicy, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if err := validateSecurityPolicy(p); err != nil {
		return nil, securityStoreError(err)
	}

	if p.ClientURLs == nil {
		p.ClientURLs = map[string]string{}
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityPolicy, error) {
		raw, err := json.Marshal(p)
		if err != nil {
			return nil, securityStoreError(err)
		}

		_, err = s.db.TxDB.Exec(
			ctx,
			`INSERT INTO iam_security_policies(project_id,environment,data) VALUES($1,$2,$3) ON
CONFLICT(project_id,environment) DO UPDATE SET data=EXCLUDED.data`,
			scope.ProjectID,
			scope.Environment,
			raw)
		if err != nil {
			return nil, securityStoreError(err)
		}

		if err := s.emit(
			ctx,
			scope,
			"security.policy.updated",
			scope.ProjectID,
			map[string]any{"version": p.Version, "mode": p.Mode}); err != nil {
			return nil, securityStoreError(err)
		}

		return &p, nil
	})
}

func (s *pgSecurity) emit(
	ctx context.Context,
	scope domain.SecurityScope,
	typ, id string,
	payload any,
) error {
	return s.emitter.Emit(
		ctx,
		domain.Event{
			Type:        typ,
			ProjectID:   scope.ProjectID,
			Environment: scope.Environment,
			AggregateID: id,
			Payload:     payload,
		})
}

func (s *pgSecurity) encrypt(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", securityStoreError(err)
	}

	return s.db.Cipher.Encrypt(string(b))
}

func (s *pgSecurity) decrypt(ciphertext string, v any) error {
	if ciphertext == "" {
		return nil
	}

	b, err := s.db.Cipher.Decrypt(ciphertext)
	if err != nil {
		return securityStoreError(err)
	}

	return securityStoreError(json.Unmarshal([]byte(b), v))
}

// Security record tables are an internal closed set, never a caller SQL identifier.
func securityTable(kind string) (string, error) {
	switch kind {
	case securityIncidents, securityDevices, securityCases, securityDeliveries:
		return "iam_security_" + kind, nil
	}

	return "", domain.ErrBadRequest
}

//nolint:funlen,cyclop // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) List(
	ctx context.Context,
	scope domain.SecurityScope,
	kind, cursor string,
	limit int,
) (api.SecurityPage, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return api.SecurityPage{}, securityStoreError(err)
	}

	table, err := securityTable(kind)
	if err != nil {
		return api.SecurityPage{}, securityStoreError(err)
	}

	if limit < 1 || limit > 100 {
		limit = 30
	}

	var after struct {
		At time.Time `json:"at"`
		ID string    `json:"id"`
	}

	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return api.SecurityPage{}, domain.ErrValidation
		}

		if err := json.Unmarshal(raw, &after); err != nil {
			return api.SecurityPage{}, domain.ErrValidation
		}
	}

	rows, err := s.db.TxDB.Query(
		ctx,
		`SELECT id,data,COALESCE(data->>'created_at',data->>'first_seen_at')::timestamptz FROM
`+table+` WHERE project_id=$1 AND environment=$2 AND ($3='' OR user_id=$3) AND ($4='' OR
(COALESCE(data->>'created_at',data->>'first_seen_at')::timestamptz,id)<($5,$4)) ORDER BY
COALESCE(data->>'created_at',data->>'first_seen_at')::timestamptz DESC,id DESC LIMIT $6`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		after.ID,
		after.At,
		limit+1)
	if err != nil {
		return api.SecurityPage{}, securityStoreError(err)
	}
	defer rows.Close()

	data := make([]json.RawMessage, 0, limit)
	next := ""
	more := false

	for rows.Next() {
		var (
			id        string
			raw       json.RawMessage
			createdAt time.Time
		)

		if err := rows.Scan(&id, &raw, &createdAt); err != nil {
			return api.SecurityPage{}, securityStoreError(err)
		}

		if len(data) == limit {
			more = true
			break
		}

		if kind == securityDevices {
			raw, err = s.deviceListJSON(ctx, scope, raw)
			if err != nil {
				return api.SecurityPage{}, securityStoreError(err)
			}
		}

		data = append(data, raw)
		after.At = createdAt
		after.ID = id

		encoded, err := json.Marshal(after)
		if err != nil {
			return api.SecurityPage{}, securityStoreError(err)
		}

		next = base64.RawURLEncoding.EncodeToString(encoded)
	}

	if err := rows.Err(); err != nil {
		return api.SecurityPage{}, securityStoreError(err)
	}

	if !more {
		next = ""
	}

	return api.SecurityPage{Data: data, HasMore: more, NextCursor: next}, nil
}

func (s *pgSecurity) load(
	ctx context.Context,
	scope domain.SecurityScope,
	kind, id string,
	v, private any,
) error {
	table, err := securityTable(kind)
	if err != nil {
		return securityStoreError(err)
	}

	var (
		raw    []byte
		secret string
	)

	col := "private_data"
	if kind == securityDevices {
		col = "''"
	}

	err = s.db.TxDB.QueryRow(
		ctx,
		`SELECT data,`+col+` FROM `+table+` WHERE id=$1 AND project_id=$2 AND environment=$3 AND ($4='' OR user_id=$4)`,
		id,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID).Scan(&raw, &secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	if err != nil {
		return securityStoreError(err)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return securityStoreError(err)
	}

	if private != nil {
		return s.decrypt(secret, private)
	}

	return nil
}

func (s *pgSecurity) save(
	ctx context.Context,
	scope domain.SecurityScope,
	kind, id, status string,
	v any,
) error {
	table, err := securityTable(kind)
	if err != nil {
		return securityStoreError(err)
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return securityStoreError(err)
	}

	if kind == securityDevices {
		_, err = s.db.TxDB.Exec(
			ctx,
			`UPDATE `+table+` SET data=$1 WHERE id=$2 AND project_id=$3 AND environment=$4 AND user_id=$5`,
			raw,
			id,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID)
	} else {
		_, err = s.db.TxDB.Exec(
			ctx,
			`UPDATE `+table+` SET data=$1,status=$2 WHERE id=$3 AND project_id=$4 AND environment=$5 AND ($6='' OR user_id=$6)`,
			raw,
			status,
			id,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID)
	}

	return securityStoreError(err)
}

func (s *pgSecurity) Incident(
	ctx context.Context,
	scope domain.SecurityScope,
	id string,
) (*domain.SecurityIncident, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	var v domain.SecurityIncident

	err = s.load(ctx, scope, securityIncidents, id, &v, nil)

	return &v, securityStoreError(err)
}

// throttle is transactional. Callers return error states, rather than roll back,
// when a failed proof consumes an attempt.
func (s *pgSecurity) throttle(
	ctx context.Context,
	scope domain.SecurityScope,
	subject, kind string,
	limit int,
	window time.Duration,
) (bool, error) {
	var count int

	err := s.db.TxDB.QueryRow(
		ctx,
		`INSERT INTO iam_security_attempts(project_id,environment,subject_hash,kind,window_start,attempts)
VALUES($1,$2,$3,$4,$5,1)
	 ON CONFLICT(project_id,environment,subject_hash,kind) DO
UPDATE SET
	 attempts=CASE WHEN iam_security_attempts.window_start<$6 THEN 1 ELSE
iam_security_attempts.attempts+1 END,
	 window_start=CASE WHEN iam_security_attempts.window_start<$6
THEN $5 ELSE iam_security_attempts.window_start END RETURNING attempts`,
		scope.ProjectID,
		scope.Environment,
		accountHashToken(subject),
		kind,
		nowIn(ctx),
		nowIn(ctx).Add(-window)).Scan(&count)

	return count <= limit, securityStoreError(err)
}

//nolint:funlen // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) RegisterDevice(
	ctx context.Context,
	scope domain.SecurityScope,
	name string,
) (*domain.SecurityDeviceRegistration, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if scope.AccountID == "" || scope.SessionID == "" {
		return nil, domain.ErrUnauthorized
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityDeviceRegistration, error) {
		var live bool

		err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE id=$1 AND project_id=$2 AND environment=$3
AND user_id=$4)`,
			scope.SessionID,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID).Scan(&live)
		if err != nil {
			return nil, securityStoreError(err)
		}

		if !live {
			return nil, domain.ErrSessionExpired
		}

		ok, err := s.throttle(
			ctx,
			scope,
			scope.AccountID,
			"device",
			securityProofBurst,
			time.Hour)
		if err != nil {
			return nil, securityStoreError(err)
		}

		if !ok {
			return nil, domain.ErrRateLimited
		}

		token, err := accountRandomToken(securityTokenBytes)
		if err != nil {
			return nil, securityStoreError(err)
		}

		now := nowIn(ctx)
		v := domain.SecurityDevice{
			ID:          newUUID(),
			Name:        strings.TrimSpace(name),
			FirstSeenAt: now,
			LastSeenAt:  now,
			SessionIDs:  []string{scope.SessionID},
		}

		raw, err := json.Marshal(v)
		if err != nil {
			return nil, securityStoreError(err)
		}

		_, err = s.db.TxDB.Exec(
			ctx,
			`INSERT INTO iam_security_devices(id,project_id,environment,user_id,token_hash,data)
VALUES($1,$2,$3,$4,$5,$6)`,
			v.ID,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID,
			accountHashToken(token),
			raw)
		if err != nil {
			return nil, securityStoreError(err)
		}

		_, err = s.db.TxDB.Exec(
			ctx,
			`UPDATE iam_sessions SET data=jsonb_set(data,'{device_id}',to_jsonb($1::text)) WHERE
id=$2 AND user_id=$3 AND project_id=$4 AND environment=$5`,
			v.ID,
			scope.SessionID,
			scope.AccountID,
			scope.ProjectID,
			scope.Environment)

		v.Current = true

		return &domain.SecurityDeviceRegistration{Device: v, DeviceToken: token}, securityStoreError(err)
	})
}

func (s *pgSecurity) UpdateDevice(
	ctx context.Context,
	scope domain.SecurityScope,
	id, action, name, proof string,
) (*domain.SecurityDevice, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityDevice, error) {
		var device domain.SecurityDevice
		if err := s.load(ctx, scope, securityDevices, id, &device, nil); err != nil {
			return nil, securityStoreError(err)
		}

		current := slices.Contains(device.SessionIDs, scope.SessionID)

		switch action {
		case "rename":
			device.Name = strings.TrimSpace(name)
		case "untrust":
			device.TrustedUntil = nil
		case securityTrust:
			if err := s.trustDevice(ctx, scope, &device, proof); err != nil {
				return nil, securityStoreError(err)
			}

		case "revoke":
			ids, err := s.deviceSessionIDs(ctx, scope, id)
			if err != nil {
				return nil, err
			}

			current = slices.Contains(ids, scope.SessionID)
			for _, sid := range ids {
				if err := s.revokeSession(ctx, scope, sid); err != nil {
					return nil, securityStoreError(err)
				}
			}

			device.Revoked = true
			device.TrustedUntil = nil
		default:
			return nil, domain.ErrBadRequest
		}

		if err := s.save(ctx, scope, securityDevices, id, "", device); err != nil {
			return nil, securityStoreError(err)
		}

		if err := s.emit(
			ctx,
			scope,
			"security.device."+action,
			id,
			map[string]any{securityAccountID: scope.AccountID, "device_id": id}); err != nil {
			return nil, securityStoreError(err)
		}

		device.Current = current

		return &device, nil
	})
}

func (s *pgSecurity) revokeSession(ctx context.Context, scope domain.SecurityScope, id string) error {
	var exists bool
	if err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE id=$1 AND project_id=$2 AND environment=$3
AND user_id=$4)`,
		id,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID).Scan(&exists); err != nil {
		return securityStoreError(err)
	}

	if !exists {
		return nil
	}

	return NewPgCoreAuth(s.db, s.emitter, nil).coreAuthRevokeSession(ctx, scope.ProjectID, id)
}

//nolint:funlen,cyclop // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) protect(
	ctx context.Context,
	scope domain.SecurityScope,
	f *securityFlow,
) error {
	tag, err := s.db.TxDB.Exec(
		ctx,
		`INSERT INTO iam_security_guards(project_id,environment,user_id,created_at,flow_id)
VALUES($1,$2,$3,$4,$5) ON CONFLICT(project_id,environment,user_id) DO UPDATE SET flow_id=EXCLUDED.flow_id
WHERE iam_security_guards.flow_id=$5 OR $6`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		nowIn(ctx),
		f.ID,
		f.SupportGrant)
	if err != nil {
		return securityStoreError(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrConflict.WithMessage("Another recovery is already protecting this account")
	}

	ids := []string{}

	if f.State.AllSessions {
		rows, err := s.db.TxDB.Query(
			ctx,
			`SELECT id FROM iam_sessions WHERE project_id=$1 AND environment=$2 AND user_id=$3`,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID)
		if err != nil {
			return securityStoreError(err)
		}

		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return securityStoreError(err)
			}

			ids = append(ids, id)
		}

		err = rows.Err()
		rows.Close()

		if err != nil {
			return securityStoreError(err)
		}
	} else if f.TargetSession != "" {
		ids = append(ids, f.TargetSession)
	}

	for _, id := range ids {
		if err := s.revokeSession(ctx, scope, id); err != nil {
			return securityStoreError(err)
		}
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_security_devices SET data=(data-'trusted_until') || jsonb_build_object('revoked',true)
WHERE project_id=$1 AND environment=$2 AND user_id=$3 AND ($4 OR data->'session_ids'
? $5)`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		f.State.AllSessions,
		f.TargetSession)
	if err != nil {
		return securityStoreError(err)
	}

	f.Protected = true

	f.State.RevokedSessionIDs = ids
	if f.IncidentID != "" {
		inc, err := s.Incident(ctx, scope, f.IncidentID)
		if err != nil {
			return securityStoreError(err)
		}

		inc.Status = "protecting"
		inc.RevokedSessionIDs = ids

		inc.UpdatedAt = nowIn(ctx)
		if err := s.save(ctx, scope, securityIncidents, inc.ID, inc.Status, inc); err != nil {
			return securityStoreError(err)
		}
	}

	if err := s.notifyStatus(ctx, f, "account_protected"); err != nil {
		return securityStoreError(err)
	}

	return s.emit(
		ctx,
		scope,
		"security.account.protected",
		scope.AccountID,
		map[string]any{
			securityAccountID:     scope.AccountID,
			"all_sessions":        f.State.AllSessions,
			"revoked_session_ids": ids,
		})
}

func securityMask(contact string) string {
	if createdAt := strings.LastIndex(contact, "@"); createdAt > 0 {
		return contact[:1] + "***" + contact[createdAt:]
	}

	if len(contact) > securityPhoneVisibleDigits {
		return "***" + contact[len(contact)-securityPhoneVisibleDigits:]
	}

	return "***"
}

func securityStoreError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("account security store: %w", err)
}

func validateSecurityPolicy(policy domain.SecurityPolicy) error {
	if policy.Version != 1 || !slices.Contains(
		[]string{securityDisabled, "observe", securityEnforce},
		policy.Mode) {
		return domain.ErrValidation
	}

	bounds := [][3]int{
		{policy.FlowTTLSeconds, 300, 3600},
		{policy.TrustTTLSeconds, 300, 7776000},
		{policy.RetentionDays, 1, 3650},
		{policy.FailureThreshold, 2, 100},
		{policy.FailureWindowSeconds, 60, 86400},
		{policy.NotificationCooldownSeconds, 60, 86400},
	}
	for _, bound := range bounds {
		if bound[0] < bound[1] || bound[0] > bound[2] {
			return domain.ErrValidation.WithMessage("Invalid security policy bounds")
		}
	}

	if (policy.ContinueURL != "" || policy.Notify) && !securityURLValid(policy.ContinueURL) {
		return domain.ErrInvalidRedirectURI
	}

	for client, uri := range policy.ClientURLs {
		if client == "" || !securityURLValid(uri) {
			return domain.ErrInvalidRedirectURI
		}
	}

	return nil
}

func (s *pgSecurity) trustDevice(
	ctx context.Context,
	scope domain.SecurityScope,
	device *domain.SecurityDevice,
	proof string,
) error {
	id := device.ID

	f, err := s.loadFlow(ctx, scope, proof)
	if err != nil {
		return securityStoreError(err)
	}

	if f.CurrentToken != proof ||
		f.SupportGrant ||
		f.Purpose != securityTrust ||
		f.TargetDeviceID != id ||
		f.State.Status != securityPending ||
		f.AccountID != scope.AccountID ||
		!f.Proved ||
		f.State.Step != securityReview ||
		device.Revoked ||
		!slices.Contains(device.SessionIDs, f.TargetSession) {
		return domain.ErrStepUpRequired
	}

	policy, err := s.Policy(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	until := nowIn(ctx).Add(time.Duration(policy.TrustTTLSeconds) * time.Second)
	device.TrustedUntil = &until
	f.State.Status = securityCompleted
	f.State.Step = securityCompleted

	f.State.Outcome = "device_trusted"
	if err := s.saveFlow(ctx, f, proof); err != nil {
		return securityStoreError(err)
	}

	return nil
}

func (s *pgSecurity) deviceListJSON(
	ctx context.Context,
	scope domain.SecurityScope,
	raw json.RawMessage,
) (json.RawMessage, error) {
	var err error

	var device domain.SecurityDevice
	if err := json.Unmarshal(raw, &device); err != nil {
		return nil, securityStoreError(err)
	}

	device.Current = slices.Contains(device.SessionIDs, scope.SessionID)
	if device.TrustedUntil != nil && !nowIn(ctx).Before(*device.TrustedUntil) {
		device.TrustedUntil = nil
	}

	raw, err = json.Marshal(device)
	if err != nil {
		return nil, securityStoreError(err)
	}

	return raw, nil
}

func (s *pgSecurity) deviceSessionIDs(ctx context.Context, scope domain.SecurityScope, id string) ([]string, error) {
	rows, err := s.db.TxDB.Query(
		ctx,
		`SELECT id FROM iam_sessions
 WHERE project_id=$1 AND environment=$2 AND user_id=$3 AND data->>'device_id'=$4`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		id)
	if err != nil {
		return nil, securityStoreError(err)
	}
	defer rows.Close()

	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])

	return ids, securityStoreError(err)
}
