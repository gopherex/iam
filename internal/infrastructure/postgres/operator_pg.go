package postgres

// Operator adapter — backs api.OperatorProjects over four envelopes:
//   - iam_projects      (project aggregate)
//   - iam_environments  (per-project environment; PK = project_id+name)
//   - iam_admin_tokens  (opaque admin token: store only the sha256 hash)
//   - iam_config        (per project/env/key JSON config; PK = project_id+environment+key)
//
// Tenant boundary: every read filters by project_id; a row whose project_id
// does not match the requested one is treated as not-found. Config and feature
// state live under reserved iam_config keys ("features" / "config") scoped to
// the project's default environment.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// operatorDefaultEnv is the environment a freshly-created project starts with
// and the environment that config / features default to.
const operatorDefaultEnv = "live"

// reserved iam_config keys used by the operator surface.
const (
	operatorConfigKeyFeatures = "features"
	operatorConfigKeyConfig   = "config"
)

// PgOperator persists the operator/project aggregates.
type PgOperator struct {
	db      *DB
	emitter Emitter
}

// NewPgOperator builds the operator adapter.
func NewPgOperator(db *DB, emitter Emitter) *PgOperator {
	return &PgOperator{db: db, emitter: emitter}
}

var _ api.OperatorProjects = (*PgOperator)(nil)

// ===== Projects =====

func (a *PgOperator) CreateProject(ctx context.Context, cmd domain.ProjectCmd) (*domain.Project, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Project, error) {
		proj := &domain.Project{
			ID:               newUUID(),
			Name:             cmd.Name,
			Slug:             cmd.Slug,
			DefaultLocale:    cmd.DefaultLocale,
			SupportedLocales: []string{},
			Environments:     []string{operatorDefaultEnv},
			CreatedAt:        nowUTC(),
		}
		if proj.Slug == "" {
			proj.Slug = proj.ID
		}
		if err := a.insertProject(ctx, proj); err != nil {
			return nil, err
		}
		// Seed the project's default environment alongside the project.
		env := &domain.Environment{
			ProjectID: proj.ID,
			Name:      operatorDefaultEnv,
			CreatedAt: proj.CreatedAt,
		}
		if err := a.insertEnvironment(ctx, env); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "project.created",
			ProjectID:   proj.ID,
			Environment: "",
			AggregateID: proj.ID,
			Payload:     proj,
		}); err != nil {
			return nil, err
		}
		return proj, nil
	})
}

