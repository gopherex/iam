// Code scaffolded for IAM handler groups. Each XxxService embeds
// oas.UnimplementedHandler (so non-1.0.0 / unwritten ops auto-return
// not-implemented) and panics on every v1.0.0 op until implemented.

package api

import "github.com/gopherex/iam/internal/oas"

// OAuthSocialService implements the OAuthSocialHandler slice of oas.Handler.
type OAuthSocialService struct{ oas.UnimplementedHandler }

var _ oas.Handler = (*OAuthSocialService)(nil)
