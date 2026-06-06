package postgres

// Postgres adapters for the per-project administration ports declared in
// pkg/api/admin.go:
//
//   - pgAdminUsers          -> api.AdminUsers          (iam_users)
//   - pgAdminApps           -> api.AdminApps           (iam_app_clients + iam_app_secrets)
//   - pgAdminConfig         -> api.AdminConfig         (iam_config + iam_providers + iam_email_templates)
//   - pgAdminKeys           -> api.AdminKeys           (iam_signing_keys + iam_token_profiles)
//   - pgAdminAccessRequests -> api.AdminAccessRequests (iam_access_requests)
//
// Each aggregate is persisted as a `data jsonb` envelope; the typed columns
// (project_id, status, email, kind, key, environment, ...) are lookup-only and
// derived from the marshalled struct. Every query is scoped by project_id (the
// tenant boundary): a row whose project_id does not match the request is a
// not-found. Reads run on db.Bobx(); every mutation is wrapped in
// db.withTx / withTxRet (serializable + mandatory retry).
//
// Secrets/codes are minted with crypto/rand and persisted only as a sha256
// hash; passwords are bcrypt-hashed. Signing keys are generated here (RSA-2048
// PEM) and the impersonation link / token-profile preview are signed via the
// jwx Signer. Every domain-event emission point carries a
// `// TODO outbox event: <name>` comment (no outbox logic).

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/go-faster/jx"
	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"golang.org/x/crypto/bcrypt"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// adminDefaultEnvironment is the environment used when a command omits one.
// iam_config / iam_signing_keys default their environment column to "live".
const adminDefaultEnvironment = "live"

// adminTokenProfilePreviewTTL bounds the signed sample token returned by
// PreviewTokenProfile; it is a throwaway example, so a short lifetime is fine.
const adminTokenProfilePreviewTTL = 5 * time.Minute

func adminEnv(env string) string {
	if env == "" {
		return adminDefaultEnvironment
	}
	return env
}

// adminIsNotFound reports whether err is (or wraps) a no-rows / not-found from
// any layer: the raw pgx.ErrNoRows a bob .One() returns, the package
// ErrNotFound translatePgErr produces, or a domain not-found.
func adminIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, domain.ErrNotFound)
}

// adminRandomToken returns a URL-safe opaque token of n random bytes and its
// sha256 hex hash. Only the hash is ever persisted.
func adminRandomToken(n int) (token, hash string, err error) {
	buf := make([]byte, n)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, adminSHA256(token), nil
}

// adminSHA256 hashes an opaque token for at-rest storage.
func adminSHA256(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// =====================================================================
// AdminUsers — iam_users
// =====================================================================

type pgAdminUsers struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminUsers builds the Postgres-backed AdminUsers adapter.
func NewPgAdminUsers(db *DB, emitter Emitter) *pgAdminUsers {
	return &pgAdminUsers{db: db, emitter: emitter}
}

var _ api.AdminUsers = (*pgAdminUsers)(nil)

// findUser loads a user row enforcing the tenant boundary.
func (a *pgAdminUsers) findUser(ctx context.Context, projectID, accountID string) (*models.IamUser, *domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		if adminIsNotFound(translatePgErr("user", err)) {
			return nil, nil, domain.ErrUserNotFound
		}
		return nil, nil, err
	}
	if row.ProjectID != projectID {
		return nil, nil, domain.ErrUserNotFound
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, nil, err
	}
	return row, &acc, nil
}

func (a *pgAdminUsers) List(ctx context.Context, projectID string) ([]domain.Account, error) {
	rows, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Account, 0, len(rows))
	for _, row := range rows {
		var acc domain.Account
		if err := unmarshal(row.Data, &acc); err != nil {
			return nil, err
		}
		out = append(out, acc)
	}
	return out, nil
}

func (a *pgAdminUsers) Get(ctx context.Context, projectID, accountID string) (*domain.Account, error) {
	_, acc, err := a.findUser(ctx, projectID, accountID)
	return acc, err
}

func (a *pgAdminUsers) Create(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		now := nowUTC()
		acc := &domain.Account{
			ID:           newUUID(),
			ProjectID:    cmd.ProjectID,
			Kind:         "human",
			Status:       "active",
			PrimaryEmail: cmd.Email,
			PrimaryPhone: cmd.Phone,
			Name:         cmd.Name,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		raw, err := marshal(acc)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamUserSetter{
			ID:        &acc.ID,
			ProjectID: &acc.ProjectID,
			Kind:      ptr(acc.Kind),
			Status:    ptr(acc.Status),
			Data:      &rm,
		}
		if acc.PrimaryEmail != "" {
			v := null.From(acc.PrimaryEmail)
			setter.PrimaryEmail = &v
		}
		if acc.PrimaryPhone != "" {
			v := null.From(acc.PrimaryPhone)
			setter.PrimaryPhone = &v
		}
		if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrEmailExists
			}
			return nil, err
		}

		// Password credential (bcrypt) is stored on iam_credentials when supplied.
		if cmd.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			cred := map[string]any{"user_id": acc.ID, "type": "password"}
			craw, err := marshal(cred)
			if err != nil {
				return nil, err
			}
			crm := json.RawMessage(craw)
			cs := &models.IamCredentialSetter{
				ID:        ptr(newUUID()),
				ProjectID: &acc.ProjectID,
				UserID:    &acc.ID,
				Type:      ptr("password"),
				Secret:    ptr(string(hash)),
				Data:      &crm,
			}
			if _, err := models.IamCredentials.Insert(cs).One(ctx, a.db.Bobx()); err != nil {
				return nil, err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.created",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

func (a *pgAdminUsers) Update(ctx context.Context, cmd domain.AdminUserUpdateCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, cmd.ProjectID, cmd.AccountID)
		if err != nil {
			return nil, err
		}
		if cmd.Name != "" {
			acc.Name = cmd.Name
		}
		if cmd.Locale != "" {
			acc.Locale = cmd.Locale
		}
		acc.UpdatedAt = nowUTC()
		if err := a.persist(ctx, row, acc); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.updated",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

func (a *pgAdminUsers) Ban(ctx context.Context, projectID, accountID string) error {
	_, err := a.BanWith(ctx, domain.AdminUserBanCmd{ProjectID: projectID, AccountID: accountID})
	return err
}

func (a *pgAdminUsers) BanWith(ctx context.Context, cmd domain.AdminUserBanCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, cmd.ProjectID, cmd.AccountID)
		if err != nil {
			return nil, err
		}
		acc.Status = "banned"
		acc.UpdatedAt = nowUTC()
		if err := a.persistWithExtra(ctx, row, acc, func(m map[string]any) {
			if cmd.Reason != "" {
				m["ban_reason"] = cmd.Reason
			}
			if !cmd.Until.IsZero() {
				m["ban_until"] = cmd.Until.UTC()
			}
		}); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.banned",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

func (a *pgAdminUsers) Unban(ctx context.Context, projectID, accountID string) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, projectID, accountID)
		if err != nil {
			return nil, err
		}
		acc.Status = "active"
		acc.UpdatedAt = nowUTC()
		if err := a.persist(ctx, row, acc); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.unbanned",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

