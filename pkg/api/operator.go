// Code scaffolded for IAM handler groups.
//
// OperatorService is pure orchestration: it holds aggregate-port interfaces (deps) and
// nothing else. It embeds oas.UnimplementedHandler so any operation it does not
// override returns not-implemented, and panics on every v1.0.0 operation until
// written. Each port method is atomic in its adapter — services never open a
// transaction.

package api

import (
	"context"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type OperatorProjects interface {
	CreateProject(ctx context.Context, cmd domain.ProjectCmd) (*domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
	GetProject(ctx context.Context, projectID string) (*domain.Project, error)
	UpdateProject(ctx context.Context, cmd domain.OperatorProjectPatchCmd) (*domain.Project, error)
	DeleteProject(ctx context.Context, projectID string, hard bool) error
	CreateEnvironment(ctx context.Context, cmd domain.EnvironmentCmd) (*domain.Environment, error)
	ListEnvironments(ctx context.Context, projectID string) ([]domain.Environment, error)
	GetEnvironment(ctx context.Context, projectID, env string) (*domain.Environment, error)
	DeleteEnvironment(ctx context.Context, projectID, env string) error
	MintAdminToken(ctx context.Context, cmd domain.OperatorAdminTokenCmd) (string, time.Time, error)
	ListAdminTokens(ctx context.Context, projectID string) ([]domain.OperatorAdminToken, error)
	RevokeAdminToken(ctx context.Context, projectID, tokenID string) error
	PlanConfig(ctx context.Context, cmd domain.OperatorConfigCmd) (map[string]any, error)
	ApplyConfig(ctx context.Context, cmd domain.OperatorConfigCmd) (map[string]any, error)
	ExportConfig(ctx context.Context, projectID string) (map[string]any, error)
	GetFeatures(ctx context.Context, projectID string) (map[string]bool, error)
	UpdateFeatures(ctx context.Context, cmd domain.OperatorFeaturesCmd) (map[string]bool, error)
}

type OperatorDeps struct{ Projects OperatorProjects }

// OperatorService implements the OperatorHandler slice of oas.Handler.
type OperatorService struct {
	oas.UnimplementedHandler
	deps OperatorDeps
}

// NewOperatorService builds the Operator service from its dependencies.
func NewOperatorService(deps OperatorDeps) *OperatorService { return &OperatorService{deps: deps} }

var _ oas.Handler = (*OperatorService)(nil)

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectId(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdParams) (r *oas.Ok, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	if err := s.deps.Projects.DeleteProject(ctx, params.ProjectID, params.Hard.Or(false)); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenId(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenIdParams) (r *oas.Ok, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	if err := s.deps.Projects.RevokeAdminToken(ctx, params.ProjectID, params.TokenID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnv(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnvParams) (r *oas.Ok, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	if err := s.deps.Projects.DeleteEnvironment(ctx, params.ProjectID, params.Env); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OperatorService) GetMgmtV1Projects(ctx context.Context, params oas.GetMgmtV1ProjectsParams) (r *oas.GetMgmtV1ProjectsOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	projects, err := s.deps.Projects.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]oas.Project, 0, len(projects))
	for i := range projects {
		data = append(data, oasProject(&projects[i]))
	}
	return &oas.GetMgmtV1ProjectsOK{Data: data}, nil
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectId(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdParams) (r *oas.GetMgmtV1ProjectsByProjectIdOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	proj, err := s.deps.Projects.GetProject(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	return &oas.GetMgmtV1ProjectsByProjectIdOK{
		Project: oas.NewOptProject(oasProject(proj)),
	}, nil
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdAdminTokens(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdAdminTokensParams) (r oas.GetMgmtV1ProjectsByProjectIdAdminTokensOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	tokens, err := s.deps.Projects.ListAdminTokens(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	data := make([]any, 0, len(tokens))
	for i := range tokens {
		data = append(data, oasOperatorAdminToken(&tokens[i]))
	}
	return oasRawMap[oas.GetMgmtV1ProjectsByProjectIdAdminTokensOK](map[string]any{"data": data}), nil
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdConfigExport(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdConfigExportParams) (r oas.GetMgmtV1ProjectsByProjectIdConfigExportRes, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cfg, err := s.deps.Projects.ExportConfig(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	out := oasRawMap[oas.GetMgmtV1ProjectsByProjectIdConfigExportOKApplicationJSON](cfg)
	return &out, nil
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdEnvironments(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdEnvironmentsParams) (r *oas.GetMgmtV1ProjectsByProjectIdEnvironmentsOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	envs, err := s.deps.Projects.ListEnvironments(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.Environment, 0, len(envs))
	for i := range envs {
		data = append(data, oasEnvironment(&envs[i]))
	}
	return &oas.GetMgmtV1ProjectsByProjectIdEnvironmentsOK{Data: data}, nil
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdEnvironmentsByEnv(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvParams) (r *oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	env, err := s.deps.Projects.GetEnvironment(ctx, params.ProjectID, params.Env)
	if err != nil {
		return nil, err
	}
	return &oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvOK{
		Environment: oas.NewOptEnvironment(oasEnvironment(env)),
	}, nil
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdFeatures(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdFeaturesParams) (r oas.GetMgmtV1ProjectsByProjectIdFeaturesOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	features, err := s.deps.Projects.GetFeatures(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	return oas.GetMgmtV1ProjectsByProjectIdFeaturesOK(features), nil
}

func (s *OperatorService) PatchMgmtV1ProjectsByProjectId(ctx context.Context, req oas.PatchMgmtV1ProjectsByProjectIdReq, params oas.PatchMgmtV1ProjectsByProjectIdParams) (r *oas.PatchMgmtV1ProjectsByProjectIdOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	patch := anyMap(req)
	cmd := domain.OperatorProjectPatchCmd{
		ProjectID:     params.ProjectID,
		Name:          rawMapString(patch, "name"),
		Slug:          rawMapString(patch, "slug"),
		DefaultLocale: rawMapString(patch, "default_locale"),
	}
	proj, err := s.deps.Projects.UpdateProject(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PatchMgmtV1ProjectsByProjectIdOK{
		Project: oas.NewOptProject(oasProject(proj)),
	}, nil
}

func (s *OperatorService) PatchMgmtV1ProjectsByProjectIdFeatures(ctx context.Context, req oas.PatchMgmtV1ProjectsByProjectIdFeaturesReq, params oas.PatchMgmtV1ProjectsByProjectIdFeaturesParams) (r oas.PatchMgmtV1ProjectsByProjectIdFeaturesOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cmd := domain.OperatorFeaturesCmd{
		ProjectID: params.ProjectID,
		Features:  map[string]bool(req),
	}
	features, err := s.deps.Projects.UpdateFeatures(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return oas.PatchMgmtV1ProjectsByProjectIdFeaturesOK(features), nil
}

func (s *OperatorService) PostMgmtV1Projects(ctx context.Context, req *oas.PostMgmtV1ProjectsReq, params oas.PostMgmtV1ProjectsParams) (r *oas.PostMgmtV1ProjectsCreated, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cmd := domain.ProjectCmd{
		Name:          req.Name,
		Slug:          req.Slug.Or(""),
		DefaultLocale: req.DefaultLocale.Or(""),
	}
	proj, err := s.deps.Projects.CreateProject(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PostMgmtV1ProjectsCreated{
		Project: oas.NewOptProject(oasProject(proj)),
	}, nil
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdAdminTokens(ctx context.Context, req *oas.PostMgmtV1ProjectsByProjectIdAdminTokensReq, params oas.PostMgmtV1ProjectsByProjectIdAdminTokensParams) (r *oas.PostMgmtV1ProjectsByProjectIdAdminTokensOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cmd := domain.OperatorAdminTokenCmd{
		ProjectID: params.ProjectID,
		Name:      req.Name,
		Scopes:    req.Scopes,
	}
	if expiresAt, ok := req.ExpiresAt.Get(); ok {
		cmd.ExpiresAt = expiresAt
	}
	token, expiresAt, err := s.deps.Projects.MintAdminToken(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PostMgmtV1ProjectsByProjectIdAdminTokensOK{
		AdminToken: oas.NewOptString(token),
		ExpiresAt:  oas.NewOptTimestamp(oas.Timestamp(expiresAt)),
	}, nil
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdConfigApply(ctx context.Context, req oas.PostMgmtV1ProjectsByProjectIdConfigApplyReq, params oas.PostMgmtV1ProjectsByProjectIdConfigApplyParams) (r oas.PostMgmtV1ProjectsByProjectIdConfigApplyOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cmd := domain.OperatorConfigCmd{
		ProjectID: params.ProjectID,
		Config:    anyMap(req),
	}
	result, err := s.deps.Projects.ApplyConfig(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return oasRawMap[oas.PostMgmtV1ProjectsByProjectIdConfigApplyOK](result), nil
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdConfigPlan(ctx context.Context, req oas.PostMgmtV1ProjectsByProjectIdConfigPlanReq, params oas.PostMgmtV1ProjectsByProjectIdConfigPlanParams) (r oas.PostMgmtV1ProjectsByProjectIdConfigPlanOK, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cmd := domain.OperatorConfigCmd{
		ProjectID: params.ProjectID,
		Config:    anyMap(req),
	}
	result, err := s.deps.Projects.PlanConfig(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return oasRawMap[oas.PostMgmtV1ProjectsByProjectIdConfigPlanOK](result), nil
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdEnvironments(ctx context.Context, req *oas.PostMgmtV1ProjectsByProjectIdEnvironmentsReq, params oas.PostMgmtV1ProjectsByProjectIdEnvironmentsParams) (r *oas.PostMgmtV1ProjectsByProjectIdEnvironmentsCreated, _ error) {
	if _, err := requireOperator(ctx); err != nil {
		return nil, err
	}
	cmd := domain.EnvironmentCmd{
		ProjectID: params.ProjectID,
		Name:      req.Name,
	}
	env, err := s.deps.Projects.CreateEnvironment(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PostMgmtV1ProjectsByProjectIdEnvironmentsCreated{
		Environment: oas.NewOptEnvironment(oasEnvironment(env)),
	}, nil
}

// oasProject maps a domain Project to its oas wire representation.
func oasProject(p *domain.Project) oas.Project {
	out := oas.Project{
		ID:               oas.NewOptString(p.ID),
		Name:             oas.NewOptString(p.Name),
		Slug:             oas.NewOptString(p.Slug),
		SupportedLocales: p.SupportedLocales,
		Environments:     p.Environments,
		CreatedAt:        oas.NewOptTimestamp(oas.Timestamp(p.CreatedAt)),
	}
	// Only set the default locale when present: an empty string fails the oas
	// locale pattern on response validation, which would reject the response.
	if p.DefaultLocale != "" {
		out.DefaultLocale = oas.NewOptString(p.DefaultLocale)
	}
	return out
}

// oasEnvironment maps a domain Environment to its oas wire representation.
func oasEnvironment(e *domain.Environment) oas.Environment {
	return oas.Environment{
		Name:      oas.NewOptString(e.Name),
		ProjectID: oas.NewOptString(e.ProjectID),
		Issuer:    oas.NewOptString(e.Issuer),
		CreatedAt: oas.NewOptTimestamp(oas.Timestamp(e.CreatedAt)),
	}
}

// oasOperatorAdminToken maps a domain admin-token read model into a free-form
// JSON object for the schemaless /mgmt admin-tokens listing.
func oasOperatorAdminToken(t *domain.OperatorAdminToken) map[string]any {
	m := map[string]any{
		"id":         t.ID,
		"project_id": t.ProjectID,
		"revoked":    t.Revoked,
	}
	if t.Name != "" {
		m["name"] = t.Name
	}
	if len(t.Scopes) > 0 {
		m["scopes"] = t.Scopes
	}
	if !t.CreatedAt.IsZero() {
		m["created_at"] = t.CreatedAt
	}
	if !t.ExpiresAt.IsZero() {
		m["expires_at"] = t.ExpiresAt
	}
	return m
}

// rawMapString extracts a string value for key from a decoded free-form JSON
// patch map; it returns "" when the key is absent or not a string ("no change").
func rawMapString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
