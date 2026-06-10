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
import { IamAuthError } from './types';

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
  constructor(
    private readonly _client: Client,
    private readonly _headers: () => { 'X-Client-Id': string },
  ) {}

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
    const r = await postV1DeviceApprove({
      client: this._client,
      headers: this._headers(),
      body: { user_code: userCode },
    });
    return { error: r.error ? oidcError(r) : null };
  }

  /** Deny a device-authorization request identified by `userCode`. */
  async denyDevice(userCode: string): Promise<{ error: IamAuthError | null }> {
    const r = await postV1DeviceDeny({
      client: this._client,
      headers: this._headers(),
      body: { user_code: userCode },
    });
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
    const r = await postV1OauthInteractionByInteractionIdLogin({
      client: this._client,
      headers: this._headers(),
      path: { interaction_id: interactionId },
      body: { flow_token: payload?.flowToken },
    });
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
    const r = await postV1OauthInteractionByInteractionIdConsent({
      client: this._client,
      headers: this._headers(),
      path: { interaction_id: interactionId },
      body: {
        granted_scopes: payload?.grantedScopes,
        remember: payload?.remember,
      },
    });
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
    const r = await postV1OauthInteractionByInteractionIdReject({
      client: this._client,
      headers: this._headers(),
      path: { interaction_id: interactionId },
      body: {
        error: payload?.error,
        error_description: payload?.errorDescription,
      },
    });
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
    const r = await deleteV1OauthGrantsByGrantId({
      client: this._client,
      headers: this._headers(),
      path: { grant_id: grantId },
    });
    return { error: r.error ? oidcError(r) : null };
  }
}