func (a *pgAdminUsers) Delete(ctx context.Context, projectID, accountID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.findUser(ctx, projectID, accountID)
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

func (a *pgAdminUsers) VerifyEmail(ctx context.Context, projectID, accountID string) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, projectID, accountID)
		if err != nil {
			return nil, err
		}
		acc.EmailVerified = true
		acc.UpdatedAt = nowUTC()
		if err := a.persist(ctx, row, acc); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.email_verified",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

func (a *pgAdminUsers) VerifyPhone(ctx context.Context, projectID, accountID string) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, projectID, accountID)
		if err != nil {
			return nil, err
		}
		acc.UpdatedAt = nowUTC()
		if err := a.persistWithExtra(ctx, row, acc, func(m map[string]any) {
			m["phone_verified"] = true
		}); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.phone_verified",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

func (a *pgAdminUsers) SetPassword(ctx context.Context, cmd domain.AdminUserPasswordCmd) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		// Tenant boundary.
		if _, _, err := a.findUser(ctx, cmd.ProjectID, cmd.AccountID); err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		existing, err := models.IamCredentials.Query(
			sm.Where(models.IamCredentials.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Where(models.IamCredentials.Columns.UserID.EQ(psql.Arg(cmd.AccountID))),
			sm.Where(models.IamCredentials.Columns.Type.EQ(psql.Arg("password"))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			cr := existing[0]
			if err := cr.Update(ctx, a.db.Bobx(), &models.IamCredentialSetter{
				Secret:    ptr(string(hash)),
				UpdatedAt: ptr(nowUTC()),
			}); err != nil {
				return err
			}
		} else {
			cred := map[string]any{"user_id": cmd.AccountID, "type": "password"}
			craw, err := marshal(cred)
			if err != nil {
				return err
			}
			crm := json.RawMessage(craw)
			if _, err := models.IamCredentials.Insert(&models.IamCredentialSetter{
				ID:        ptr(newUUID()),
				ProjectID: &cmd.ProjectID,
				UserID:    &cmd.AccountID,
				Type:      ptr("password"),
				Secret:    ptr(string(hash)),
				Data:      &crm,
			}).One(ctx, a.db.Bobx()); err != nil {
				return err
			}
		}
		if cmd.RevokeSessions {
			if _, err := a.revokeSessions(ctx, cmd.ProjectID, cmd.AccountID, ""); err != nil {
				return err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.password_set",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     map[string]any{"id": cmd.AccountID, "project_id": cmd.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminUsers) Anonymize(ctx context.Context, cmd domain.AdminUserAnonymizeCmd) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, acc, err := a.findUser(ctx, cmd.ProjectID, cmd.AccountID)
		if err != nil {
			return err
		}
		// Scrub PII from the aggregate and clear the lookup columns.
		acc.PrimaryEmail = ""
		acc.PrimaryPhone = ""
		acc.Name = ""
		acc.EmailVerified = false
		acc.Status = "deactivated"
		acc.UpdatedAt = nowUTC()
		raw, err := marshal(acc)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		nullEmail := null.FromPtr[string](nil)
		nullPhone := null.FromPtr[string](nil)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamUserSetter{
			Status:       ptr(acc.Status),
			PrimaryEmail: &nullEmail,
			PrimaryPhone: &nullPhone,
			Data:         &rm,
			UpdatedAt:    ptr(acc.UpdatedAt),
		}); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.anonymized",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.AccountID,
			Payload:     acc,
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminUsers) Export(ctx context.Context, projectID, accountID string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		if _, _, err := a.findUser(ctx, projectID, accountID); err != nil {
			return "", err
		}
		jobID := newUUID()
		job := map[string]any{"type": "user_export", "user_id": accountID, "status": "running"}
		jraw, err := marshal(job)
		if err != nil {
			return "", err
		}
		jrm := json.RawMessage(jraw)
		if _, err := models.IamJobs.Insert(&models.IamJobSetter{
			ID:        &jobID,
			ProjectID: &projectID,
			Type:      ptr("user_export"),
			Status:    ptr("running"),
			Data:      &jrm,
		}).One(ctx, a.db.Bobx()); err != nil {
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.export_requested",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: accountID,
			Payload:     map[string]any{"job_id": jobID, "user_id": accountID, "project_id": projectID},
		}); err != nil {
			return "", err
		}
		return jobID, nil
	})
}

func (a *pgAdminUsers) Impersonate(ctx context.Context, cmd domain.AdminUserImpersonateCmd) (*domain.AdminImpersonation, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminImpersonation, error) {
		if _, _, err := a.findUser(ctx, cmd.ProjectID, cmd.AccountID); err != nil {
			return nil, err
		}
		ttl := cmd.DurationSeconds
		if ttl <= 0 {
			ttl = 300
		}
		expiresAt := nowUTC().Add(time.Duration(ttl) * time.Second)
		// Short-TTL signed impersonation JWT (jwx, project Signer): typ=impersonation,
		// sub=target user, act=admin actor. Persist only its hash so the challenge row
		// can gate single-use redemption.
		token, err := a.db.Signer().Sign(ctx, cmd.ProjectID, adminDefaultEnvironment, map[string]any{
			"iss": cmd.ProjectID,
			"sub": cmd.AccountID,
			"pid": cmd.ProjectID,
			"act": cmd.ActorID,
			"typ": "impersonation",
			"env": adminDefaultEnvironment,
		}, time.Duration(ttl)*time.Second)
		if err != nil {
			return nil, err
		}
		hash := adminSHA256(token)
		ch := map[string]any{
			"type":     "impersonation",
			"user_id":  cmd.AccountID,
			"actor_id": cmd.ActorID,
			"reason":   cmd.Reason,
		}
		chraw, err := marshal(ch)
		if err != nil {
			return nil, err
		}
		chrm := json.RawMessage(chraw)
		subj := null.From(cmd.AccountID)
		codeHash := null.From(hash)
		if _, err := models.IamChallenges.Insert(&models.IamChallengeSetter{
			ID:        ptr(newUUID()),
			ProjectID: &cmd.ProjectID,
			Type:      ptr("impersonation"),
			Subject:   &subj,
			CodeHash:  &codeHash,
			ExpiresAt: &expiresAt,
			Data:      &chrm,
		}).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.impersonation_started",
			ProjectID:   cmd.ProjectID,
			Environment: adminDefaultEnvironment,
			AggregateID: cmd.AccountID,
			Payload:     map[string]any{"user_id": cmd.AccountID, "actor_id": cmd.ActorID, "project_id": cmd.ProjectID, "reason": cmd.Reason},
		}); err != nil {
			return nil, err
		}
		return &domain.AdminImpersonation{
			URL:       fmt.Sprintf("/impersonate?token=%s", token),
			ExpiresAt: expiresAt,
		}, nil
	})
}

