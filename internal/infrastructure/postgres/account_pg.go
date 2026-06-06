package postgres

// AccountStore Postgres adapter — the user-self slice of the Account aggregate.
//
// Covers three envelopes: iam_users (profile/delete), iam_sessions
// (list/get/rename/trust/revoke), iam_identities (list/unlink/merge), plus the
// supporting iam_consents, iam_activity and iam_jobs tables. Every aggregate is
// stored as the `data` jsonb envelope; the typed columns are lookup keys only.
//
// Tenant boundary: every read filters by project_id (and the account/user id
// where the port already scopes by account). A row whose project_id does not
// match the requested one is treated as not-found.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgAccountStore implements api.AccountStore over the iam_users / iam_sessions /
// iam_identities envelopes plus the consent / activity / job tables.
type pgAccountStore struct {
	db      *DB
	emitter Emitter
}

// NewPgAccountStore builds the Postgres-backed AccountStore.
func NewPgAccountStore(db *DB, emitter Emitter) *pgAccountStore {
	return &pgAccountStore{db: db, emitter: emitter}
}

var _ api.AccountStore = (*pgAccountStore)(nil)

// ===== account-local helpers =====

// accountRandomToken returns a hex-encoded cryptographically-random token.
func accountRandomToken(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// accountRandomCode returns a short numeric-ish opaque code (hex of 4 bytes).
func accountRandomCode() (string, error) { return accountRandomToken(4) }

// accountSeconds converts a seconds count into a time.Duration.
func accountSeconds(n int) time.Duration { return time.Duration(n) * time.Second }

// accountHashToken returns the sha256 hex digest of an opaque token; only the
// digest is ever persisted, never the plaintext.
func accountHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// accountLoadUser fetches the user envelope and enforces the tenant boundary.
func (a *pgAccountStore) accountLoadUser(ctx context.Context, projectID, accountID string) (*models.IamUser, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	if row.ProjectID != projectID {
		return nil, domain.ErrUserNotFound
	}
	return row, nil
}

// accountLoadSession fetches a session owned by the account (account == user).
func (a *pgAccountStore) accountLoadSession(ctx context.Context, accountID, sessionID string) (*models.IamSession, error) {
	row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	if row.UserID != accountID {
		return nil, domain.ErrSessionNotFound
	}
	return row, nil
}

// accountSessionToDomain maps a session envelope to the domain Session.
func accountSessionToDomain(row *models.IamSession) (*domain.Session, error) {
	var s domain.Session
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &s); err != nil {
			return nil, err
		}
	}
	// envelope columns win for the queryable fields
	s.ID = row.ID
	s.AccountID = row.UserID
	s.ProjectID = row.ProjectID
	if v, ok := row.ClientID.Get(); ok {
		s.ClientID = v
	}
	s.AAL = int(row.Aal)
	s.CreatedAt = row.CreatedAt
	if v, ok := row.ExpiresAt.Get(); ok {
		s.ExpiresIn = int(v.Sub(row.CreatedAt).Seconds())
	}
	return &s, nil
}

// accountIdentityToDomain maps an identity envelope to the domain Identity.
func accountIdentityToDomain(row *models.IamIdentity) (*domain.Identity, error) {
	var id domain.Identity
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &id); err != nil {
			return nil, err
		}
	}
	id.ID = row.ID
	id.Type = row.Type
	if v, ok := row.Provider.Get(); ok {
		id.Provider = v
	}
	if v, ok := row.ProviderAccountID.Get(); ok {
		id.ProviderAccountID = v
	}
	if v, ok := row.Email.Get(); ok {
		id.Email = v
	}
	return &id, nil
}

// ===== api.AccountStore =====

