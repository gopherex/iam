import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/states';

export function ProvidersPage() {
  return (
    <div>
      <PageHeader title="Providers" />
      <EmptyState title="Coming soon" description="This screen is being built." />
    </div>
  );
}
