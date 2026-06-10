/**
 * IamInvitesAdmin — invitation management (create / list / revoke).
 *
 * This is an ADMIN surface: it authenticates with a project-admin token, NOT an
 * end-user session, so it is deliberately separate from the user-facing
 * createIamClient (whose bearer is the signed-in user's access token). Use it
 * server-side (or in the admin panel) where you hold an admin token.
 *
 * The redemption side (an invited user accepting an invite to sign up) lives in
 * the user client: `iam.flow.redeemInvite(token, …)` / `flow.start({ kind:
 * 'signup', inviteToken })`.
 */

import { createClient, createConfig, type Client } from '@hey-api/client-fetch';
import {
  postV1ProjectsByProjectIdAdminInvites,
  getV1ProjectsByProjectIdAdminInvites,
  postV1ProjectsByProjectIdAdminInvitesByInviteIdRevoke,
  type Invite,
  type InviteCreated,
  type ClientOptions as GeneratedClientOptions,
} from '../gen';
import { IamAuthError } from './types';

export interface IamInvitesAdminOptions {
  /** API base URL (same as IamClientOptions.baseUrl). */
  baseUrl: string;
  /** The project id (the {project_id} path segment). */
  projectId: string;
  /** A project-admin token authorizing the admin invite endpoints. */
  adminToken: string;
  /** Environment the invites belong to (X-Environment); defaults to live. */
  environment?: string;
}

function inviteError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('invite request failed', 'invite_request_failed', status);
}

export class IamInvitesAdmin {
  private readonly client: Client;
  private readonly projectId: string;
  private readonly hdr: Record<string, string>;

  constructor(opts: IamInvitesAdminOptions) {
    this.client = createClient(createConfig<GeneratedClientOptions>({ baseUrl: opts.baseUrl }));
    this.projectId = opts.projectId;
    this.hdr = { Authorization: `Bearer ${opts.adminToken}` };
    if (opts.environment) this.hdr['X-Environment'] = opts.environment;
  }

  /**
   * Create an invitation. The raw token is returned exactly once (invite_token);
   * when `email` is set the invite is email-bound and an invite email is sent.
   */
  async create(params?: {
    email?: string;
    expiresAt?: string;
    redirectTo?: string;
  }): Promise<{ data: InviteCreated | null; error: IamAuthError | null }> {
    const r = await postV1ProjectsByProjectIdAdminInvites({
      client: this.client,
      headers: this.hdr,
      path: { project_id: this.projectId },
      body: { email: params?.email, expires_at: params?.expiresAt, redirect_to: params?.redirectTo },
    });
    if (r.error) return { data: null, error: inviteError(r) };
    return { data: r.data ?? null, error: null };
  }

  /** List the project's invitations (for the configured environment). */
  async list(): Promise<{ data: Invite[]; error: IamAuthError | null }> {
    const r = await getV1ProjectsByProjectIdAdminInvites({
      client: this.client,
      headers: this.hdr,
      path: { project_id: this.projectId },
    });
    if (r.error) return { data: [], error: inviteError(r) };
    return { data: r.data?.invites ?? [], error: null };
  }

  /** Revoke a pending invitation by id. */
  async revoke(inviteId: string): Promise<{ error: IamAuthError | null }> {
    const r = await postV1ProjectsByProjectIdAdminInvitesByInviteIdRevoke({
      client: this.client,
      headers: this.hdr,
      path: { project_id: this.projectId, invite_id: inviteId },
    });
    return { error: r.error ? inviteError(r) : null };
  }
}

/** Construct an admin invites client. Requires a project-admin token. */
export function createIamInvitesAdmin(opts: IamInvitesAdminOptions): IamInvitesAdmin {
  return new IamInvitesAdmin(opts);
}
