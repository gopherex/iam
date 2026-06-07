import { FolderKanban } from 'lucide-react';
import { AccountMenu } from '@/components/account-menu';
import { AppShell, type NavItem } from '@/layout/app-shell';

const nav: NavItem[] = [{ label: 'Projects', to: '/projects', icon: FolderKanban }];

export function OperatorLayout() {
  return <AppShell nav={nav} topRight={<AccountMenu />} />;
}
