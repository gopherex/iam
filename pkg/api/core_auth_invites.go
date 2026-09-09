package api

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

// memberInviteScope resolves the caller's project/env from the authenticated
// principal — member invites are strictly self-scoped.
type memberInviteScope struct {
	ProjectID string
	Env       string
	UserID    string
}

func memberInviteScopeFrom(ctx context.Context) (memberInviteScope, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return memberInviteScope{}, err
	}

	env := p.Environment
	if env == "" {
		env = "live"
	}

	return memberInviteScope{ProjectID: p.ProjectID, Env: env, UserID: p.AccountID}, nil
}

func oasInviteQuota(q domain.InviteQuota) oas.InviteQuota {
	return oas.InviteQuota{
		Cap:  oas.NewOptInt(q.Cap),
		Used: oas.NewOptInt(q.Used),
		Left: oas.NewOptInt(q.Left),
	}
}

// PostV1AuthInvites creates a member invitation: email-bound, on the caller's
// own quota. The invitee is emailed immediately; the raw token returns once
// for the caller to share.
func (s *CoreAuthService) PostV1AuthInvites(
	ctx context.Context,
	req *oas.PostV1AuthInvitesReq,
) (*oas.PostV1AuthInvitesCreated, error) {
	scope, err := memberInviteScopeFrom(ctx)
	if err != nil {
		return nil, err
	}

	created, err := s.deps.Invites.Create(ctx, domain.InviteCreateCmd{
		ProjectID:   scope.ProjectID,
		Environment: scope.Env,
		Email:       req.Email,
		RedirectTo:  req.RedirectTo.Or(""),
		CreatedBy:   scope.UserID,
	})
	if err != nil {
		return nil, err
	}

	quota, err := s.deps.Invites.InviteQuota(ctx, scope.ProjectID, scope.Env, scope.UserID)
	if err != nil {
		return nil, err
	}

	out := &oas.PostV1AuthInvitesCreated{
		Invite: oas.NewOptInviteCreated(oasInviteCreated(created)),
		Quota:  oas.NewOptInviteQuota(oasInviteQuota(quota)),
	}

	return out, nil
}

// GetV1AuthInvites lists the caller's own invitations and their quota.
func (s *CoreAuthService) GetV1AuthInvites(
	ctx context.Context,
) (*oas.GetV1AuthInvitesOK, error) {
	scope, err := memberInviteScopeFrom(ctx)
	if err != nil {
		return nil, err
	}

	invites, err := s.deps.Invites.ListByCreator(ctx, scope.ProjectID, scope.Env, scope.UserID)
	if err != nil {
		return nil, err
	}

	quota, err := s.deps.Invites.InviteQuota(ctx, scope.ProjectID, scope.Env, scope.UserID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.Invite, 0, len(invites))
	for i := range invites {
		data = append(data, oasInvite(&invites[i]))
	}

	return &oas.GetV1AuthInvitesOK{
		Invites: data,
		Quota:   oas.NewOptInviteQuota(oasInviteQuota(quota)),
	}, nil
}

// PostV1AuthInvitesByInviteIdRevoke revokes the caller's own pending
// invitation, freeing its quota slot.
func (s *CoreAuthService) PostV1AuthInvitesByInviteIdRevoke(
	ctx context.Context,
	params oas.PostV1AuthInvitesByInviteIdRevokeParams,
) (*oas.Ok, error) {
	scope, err := memberInviteScopeFrom(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.deps.Invites.RevokeOwn(ctx, scope.ProjectID, scope.Env, params.InviteID, scope.UserID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}
