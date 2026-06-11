/**
 * FlowController — stateful, framework-agnostic controller for the server-side
 * resumable auth flow API (§6/§8 of docs/design/resumable-auth-flows.md).
 *
 * Responsibilities:
 *  - Start, resume (cookie or token), submit, resend, and abandon flows.
 *  - Persist the current flow_token in localStorage (key `iam.flow`).
 *  - Broadcast state changes across tabs via BroadcastChannel('iam:flow').
 *  - On completion, hand the session to the IamAuth client (optional) and clear storage.
 *  - Map API errors to IamAuthError.
 */

import { createClient, createConfig, type Client } from '@hey-api/client-fetch';
import {
  postV1AuthFlows,
  getV1AuthFlowsCurrent,
  getV1AuthFlowsByFlowToken,
  postV1AuthFlowsByFlowTokenSubmit,
  postV1AuthFlowsByFlowTokenResend,
  deleteV1AuthFlowsByFlowToken,
  type FlowState,
  type ClientOptions as GeneratedClientOptions,
} from '../gen';
import { IamAuthError } from './types';
import type { IamAuth } from './client';

// ---- types ----

export type FlowKind = 'signup' | 'signin' | 'recovery' | 'email_change';

export interface FlowControllerOptions {
  /** API base URL (same as IamClientOptions.baseUrl). */
  baseUrl: string;
  /** Public client id sent as X-Client-Id on every call. */
  clientId: string;
  /**
   * Project environment sent as X-Environment on every flow call (test/live
   * isolation). Defaults to the project's "live" environment when omitted.
   */
  environment?: string;
  /**
   * If provided, the controller will call `auth.acceptFlowSession(tokens, user)`
   * when the flow completes, triggering SIGNED_IN across the app.
   */
  auth?: IamAuth;
  /**
   * localStorage key for persisting the flow_token (default: 'iam.flow').
   */
  storageKey?: string;
}

export type FlowChangeCallback = (state: FlowState | null, error: IamAuthError | null) => void;

export interface FlowController {
  /**
   * Start a new flow. Replaces any in-progress flow in storage.
   *
   * @param params.kind  One of 'signup' | 'signin' | 'recovery' | 'email_change'
   * @param params.email Optional — pre-filled credential
   * @param params.password Optional — pre-filled credential
   * @param params.name  Optional — display name for signup
   * @param params.redirectTo Optional — per-flow override for the cross-device
   *   "continue" deep-link base. Honoured only when its origin matches the
   *   project's configured app_base_url; otherwise ignored server-side.
   */
  start(params: {
    kind: FlowKind;
    /**
     * Authentication method driving the flow. Signin supports password
     * (default), phone_otp, magic_link, passkey and oauth. Recovery supports
     * email (default) and phone_otp. Ignored for signup.
     */
    method?: 'password' | 'phone_otp' | 'magic_link' | 'passkey' | 'oauth';
    email?: string;
    /** E.164 phone for the phone_otp method (signin/recovery). */
    phone?: string;
    /** OAuth provider id for the oauth signin method. */
    provider?: string;
    password?: string;
    name?: string;
    captchaToken?: string;
    redirectTo?: string;
    /** Preferred language for this flow's emails (e.g. 'ru'). */
    locale?: string;
    /**
     * Invitation token (inv_…) to redeem when the project's registration mode is
     * invite_only. Without a valid token an invite_only signup is blocked
     * (step:blocked, error:invite_required).
     */
    inviteToken?: string;
  }): Promise<{ state: FlowState | null; error: IamAuthError | null }>;

  /**
   * Redeem an invitation: start a signup flow carrying the invite token. Shortcut
   * for start({ kind: 'signup', inviteToken, ... }) — the only way to register
   * when the project's registration mode is invite_only.
   */
  redeemInvite(
    inviteToken: string,
    params?: { email?: string; password?: string; name?: string; locale?: string },
  ): Promise<{ state: FlowState | null; error: IamAuthError | null }>;

