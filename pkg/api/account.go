// Code scaffolded for IAM handler groups.
//
// AccountService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type AccountStore interface {
	Get(ctx context.Context, projectID, accountID string) (*domain.Account, error)
	UpdateProfile(ctx context.Context, cmd domain.ProfileUpdateCmd) (*domain.Account, error)
	Delete(ctx context.Context, projectID, accountID string) error
	ListSessions(ctx context.Context, accountID string) ([]domain.Session, error)
	RevokeSession(ctx context.Context, accountID, sessionID string) error
	ListIdentities(ctx context.Context, accountID string) ([]domain.Identity, error)

	// Capabilities returns the feature/capability flags available to the account.
	Capabilities(ctx context.Context, projectID, accountID string) (map[string]bool, error)
	// GetSession resolves a single session owned by the account.
	GetSession(ctx context.Context, accountID, sessionID string) (*domain.Session, error)
	// RenameSession sets a device name on one of the account's sessions.
	RenameSession(ctx context.Context, cmd domain.AccountRenameSessionCmd) (*domain.Session, error)
	// TrustSession marks a session trusted for the given duration.
	TrustSession(ctx context.Context, cmd domain.AccountTrustSessionCmd) (*domain.Session, error)
	// RevokeSessions bulk-revokes the account's sessions; returns the count revoked.
	RevokeSessions(ctx context.Context, cmd domain.AccountRevokeSessionsCmd) (int, error)
	// UnlinkIdentity removes a linked identity from the account.
	UnlinkIdentity(ctx context.Context, accountID, identityID string) error
	// Activity returns the account's paginated activity log.
	Activity(ctx context.Context, cmd domain.AccountActivityCmd) (*domain.AccountActivityPage, error)
	// Consents returns the account's recorded consent acceptances.
	Consents(ctx context.Context, accountID string) ([]domain.AccountConsent, error)
	// AcceptConsents records consent acceptances and returns the updated set.
	AcceptConsents(ctx context.Context, cmd domain.AccountAcceptConsentsCmd) ([]domain.AccountConsent, error)
	// StartExport kicks off a data-export job and returns its identifier.
	StartExport(ctx context.Context, accountID string) (*domain.AccountExportJob, error)
	// ExportStatus reports the state of a data-export job.
	ExportStatus(ctx context.Context, accountID, jobID string) (*domain.AccountExportJob, error)
	// StartIdentityMerge begins merging another identity into the account.
	StartIdentityMerge(ctx context.Context, cmd domain.AccountMergeStartCmd) (*domain.Challenge, error)
	// ConfirmIdentityMerge completes a pending identity merge.
	ConfirmIdentityMerge(ctx context.Context, cmd domain.AccountMergeConfirmCmd) (*domain.Account, []domain.Identity, error)
}

type AccountDeps struct{ Accounts AccountStore }

// AccountService implements the AccountHandler slice of oas.Handler.
type AccountService struct {
	oas.UnimplementedHandler
	deps AccountDeps
}

// NewAccountService builds the Account service from its dependencies.
func NewAccountService(deps AccountDeps) *AccountService { return &AccountService{deps: deps} }

var _ oas.Handler = (*AccountService)(nil)

