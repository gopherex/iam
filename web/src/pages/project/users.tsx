import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/states';

export function UsersPage() {
  return (
    <div>
      <PageHeader title="Users" />
      <EmptyState title="Coming soon" description="This screen is being built." />
    </div>
  );
}
