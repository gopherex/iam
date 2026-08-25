/**
 * IamClientRegistration — dynamic client registration (RFC 7591) and the client
 * management it implies (RFC 7592).
 *
 * There are two credentials here, and they are not interchangeable:
 *
 *  - the *initial access token* registers new clients. IAM is multi-tenant, so
 *    registration is never open: this is a project-admin token, and it is also
 *    what decides which project the new client lands in.
 *  - the *registration access token*, returned once at registration, reads,
 *    updates and deletes exactly one client — the one it was issued for.
 *
 * A client created through the admin API is managed through the admin API; only
 * a dynamically registered client is manageable here.
 */

import { createClient, createConfig, type Client } from '@hey-api/client-fetch';
import {
  postOauth2Register,
  getOauth2RegisterByClientId,
  putOauth2RegisterByClientId,
  deleteOauth2RegisterByClientId,
  type ClientRegistration,
  type ClientRegistrationResponse,
  type ClientOptions as GeneratedClientOptions,
} from '../gen';
import { IamAuthError } from './types';

export interface IamClientRegistrationOptions {
  /** API base URL (same as IamClientOptions.baseUrl). */
  baseUrl: string;
  /**
   * The initial access token authorizing `register`. A project-admin token;
   * omit it when the instance is only used for RFC 7592 self-management.
   */
  initialAccessToken?: string;
  /** Environment the client belongs to (X-Environment); defaults to live. */
  environment?: string;
}

function registrationError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('registration request failed', 'registration_request_failed', status);
}

export class IamClientRegistration {
  private readonly client: Client;
  private readonly initialAccessToken?: string;
  private readonly environment?: string;

  constructor(opts: IamClientRegistrationOptions) {
    this.client = createClient(createConfig<GeneratedClientOptions>({ baseUrl: opts.baseUrl }));
    this.initialAccessToken = opts.initialAccessToken;
    this.environment = opts.environment;
  }

  private hdr(token: string | undefined): Record<string, string> {
    const h: Record<string, string> = {};
    if (token) h.Authorization = `Bearer ${token}`;
    if (this.environment) h['X-Environment'] = this.environment;
    return h;
  }

  /**
   * Register a client. The response carries `client_secret` (confidential
   * clients only) and `registration_access_token` exactly once — persist them
   * now or they are unrecoverable.
   */
  async register(
    metadata: ClientRegistration,
  ): Promise<{ data: ClientRegistrationResponse | null; error: IamAuthError | null }> {
    const r = await postOauth2Register({
      client: this.client,
      headers: this.hdr(this.initialAccessToken),
      body: metadata,
    });
    if (r.error) return { data: null, error: registrationError(r) };
    return { data: r.data ?? null, error: null };
  }

  /** Read a registered client's metadata with its registration access token. */
  async read(
    clientId: string,
    registrationAccessToken: string,
  ): Promise<{ data: ClientRegistrationResponse | null; error: IamAuthError | null }> {
    const r = await getOauth2RegisterByClientId({
      client: this.client,
      headers: this.hdr(registrationAccessToken),
      path: { client_id: clientId },
    });
    if (r.error) return { data: null, error: registrationError(r) };
    return { data: r.data ?? null, error: null };
  }

  /**
   * Replace a registered client's metadata. RFC 7592 makes this a replacement,
   * not a patch: anything left out of `metadata` is cleared. Read first, edit
   * the result, send it back.
   */
  async update(
    clientId: string,
    registrationAccessToken: string,
    metadata: ClientRegistration,
  ): Promise<{ data: ClientRegistrationResponse | null; error: IamAuthError | null }> {
    const r = await putOauth2RegisterByClientId({
      client: this.client,
      headers: this.hdr(registrationAccessToken),
      path: { client_id: clientId },
      body: metadata,
    });
    if (r.error) return { data: null, error: registrationError(r) };
    return { data: r.data ?? null, error: null };
  }

  /** Delete a registered client. */
  async delete(
    clientId: string,
    registrationAccessToken: string,
  ): Promise<{ error: IamAuthError | null }> {
    const r = await deleteOauth2RegisterByClientId({
      client: this.client,
      headers: this.hdr(registrationAccessToken),
      path: { client_id: clientId },
    });
    return { error: r.error ? registrationError(r) : null };
  }
}

/** Construct a dynamic-client-registration client. */
export function createIamClientRegistration(
  opts: IamClientRegistrationOptions,
): IamClientRegistration {
  return new IamClientRegistration(opts);
}
