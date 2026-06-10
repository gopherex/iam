import { getMgmtV1ProjectsByProjectIdEnvironments } from '@gopherex/iam-sdk';
import { call } from '@/lib/sdk';
import { setEnvironments } from '@/stores/auth';

/**
 * Load the project's declared environments (operator API) into the $environments
 * store that drives the header switcher. Falls back to ['live'] on any error so
 * the switcher always offers at least the seeded environment.
 */
export async function loadEnvironments(projectId: string): Promise<void> {
  try {
    const res = await call(
      getMgmtV1ProjectsByProjectIdEnvironments({ path: { project_id: projectId } }),
    );
    const names = (res.data ?? []).map((e) => e.name ?? '').filter(Boolean);
    setEnvironments(names);
  } catch {
    setEnvironments(['live']);
  }
}
