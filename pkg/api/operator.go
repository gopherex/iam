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

type operatorProjects interface {
	CreateProject(ctx context.Context, cmd domain.ProjectCmd) (*domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
	GetProject(ctx context.Context, projectID string) (*domain.Project, error)
	CreateEnvironment(ctx context.Context, cmd domain.EnvironmentCmd) (*domain.Environment, error)
	MintAdminToken(ctx context.Context, projectID string) (string, error)
}

type OperatorDeps struct{ Projects operatorProjects }

// OperatorService implements the OperatorHandler slice of oas.Handler.
type OperatorService struct {
	oas.UnimplementedHandler
	deps OperatorDeps
}

// NewOperatorService builds the Operator service from its dependencies.
func NewOperatorService(deps OperatorDeps) *OperatorService { return &OperatorService{deps: deps} }

var _ oas.Handler = (*OperatorService)(nil)

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectId(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdParams) (r oas.DeleteMgmtV1ProjectsByProjectIdRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenId(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenIdParams) (r oas.DeleteMgmtV1ProjectsByProjectIdAdminTokensByTokenIdRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnv(ctx context.Context, params oas.DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnvParams) (r oas.DeleteMgmtV1ProjectsByProjectIdEnvironmentsByEnvRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1Projects(ctx context.Context, params oas.GetMgmtV1ProjectsParams) (r oas.GetMgmtV1ProjectsRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectId(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdParams) (r oas.GetMgmtV1ProjectsByProjectIdRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdAdminTokens(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdAdminTokensParams) (r oas.GetMgmtV1ProjectsByProjectIdAdminTokensRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdConfigExport(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdConfigExportParams) (r oas.GetMgmtV1ProjectsByProjectIdConfigExportRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdEnvironments(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdEnvironmentsParams) (r oas.GetMgmtV1ProjectsByProjectIdEnvironmentsRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdEnvironmentsByEnv(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvParams) (r oas.GetMgmtV1ProjectsByProjectIdEnvironmentsByEnvRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) GetMgmtV1ProjectsByProjectIdFeatures(ctx context.Context, params oas.GetMgmtV1ProjectsByProjectIdFeaturesParams) (r oas.GetMgmtV1ProjectsByProjectIdFeaturesRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PatchMgmtV1ProjectsByProjectId(ctx context.Context, req oas.PatchMgmtV1ProjectsByProjectIdReq, params oas.PatchMgmtV1ProjectsByProjectIdParams) (r oas.PatchMgmtV1ProjectsByProjectIdRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PatchMgmtV1ProjectsByProjectIdFeatures(ctx context.Context, req oas.PatchMgmtV1ProjectsByProjectIdFeaturesReq, params oas.PatchMgmtV1ProjectsByProjectIdFeaturesParams) (r oas.PatchMgmtV1ProjectsByProjectIdFeaturesRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PostMgmtV1Projects(ctx context.Context, req *oas.PostMgmtV1ProjectsReq, params oas.PostMgmtV1ProjectsParams) (r oas.PostMgmtV1ProjectsRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdAdminTokens(ctx context.Context, req *oas.PostMgmtV1ProjectsByProjectIdAdminTokensReq, params oas.PostMgmtV1ProjectsByProjectIdAdminTokensParams) (r oas.PostMgmtV1ProjectsByProjectIdAdminTokensRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdConfigApply(ctx context.Context, req oas.PostMgmtV1ProjectsByProjectIdConfigApplyReq, params oas.PostMgmtV1ProjectsByProjectIdConfigApplyParams) (r oas.PostMgmtV1ProjectsByProjectIdConfigApplyRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdConfigPlan(ctx context.Context, req oas.PostMgmtV1ProjectsByProjectIdConfigPlanReq, params oas.PostMgmtV1ProjectsByProjectIdConfigPlanParams) (r oas.PostMgmtV1ProjectsByProjectIdConfigPlanRes, _ error) {
	panic("implement me")
}

func (s *OperatorService) PostMgmtV1ProjectsByProjectIdEnvironments(ctx context.Context, req *oas.PostMgmtV1ProjectsByProjectIdEnvironmentsReq, params oas.PostMgmtV1ProjectsByProjectIdEnvironmentsParams) (r oas.PostMgmtV1ProjectsByProjectIdEnvironmentsRes, _ error) {
	panic("implement me")
}
