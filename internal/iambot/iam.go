package iambot

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/gopherex/iam/internal/oas"
)

// adminTokenSource satisfies oas.SecuritySource with a static project-admin
// token; every other scheme is skipped (the bot only calls adminToken ops).
type adminTokenSource struct{ token string }

func (s adminTokenSource) AdminToken(context.Context, oas.OperationName, *oas.Client) (oas.AdminToken, error) {
	return oas.AdminToken{Token: s.token}, nil
}

func (adminTokenSource) BearerAuth(context.Context, oas.OperationName, *oas.Client) (oas.BearerAuth, error) {
	return oas.BearerAuth{}, ogenerrors.ErrSkipServerSecurity
}

func (adminTokenSource) ClientSecretBasic(
	context.Context, oas.OperationName, *oas.Client,
) (oas.ClientSecretBasic, error) {
	return oas.ClientSecretBasic{}, ogenerrors.ErrSkipServerSecurity
}

func (adminTokenSource) MasterKey(context.Context, oas.OperationName, *oas.Client) (oas.MasterKey, error) {
	return oas.MasterKey{}, ogenerrors.ErrSkipServerSecurity
}

func (adminTokenSource) OAuth2(context.Context, oas.OperationName, *oas.Client) (oas.OAuth2, error) {
	return oas.OAuth2{}, ogenerrors.ErrSkipServerSecurity
}

func (adminTokenSource) RegistrationToken(
	context.Context, oas.OperationName, *oas.Client,
) (oas.RegistrationToken, error) {
	return oas.RegistrationToken{}, ogenerrors.ErrSkipServerSecurity
}

func (adminTokenSource) ScimToken(context.Context, oas.OperationName, *oas.Client) (oas.ScimToken, error) {
	return oas.ScimToken{}, ogenerrors.ErrSkipServerSecurity
}

func (adminTokenSource) ServiceToken(context.Context, oas.OperationName, *oas.Client) (oas.ServiceToken, error) {
	return oas.ServiceToken{}, ogenerrors.ErrSkipServerSecurity
}

// IAM is the thin admin-API surface the bot needs.
type IAM struct {
	api       *oas.Client
	projectID string
	env       string
}

// NewIAM builds the admin-API client for one project.
func NewIAM(baseURL, adminToken, projectID, env string) (*IAM, error) {
	api, err := oas.NewClient(baseURL, adminTokenSource{token: adminToken})
	if err != nil {
		return nil, fmt.Errorf("iam client: %w", err)
	}

	return &IAM{api: api, projectID: projectID, env: env}, nil
}

// envOpt returns the X-Environment parameter value (empty means "live").
func (c *IAM) envOpt() oas.OptString {
	if c.env == "" {
		return oas.OptString{}
	}

	return oas.NewOptString(c.env)
}

// AccessRequest is the bot-facing view of a pending request.
type AccessRequest struct {
	ID     string
	Email  string
	Reason string
	Status string
}

// ListPending returns every pending access request, following the keyset
// pagination until the listing is exhausted.
func (c *IAM) ListPending(ctx context.Context) ([]AccessRequest, error) {
	var out []AccessRequest

	var cursor string

	for {
		params := oas.GetV1ProjectsByProjectIdAdminAccessRequestsParams{
			ProjectID:    c.projectID,
			Status:       oas.NewOptString("pending"),
			XEnvironment: c.envOpt(),
		}
		if cursor != "" {
			params.Cursor = oas.NewOptString(cursor)
		}

		page, err := c.api.GetV1ProjectsByProjectIdAdminAccessRequests(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("list access requests: %w", err)
		}

		for i := range page.Data {
			out = append(out, accessRequestFromOAS(&page.Data[i]))
		}

		if !page.HasMore.Or(false) {
			return out, nil
		}

		next, ok := page.NextCursor.Get()
		if !ok || next == "" {
			return out, nil
		}

		cursor = next
	}
}

func accessRequestFromOAS(r *oas.AccessRequest) AccessRequest {
	out := AccessRequest{}
	if v, ok := r.ID.Get(); ok {
		out.ID = v
	}

	if v, ok := r.Email.Get(); ok {
		out.Email = v
	}

	if v, ok := r.Reason.Get(); ok {
		out.Reason = v
	}

	if v, ok := r.Status.Get(); ok {
		out.Status = string(v)
	}

	return out
}

// ErrNotFound marks a decision on a request that is gone (already decided by
// someone else, or deleted).
var ErrNotFound = errors.New("iambot: access request not found")

// Approve approves a pending request. IAM mints the email-bound invite and
// sends the decision email with the magic link as part of the same call.
func (c *IAM) Approve(ctx context.Context, id string) error {
	_, err := c.api.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApprove(ctx, nil,
		oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveParams{
			ProjectID:    c.projectID,
			ID:           id,
			XEnvironment: c.envOpt(),
		})
	if isNotFound(err) {
		return ErrNotFound
	}

	return err
}

// Deny denies a pending request with an optional reason (surfaced in the
// denial email when set).
func (c *IAM) Deny(ctx context.Context, id, reason string) error {
	req := oas.OptPostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyReq{}
	if reason != "" {
		req = oas.NewOptPostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyReq(
			oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyReq{
				Reason: oas.NewOptString(reason),
			})
	}

	_, err := c.api.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDeny(ctx, req,
		oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyParams{
			ProjectID:    c.projectID,
			ID:           id,
			XEnvironment: c.envOpt(),
		})
	if isNotFound(err) {
		return ErrNotFound
	}

	return err
}

// CreateInvite mints an email-bound invitation; the raw token is returned
// exactly once (and IAM emails it to the invitee as well).
func (c *IAM) CreateInvite(ctx context.Context, email string) (string, error) {
	created, err := c.api.PostV1ProjectsByProjectIdAdminInvites(ctx, &oas.InviteCreateRequest{
		Email: oas.NewOptString(email),
	}, oas.PostV1ProjectsByProjectIdAdminInvitesParams{
		ProjectID:    c.projectID,
		XEnvironment: c.envOpt(),
	})
	if err != nil {
		return "", fmt.Errorf("create invite: %w", err)
	}

	return created.InviteToken, nil
}

// isNotFound maps the generated client's default-error with a 404 status.
func isNotFound(err error) bool {
	var nf *oas.DefaultStatusCode
	if errors.As(err, &nf) {
		return nf.StatusCode == http.StatusNotFound
	}

	return false
}