// Get returns the account profile, enforcing the tenant boundary.
func (a *pgAccountStore) Get(ctx context.Context, projectID, accountID string) (*domain.Account, error) {
	row, err := a.accountLoadUser(ctx, projectID, accountID)
	if err != nil {
		return nil, err
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

// UpdateProfile applies a profile patch and persists the updated aggregate.
func (a *pgAccountStore) UpdateProfile(ctx context.Context, cmd domain.ProfileUpdateCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, err := a.accountLoadUser(ctx, cmd.ProjectID, cmd.AccountID)
		if err != nil {
			return nil, err
		}
		var acc domain.Account
		if err := unmarshal(row.Data, &acc); err != nil {
			return nil, err
		}
		if cmd.Name != "" {
			acc.Name = cmd.Name
		}
		if cmd.Locale != "" {
			acc.Locale = cmd.Locale
		}
		acc.UpdatedAt = nowUTC()

		// Merge the typed account back over the existing envelope so non-Account
		// keys (e.g. avatar_url) survive the round-trip.
		env := map[string]any{}
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &env); err != nil {
				return nil, err
			}
		}
		accRaw, err := marshal(&acc)
		if err != nil {
			return nil, err
		}
		accMap := map[string]any{}
		if err := unmarshal(accRaw, &accMap); err != nil {
			return nil, err
		}
		for k, v := range accMap {
			env[k] = v
		}
		if cmd.AvatarURL != "" {
			env["avatar_url"] = cmd.AvatarURL
		}
		raw, err := marshal(env)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamUserSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.profile_updated",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return &acc, nil
	})
}

// Delete removes the account, enforcing the tenant boundary.
func (a *pgAccountStore) Delete(ctx context.Context, projectID, accountID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := a.accountLoadUser(ctx, projectID, accountID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: accountID,
			Payload:     map[string]any{"id": accountID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ListSessions returns the account's sessions, newest first.
func (a *pgAccountStore) ListSessions(ctx context.Context, accountID string) ([]domain.Session, error) {
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(accountID))),
		sm.OrderBy(models.IamSessions.Columns.CreatedAt).Desc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		s, err := accountSessionToDomain(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, nil
}

// GetSession resolves a single session owned by the account.
func (a *pgAccountStore) GetSession(ctx context.Context, accountID, sessionID string) (*domain.Session, error) {
	row, err := a.accountLoadSession(ctx, accountID, sessionID)
	if err != nil {
		return nil, err
	}
	return accountSessionToDomain(row)
}

// RevokeSession deletes one of the account's sessions.
func (a *pgAccountStore) RevokeSession(ctx context.Context, accountID, sessionID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := a.accountLoadSession(ctx, accountID, sessionID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "session.revoked",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     map[string]any{"id": row.ID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// RenameSession sets a human-friendly device name on one of the account's
// sessions (stored in the session envelope).
func (a *pgAccountStore) RenameSession(ctx context.Context, cmd domain.AccountRenameSessionCmd) (*domain.Session, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Session, error) {
		row, err := a.accountLoadSession(ctx, cmd.AccountID, cmd.SessionID)
		if err != nil {
			return nil, err
		}
		// device name lives in the envelope; merge it in.
		var env map[string]any
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &env); err != nil {
				return nil, err
			}
		}
		if env == nil {
			env = map[string]any{}
		}
		env["device_name"] = cmd.DeviceName
		raw, err := marshal(env)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamSessionSetter{Data: &rm, LastActiveAt: ptr(nowUTC())}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		sess, err := accountSessionToDomain(row)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "session.renamed",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     sess,
		}); err != nil {
			return nil, err
		}
		return sess, nil
	})
}

// TrustSession marks a session trusted for the given duration.
func (a *pgAccountStore) TrustSession(ctx context.Context, cmd domain.AccountTrustSessionCmd) (*domain.Session, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Session, error) {
		row, err := a.accountLoadSession(ctx, cmd.AccountID, cmd.SessionID)
		if err != nil {
			return nil, err
		}
		trusted := true
		exp := null.From(nowUTC().Add(accountSeconds(cmd.DurationSeconds)))
		setter := &models.IamSessionSetter{Trusted: &trusted, ExpiresAt: &exp}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		sess, err := accountSessionToDomain(row)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "session.trusted",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     sess,
		}); err != nil {
			return nil, err
		}
		return sess, nil
	})
}