func (a *pgAdminUsers) ResetMFA(ctx context.Context, projectID, accountID string, factorIDs []string) (int, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (int, error) {
		if _, _, err := a.findUser(ctx, projectID, accountID); err != nil {
			return 0, err
		}
		factors, err := models.IamFactors.Query(
			sm.Where(models.IamFactors.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamFactors.Columns.UserID.EQ(psql.Arg(accountID))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return 0, err
		}
		want := make(map[string]struct{}, len(factorIDs))
		for _, id := range factorIDs {
			want[id] = struct{}{}
		}
		removed := 0
		for _, f := range factors {
			if len(want) > 0 {
				if _, ok := want[f.ID]; !ok {
					continue
				}
			}
			if err := f.Delete(ctx, a.db.Bobx()); err != nil {
				return removed, err
			}
			removed++
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.mfa_reset",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: accountID,
			Payload:     map[string]any{"id": accountID, "project_id": projectID, "removed": removed},
		}); err != nil {
			return 0, err
		}
		return removed, nil
	})
}

func (a *pgAdminUsers) ListIdentities(ctx context.Context, projectID, accountID string) ([]domain.Identity, error) {
	if _, _, err := a.findUser(ctx, projectID, accountID); err != nil {
		return nil, err
	}
	rows, err := models.IamIdentities.Query(
		sm.Where(models.IamIdentities.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamIdentities.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Identity, 0, len(rows))
	for _, row := range rows {
		var id domain.Identity
		if err := unmarshal(row.Data, &id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func (a *pgAdminUsers) DeleteIdentity(ctx context.Context, projectID, accountID, identityID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamIdentity(ctx, a.db.Bobx(), identityID)
		if err != nil {
			if adminIsNotFound(translatePgErr("identity", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.UserID != accountID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.identity_deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: identityID,
			Payload:     map[string]any{"id": identityID, "user_id": accountID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminUsers) ListSessions(ctx context.Context, projectID, accountID string) ([]domain.Session, error) {
	if _, _, err := a.findUser(ctx, projectID, accountID); err != nil {
		return nil, err
	}
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		var s domain.Session
		if err := unmarshal(row.Data, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (a *pgAdminUsers) DeleteSession(ctx context.Context, projectID, accountID, sessionID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
		if err != nil {
			if adminIsNotFound(translatePgErr("session", err)) {
				return domain.ErrSessionNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.UserID != accountID {
			return domain.ErrSessionNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.session_revoked",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: sessionID,
			Payload:     map[string]any{"id": sessionID, "user_id": accountID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminUsers) RevokeSessions(ctx context.Context, cmd domain.AdminUserSessionsRevokeCmd) (int, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (int, error) {
		if _, _, err := a.findUser(ctx, cmd.ProjectID, cmd.AccountID); err != nil {
			return 0, err
		}
		return a.revokeSessions(ctx, cmd.ProjectID, cmd.AccountID, cmd.ExceptSessionID)
	})
}

// revokeSessions deletes all sessions for a user except optionally one. The
// caller is responsible for the tenant boundary check and the transaction.
func (a *pgAdminUsers) revokeSessions(ctx context.Context, projectID, accountID, exceptID string) (int, error) {
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return 0, err
	}
	revoked := 0
	for _, row := range rows {
		if exceptID != "" && row.ID == exceptID {
			continue
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return revoked, err
		}
		revoked++
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "user.sessions_revoked",
		ProjectID:   projectID,
		Environment: "",
		AggregateID: accountID,
		Payload:     map[string]any{"id": accountID, "project_id": projectID, "revoked": revoked},
	}); err != nil {
		return 0, err
	}
	return revoked, nil
}

// persist writes the account aggregate back, keeping lookup columns in sync.
func (a *pgAdminUsers) persist(ctx context.Context, row *models.IamUser, acc *domain.Account) error {
	return a.persistWithExtra(ctx, row, acc, nil)
}

// persistWithExtra writes the account aggregate plus extra envelope fields
// (merged into the jsonb only) and keeps the status/email/phone columns in sync.
func (a *pgAdminUsers) persistWithExtra(ctx context.Context, row *models.IamUser, acc *domain.Account, extra func(map[string]any)) error {
	raw, err := marshal(acc)
	if err != nil {
		return err
	}
	if extra != nil {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return err
		}
		extra(m)
		if raw, err = json.Marshal(m); err != nil {
			return err
		}
	}
	rm := json.RawMessage(raw)
	setter := &models.IamUserSetter{
		Status:    ptr(acc.Status),
		Data:      &rm,
		UpdatedAt: ptr(acc.UpdatedAt),
	}
	if acc.PrimaryEmail != "" {
		v := null.From(acc.PrimaryEmail)
		setter.PrimaryEmail = &v
	}
	if acc.PrimaryPhone != "" {
		v := null.From(acc.PrimaryPhone)
		setter.PrimaryPhone = &v
	}
	if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailExists
		}
		return err
	}
	return nil
}

// =====================================================================
// AdminApps — iam_app_clients + iam_app_secrets
// =====================================================================

type pgAdminApps struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminApps builds the Postgres-backed AdminApps adapter.
func NewPgAdminApps(db *DB, emitter Emitter) *pgAdminApps {
	return &pgAdminApps{db: db, emitter: emitter}
}

var _ api.AdminApps = (*pgAdminApps)(nil)

func (a *pgAdminApps) findApp(ctx context.Context, projectID, appID string) (*models.IamAppClient, *domain.AppClient, error) {
	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), appID)
	if err != nil {
		if adminIsNotFound(translatePgErr("app_client", err)) {
			return nil, nil, domain.ErrClientNotFound
		}
		return nil, nil, err
	}
	if row.ProjectID != projectID {
		return nil, nil, domain.ErrClientNotFound
	}
	var app domain.AppClient
	if err := unmarshal(row.Data, &app); err != nil {
		return nil, nil, err
	}
	return row, &app, nil
}

func (a *pgAdminApps) List(ctx context.Context, projectID string) ([]domain.AppClient, error) {
	rows, err := models.IamAppClients.Query(
		sm.Where(models.IamAppClients.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.AppClient, 0, len(rows))
	for _, row := range rows {
		var app domain.AppClient
		if err := unmarshal(row.Data, &app); err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, nil
}

func (a *pgAdminApps) Get(ctx context.Context, projectID, appID string) (*domain.AppClient, error) {
	_, app, err := a.findApp(ctx, projectID, appID)
	return app, err
}

func (a *pgAdminApps) Create(ctx context.Context, cmd domain.AppClientCmd) (*domain.AppClient, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AppClient, error) {
		app := &domain.AppClient{
			ID:           newUUID(),
			ProjectID:    cmd.ProjectID,
			Name:         cmd.Name,
			Type:         cmd.Type,
			Environment:  adminDefaultEnvironment,
			RedirectURIs: cmd.RedirectURIs,
		}
		raw, err := marshal(app)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if _, err := models.IamAppClients.Insert(&models.IamAppClientSetter{
			ID:          &app.ID,
			ProjectID:   &app.ProjectID,
			Environment: ptr(app.Environment),
			Name:        ptr(app.Name),
			Type:        ptr(app.Type),
			Data:        &rm,
		}).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.created",
			ProjectID:   app.ProjectID,
			Environment: app.Environment,
			AggregateID: app.ID,
			Payload:     app,
		}); err != nil {
			return nil, err
		}
		return app, nil
	})
}

func (a *pgAdminApps) Update(ctx context.Context, projectID, appID string, patch map[string]any) (*domain.AppClient, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AppClient, error) {
		row, app, err := a.findApp(ctx, projectID, appID)
		if err != nil {
			return nil, err
		}
		if v, ok := patch["name"].(string); ok && v != "" {
			app.Name = v
		}
		if v, ok := patch["type"].(string); ok && v != "" {
			app.Type = v
		}
		if v, ok := patch["redirect_uris"].([]string); ok {
			app.RedirectURIs = v
		} else if v, ok := patch["redirect_uris"].([]any); ok {
			uris := make([]string, 0, len(v))
			for _, u := range v {
				if s, ok := u.(string); ok {
					uris = append(uris, s)
				}
			}
			app.RedirectURIs = uris
		}
		raw, err := marshal(app)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAppClientSetter{
			Name:      ptr(app.Name),
			Type:      ptr(app.Type),
			Data:      &rm,
			UpdatedAt: ptr(nowUTC()),
		}); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.updated",
			ProjectID:   app.ProjectID,
			Environment: app.Environment,
			AggregateID: app.ID,
			Payload:     app,
		}); err != nil {
			return nil, err
		}
		return app, nil
	})
}

