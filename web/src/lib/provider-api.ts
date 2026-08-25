/**
 * Cookie-mode API helpers for the hosted OIDC provider pages.
 *
 * These pages authenticate the END USER, not an operator, and they do it the way
 * a browser should: the session lives in HttpOnly cookies the page cannot read,
 * and every state-changing call carries a CSRF token bound to the project. The
 * admin console's bearer-token store (stores/auth) is a different credential
 * model entirely and is deliberately not used here.
 *
 * The generated operations from the SDK are called directly — this module adds
 * only the CSRF handshake, not a second API layer.
 */

import { getV1Csrf } from '@gopherex/iam-sdk';
import { call } from '@/lib/sdk';

/**
 * csrfHeaders returns the headers a cookie-mode POST needs.
 *
 * The synchronizer token is issued per project (X-Client-Id) and verified
 * against it, so both travel together. Safe methods need neither.
 */
export async function csrfHeaders(projectId: string, environment?: string): Promise<Record<string, string>> {
  const headers: Record<string, string> = { 'X-Client-Id': projectId };
  if (environment) headers['X-Environment'] = environment;

  const res = await call(
    getV1Csrf({ headers: { 'X-Client-Id': projectId, 'X-Environment': environment } }),
  );

  const token = (res as { csrf_token?: string } | undefined)?.csrf_token;
  if (token) headers['X-Csrf-Token'] = token;

  return headers;
}

/**
 * providerError maps an API error onto the message key the pages render. The
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
