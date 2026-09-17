import { createClient, createConfig, type Client } from '@hey-api/client-fetch';
import {
  listSecurityIncidents, getSecurityIncident, listSecurityDevices,
  registerSecurityDevice, updateSecurityDevice, startSecurityFlow,
  startSecurityRecovery, exchangeSecurityContinuation, getSecurityFlow,
  submitSecurityFlow, resendSecurityFlow,
  type SecurityFlowInput, type SecurityFlowState, type SecurityDeviceInput,
} from '../gen';
import type { IamAuth } from './client';
import { IamAuthError, type IamClientOptions, type StorageAdapter } from './types';
import { flowStorageKey, restrictedFlowStorage } from './flow-storage';

export type SecurityAction = 'cancel_deletion' | 'authorize_deletion' | 'begin_passkey' | 'finish_passkey' | 'set_phone' | 'verify_phone' | 'complete_recovery' | 'trust_device' | 'select_factor' | 'verify_code' | 'verify_mfa' | 'verify_recovery_code' |
  'confirm_activity' | 'report_activity' | 'set_password' | 'request_support' |
  'support_message' | 'abandon';
export type SecurityFlowResult = { state: SecurityFlowState | null; error: IamAuthError | null };
export type SecurityFlowListener = (state: SecurityFlowState | null, error: IamAuthError | null) => void;

function securityError(r: { error?: unknown; response?: Response }): IamAuthError | null {
  if (!r.error) return null;
  const e = r.error as { error?: { code?: string; message?: string } };
  return new IamAuthError(e.error?.message ?? 'Security request failed', e.error?.code ?? 'request_failed', r.response?.status);
}

/** A separate instance of the server-owned flow pattern. It cannot accept a
 * login session, and continues working after the ordinary session is revoked. */
export class SecurityFlowController {
  private state: SecurityFlowState | null = null;
  private listeners = new Set<SecurityFlowListener>();
  private storage: StorageAdapter;
  private key: string;
  private publicClient: Client;
  private queue: Promise<unknown> = Promise.resolve();

  constructor(private opts: IamClientOptions, private auth: IamAuth) {
    this.storage = opts.securityStorage ?? restrictedFlowStorage();
    this.key = flowStorageKey(opts.baseUrl, opts.clientId, opts.environment, 'security.flow');
    this.publicClient = createClient(createConfig({ baseUrl: opts.baseUrl }));
  }

