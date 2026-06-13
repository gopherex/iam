import { AlertCircle, Loader2 } from 'lucide-react';
import type { ReactNode } from 'react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';

interface DetailedError extends Error {
  code?: string;
  status?: number;
  details?: Record<string, unknown>;
}

function formatDetails(details: Record<string, unknown> | undefined) {
  if (!details || Object.keys(details).length === 0) return null;
  try {
    return JSON.stringify(details, null, 2);
  } catch {
    return String(details);
  }
}

export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-sm text-muted-foreground">
      <Loader2 className="size-4 animate-spin" />
      {label}
    </div>
  );
}

export function ErrorState({ error, onRetry }: { error: Error; onRetry?: () => void }) {
  const detailed = error as DetailedError;
  const details = formatDetails(detailed.details);
  const meta = [
    detailed.status ? `HTTP ${detailed.status}` : null,
    detailed.code ?? null,
  ].filter((item): item is string => Boolean(item));

  return (
    <Alert variant="destructive" className="my-4">
      <AlertCircle className="size-4" />
      <AlertTitle>Something went wrong</AlertTitle>
      <AlertDescription className="flex flex-col items-start gap-2">
        <span>{error.message}</span>
        {meta.length > 0 && (
          <div className="flex max-w-full flex-wrap gap-2">
            {meta.map((item) => (
              <span key={item} className="rounded border border-destructive/30 px-2 py-0.5 font-mono text-xs">
                {item}
              </span>
            ))}
          </div>
        )}
        {details && (
          <pre className="max-w-full overflow-x-auto whitespace-pre-wrap break-words rounded border border-destructive/30 bg-destructive/10 p-2 font-mono text-xs">
            {details}
          </pre>
        )}
        {onRetry && (
          <Button size="sm" variant="outline" onClick={onRetry}>
            Retry
          </Button>
        )}
      </AlertDescription>
    </Alert>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-16 text-center">
      <div className="space-y-1">
        <p className="text-sm font-medium">{title}</p>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </div>
      {action}
    </div>
  );
}
