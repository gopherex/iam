package postgres

// Back-channel logout delivery (OpenID Connect Back-Channel Logout 1.0).
//
// The discovery document has always advertised backchannel_logout_supported,
// and nothing ever sent a logout token: a relying party that trusted the
// metadata kept its sessions alive after the user logged out of IAM. This is the
// sending half.
//
// Delivery runs from the outbox publisher, not from the request that ended the
// session, so a slow or unreachable relying party cannot hold up a logout and a
// failed POST is retried rather than lost.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// errBackchannelDelivery / errBackchannelRejected are the delivery failures the
// outbox retries on.
var (
	errBackchannelDelivery = errors.New("backchannel logout not delivered")
	errBackchannelRejected = errors.New("relying party rejected the logout token")
)

// backchannelLogoutEventType is the SET event a logout token carries.
const backchannelLogoutEventType = "http://schemas.openid.net/event/backchannel-logout"

// backchannelLogoutTTL bounds a logout token. It is presented immediately by the
// receiving RP, so it stays short.
const backchannelLogoutTTL = 2 * time.Minute

// BackchannelLogoutTarget is one relying party to notify.
type BackchannelLogoutTarget struct {
	ClientID string
	URI      string
}

// backchannelTargets lists the clients that hold a grant on a session and have
// asked to be told when it ends.
//
// "Holds a grant" is read from the refresh tokens issued for the session,
// revoked ones included: a client whose tokens were just revoked by the logout
// is precisely the one that needs to hear about it.
func backchannelTargets(ctx context.Context, db *DB, projectID, sessionID string) ([]BackchannelLogoutTarget, error) {
	if projectID == "" || sessionID == "" {
		return nil, nil
	}

	rows, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamRefreshTokens.Columns.SessionID.EQ(psql.Arg(sessionID))),
	).All(ctx, db.Bobx())
	if err != nil {
		return nil, fmt.Errorf("read session grants: %w", err)
	}

	seen := make(map[string]struct{}, len(rows))
	out := make([]BackchannelLogoutTarget, 0, len(rows))

	for _, row := range rows {
		var data oidcRefreshData
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &data); err != nil {
				continue
			}
		}

		if data.ClientID == "" {
			continue
		}

		if _, dup := seen[data.ClientID]; dup {
			continue
		}

		seen[data.ClientID] = struct{}{}

		clientRow, err := models.FindIamAppClient(ctx, db.Bobx(), data.ClientID)
		if err != nil {
			continue
		}

		var app domain.AppClient
		if err := unmarshal(clientRow.Data, &app); err != nil {
			continue
		}

		if app.BackchannelLogoutURI == "" {
			continue
		}

		out = append(out, BackchannelLogoutTarget{ClientID: data.ClientID, URI: app.BackchannelLogoutURI})
	}

	return out, nil
}

// DeliverBackchannelLogout notifies every relying party holding a grant on the
// ended session. Delivery is best-effort per RP — one unreachable client must not
// stop the others — and the caller (the outbox) retries the batch.
func DeliverBackchannelLogout(
	ctx context.Context, db *DB, client *http.Client, projectID, env, sessionID, subject string,
) error {
	targets, err := backchannelTargets(ctx, db, projectID, sessionID)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		return nil
	}

	if env == "" {
		env = oidcDefaultEnv
	}

	issuer := oidcIssuer(db.PublicURL, projectID, env)

	var failed []string

	for _, target := range targets {
		token, err := mintLogoutToken(ctx, db, issuer, projectID, env, target.ClientID, sessionID, subject)
		if err != nil {
			return err
		}

		if err := postLogoutToken(ctx, client, target.URI, token); err != nil {
			failed = append(failed, target.ClientID)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("%w: %s", errBackchannelDelivery, strings.Join(failed, ", "))
	}

	return nil
}

// mintLogoutToken builds the signed logout_token for one relying party. The
// claim set is the one the spec requires, and deliberately carries no `nonce`:
// its presence is how a receiver tells a logout token from an id_token.
func mintLogoutToken(
	ctx context.Context, db *DB, issuer, projectID, env, clientID, sessionID, subject string,
) (string, error) {
	claims := map[string]any{
		claimIssuer:   issuer,
		claimAudience: clientID,
		claimTokenID:  newUUID(),
		"events":      map[string]any{backchannelLogoutEventType: map[string]any{}},
	}

	if subject != "" {
		claims[claimSubject] = subject
	}

	if sessionID != "" {
		claims[claimSessionID] = sessionID
	}

	token, err := db.Signer().Sign(ctx, projectID, env, claims, backchannelLogoutTTL)
	if err != nil {
		return "", fmt.Errorf("sign logout token: %w", err)
	}

	return token, nil
}

// postLogoutToken delivers one token. The URI is operator-configured but still
// goes through the hardened client (SSRF guards, redirect limits) that webhook
// delivery uses — a logout notification is an outbound request to an address we
// were told about, exactly like a webhook.
func postLogoutToken(ctx context.Context, client *http.Client, uri, token string) error {
	body := url.Values{"logout_token": {token}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build logout request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cache-Control", "no-store")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver logout token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%w: status %d", errBackchannelRejected, resp.StatusCode)
	}

	return nil
}
