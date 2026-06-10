/**
 * IamMfa — MFA enrollment and factor management.
 * Login challenge/verify stay on the root IamAuth client (they mutate the
 * session). This namespace covers enrollment and factor lifecycle only.
 * Constructed by createIamClient; use via iam.mfa.
 */

import type { Client } from '@hey-api/client-fetch';
import {
  getV1AuthMfaFactors,
  postV1AuthMfaTotpEnroll,
  postV1AuthMfaTotpVerify,
  postV1AuthMfaSmsEnroll,
  postV1AuthMfaEmailEnroll,
  postV1AuthMfaWebauthnEnrollOptions,
  postV1AuthMfaWebauthnEnrollVerify,
  postV1AuthMfaRecoveryCodesGenerate,
  deleteV1AuthMfaFactorsByFactorId,
  type Factor,
} from '../gen';
import { IamAuthError } from './types';

// ---------------------------------------------------------------------------
// Public result types
// ---------------------------------------------------------------------------

export interface TotpEnrollResult {
  factorId: string;
  secret: string;
  otpauthUrl: string;
  qrSvg: string | null;
}

export interface TotpVerifyResult {
  factor: Factor | undefined;
}

export interface SmsEnrollResult {
  factorId: string;
  challengeId: string;
}

export interface EmailEnrollResult {
  factorId: string;
  challengeId: string;
}

export interface WebAuthnEnrollOptionsResult {
  challengeId: string;
  publicKey: Record<string, unknown>;
}

export interface WebAuthnEnrollVerifyResult {
  factor: Factor | undefined;
}

export interface RecoveryCodesResult {
  codes: Array<string>;
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

function mfaError(result: { error?: unknown; response?: Response }): IamAuthError {
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

export class IamMfa {
  constructor(
    private readonly _client: Client,
    private readonly _headers: () => { 'X-Client-Id': string },
  ) {}

  async listFactors(): Promise<{ data: { data: Array<Factor> } | null; error: IamAuthError | null }> {
    const r = await getV1AuthMfaFactors({ client: this._client, headers: this._headers() });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { data?: Array<Factor> } | undefined;
    return { data: { data: data?.data ?? [] }, error: null };
  }

  async enrollTotp(params?: {
    name?: string;
  }): Promise<{ data: TotpEnrollResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaTotpEnroll({
      client: this._client,
      headers: this._headers(),
      body: params?.name !== undefined ? { name: params.name } : undefined,
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { factor_id?: string; secret?: string; otpauth_url?: string; qr_svg?: string | null } | undefined;
    if (!data?.factor_id || !data.secret || !data.otpauth_url) {
      return { data: null, error: new IamAuthError('incomplete totp enroll response', 'invalid_response', r.response?.status) };
    }
    return {
      data: {
        factorId: data.factor_id,
        secret: data.secret,
        otpauthUrl: data.otpauth_url,
        qrSvg: data.qr_svg ?? null,
      },
      error: null,
    };
  }

  async verifyTotp(params: {
    factorId: string;
    code: string;
  }): Promise<{ data: TotpVerifyResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaTotpVerify({
      client: this._client,
      headers: this._headers(),
      body: { factor_id: params.factorId, code: params.code },
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { factor?: Factor } | undefined;
    return { data: { factor: data?.factor }, error: null };
  }

  async enrollSms(phone: string): Promise<{ data: SmsEnrollResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaSmsEnroll({
      client: this._client,
      headers: this._headers(),
      body: { phone },
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { factor_id?: string; challenge_id?: string } | undefined;
    if (!data?.factor_id || !data.challenge_id) {
      return { data: null, error: new IamAuthError('incomplete sms enroll response', 'invalid_response', r.response?.status) };
    }
    return { data: { factorId: data.factor_id, challengeId: data.challenge_id }, error: null };
  }

  async enrollEmail(email: string): Promise<{ data: EmailEnrollResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaEmailEnroll({
      client: this._client,
      headers: this._headers(),
      body: { email },
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { factor_id?: string; challenge_id?: string } | undefined;
    if (!data?.factor_id || !data.challenge_id) {
      return { data: null, error: new IamAuthError('incomplete email enroll response', 'invalid_response', r.response?.status) };
    }
    return { data: { factorId: data.factor_id, challengeId: data.challenge_id }, error: null };
  }

  async enrollWebAuthnOptions(params?: {
    name?: string;
  }): Promise<{ data: WebAuthnEnrollOptionsResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaWebauthnEnrollOptions({
      client: this._client,
      headers: this._headers(),
      body: params?.name !== undefined ? { name: params.name } : undefined,
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { challenge_id?: string; publicKey?: Record<string, unknown> } | undefined;
    if (!data?.challenge_id) {
      return { data: null, error: new IamAuthError('missing challenge_id', 'invalid_response', r.response?.status) };
    }
    return { data: { challengeId: data.challenge_id, publicKey: data.publicKey ?? {} }, error: null };
  }

  async enrollWebAuthnVerify(params: {
    challengeId: string;
    credential: Record<string, unknown>;
  }): Promise<{ data: WebAuthnEnrollVerifyResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaWebauthnEnrollVerify({
      client: this._client,
      headers: this._headers(),
      body: { challenge_id: params.challengeId, credential: params.credential },
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { factor?: Factor } | undefined;
    return { data: { factor: data?.factor }, error: null };
  }

  async generateRecoveryCodes(params?: {
    regenerate?: boolean;
  }): Promise<{ data: RecoveryCodesResult | null; error: IamAuthError | null }> {
    const r = await postV1AuthMfaRecoveryCodesGenerate({
      client: this._client,
      headers: this._headers(),
      body: params?.regenerate !== undefined ? { regenerate: params.regenerate } : undefined,
    });
    if (r.error) return { data: null, error: mfaError(r) };
    const data = r.data as { codes?: Array<string> } | undefined;
    return { data: { codes: data?.codes ?? [] }, error: null };
  }

  async removeFactor(id: string): Promise<{ error: IamAuthError | null }> {
    const r = await deleteV1AuthMfaFactorsByFactorId({
      client: this._client,
      headers: this._headers(),
      path: { factor_id: id },
    });
    return { error: r.error ? mfaError(r) : null };
  }
}
