import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/states';

export function DomainsPage() {
  return (
    <div>
      <PageHeader title="Domains" />
      <EmptyState title="Coming soon" description="This screen is being built." />
    </div>
  );
}