  /**
   * Submit an action for the current step.
   *
   * @param action  The action string (e.g. 'submit', 'verify_email', 'mfa', 'accept_consents', 'request_access')
   * @param payload Key/value payload for the step (code, password, etc.)
   */
  submit(
    action: string,
    payload?: Record<string, unknown>,
  ): Promise<{ state: FlowState | null; error: IamAuthError | null }>;

  /**
   * Re-send the current challenge (email/SMS code).
   * Returns 429 error when rate-limited (resend_at not yet reached).
   */
  resend(): Promise<{ state: FlowState | null; error: IamAuthError | null }>;

  /**
   * Resume the flow identified by the HttpOnly `iam_flow` cookie.
   * Use this on page load to restore state across refreshes without a token
   * in JS. Falls back to the token in localStorage if the cookie call returns 404.
   */
  resume(): Promise<{ state: FlowState | null; error: IamAuthError | null }>;

  /**
   * Resume a specific flow by explicit token (deep-link: ?flow=<token>).
   * Also stores the token in localStorage so subsequent resume() calls work.
   */
  resumeByToken(token: string): Promise<{ state: FlowState | null; error: IamAuthError | null }>;

  /**
   * Abandon (DELETE) the current flow and clear local storage.
   */
  abandon(): Promise<{ error: IamAuthError | null }>;

  /**
   * Register a callback for state changes. Fires immediately if there is
   * already an in-memory state.
   *
   * @returns Unsubscribe function.
   */
  onChange(cb: FlowChangeCallback): () => void;

  /** Current in-memory FlowState (null if no flow active). */
  readonly currentState: FlowState | null;
}

// ---- internal helpers ----

const FLOW_BROADCAST = 'iam:flow';

function flowError(result: { error?: unknown; response?: Response }): IamAuthError {
  const status = result.response?.status;
  const env = result.error as { error?: { code?: string; message?: string } } | undefined;
  if (env?.error?.code) {
    return new IamAuthError(env.error.message ?? env.error.code, env.error.code, status);
  }
  return new IamAuthError('flow request failed', 'flow_request_failed', status);
}

// ---- factory ----

/**
 * Create a stateful FlowController. One instance per app / per login page.
 *
 * @example
 * const flow = createFlowController({ baseUrl: '', clientId: 'my-client' });
 * flow.onChange((state, err) => { ... });
 * await flow.resume();
 */
