package api

import (
	"context"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

// oasInvite maps a domain.Invite onto the wire oas.Invite.
func oasInvite(inv *domain.Invite) oas.Invite {
	out := oas.Invite{
		ID:     inv.ID,
		Status: oas.InviteStatus(inv.Status),
	}
	if inv.Email != "" {
		out.Email = oas.NewOptString(inv.Email)
	}
	if !inv.ExpiresAt.IsZero() {
		out.ExpiresAt = oas.NewOptTimestamp(oas.Timestamp(inv.ExpiresAt))
	}
	if !inv.CreatedAt.IsZero() {
		out.CreatedAt = oas.NewOptTimestamp(oas.Timestamp(inv.CreatedAt))
	}
	return out
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminInvites(ctx context.Context, req *oas.InviteCreateRequest, params oas.PostV1ProjectsByProjectIdAdminInvitesParams) (*oas.InviteCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	cmd := domain.InviteCreateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Email:       req.Email.Or(""),
		RedirectTo:  req.RedirectTo.Or(""),
	}
	if v, ok := req.ExpiresAt.Get(); ok {
		cmd.ExpiresAt = time.Time(v)
	}
	created, err := s.deps.Invites.Create(ctx, cmd)
	if err != nil {
		return nil, err
	}
	out := &oas.InviteCreated{
		ID:          created.ID,
		Status:      oas.InviteCreatedStatus(created.Status),
		InviteToken: created.Token,
	}
	if created.Email != "" {
		out.Email = oas.NewOptString(created.Email)
	}
	if !created.ExpiresAt.IsZero() {
		out.ExpiresAt = oas.NewOptTimestamp(oas.Timestamp(created.ExpiresAt))
	}
	if !created.CreatedAt.IsZero() {
		out.CreatedAt = oas.NewOptTimestamp(oas.Timestamp(created.CreatedAt))
	}
	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminInvites(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminInvitesParams) (*oas.GetV1ProjectsByProjectIdAdminInvitesOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	invites, err := s.deps.Invites.List(ctx, domain.InviteListCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
	})
	if err != nil {
		return nil, err
	}
	data := make([]oas.Invite, 0, len(invites))
	for i := range invites {
		data = append(data, oasInvite(&invites[i]))
	}
	return &oas.GetV1ProjectsByProjectIdAdminInvitesOK{Invites: data}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminInvitesByInviteIdRevoke(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminInvitesByInviteIdRevokeParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := s.deps.Invites.Revoke(ctx, domain.InviteRevokeCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		InviteID:    params.InviteID,
	}); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}