func (a *pgAdminApps) Delete(ctx context.Context, projectID, appID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.findApp(ctx, projectID, appID)
		if err != nil {
			return err
		}
		// Cascade the app's secrets first.
		secrets, err := models.IamAppSecrets.Query(
			sm.Where(models.IamAppSecrets.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamAppSecrets.Columns.AppID.EQ(psql.Arg(appID))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return err
		}
		for _, s := range secrets {
			if err := s.Delete(ctx, a.db.Bobx()); err != nil {
				return err
			}
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: appID,
			Payload:     map[string]any{"id": appID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminApps) AddSecret(ctx context.Context, projectID, appID, name string) (*domain.AdminSecret, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminSecret, error) {
		if _, _, err := a.findApp(ctx, projectID, appID); err != nil {
			return nil, err
		}
		// Mint an opaque client secret; persist only its sha256 hash.
		secret, hash, err := adminRandomToken(32)
		if err != nil {
			return nil, err
		}
		secretID := newUUID()
		meta := map[string]any{"name": name}
		mraw, err := marshal(meta)
		if err != nil {
			return nil, err
		}
		mrm := json.RawMessage(mraw)
		if _, err := models.IamAppSecrets.Insert(&models.IamAppSecretSetter{
			ID:        &secretID,
			ProjectID: &projectID,
			AppID:     &appID,
			Hash:      &hash,
			Data:      &mrm,
		}).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.secret_created",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: secretID,
			Payload:     map[string]any{"id": secretID, "app_id": appID, "project_id": projectID},
		}); err != nil {
			return nil, err
		}
		return &domain.AdminSecret{
			SecretID:     secretID,
			ClientID:     appID,
			ClientSecret: secret,
		}, nil
	})
}

func (a *pgAdminApps) DeleteSecret(ctx context.Context, projectID, appID, secretID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamAppSecret(ctx, a.db.Bobx(), secretID)
		if err != nil {
			if adminIsNotFound(translatePgErr("app_secret", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.AppID != appID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.secret_deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: secretID,
			Payload:     map[string]any{"id": secretID, "app_id": appID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// =====================================================================
// AdminConfig — iam_config + iam_providers + iam_email_templates
// =====================================================================

type pgAdminConfig struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminConfig builds the Postgres-backed AdminConfig adapter.
func NewPgAdminConfig(db *DB, emitter Emitter) *pgAdminConfig {
	return &pgAdminConfig{db: db, emitter: emitter}
}

var _ api.AdminConfig = (*pgAdminConfig)(nil)

// getConfigDoc reads one iam_config(project, env, key) envelope as a doc map.
func (a *pgAdminConfig) getConfigDoc(ctx context.Context, projectID, env, key string) (domain.AdminConfigDoc, error) {
	row, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(adminEnv(env)))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg(key))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if adminIsNotFound(err) {
			return domain.AdminConfigDoc{}, nil // unset config is an empty doc
		}
		return nil, err
	}
	doc := domain.AdminConfigDoc{}
	if len(row.Data) > 0 {
		if err := json.Unmarshal(row.Data, &doc); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// putConfigDoc upserts one iam_config(project, env, key) envelope from a doc.
func (a *pgAdminConfig) putConfigDoc(ctx context.Context, projectID, env, key string, doc domain.AdminConfigDoc) (domain.AdminConfigDoc, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (domain.AdminConfigDoc, error) {
		raw, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		env = adminEnv(env)
		existing, err := models.IamConfigs.Query(
			sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(env))),
			sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg(key))),
		).One(ctx, a.db.Bobx())
		if err != nil && !adminIsNotFound(err) {
			return nil, err
		}
		if err == nil {
			if uerr := existing.Update(ctx, a.db.Bobx(), &models.IamConfigSetter{
				Data:      &rm,
				UpdatedAt: ptr(nowUTC()),
			}); uerr != nil {
				return nil, uerr
			}
		} else {
			if _, ierr := models.IamConfigs.Insert(&models.IamConfigSetter{
				ProjectID:   &projectID,
				Environment: &env,
				Key:         &key,
				Data:        &rm,
			}).One(ctx, a.db.Bobx()); ierr != nil {
				return nil, ierr
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.updated",
			ProjectID:   projectID,
			Environment: env,
			AggregateID: projectID,
			Payload:     map[string]any{"project_id": projectID, "environment": env, "key": key, "doc": doc},
		}); err != nil {
			return nil, err
		}
		return doc, nil
	})
}

func (a *pgAdminConfig) GetAuthConfig(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "auth")
}

func (a *pgAdminConfig) UpdateAuthConfig(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "auth", cmd.Doc)
}

func (a *pgAdminConfig) GetPasswordPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "password_policy")
}

func (a *pgAdminConfig) UpdatePasswordPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "password_policy", cmd.Doc)
}

func (a *pgAdminConfig) GetSessionPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "session_policy")
}

func (a *pgAdminConfig) UpdateSessionPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "session_policy", cmd.Doc)
}

