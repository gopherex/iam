import { atom, computed } from 'nanostores';

// The admin panel authenticates as the operator with the master key (Bearer on
// /mgmt/* operator calls), and per project with an admin token (Bearer on the
// project's /v1/.../admin/* calls). Both credentials are held in memory only.

export interface ProjectRef {
  id: string;
  name: string;
}

export const $masterKey = atom<string | null>(null);
export const $project = atom<ProjectRef | null>(null);
export const $adminToken = atom<string | null>(null);

export const $authed = computed($masterKey, (k) => !!k);

export function setMasterKey(key: string | null): void {
	$masterKey.set(key);
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
