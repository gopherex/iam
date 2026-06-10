/**
 * IamAccount — authenticated self-service account operations.
 * All methods share the same bearer-carrying Client and headers()
 * helper as IamAuth, and return the same { data, error: IamAuthError | null }
 * shape. Constructed by createIamClient; use via iam.account.
 */

import type { Client } from '@hey-api/client-fetch';
import {
  getV1UsersMe,
  patchV1UsersMe,
  deleteV1UsersMe,
  getV1AccountCapabilities,
  getV1UsersMeActivity,
  postV1UsersMeExport,
  getV1UsersMeExportByJobId,
  getV1UsersMeConsents,
  postV1UsersMeConsents,
  getV1AuthIdentities,
  deleteV1AuthIdentitiesByIdentityId,
  postV1AuthIdentitiesMergeStart,
  postV1AuthIdentitiesMergeConfirm,
  getV1Sessions,
  getV1SessionsCurrent,
  patchV1SessionsBySessionId,
  postV1SessionsBySessionIdTrust,
  deleteV1SessionsBySessionId,
  deleteV1Sessions,
  type User,
  type Identity,
  type ActivityEvent,
  type ConsentAcceptance,
  type ConsentDocRef,
  type Timestamp,
  type Session as ApiSession,
  type PageMeta,
} from '../gen';
import { IamAuthError } from './types';

// ---------------------------------------------------------------------------
// Inlined response-shape types (keep public so callers can annotate)
// ---------------------------------------------------------------------------

export interface AccountProfile {
  user: User;
}

export interface AccountCapabilities {
  capabilities: Record<string, unknown>;
}

export interface AccountActivity {
  data: Array<ActivityEvent>;
  next_cursor?: string | null;
  has_more?: boolean;
}

export interface ExportJob {
  jobId: string;
}

export interface ExportStatus {
  status: string;
  downloadUrl: string | null;
}

export interface ConsentRecord extends ConsentDocRef {
  accepted_at?: Timestamp;
  locale?: string;
}

export interface IdentityList {
  data: Array<Identity>;
}

export interface MergeStartResult {
  challengeId: string;
}

export interface MergeConfirmResult {
  user: User | undefined;
  identities: Array<Identity>;
}

export interface SessionList {
  data: Array<ApiSession>;
}

export interface CurrentSessionResult {
  session: ApiSession | undefined;
}

export interface SessionResult {
  session: ApiSession | undefined;
}

export interface RevokeAllResult {
  revokedCount: number;
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

function accountError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('request failed', 'request_failed', status);
}

// ---------------------------------------------------------------------------
// Class
// ---------------------------------------------------------------------------

export class IamAccount {
  constructor(
    private readonly _client: Client,
    private readonly _headers: () => { 'X-Client-Id': string },
  ) {}

  // ---- profile -----------------------------------------------------------

