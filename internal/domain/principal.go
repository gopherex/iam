package domain

// PrincipalKind is the kind of authenticated caller.
type PrincipalKind string

const (
	PrincipalUser     PrincipalKind = "user"     // bearerAuth — end-user access token
	PrincipalAdmin    PrincipalKind = "admin"    // adminToken — project admin
	PrincipalOperator PrincipalKind = "operator" // masterKey
	PrincipalService  PrincipalKind = "service"  // serviceToken / API key
	PrincipalSCIM     PrincipalKind = "scim"     // scimToken
	PrincipalClient   PrincipalKind = "client"   // OAuth client (client_secret_basic)
)

// Principal is the authenticated subject a SecurityHandler resolves from a
// credential and carries in the request context. Handlers read it instead of
// re-parsing tokens.
type Principal struct {
	Kind         PrincipalKind
	AccountID    string // the user, for PrincipalUser
	ProjectID    string
	Environment  string
	SessionID    string
	ClientID     string
	ConnectionID string // for PrincipalSCIM
	Scopes       []string
	AAL          int
}
