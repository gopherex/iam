import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/states';

export function AccessRequestsPage() {
  return (
    <div>
      <PageHeader title="Access Requests" />
      <EmptyState title="Coming soon" description="This screen is being built." />
    </div>
  );
}
