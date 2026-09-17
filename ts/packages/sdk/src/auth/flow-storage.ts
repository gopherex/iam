import type { StorageAdapter } from './types';
import { MemoryStorage } from './storage';

/** Namespace capabilities by deployment, tenant, environment and purpose. */
export function flowStorageKey(baseUrl: string, project: string, environment: string | undefined, purpose: string): string {
  return `iam.${purpose}:${encodeURIComponent(baseUrl.replace(/\/$/, ''))}:${encodeURIComponent(project)}:${encodeURIComponent(environment ?? 'live')}`;
}

export function restrictedFlowStorage(): StorageAdapter {
  const memory = new MemoryStorage();
  let persistent: Storage | null = null;
  try { if (typeof sessionStorage !== 'undefined') persistent = sessionStorage; } catch { /* private browser */ }
  return {
    getItem(key) {
      try { return persistent ? persistent.getItem(key) : memory.getItem(key); }
      catch { persistent = null; return memory.getItem(key); }
    },
    setItem(key, value) {
      memory.setItem(key, value);
      try { persistent?.setItem(key, value); } catch { persistent = null; }
    },
    removeItem(key) {
      memory.removeItem(key);
      try { persistent?.removeItem(key); } catch { persistent = null; }
    },
  };
}
