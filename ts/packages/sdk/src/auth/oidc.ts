import type { Client } from '@hey-api/client-fetch';
import {
  getV1Device,
  postV1DeviceApprove,
  postV1DeviceDeny,
  getV1OauthInteractionByInteractionId,
  postV1OauthInteractionByInteractionIdLogin,
  postV1OauthInteractionByInteractionIdConsent,
  postV1OauthInteractionByInteractionIdReject,
  getV1OauthGrants,
  deleteV1OauthGrantsByGrantId,
  type GetV1DeviceResponse,
  type GetV1OauthInteractionByInteractionIdResponse,
  type PostV1OauthInteractionByInteractionIdLoginResponse,
  type PostV1OauthInteractionByInteractionIdConsentResponse,
  type PostV1OauthInteractionByInteractionIdRejectResponse,
  type GetV1OauthGrantsResponse,
} from '../gen';
import { client as sharedClient } from '../gen/client.gen';
import { IamAuthError } from './types';
import { createCsrfProvider, csrfFailed, type CsrfProvider } from './csrf';

function oidcError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('request failed', 'request_failed', status);
}

/**
 * OIDC-provider end-user namespace.
 * Use this when building your own consent, device-authorization, or interaction UI.
 */
export class IamOidc {
  private readonly _csrf: CsrfProvider;

  constructor(
    private readonly _client: Client,
    private readonly _headers: () => { 'X-Client-Id': string },
    csrf?: CsrfProvider,
  ) {
    this._csrf = csrf ?? createCsrfProvider(_client, () => this._headers() as Record<string, string>);
  }

  /**
   * Headers for a state-changing call: the tenant plus a CSRF token.
   *
   * The token is attached unconditionally. A cookie-mode browser needs it, and a
   * bearer caller is never challenged, so carrying it costs nothing and removes
   * the one thing every consumer would otherwise have to reimplement.
   */
  private async _writeHeaders(): Promise<Record<string, string>> {
    const headers: Record<string, string> = { ...(this._headers() as Record<string, string>) };
    const token = await this._csrf.token();
    if (token) headers['X-Csrf-Token'] = token;
    return headers;
  }

  /**
   * Run a state-changing call, refreshing the CSRF token once if the server
   * rejects it. A rotated token must not surface to the user as a failed login.
   */
  private async _write<T>(call: (headers: Record<string, string>) => Promise<T>): Promise<T> {
    const first = await call(await this._writeHeaders());
    if (!csrfFailed(first as { error?: unknown })) return first;

    this._csrf.invalidate();
    return call(await this._writeHeaders());
  }

  /**
   * Retrieve the device-authorization page data (client info + requested scopes).
   * Pass `userCode` to pre-fill the code; without it, the response still contains
   * the waiting state for a polling UI.
   */
  async getDevice(userCode: string): Promise<{ data: GetV1DeviceResponse | null; error: IamAuthError | null }> {
    const r = await getV1Device({
      client: this._client,
      headers: this._headers(),
      query: { user_code: userCode },
    });
    if (r.error) return { data: null, error: oidcError(r) };
    return { data: r.data ?? null, error: null };
  }

  /** Approve a device-authorization request identified by `userCode`. */
  async approveDevice(userCode: string): Promise<{ error: IamAuthError | null }> {
    const r = await this._write((headers) =>
      postV1DeviceApprove({
        client: this._client,
        headers: headers as never,
        body: { user_code: userCode },
      }),
    );
    return { error: r.error ? oidcError(r) : null };
  }

  /** Deny a device-authorization request identified by `userCode`. */
  async denyDevice(userCode: string): Promise<{ error: IamAuthError | null }> {
    const r = await this._write((headers) =>
      postV1DeviceDeny({
        client: this._client,
        headers: headers as never,
        body: { user_code: userCode },
      }),
    );
    return { error: r.error ? oidcError(r) : null };
  }

  /** Fetch the context (stage, client, requested scopes) for an interaction. */
  async getInteraction(interactionId: string): Promise<{ data: GetV1OauthInteractionByInteractionIdResponse | null; error: IamAuthError | null }> {
    const r = await getV1OauthInteractionByInteractionId({
      client: this._client,
      headers: this._headers(),
      path: { interaction_id: interactionId },
    });
    if (r.error) return { data: null, error: oidcError(r) };
    return { data: r.data ?? null, error: null };
  }

