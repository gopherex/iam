import { createClient, createConfig, type Client } from '@hey-api/client-fetch';
import {
  postV1AuthGuest,
  postV1AuthSignInPassword,
  postV1AuthSignUp,
  postV1AuthOtpStart,
  postV1AuthOtpVerify,
  postV1AuthMagicLinkStart,
  postV1AuthMagicLinkVerify,
  postV1AuthTokenExchange,
  postV1AuthOauthExchange,
  postV1AuthTokenRefresh,
  postV1AuthSignOut,
  postV1AuthEmailVerificationStart,
  postV1AuthEmailVerificationVerify,
  postV1AuthPhoneVerificationStart,
  postV1AuthPhoneVerificationVerify,
  postV1AuthPasswordForgot,
  postV1AuthPasswordReset,
  postV1AuthPasswordChange,
  postV1AuthSessionStepUp,
  postV1AuthMfaChallenge,
  postV1AuthMfaVerify,
  postV1AuthWebauthnLoginOptions,
  postV1AuthWebauthnLoginVerify,
  getV1ConfigPublic,
  type AuthResultOrNextStep,
  type ClientOptions as GeneratedClientOptions,
  type ConsentAcceptance,
  type GetV1AuthOauthByProviderStartData,
  type PhoneVerifyResult,
  type PublicConfig,
  type StepUpResult,
  type SessionTokens,
  type User,
} from '../gen';
import { defaultStorage, MemoryStorage } from './storage';
import {
  IamAuthError,
  type AuthChangeEvent,
  type AuthResponse,
  type IamClientOptions,
  type Session,
  type StorageAdapter,
  type Subscription,
} from './types';

const RETRY_HEADER = 'X-IAM-Retry';
const BROADCAST_NAME = 'iam:auth';

type Listener = (event: AuthChangeEvent, session: Session | null) => void;

type ChallengeResponse = { data: { challengeId: string } | null; error: IamAuthError | null };

