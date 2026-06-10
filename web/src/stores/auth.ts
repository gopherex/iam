import { atom, computed } from 'nanostores';

// The admin panel authenticates as the operator with the master key (Bearer on
// /mgmt/* operator calls), and per project with an admin token (Bearer on the
// project's /v1/.../admin/* calls).
//
// The master key is persisted in sessionStorage so a page reload does not drop
// the operator out to /login. sessionStorage (not localStorage) is deliberate:
// it survives reloads within the tab but is cleared when the tab closes, so the
// god-credential is not left at rest across browser sessions or shared between
// tabs. The per-project admin token stays in memory only — it is short-lived
// (1h) and re-minted on demand, so persisting it buys nothing.

const MASTER_KEY_STORAGE = 'iam.masterKey';
const ENVIRONMENT_STORAGE = 'iam.environment';
const DEFAULT_ENVIRONMENT = 'live';

function readStoredMasterKey(): string | null {
  try {
    return sessionStorage.getItem(MASTER_KEY_STORAGE);
  } catch {
    return null;
  }
}

function writeStoredMasterKey(key: string | null): void {
  try {
    if (key) sessionStorage.setItem(MASTER_KEY_STORAGE, key);
    else sessionStorage.removeItem(MASTER_KEY_STORAGE);
  } catch {
    // Storage unavailable (private mode / disabled) — fall back to memory only.
  }
}

function readStoredEnvironment(): string {
  try {
    return sessionStorage.getItem(ENVIRONMENT_STORAGE) ?? DEFAULT_ENVIRONMENT;
  } catch {
    return DEFAULT_ENVIRONMENT;
  }
}

function writeStoredEnvironment(env: string): void {
  try {
    sessionStorage.setItem(ENVIRONMENT_STORAGE, env);
  } catch {
    // Storage unavailable (private mode / disabled) — fall back to memory only.
  }
}

export interface ProjectRef {
  id: string;
  name: string;
}

export const $masterKey = atom<string | null>(readStoredMasterKey());
export const $project = atom<ProjectRef | null>(null);
export const $adminToken = atom<string | null>(null);

// The environment (live / staging / dev) project-admin data views operate in,
// giving Stripe-like test/live data isolation. Persisted in sessionStorage like
// the master key so a reload keeps the selected environment within the tab.
export const $environment = atom<string>(readStoredEnvironment());

// The project's actually-declared environments (from the operator API), driving
// the header switcher. Defaults to just live until loaded — the switcher must
// only offer environments that exist, or the server rejects X-Environment (400).
export const $environments = atom<string[]>(['live']);

export const $authed = computed($masterKey, (k) => !!k);

export function setEnvironments(names: string[]): void {
  const list = names.length ? names : ['live'];
  $environments.set(list);
  // If the selected env no longer exists, snap back to a valid one.
  if (!list.includes($environment.get())) setEnvironment(list.includes('live') ? 'live' : list[0]);
}

export function setMasterKey(key: string | null): void {
	$masterKey.set(key);
	writeStoredMasterKey(key);
}

export function setEnvironment(env: string): void {
  $environment.set(env);
  writeStoredEnvironment(env);
}

export function setProjectContext(project: ProjectRef | null, adminToken: string | null): void {
  $project.set(project);
  $adminToken.set(adminToken);
}

export function logout(): void {
  setMasterKey(null);
  setProjectContext(null, null);
}

/** Non-reactive snapshot for the request interceptor. */
export function authSnapshot(): {
  masterKey: string | null;
  adminToken: string | null;
  environment: string;
} {
  return {
    masterKey: $masterKey.get(),
    adminToken: $adminToken.get(),
    environment: $environment.get(),
  };
}