func (a *PgOperator) insertProject(ctx context.Context, proj *domain.Project) error {
	raw, err := marshal(proj)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamProjectSetter{
		ID:        &proj.ID,
		Slug:      &proj.Slug,
		Name:      &proj.Name,
		CreatedAt: ptr(proj.CreatedAt),
		UpdatedAt: ptr(proj.CreatedAt),
		Data:      &rm,
	}
	if _, err := models.IamProjects.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (a *PgOperator) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := models.IamProjects.Query().All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Project, 0, len(rows))
	for _, row := range rows {
		var p domain.Project
		if err := unmarshal(row.Data, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// GetProject reads + decodes a project row, mapping no-rows to project-not-found.
func (a *PgOperator) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	row, err := models.FindIamProject(ctx, a.db.Bobx(), projectID)
	if err != nil {
		if isStorageNotFound(translatePgErr("project", err)) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	var p domain.Project
	if err := unmarshal(row.Data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (a *PgOperator) UpdateProject(ctx context.Context, cmd domain.OperatorProjectPatchCmd) (*domain.Project, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Project, error) {
		row, err := models.FindIamProject(ctx, a.db.Bobx(), cmd.ProjectID)
		if err != nil {
			if isStorageNotFound(translatePgErr("project", err)) {
				return nil, domain.ErrProjectNotFound
			}
			return nil, err
		}
		var p domain.Project
		if err := unmarshal(row.Data, &p); err != nil {
			return nil, err
		}
		if cmd.Name != "" {
			p.Name = cmd.Name
		}
		if cmd.Slug != "" {
			p.Slug = cmd.Slug
		}
		if cmd.DefaultLocale != "" {
			p.DefaultLocale = cmd.DefaultLocale
		}
		raw, err := marshal(&p)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamProjectSetter{
			Name:      &p.Name,
			Slug:      &p.Slug,
			Data:      &rm,
			UpdatedAt: ptr(nowUTC()),
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "project.updated",
			ProjectID:   p.ID,
			Environment: "",
			AggregateID: p.ID,
			Payload:     &p,
		}); err != nil {
			return nil, err
		}
		return &p, nil
	})
}

func (a *PgOperator) DeleteProject(ctx context.Context, projectID string, hard bool) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamProject(ctx, a.db.Bobx(), projectID)
		if err != nil {
			if isStorageNotFound(translatePgErr("project", err)) {
				return domain.ErrProjectNotFound
			}
			return err
		}
		// Both soft and hard delete remove the row here; outbox carries the
		// distinction downstream. (No soft-delete column on the envelope.)
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "project.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: projectID,
			Payload:     map[string]any{"id": projectID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ===== Environments =====

func (a *PgOperator) CreateEnvironment(ctx context.Context, cmd domain.EnvironmentCmd) (*domain.Environment, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Environment, error) {
		// Tenant boundary: the project must exist.
		prow, err := models.FindIamProject(ctx, a.db.Bobx(), cmd.ProjectID)
		if err != nil {
			if isStorageNotFound(translatePgErr("project", err)) {
				return nil, domain.ErrProjectNotFound
			}
			return nil, err
		}
		env := &domain.Environment{
			ProjectID: cmd.ProjectID,
			Name:      cmd.Name,
			CreatedAt: nowUTC(),
		}
		if err := a.insertEnvironment(ctx, env); err != nil {
			return nil, err
		}
		// Keep the project's environment list in sync.
		var p domain.Project
		if err := unmarshal(prow.Data, &p); err != nil {
			return nil, err
		}
		if !containsString(p.Environments, env.Name) {
			p.Environments = append(p.Environments, env.Name)
			raw, err := marshal(&p)
			if err != nil {
				return nil, err
			}
			rm := json.RawMessage(raw)
			if err := prow.Update(ctx, a.db.Bobx(), &models.IamProjectSetter{
				Data:      &rm,
				UpdatedAt: ptr(nowUTC()),
			}); err != nil {
				return nil, err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "environment.created",
			ProjectID:   env.ProjectID,
			Environment: env.Name,
			AggregateID: env.ProjectID,
			Payload:     env,
		}); err != nil {
			return nil, err
		}
		return env, nil
	})
}

