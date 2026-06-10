import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { RequireAuth } from '@/components/require-auth';
import { Toaster } from '@/components/ui/sonner';
import { OperatorLayout } from '@/layout/operator-layout';
import { ProjectLayout } from '@/layout/project-layout';
import { FlowPage } from '@/pages/flow';
import { LoginPage } from '@/pages/login';
import { ProjectsPage } from '@/pages/projects';
import { AccessRequestsPage } from '@/pages/project/access-requests';
import { ApiKeysPage } from '@/pages/project/api-keys';
import { ClientsPage } from '@/pages/project/clients';
import { ConfigPage } from '@/pages/project/config';
import { ConnectionsPage } from '@/pages/project/connections';
import { DomainsPage } from '@/pages/project/domains';
import { ProvidersPage } from '@/pages/project/providers';
import { ProjectOverview } from '@/pages/project/overview';
import { ServiceAccountsPage } from '@/pages/project/service-accounts';
import { SigningKeysPage } from '@/pages/project/signing-keys';
import { UsersPage } from '@/pages/project/users';

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/flow" element={<FlowPage />} />

        <Route element={<RequireAuth />}>
          <Route element={<OperatorLayout />}>
            <Route index element={<Navigate to="/projects" replace />} />
            <Route path="projects" element={<ProjectsPage />} />
          </Route>

          <Route path="projects/:projectId" element={<ProjectLayout />}>
            <Route index element={<ProjectOverview />} />
            <Route path="users" element={<UsersPage />} />
            <Route path="clients" element={<ClientsPage />} />
            <Route path="service-accounts" element={<ServiceAccountsPage />} />
            <Route path="api-keys" element={<ApiKeysPage />} />
            <Route path="connections" element={<ConnectionsPage />} />
            <Route path="domains" element={<DomainsPage />} />
            <Route path="signing-keys" element={<SigningKeysPage />} />
            <Route path="providers" element={<ProvidersPage />} />
            <Route path="config" element={<ConfigPage />} />
            <Route path="access-requests" element={<AccessRequestsPage />} />
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <Toaster richColors position="top-right" />
    </BrowserRouter>
  );
}
