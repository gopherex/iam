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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"text/template"
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
// any layer: the sql.ErrNoRows a bob .One() returns (bob wraps the pgx driver in
// database/sql semantics), the raw pgx.ErrNoRows, the package ErrNotFound
// translatePgErr produces, or a domain not-found.
func adminIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, pgx.ErrNoRows) ||
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

// findUser loads a user row enforcing the (project, environment) tenant boundary.
func (a *pgAdminUsers) findUser(ctx context.Context, projectID, environment, accountID string) (*models.IamUser, *domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		if adminIsNotFound(translatePgErr("user", err)) {
			return nil, nil, domain.ErrUserNotFound
		}
		return nil, nil, err
	}
	if row.ProjectID != projectID || row.Environment != adminEnv(environment) {
		return nil, nil, domain.ErrUserNotFound
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, nil, err
	}
	return row, &acc, nil
}

func (a *pgAdminUsers) List(ctx context.Context, projectID, environment string) ([]domain.Account, error) {
	rows, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamUsers.Columns.Environment.EQ(psql.Arg(adminEnv(environment)))),
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

func (a *pgAdminUsers) Get(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error) {
	_, acc, err := a.findUser(ctx, projectID, environment, accountID)
	return acc, err
}

func (a *pgAdminUsers) Create(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		now := nowUTC()
		env := adminEnv(cmd.Environment)
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
			ID:          &acc.ID,
			ProjectID:   &acc.ProjectID,
			Environment: &env,
			Kind:        ptr(acc.Kind),
			Status:      ptr(acc.Status),
			Data:        &rm,
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
			Environment: env,
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
		row, acc, err := a.findUser(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID)
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

func (a *pgAdminUsers) Ban(ctx context.Context, projectID, environment, accountID string) error {
	_, err := a.BanWith(ctx, domain.AdminUserBanCmd{ProjectID: projectID, Environment: environment, AccountID: accountID})
	return err
}

func (a *pgAdminUsers) BanWith(ctx context.Context, cmd domain.AdminUserBanCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID)
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

func (a *pgAdminUsers) Unban(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, projectID, environment, accountID)
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

func (a *pgAdminUsers) Delete(ctx context.Context, projectID, environment, accountID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.findUser(ctx, projectID, environment, accountID)
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

func (a *pgAdminUsers) VerifyEmail(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, projectID, environment, accountID)
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

func (a *pgAdminUsers) VerifyPhone(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		row, acc, err := a.findUser(ctx, projectID, environment, accountID)
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
		if _, _, err := a.findUser(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID); err != nil {
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
			if _, err := a.revokeSessions(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID, ""); err != nil {
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
		row, acc, err := a.findUser(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID)
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

func (a *pgAdminUsers) Export(ctx context.Context, projectID, environment, accountID string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		if _, _, err := a.findUser(ctx, projectID, environment, accountID); err != nil {
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
		env := adminEnv(cmd.Environment)
		if _, _, err := a.findUser(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID); err != nil {
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
		token, err := a.db.Signer().Sign(ctx, cmd.ProjectID, env, map[string]any{
			"iss": cmd.ProjectID,
			"sub": cmd.AccountID,
			"pid": cmd.ProjectID,
			"act": cmd.ActorID,
			"typ": "impersonation",
			"env": env,
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
			Environment: env,
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

func (a *pgAdminUsers) ResetMFA(ctx context.Context, projectID, environment, accountID string, factorIDs []string) (int, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (int, error) {
		if _, _, err := a.findUser(ctx, projectID, environment, accountID); err != nil {
			return 0, err
		}
		factors, err := models.IamFactors.Query(
			sm.Where(models.IamFactors.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamFactors.Columns.Environment.EQ(psql.Arg(adminEnv(environment)))),
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

func (a *pgAdminUsers) ListIdentities(ctx context.Context, projectID, environment, accountID string) ([]domain.Identity, error) {
	if _, _, err := a.findUser(ctx, projectID, environment, accountID); err != nil {
		return nil, err
	}
	rows, err := models.IamIdentities.Query(
		sm.Where(models.IamIdentities.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamIdentities.Columns.Environment.EQ(psql.Arg(adminEnv(environment)))),
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

func (a *pgAdminUsers) DeleteIdentity(ctx context.Context, projectID, environment, accountID, identityID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamIdentity(ctx, a.db.Bobx(), identityID)
		if err != nil {
			if adminIsNotFound(translatePgErr("identity", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.Environment != adminEnv(environment) || row.UserID != accountID {
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

func (a *pgAdminUsers) ListSessions(ctx context.Context, projectID, environment, accountID string) ([]domain.Session, error) {
	if _, _, err := a.findUser(ctx, projectID, environment, accountID); err != nil {
		return nil, err
	}
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.Environment.EQ(psql.Arg(adminEnv(environment)))),
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		// Use the shared mapper so envelope columns (last_active_at, trusted, aal,
		// client_id) win over the at-mint snapshot in data — admins see live state.
		s, err := accountSessionToDomain(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, nil
}

func (a *pgAdminUsers) DeleteSession(ctx context.Context, projectID, environment, accountID, sessionID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
		if err != nil {
			if adminIsNotFound(translatePgErr("session", err)) {
				return domain.ErrSessionNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.Environment != adminEnv(environment) || row.UserID != accountID {
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
		if _, _, err := a.findUser(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID); err != nil {
			return 0, err
		}
		return a.revokeSessions(ctx, cmd.ProjectID, cmd.Environment, cmd.AccountID, cmd.ExceptSessionID)
	})
}

// revokeSessions deletes all sessions for a user except optionally one. The
// caller is responsible for the tenant boundary check and the transaction.
func (a *pgAdminUsers) revokeSessions(ctx context.Context, projectID, environment, accountID, exceptID string) (int, error) {
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.Environment.EQ(psql.Arg(adminEnv(environment)))),
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

func (a *pgAdminApps) findApp(ctx context.Context, projectID, environment, appID string) (*models.IamAppClient, *domain.AppClient, error) {
	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), appID)
	if err != nil {
		if adminIsNotFound(translatePgErr("app_client", err)) {
			return nil, nil, domain.ErrClientNotFound
		}
		return nil, nil, err
	}
	if row.ProjectID != projectID || row.Environment != adminEnv(environment) {
		return nil, nil, domain.ErrClientNotFound
	}
	var app domain.AppClient
	if err := unmarshal(row.Data, &app); err != nil {
		return nil, nil, err
	}
	return row, &app, nil
}

func (a *pgAdminApps) List(ctx context.Context, projectID, environment string) ([]domain.AppClient, error) {
	rows, err := models.IamAppClients.Query(
		sm.Where(models.IamAppClients.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamAppClients.Columns.Environment.EQ(psql.Arg(adminEnv(environment)))),
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

func (a *pgAdminApps) Get(ctx context.Context, projectID, environment, appID string) (*domain.AppClient, error) {
	_, app, err := a.findApp(ctx, projectID, environment, appID)
	return app, err
}

// AllOrigins returns the de-duplicated union of every app client's allowed
// origins across all projects/environments — the dynamic CORS allow-list. CORS
// preflight (OPTIONS) carries no X-Client-Id, so the allow decision can only be
// made against this global union (tenant isolation is enforced separately by
// X-Client-Id + tokens; CORS only governs browser read access).
func (a *pgAdminApps) AllowedOrigins(ctx context.Context) ([]string, error) {
	rows, err := models.IamAppClients.Query().All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, row := range rows {
		if len(row.Data) == 0 {
			continue
		}
		var app struct {
			AllowedOrigins []string `json:"AllowedOrigins"`
		}
		if unmarshal(row.Data, &app) != nil {
			continue
		}
		for _, o := range domain.NormalizeOrigins(app.AllowedOrigins) {
			if _, ok := seen[o]; ok {
				continue
			}
			seen[o] = struct{}{}
			out = append(out, o)
		}
	}
	return out, nil
}

func (a *pgAdminApps) Create(ctx context.Context, cmd domain.AppClientCmd) (*domain.AppClient, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AppClient, error) {
		app := &domain.AppClient{
			ID:             newUUID(),
			ProjectID:      cmd.ProjectID,
			Name:           cmd.Name,
			Type:           cmd.Type,
			Environment:    adminEnv(cmd.Environment),
			RedirectURIs:   cmd.RedirectURIs,
			AllowedOrigins: domain.NormalizeOrigins(cmd.AllowedOrigins),
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

func (a *pgAdminApps) Update(ctx context.Context, projectID, environment, appID string, patch map[string]any) (*domain.AppClient, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AppClient, error) {
		row, app, err := a.findApp(ctx, projectID, environment, appID)
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
		if v, ok := patch["allowed_origins"].([]string); ok {
			app.AllowedOrigins = domain.NormalizeOrigins(v)
		} else if v, ok := patch["allowed_origins"].([]any); ok {
			origins := make([]string, 0, len(v))
			for _, o := range v {
				if s, ok := o.(string); ok {
					origins = append(origins, s)
				}
			}
			app.AllowedOrigins = domain.NormalizeOrigins(origins)
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

func (a *pgAdminApps) Delete(ctx context.Context, projectID, environment, appID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.findApp(ctx, projectID, environment, appID)
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

func (a *pgAdminApps) AddSecret(ctx context.Context, projectID, environment, appID, name string) (*domain.AdminSecret, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminSecret, error) {
		if _, _, err := a.findApp(ctx, projectID, environment, appID); err != nil {
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

func (a *pgAdminApps) DeleteSecret(ctx context.Context, projectID, environment, appID, secretID string) error {
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
		// Parse real JSON into raw per-key values. (json.Unmarshal into
		// map[string]jx.Raw would base64-decode each value, the inverse of the
		// old base64 write path — see putConfigDoc.)
		d := jx.DecodeBytes(row.Data)
		if err := d.Obj(func(d *jx.Decoder, key string) error {
			raw, err := d.Raw()
			if err != nil {
				return err
			}
			doc[key] = jx.Raw(raw)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// configDocToRawJSON flattens a domain.AdminConfigDoc (map[string]jx.Raw) into a
// single plain-JSON object, identical to what putConfigDoc persists. Used to feed
// the configspec validators their canonical bytes before write (fail-closed).
func configDocToRawJSON(doc domain.AdminConfigDoc) ([]byte, error) {
	rawDoc := make(map[string]json.RawMessage, len(doc))
	for k, v := range doc {
		rawDoc[k] = json.RawMessage(v)
	}
	return json.Marshal(rawDoc)
}

// putConfigDoc upserts one iam_config(project, env, key) envelope from a doc.
func (a *pgAdminConfig) putConfigDoc(ctx context.Context, projectID, env, key string, doc domain.AdminConfigDoc) (domain.AdminConfigDoc, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (domain.AdminConfigDoc, error) {
		// Store REAL JSON. domain.AdminConfigDoc is map[string]jx.Raw; json.Marshal
		// of that base64-encodes each value ([]byte semantics), which round-trips
		// via getConfigDoc but is opaque to any plain-JSON reader (flow engine,
		// public config). Convert through json.RawMessage so values stay raw JSON.
		rawDoc := make(map[string]json.RawMessage, len(doc))
		for k, v := range doc {
			rawDoc[k] = json.RawMessage(v)
		}
		raw, err := json.Marshal(rawDoc)
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
	raw, err := configDocToRawJSON(cmd.Doc)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseAuthConfig(raw)
	if err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "auth", cmd.Doc)
}

func (a *pgAdminConfig) GetPasswordPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "password_policy")
}

func (a *pgAdminConfig) UpdatePasswordPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	raw, err := configDocToRawJSON(cmd.Doc)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParsePasswordPolicy(raw)
	if err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "password_policy", cmd.Doc)
}

func (a *pgAdminConfig) GetSessionPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "session_policy")
}

func (a *pgAdminConfig) UpdateSessionPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	raw, err := configDocToRawJSON(cmd.Doc)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseSessionPolicy(raw)
	if err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "session_policy", cmd.Doc)
}
func (a *pgAdminConfig) GetRateLimits(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "rate_limits")
}
func (a *pgAdminConfig) UpdateRateLimits(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	raw, err := configDocToRawJSON(cmd.Doc)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseRateLimits(raw)
	if err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "rate_limits", cmd.Doc)
}
func (a *pgAdminConfig) GetMfaPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "mfa_policy")
}
func (a *pgAdminConfig) UpdateMfaPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	raw, err := configDocToRawJSON(cmd.Doc)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseMFAPolicy(raw)
	if err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return a.putConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "mfa_policy", cmd.Doc)
}

func (a *pgAdminConfig) GetConsent(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error) {
	return a.getConfigDoc(ctx, cmd.ProjectID, cmd.Environment, "consent")
}

func (a *pgAdminConfig) PutConsent(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error) {
	raw, err := configDocToRawJSON(cmd.Doc)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseConsentConfig(raw)
	if err != nil {
		return nil, err
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
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
	if err := domain.FeaturesSpec(cmd.Features).Validate(); err != nil {
		return nil, err
	}
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
	Type string `json:"type"`
	// Config is stored as json.RawMessage (not jx.Raw) so json.Marshal writes the
	// values verbatim. jx.Raw is a bare []byte alias and would be base64-encoded.
	Config map[string]json.RawMessage `json:"config"`
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
		p, err := adminProviderToDomain(a.db.Cipher, row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func adminProviderToDomain(cipher Cipher, row *models.IamProvider) (domain.AdminProvider, error) {
	p := domain.AdminProvider{ID: row.ID, Type: row.Provider, Enabled: row.Enabled}
	if len(row.Data) > 0 {
		var d adminProviderData
		if err := json.Unmarshal(row.Data, &d); err == nil {
			if d.Type != "" {
				p.Type = d.Type
			}
			cfg, err := decryptProviderConfig(cipher, jsonToRaw(d.Config))
			if err != nil {
				return domain.AdminProvider{}, err
			}
			p.Config = cfg
		}
	}
	return p, nil
}

func (a *pgAdminConfig) createProvider(ctx context.Context, kind string, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error) {
	if err := (domain.ProviderConfigSpec{Kind: kind, Type: cmd.Type}).Validate(); err != nil {
		return nil, err
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminProvider, error) {
		id := cmd.ID
		if id == "" {
			id = newUUID()
		}
		encCfg, err := encryptProviderConfig(a.db.Cipher, cmd.Config)
		if err != nil {
			return nil, err
		}
		d := adminProviderData{Type: cmd.Type, Config: rawToJSON(encCfg)}
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
	if err := (domain.ProviderConfigSpec{Kind: kind, Type: cmd.Type}).Validate(); err != nil {
		return nil, err
	}
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
		encCfg, err := encryptProviderConfig(a.db.Cipher, cmd.Config)
		if err != nil {
			return nil, err
		}
		d := adminProviderData{Type: cmd.Type, Config: rawToJSON(encCfg)}
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
	out := make(map[string]jx.Raw, len(rows)+len(domain.BuiltinEmailTemplates))
	// Seed with the built-in catalogue so every system template is always listed
	// (editable/previewable/testable) even before a project customises it. Bodies
	// are built as plain values and json-marshaled — NOT map[string]jx.Raw, which
	// encoding/json base64-encodes (jx.Raw is []byte).
	for _, t := range domain.BuiltinEmailTemplates {
		c := t.Copy(adminTemplateLocale)
		raw, err := json.Marshal(map[string]any{
			"id":         t.Key,
			"name":       t.Name,
			"locale":     "",
			"subject":    c.Subject,
			"text":       c.Text,
			"html":       c.HTML,
			"customized": false,
		})
		if err != nil {
			return nil, err
		}
		out[t.Key] = jx.Raw(raw)
	}
	// Overlay project overrides (mark them customized).
	for _, row := range rows {
		key := row.Key
		if row.Locale != "" && row.Locale != adminTemplateLocale {
			key = row.Key + ":" + row.Locale
		}
		body := map[string]any{}
		if len(row.Data) > 0 {
			if err := json.Unmarshal(row.Data, &body); err != nil {
				return nil, err
			}
		}
		if bt := domain.BuiltinEmailTemplateByKey(row.Key); bt != nil {
			if _, ok := body["name"]; !ok {
				body["name"] = bt.Name
			}
		}
		body["id"] = row.Key
		body["locale"] = row.Locale
		body["customized"] = true
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		out[key] = jx.Raw(raw)
	}
	return out, nil
}

func (a *pgAdminConfig) UpdateEmailTemplate(ctx context.Context, cmd domain.AdminTemplateUpdateCmd) (map[string]jx.Raw, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]jx.Raw, error) {
		existing, err := models.IamEmailTemplates.Query(
			sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(cmd.TemplateID))),
			sm.Where(models.IamEmailTemplates.Columns.Locale.EQ(psql.Arg(adminTemplateLocaleFromPatch(cmd.Patch)))),
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
		locale := adminTemplateLocaleFromPatch(body)
		body["id"] = adminRawString(cmd.TemplateID)
		body["locale"] = adminRawString(locale)
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
				Locale:    ptr(locale),
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
			Payload:     map[string]any{"project_id": cmd.ProjectID, "template_id": cmd.TemplateID, "locale": locale, "body": body},
		}); err != nil {
			return nil, err
		}
		return body, nil
	})
}

func (a *pgAdminConfig) PreviewEmailTemplate(ctx context.Context, cmd domain.AdminTemplatePreviewCmd) (*domain.AdminTemplatePreview, error) {
	row, err := a.findEmailTemplate(ctx, cmd.ProjectID, cmd.TemplateID, adminTemplateLocaleOrDefault(cmd.Locale))
	if err != nil {
		return nil, err
	}
	body := map[string]string{}
	if len(row.Data) > 0 {
		_ = json.Unmarshal(row.Data, &body) // best-effort: only string fields render
	}
	data := adminTemplateData(cmd.Data)
	subject, err := renderAdminTemplate(body["subject"], data)
	if err != nil {
		return nil, domain.ErrValidation.WithMessage("subject template is invalid")
	}
	html, err := renderAdminTemplate(body["html"], data)
	if err != nil {
		return nil, domain.ErrValidation.WithMessage("html template is invalid")
	}
	text, err := renderAdminTemplate(body["text"], data)
	if err != nil {
		return nil, domain.ErrValidation.WithMessage("text template is invalid")
	}
	return &domain.AdminTemplatePreview{
		Subject: subject,
		HTML:    html,
		Text:    text,
	}, nil
}

func (a *pgAdminConfig) SendTestEmail(ctx context.Context, cmd domain.AdminTemplateSendTestCmd) error {
	// Tenant boundary: the template must exist for the project.
	if _, err := a.findEmailTemplate(ctx, cmd.ProjectID, cmd.TemplateID, adminTemplateLocaleOrDefault(cmd.Locale)); err != nil {
		return err
	}
	ok, err := a.hasEnabledProvider(ctx, cmd.ProjectID, "email")
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrValidation.WithMessage("enabled email provider is required")
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "config.test_email_requested",
		ProjectID:   cmd.ProjectID,
		Environment: adminEnv(cmd.Environment),
		AggregateID: cmd.TemplateID,
		Payload: map[string]any{
			"project_id":    cmd.ProjectID,
			"template_id":   cmd.TemplateID,
			"to":            cmd.To,
			"locale":        adminTemplateLocaleOrDefault(cmd.Locale),
			"template_data": adminTemplateData(cmd.Data),
		},
	}); err != nil {
		return err
	}
	return nil
}

func (a *pgAdminConfig) hasEnabledProvider(ctx context.Context, projectID, kind string) (bool, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg(kind))),
		sm.Where(models.IamProviders.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (a *pgAdminConfig) findEmailTemplate(ctx context.Context, projectID, key, locale string) (*models.IamEmailTemplate, error) {
	row, err := models.IamEmailTemplates.Query(
		sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(key))),
		sm.Where(models.IamEmailTemplates.Columns.Locale.EQ(psql.Arg(locale))),
	).One(ctx, a.db.Bobx())
	if err == nil {
		return row, nil
	}
	if !adminIsNotFound(err) {
		return nil, err
	}
	// No project override for the requested locale; try the default locale.
	if locale != adminTemplateLocale {
		row, err = models.IamEmailTemplates.Query(
			sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(key))),
			sm.Where(models.IamEmailTemplates.Columns.Locale.EQ(psql.Arg(adminTemplateLocale))),
		).One(ctx, a.db.Bobx())
		if err == nil {
			return row, nil
		}
		if !adminIsNotFound(err) {
			return nil, err
		}
	}
	// No override at all: fall back to the built-in catalogue so previews and
	// test-sends work for system templates the project has never customised.
	if syn := builtinTemplateRow(projectID, key, locale); syn != nil {
		return syn, nil
	}
	return nil, domain.ErrNotFound
}

// builtinTemplateRow synthesizes an in-memory (unpersisted) template row from the
// domain catalogue, so callers that read row.Data render the system default.
func builtinTemplateRow(projectID, key, locale string) *models.IamEmailTemplate {
	t := domain.BuiltinEmailTemplateByKey(key)
	if t == nil {
		return nil
	}
	if locale == "" {
		locale = adminTemplateLocale
	}
	c := t.Copy(locale)
	body, _ := json.Marshal(map[string]string{"subject": c.Subject, "text": c.Text, "html": c.HTML})
	return &models.IamEmailTemplate{
		ProjectID: projectID,
		Key:       key,
		Locale:    locale,
		Data:      body,
	}
}

func adminTemplateLocaleOrDefault(locale string) string {
	if locale == "" {
		return adminTemplateLocale
	}
	return locale
}

func adminTemplateLocaleFromPatch(patch map[string]jx.Raw) string {
	if raw, ok := patch["locale"]; ok {
		var locale string
		if err := json.Unmarshal(raw, &locale); err == nil && locale != "" {
			return locale
		}
	}
	return adminTemplateLocale
}

func adminRawString(s string) jx.Raw {
	raw, _ := json.Marshal(s)
	return jx.Raw(raw)
}

func adminTemplateData(in map[string]jx.Raw) map[string]any {
	out := map[string]any{
		"code":             "123456",
		"token":            "sample-token",
		"link":             "https://example.test/auth/callback?token=sample-token",
		"email":            "user@example.com",
		"to":               "user@example.com",
		"project_name":     "IAM",
		"challenge_id":     "chl_sample",
		"template_id":      "sample",
		"reset_url":        "https://example.test/reset?token=sample-token",
		"magic_link":       "https://example.test/auth/magic?token=sample-token",
		"verification_url": "https://example.test/auth/verify?token=sample-token",
	}
	for k, raw := range in {
		var v any
		if err := json.Unmarshal(raw, &v); err == nil {
			out[k] = v
		}
	}
	return out
}

func renderAdminTemplate(src string, data map[string]any) (string, error) {
	if src == "" {
		return "", nil
	}
	tpl, err := template.New("email").Option("missingkey=zero").Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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
		sm.Where(models.IamAccessRequests.Columns.Environment.EQ(psql.Arg(adminEnv(cmd.Environment)))),
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
