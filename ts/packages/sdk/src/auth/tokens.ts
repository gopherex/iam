import type { Client } from '@hey-api/client-fetch';
import {
  postV1TokensIntrospect,
  postV1TokensRevoke,
  postV1TokensVerify,
  getV1TokensCurrent,
  type PostV1TokensIntrospectResponse,
  type PostV1TokensVerifyResponse,
  type GetV1TokensCurrentResponse,
} from '../gen';
import { IamAuthError } from './types';

function tokensError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('request failed', 'request_failed', status);
}

/** Token-inspection namespace — suitable for resource servers and admin tooling. */
export class IamTokens {
  constructor(
    private readonly _client: Client,
    private readonly _headers: () => { 'X-Client-Id': string },
  ) {}

  /**
   * Live token introspection (RFC 7662 style).
   * Returns `{ active, … }` for the given opaque or JWT token.
   */
  async introspect(token: string): Promise<{ data: PostV1TokensIntrospectResponse | null; error: IamAuthError | null }> {
    const r = await postV1TokensIntrospect({
      client: this._client,
      headers: this._headers(),
      body: { token },
    });
    if (r.error) return { data: null, error: tokensError(r) };
    return { data: r.data ?? null, error: null };
  }

  /**
   * Revoke a token or session.
   * Pass `token` for an access/refresh token, or `sessionId` to target a specific session.
   */
  async revoke(params: { token?: string; sessionId?: string; reason?: string }): Promise<{ error: IamAuthError | null }> {
    const r = await postV1TokensRevoke({
      client: this._client,
      headers: this._headers(),
      body: {
        token: params.token,
        session_id: params.sessionId,
        reason: params.reason,
      },
    });
    return { error: r.error ? tokensError(r) : null };
  }

  /**
   * Verify a token server-side (signature + expiry + audience check).
   * Returns `{ valid, claims, error }`.
   */
  async verify(token: string, audience?: string): Promise<{ data: PostV1TokensVerifyResponse | null; error: IamAuthError | null }> {
    const r = await postV1TokensVerify({
      client: this._client,
      headers: this._headers(),
      body: { token, audience },
    });
    if (r.error) return { data: null, error: tokensError(r) };
    return { data: r.data ?? null, error: null };
  }

  /**
   * Return the claims for the currently authenticated bearer token.
   */
  async getCurrent(): Promise<{ data: GetV1TokensCurrentResponse | null; error: IamAuthError | null }> {
    const r = await getV1TokensCurrent({
      client: this._client,
      headers: this._headers(),
    });
    if (r.error) return { data: null, error: tokensError(r) };
    return { data: r.data ?? null, error: null };
  }
}
