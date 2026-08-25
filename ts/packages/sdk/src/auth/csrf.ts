import type { Client } from '@hey-api/client-fetch';
import { getV1Csrf } from '../gen';

/**
 * CSRF tokens for cookie-mode requests.
 *
 * A browser that authenticates with the HttpOnly session cookie sends its
 * credential ambiently, so IAM challenges every state-changing cookie-mode
 * request for a synchronizer token bound to the project (see /v1/csrf). A caller
 * that uses a bearer token is not challenged — the credential is not ambient —
 * so attaching the header there is simply ignored.
 *
 * The token is cached because it is stable for the project, and invalidated when
 * the server says it is not: a rotated or expired token is retried once rather
 * than surfacing to the caller as a login failure.
 */
export interface CsrfProvider {
  /** Current token, fetching one if needed. Undefined when unobtainable. */
  token(): Promise<string | undefined>;
  /** Drop the cached token so the next call fetches a fresh one. */
  invalidate(): void;
}

export function createCsrfProvider(
  client: Client,
  headers: () => Record<string, string>,
): CsrfProvider {
  let cached: string | undefined;
  let inflight: Promise<string | undefined> | null = null;

  async function fetchToken(): Promise<string | undefined> {
    const r = await getV1Csrf({ client, headers: headers() as never });
    const token = (r.data as { csrf_token?: string } | undefined)?.csrf_token;
    cached = token;
    return token;
  }

  return {
    async token() {
      if (cached) return cached;
      // Collapse concurrent callers onto one request: the consent screen fires
      // several calls in a row and each would otherwise mint its own token.
      inflight ??= fetchToken().finally(() => {
        inflight = null;
      });
      return inflight;
    },
    invalidate() {
      cached = undefined;
    },
  };
}

/** csrfFailed reports whether a result was rejected for a bad CSRF token. */
export function csrfFailed(result: { error?: unknown }): boolean {
  const env = result.error as { error?: { code?: string } } | undefined;
  return env?.error?.code === 'invalid_csrf';
}
