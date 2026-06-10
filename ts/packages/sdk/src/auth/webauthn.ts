/**
 * IamWebAuthn — passkey (WebAuthn) management: register, list, rename, delete.
 * Browser-ceremony helpers guard for non-browser environments.
 * Login (passkey sign-in) stays on the root IamAuth.signInWithWebAuthn.
 * Constructed by createIamClient; use via iam.webauthn.
 */

import type { Client } from '@hey-api/client-fetch';
import {
  postV1AuthWebauthnRegisterOptions,
  postV1AuthWebauthnRegisterVerify,
  getV1AuthWebauthnCredentials,
  patchV1AuthWebauthnCredentialsByCredentialId,
  deleteV1AuthWebauthnCredentialsByCredentialId,
  type WebAuthnCredential,
} from '../gen';
import { IamAuthError } from './types';

// ---------------------------------------------------------------------------
// Public result types
// ---------------------------------------------------------------------------

export interface RegisterOptionsResult {
  challengeId: string;
  publicKey: Record<string, unknown>;
}

export interface RegisterVerifyResult {
  credential: WebAuthnCredential | undefined;
}

export interface CredentialList {
  data: Array<WebAuthnCredential>;
}

export interface CredentialResult {
  credential: WebAuthnCredential | undefined;
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

function webauthnError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('request failed', 'request_failed', status);
}

// ---- base64url <-> ArrayBuffer helpers (same as client.ts) ----------------

function b64urlToBuf(s: string): ArrayBuffer {
  const pad = s.length % 4 === 0 ? '' : '='.repeat(4 - (s.length % 4));
  const b64 = (s.replace(/-/g, '+').replace(/_/g, '/')) + pad;
  const bin = atob(b64);
  const buf = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i);
  return buf.buffer;
}

function bufToB64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let bin = '';
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function decodeCreationOptions(pk: Record<string, unknown>): PublicKeyCredentialCreationOptions {
  const user = pk.user as { id: string; [k: string]: unknown } | undefined;
  return {
    ...(pk as unknown as PublicKeyCredentialCreationOptions),
    challenge: b64urlToBuf(pk.challenge as string),
    user: user ? { ...user, id: b64urlToBuf(user.id) } : (pk.user as BufferSource),
    excludeCredentials: ((pk.excludeCredentials ?? []) as Array<{ id: string; [k: string]: unknown }>).map(
      (c) => ({ ...c, id: b64urlToBuf(c.id) }),
    ),
  } as PublicKeyCredentialCreationOptions;
}

function encodeAttestation(cred: PublicKeyCredential): Record<string, unknown> {
  const r = cred.response as AuthenticatorAttestationResponse;
  return {
    id: cred.id,
    raw_id: bufToB64url(cred.rawId),
    type: cred.type,
    response: {
      client_data_json: bufToB64url(r.clientDataJSON),
      attestation_object: bufToB64url(r.attestationObject),
    },
  };
}

// ---------------------------------------------------------------------------
// Class
// ---------------------------------------------------------------------------

export class IamWebAuthn {
  constructor(
    private readonly _client: Client,
    private readonly _headers: () => { 'X-Client-Id': string },
  ) {}

  /** POST /v1/auth/webauthn/register/options — get creation challenge. */
  async registerOptions(params?: {
    name?: string;
    residentKey?: string;
    userVerification?: string;
  }): Promise<{ data: RegisterOptionsResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthWebauthnRegisterOptions({
      client: this._client,
      headers: this._headers(),
      body: params
        ? {
            name: params.name ?? null,
            resident_key: params.residentKey ?? null,
            user_verification: params.userVerification ?? null,
          }
        : undefined,
    });
    if (r.error) return { data: null, error: webauthnError(r) };
    const data = r.data as { challenge_id?: string; publicKey?: Record<string, unknown> } | undefined;
    if (!data?.challenge_id) {
      return { data: null, error: new IamAuthError('missing challenge_id', 'invalid_response', r.response?.status) };
    }
    return { data: { challengeId: data.challenge_id, publicKey: data.publicKey ?? {} }, error: null };
  }

  /** POST /v1/auth/webauthn/register/verify — confirm attestation. */
  async registerVerify(params: {
    challengeId: string;
    credential: Record<string, unknown>;
    name?: string;
  }): Promise<{ data: RegisterVerifyResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthWebauthnRegisterVerify({
      client: this._client,
      headers: this._headers(),
      body: {
        challenge_id: params.challengeId,
        credential: params.credential,
        name: params.name ?? null,
      },
    });
    if (r.error) return { data: null, error: webauthnError(r) };
    const data = r.data as { credential?: WebAuthnCredential } | undefined;
    return { data: { credential: data?.credential }, error: null };
  }

  /**
   * Convenience: run the full browser ceremony between registerOptions and
   * registerVerify. Guards for non-browser environments.
   */
  async registerPasskey(params?: {
    name?: string;
    residentKey?: string;
    userVerification?: string;
  }): Promise<{ data: RegisterVerifyResult | null; error: IamAuthError | null }> {
    if (typeof navigator === 'undefined' || !navigator.credentials) {
      return {
        data: null,
        error: new IamAuthError('WebAuthn unavailable in this environment', 'webauthn_unavailable'),
      };
    }

    const optResult = await this.registerOptions(params);
    if (optResult.error || !optResult.data) {
      return { data: null, error: optResult.error ?? new IamAuthError('no options returned', 'invalid_response') };
    }

    const publicKey = decodeCreationOptions(optResult.data.publicKey);
    let cred: PublicKeyCredential | null;
    try {
      cred = (await navigator.credentials.create({ publicKey })) as PublicKeyCredential | null;
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'WebAuthn error';
      return { data: null, error: new IamAuthError(msg, 'webauthn_error') };
    }

    if (!cred) {
      return { data: null, error: new IamAuthError('WebAuthn cancelled', 'webauthn_cancelled') };
    }

    return this.registerVerify({
      challengeId: optResult.data.challengeId,
      credential: encodeAttestation(cred),
      name: params?.name,
    });
  }

  /** GET /v1/auth/webauthn/credentials — list registered passkeys. */
  async listCredentials(): Promise<{ data: CredentialList | null; error: IamAuthError | null }> {
    const r = await getV1AuthWebauthnCredentials({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: webauthnError(r) };
    const data = r.data as { data?: Array<WebAuthnCredential> } | undefined;
    return { data: { data: data?.data ?? [] }, error: null };
  }

  /** PATCH /v1/auth/webauthn/credentials/{credential_id} — rename a passkey. */
  async renameCredential(
    id: string,
    name: string,
  ): Promise<{ data: CredentialResult | null; error: IamAuthError | null }> {
    const r = await patchV1AuthWebauthnCredentialsByCredentialId({
      client: this._client,
      headers: this._headers(),
      path: { credential_id: id },
      body: { name },
    });
    if (r.error) return { data: null, error: webauthnError(r) };
    const data = r.data as { credential?: WebAuthnCredential } | undefined;
    return { data: { credential: data?.credential }, error: null };
  }

  /** DELETE /v1/auth/webauthn/credentials/{credential_id} — remove a passkey. */
  async deleteCredential(id: string): Promise<{ error: IamAuthError | null }> {
    const r = await deleteV1AuthWebauthnCredentialsByCredentialId({
      client: this._client,
      headers: this._headers(),
      path: { credential_id: id },
    });
    return { error: r.error ? webauthnError(r) : null };
  }
}
