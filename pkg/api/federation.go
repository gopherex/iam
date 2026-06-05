// Code scaffolded for IAM handler groups.
//
// FederationService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type FederationConnections interface {
	CreateConnection(ctx context.Context, cmd domain.ConnectionCmd) (*domain.Connection, error)
	GetConnection(ctx context.Context, projectID, id string) (*domain.Connection, error)
	ListConnections(ctx context.Context, projectID string) ([]domain.Connection, error)
	AddDomain(ctx context.Context, projectID, connectionID, name string) (*domain.Domain, error)
	VerifyDomain(ctx context.Context, projectID, domainID string) (*domain.Domain, error)
}

type FederationDeps struct{ Connections FederationConnections }

// FederationService implements the FederationHandler slice of oas.Handler.
type FederationService struct {
	oas.UnimplementedHandler
	deps FederationDeps
}

// NewFederationService builds the Federation service from its dependencies.
func NewFederationService(deps FederationDeps) *FederationService {
	return &FederationService{deps: deps}
}

var _ oas.Handler = (*FederationService)(nil)

func (s *FederationService) DeleteV1ProjectsByProjectIdAdminDomainsByDomainId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminDomainsByDomainIdParams) (*oas.Ok, error) {
	panic("implement me")
}

func (s *FederationService) DeleteV1ProjectsByProjectIdAdminSsoConnectionsById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminSsoConnectionsByIdParams) (*oas.Ok, error) {
	panic("implement me")
}

func (s *FederationService) DeleteV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensByTokenId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensByTokenIdParams) (*oas.Ok, error) {
	panic("implement me")
}

func (s *FederationService) DeleteV1ScimV2ByConnectionIdGroupsByGroupId(ctx context.Context, params oas.DeleteV1ScimV2ByConnectionIdGroupsByGroupIdParams) error {
	panic("implement me")
}

func (s *FederationService) DeleteV1ScimV2ByConnectionIdUsersByScimUserId(ctx context.Context, params oas.DeleteV1ScimV2ByConnectionIdUsersByScimUserIdParams) error {
	panic("implement me")
}

func (s *FederationService) GetV1ProjectsByProjectIdAdminDomains(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminDomainsParams) (*oas.GetV1ProjectsByProjectIdAdminDomainsOK, error) {
	// No port lists domains; only single domain add/verify are exposed.
	panic("implement me")
}

func (s *FederationService) GetV1ProjectsByProjectIdAdminSsoConnections(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSsoConnectionsParams) (*oas.GetV1ProjectsByProjectIdAdminSsoConnectionsOK, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	conns, err := s.deps.Connections.ListConnections(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.SSOConnection, 0, len(conns))
	for i := range conns {
		data = append(data, oasConnection(&conns[i]))
	}
	return &oas.GetV1ProjectsByProjectIdAdminSsoConnectionsOK{
		Data: data,
	}, nil
}

