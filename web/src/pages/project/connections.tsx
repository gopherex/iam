import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/states';

export function ConnectionsPage() {
  return (
    <div>
      <PageHeader title="SSO Connections" />
      <EmptyState title="Coming soon" description="This screen is being built." />
    </div>
  );
}