  get currentState(): SecurityFlowState | null { return this.state; }
  onChange(cb: SecurityFlowListener): () => void {
    this.listeners.add(cb); cb(this.state, null);
    return () => this.listeners.delete(cb);
  }
  private notify(error: IamAuthError | null): void {
    for (const cb of this.listeners) cb(this.state, error);
  }
  private headers(token?: string) {
    return { 'X-Client-Id': this.opts.clientId, 'X-Environment': this.opts.environment ?? 'live', 'X-Security-Token': token ?? '' };
  }
  private async handle(r: { data?: unknown; error?: unknown; response?: Response }): Promise<SecurityFlowResult> {
    const error = securityError(r);
    if (error) {
      if (error.code === 'flow_expired' || error.code === 'flow_not_found') {
        await this.storage.removeItem(this.key); this.state = null;
      }
      this.notify(error); return { state: this.state, error };
    }
    const state = r.data as SecurityFlowState;
    if (!state || !Array.isArray(state.next_actions) || typeof state.version !== 'number') {
      const error = new IamAuthError('Invalid security flow response', 'invalid_response');
      this.notify(error); return { state: this.state, error };
    }
    this.state = state;
    if (state.outcome === "sign_in_approved" && state.flow_token) this.auth.setSecuritySignInProof(state.flow_token);
    if (state.status === 'pending' && state.flow_token) await this.storage.setItem(this.key, state.flow_token);
    else await this.storage.removeItem(this.key);
    if (state.revoked_session_ids.length > 0) await this.auth.clearRevokedSecuritySession(state.revoked_session_ids);
    const proofError = state.error_code ? new IamAuthError(state.error_code.replaceAll('_', ' '), state.error_code) : null;
    this.notify(proofError); return { state, error: proofError };
  }
  private serial(action: () => Promise<SecurityFlowResult>): Promise<SecurityFlowResult> {
    const run = this.queue.then(action, action).catch((cause: unknown) => {
      const error = cause instanceof IamAuthError ? cause : new IamAuthError(cause instanceof Error ? cause.message : 'Network request failed', 'network_error');
      this.notify(error); return { state: this.state, error };
    });
    this.queue = run; return run;
  }
  start(input: { incidentId?: string; sessionId?: string; deviceId?: string } = {}): Promise<SecurityFlowResult> {
    return this.serial(async () => {
      await this.auth.ready();
      return this.handle(await startSecurityFlow({ client: this.auth.client, headers: this.headers(), body: { incident_id: input.incidentId, device_id: input.deviceId, session_id: input.sessionId } }));
    });
  }
  recover(input: { identifier: string; contact: string }): Promise<SecurityFlowResult> {
    return this.serial(async () => this.handle(await startSecurityRecovery({ client: this.publicClient, headers: this.headers(), body: input })));
  }
  private async exchangePending(pending: { token: string; key: string }): Promise<SecurityFlowResult> {
    const result = await this.handle(await exchangeSecurityContinuation({ client: this.publicClient, headers: this.headers(), body: { continuation_token: pending.token, exchange_key: pending.key } }));
    if (!result.error || ['invalid_token', 'flow_expired', 'flow_not_found'].includes(result.error.code)) {
      await this.storage.removeItem(this.key + '.exchange');
    }
    return result;
  }
  private async prepareExchange(token: string): Promise<{ token: string; key: string }> {
    const stored = await this.storage.getItem(this.key + '.exchange');
    if (stored) {
      const pending = JSON.parse(stored) as { token: string; key: string };
      if (pending.token === token) return pending;
    }
    const key = Array.from(crypto.getRandomValues(new Uint8Array(32)), byte => byte.toString(16).padStart(2, '0')).join('');
    const pending = { token, key };
    const serialized = JSON.stringify(pending);
    await this.storage.setItem(this.key + '.exchange', serialized);
    if (await this.storage.getItem(this.key + '.exchange') !== serialized) throw new IamAuthError('Cannot save continuation', 'storage_error');
    await this.storage.removeItem(this.key);
    this.state = null;
    return pending;
  }
  exchange(token: string): Promise<SecurityFlowResult> {
    return this.serial(async () => this.exchangePending(await this.prepareExchange(token)));
  }
  /** Persist the exchange before removing the fragment or making a request. */
  continueFromURL(raw: string): Promise<SecurityFlowResult> {
    return this.serial(async () => {
      const url = new URL(raw);
      const token = new URLSearchParams(url.hash.slice(1)).get('iam_security');
      if (!token) return { state: this.state, error: new IamAuthError('Missing continuation', 'invalid_token') };
      const pending = await this.prepareExchange(token);
      if (typeof window !== 'undefined' && raw === window.location.href) {
        const fragment = new URLSearchParams(url.hash.slice(1));
        fragment.delete('iam_security');
        const rest = fragment.toString();
        window.history.replaceState(window.history.state, '', url.pathname + url.search + (rest ? '#' + rest : ''));
      }
      return this.exchangePending(pending);
    });
  }
  resumeByToken(token: string): Promise<SecurityFlowResult> {
    return this.serial(async () => this.handle(await getSecurityFlow({ client: this.publicClient, headers: this.headers(token) })));
  }
  resume(): Promise<SecurityFlowResult> {
    return this.serial(async () => {
      const token = this.state?.flow_token ?? await this.storage.getItem(this.key);
      if (!token) {
        const pending = await this.storage.getItem(this.key + '.exchange');
        if (pending) return this.exchangePending(JSON.parse(pending) as { token: string; key: string });
        return { state: null, error: null };
      }
      return this.handle(await getSecurityFlow({ client: this.publicClient, headers: this.headers(token) }));
    });
  }
  submit(action: SecurityAction, input: Omit<SecurityFlowInput, 'action' | 'version'> = {}): Promise<SecurityFlowResult> {
    return this.serial(async () => {
      const state = this.state;
      if (!state?.flow_token || !state.next_actions.includes(action)) {
        return { state, error: new IamAuthError('Action is not available in this flow', 'invalid_flow_action') };
      }
      return this.handle(await submitSecurityFlow({ client: this.publicClient, headers: this.headers(state.flow_token), body: { ...input, action, version: state.version } }));
    });
  }
  cancelDeletion(requestId: string): Promise<SecurityFlowResult> {
    return this.submit('cancel_deletion', { request_id: requestId });
  }
  resend(): Promise<SecurityFlowResult> {
    return this.serial(async () => {
      if (!this.state?.flow_token) return { state: this.state, error: new IamAuthError('No active flow', 'no_active_flow') };
      return this.handle(await resendSecurityFlow({ client: this.publicClient, headers: this.headers(this.state.flow_token) }));
    });
  }
  async verifyPasskey(): Promise<SecurityFlowResult> {
    const options = this.state?.challenge;
    if (!options || typeof navigator === 'undefined' || !navigator.credentials) return { state: this.state, error: new IamAuthError('Passkey challenge is unavailable', 'invalid_flow_action') };
    try {
      const decode = (value: string) => Uint8Array.from(atob(value.replaceAll('-', '+').replaceAll('_', '/')), c => c.charCodeAt(0));
      const encode = (value: ArrayBuffer) => btoa(String.fromCharCode(...new Uint8Array(value))).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
      const publicKey = { ...options, challenge: decode(options.challenge as string), allowCredentials: (options.allowCredentials as Array<{id:string;type:PublicKeyCredentialType}> | undefined)?.map(c => ({ ...c, id: decode(c.id) })) } as PublicKeyCredentialRequestOptions;
      const result = await navigator.credentials.get({ publicKey }) as PublicKeyCredential | null;
      if (!result) throw new Error('Passkey verification cancelled');
      const response = result.response as AuthenticatorAssertionResponse;
      return this.submit('verify_mfa', { factor_id: 'passkey', credential: { id: result.id, rawId: encode(result.rawId), type: result.type, response: { authenticatorData: encode(response.authenticatorData), clientDataJSON: encode(response.clientDataJSON), signature: encode(response.signature), userHandle: response.userHandle ? encode(response.userHandle) : null }, clientExtensionResults: result.getClientExtensionResults() } });
    } catch (e) { return { state: this.state, error: new IamAuthError(e instanceof Error ? e.message : 'Passkey verification failed', 'passkey_failed') }; }
  }
  async registerRecoveryPasskey(): Promise<SecurityFlowResult> {
    const started = await this.submit('begin_passkey');
    if (started.error || !started.state?.challenge) return started;
    try {
      const options = started.state.challenge;
      const decode = (value: string) => Uint8Array.from(atob(value.replaceAll('-', '+').replaceAll('_', '/')), c => c.charCodeAt(0));
      const encode = (value: ArrayBuffer) => btoa(String.fromCharCode(...new Uint8Array(value))).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
      const user = options.user as { id: string; name: string; displayName: string };
      const publicKey = { ...options, rp: options.rp as PublicKeyCredentialRpEntity, pubKeyCredParams: options.pubKeyCredParams as PublicKeyCredentialParameters[], excludeCredentials: (options.excludeCredentials as Array<{id:string;type:PublicKeyCredentialType}> | undefined)?.map(c => ({ ...c, id: decode(c.id) })), challenge: decode(options.challenge as string), user: { ...user, id: decode(user.id) } } as PublicKeyCredentialCreationOptions;
      const credential = await navigator.credentials.create({ publicKey }) as PublicKeyCredential | null;
      if (!credential) throw new Error('Passkey registration cancelled');
      const response = credential.response as AuthenticatorAttestationResponse;
      return this.submit('finish_passkey', { credential: { id: credential.id, rawId: encode(credential.rawId), type: credential.type, response: { attestationObject: encode(response.attestationObject), clientDataJSON: encode(response.clientDataJSON), transports: response.getTransports?.() ?? [] }, clientExtensionResults: credential.getClientExtensionResults() } });
    } catch (error) { return { state: this.state, error: new IamAuthError(error instanceof Error ? error.message : 'Passkey registration failed', 'passkey_failed') }; }
  }
  confirmActivity() { return this.submit('confirm_activity'); }
  reportActivity(options: { allSessions?: boolean; sessionId?: string } = {}) { return this.submit('report_activity', { all_sessions: options.allSessions ?? false, session_id: options.sessionId }); }
  verifyCode(code: string) { return this.submit('verify_code', { code }); }
  setPassword(newPassword: string) { return this.submit('set_password', { new_password: newPassword }); }
  requestSupport(input: { contact?: string; identifier?: string; message?: string } = {}) { return this.submit('request_support', input); }
  abandon() { return this.submit('abandon'); }
}