func (s *FederationService) GetV1ProjectsByProjectIdAdminSsoConnectionsById(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdParams) (*oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdOK, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	conn, err := s.deps.Connections.GetConnection(ctx, params.ProjectID, params.ID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdOK{
		Connection: oas.NewOptSSOConnection(oasConnection(conn)),
	}, nil
}

func (s *FederationService) GetV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokens(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensParams) (*oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensOK, error) {
	panic("implement me")
}

func (s *FederationService) GetV1ScimV2ByConnectionIdGroups(ctx context.Context, params oas.GetV1ScimV2ByConnectionIdGroupsParams) (oas.GetV1ScimV2ByConnectionIdGroupsOK, error) {
	panic("implement me")
}

func (s *FederationService) GetV1ScimV2ByConnectionIdGroupsByGroupId(ctx context.Context, params oas.GetV1ScimV2ByConnectionIdGroupsByGroupIdParams) (oas.GetV1ScimV2ByConnectionIdGroupsByGroupIdOK, error) {
	panic("implement me")
}

func (s *FederationService) GetV1ScimV2ByConnectionIdUsers(ctx context.Context, params oas.GetV1ScimV2ByConnectionIdUsersParams) (oas.GetV1ScimV2ByConnectionIdUsersOK, error) {
	panic("implement me")
}

func (s *FederationService) GetV1ScimV2ByConnectionIdUsersByScimUserId(ctx context.Context, params oas.GetV1ScimV2ByConnectionIdUsersByScimUserIdParams) (oas.GetV1ScimV2ByConnectionIdUsersByScimUserIdOK, error) {
	panic("implement me")
}

func (s *FederationService) GetV1SsoConnectionsResolve(ctx context.Context, params oas.GetV1SsoConnectionsResolveParams) (*oas.GetV1SsoConnectionsResolveOK, error) {
	panic("implement me")
}

func (s *FederationService) GetV1SsoOidcByConnectionIdCallback(ctx context.Context, params oas.GetV1SsoOidcByConnectionIdCallbackParams) (*oas.GetV1SsoOidcByConnectionIdCallbackFound, error) {
	panic("implement me")
}

func (s *FederationService) GetV1SsoOidcByConnectionIdStart(ctx context.Context, params oas.GetV1SsoOidcByConnectionIdStartParams) (*oas.GetV1SsoOidcByConnectionIdStartFound, error) {
	panic("implement me")
}

func (s *FederationService) GetV1SsoSamlByConnectionIdLogin(ctx context.Context, params oas.GetV1SsoSamlByConnectionIdLoginParams) (*oas.GetV1SsoSamlByConnectionIdLoginFound, error) {
	panic("implement me")
}

func (s *FederationService) GetV1SsoSamlByConnectionIdMetadata(ctx context.Context, params oas.GetV1SsoSamlByConnectionIdMetadataParams) (oas.GetV1SsoSamlByConnectionIdMetadataOK, error) {
	panic("implement me")
}

func (s *FederationService) PatchV1ProjectsByProjectIdAdminSsoConnectionsById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdReq, params oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdParams) (*oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdOK, error) {
	panic("implement me")
}

func (s *FederationService) PatchV1ScimV2ByConnectionIdGroupsByGroupId(ctx context.Context, req oas.PatchV1ScimV2ByConnectionIdGroupsByGroupIdReq, params oas.PatchV1ScimV2ByConnectionIdGroupsByGroupIdParams) (oas.PatchV1ScimV2ByConnectionIdGroupsByGroupIdOK, error) {
	panic("implement me")
}

func (s *FederationService) PatchV1ScimV2ByConnectionIdUsersByScimUserId(ctx context.Context, req *oas.ScimUser, params oas.PatchV1ScimV2ByConnectionIdUsersByScimUserIdParams) (oas.PatchV1ScimV2ByConnectionIdUsersByScimUserIdOK, error) {
	panic("implement me")
}

func (s *FederationService) PostV1ProjectsByProjectIdAdminDomains(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminDomainsReq, params oas.PostV1ProjectsByProjectIdAdminDomainsParams) (*oas.PostV1ProjectsByProjectIdAdminDomainsCreated, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	dom, err := s.deps.Connections.AddDomain(ctx, params.ProjectID, req.ConnectionID.Or(""), req.Domain)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminDomainsCreated{
		Domain: oas.NewOptDomain(oasDomain(dom)),
	}, nil
}

