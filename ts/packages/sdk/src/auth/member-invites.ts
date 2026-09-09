/**
 * IamMemberInvites — user-scoped invitations ("invite a friend").
 *
 * Unlike IamInvitesAdmin this surface rides the signed-in user's session:
 * invitations are email-bound, counted against the caller's invite cap
 * (project member_invites default or a per-user admin override), and only
 * the caller's own invitations are listable/revocable. The invitee gets the
 * invitation email immediately; create() also returns the raw invite_token
 * once so the caller can share the link themselves.
 */

import { type Client } from '@hey-api/client-fetch';
import {
  getV1AuthInvites,
  postV1AuthInvites,
  postV1AuthInvitesByInviteIdRevoke,
  type Invite,
  type InviteCreated,
  type InviteQuota,
} from '../gen';
import { IamAuthError } from './types';

function inviteError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('invite request failed', 'invite_request_failed', status);
}

export interface MemberInviteCreated {
  invite: InviteCreated;
  quota: InviteQuota;
}

export interface MemberInviteList {
  invites: Invite[];
  quota: InviteQuota;
}

export class IamMemberInvites {
  private readonly client: Client;
  private readonly headers: () => Record<string, string>;

  /** @internal — constructed by createIamClient. */
  constructor(client: Client, headers: () => Record<string, string>) {
    this.client = client;
    this.headers = headers;
  }

  /**
   * Invite a person by email. The invitee is emailed immediately; the raw
   * invite_token (and its shareable link) is returned exactly once. Fails
   * with `forbidden` when member invitations are disabled and
   * `invite_quota_exceeded` when the caller's cap is reached.
   */
  async create(params: { email: string; redirectTo?: string }): Promise<{ data: MemberInviteCreated | null; error: IamAuthError | null }> {
    const r = await postV1AuthInvites({
      client: this.client,
      headers: this.headers(),
      body: { email: params.email, redirect_to: params.redirectTo },
    });
    if (r.error) return { data: null, error: inviteError(r) };
    const data = r.data ?? {};
    if (!data.invite || !data.quota) {
      return { data: null, error: new IamAuthError('unexpected invite response', 'invite_unexpected_response', r.response?.status) };
    }
    return { data: { invite: data.invite, quota: data.quota }, error: null };
  }

  /** List the caller's own invitations with their remaining quota. */
  async list(): Promise<{ data: MemberInviteList | null; error: IamAuthError | null }> {
    const r = await getV1AuthInvites({ client: this.client, headers: this.headers() });
    if (r.error) return { data: null, error: inviteError(r) };
    return {
      data: { invites: r.data?.invites ?? [], quota: r.data?.quota ?? { cap: 0, used: 0, left: 0 } },
      error: null,
    };
  }

  /** Revoke the caller's own pending invitation (frees its quota slot). */
  async revoke(inviteId: string): Promise<{ error: IamAuthError | null }> {
    const r = await postV1AuthInvitesByInviteIdRevoke({
      client: this.client,
      headers: this.headers(),
      path: { invite_id: inviteId },
    });
    return { error: r.error ? inviteError(r) : null };
  }
}
