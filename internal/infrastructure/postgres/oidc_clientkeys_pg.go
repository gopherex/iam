package postgres

// Client-held keys: the public keys a CLIENT signs with, as opposed to the ones
// this service signs with.
//
// Two features need them and neither works without them:
//
//   - private_key_jwt (RFC 7523): the client authenticates at the token endpoint
//     with an assertion it signed, so no shared secret exists to leak;
//   - request objects (RFC 9101): the whole authorization request arrives as a
//     JWT the client signed, so the browser cannot tamper with its parameters.
//
// A client publishes them inline (`jwks`) or at a URL (`jwks_uri`). The URL is
// operator-configured but still fetched through the hardened outbound client —
// it is a request to an address we were told about, exactly like a webhook — and
// cached, because the token endpoint would otherwise fetch on every call.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/gopherex/iam/internal/domain"
)

// clientKeysTTL is how long a fetched JWKS is reused. Short enough that a
// rotated key is picked up without a restart, long enough that the token
// endpoint is not a proxy for the client's key server.
const clientKeysTTL = 5 * time.Minute

// clientKeysMaxBytes bounds a fetched JWKS document.
const clientKeysMaxBytes = 512 << 10

// clientKeysFetchTimeout bounds one JWKS fetch. The token endpoint waits on it,
// so it stays short.
const clientKeysFetchTimeout = 10 * time.Second

var (
	// errNoClientKeys is returned when a client has published no keys at all.
	errNoClientKeys = errors.New("client has published no keys")
	// errClientJWKSFetch is returned when the client's key server refuses us.
	errClientJWKSFetch = errors.New("client jwks fetch failed")
)

// clientKeyCache caches parsed key sets by their source URL.
type clientKeyCache struct {
	mu      sync.RWMutex
	entries map[string]clientKeyEntry
	client  *http.Client
}

type clientKeyEntry struct {
	set       jwk.Set
	expiresAt time.Time
}

func newClientKeyCache() *clientKeyCache {
	return &clientKeyCache{
		entries: map[string]clientKeyEntry{},
		client:  NewOutboundHTTPClient(clientKeysFetchTimeout),
	}
}

// keysFor returns the public keys a client signs with, preferring the inline set
// — it needs no network and cannot be made unavailable by the client's own
// infrastructure.
func (c *clientKeyCache) keysFor(ctx context.Context, app *domain.AppClient) (jwk.Set, error) {
	if app.JWKS != "" {
		set, err := jwk.Parse([]byte(app.JWKS))
		if err != nil {
			return nil, fmt.Errorf("parse client jwks: %w", err)
		}

		return set, nil
	}

	if app.JWKSURI == "" {
		return nil, errNoClientKeys
	}

	return c.fetch(ctx, app.JWKSURI)
}

func (c *clientKeyCache) fetch(ctx context.Context, uri string) (jwk.Set, error) {
	c.mu.RLock()
	entry, ok := c.entries[uri]
	c.mu.RUnlock()

	if ok && entry.expiresAt.After(nowIn(ctx)) {
		return entry.set, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch client jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", errClientJWKSFetch, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, clientKeysMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("read client jwks: %w", err)
	}

	set, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse client jwks: %w", err)
	}

	c.mu.Lock()
	c.entries[uri] = clientKeyEntry{set: set, expiresAt: nowUTC().Add(clientKeysTTL)}
	c.mu.Unlock()

	return set, nil
}