export function createFlowController(opts: FlowControllerOptions): FlowController {
  const storageKey = opts.storageKey ?? 'iam.flow';

  const httpClient: Client = createClient(
    createConfig<GeneratedClientOptions>({ baseUrl: opts.baseUrl }),
  );

  const headers = (): { 'X-Client-Id': string; 'X-Environment'?: string } => {
    const h: { 'X-Client-Id': string; 'X-Environment'?: string } = { 'X-Client-Id': opts.clientId };
    if (opts.environment) h['X-Environment'] = opts.environment;
    return h;
  };

  let currentState: FlowState | null = null;
  const listeners = new Set<FlowChangeCallback>();
  let channel: BroadcastChannel | null = null;

  if (typeof BroadcastChannel !== 'undefined') {
    channel = new BroadcastChannel(FLOW_BROADCAST);
    channel.onmessage = () => {
      // Re-read from storage when another tab signals a change.
      // We don't get the full state over the channel to avoid races; let the
      // receiver call resume() if it wants a full refresh.
      const token = readToken();
      if (!token && currentState) {
        // another tab cleared the flow
        notify(null, null);
      }
    };
  }

  function readToken(): string | null {
    try {
      return typeof localStorage !== 'undefined' ? localStorage.getItem(storageKey) : null;
    } catch {
      return null;
    }
  }

  function writeToken(token: string): void {
    try {
      if (typeof localStorage !== 'undefined') localStorage.setItem(storageKey, token);
    } catch {
      /* ignore */
    }
  }

  function clearToken(): void {
    try {
      if (typeof localStorage !== 'undefined') localStorage.removeItem(storageKey);
    } catch {
      /* ignore */
    }
    channel?.postMessage({ cleared: true });
  }

  function notify(state: FlowState | null, error: IamAuthError | null): void {
    currentState = state;
    for (const cb of listeners) cb(state, error);
  }

  async function handleState(
    result: { data?: unknown; error?: unknown; response?: Response },
    opts_?: { suppressNotify?: boolean },
  ): Promise<{ state: FlowState | null; error: IamAuthError | null }> {
    if (result.error) {
      const err = flowError(result);
      if (!opts_?.suppressNotify) notify(currentState, err);
      return { state: null, error: err };
    }

    const state = result.data as FlowState;

    // Persist the (possibly rotated) token.
    if (state.flow_token) writeToken(state.flow_token);

    // On completion: hand the session to IamAuth, then clear flow state.
    if (state.status === 'completed' && state.session) {
      if (opts.auth) {
        // acceptFlowSession fetches the user profile with the new token and
        // emits SIGNED_IN, exactly like a normal sign-in would.
        await opts.auth.acceptFlowSession(state.session);
      }
      clearToken();
      notify(state, null);
      return { state, error: null };
    }

    // On expired/aborted: clear storage, emit state so UI can show restart.
    if (state.status === 'expired' || state.status === 'aborted') {
      clearToken();
    }

    notify(state, null);
    return { state, error: null };
  }

  // ---- public API ----

  const controller: FlowController = {
    get currentState(): FlowState | null {
      return currentState;
    },

    async start(params) {
      const r = await postV1AuthFlows({
        client: httpClient,
        headers: headers(),
        body: {
          kind: params.kind,
          method: params.method,
          email: params.email,
          phone: params.phone,
          provider: params.provider,
          password: params.password,
          name: params.name,
          captcha_token: params.captchaToken,
          redirect_to: params.redirectTo,
          locale: params.locale,
          invite_token: params.inviteToken,
        },
      });
      return handleState(r);
    },

    async redeemInvite(inviteToken, params) {
      return controller.start({ kind: 'signup', inviteToken, ...params });
    },

    async submit(action, payload) {
      const token = currentState?.flow_token ?? readToken();
      if (!token) {
        const err = new IamAuthError('no active flow', 'no_active_flow');
        notify(currentState, err);
        return { state: null, error: err };
      }
      const r = await postV1AuthFlowsByFlowTokenSubmit({
        client: httpClient,
        headers: headers(),
        path: { flow_token: token },
        body: { action, payload },
      });
      return handleState(r);
    },

    async resend() {
      const token = currentState?.flow_token ?? readToken();
      if (!token) {
        const err = new IamAuthError('no active flow', 'no_active_flow');
        notify(currentState, err);
        return { state: null, error: err };
      }
      const r = await postV1AuthFlowsByFlowTokenResend({
        client: httpClient,
        headers: headers(),
        path: { flow_token: token },
      });
      return handleState(r);
    },

    async resume() {
      // Try the HttpOnly cookie path first (works on same origin after page load).
      const cookieResult = await getV1AuthFlowsCurrent({
        client: httpClient,
        headers: headers(),
      });
      if (!cookieResult.error && cookieResult.data) {
        return handleState(cookieResult);
      }

      // Fall back to the token in localStorage (explicit token resume).
      const storedToken = readToken();
      if (storedToken) {
        return controller.resumeByToken(storedToken);
      }

      // No active flow found.
      notify(null, null);
      return { state: null, error: null };
    },

    async resumeByToken(token) {
      writeToken(token);
      const r = await getV1AuthFlowsByFlowToken({
        client: httpClient,
        headers: headers(),
        path: { flow_token: token },
      });
      return handleState(r);
    },

    async abandon() {
      const token = currentState?.flow_token ?? readToken();
      if (!token) {
        clearToken();
        notify(null, null);
        return { error: null };
      }
      const r = await deleteV1AuthFlowsByFlowToken({
        client: httpClient,
        headers: headers(),
        path: { flow_token: token },
      });
      clearToken();
      notify(null, null);
      if (r.error) return { error: flowError(r) };
      return { error: null };
    },

    onChange(cb) {
      listeners.add(cb);
      // Fire immediately with current state.
      cb(currentState, null);
      return () => {
        listeners.delete(cb);
      };
    },
  };

  return controller;
}