func (a *pgAdminConfig) GetConsent(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "consent")
}

func (a *pgAdminConfig) PutConsent(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "consent", cmd.Doc)
}

func (a *pgAdminConfig) GetFeatures(ctx context.Context, cmd domain.AdminConfigGetCmd) (map[string]bool, error) {
	row, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(adminEnv(cmd.Environment)))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("features"))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if adminIsNotFound(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	out := map[string]bool{}
	if len(row.Data) > 0 {
		if err := json.Unmarshal(row.Data, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (a *pgAdminConfig) PutFeatures(ctx context.Context, cmd domain.AdminFeaturesUpdateCmd) (map[string]bool, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]bool, error) {
		raw, err := json.Marshal(cmd.Features)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		env := adminEnv(cmd.Environment)
		existing, err := models.IamConfigs.Query(
			sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(env))),
			sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("features"))),
		).One(ctx, a.db.Bobx())
		if err != nil && !adminIsNotFound(err) {
			return nil, err
		}
		if err == nil {
			if uerr := existing.Update(ctx, a.db.Bobx(), &models.IamConfigSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}); uerr != nil {
				return nil, uerr
			}
		} else {
			if _, ierr := models.IamConfigs.Insert(&models.IamConfigSetter{
				ProjectID:   &cmd.ProjectID,
				Environment: &env,
				Key:         ptr("features"),
				Data:        &rm,
			}).One(ctx, a.db.Bobx()); ierr != nil {
				return nil, ierr
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.features_updated",
			ProjectID:   cmd.ProjectID,
			Environment: env,
			AggregateID: cmd.ProjectID,
			Payload:     map[string]any{"project_id": cmd.ProjectID, "environment": env, "features": cmd.Features},
		}); err != nil {
			return nil, err
		}
		return cmd.Features, nil
	})
}

func (a *pgAdminConfig) GetI18n(ctx context.Context, cmd domain.AdminConfigGetCmd, locale string) (map[string]jx.Raw, error) {
	row, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(adminEnv(cmd.Environment)))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("i18n:"+locale))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if adminIsNotFound(err) {
			return map[string]jx.Raw{}, nil
		}
		return nil, err
	}
	out := map[string]jx.Raw{}
	if len(row.Data) > 0 {
		if err := json.Unmarshal(row.Data, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (a *pgAdminConfig) PutI18n(ctx context.Context, cmd domain.AdminI18nUpdateCmd) (map[string]jx.Raw, error) {
	doc := domain.AdminConfigDoc(cmd.Messages)
	// Persist + emit atomically (nested withTx joins putConfigDoc's tx).
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		if _, err := a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "i18n:"+cmd.Locale, doc); err != nil {
			return err
		}
		return a.emitter.Emit(ctx, domain.Event{
			Type:        "config.i18n_updated",
			ProjectID:   cmd.ProjectID,
			Environment: adminEnv(cmd.Environment),
			AggregateID: cmd.ProjectID,
			Payload:     map[string]any{"project_id": cmd.ProjectID, "environment": cmd.Environment, "locale": cmd.Locale, "messages": cmd.Messages},
		})
	}); err != nil {
		return nil, err
	}
	return cmd.Messages, nil
}

// ----- providers (email / sms) -----

// adminProviderData mirrors the provider config persisted in the iam_providers
// data envelope; the kind/provider/enabled columns are lookup-only.
type adminProviderData struct {
	Type   string            `json:"type"`
	Config map[string]jx.Raw `json:"config"`
}

func (a *pgAdminConfig) listProviders(ctx context.Context, projectID, kind string) ([]domain.AdminProvider, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg(kind))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdminProvider, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminProviderToDomain(row))
	}
	return out, nil
}

func adminProviderToDomain(row *models.IamProvider) domain.AdminProvider {
	p := domain.AdminProvider{ID: row.ID, Type: row.Provider, Enabled: row.Enabled}
	if len(row.Data) > 0 {
		var d adminProviderData
		if err := json.Unmarshal(row.Data, &d); err == nil {
			if d.Type != "" {
				p.Type = d.Type
			}
			p.Config = d.Config
		}
	}
	return p
}

func (a *pgAdminConfig) createProvider(ctx context.Context, kind string, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminProvider, error) {
		id := cmd.ID
		if id == "" {
			id = newUUID()
		}
		d := adminProviderData{Type: cmd.Type, Config: cmd.Config}
		raw, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if _, err := models.IamProviders.Insert(&models.IamProviderSetter{
			ID:        &id,
			ProjectID: &cmd.ProjectID,
			Kind:      &kind,
			Provider:  ptr(cmd.Type),
			Enabled:   ptr(cmd.Enabled),
			Data:      &rm,
		}).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		p := &domain.AdminProvider{ID: id, Type: cmd.Type, Config: cmd.Config, Enabled: cmd.Enabled}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.provider_created",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: id,
			Payload:     p,
		}); err != nil {
			return nil, err
		}
		return p, nil
	})
}

func (a *pgAdminConfig) updateProvider(ctx context.Context, kind string, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminProvider, error) {
		row, err := models.FindIamProvider(ctx, a.db.Bobx(), cmd.ID)
		if err != nil {
			if adminIsNotFound(translatePgErr("provider", err)) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		if row.ProjectID != cmd.ProjectID || row.Kind != kind {
			return nil, domain.ErrNotFound
		}
		d := adminProviderData{Type: cmd.Type, Config: cmd.Config}
		raw, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamProviderSetter{
			Provider:  ptr(cmd.Type),
			Enabled:   ptr(cmd.Enabled),
			Data:      &rm,
			UpdatedAt: ptr(nowUTC()),
		}); err != nil {
			return nil, err
		}
		p := &domain.AdminProvider{ID: cmd.ID, Type: cmd.Type, Config: cmd.Config, Enabled: cmd.Enabled}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.provider_updated",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.ID,
			Payload:     p,
		}); err != nil {
			return nil, err
		}
		return p, nil
	})
}

func (a *pgAdminConfig) deleteProvider(ctx context.Context, kind string, cmd domain.AdminProviderDeleteCmd) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamProvider(ctx, a.db.Bobx(), cmd.ID)
		if err != nil {
			if adminIsNotFound(translatePgErr("provider", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != cmd.ProjectID || row.Kind != kind {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.provider_deleted",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.ID,
			Payload:     map[string]any{"id": cmd.ID, "project_id": cmd.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminConfig) ListEmailProviders(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminProvider, error) {
	return a.listProviders(ctx, cmd.ProjectID, "email")
}

func (a *pgAdminConfig) CreateEmailProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	return a.createProvider(ctx, "email", cmd)
}

func (a *pgAdminConfig) UpdateEmailProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	return a.updateProvider(ctx, "email", cmd)
}

func (a *pgAdminConfig) DeleteEmailProvider(ctx context.Context, cmd domain.AdminProviderDeleteCmd) error {
	return a.deleteProvider(ctx, "email", cmd)
}

func (a *pgAdminConfig) ListSmsProviders(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminProvider, error) {
	return a.listProviders(ctx, cmd.ProjectID, "sms")
}

func (a *pgAdminConfig) CreateSmsProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	return a.createProvider(ctx, "sms", cmd)
}

func (a *pgAdminConfig) UpdateSmsProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	return a.updateProvider(ctx, "sms", cmd)
}