  /**
   * Attach the currently authenticated user to the interaction (login step).
   * Returns the redirect target URL.
   */
  async loginInteraction(
    interactionId: string,
    payload?: { flowToken?: string },
  ): Promise<{ data: PostV1OauthInteractionByInteractionIdLoginResponse | null; error: IamAuthError | null }> {
    const r = await this._write((headers) =>
      postV1OauthInteractionByInteractionIdLogin({
        client: this._client,
        headers: headers as never,
        path: { interaction_id: interactionId },
        body: { flow_token: payload?.flowToken },
      }),
    );
    if (r.error) return { data: null, error: oidcError(r) };
    return { data: r.data ?? null, error: null };
  }

  /**
   * Record consent for the interaction (consent step).
   * Returns the redirect target URL.
   */
  async consentInteraction(
    interactionId: string,
    payload?: { grantedScopes?: Array<string>; remember?: boolean },
  ): Promise<{ data: PostV1OauthInteractionByInteractionIdConsentResponse | null; error: IamAuthError | null }> {
    const r = await this._write((headers) =>
      postV1OauthInteractionByInteractionIdConsent({
        client: this._client,
        headers: headers as never,
        path: { interaction_id: interactionId },
        body: {
          granted_scopes: payload?.grantedScopes,
          remember: payload?.remember,
        },
      }),
    );
    if (r.error) return { data: null, error: oidcError(r) };
    return { data: r.data ?? null, error: null };
  }

  /**
   * Reject/cancel an interaction.
   * Returns the redirect target URL with an error response.
   */
  async rejectInteraction(
    interactionId: string,
    payload?: { error?: string; errorDescription?: string },
  ): Promise<{ data: PostV1OauthInteractionByInteractionIdRejectResponse | null; error: IamAuthError | null }> {
    const r = await this._write((headers) =>
      postV1OauthInteractionByInteractionIdReject({
        client: this._client,
        headers: headers as never,
        path: { interaction_id: interactionId },
        body: {
          error: payload?.error,
          error_description: payload?.errorDescription,
        },
      }),
    );
    if (r.error) return { data: null, error: oidcError(r) };
    return { data: r.data ?? null, error: null };
  }

  /** List the current user's authorized OAuth grants (consented applications). */
  async listGrants(params?: { cursor?: string; limit?: number }): Promise<{ data: GetV1OauthGrantsResponse | null; error: IamAuthError | null }> {
    const r = await getV1OauthGrants({
      client: this._client,
      headers: this._headers(),
      query: params,
    });
    if (r.error) return { data: null, error: oidcError(r) };
    return { data: r.data ?? null, error: null };
  }

  /** Revoke an OAuth grant (de-authorize an application). */
  async revokeGrant(grantId: string): Promise<{ error: IamAuthError | null }> {
    const r = await this._write((headers) =>
      deleteV1OauthGrantsByGrantId({
        client: this._client,
        headers: headers as never,
        path: { grant_id: grantId },
      }),
    );
    return { error: r.error ? oidcError(r) : null };
  }
}

/**
 * createIamOidc builds the OIDC end-user namespace on its own, for a page that
 * has no session of its own to manage.
 *
 * The hosted login / consent / device screens are exactly that case: they
 * authenticate the user through the flow engine and then act on the interaction
 * with the resulting cookie session. They need this namespace and nothing else
 * from the client, and they must not have to hand-roll the CSRF handshake it
 * already performs.
 */
export function createIamOidc(options: {
  /** IAM base URL. Empty (the default) means same-origin. */
  baseUrl?: string;
  /** Project id, sent as X-Client-Id. */
  clientId: string;
  /** Project environment, sent as X-Environment when set. */
  environment?: string;
  /** Fetch client to use; defaults to the SDK's shared one. */
  client?: Client;
}): IamOidc {
  const http = options.client ?? sharedClient;
  if (options.baseUrl !== undefined) {
    http.setConfig({ baseUrl: options.baseUrl });
  }

  const headers = () => {
    const out: Record<string, string> = { 'X-Client-Id': options.clientId };
    if (options.environment) out['X-Environment'] = options.environment;
    return out as { 'X-Client-Id': string };
  };

  return new IamOidc(http, headers);
}
