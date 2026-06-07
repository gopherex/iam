import { useStore } from '@nanostores/react';
import {
  getMgmtV1ProjectsByProjectId,
  postMgmtV1ProjectsByProjectIdAdminTokens,
} from '@gopherex/iam-sdk';
import {
  AppWindow,
  Bot,
  Globe,
  Inbox,
  KeyRound,
  KeySquare,
  LayoutDashboard,
  Network,
  Send,
  Settings2,
  Users,
} from 'lucide-react';
import { useEffect, useState } from 'react';
import { Navigate, useParams } from 'react-router-dom';
import { AccountMenu } from '@/components/account-menu';
import { ProjectSwitcher } from '@/components/project-switcher';
import { ErrorState, LoadingState } from '@/components/states';
import { AppShell, type NavItem } from '@/layout/app-shell';
import { call } from '@/lib/sdk';
import { $adminToken, $project, setProjectContext } from '@/stores/auth';

function projectNav(id: string): NavItem[] {
  const base = `/projects/${id}`;
  return [
    { label: 'Overview', to: base, icon: LayoutDashboard, end: true },
    { label: 'Users', to: `${base}/users`, icon: Users },
    { label: 'App Clients', to: `${base}/clients`, icon: AppWindow },
    { label: 'Service Accounts', to: `${base}/service-accounts`, icon: Bot },
    { label: 'API Keys', to: `${base}/api-keys`, icon: KeyRound },
    { label: 'SSO Connections', to: `${base}/connections`, icon: Network },
    { label: 'Domains', to: `${base}/domains`, icon: Globe },
    { label: 'Signing Keys', to: `${base}/signing-keys`, icon: KeySquare },
    { label: 'Providers', to: `${base}/providers`, icon: Send },
    { label: 'Configuration', to: `${base}/config`, icon: Settings2 },
    { label: 'Access Requests', to: `${base}/access-requests`, icon: Inbox },
  ];
}

export function ProjectLayout() {
  const { projectId } = useParams();
  const adminToken = useStore($adminToken);
  const project = useStore($project);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!projectId) return;
    if (project?.id === projectId && adminToken) return;
    let cancelled = false;
    (async () => {
      try {
        const p = await call(getMgmtV1ProjectsByProjectId({ path: { project_id: projectId } }));
        const tk = await call(
          postMgmtV1ProjectsByProjectIdAdminTokens({
            path: { project_id: projectId },
            body: { name: 'admin-panel' },
          }),
        );
        if (cancelled) return;
        setProjectContext(
          { id: projectId, name: p.project?.name ?? projectId },
          tk.admin_token ?? null,
        );
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e : new Error(String(e)));
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  if (!projectId) return <Navigate to="/projects" replace />;
  if (error) {
    return (
      <div className="mx-auto max-w-2xl p-8">
        <ErrorState error={error} />
      </div>
    );
  }
  if (!adminToken || project?.id !== projectId) {
    return (
      <div className="grid min-h-svh place-items-center">
        <LoadingState label="Opening project…" />
      </div>
    );
  }

  return (
    <AppShell
      nav={projectNav(projectId)}
      topLeft={<ProjectSwitcher />}
      topRight={<AccountMenu />}
    />
  );
}