func (a *pgAdminConfig) DeleteSmsProvider(ctx context.Context, cmd domain.AdminProviderDeleteCmd) error {
	return a.deleteProvider(ctx, "sms", cmd)
}

// ----- email templates (iam_email_templates) -----

// adminTemplateLocale is the locale a template is stored/keyed under when a
// command does not carry one.
const adminTemplateLocale = "en"

func (a *pgAdminConfig) ListEmailTemplates(ctx context.Context, cmd domain.AdminConfigGetCmd) (map[string]jx.Raw, error) {
	rows, err := models.IamEmailTemplates.Query(
		sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make(map[string]jx.Raw, len(rows))
	for _, row := range rows {
		out[row.Key] = jx.Raw(row.Data)
	}
	return out, nil
}

func (a *pgAdminConfig) UpdateEmailTemplate(ctx context.Context, cmd domain.AdminTemplateUpdateCmd) (map[string]jx.Raw, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]jx.Raw, error) {
		existing, err := models.IamEmailTemplates.Query(
			sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(cmd.TemplateID))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return nil, err
		}
		// Merge the patch onto the current template body.
		body := map[string]jx.Raw{}
		var cur *models.IamEmailTemplate
		if len(existing) > 0 {
			cur = existing[0]
			if len(cur.Data) > 0 {
				if err := json.Unmarshal(cur.Data, &body); err != nil {
					return nil, err
				}
			}
		}
		for k, v := range cmd.Patch {
			body[k] = v
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if cur != nil {
			if err := cur.Update(ctx, a.db.Bobx(), &models.IamEmailTemplateSetter{
				Data:      &rm,
				UpdatedAt: ptr(nowUTC()),
			}); err != nil {
				return nil, err
			}
		} else {
			if _, err := models.IamEmailTemplates.Insert(&models.IamEmailTemplateSetter{
				ID:        ptr(newUUID()),
				ProjectID: &cmd.ProjectID,
				Key:       ptr(cmd.TemplateID),
				Locale:    ptr(adminTemplateLocale),
				Data:      &rm,
			}).One(ctx, a.db.Bobx()); err != nil {
				return nil, err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.email_template_updated",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.TemplateID,
			Payload:     map[string]any{"project_id": cmd.ProjectID, "template_id": cmd.TemplateID, "body": body},
		}); err != nil {
			return nil, err
		}
		return body, nil
	})
}

