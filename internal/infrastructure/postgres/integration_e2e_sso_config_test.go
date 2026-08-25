//go:build integration

package postgres

// integration_e2e_sso_config_test.go — configuring an enterprise SSO connection
// through the admin API.
//
// The thing worth proving is that a connection is usable after nothing but API
// calls. Before this, the local half of a SAML connection (the SP entity id and
// ACS URL that assertion validation matches against) and the OIDC endpoint trio
// had no API path at all, so a connection could be created and never work.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ssoConnection is the subset of a connection the tests assert on.
type ssoConnection struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// e2eSSOConnection creates a connection and returns it.
func e2eSSOConnection(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token, typ string) ssoConnection {
	t.Helper()

	r := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/sso/connections", ts.URL, projectID),
		map[string]any{"type": typ, "name": "Acme " + typ, "domains": []string{"acme.example.com"}},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)

	var resp struct {
		Connection ssoConnection `json:"connection"`
	}
	e2eDecode(t, r, &resp)

	if resp.Connection.ID == "" {
		t.Fatalf("create connection: no id, body: %s", r.Body)
	}

	return resp.Connection
}

// TestE2ESAMLConnectionIsUsableAfterConfiguration: a SAML connection created and
// configured through the API must render SP metadata naming this deployment.
func TestE2ESAMLConnectionIsUsableAfterConfiguration(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	conn := e2eSSOConnection(t, ctx, ts, projectID, adminToken, "saml")

	// A stand-in IdP publishing its metadata document.
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/samlmetadata+xml")
		fmt.Fprintf(w, samlIDPMetadataTemplate, "https://idp.example.com/entity",
			samlTestCertificateB64, "https://idp.example.com/sso")
	}))
	defer idp.Close()

	// The IdP half is all an administrator should have to supply.
	patch := e2eReq(t, ctx, http.MethodPatch,
		fmt.Sprintf("%s/v1/projects/%s/admin/sso/connections/%s", ts.URL, projectID, conn.ID),
		map[string]any{"saml_metadata_url": idp.URL + "/metadata"},
		e2eBearer(adminToken))
	e2eWantStatus(t, patch, http.StatusOK)

	// The local half is derived, so the metadata document is complete without
	// anybody having typed an entity id or an ACS URL.
	meta := e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/sso/saml/%s/metadata", ts.URL, conn.ID), nil, nil)
	e2eWantStatus(t, meta, http.StatusOK)

	doc := string(meta.Body)

	wantACS := fmt.Sprintf("%s/v1/sso/saml/%s/acs", testDB.PublicURL, conn.ID)
	if !strings.Contains(doc, wantACS) {
		t.Errorf("SP metadata does not advertise the ACS URL %q:\n%s", wantACS, doc)
	}

	wantEntity := fmt.Sprintf("%s/v1/sso/saml/%s/metadata", testDB.PublicURL, conn.ID)
	if !strings.Contains(doc, wantEntity) {
		t.Errorf("SP metadata does not carry the entity id %q:\n%s", wantEntity, doc)
	}

	// An explicit override still wins — a customer whose IdP was configured
	// against an older entity id must not be forced to re-register.
	const custom = "https://sp.acme.example.com/saml"

	over := e2eReq(t, ctx, http.MethodPatch,
		fmt.Sprintf("%s/v1/projects/%s/admin/sso/connections/%s", ts.URL, projectID, conn.ID),
		map[string]any{"saml_entity_id": custom}, e2eBearer(adminToken))
	e2eWantStatus(t, over, http.StatusOK)

	meta = e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/sso/saml/%s/metadata", ts.URL, conn.ID), nil, nil)
	e2eWantStatus(t, meta, http.StatusOK)

	if !strings.Contains(string(meta.Body), custom) {
		t.Errorf("explicit entity id %q was not honoured:\n%s", custom, meta.Body)
	}
}

