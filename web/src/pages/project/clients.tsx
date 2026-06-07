import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/states';

export function ClientsPage() {
  return (
    <div>
      <PageHeader title="App Clients" />
      <EmptyState title="Coming soon" description="This screen is being built." />
    </div>
  );
}
