import { useCallback, useEffect, useState } from 'react';

export interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  reload: () => void;
}

/**
 * Minimal data-fetching hook: runs `fn` on mount and whenever `deps` change,
 * exposing loading/error/data plus a manual `reload` (e.g. after a mutation).
 */
export function useApi<T>(fn: () => Promise<T>, deps: unknown[] = []): ApiState<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const run = useCallback(() => {
    setLoading(true);
    setError(null);
    fn()
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e : new Error(String(e))))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    run();
  }, [run]);

  return { data, loading, error, reload: run };
}