function toSession(tokens: SessionTokens, user: User): Session {
  return {
    access_token: tokens.access_token,
    refresh_token: tokens.refresh_token,
    token_type: tokens.token_type,
    expires_at: Date.now() + tokens.expires_in * 1000,
    user,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isStoredSession(value: unknown): value is Session {
  if (!isRecord(value)) return false;
  if (typeof value.access_token !== 'string' || value.access_token === '') return false;
  if (value.refresh_token !== undefined && typeof value.refresh_token !== 'string') return false;
  if (typeof value.token_type !== 'string' || value.token_type === '') return false;
  if (typeof value.expires_at !== 'number' || !Number.isFinite(value.expires_at)) return false;
  if (!isRecord(value.user) || typeof value.user.id !== 'string' || value.user.id === '') return false;
  return true;
}

export class IamConfig {
  constructor(
    private readonly client: Client,
    private readonly clientId: string,
  ) {}

  async getPublicConfig(): Promise<{ data: PublicConfig | null; error: IamAuthError | null }> {
    const r = await getV1ConfigPublic({
      client: this.client,
      headers: { 'X-Client-Id': this.clientId },
    });
    if (r.error) return { data: null, error: authError(r) };
    return { data: r.data ?? null, error: null };
  }
}

function authError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('request failed', 'request_failed', status);
}

/**
 * IamAuth is the Supabase-style auth surface: stateful session management with
 * automatic persistence, background refresh, cross-tab sync, and a typed method
 * per sign-in flow. Construct it via createIamClient.
 */
export class IamAuth {
  private session: Session | null = null;
  private listeners = new Set<Listener>();
  private storage: StorageAdapter;
  private storageKey: string;
  private clientId: string;
  private persist: boolean;
  private autoRefresh: boolean;
  private marginMs: number;
  private refreshTimer: ReturnType<typeof setTimeout> | null = null;
  private inflightRefresh: Promise<boolean> | null = null;
  private channel: BroadcastChannel | null = null;
  private initialized: Promise<void>;
  public readonly client: Client;

  constructor(opts: IamClientOptions) {
    this.clientId = opts.clientId;
    this.persist = opts.persistSession ?? true;
    this.storage = opts.storage ?? (this.persist ? defaultStorage() : new MemoryStorage());
    this.storageKey = opts.storageKey ?? 'iam.session';
    this.autoRefresh = opts.autoRefresh ?? true;
    this.marginMs = (opts.refreshMarginSeconds ?? 30) * 1000;
    this.client = createClient(createConfig<GeneratedClientOptions>({
      baseUrl: opts.baseUrl,
      auth: async () => this.session?.access_token,
    }));

    this.installInterceptors();
    if (this.persist && (opts.multiTab ?? true) && typeof BroadcastChannel !== 'undefined') {
      this.channel = new BroadcastChannel(BROADCAST_NAME);
      this.channel.onmessage = () => void this.reloadFromStorage();
    }
    this.initialized = this.loadInitial();
  }

  // ----- public API -----

  /** Resolves once the initial (persisted) session has been loaded. */
  async ready(): Promise<void> {
    await this.initialized;
  }

  /** The current session, or null. */
  async getSession(): Promise<Session | null> {
    await this.ready();
    return this.session;
  }

  /** The current user, or null. */
  async getUser(): Promise<User | null> {
    return (await this.getSession())?.user ?? null;
  }

  /** Subscribe to auth lifecycle changes. Fires INITIAL_SESSION immediately. */
  onAuthStateChange(cb: Listener): Subscription {
    this.listeners.add(cb);
    void this.ready().then(() => cb('INITIAL_SESSION', this.session));
    return { unsubscribe: () => this.listeners.delete(cb) };
  }

  async signInWithPassword(params: { email?: string; phone?: string; password: string }): Promise<AuthResponse> {
    const r = await postV1AuthSignInPassword({
      client: this.client,
      headers: this.headers(),
      body: { email: params.email ?? null, phone: params.phone ?? null, password: params.password },
    });
    return this.handle(r);
  }

  async signInAnonymously(): Promise<AuthResponse> {
    const r = await postV1AuthGuest({
      client: this.client,
      headers: this.headers(),
      body: {},
    });
    return this.handle(r);
  }

  async signUp(params: {
    email?: string;
    phone?: string;
    password?: string;
    name?: string;
    metadata?: Record<string, unknown>;
    consents?: Array<ConsentAcceptance>;
  }): Promise<AuthResponse> {
    const r = await postV1AuthSignUp({
      client: this.client,
      headers: this.headers(),
      body: {
        email: params.email ?? null,
        phone: params.phone ?? null,
        password: params.password ?? null,
        name: params.name ?? null,
        metadata: params.metadata,
        consents: params.consents ?? [],
      },
    });
    return this.handle(r);
  }

  /** Start an OTP flow (email/SMS); returns the challenge id to pass to verifyOtp. */
  async signInWithOtp(params: {
    identifier: string;
    channel?: 'email' | 'sms' | 'whatsapp';
    purpose?: 'signin' | 'signup' | 'verify';
  }): Promise<{ data: { challengeId: string } | null; error: IamAuthError | null }> {
    const r = await postV1AuthOtpStart({
      client: this.client,
      headers: this.headers(),
      body: {
        identifier: params.identifier,
        channel: params.channel ?? 'email',
        purpose: params.purpose ?? 'signin',
      },
    });
    if (r.error) return { data: null, error: authError(r) };
    return { data: { challengeId: (r.data as { challenge_id: string }).challenge_id }, error: null };
  }

  async verifyOtp(params: { challengeId: string; code: string }): Promise<AuthResponse> {
    const r = await postV1AuthOtpVerify({
      client: this.client,
      headers: this.headers(),
      body: { challenge_id: params.challengeId, code: params.code },
    });
    return this.handle(r);
  }

  /** Send a magic link. The link returns to redirectTo with a token; verifyMagicLink finishes it. */
  async signInWithMagicLink(params: {
    email: string;
    redirectTo: string;
    purpose?: 'signin' | 'signup';
  }): Promise<{ error: IamAuthError | null }> {
    const r = await postV1AuthMagicLinkStart({
      client: this.client,
      headers: this.headers(),
      body: { email: params.email, redirect_to: params.redirectTo, purpose: params.purpose ?? 'signin' },
    });
    return { error: r.error ? authError(r) : null };
  }

  async verifyMagicLink(params: { token: string }): Promise<AuthResponse> {
    const r = await postV1AuthMagicLinkVerify({ client: this.client, headers: this.headers(), body: { token: params.token } });
    return this.handle(r);
  }

  async sendEmailVerification(params: {
    email?: string;
    redirectTo?: string;
    locale?: string;
  }): Promise<ChallengeResponse> {
    const r = await postV1AuthEmailVerificationStart({
      client: this.client,
      headers: this.headers(),
      body: {
        email: params.email,
        redirect_to: params.redirectTo ?? null,
        locale: params.locale ?? null,
      },
    });
    return this.challengeResult(r);
  }

  async verifyEmail(params: { challengeId: string; code: string }): Promise<AuthResponse> {
    const r = await postV1AuthEmailVerificationVerify({
      client: this.client,
      headers: this.headers(),
      body: { challenge_id: params.challengeId, code: params.code },
    });
    return this.handle(r);
  }

  async sendPhoneVerification(params: {
    phone: string;
    channel?: 'sms' | 'whatsapp';
    locale?: string;
  }): Promise<ChallengeResponse> {
    const r = await postV1AuthPhoneVerificationStart({
      client: this.client,
      headers: this.headers(),
      body: {
        phone: params.phone,
        channel: params.channel ?? 'sms',
        locale: params.locale ?? null,
      },
    });
    return this.challengeResult(r);
  }

  async verifyPhone(params: { challengeId: string; code: string }): Promise<AuthResponse> {
    const r = await postV1AuthPhoneVerificationVerify({
      client: this.client,
      headers: this.headers(),
      body: { challenge_id: params.challengeId, code: params.code },
    });
    return this.handlePhoneVerify(r);
  }

  /**
   * Begin an OAuth/social sign-in. In a browser this redirects to the provider
   * (unless skipBrowserRedirect); the provider returns to your app with a `code`
   * query param to pass to exchangeCodeForSession. Returns the start URL.
   */
  signInWithOAuth(params: {
    provider: string;
    redirectTo?: string;
    state?: string;
    codeChallenge?: string;
    prompt?: string;
    loginHint?: string;
    skipBrowserRedirect?: boolean;
  }): { url: string } {
    const redirectTo = params.redirectTo ?? (typeof window !== 'undefined' ? window.location.href : undefined);
    if (!redirectTo) {
      throw new IamAuthError('redirectTo is required outside a browser', 'redirect_to_required');
    }
    const url = this.client.buildUrl<GetV1AuthOauthByProviderStartData>({
      baseUrl: this.client.getConfig().baseUrl,
      url: '/v1/auth/oauth/{provider}/start',
      path: { provider: params.provider },
      query: {
        client_id: this.clientId,
        redirect_to: redirectTo,
        state: params.state,
        code_challenge: params.codeChallenge,
        prompt: params.prompt,
        login_hint: params.loginHint,
      },
    });
    if (!params.skipBrowserRedirect && typeof window !== 'undefined') {
      window.location.assign(url);
    }
    return { url };
  }

  /** Exchange the auth code from an OAuth/redirect callback for a session. */
  async exchangeCodeForSession(params: { code: string; codeVerifier?: string }): Promise<AuthResponse> {
    const r = await postV1AuthTokenExchange({
      client: this.client,
      headers: this.headers(),
      body: { grant_type: 'auth_code', code: params.code, code_verifier: params.codeVerifier ?? null },
    });
    return this.handle(r);
  }

  async exchangeOAuthCodeForSession(params: { code: string; codeVerifier?: string }): Promise<AuthResponse> {
    const r = await postV1AuthOauthExchange({
      client: this.client,
      headers: this.headers(),
      body: { code: params.code, code_verifier: params.codeVerifier ?? null },
    });
    return this.handle(r);
  }

  async resetPasswordForEmail(params: {
    email: string;
    redirectTo?: string;
    locale?: string;
    captchaToken?: string;
  }): Promise<{ error: IamAuthError | null }> {
    const r = await postV1AuthPasswordForgot({
      client: this.client,
      headers: this.headers(),
      body: {
        email: params.email,
        redirect_to: params.redirectTo ?? null,
        locale: params.locale ?? null,
        captcha_token: params.captchaToken ?? null,
      },
    });
    return { error: r.error ? authError(r) : null };
  }

  async resetPassword(params: {
    newPassword: string;
    token?: string;
    challengeId?: string;
    code?: string;
  }): Promise<AuthResponse> {
    const r = await postV1AuthPasswordReset({
      client: this.client,
      headers: this.headers(),
      body: {
        new_password: params.newPassword,
        token: params.token ?? null,
        challenge_id: params.challengeId ?? null,
        code: params.code ?? null,
      },
    });
    return this.handle(r);
  }

  async updatePassword(params: {
    currentPassword?: string;
    newPassword: string;
    revokeOtherSessions?: boolean;
  }): Promise<{ error: IamAuthError | null }> {
    const r = await postV1AuthPasswordChange({
      client: this.client,
      body: {
        current_password: params.currentPassword ?? null,
        new_password: params.newPassword,
        revoke_other_sessions: params.revokeOtherSessions,
      },
    });
    return { error: r.error ? authError(r) : null };
  }

  async stepUp(params: {
    purpose: string;
    requiredAal?: 1 | 2;
    maxAgeSeconds?: number;
  }): Promise<AuthResponse> {
    const r = await postV1AuthSessionStepUp({
      client: this.client,
      body: {
        purpose: params.purpose,
        required_aal: params.requiredAal,
        max_age_seconds: params.maxAgeSeconds ?? null,
      },
    });
    if (r.error) return { data: { session: null, user: null }, error: authError(r) };
    const payload = r.data as StepUpResult;
    if (payload.result_type === 'next_step') {
      return {
        data: {
          session: null,
          user: null,
          nextStep: payload.next_step,
          factors: payload.factors,
          flowToken: payload.flow_token,
          documents: payload.documents,
        },
        error: null,
      };
    }
    return { data: { session: this.session, user: this.session?.user ?? null }, error: null };
  }

  async challengeMfa(params: {
    flowToken?: string;
    factorId?: string;
    type?: string;
  } = {}): Promise<ChallengeResponse> {
    const r = await postV1AuthMfaChallenge({
      client: this.client,
      headers: this.headers(),
      body: {
        flow_token: params.flowToken ?? null,
        factor_id: params.factorId ?? null,
        type: params.type ?? null,
      },
    });
    return this.challengeResult(r);
  }

  async verifyMfa(params: {
    flowToken?: string;
    challengeId?: string;
    factorId?: string;
    code?: string;
    credential?: Record<string, unknown>;
    recoveryCode?: string;
  }): Promise<AuthResponse> {
    const r = await postV1AuthMfaVerify({
      client: this.client,
      headers: this.headers(),
      body: {
        flow_token: params.flowToken ?? null,
        challenge_id: params.challengeId ?? null,
        factor_id: params.factorId ?? null,
        code: params.code ?? null,
        credential: params.credential ?? null,
        recovery_code: params.recoveryCode ?? null,
      },
    });
    return this.handle(r);
  }

  /** Passkey/WebAuthn sign-in (browser only — uses navigator.credentials). */
  async signInWithWebAuthn(params: { email?: string; username?: string } = {}): Promise<AuthResponse> {
    if (typeof navigator === 'undefined' || !navigator.credentials) {
      return { data: { session: null, user: null }, error: new IamAuthError('WebAuthn unavailable', 'webauthn_unavailable') };
    }
    const opt = await postV1AuthWebauthnLoginOptions({
      client: this.client,
      headers: this.headers(),
      body: { email: params.email ?? null, username: params.username ?? null },
    });
    if (opt.error) return { data: { session: null, user: null }, error: authError(opt) };

    const data = opt.data as { challenge_id?: string; publicKey?: Record<string, unknown> };
    const publicKey = decodeRequestOptions(data.publicKey);
    const cred = (await navigator.credentials.get({ publicKey })) as PublicKeyCredential | null;
    if (!cred) return { data: { session: null, user: null }, error: new IamAuthError('WebAuthn cancelled', 'webauthn_cancelled') };

    const r = await postV1AuthWebauthnLoginVerify({
      client: this.client,
      headers: this.headers(),
      body: { challenge_id: data.challenge_id ?? '', credential: encodeAssertion(cred) },
    });
    return this.handle(r);
  }

  /** Force a token refresh now. Returns the refreshed session or null. */
  async refreshSession(): Promise<Session | null> {
    const ok = await this.refreshSingleFlight();
    return ok ? this.session : null;
  }

  /** Revoke the current session server-side and clear local state. */
  async signOut(): Promise<{ error: IamAuthError | null }> {
    await this.ready();
    let error: IamAuthError | null = null;
    if (this.session) {
      const r = await postV1AuthSignOut({ client: this.client, headers: this.headers(), body: {} });
      if (r.error) error = authError(r);
    }
    await this.setSession(null, 'SIGNED_OUT');
    return { error };
  }

  // ----- engine -----

  private headers(): { 'X-Client-Id': string } {
    return { 'X-Client-Id': this.clientId };
  }

  private async handle(result: { data?: unknown; error?: unknown; response?: Response }): Promise<AuthResponse> {
    if (result.error) return { data: { session: null, user: null }, error: authError(result) };
    const payload = result.data as AuthResultOrNextStep;
    if (payload.result_type === 'authenticated') {
      const session = toSession(payload.session, payload.user);
      await this.setSession(session, 'SIGNED_IN');
      return { data: { session, user: payload.user, nextStep: payload.next_step ?? undefined }, error: null };
    }
    return {
      data: {
        session: null,
        user: null,
        nextStep: payload.next_step,
        factors: payload.factors,
        flowToken: payload.flow_token,
        documents: payload.documents,
      },
      error: null,
    };
  }

  private challengeResult(result: { data?: unknown; error?: unknown; response?: Response }): ChallengeResponse {
    if (result.error) return { data: null, error: authError(result) };
    const data = result.data as { challenge_id?: string } | undefined;
    if (!data?.challenge_id) {
      return { data: null, error: new IamAuthError('challenge id missing', 'invalid_response', result.response?.status) };
    }
    return { data: { challengeId: data.challenge_id }, error: null };
  }

  private async handlePhoneVerify(result: { data?: unknown; error?: unknown; response?: Response }): Promise<AuthResponse> {
    if (result.error) return { data: { session: null, user: null }, error: authError(result) };
    const payload = result.data as PhoneVerifyResult;
    if (payload.result_type === 'authenticated') {
      const session = toSession(payload.session, payload.user);
      await this.setSession(session, 'SIGNED_IN');
      return { data: { session, user: payload.user, nextStep: payload.next_step ?? undefined }, error: null };
    }
    if (payload.result_type === 'user_updated') {
      const session = this.session ? { ...this.session, user: payload.user } : null;
      if (session) await this.setSession(session, 'USER_UPDATED');
      return { data: { session, user: payload.user }, error: null };
    }
    return { data: { session: null, user: null }, error: new IamAuthError('unsupported phone verification response', 'invalid_response', result.response?.status) };
  }

  private async setSession(session: Session | null, event: AuthChangeEvent): Promise<void> {
    this.session = session;
    if (this.persist) {
      try {
        if (session) await this.storage.setItem(this.storageKey, JSON.stringify(session));
        else await this.storage.removeItem(this.storageKey);
      } catch {
        /* ignore storage failures */
      }
    }
    this.scheduleRefresh();
    this.emit(event);
    this.channel?.postMessage({ at: Date.now() });
  }

  private async loadInitial(): Promise<void> {
    if (!this.persist) return;
    await this.reloadFromStorage(false);
    this.scheduleRefresh();
  }

  /** Re-read the session from storage (used on init and on cross-tab broadcast). */
  private async reloadFromStorage(emitChange = true): Promise<void> {
    try {
      const raw = await this.storage.getItem(this.storageKey);
      const parsed = raw ? JSON.parse(raw) : null;
      let next = isStoredSession(parsed) ? parsed : null;
      if (raw && !next) {
        await this.storage.removeItem(this.storageKey);
      }
      if (next && next.expires_at <= Date.now() && !next.refresh_token) {
        await this.storage.removeItem(this.storageKey);
        next = null;
      }
      const changed = (next?.access_token ?? null) !== (this.session?.access_token ?? null);
      this.session = next;
      if (changed && emitChange) {
        this.scheduleRefresh();
        this.emit(next ? 'TOKEN_REFRESHED' : 'SIGNED_OUT');
      }
    } catch {
      /* ignore */
    }
  }

  private emit(event: AuthChangeEvent): void {
    for (const cb of this.listeners) cb(event, this.session);
  }

  private scheduleRefresh(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
    if (!this.autoRefresh || !this.session?.refresh_token) return;
    const delay = Math.max(0, this.session.expires_at - Date.now() - this.marginMs);
    this.refreshTimer = setTimeout(() => void this.refreshSingleFlight(), delay);
  }

  /** Refresh, de-duplicating concurrent callers (proactive timer + 401 retry). */
  private refreshSingleFlight(): Promise<boolean> {
    if (this.inflightRefresh) return this.inflightRefresh;
    this.inflightRefresh = this.doRefresh().finally(() => {
      this.inflightRefresh = null;
    });
    return this.inflightRefresh;
  }

  private async doRefresh(): Promise<boolean> {
    const rt = this.session?.refresh_token;
    if (!rt) return false;
    const r = await postV1AuthTokenRefresh({ client: this.client, headers: this.headers(), body: { refresh_token: rt } });
    if (r.error || !r.data) {
      await this.setSession(null, 'SIGNED_OUT');
      return false;
    }
    const body = r.data as { user?: User; session?: SessionTokens } | undefined;
    // The refresh response is an AuthResult; reuse the existing user when absent.
    const tokens = (body?.session ?? (r.data as unknown as SessionTokens));
    const user = body?.user ?? this.session?.user;
    if (!tokens?.access_token || !user) {
      await this.setSession(null, 'SIGNED_OUT');
      return false;
    }
    await this.setSession(toSession(tokens, user), 'TOKEN_REFRESHED');
    return true;
  }

  private installInterceptors(): void {
    this.client.interceptors.request.use((request: Request) => {
      if (this.clientId && !request.headers.has('X-Client-Id')) {
        request.headers.set('X-Client-Id', this.clientId);
      }
      return request;
    });
    this.client.interceptors.response.use(async (response: Response, request: Request) => {
      if (
        response.status !== 401 ||
        request.headers.has(RETRY_HEADER) ||
        request.url.includes('/v1/auth/token/refresh') ||
        !this.session?.refresh_token
      ) {
        return response;
      }
      const ok = await this.refreshSingleFlight();
      if (!ok || !this.session) return response;
      const retry = request.clone();
      retry.headers.set('Authorization', `Bearer ${this.session.access_token}`);
      retry.headers.set(RETRY_HEADER, '1');
      return fetch(retry);
    });
  }
}

// ----- WebAuthn (base64url <-> ArrayBuffer) helpers -----

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

function decodeRequestOptions(pk: any): PublicKeyCredentialRequestOptions {
  return {
    ...pk,
    challenge: b64urlToBuf(pk.challenge),
    allowCredentials: (pk.allowCredentials ?? []).map((c: any) => ({ ...c, id: b64urlToBuf(c.id) })),
  };
}

function encodeAssertion(cred: PublicKeyCredential): Record<string, unknown> {
  const r = cred.response as AuthenticatorAssertionResponse;
  return {
    id: cred.id,
    raw_id: bufToB64url(cred.rawId),
    type: cred.type,
    response: {
      client_data_json: bufToB64url(r.clientDataJSON),
      authenticator_data: bufToB64url(r.authenticatorData),
      signature: bufToB64url(r.signature),
      user_handle: r.userHandle ? bufToB64url(r.userHandle) : null,
    },
  };
}

/**
 * createIamClient configures the shared SDK client (base URL, bearer auth,
 * refresh + client-id interceptors) and returns the Supabase-style `auth`
 * surface plus the underlying client for raw/management calls.
 */
export function createIamClient(options: IamClientOptions): {
  auth: IamAuth;
  config: IamConfig;
  client: Client;
} {
  const auth = new IamAuth(options);
  return { auth, config: new IamConfig(auth.client, options.clientId), client: auth.client };
}
