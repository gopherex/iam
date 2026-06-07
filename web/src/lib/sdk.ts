import { client } from '@gopherex/iam-sdk';
import { authSnapshot } from '@/stores/auth';

// Configure the shared SDK fetch client for the admin panel. Same-origin in
// production (the Go server serves this SPA); the Vite dev server proxies /v1
// and /mgmt to the API.
client.setConfig({ baseUrl: '' });

// Attach the right bearer per call: the master key on operator (/mgmt/*) calls,
// the project admin token on everything else (falling back to the master key).
client.interceptors.request.use((request: Request) => {
  if (!request.headers.has('Authorization')) {
    const { masterKey, adminToken } = authSnapshot();
    const token = request.url.includes('/mgmt/') ? masterKey : adminToken ?? masterKey;
    if (token) request.headers.set('Authorization', `Bearer ${token}`);
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
