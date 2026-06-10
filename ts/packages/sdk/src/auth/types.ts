import type { ConsentDocRef, Factor, NextStep, User } from '../gen';

/**
 * A pluggable key/value store for persisting the session. Browser builds default
 * to localStorage; pass your own for SSR / React Native / Node. Methods may be
 * sync or async.
 */
export interface StorageAdapter {
  getItem(key: string): string | null | Promise<string | null>;
  setItem(key: string, value: string): void | Promise<void>;
  removeItem(key: string): void | Promise<void>;
}

/** The persisted session: the tokens plus the resolved user and absolute expiry. */
export interface Session {
  access_token: string;
  refresh_token?: string;
  token_type: string;
  /** Absolute expiry in epoch milliseconds (derived from expires_in at issue). */
  expires_at: number;
  user: User;
}

/** Auth lifecycle events, emitted to onAuthStateChange subscribers. */
export type AuthChangeEvent =
  | 'INITIAL_SESSION'
  | 'SIGNED_IN'
  | 'SIGNED_OUT'
  | 'TOKEN_REFRESHED'
  | 'USER_UPDATED';

/** A stable auth error carrying the API's error code where available. */
export class IamAuthError extends Error {
  code: string;
  status?: number;
  constructor(message: string, code = 'auth_error', status?: number) {
    super(message);
    this.name = 'IamAuthError';
    this.code = code;
    this.status = status;
  }
}

/**
 * The result of an auth action. On success `data.session`/`data.user` are set;
 * when the server requires a further step (MFA, email verification, …) `session`
 * is null and `nextStep` (plus `factors`/`flowToken`) describe what to do next.
 */
export interface AuthData {
  session: Session | null;
  user: User | null;
  nextStep?: NextStep;
  factors?: Array<Factor>;
  flowToken?: string;
  documents?: Array<ConsentDocRef>;
}

export type AuthResponse =
  | { data: AuthData; error: null }
  | { data: { session: null; user: null }; error: IamAuthError };

/** Unsubscribe handle returned by onAuthStateChange. */
export interface Subscription {
  unsubscribe(): void;
}

export interface IamClientOptions {
  /** API base URL, e.g. https://iam.acme.com */
  baseUrl: string;
  /** Public client id sent as X-Client-Id on auth calls. */
  clientId: string;
  /**
   * Project environment sent as X-Environment on every auth call, giving
   * Stripe-like test/live data isolation (e.g. "test", "live", "staging").
   * Defaults to the project's "live" environment when omitted.
   */
  environment?: string;
  /** Session store (default: localStorage in the browser, in-memory otherwise). */
  storage?: StorageAdapter;
  /** Storage key for the persisted session (default: "iam.session"). */
  storageKey?: string;
  /** Persist the session across reloads (default: true). */
  persistSession?: boolean;
  /** Refresh the access token before it expires (default: true). */
  autoRefresh?: boolean;
  /** Refresh this many seconds before expiry (default: 30). */
  refreshMarginSeconds?: number;
  /** Sync sign-in/out/refresh across browser tabs (default: true). */
  multiTab?: boolean;
}
