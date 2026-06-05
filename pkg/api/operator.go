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

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type OperatorProjects interface {
	CreateProject(ctx context.Context, cmd domain.ProjectCmd) (*domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
	GetProject(ctx context.Context, projectID string) (*domain.Project, error)
	CreateEnvironment(ctx context.Context, cmd domain.EnvironmentCmd) (*domain.Environment, error)
	MintAdminToken(ctx context.Context, projectID string) (string, error)
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
	panic("implement me")
}

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenId(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnv(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnvParams) (r *oas.Ok, _ error) {
	panic("implement me")
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
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdConfigExport(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdConfigExportParams) (r oas.GetMgmtV1ProjectsByProjectIdConfigExportRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdEnvironments(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdEnvironmentsParams) (r *oas.GetMgmtV1ProjectsByProjectIdEnvironmentsOK, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdEnvironmentsByEnv(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvParams) (r *oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvOK, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdFeatures(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdFeaturesParams) (r oas.GetMgmtV1ProjectsByProjectIdFeaturesOK, _ error) {
	panic("implement me")
}

func (s *OperatorService) PatchMgmtV1ProjectsByProjectId(ctx context.Context, req oas.PatchMgmtV1ProjectsByProjectIdReq, params oas.PatchMgmtV1ProjectsByProjectIdParams) (r *oas.PatchMgmtV1ProjectsByProjectIdOK, _ error) {
	panic("implement me")
}

func (s *OperatorService) PatchMgmtV1ProjectsByProjectIdFeatures(ctx context.Context, req oas.PatchMgmtV1ProjectsByProjectIdFeaturesReq, params oas.PatchMgmtV1ProjectsByProjectIdFeaturesParams) (r oas.PatchMgmtV1ProjectsByProjectIdFeaturesOK, _ error) {
	panic("implement me")
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
	token, err := s.deps.Projects.MintAdminToken(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	return &oas.PostMgmtV1ProjectsByProjectIdAdminTokensOK{
		AdminToken: oas.NewOptString(token),
	}, nil
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdConfigApply(ctx context.Context, req oas.PostMgmtV1ProjectsByProjectIdConfigApplyReq, params oas.PostMgmtV1ProjectsByProjectIdConfigApplyParams) (r oas.PostMgmtV1ProjectsByProjectIdConfigApplyOK, _ error) {
	panic("implement me")
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdConfigPlan(ctx context.Context, req oas.PostMgmtV1ProjectsByProjectIdConfigPlanReq, params oas.PostMgmtV1ProjectsByProjectIdConfigPlanParams) (r oas.PostMgmtV1ProjectsByProjectIdConfigPlanOK, _ error) {
	panic("implement me")
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
	return oas.Project{
		ID:               oas.NewOptString(p.ID),
		Name:             oas.NewOptString(p.Name),
		Slug:             oas.NewOptString(p.Slug),
		DefaultLocale:    oas.NewOptString(p.DefaultLocale),
		SupportedLocales: p.SupportedLocales,
		Environments:     p.Environments,
		CreatedAt:        oas.NewOptTimestamp(oas.Timestamp(p.CreatedAt)),
	}
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
