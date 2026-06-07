import type { LucideIcon } from 'lucide-react';
import { ShieldCheck } from 'lucide-react';
import type { ReactNode } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { ThemeToggle } from '@/components/theme-toggle';
import { cn } from '@/lib/utils';

export interface NavItem {
  label: string;
  to: string;
  icon: LucideIcon;
  end?: boolean;
}

export function AppShell({
  nav,
  sidebarHeader,
  topLeft,
  topRight,
}: {
  nav: NavItem[];
  sidebarHeader?: ReactNode;
  topLeft?: ReactNode;
  topRight?: ReactNode;
}) {
  return (
    <div className="grid min-h-svh grid-cols-[15rem_minmax(0,1fr)]">
      <aside className="flex flex-col border-r bg-sidebar text-sidebar-foreground">
        <div className="flex h-14 items-center gap-2 border-b px-4">
          <div className="flex size-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <ShieldCheck className="size-4" />
          </div>
          <span className="font-heading text-sm font-semibold tracking-tight">IAM Console</span>
        </div>
        {sidebarHeader && <div className="border-b px-3 py-3">{sidebarHeader}</div>}
        <nav className="flex-1 space-y-0.5 overflow-y-auto p-3">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                    : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
                )
              }
            >
              <item.icon className="size-4 shrink-0" />
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="flex min-w-0 flex-col">
        <header className="sticky top-0 z-10 flex h-14 items-center justify-between gap-4 border-b bg-background/85 px-6 backdrop-blur">
          <div className="flex min-w-0 items-center gap-3">{topLeft}</div>
          <div className="flex items-center gap-1">
            {topRight}
            <ThemeToggle />
          </div>
        </header>
        <main className="min-w-0 flex-1 overflow-auto">
          <div className="mx-auto max-w-7xl p-6 lg:p-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
