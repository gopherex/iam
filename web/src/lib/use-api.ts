import { useStore } from '@nanostores/react';
import { useCallback, useEffect, useState } from 'react';
import { $environment } from '@/stores/auth';

export interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  reload: () => void;
}

/**
 * Minimal data-fetching hook: runs `fn` on mount and whenever `deps` change,
 * exposing loading/error/data plus a manual `reload` (e.g. after a mutation).
 *
 * The selected environment ($environment) is an implicit dependency of EVERY
 * query: project-admin data is env-scoped (lib/sdk.ts stamps X-Environment), so
 * switching the env in the header must re-fetch every page automatically.
 */
export function useApi<T>(fn: () => Promise<T>, deps: unknown[] = []): ApiState<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const environment = useStore($environment);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const run = useCallback(() => {
    setLoading(true);
    setError(null);
    fn()
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e : new Error(String(e))))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, environment]);

  useEffect(() => {
    run();
  }, [run]);

  return { data, loading, error, reload: run };
}