func (a *pgAdminConfig) PreviewEmailTemplate(ctx context.Context, cmd domain.AdminTemplatePreviewCmd) (*domain.AdminTemplatePreview, error) {
	rows, err := models.IamEmailTemplates.Query(
		sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(cmd.TemplateID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	body := map[string]string{}
	if len(rows[0].Data) > 0 {
		_ = json.Unmarshal(rows[0].Data, &body) // best-effort: only string fields render
	}
	// NOTE: real rendering (merge cmd.Data into the template) lives in the
	// notification layer; here we surface the stored subject/html/text as-is.
	return &domain.AdminTemplatePreview{
		Subject: body["subject"],
		HTML:    body["html"],
		Text:    body["text"],
	}, nil
}

func (a *pgAdminConfig) SendTestEmail(ctx context.Context, cmd domain.AdminTemplateSendTestCmd) error {
	// Tenant boundary: the template must exist for the project.
	rows, err := models.IamEmailTemplates.Query(
		sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(cmd.TemplateID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return domain.ErrNotFound
	}
	// Actual delivery is the notification layer's responsibility (no SMTP here).
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "config.test_email_requested",
		ProjectID:   cmd.ProjectID,
		Environment: "",
		AggregateID: cmd.TemplateID,
		Payload:     map[string]any{"project_id": cmd.ProjectID, "template_id": cmd.TemplateID, "to": cmd.To},
	}); err != nil {
		return err
	}
	return nil
}

// =====================================================================
// AdminKeys — iam_signing_keys + iam_token_profiles
// =====================================================================

type pgAdminKeys struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminKeys builds the Postgres-backed AdminKeys adapter.
func NewPgAdminKeys(db *DB, emitter Emitter) *pgAdminKeys {
	return &pgAdminKeys{db: db, emitter: emitter}
}

var _ api.AdminKeys = (*pgAdminKeys)(nil)

func adminSigningKeyToDomain(row *models.IamSigningKey) domain.AdminSigningKey {
	return domain.AdminSigningKey{
		Kid:       row.Kid,
		Alg:       row.Alg,
		Use:       row.Use,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
	}
}

func (a *pgAdminKeys) ListSigningKeys(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminSigningKey, error) {
	rows, err := models.IamSigningKeys.Query(
		sm.Where(models.IamSigningKeys.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamSigningKeys.Columns.Environment.EQ(psql.Arg(adminEnv(cmd.Environment)))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdminSigningKey, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminSigningKeyToDomain(row))
	}
	return out, nil
}

func (a *pgAdminKeys) DeleteSigningKey(ctx context.Context, cmd domain.AdminConfigGetCmd, kid string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSigningKey(ctx, a.db.Bobx(), kid)
		if err != nil {
			if adminIsNotFound(translatePgErr("signing_key", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != cmd.ProjectID || row.Environment != adminEnv(cmd.Environment) {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "keys.signing_key_deleted",
			ProjectID:   cmd.ProjectID,
			Environment: adminEnv(cmd.Environment),
			AggregateID: kid,
			Payload:     map[string]any{"kid": kid, "project_id": cmd.ProjectID, "environment": cmd.Environment},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminKeys) RotateSigningKeys(ctx context.Context, cmd domain.AdminJWKSRotateCmd) (*domain.AdminSigningKey, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminSigningKey, error) {
		env := adminEnv(cmd.Environment)
		status := "inactive"
		if cmd.Activate {
			status = "active"
			// Demote the currently-active key(s) when activating the new one.
			active, err := models.IamSigningKeys.Query(
				sm.Where(models.IamSigningKeys.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
				sm.Where(models.IamSigningKeys.Columns.Environment.EQ(psql.Arg(env))),
				sm.Where(models.IamSigningKeys.Columns.Status.EQ(psql.Arg("active"))),
			).All(ctx, a.db.Bobx())
			if err != nil {
				return nil, err
			}
			for _, k := range active {
				if uerr := k.Update(ctx, a.db.Bobx(), &models.IamSigningKeySetter{Status: ptr("retired")}); uerr != nil {
					return nil, uerr
				}
			}
		}
		kid := newUUID()
		key := domain.AdminSigningKey{
			Kid:       kid,
			Alg:       "RS256",
			Use:       "sig",
			Status:    status,
			CreatedAt: nowUTC(),
		}
		raw, err := marshal(key)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		// Generate the RSA-2048 private key material and persist it (PEM) so the
		// Signer can mint/verify project tokens with this kid.
		pemStr, err := newRSAKeyPEM()
		if err != nil {
			return nil, err
		}
		encPem, err := a.db.Cipher.Encrypt(pemStr)
		if err != nil {
			return nil, err
		}
		pv := null.From(encPem)
		if _, err := models.IamSigningKeys.Insert(&models.IamSigningKeySetter{
			Kid:         &kid,
			ProjectID:   &cmd.ProjectID,
			Environment: &env,
			Alg:         ptr(key.Alg),
			Use:         ptr(key.Use),
			Status:      ptr(key.Status),
			PrivatePem:  &pv,
			Data:        &rm,
		}).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "keys.signing_keys_rotated",
			ProjectID:   cmd.ProjectID,
			Environment: env,
			AggregateID: kid,
			Payload:     &key,
		}); err != nil {
			return nil, err
		}
		return &key, nil
	})
}

func (a *pgAdminKeys) ActivateSigningKey(ctx context.Context, cmd domain.AdminConfigGetCmd, kid string) (*domain.AdminSigningKey, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminSigningKey, error) {
		env := adminEnv(cmd.Environment)
		row, err := models.FindIamSigningKey(ctx, a.db.Bobx(), kid)
		if err != nil {
			if adminIsNotFound(translatePgErr("signing_key", err)) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		if row.ProjectID != cmd.ProjectID || row.Environment != env {
			return nil, domain.ErrNotFound
		}
		// Retire the current active key(s) before promoting this one.
		active, err := models.IamSigningKeys.Query(
			sm.Where(models.IamSigningKeys.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Where(models.IamSigningKeys.Columns.Environment.EQ(psql.Arg(env))),
			sm.Where(models.IamSigningKeys.Columns.Status.EQ(psql.Arg("active"))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return nil, err
		}
		for _, k := range active {
			if k.Kid == kid {
				continue
			}
			if uerr := k.Update(ctx, a.db.Bobx(), &models.IamSigningKeySetter{Status: ptr("retired")}); uerr != nil {
				return nil, uerr
			}
		}
		if err := row.Update(ctx, a.db.Bobx(), &models.IamSigningKeySetter{Status: ptr("active")}); err != nil {
			return nil, err
		}
		out := adminSigningKeyToDomain(row)
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "keys.signing_key_activated",
			ProjectID:   cmd.ProjectID,
			Environment: env,
			AggregateID: kid,
			Payload:     &out,
		}); err != nil {
			return nil, err
		}
		return &out, nil
	})
}

// ----- token profiles (iam_token_profiles) -----

func adminTokenProfileToDomain(row *models.IamTokenProfile) domain.AdminTokenProfile {
	p := domain.AdminTokenProfile{ID: row.ID, Name: row.Name}
	if len(row.Data) > 0 {
		var d struct {
			Name           string            `json:"name"`
			Audience       string            `json:"audience"`
			AccessTTL      int               `json:"access_ttl"`
			RefreshTTL     int               `json:"refresh_ttl"`
			ClaimsTemplate map[string]jx.Raw `json:"claims_template"`
		}
		if err := json.Unmarshal(row.Data, &d); err == nil {
			if d.Name != "" {
				p.Name = d.Name
			}
			p.Audience = d.Audience
			p.AccessTTL = d.AccessTTL
			p.RefreshTTL = d.RefreshTTL
			p.ClaimsTemplate = d.ClaimsTemplate
		}
	}
	return p
}

func (a *pgAdminKeys) ListTokenProfiles(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminTokenProfile, error) {
	rows, err := models.IamTokenProfiles.Query(
		sm.Where(models.IamTokenProfiles.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdminTokenProfile, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminTokenProfileToDomain(row))
	}
	return out, nil
}

// adminProfileName extracts the "name" field from a profile doc, if present.
func adminProfileName(doc domain.AdminConfigDoc) string {
	if raw, ok := doc["name"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return ""
}

func (a *pgAdminKeys) CreateTokenProfile(ctx context.Context, cmd domain.AdminTokenProfileCmd) (*domain.AdminTokenProfile, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminTokenProfile, error) {
		id := cmd.ID
		if id == "" {
			id = newUUID()
		}
		name := adminProfileName(cmd.Profile)
		raw, err := json.Marshal(cmd.Profile)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if _, err := models.IamTokenProfiles.Insert(&models.IamTokenProfileSetter{
			ID:        &id,
			ProjectID: &cmd.ProjectID,
			Name:      &name,
			Data:      &rm,
		}).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		row, err := models.FindIamTokenProfile(ctx, a.db.Bobx(), id)
		if err != nil {
			return nil, err
		}
		out := adminTokenProfileToDomain(row)
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "keys.token_profile_created",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: id,
			Payload:     &out,
		}); err != nil {
			return nil, err
		}
		return &out, nil
	})
}

func (a *pgAdminKeys) UpdateTokenProfile(ctx context.Context, cmd domain.AdminTokenProfileCmd) (*domain.AdminTokenProfile, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminTokenProfile, error) {
		row, err := models.FindIamTokenProfile(ctx, a.db.Bobx(), cmd.ID)
		if err != nil {
			if adminIsNotFound(translatePgErr("token_profile", err)) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		if row.ProjectID != cmd.ProjectID {
			return nil, domain.ErrNotFound
		}
		name := adminProfileName(cmd.Profile)
		if name == "" {
			name = row.Name
		}
		raw, err := json.Marshal(cmd.Profile)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamTokenProfileSetter{
			Name:      &name,
			Data:      &rm,
			UpdatedAt: ptr(nowUTC()),
		}); err != nil {
			return nil, err
		}
		out := adminTokenProfileToDomain(row)
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "keys.token_profile_updated",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.ID,
			Payload:     &out,
		}); err != nil {
			return nil, err
		}
		return &out, nil
	})
}