// RevokeSessions bulk-deletes the account's sessions; returns the count revoked.
func (a *pgAccountStore) RevokeSessions(ctx context.Context, cmd domain.AccountRevokeSessionsCmd) (int, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (int, error) {
		rows, err := models.IamSessions.Query(
			sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(cmd.AccountID))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return 0, err
		}
		victims := make(models.IamSessionSlice, 0, len(rows))
		for _, row := range rows {
			if cmd.ExceptCurrent && cmd.ExceptSessionID != "" && row.ID == cmd.ExceptSessionID {
				continue
			}
			victims = append(victims, row)
		}
		if len(victims) == 0 {
			return 0, nil
		}
		if err := victims.DeleteAll(ctx, a.db.Bobx()); err != nil {
			return 0, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "sessions.revoked",
			ProjectID:   victims[0].ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     map[string]any{"account_id": cmd.AccountID, "project_id": victims[0].ProjectID, "count": len(victims)},
		}); err != nil {
			return 0, err
		}
		return len(victims), nil
	})
}

// ListIdentities returns the account's linked identities.
func (a *pgAccountStore) ListIdentities(ctx context.Context, accountID string) ([]domain.Identity, error) {
	rows, err := models.IamIdentities.Query(
		sm.Where(models.IamIdentities.Columns.UserID.EQ(psql.Arg(accountID))),
		sm.OrderBy(models.IamIdentities.Columns.CreatedAt).Desc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Identity, 0, len(rows))
	for _, row := range rows {
		id, err := accountIdentityToDomain(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *id)
	}
	return out, nil
}

