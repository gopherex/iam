import type { StorageAdapter } from './types';

/** In-memory fallback used when no Web Storage is available (SSR / Node). */
export class MemoryStorage implements StorageAdapter {
  private store = new Map<string, string>();
  getItem(key: string): string | null {
    return this.store.has(key) ? (this.store.get(key) as string) : null;
  }
  setItem(key: string, value: string): void {
    this.store.set(key, value);
  }
  removeItem(key: string): void {
    this.store.delete(key);
  }
}

/** Returns localStorage when usable, otherwise an in-memory store. */
export function defaultStorage(): StorageAdapter {
  try {
    if (typeof globalThis !== 'undefined' && (globalThis as any).localStorage) {
      const ls = (globalThis as any).localStorage as StorageAdapter;
      // Probe (Safari private mode throws on setItem).
      const probe = '__iam_probe__';
      ls.setItem(probe, '1');
      ls.removeItem(probe);
      return ls;
    }
  } catch {
    // fall through to memory
  }
  return new MemoryStorage();
}
