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
    let cancelled = false;

    // Mint (or re-mint) the project admin token. The token has a 1h TTL, so a
    // long-lived panel session must re-mint before it expires or project calls
    // start failing with 401. We re-mint on mount and then on a timer well
    // inside the TTL.
    const TTL_MS = 60 * 60 * 1000;
    const REMINT_MS = 50 * 60 * 1000;
    const mint = async (withProject: boolean) => {
      try {
        const name = withProject
          ? (await call(getMgmtV1ProjectsByProjectId({ path: { project_id: projectId } }))).project
              ?.name ?? projectId
          : null;
        const tk = await call(
          postMgmtV1ProjectsByProjectIdAdminTokens({
            path: { project_id: projectId },
            body: {
              name: 'admin-panel',
              scopes: ['admin:ui'],
              expires_at: new Date(Date.now() + TTL_MS).toISOString(),
            },
          }),
        );
        if (cancelled) return;
        if (withProject) setProjectContext({ id: projectId, name: name ?? projectId }, tk.admin_token ?? null);
        else $adminToken.set(tk.admin_token ?? null);
      } catch (e) {
        if (!cancelled && withProject) setError(e instanceof Error ? e : new Error(String(e)));
      }
    };

    if (!(project?.id === projectId && adminToken)) {
      void mint(true);
    }
    const timer = window.setInterval(() => void mint(false), REMINT_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
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
