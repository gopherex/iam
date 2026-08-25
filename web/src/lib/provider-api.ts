/**
 * Error mapping for the hosted OIDC provider pages.
 *
 * The API calls themselves go through the SDK's `IamOidc` namespace (see
 * createIamOidc), which owns the cookie-mode CSRF handshake — the pages must not
 * carry a second implementation of it.
 */

/**
 * providerErrorKey maps an API error onto the message key the pages render. The
 * codes are the stable machine codes from the error envelope; branching on them
 * (never on the localized message) is the documented contract.
 */
export function providerErrorKey(code: string | undefined): string {
  switch (code) {
    case 'flow_expired':
    case 'flow_not_found':
    case 'challenge_expired':
      return 'provider.error.expired';
    case 'not_found':
      return 'provider.error.notFound';
    case 'forbidden':
    case 'access_denied':
      return 'provider.error.denied';
    default:
      return 'provider.error.generic';
  }
}