// UnlinkIdentity removes a linked identity from the account.
func (a *pgAccountStore) UnlinkIdentity(ctx context.Context, accountID, identityID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamIdentity(ctx, a.db.Bobx(), identityID)
		if err != nil {
			if isNoRows(err) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.UserID != accountID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "identity.unlinked",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     map[string]any{"id": row.ID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// Capabilities returns the feature/capability flags available to the account.
// Flags derive from the account status held in the user envelope.
func (a *pgAccountStore) Capabilities(ctx context.Context, projectID, accountID string) (map[string]bool, error) {
	row, err := a.accountLoadUser(ctx, projectID, accountID)
	if err != nil {
		return nil, err
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, err
	}
	active := acc.Status == "active"
	return map[string]bool{
		"can_login":           active,
		"can_update_profile":  active,
		"can_manage_sessions": active,
		"email_verified":      acc.EmailVerified,
	}, nil
}

// Activity returns the account's paginated activity log. The cursor is the id of
// the last event from the previous page (keyset over the `at` ordering).
func (a *pgAccountStore) Activity(ctx context.Context, cmd domain.AccountActivityCmd) (*domain.AccountActivityPage, error) {
	limit := cmd.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	qmods := []bob.Mod[*dialect.SelectQuery]{
		sm.Where(models.IamActivities.Columns.UserID.EQ(psql.Arg(cmd.AccountID))),
	}
	if cmd.Type != "" {
		qmods = append(qmods, sm.Where(models.IamActivities.Columns.Type.EQ(psql.Arg(cmd.Type))))
	}
	if cmd.Cursor != "" {
		// keyset: rows strictly older than the cursor row's id (lexically safe
		// for time-ordered uuids/ulids used as activity ids).
		qmods = append(qmods, sm.Where(models.IamActivities.Columns.ID.LT(psql.Arg(cmd.Cursor))))
	}
	qmods = append(qmods,
		sm.OrderBy(models.IamActivities.Columns.At).Desc(),
		sm.Limit(limit+1),
	)

	rows, err := models.IamActivities.Query(qmods...).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}

	page := &domain.AccountActivityPage{}
	if len(rows) > limit {
		page.HasMore = true
		rows = rows[:limit]
	}
	for _, row := range rows {
		ev := domain.AccountActivityEvent{
			ID:   row.ID,
			Type: row.Type,
			At:   row.At,
		}
		if len(row.Data) > 0 {
			var env struct {
				IP     string `json:"ip"`
				Device string `json:"device"`
			}
			if err := unmarshal(row.Data, &env); err != nil {
				return nil, err
			}
			ev.IP = env.IP
			ev.Device = env.Device
		}
		page.Events = append(page.Events, ev)
	}
	if page.HasMore && len(page.Events) > 0 {
		page.NextCursor = page.Events[len(page.Events)-1].ID
	}
	return page, nil
}

// Consents returns the account's recorded consent acceptances.
func (a *pgAccountStore) Consents(ctx context.Context, accountID string) ([]domain.AccountConsent, error) {
	rows, err := models.IamConsents.Query(
		sm.Where(models.IamConsents.Columns.UserID.EQ(psql.Arg(accountID))),
		sm.OrderBy(models.IamConsents.Columns.AcceptedAt).Desc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.AccountConsent, 0, len(rows))
	for _, row := range rows {
		c := domain.AccountConsent{
			Key:        row.DocKey,
			Version:    row.Version,
			AcceptedAt: row.AcceptedAt,
		}
		if v, ok := row.Locale.Get(); ok {
			c.Locale = v
		}
		out = append(out, c)
	}
	return out, nil
}

// AcceptConsents records consent acceptances and returns the updated set.
func (a *pgAccountStore) AcceptConsents(ctx context.Context, cmd domain.AccountAcceptConsentsCmd) ([]domain.AccountConsent, error) {
	_, err := withTxRet(ctx, a.db, func(ctx context.Context) (struct{}, error) {
		// resolve the project from the owning user so consent rows are scoped.
		user, err := models.FindIamUser(ctx, a.db.Bobx(), cmd.AccountID)
		if err != nil {
			if isNoRows(err) {
				return struct{}{}, domain.ErrUserNotFound
			}
			return struct{}{}, err
		}
		for _, acc := range cmd.Accept {
			now := nowUTC()
			setter := &models.IamConsentSetter{
				ID:         ptr(newUUID()),
				ProjectID:  ptr(user.ProjectID),
				UserID:     ptr(cmd.AccountID),
				DocKey:     ptr(acc.Key),
				Version:    ptr(acc.Version),
				AcceptedAt: ptr(now),
			}
			if _, err := models.IamConsents.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
				return struct{}{}, err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "account.consents_accepted",
			ProjectID:   user.ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     map[string]any{"account_id": cmd.AccountID, "project_id": user.ProjectID, "accepted": cmd.Accept},
		}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	return a.Consents(ctx, cmd.AccountID)
}

// StartExport kicks off a data-export job and returns its identifier.
func (a *pgAccountStore) StartExport(ctx context.Context, accountID string) (*domain.AccountExportJob, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AccountExportJob, error) {
		user, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			if isNoRows(err) {
				return nil, domain.ErrUserNotFound
			}
			return nil, err
		}
		jobID := newUUID()
		job := &domain.AccountExportJob{JobID: jobID, Status: "pending"}

		env := map[string]any{
			"job_id":     jobID,
			"account_id": accountID,
			"status":     job.Status,
		}
		raw, err := marshal(env)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		now := nowUTC()
		setter := &models.IamJobSetter{
			ID:        ptr(jobID),
			ProjectID: ptr(user.ProjectID),
			Type:      ptr("account_export"),
			Status:    ptr(job.Status),
			CreatedAt: ptr(now),
			UpdatedAt: ptr(now),
			Data:      &rm,
		}
		if _, err := models.IamJobs.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "account.export_started",
			ProjectID:   user.ProjectID,
			Environment: "",
			AggregateID: accountID,
			Payload:     job,
		}); err != nil {
			return nil, err
		}
		return job, nil
	})
}

// ExportStatus reports the state of a data-export job owned by the account.
func (a *pgAccountStore) ExportStatus(ctx context.Context, accountID, jobID string) (*domain.AccountExportJob, error) {
	row, err := models.FindIamJob(ctx, a.db.Bobx(), jobID)
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var env struct {
		JobID       string `json:"job_id"`
		AccountID   string `json:"account_id"`
		Status      string `json:"status"`
		DownloadURL string `json:"download_url"`
	}
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &env); err != nil {
			return nil, err
		}
	}
	if env.AccountID != accountID { // tenant/ownership boundary
		return nil, domain.ErrNotFound
	}
	job := &domain.AccountExportJob{
		JobID:       jobID,
		Status:      row.Status,
		DownloadURL: env.DownloadURL,
	}
	return job, nil
}