  async getProfile(): Promise<{ data: AccountProfile | null; error: IamAuthError | null }> {
    const r = await getV1UsersMe({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { user?: User } | undefined;
    if (!data?.user) return { data: null, error: new IamAuthError('unexpected response', 'invalid_response', r.response?.status) };
    return { data: { user: data.user }, error: null };
  }

  async updateProfile(patch: {
    name?: string;
    avatarUrl?: string | null;
    locale?: string;
    metadata?: Record<string, unknown>;
  }): Promise<{ data: AccountProfile | null; error: IamAuthError | null }> {
    const r = await patchV1UsersMe({
      client: this._client,
      headers: this._headers(),
      body: {
        name: patch.name,
        avatar_url: patch.avatarUrl,
        locale: patch.locale,
        metadata: patch.metadata,
      },
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { user?: User } | undefined;
    if (!data?.user) return { data: null, error: new IamAuthError('unexpected response', 'invalid_response', r.response?.status) };
    return { data: { user: data.user }, error: null };
  }

  async deleteAccount(params?: {
    password?: string;
    reason?: string;
  }): Promise<{ error: IamAuthError | null }> {
    const r = await deleteV1UsersMe({
      client: this._client,
      headers: this._headers(),
      body: params ? { password: params.password, reason: params.reason } : undefined,
    });
    return { error: r.error ? accountError(r) : null };
  }

  // ---- capabilities -------------------------------------------------------

  async getCapabilities(): Promise<{ data: AccountCapabilities | null; error: IamAuthError | null }> {
    const r = await getV1AccountCapabilities({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { capabilities?: Record<string, unknown> } | undefined;
    return { data: { capabilities: data?.capabilities ?? {} }, error: null };
  }

  // ---- activity -----------------------------------------------------------

  async getActivity(params?: {
    type?: string;
    cursor?: string;
    limit?: number;
  }): Promise<{ data: AccountActivity | null; error: IamAuthError | null }> {
    const r = await getV1UsersMeActivity({
      client: this._client,
      headers: this._headers(),
      query: params,
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as ({ data?: Array<ActivityEvent> } & PageMeta) | undefined;
    return {
      data: {
        data: data?.data ?? [],
        next_cursor: data?.next_cursor,
        has_more: data?.has_more,
      },
      error: null,
    };
  }

  // ---- data export --------------------------------------------------------

  async startExport(): Promise<{ data: ExportJob | null; error: IamAuthError | null }> {
    const r = await postV1UsersMeExport({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { job_id?: string } | undefined;
    if (!data?.job_id) return { data: null, error: new IamAuthError('missing job_id', 'invalid_response', r.response?.status) };
    return { data: { jobId: data.job_id }, error: null };
  }

  async getExport(jobId: string): Promise<{ data: ExportStatus | null; error: IamAuthError | null }> {
    const r = await getV1UsersMeExportByJobId({
      client: this._client,
      headers: this._headers(),
      path: { job_id: jobId },
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { status?: string; download_url?: string | null } | undefined;
    return { data: { status: data?.status ?? '', downloadUrl: data?.download_url ?? null }, error: null };
  }

  // ---- consents -----------------------------------------------------------

  async getConsents(): Promise<{ data: { consents: Array<ConsentRecord> } | null; error: IamAuthError | null }> {
    const r = await getV1UsersMeConsents({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { consents?: Array<ConsentRecord> } | undefined;
    return { data: { consents: data?.consents ?? [] }, error: null };
  }

  async acceptConsents(accept: Array<ConsentAcceptance>): Promise<{ error: IamAuthError | null }> {
    const r = await postV1UsersMeConsents({
      client: this._client,
      headers: this._headers(),
      body: { accept },
    });
    return { error: r.error ? accountError(r) : null };
  }

  // ---- linked identities --------------------------------------------------

  async listIdentities(): Promise<{ data: IdentityList | null; error: IamAuthError | null }> {
    const r = await getV1AuthIdentities({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { data?: Array<Identity> } | undefined;
    return { data: { data: data?.data ?? [] }, error: null };
  }

  async unlinkIdentity(id: string): Promise<{ error: IamAuthError | null }> {
    const r = await deleteV1AuthIdentitiesByIdentityId({
      client: this._client,
      headers: this._headers(),
      path: { identity_id: id },
    });
    return { error: r.error ? accountError(r) : null };
  }

  async startIdentityMerge(
    targetIdentifier: string,
  ): Promise<{ data: MergeStartResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthIdentitiesMergeStart({
      client: this._client,
      headers: this._headers(),
      body: { target_identifier: targetIdentifier },
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { challenge_id?: string } | undefined;
    if (!data?.challenge_id) return { data: null, error: new IamAuthError('missing challenge_id', 'invalid_response', r.response?.status) };
    return { data: { challengeId: data.challenge_id }, error: null };
  }

  async confirmIdentityMerge(params: {
    challengeId: string;
    code: string;
  }): Promise<{ data: MergeConfirmResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthIdentitiesMergeConfirm({
      client: this._client,
      headers: this._headers(),
      body: { challenge_id: params.challengeId, code: params.code },
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { user?: User; identities?: Array<Identity> } | undefined;
    return { data: { user: data?.user, identities: data?.identities ?? [] }, error: null };
  }

  // ---- sessions -----------------------------------------------------------

  async listSessions(): Promise<{ data: SessionList | null; error: IamAuthError | null }> {
    const r = await getV1Sessions({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { data?: Array<ApiSession> } | undefined;
    return { data: { data: data?.data ?? [] }, error: null };
  }

  async getCurrentSession(): Promise<{ data: CurrentSessionResult | null; error: IamAuthError | null }> {
    const r = await getV1SessionsCurrent({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { session?: ApiSession } | undefined;
    return { data: { session: data?.session }, error: null };
  }

  async renameSession(
    id: string,
    deviceName: string,
  ): Promise<{ data: SessionResult | null; error: IamAuthError | null }> {
    const r = await patchV1SessionsBySessionId({
      client: this._client,
      headers: this._headers(),
      path: { session_id: id },
      body: { device_name: deviceName },
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { session?: ApiSession } | undefined;
    return { data: { session: data?.session }, error: null };
  }

  async trustSession(
    id: string,
    durationSeconds: number,
  ): Promise<{ data: SessionResult | null; error: IamAuthError | null }> {
    const r = await postV1SessionsBySessionIdTrust({
      client: this._client,
      headers: this._headers(),
      path: { session_id: id },
      body: { duration_seconds: durationSeconds },
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { session?: ApiSession } | undefined;
    return { data: { session: data?.session }, error: null };
  }

  async revokeSession(id: string): Promise<{ error: IamAuthError | null }> {
    const r = await deleteV1SessionsBySessionId({
      client: this._client,
      headers: this._headers(),
      path: { session_id: id },
    });
    return { error: r.error ? accountError(r) : null };
  }

  async revokeAllSessions(params?: {
    exceptCurrent?: boolean;
  }): Promise<{ data: RevokeAllResult | null; error: IamAuthError | null }> {
    const r = await deleteV1Sessions({
      client: this._client,
      headers: this._headers(),
      body: params?.exceptCurrent !== undefined ? { except_current: params.exceptCurrent } : undefined,
    });
    if (r.error) return { data: null, error: accountError(r) };
    const data = r.data as { revoked_count?: number } | undefined;
    return { data: { revokedCount: data?.revoked_count ?? 0 }, error: null };
  }
}
