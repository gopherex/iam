import { atom, computed } from 'nanostores';

// The admin panel authenticates as the operator with the master key (Bearer on
// /mgmt/* operator calls), and per project with an admin token (Bearer on the
// project's /v1/.../admin/* calls). The master key is persisted; the admin token
// is held in memory for the active project.

const MASTER_KEY = 'iam.master_key';

export interface ProjectRef {
  id: string;
  name: string;
}

export const $masterKey = atom<string | null>(localStorage.getItem(MASTER_KEY));
export const $project = atom<ProjectRef | null>(null);
export const $adminToken = atom<string | null>(null);

export const $authed = computed($masterKey, (k) => !!k);

export function setMasterKey(key: string | null): void {
  $masterKey.set(key);
  if (key) localStorage.setItem(MASTER_KEY, key);
  else localStorage.removeItem(MASTER_KEY);
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
export function authSnapshot(): { masterKey: string | null; adminToken: string | null } {
  return { masterKey: $masterKey.get(), adminToken: $adminToken.get() };
}
