// Code scaffolded for IAM handler groups. Each XxxService embeds
// oas.UnimplementedHandler (so non-1.0.0 / unwritten ops auto-return
// not-implemented) and panics on every v1.0.0 op until implemented.

package api

import "github.com/gopherex/iam/internal/oas"

// OIDCProviderService implements the OIDCProviderHandler slice of oas.Handler.
type OIDCProviderService struct{ oas.UnimplementedHandler }

var _ oas.Handler = (*OIDCProviderService)(nil)
