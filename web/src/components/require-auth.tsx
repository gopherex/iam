import { useStore } from '@nanostores/react';
import { Navigate, Outlet } from 'react-router-dom';
import { $authed } from '@/stores/auth';

export function RequireAuth() {
  const authed = useStore($authed);
  if (!authed) return <Navigate to="/login" replace />;
  return <Outlet />;
}