func (s *FederationService) PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerify(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerifyParams) (*oas.PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerifyOK, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	dom, err := s.deps.Connections.VerifyDomain(ctx, params.ProjectID, params.DomainID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerifyOK{
		Domain: oas.NewOptDomain(oasDomain(dom)),
	}, nil
}

func (s *FederationService) PostV1ProjectsByProjectIdAdminSsoConnections(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminSsoConnectionsReq, params oas.PostV1ProjectsByProjectIdAdminSsoConnectionsParams) (*oas.PostV1ProjectsByProjectIdAdminSsoConnectionsCreated, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	cmd := domain.ConnectionCmd{
		ProjectID: params.ProjectID,
		Type:      string(req.Type),
		Name:      req.Name,
		Domains:   req.Domains,
	}
	conn, err := s.deps.Connections.CreateConnection(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminSsoConnectionsCreated{
		Connection: oas.NewOptSSOConnection(oasConnection(conn)),
	}, nil
}

func (s *FederationService) PostV1ProjectsByProjectIdAdminSsoConnectionsByIdRotateCertificate(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdRotateCertificateParams) (*oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdRotateCertificateOK, error) {
	panic("implement me")
}

func (s *FederationService) PostV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokens(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensReq, params oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensParams) (*oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensCreated, error) {
	panic("implement me")
}

func (s *FederationService) PostV1ProjectsByProjectIdAdminSsoConnectionsByIdTest(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdTestParams) (*oas.PostV1ProjectsByProjectIdAdminSsoConnectionsByIdTestOK, error) {
	panic("implement me")
}

func (s *FederationService) PostV1ScimV2ByConnectionIdGroups(ctx context.Context, req oas.PostV1ScimV2ByConnectionIdGroupsReq, params oas.PostV1ScimV2ByConnectionIdGroupsParams) (oas.PostV1ScimV2ByConnectionIdGroupsCreated, error) {
	panic("implement me")
}

func (s *FederationService) PostV1ScimV2ByConnectionIdUsers(ctx context.Context, req *oas.ScimUser, params oas.PostV1ScimV2ByConnectionIdUsersParams) (oas.PostV1ScimV2ByConnectionIdUsersCreated, error) {
	panic("implement me")
}

func (s *FederationService) PostV1SsoExchange(ctx context.Context, req *oas.PostV1SsoExchangeReq, params oas.PostV1SsoExchangeParams) (*oas.AuthResult, error) {
	panic("implement me")
}

func (s *FederationService) PostV1SsoSamlByConnectionIdAcs(ctx context.Context, req oas.OptPostV1SsoSamlByConnectionIdAcsReq, params oas.PostV1SsoSamlByConnectionIdAcsParams) (*oas.PostV1SsoSamlByConnectionIdAcsFound, error) {
	panic("implement me")
}

func (s *FederationService) PostV1SsoSamlByConnectionIdSlo(ctx context.Context, params oas.PostV1SsoSamlByConnectionIdSloParams) (*oas.PostV1SsoSamlByConnectionIdSloFound, error) {
	panic("implement me")
}

func (s *FederationService) PutV1ScimV2ByConnectionIdGroupsByGroupId(ctx context.Context, req oas.PutV1ScimV2ByConnectionIdGroupsByGroupIdReq, params oas.PutV1ScimV2ByConnectionIdGroupsByGroupIdParams) (oas.PutV1ScimV2ByConnectionIdGroupsByGroupIdOK, error) {
	panic("implement me")
}

func (s *FederationService) PutV1ScimV2ByConnectionIdUsersByScimUserId(ctx context.Context, req *oas.ScimUser, params oas.PutV1ScimV2ByConnectionIdUsersByScimUserIdParams) (oas.PutV1ScimV2ByConnectionIdUsersByScimUserIdOK, error) {
	panic("implement me")
}

// oasConnection maps a domain Connection to its oas representation.
func oasConnection(c *domain.Connection) oas.SSOConnection {
	out := oas.SSOConnection{
		ID:      oas.NewOptString(c.ID),
		Name:    oas.NewOptString(c.Name),
		Status:  oas.NewOptString(c.Status),
		Domains: c.Domains,
	}
	if c.Type != "" {
		out.Type = oas.NewOptSSOConnectionType(oas.SSOConnectionType(c.Type))
	}
	if c.ExternalRef != "" {
		out.ExternalRef = oas.NewOptNilString(c.ExternalRef)
	}
	return out
}

// oasDomain maps a domain Domain to its oas representation.
func oasDomain(d *domain.Domain) oas.Domain {
	out := oas.Domain{
		ID:     oas.NewOptString(d.ID),
		Domain: oas.NewOptString(d.Domain),
	}
	if d.Status != "" {
		out.Status = oas.NewOptDomainStatus(oas.DomainStatus(d.Status))
	}
	if d.ConnectionID != "" {
		out.ConnectionID = oas.NewOptNilString(d.ConnectionID)
	}
	return out
}
