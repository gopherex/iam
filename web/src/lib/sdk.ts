import { client } from '@gopherex/iam-sdk';
import { authSnapshot } from '@/stores/auth';

// Configure the shared SDK fetch client for the admin panel. Same-origin in
// production (the Go server serves this SPA); the Vite dev server proxies /v1
// and /mgmt to the API.
client.setConfig({ baseUrl: '' });

// Attach the right bearer per call: the master key on operator (/mgmt/*) calls,
// the project admin token on everything else (falling back to the master key).
//
// Also attach the selected environment to project-admin data calls. Admin
// endpoints accept an X-Environment header for Stripe-like test/live isolation;
// the global $environment store picks which environment the data views read. We
// only set it when the caller hasn't set one explicitly (e.g. the Configuration
// page passes its own per-tab X-Environment), and only for project-admin routes.
client.interceptors.request.use((request: Request) => {
  if (!request.headers.has('Authorization')) {
    const { masterKey, adminToken } = authSnapshot();
    const token = request.url.includes('/mgmt/') ? masterKey : adminToken ?? masterKey;
    if (token) request.headers.set('Authorization', `Bearer ${token}`);
  }
  if (!request.headers.has('X-Environment') && /\/v1\/projects\/[^/]+\/admin\//.test(request.url)) {
    const { environment } = authSnapshot();
    if (environment) request.headers.set('X-Environment', environment);
  }
  return request;
});

export interface ApiError extends Error {
  code?: string;
  status?: number;
}

/**
 * Unwrap a generated operation's `{ data, error, response }` result: returns the
 * data on success, throws a normalized ApiError otherwise.
 */
export async function call<T>(
  promise: Promise<{ data?: T; error?: unknown; response?: Response }>,
): Promise<T> {
  const r = await promise;
  if (r.error || (r.response && !r.response.ok)) {
    const env = r.error as { error?: { code?: string; message?: string } } | undefined;
    const err: ApiError = new Error(
      env?.error?.message ?? (r.error as Error)?.message ?? `Request failed (${r.response?.status ?? '?'})`,
    );
    err.code = env?.error?.code;
    err.status = r.response?.status;
    throw err;
  }
  return r.data as T;
}