export class IamSecurity {
  readonly flow: SecurityFlowController;
  constructor(private opts: IamClientOptions, private auth: IamAuth) { this.flow = new SecurityFlowController(opts, auth); }
  async listIncidents(query?: { cursor?: string; limit?: number }) {
    await this.auth.ready(); const r = await listSecurityIncidents({ client: this.auth.client, headers: this.auth.headers(), query });
    return { data: r.data ?? null, error: securityError(r) };
  }
  async getIncident(id: string) {
    const r = await getSecurityIncident({ client: this.auth.client, headers: this.auth.headers(), path: { incident_id: id } });
    return { data: r.data ?? null, error: securityError(r) };
  }
  async listDevices(query?: { cursor?: string; limit?: number }) {
    const r = await listSecurityDevices({ client: this.auth.client, headers: this.auth.headers(), query });
    return { data: r.data ?? null, error: securityError(r) };
  }
  async registerDevice(name = this.opts.deviceName ?? '') {
    const r = await registerSecurityDevice({ client: this.auth.client, headers: this.auth.headers(), body: { name } });
    if (r.data) await this.auth.rememberSecurityDevice(r.data.device_token);
    return { data: r.data ?? null, error: securityError(r) };
  }
  async updateDevice(id: string, input: SecurityDeviceInput) {
    const r = await updateSecurityDevice({ client: this.auth.client, headers: this.auth.headers(), path: { device_id: id }, body: input });
    if (r.data && input.action === 'revoke') await this.auth.clearRevokedSecuritySession(r.data.session_ids, r.data.current);
    return { data: r.data ?? null, error: securityError(r) };
  }
}