func (a *pgAdminKeys) DeleteTokenProfile(ctx context.Context, cmd domain.AdminConfigGetCmd, profileID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamTokenProfile(ctx, a.db.Bobx(), profileID)
		if err != nil {
			if adminIsNotFound(translatePgErr("token_profile", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != cmd.ProjectID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "keys.token_profile_deleted",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: profileID,
			Payload:     map[string]any{"id": profileID, "project_id": cmd.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgAdminKeys) PreviewTokenProfile(ctx context.Context, cmd domain.AdminTokenProfilePreviewCmd) (map[string]jx.Raw, error) {
	row, err := models.FindIamTokenProfile(ctx, a.db.Bobx(), cmd.ProfileID)
	if err != nil {
		if adminIsNotFound(translatePgErr("token_profile", err)) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if row.ProjectID != cmd.ProjectID {
		return nil, domain.ErrNotFound
	}
	// Surface the profile's claims template as the previewed claim set, AND mint a
	// real signed sample token over those same claims with the project's active
	// signing key (jwx, db.Signer()). The signed JWT is returned alongside the
	// claims under the synthetic "_sample_token" key so callers can inspect both
	// the resolved claims and a verifiable example token.
	profile := adminTokenProfileToDomain(row)
	claims := profile.ClaimsTemplate
	if claims == nil {
		claims = map[string]jx.Raw{}
	}
	// Decode each jx.Raw claim into a plain value for the Signer's claim map.
	sampleClaims := make(map[string]any, len(claims)+1)
	for k, v := range claims {
		var dv any
		if err := json.Unmarshal([]byte(v), &dv); err != nil {
			return nil, err
		}
		sampleClaims[k] = dv
	}
	if cmd.UserID != "" {
		sampleClaims["sub"] = cmd.UserID
	}
	sampleClaims["typ"] = "access"
	sampleClaims["env"] = adminEnv(cmd.Environment)
	token, err := a.db.Signer().Sign(ctx, cmd.ProjectID, adminEnv(cmd.Environment), sampleClaims, adminTokenProfilePreviewTTL)
	if err != nil {
		return nil, err
	}
	out := make(map[string]jx.Raw, len(claims)+1)
	for k, v := range claims {
		out[k] = v
	}
	tokraw, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	out["_sample_token"] = jx.Raw(tokraw)
	return out, nil
}

// =====================================================================
// AdminAccessRequests — iam_access_requests
// =====================================================================

type pgAdminAccessRequests struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminAccessRequests builds the Postgres-backed AdminAccessRequests adapter.
func NewPgAdminAccessRequests(db *DB, emitter Emitter) *pgAdminAccessRequests {
	return &pgAdminAccessRequests{db: db, emitter: emitter}
}

var _ api.AdminAccessRequests = (*pgAdminAccessRequests)(nil)

// adminAccessRequestPageSize bounds one access-request listing page.
const adminAccessRequestPageSize = 50

func accessRequestToDomain(row *models.IamAccessRequest) domain.CoreAuthAccessRequest {
	ar := domain.CoreAuthAccessRequest{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Email:     row.Email,
		Status:    row.Status,
	}
	if len(row.Data) > 0 {
		_ = unmarshal(row.Data, &ar) // envelope carries Reason and any extra fields
	}
	// Keep lookup columns authoritative over the envelope copy.
	ar.ID = row.ID
	ar.ProjectID = row.ProjectID
	ar.Email = row.Email
	ar.Status = row.Status
	return ar
}

func (a *pgAdminAccessRequests) findRequest(ctx context.Context, projectID, requestID string) (*models.IamAccessRequest, error) {
	row, err := models.FindIamAccessRequest(ctx, a.db.Bobx(), requestID)
	if err != nil {
		if adminIsNotFound(translatePgErr("access_request", err)) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if row.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}
	return row, nil
}

func (a *pgAdminAccessRequests) List(ctx context.Context, cmd domain.AdminAccessRequestListCmd) (*domain.AdminAccessRequestPage, error) {
	mods := []bob.Mod[*dialect.SelectQuery]{
		sm.Where(models.IamAccessRequests.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
	}
	if cmd.Status != "" {
		mods = append(mods, sm.Where(models.IamAccessRequests.Columns.Status.EQ(psql.Arg(cmd.Status))))
	}
	// Keyset pagination on the id column (lexicographic cursor).
	if cmd.Cursor != "" {
		mods = append(mods, sm.Where(models.IamAccessRequests.Columns.ID.GT(psql.Arg(cmd.Cursor))))
	}
	mods = append(mods, sm.OrderBy(models.IamAccessRequests.Columns.ID))
	mods = append(mods, sm.Limit(adminAccessRequestPageSize+1))

	rows, err := models.IamAccessRequests.Query(mods...).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	page := &domain.AdminAccessRequestPage{}
	if len(rows) > adminAccessRequestPageSize {
		page.HasMore = true
		rows = rows[:adminAccessRequestPageSize]
	}
	page.Items = make([]domain.CoreAuthAccessRequest, 0, len(rows))
	for _, row := range rows {
		page.Items = append(page.Items, accessRequestToDomain(row))
	}
	if page.HasMore && len(rows) > 0 {
		page.NextCursor = rows[len(rows)-1].ID
	}
	return page, nil
}

func (a *pgAdminAccessRequests) Approve(ctx context.Context, cmd domain.AdminAccessRequestDecisionCmd) (map[string]jx.Raw, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]jx.Raw, error) {
		row, err := a.findRequest(ctx, cmd.ProjectID, cmd.RequestID)
		if err != nil {
			return nil, err
		}
		ar := accessRequestToDomain(row)
		ar.Status = "approved"
		if err := a.persistDecision(ctx, row, ar, cmd.ActorID, ""); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "access_request.approved",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: ar.ID,
			Payload:     ar,
		}); err != nil {
			return nil, err
		}
		out := map[string]jx.Raw{
			"id":     jx.Raw(adminJSONString(ar.ID)),
			"status": jx.Raw(adminJSONString(ar.Status)),
			"email":  jx.Raw(adminJSONString(ar.Email)),
		}
		return out, nil
	})
}

func (a *pgAdminAccessRequests) Deny(ctx context.Context, cmd domain.AdminAccessRequestDecisionCmd) (*domain.CoreAuthAccessRequest, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.CoreAuthAccessRequest, error) {
		row, err := a.findRequest(ctx, cmd.ProjectID, cmd.RequestID)
		if err != nil {
			return nil, err
		}
		ar := accessRequestToDomain(row)
		ar.Status = "denied"
		ar.Reason = cmd.Reason
		if err := a.persistDecision(ctx, row, ar, cmd.ActorID, cmd.Reason); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "access_request.denied",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: ar.ID,
			Payload:     ar,
		}); err != nil {
			return nil, err
		}
		return &ar, nil
	})
}

// persistDecision writes the decided access request back, keeping the status
// column in sync and recording the deciding actor/reason in the envelope.
func (a *pgAdminAccessRequests) persistDecision(ctx context.Context, row *models.IamAccessRequest, ar domain.CoreAuthAccessRequest, actorID, reason string) error {
	raw, err := marshal(ar)
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	m["decided_by"] = actorID
	m["decided_at"] = nowUTC()
	if reason != "" {
		m["reason"] = reason
	}
	merged, err := json.Marshal(m)
	if err != nil {
		return err
	}
	rm := json.RawMessage(merged)
	return row.Update(ctx, a.db.Bobx(), &models.IamAccessRequestSetter{
		Status:    ptr(ar.Status),
		Data:      &rm,
		UpdatedAt: ptr(nowUTC()),
	})
}

// adminJSONString encodes s as a JSON string literal for embedding in a jx.Raw.
func adminJSONString(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}