func (s *AccountService) DeleteV1AuthIdentitiesByIdentityId(ctx context.Context, params oas.DeleteV1AuthIdentitiesByIdentityIdParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.UnlinkIdentity(ctx, p.AccountID, params.IdentityID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AccountService) DeleteV1Sessions(ctx context.Context, req oas.OptDeleteV1SessionsReq) (*oas.DeleteV1SessionsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	cmd := domain.AccountRevokeSessionsCmd{AccountID: p.AccountID}
	if v, ok := req.Get(); ok && v.ExceptCurrent.Or(false) {
		cmd.ExceptCurrent = true
		cmd.ExceptSessionID = p.SessionID
	}
	n, err := s.deps.Accounts.RevokeSessions(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.DeleteV1SessionsOK{RevokedCount: oas.NewOptInt(n)}, nil
}

func (s *AccountService) DeleteV1SessionsBySessionId(ctx context.Context, params oas.DeleteV1SessionsBySessionIdParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.RevokeSession(ctx, p.AccountID, params.SessionID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AccountService) DeleteV1UsersMe(ctx context.Context, req oas.OptDeleteV1UsersMeReq) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.Delete(ctx, p.ProjectID, p.AccountID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AccountService) GetV1AccountCapabilities(ctx context.Context) (*oas.GetV1AccountCapabilitiesOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	caps, err := s.deps.Accounts.Capabilities(ctx, p.ProjectID, p.AccountID)
	if err != nil {
		return nil, err
	}
	raw := make(map[string]any, len(caps))
	for k, v := range caps {
		raw[k] = v
	}
	return &oas.GetV1AccountCapabilitiesOK{
		Capabilities: oas.NewOptGetV1AccountCapabilitiesOKCapabilities(
			oasRawMap[oas.GetV1AccountCapabilitiesOKCapabilities](raw),
		),
	}, nil
}

func (s *AccountService) GetV1AuthIdentities(ctx context.Context) (*oas.GetV1AuthIdentitiesOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := s.deps.Accounts.ListIdentities(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.Identity, 0, len(ids))
	for i := range ids {
		data = append(data, oasIdentity(&ids[i]))
	}
	return &oas.GetV1AuthIdentitiesOK{Data: data}, nil
}

func (s *AccountService) GetV1Sessions(ctx context.Context) (*oas.GetV1SessionsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := s.deps.Accounts.ListSessions(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.Session, 0, len(sessions))
	for i := range sessions {
		sessions[i].Current = sessions[i].ID == p.SessionID
		data = append(data, oasSession(&sessions[i]))
	}
	return &oas.GetV1SessionsOK{Data: data}, nil
}

func (s *AccountService) GetV1SessionsCurrent(ctx context.Context) (*oas.GetV1SessionsCurrentOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := s.deps.Accounts.GetSession(ctx, p.AccountID, p.SessionID)
	if err != nil {
		return nil, err
	}
	sess.Current = true
	return &oas.GetV1SessionsCurrentOK{Session: oas.NewOptSession(oasSession(sess))}, nil
}

func (s *AccountService) GetV1UsersMe(ctx context.Context) (*oas.GetV1UsersMeOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.deps.Accounts.Get(ctx, p.ProjectID, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1UsersMeOK{User: oas.NewOptUser(oasUser(acct))}, nil
}

func (s *AccountService) GetV1UsersMeActivity(ctx context.Context, params oas.GetV1UsersMeActivityParams) (*oas.GetV1UsersMeActivityOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, err := s.deps.Accounts.Activity(ctx, domain.AccountActivityCmd{
		AccountID: p.AccountID,
		Type:      params.Type.Or(""),
		Cursor:    params.Cursor.Or(""),
		Limit:     params.Limit.Or(0),
	})
	if err != nil {
		return nil, err
	}
	data := make([]oas.ActivityEvent, 0, len(page.Events))
	for i := range page.Events {
		data = append(data, oasAccountActivityEvent(&page.Events[i]))
	}
	out := &oas.GetV1UsersMeActivityOK{Data: data, HasMore: oas.NewOptBool(page.HasMore)}
	if page.NextCursor != "" {
		out.NextCursor = oas.NewOptNilString(page.NextCursor)
	}
	return out, nil
}

func (s *AccountService) GetV1UsersMeConsents(ctx context.Context) (*oas.GetV1UsersMeConsentsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	consents, err := s.deps.Accounts.Consents(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	items := make([]oas.GetV1UsersMeConsentsOKConsentsItem, 0, len(consents))
	for i := range consents {
		items = append(items, oasAccountConsent(&consents[i]))
	}
	return &oas.GetV1UsersMeConsentsOK{Consents: items}, nil
}

func (s *AccountService) GetV1UsersMeExportByJobId(ctx context.Context, params oas.GetV1UsersMeExportByJobIdParams) (*oas.GetV1UsersMeExportByJobIdOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	job, err := s.deps.Accounts.ExportStatus(ctx, p.AccountID, params.JobID)
	if err != nil {
		return nil, err
	}
	out := &oas.GetV1UsersMeExportByJobIdOK{Status: oas.NewOptString(job.Status)}
	if job.DownloadURL != "" {
		out.DownloadURL = oas.NewOptNilString(job.DownloadURL)
	}
	return out, nil
}

func (s *AccountService) PatchV1SessionsBySessionId(ctx context.Context, req *oas.PatchV1SessionsBySessionIdReq, params oas.PatchV1SessionsBySessionIdParams) (*oas.PatchV1SessionsBySessionIdOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := s.deps.Accounts.RenameSession(ctx, domain.AccountRenameSessionCmd{
		AccountID:  p.AccountID,
		SessionID:  params.SessionID,
		DeviceName: req.DeviceName,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PatchV1SessionsBySessionIdOK{Session: oas.NewOptSession(oasSession(sess))}, nil
}

func (s *AccountService) PatchV1UsersMe(ctx context.Context, req *oas.PatchV1UsersMeReq) (*oas.PatchV1UsersMeOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.deps.Accounts.UpdateProfile(ctx, domain.ProfileUpdateCmd{
		ProjectID: p.ProjectID,
		AccountID: p.AccountID,
		Name:      req.Name.Or(""),
		AvatarURL: req.AvatarURL.Or(""),
		Locale:    req.Locale.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PatchV1UsersMeOK{User: oas.NewOptUser(oasUser(acct))}, nil
}

func (s *AccountService) PostV1AuthIdentitiesMergeConfirm(ctx context.Context, req *oas.PostV1AuthIdentitiesMergeConfirmReq) (*oas.PostV1AuthIdentitiesMergeConfirmOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, ids, err := s.deps.Accounts.ConfirmIdentityMerge(ctx, domain.AccountMergeConfirmCmd{
		AccountID:   p.AccountID,
		ChallengeID: req.ChallengeID,
		Code:        req.Code,
	})
	if err != nil {
		return nil, err
	}
	data := make([]oas.Identity, 0, len(ids))
	for i := range ids {
		data = append(data, oasIdentity(&ids[i]))
	}
	return &oas.PostV1AuthIdentitiesMergeConfirmOK{
		User:       oas.NewOptUser(oasUser(acct)),
		Identities: data,
	}, nil
}

func (s *AccountService) PostV1AuthIdentitiesMergeStart(ctx context.Context, req *oas.PostV1AuthIdentitiesMergeStartReq) (*oas.PostV1AuthIdentitiesMergeStartOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ch, err := s.deps.Accounts.StartIdentityMerge(ctx, domain.AccountMergeStartCmd{
		AccountID:        p.AccountID,
		TargetIdentifier: req.TargetIdentifier,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthIdentitiesMergeStartOK{ChallengeID: oas.NewOptString(ch.ID)}, nil
}

func (s *AccountService) PostV1SessionsBySessionIdTrust(ctx context.Context, req *oas.PostV1SessionsBySessionIdTrustReq, params oas.PostV1SessionsBySessionIdTrustParams) (*oas.PostV1SessionsBySessionIdTrustOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := s.deps.Accounts.TrustSession(ctx, domain.AccountTrustSessionCmd{
		AccountID:       p.AccountID,
		SessionID:       params.SessionID,
		DurationSeconds: req.DurationSeconds,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1SessionsBySessionIdTrustOK{Session: oas.NewOptSession(oasSession(sess))}, nil
}

func (s *AccountService) PostV1UsersMeConsents(ctx context.Context, req *oas.PostV1UsersMeConsentsReq) (*oas.PostV1UsersMeConsentsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	accept := make([]domain.AccountConsentAcceptance, 0, len(req.Accept))
	for _, a := range req.Accept {
		accept = append(accept, domain.AccountConsentAcceptance{Key: a.Key, Version: a.Version})
	}
	consents, err := s.deps.Accounts.AcceptConsents(ctx, domain.AccountAcceptConsentsCmd{
		AccountID: p.AccountID,
		Accept:    accept,
	})
	if err != nil {
		return nil, err
	}
	items := make([]oas.PostV1UsersMeConsentsOKConsentsItem, 0, len(consents))
	for i := range consents {
		items = append(items, oasAccountConsentRaw(&consents[i]))
	}
	return &oas.PostV1UsersMeConsentsOK{Consents: items}, nil
}

func (s *AccountService) PostV1UsersMeExport(ctx context.Context) (*oas.PostV1UsersMeExportOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	job, err := s.deps.Accounts.StartExport(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1UsersMeExportOK{JobID: oas.NewOptString(job.JobID)}, nil
}

// oasIdentity maps a domain Identity to its wire representation.
func oasIdentity(i *domain.Identity) oas.Identity {
	id := oas.Identity{
		ID:   i.ID,
		Type: oas.IdentityType(i.Type),
	}
	if i.Provider != "" {
		id.Provider = oas.NewOptNilString(i.Provider)
	}
	if i.ProviderAccountID != "" {
		id.ProviderAccountID = oas.NewOptNilString(i.ProviderAccountID)
	}
	if i.Email != "" {
		id.Email = oas.NewOptNilString(i.Email)
	}
	return id
}

// oasAccountActivityEvent maps a domain activity event to its wire form.
func oasAccountActivityEvent(e *domain.AccountActivityEvent) oas.ActivityEvent {
	ev := oas.ActivityEvent{
		ID:   oas.NewOptString(e.ID),
		Type: oas.NewOptString(e.Type),
		At:   oas.NewOptTimestamp(oas.Timestamp(e.At)),
	}
	if e.IP != "" {
		ev.IP = oas.NewOptNilString(e.IP)
	}
	if e.Device != "" {
		ev.Device = oas.NewOptNilString(e.Device)
	}
	return ev
}

// oasAccountConsent maps a domain consent to the GET consents list item.
func oasAccountConsent(c *domain.AccountConsent) oas.GetV1UsersMeConsentsOKConsentsItem {
	item := oas.GetV1UsersMeConsentsOKConsentsItem{
		Key:        oas.NewOptString(c.Key),
		Version:    oas.NewOptString(c.Version),
		AcceptedAt: oas.NewOptTimestamp(oas.Timestamp(c.AcceptedAt)),
	}
	// Only set locale when present: an empty string fails the oas locale
	// pattern on response validation, which would reject the whole response.
	if c.Locale != "" {
		item.Locale = oas.NewOptString(c.Locale)
	}
	if c.URL != "" {
		item.URL = oas.NewOptNilString(c.URL)
	}
	return item
}

// oasAccountConsentRaw maps a domain consent to the POST consents raw-map item.
func oasAccountConsentRaw(c *domain.AccountConsent) oas.PostV1UsersMeConsentsOKConsentsItem {
	raw := map[string]any{
		"key":     c.Key,
		"version": c.Version,
	}
	if !c.AcceptedAt.IsZero() {
		raw["accepted_at"] = c.AcceptedAt
	}
	if c.URL != "" {
		raw["url"] = c.URL
	}
	if c.Locale != "" {
		raw["locale"] = c.Locale
	}
	return oasRawMap[oas.PostV1UsersMeConsentsOKConsentsItem](raw)
}
