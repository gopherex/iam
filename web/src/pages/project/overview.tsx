import { getMgmtV1ProjectsByProjectId } from '@gopherex/iam-sdk';
import type { ReactNode } from 'react';
import { useParams } from 'react-router-dom';
import { CopyButton } from '@/components/copy-button';
import { PageHeader } from '@/components/page-header';
import { ErrorState, LoadingState } from '@/components/states';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

export function ProjectOverview() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () => call(getMgmtV1ProjectsByProjectId({ path: { project_id: projectId! } })),
    [projectId],
  );
  const p = data?.project;

  return (
    <div>
      <PageHeader title={p?.name ?? 'Overview'} description="Project configuration at a glance." />
      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <InfoCard label="Project ID">{projectId && <CopyButton value={projectId} />}</InfoCard>
          <InfoCard label="Slug">{p?.slug ? `/${p.slug}` : '—'}</InfoCard>
          <InfoCard label="Default locale">{p?.default_locale ?? '—'}</InfoCard>
          <InfoCard label="Environments">
            <div className="flex flex-wrap gap-1.5">
              {(p?.environments ?? []).map((e) => (
                <Badge key={e} variant="secondary">
                  {e}
                </Badge>
              ))}
              {(p?.environments?.length ?? 0) === 0 && '—'}
            </div>
          </InfoCard>
        </div>
      )}
    </div>
  );
}

function InfoCard({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent className="text-sm">{children}</CardContent>
    </Card>
  );
}