// TestE2EOIDCConnectionDiscoversItsEndpoints: setting an issuer must be enough.
// Copying three URLs by hand is three chances to paste the wrong one, and a
// wrong jwks_uri fails much later as an unverifiable id_token.
func TestE2EOIDCConnectionDiscoversItsEndpoints(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	// A stand-in upstream provider publishing a discovery document.
	var discoveryHits int

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)

			return
		}

		discoveryHits++

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 "http://" + r.Host,
			"authorization_endpoint": "http://" + r.Host + "/authorize",
			"token_endpoint":         "http://" + r.Host + "/token",
			"jwks_uri":               "http://" + r.Host + "/jwks",
		})
	}))
	defer idp.Close()

	conn := e2eSSOConnection(t, ctx, ts, projectID, adminToken, "oidc")

	patch := e2eReq(t, ctx, http.MethodPatch,
		fmt.Sprintf("%s/v1/projects/%s/admin/sso/connections/%s", ts.URL, projectID, conn.ID),
		map[string]any{
			"oidc_issuer":        idp.URL,
			"oidc_client_id":     "acme-client",
			"oidc_client_secret": "acme-secret",
		}, e2eBearer(adminToken))
	e2eWantStatus(t, patch, http.StatusOK)

	if discoveryHits == 0 {
		t.Fatalf("the discovery document was never fetched")
	}

	// Proof the endpoints landed: starting the flow redirects to the discovered
	// authorization endpoint rather than failing on an unconfigured provider.
	b := newBrowser(t, ts)

	status, body := b.do(t, ctx, http.MethodGet,
		fmt.Sprintf("/v1/sso/oidc/%s/start?redirect_to=%s", conn.ID, "https://app.example.com/cb"), nil, nil)
	if status != http.StatusFound {
		t.Fatalf("sso start: status %d, body %s", status, body)
	}

	if !strings.HasPrefix(b.lastLocation, idp.URL+"/authorize") {
		t.Fatalf("sso start went to %q, want the discovered authorization endpoint", b.lastLocation)
	}

	// The redirect_uri we hand the provider has to be our own callback, derived
	// from the deployment's public URL — a provider rejects anything else.
	want := fmt.Sprintf("%s/v1/sso/oidc/%s/callback", testDB.PublicURL, conn.ID)
	if !strings.Contains(b.lastLocation, "redirect_uri="+url.QueryEscape(want)) {
		t.Errorf("authorization request does not carry redirect_uri %q: %s", want, b.lastLocation)
	}
}

// TestE2EOIDCConnectionRefusesABadIssuer: a typo has to fail while the person
// who made it is still looking.
func TestE2EOIDCConnectionRefusesABadIssuer(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	conn := e2eSSOConnection(t, ctx, ts, projectID, adminToken, "oidc")

	r := e2eReq(t, ctx, http.MethodPatch,
		fmt.Sprintf("%s/v1/projects/%s/admin/sso/connections/%s", ts.URL, projectID, conn.ID),
		map[string]any{"oidc_issuer": "https://nothing-here.invalid"}, e2eBearer(adminToken))
	if r.Status < 400 {
		t.Fatalf("status = %d, want a 4xx/5xx — an unreachable issuer was accepted", r.Status)
	}
}

// samlIDPMetadataTemplate is a minimal but valid IdP metadata document: an
// entity id, a redirect SSO endpoint, and a signing certificate.
const samlIDPMetadataTemplate = `<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data><X509Certificate>%s</X509Certificate></X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s"/>
  </IDPSSODescriptor>
</EntityDescriptor>`

// samlTestCertificateB64 is a throwaway self-signed certificate, DER + base64,
// standing in for an IdP's signing key in the metadata fixture.
const samlTestCertificateB64 = "MIIDFTCCAf2gAwIBAgIUQzNbIGoFK4VFkmVORSnJ5eMMDrQwDQYJKoZIhvcNAQELBQAwGjEYMBYGA1UEAwwPaWRwLmV4YW1wbGUuY29tMB4XDTI2MDgyNTIzMzIxNloXDTM2MDgyMjIzMzIxNlowGjEYMBYGA1UEAwwPaWRwLmV4YW1wbGUuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqOBSBYpvKoryCXQRyBac8IDv//xWeEwL9U5nk6+/UOszwWphmc7N3M4lgrZw4MuMGh765KeDCqrdFwppdfHVs2HB53c+PxP71PNF69hT2690kA8iECGW8DohJuTtYnPLeSBigPV2mTcgfowc/vYeUP6WGe+tAOpZxuM9XCB1TdcnWz9PMju3B5nZILUAwIMdKF1KcYQhtJDpYUH9mlhZym3X41Ew31VSJP0jxQY/FLQsYxXu7zyt7vEB82mdW/IMNLBPU/2kz3gi8cOQwY0ZHkohQkTuKQm7RfHJ8BJAbHiqZtKLCHmdcdFhh+/6+CoHRWDQUekVa/who9TviWRPWQIDAQABo1MwUTAdBgNVHQ4EFgQUbCwGB/wfMG/kBcAAmGuYMb2LqjIwHwYDVR0jBBgwFoAUbCwGB/wfMG/kBcAAmGuYMb2LqjIwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAPNX2OKeYKb98Q1sXRnqdcqzuh1a3MJeZSc25riRPLk2gHbltFj43l6/FuB7O98mUnUhj+iLk4V1LgYNAJKAfnLfUg+WEkEQ85ckXJFkRMj5agxioBFm169RIFWyw63MsfavbudNkj39atZ4zW+2l+HtXaGh30MYffhENFynEmhr7E1JZRlf+mXXefdMM0j0tzcSENSDBSTAUzrzs5Dl8P83YpGMHFikhRuZH25j1Vxsz7uJ5JJZOKOKof0YyTNsbNzcTK61UNfsee6I7aeekRzdKDrzhhhuUZ4QF3oevQ7VLtNnecCOKfSTKec1FUsyP2/Zz4J123phNgEd+TxhSZQ=="