// StartIdentityMerge begins merging another identity into the account. A
// verification challenge is created with only the sha256 of the opaque code
// persisted; the plaintext code would be delivered out-of-band.
func (a *pgAccountStore) StartIdentityMerge(ctx context.Context, cmd domain.AccountMergeStartCmd) (*domain.Challenge, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Challenge, error) {
		user, err := models.FindIamUser(ctx, a.db.Bobx(), cmd.AccountID)
		if err != nil {
			if isNoRows(err) {
				return nil, domain.ErrUserNotFound
			}
			return nil, err
		}
		code, err := accountRandomCode()
		if err != nil {
			return nil, err
		}
		challengeID := newUUID()
		expires := nowUTC().Add(accountSeconds(600)) // 10m

		env := map[string]any{
			"account_id":        cmd.AccountID,
			"target_identifier": cmd.TargetIdentifier,
			"purpose":           "identity_merge",
		}
		raw, err := marshal(env)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		codeHash := null.From(accountHashToken(code))
		subject := null.From(cmd.TargetIdentifier)
		setter := &models.IamChallengeSetter{
			ID:        ptr(challengeID),
			ProjectID: ptr(user.ProjectID),
			Type:      ptr("identity_merge"),
			Subject:   &subject,
			CodeHash:  &codeHash, // only the hash is stored, never plaintext
			ExpiresAt: ptr(expires),
			Consumed:  ptr(false),
			CreatedAt: ptr(nowUTC()),
			Data:      &rm,
		}
		if _, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		ch := &domain.Challenge{
			ID:        challengeID,
			Type:      "identity_merge",
			ExpiresAt: expires,
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "account.identity_merge_started",
			ProjectID:   user.ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     ch,
		}); err != nil {
			return nil, err
		}
		return ch, nil
	})
}

// ConfirmIdentityMerge completes a pending identity merge: verifies the code
// against the stored hash, consumes the challenge, links the target identity to
// the account and returns the refreshed account + identity set.
func (a *pgAccountStore) ConfirmIdentityMerge(ctx context.Context, cmd domain.AccountMergeConfirmCmd) (*domain.Account, []domain.Identity, error) {
	type result struct {
		acc *domain.Account
		ids []domain.Identity
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		ch, err := models.FindIamChallenge(ctx, a.db.Bobx(), cmd.ChallengeID)
		if err != nil {
			if isNoRows(err) {
				return result{}, domain.ErrChallengeInvalid
			}
			return result{}, err
		}
		if ch.Consumed {
			return result{}, domain.ErrChallengeInvalid
		}
		if !ch.ExpiresAt.After(nowUTC()) {
			return result{}, domain.ErrChallengeExpired
		}
		var env struct {
			AccountID        string `json:"account_id"`
			TargetIdentifier string `json:"target_identifier"`
		}
		if len(ch.Data) > 0 {
			if err := unmarshal(ch.Data, &env); err != nil {
				return result{}, err
			}
		}
		if env.AccountID != cmd.AccountID {
			return result{}, domain.ErrChallengeInvalid
		}
		stored, ok := ch.CodeHash.Get()
		if !ok || stored != accountHashToken(cmd.Code) {
			return result{}, domain.ErrChallengeInvalid
		}

		user, err := a.accountLoadUser(ctx, ch.ProjectID, cmd.AccountID)
		if err != nil {
			return result{}, err
		}

		// consume the challenge (single-use).
		consumed := true
		if err := ch.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: &consumed}); err != nil {
			return result{}, err
		}

		// link the target identity to the account.
		identEmail := null.From(env.TargetIdentifier)
		idSetter := &models.IamIdentitySetter{
			ID:        ptr(newUUID()),
			ProjectID: ptr(ch.ProjectID),
			UserID:    ptr(cmd.AccountID),
			Type:      ptr("email"),
			Email:     &identEmail,
			CreatedAt: ptr(nowUTC()),
		}
		if _, err := models.IamIdentities.Insert(idSetter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return result{}, domain.ErrAlreadyLinked
			}
			return result{}, err
		}
		var acc domain.Account
		if err := unmarshal(user.Data, &acc); err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "account.identity_merged",
			ProjectID:   ch.ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     acc,
		}); err != nil {
			return result{}, err
		}
		return result{acc: &acc}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	// re-read identities outside the mutation for the refreshed set.
	ids, err := a.ListIdentities(ctx, cmd.AccountID)
	if err != nil {
		return nil, nil, err
	}
	res.ids = ids
	return res.acc, res.ids, nil
}