func (a *PgOperator) insertEnvironment(ctx context.Context, env *domain.Environment) error {
	raw, err := marshal(env)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamEnvironmentSetter{
		ProjectID: &env.ProjectID,
		Name:      &env.Name,
		CreatedAt: ptr(env.CreatedAt),
		Data:      &rm,
	}
	if env.Issuer != "" {
		v := null.From(env.Issuer)
		setter.Issuer = &v
	}
	if _, err := models.IamEnvironments.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (a *PgOperator) ListEnvironments(ctx context.Context, projectID string) ([]domain.Environment, error) {
	rows, err := models.IamEnvironments.Query(
		sm.Where(models.IamEnvironments.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Environment, 0, len(rows))
	for _, row := range rows {
		var e domain.Environment
		if err := unmarshal(row.Data, &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (a *PgOperator) GetEnvironment(ctx context.Context, projectID, env string) (*domain.Environment, error) {
	row, err := models.FindIamEnvironment(ctx, a.db.Bobx(), projectID, env)
	if err != nil {
		if isStorageNotFound(translatePgErr("environment", err)) {
			return nil, domain.ErrEnvironmentNotFound
		}
		return nil, err
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrEnvironmentNotFound
	}
	var e domain.Environment
	if err := unmarshal(row.Data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (a *PgOperator) DeleteEnvironment(ctx context.Context, projectID, env string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamEnvironment(ctx, a.db.Bobx(), projectID, env)
		if err != nil {
			if isStorageNotFound(translatePgErr("environment", err)) {
				return domain.ErrEnvironmentNotFound
			}
			return err
		}
		if row.ProjectID != projectID {
			return domain.ErrEnvironmentNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// Drop the name from the project's environment list (best effort).
		if prow, perr := models.FindIamProject(ctx, a.db.Bobx(), projectID); perr == nil {
			var p domain.Project
			if uerr := unmarshal(prow.Data, &p); uerr == nil {
				p.Environments = removeString(p.Environments, env)
				if raw, merr := marshal(&p); merr == nil {
					rm := json.RawMessage(raw)
					_ = prow.Update(ctx, a.db.Bobx(), &models.IamProjectSetter{
						Data:      &rm,
						UpdatedAt: ptr(nowUTC()),
					})
				}
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "environment.deleted",
			ProjectID:   projectID,
			Environment: env,
			AggregateID: projectID,
			Payload:     map[string]any{"id": env, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ===== Admin tokens =====

func (a *PgOperator) MintAdminToken(ctx context.Context, projectID string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		if _, err := models.FindIamProject(ctx, a.db.Bobx(), projectID); err != nil {
			if isStorageNotFound(translatePgErr("project", err)) {
				return "", domain.ErrProjectNotFound
			}
			return "", err
		}
		// Admin token is a signed RS256 JWT (jwx Signer) carrying its jti; only
		// the jti's sha256 hash is persisted, keeping the token revocable.
		tok := domain.OperatorAdminToken{
			ID:        newUUID(),
			ProjectID: projectID,
			CreatedAt: nowUTC(),
			Revoked:   false,
		}
		signed, err := a.db.Signer().Sign(ctx, projectID, "live", map[string]any{
			"iss": projectID,
			"sub": projectID,
			"pid": projectID,
			"jti": tok.ID,
			"typ": "admin",
			"env": "live",
		}, 90*24*time.Hour)
		if err != nil {
			return "", err
		}
		hash := sha256Hex(signed)
		raw, err := marshal(&tok)
		if err != nil {
			return "", err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamAdminTokenSetter{
			ID:        &tok.ID,
			ProjectID: &tok.ProjectID,
			Hash:      &hash,
			CreatedAt: ptr(tok.CreatedAt),
			Data:      &rm,
		}
		if _, err := models.IamAdminTokens.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return "", domain.ErrConflict
			}
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "admin_token.minted",
			ProjectID:   tok.ProjectID,
			Environment: "",
			AggregateID: tok.ID,
			Payload:     &tok,
		}); err != nil {
			return "", err
		}
		return signed, nil
	})
}

func (a *PgOperator) ListAdminTokens(ctx context.Context, projectID string) ([]domain.OperatorAdminToken, error) {
	rows, err := models.IamAdminTokens.Query(
		sm.Where(models.IamAdminTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.OperatorAdminToken, 0, len(rows))
	for _, row := range rows {
		var t domain.OperatorAdminToken
		if err := unmarshal(row.Data, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (a *PgOperator) RevokeAdminToken(ctx context.Context, projectID, tokenID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamAdminToken(ctx, a.db.Bobx(), tokenID)
		if err != nil {
			if isStorageNotFound(translatePgErr("admin_token", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID { // tenant boundary
			return domain.ErrNotFound
		}
		var t domain.OperatorAdminToken
		if err := unmarshal(row.Data, &t); err != nil {
			return err
		}
		t.Revoked = true
		raw, err := marshal(&t)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAdminTokenSetter{Data: &rm}); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "admin_token.revoked",
			ProjectID:   t.ProjectID,
			Environment: "",
			AggregateID: t.ID,
			Payload:     &t,
		}); err != nil {
			return err
		}
		return nil
	})
}

// ===== Config (plan / apply / export) =====

// PlanConfig is a read-only diff: it returns the proposed config without
// persisting it. The current document is loaded for context; the plan echoes
// the incoming config plus the prior state under "current".
func (a *PgOperator) PlanConfig(ctx context.Context, cmd domain.OperatorConfigCmd) (map[string]any, error) {
	if _, err := a.GetProject(ctx, cmd.ProjectID); err != nil {
		return nil, err
	}
	current, err := a.readConfig(ctx, cmd.ProjectID, operatorConfigKeyConfig)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"project_id": cmd.ProjectID,
		"current":    current,
		"proposed":   cmd.Config,
	}, nil
}

func (a *PgOperator) ApplyConfig(ctx context.Context, cmd domain.OperatorConfigCmd) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		if _, err := models.FindIamProject(ctx, a.db.Bobx(), cmd.ProjectID); err != nil {
			if isStorageNotFound(translatePgErr("project", err)) {
				return nil, domain.ErrProjectNotFound
			}
			return nil, err
		}
		if err := a.writeConfig(ctx, cmd.ProjectID, operatorConfigKeyConfig, cmd.Config); err != nil {
			return nil, err
		}
		result := map[string]any{
			"project_id": cmd.ProjectID,
			"applied":    cmd.Config,
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "project.config.applied",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.ProjectID,
			Payload:     result,
		}); err != nil {
			return nil, err
		}
		return result, nil
	})
}

func (a *PgOperator) ExportConfig(ctx context.Context, projectID string) (map[string]any, error) {
	if _, err := a.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	cfg, err := a.readConfig(ctx, projectID, operatorConfigKeyConfig)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

// ===== Features =====

func (a *PgOperator) GetFeatures(ctx context.Context, projectID string) (map[string]bool, error) {
	if _, err := a.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	raw, err := a.readConfig(ctx, projectID, operatorConfigKeyFeatures)
	if err != nil {
		return nil, err
	}
	return rawToBoolMap(raw), nil
}

func (a *PgOperator) UpdateFeatures(ctx context.Context, cmd domain.OperatorFeaturesCmd) (map[string]bool, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]bool, error) {
		if _, err := models.FindIamProject(ctx, a.db.Bobx(), cmd.ProjectID); err != nil {
			if isStorageNotFound(translatePgErr("project", err)) {
				return nil, domain.ErrProjectNotFound
			}
			return nil, err
		}
		// Merge the patch over the existing feature map.
		existing := rawToBoolMap(mustReadConfig(ctx, a, cmd.ProjectID, operatorConfigKeyFeatures))
		for k, v := range cmd.Features {
			existing[k] = v
		}
		doc := make(map[string]any, len(existing))
		for k, v := range existing {
			doc[k] = v
		}
		if err := a.writeConfig(ctx, cmd.ProjectID, operatorConfigKeyFeatures, doc); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "project.features.updated",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: cmd.ProjectID,
			Payload:     existing,
		}); err != nil {
			return nil, err
		}
		return existing, nil
	})
}

// ===== config-table helpers (operator-prefixed) =====

// readConfig loads the JSON document stored under (projectID, defaultEnv, key);
// a missing row yields a nil map (no config yet), never an error.
func (a *PgOperator) readConfig(ctx context.Context, projectID, key string) (map[string]any, error) {
	row, err := models.FindIamConfig(ctx, a.db.Bobx(), projectID, operatorDefaultEnv, key)
	if err != nil {
		if isStorageNotFound(translatePgErr("config", err)) {
			return nil, nil
		}
		return nil, err
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, nil
	}
	var doc map[string]any
	if err := unmarshal(row.Data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// mustReadConfig is readConfig with the error swallowed to nil; used inside a
// merge where a decode failure should fall back to an empty document.
func mustReadConfig(ctx context.Context, a *PgOperator, projectID, key string) map[string]any {
	doc, err := a.readConfig(ctx, projectID, key)
	if err != nil {
		return nil
	}
	return doc
}

// writeConfig upserts the JSON document under (projectID, defaultEnv, key).
func (a *PgOperator) writeConfig(ctx context.Context, projectID, key string, doc map[string]any) error {
	if doc == nil {
		doc = map[string]any{}
	}
	raw, err := marshal(doc)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	now := nowUTC()
	row, err := models.FindIamConfig(ctx, a.db.Bobx(), projectID, operatorDefaultEnv, key)
	if err != nil {
		if isStorageNotFound(translatePgErr("config", err)) {
			env := operatorDefaultEnv
			setter := &models.IamConfigSetter{
				ProjectID:   &projectID,
				Environment: &env,
				Key:         &key,
				UpdatedAt:   &now,
				Data:        &rm,
			}
			if _, ierr := models.IamConfigs.Insert(setter).One(ctx, a.db.Bobx()); ierr != nil {
				return ierr
			}
			return nil
		}
		return err
	}
	return row.Update(ctx, a.db.Bobx(), &models.IamConfigSetter{
		Data:      &rm,
		UpdatedAt: &now,
	})
}

// ===== small local helpers =====

// isStorageNotFound reports whether err is the storage-level not-found sentinel.
func isStorageNotFound(err error) bool {
	return err == ErrNotFound
}

// randomToken returns a hex-encoded cryptographically-random opaque token.
func randomToken(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sha256Hex returns the hex sha256 digest of s (what we persist for tokens).
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func rawToBoolMap(raw map[string]any) map[string]bool {
	out := make(map[string]bool, len(raw))
	for k, v := range raw {
		if b, ok := v.(bool); ok {
			out[k] = b
		}
	}
	return out
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(ss []string, s string) []string {
	out := make([]string, 0, len(ss))
	for _, v := range ss {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}
