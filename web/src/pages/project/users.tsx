import {
  deleteV1ProjectsByProjectIdAdminUsersByUserId,
  deleteV1ProjectsByProjectIdAdminUsersByUserIdGrantsByGrantId,
  deleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityId,
  deleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionId,
  getV1ProjectsByProjectIdAdminUsers,
  getV1ProjectsByProjectIdAdminUsersByUserId,
  getV1ProjectsByProjectIdAdminUsersByUserIdGrants,
  getV1ProjectsByProjectIdAdminUsersByUserIdIdentities,
  getV1ProjectsByProjectIdAdminUsersByUserIdSessions,
  patchV1ProjectsByProjectIdAdminUsersByUserId,
  postV1ProjectsByProjectIdAdminUsers,
  postV1ProjectsByProjectIdAdminUsersByUserIdAnonymize,
  postV1ProjectsByProjectIdAdminUsersByUserIdBan,
  postV1ProjectsByProjectIdAdminUsersByUserIdImpersonate,
  postV1ProjectsByProjectIdAdminUsersByUserIdMfaReset,
  postV1ProjectsByProjectIdAdminUsersByUserIdPassword,
  postV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevoke,
  postV1ProjectsByProjectIdAdminUsersByUserIdUnban,
  postV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmail,
  postV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhone,
} from '@gopherex/iam-sdk';
import type {
  GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsResponse,
  Identity,
  OAuthGrant,
  User,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  Ban,
  ExternalLink,
  Fingerprint,
  KeyRound,
  Loader2,
  MoreHorizontal,
  Plus,
  RefreshCw,
  ShieldOff,
  Trash2,
  UserCheck,
  UserCog,
  UserX,
} from 'lucide-react';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';
import { CopyButton } from '@/components/copy-button';
import { DataTable } from '@/components/data-table';
import { PageHeader } from '@/components/page-header';
import { EmptyState, ErrorState, LoadingState } from '@/components/states';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// The SDK re-exports an auth-layer `Session` type that shadows the generated one.
// Derive the admin session row type from the list-sessions response to avoid the collision.
type AdminSession = NonNullable<
  NonNullable<GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsResponse>['data']
>[number];

// ─── helpers ────────────────────────────────────────────────────────────────

function fmtDate(ts?: string | null): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function statusVariant(status: User['status']): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'active':
      return 'secondary';
    case 'banned':
      return 'destructive';
    case 'suspended':
      return 'outline';
    case 'deactivated':
      return 'outline';
    default:
      return 'secondary';
  }
}

// ─── main page ───────────────────────────────────────────────────────────────

export function UsersPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminUsers({ path: { project_id: projectId! } })),
    [projectId],
  );
  const users = data?.data ?? [];

  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  const columns: ColumnDef<User, unknown>[] = [
    {
      accessorKey: 'id',
      header: 'ID',
      cell: ({ row }) => <CopyButton value={row.original.id} />,
    },
    {
      accessorFn: (u) => u.primary_email ?? u.primary_phone ?? '—',
      id: 'contact',
      header: 'Email / Phone',
      cell: ({ row }) => {
        const u = row.original;
        return (
          <span className="text-sm">
            {u.primary_email ?? u.primary_phone ?? <span className="text-muted-foreground">—</span>}
          </span>
        );
      },
    },
    {
      accessorKey: 'kind',
      header: 'Kind',
      cell: ({ row }) => (
        <Badge variant="outline" className="capitalize">
          {row.original.kind}
        </Badge>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => (
        <Badge variant={statusVariant(row.original.status)} className="capitalize">
          {row.original.status}
        </Badge>
      ),
    },
    {
      accessorFn: (u) => u.created_at ?? '',
      id: 'created_at',
      header: 'Created',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{fmtDate(row.original.created_at)}</span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <UserActionsMenu
          user={row.original}
          projectId={projectId!}
          onReload={reload}
          onViewDetail={setSelectedUser}
        />
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Users"
        description="All users in this project. Manage identities, sessions, and lifecycle actions."
        actions={<CreateUserDialog projectId={projectId!} onCreated={reload} />}
      />

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && users.length === 0 && (
        <EmptyState
          title="No users yet"
          description="Create a user or wait for sign-ups."
          action={<CreateUserDialog projectId={projectId!} onCreated={reload} />}
        />
      )}
      {!loading && !error && users.length > 0 && (
        <DataTable
          columns={columns}
          data={users}
          searchable
          searchPlaceholder="Search users…"
          emptyMessage="No users match your search."
        />
      )}

      {selectedUser && (
        <UserDetailDialog
          user={selectedUser}
          projectId={projectId!}
          onClose={() => setSelectedUser(null)}
          onReload={reload}
        />
      )}
    </div>
  );
}

// ─── row actions dropdown ────────────────────────────────────────────────────

interface UserActionsMenuProps {
  user: User;
  projectId: string;
  onReload: () => void;
  onViewDetail: (u: User) => void;
}

function UserActionsMenu({ user, projectId, onReload, onViewDetail }: UserActionsMenuProps) {
  const [setPasswordOpen, setSetPasswordOpen] = useState(false);
  const [banOpen, setBanOpen] = useState(false);
  const [impersonateOpen, setImpersonateOpen] = useState(false);
  const [anonymizeOpen, setAnonymizeOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  async function handleUnban() {
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdUnban({
          path: { project_id: projectId, user_id: user.id },
        }),
      );
      toast.success('User unbanned.');
      onReload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to unban user');
    }
  }

  async function handleVerifyEmail() {
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmail({
          path: { project_id: projectId, user_id: user.id },
        }),
      );
      toast.success('Email marked verified.');
      onReload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to verify email');
    }
  }

  async function handleVerifyPhone() {
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhone({
          path: { project_id: projectId, user_id: user.id },
        }),
      );
      toast.success('Phone marked verified.');
      onReload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to verify phone');
    }
  }

  async function handleMfaReset() {
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdMfaReset({
          path: { project_id: projectId, user_id: user.id },
        }),
      );
      toast.success('MFA factors reset.');
      onReload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to reset MFA');
    }
  }

  async function handleRevokeAllSessions() {
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevoke({
          path: { project_id: projectId, user_id: user.id },
        }),
      );
      toast.success('All sessions revoked.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to revoke sessions');
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" />}>
          <MoreHorizontal className="size-4" />
          <span className="sr-only">Open menu</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>User</DropdownMenuLabel>
          <DropdownMenuItem onClick={() => onViewDetail(user)}>
            <UserCog className="size-4" />
            View detail
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuLabel>Identity</DropdownMenuLabel>
          <DropdownMenuItem onClick={() => setSetPasswordOpen(true)}>
            <KeyRound className="size-4" />
            Set password
          </DropdownMenuItem>
          <DropdownMenuItem onClick={handleMfaReset}>
            <ShieldOff className="size-4" />
            Reset MFA
          </DropdownMenuItem>
          {user.primary_email && !user.email_verified && (
            <DropdownMenuItem onClick={handleVerifyEmail}>
              <UserCheck className="size-4" />
              Mark email verified
            </DropdownMenuItem>
          )}
          {user.primary_phone && !user.phone_verified && (
            <DropdownMenuItem onClick={handleVerifyPhone}>
              <UserCheck className="size-4" />
              Mark phone verified
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuLabel>Sessions</DropdownMenuLabel>
          <DropdownMenuItem onClick={handleRevokeAllSessions}>
            <RefreshCw className="size-4" />
            Revoke all sessions
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuLabel>Lifecycle</DropdownMenuLabel>
          <DropdownMenuItem onClick={() => setImpersonateOpen(true)}>
            <ExternalLink className="size-4" />
            Impersonate
          </DropdownMenuItem>
          {user.status === 'banned' ? (
            <DropdownMenuItem onClick={handleUnban}>
              <UserCheck className="size-4" />
              Unban
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => setBanOpen(true)}>
              <Ban className="size-4" />
              Ban
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => setAnonymizeOpen(true)} variant="destructive">
            <UserX className="size-4" />
            Anonymize
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setDeleteOpen(true)} variant="destructive">
            <Trash2 className="size-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <SetPasswordDialog
        open={setPasswordOpen}
        onOpenChange={setSetPasswordOpen}
        user={user}
        projectId={projectId}
      />
      <BanDialog
        open={banOpen}
        onOpenChange={setBanOpen}
        user={user}
        projectId={projectId}
        onBanned={onReload}
      />
      <ImpersonateDialog
        open={impersonateOpen}
        onOpenChange={setImpersonateOpen}
        user={user}
        projectId={projectId}
      />
      <AnonymizeDialog
        open={anonymizeOpen}
        onOpenChange={setAnonymizeOpen}
        user={user}
        projectId={projectId}
        onDone={onReload}
      />
      <DeleteUserDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={user}
        projectId={projectId}
        onDeleted={onReload}
      />
    </>
  );
}

// ─── create user dialog ───────────────────────────────────────────────────────

function CreateUserDialog({ projectId, onCreated }: { projectId: string; onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [kind, setKind] = useState<'human' | 'guest' | 'system'>('human');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsers({
          path: { project_id: projectId },
          body: {
            email: email || undefined,
            phone: phone || undefined,
            password: password || undefined,
            kind,
          },
        }),
      );
      toast.success('User created.');
      setOpen(false);
      setEmail('');
      setPhone('');
      setPassword('');
      setKind('human');
      onCreated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create user');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <Plus className="size-4" />
        New user
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create user</DialogTitle>
          <DialogDescription>
            Provision a new user in this project. Email or phone is required for human accounts.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cu-email">Email</Label>
            <Input
              id="cu-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="user@example.com"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cu-phone">Phone</Label>
            <Input
              id="cu-phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+1 555 000 0000"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cu-password">Password</Label>
            <Input
              id="cu-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="optional"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cu-kind">Kind</Label>
            <Select value={kind} onValueChange={(v) => setKind(v as typeof kind)}>
              <SelectTrigger id="cu-kind" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="human">human</SelectItem>
                <SelectItem value="guest">guest</SelectItem>
                <SelectItem value="system">system</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || (!email && !phone)}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create user
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ─── user detail dialog ───────────────────────────────────────────────────────

interface UserDetailDialogProps {
  user: User;
  projectId: string;
  onClose: () => void;
  onReload: () => void;
}

function UserDetailDialog({ user, projectId, onClose, onReload }: UserDetailDialogProps) {
  const [tab, setTab] = useState<'info' | 'sessions' | 'identities' | 'grants'>('info');
  const [open, setOpen] = useState(true);

  function handleOpenChange(v: boolean) {
    setOpen(v);
    if (!v) onClose();
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>User detail</DialogTitle>
          <DialogDescription>
            <CopyButton value={user.id} />
          </DialogDescription>
        </DialogHeader>
        <div className="flex gap-1 border-b pb-2">
          {(['info', 'sessions', 'identities', 'grants'] as const).map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={`rounded-md px-2.5 py-1 text-sm font-medium transition-colors ${
                tab === t
                  ? 'bg-muted text-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
        </div>
        {tab === 'info' && (
          <UserInfoTab user={user} projectId={projectId} onReload={onReload} />
        )}
        {tab === 'sessions' && (
          <UserSessionsTab user={user} projectId={projectId} />
        )}
        {tab === 'identities' && (
          <UserIdentitiesTab user={user} projectId={projectId} />
        )}
        {tab === 'grants' && (
          <UserGrantsTab user={user} projectId={projectId} />
        )}
      </DialogContent>
    </Dialog>
  );
}

// ─── info tab ────────────────────────────────────────────────────────────────

function UserInfoTab({
  user: initialUser,
  projectId,
  onReload,
}: {
  user: User;
  projectId: string;
  onReload: () => void;
}) {
  const { data, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminUsersByUserId({
          path: { project_id: projectId, user_id: initialUser.id },
        }),
      ),
    [initialUser.id],
  );
  const user = data?.user ?? initialUser;

  const [editOpen, setEditOpen] = useState(false);

  return (
    <div className="space-y-4">
      <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Status</dt>
          <dd className="mt-1">
            <Badge variant={statusVariant(user.status)} className="capitalize">
              {user.status}
            </Badge>
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Kind</dt>
          <dd className="mt-1 capitalize">{user.kind}</dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Email</dt>
          <dd className="mt-1 flex items-center gap-1.5">
            {user.primary_email ?? <span className="text-muted-foreground">—</span>}
            {user.email_verified && (
              <Badge variant="secondary" className="text-xs">verified</Badge>
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Phone</dt>
          <dd className="mt-1 flex items-center gap-1.5">
            {user.primary_phone ?? <span className="text-muted-foreground">—</span>}
            {user.phone_verified && (
              <Badge variant="secondary" className="text-xs">verified</Badge>
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Name</dt>
          <dd className="mt-1">{user.profile?.name ?? <span className="text-muted-foreground">—</span>}</dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Locale</dt>
          <dd className="mt-1">{user.profile?.locale ?? <span className="text-muted-foreground">—</span>}</dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Created</dt>
          <dd className="mt-1">{fmtDate(user.created_at)}</dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Updated</dt>
          <dd className="mt-1">{fmtDate(user.updated_at)}</dd>
        </div>
      </dl>
      <div className="flex gap-2">
        <Button size="sm" variant="outline" onClick={() => setEditOpen(true)}>
          Edit
        </Button>
        <Button size="sm" variant="ghost" onClick={reload}>
          <RefreshCw className="size-3.5" />
          Refresh
        </Button>
      </div>
      <EditUserDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        user={user}
        projectId={projectId}
        onSaved={() => {
          reload();
          onReload();
        }}
      />
    </div>
  );
}

// ─── sessions tab ─────────────────────────────────────────────────────────────

function UserSessionsTab({ user, projectId }: { user: User; projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminUsersByUserIdSessions({
          path: { project_id: projectId, user_id: user.id },
        }),
      ),
    [user.id],
  );
  const sessions = data?.data ?? [];

  async function revokeSession(sessionId: string) {
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionId({
          path: { project_id: projectId, user_id: user.id, session_id: sessionId },
        }),
      );
      toast.success('Session revoked.');
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to revoke session');
    }
  }

  async function revokeAll() {
    try {
      const result = await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevoke({
          path: { project_id: projectId, user_id: user.id },
        }),
      );
      toast.success(`Revoked ${result.revoked_count ?? 0} session(s).`);
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to revoke sessions');
    }
  }

  const sessionColumns: ColumnDef<AdminSession, unknown>[] = [
    {
      accessorKey: 'id',
      header: 'Session ID',
      cell: ({ row }) => row.original.id ? <CopyButton value={row.original.id} /> : '—',
    },
    {
      accessorKey: 'device_name',
      header: 'Device',
      cell: ({ row }) => row.original.device_name ?? <span className="text-muted-foreground">—</span>,
    },
    {
      accessorKey: 'aal',
      header: 'AAL',
      cell: ({ row }) => row.original.aal ?? '—',
    },
    {
      accessorFn: (s) => s.last_active_at ?? '',
      id: 'last_active_at',
      header: 'Last active',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{fmtDate(row.original.last_active_at)}</span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button
          size="sm"
          variant="destructive"
          onClick={() => row.original.id && revokeSession(row.original.id)}
        >
          Revoke
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{sessions.length} session(s)</p>
        {sessions.length > 0 && (
          <Button size="sm" variant="destructive" onClick={revokeAll}>
            Revoke all
          </Button>
        )}
      </div>
      {loading && <LoadingState label="Loading sessions…" />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && (
        <DataTable
          columns={sessionColumns}
          data={sessions}
          searchable={false}
          emptyMessage="No active sessions."
        />
      )}
    </div>
  );
}

// ─── identities tab ───────────────────────────────────────────────────────────

function UserIdentitiesTab({ user, projectId }: { user: User; projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminUsersByUserIdIdentities({
          path: { project_id: projectId, user_id: user.id },
        }),
      ),
    [user.id],
  );
  const identities = data?.data ?? [];

  async function deleteIdentity(identityId: string) {
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityId({
          path: { project_id: projectId, user_id: user.id, identity_id: identityId },
        }),
      );
      toast.success('Identity removed.');
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to remove identity');
    }
  }

  const identityColumns: ColumnDef<Identity, unknown>[] = [
    {
      accessorKey: 'id',
      header: 'ID',
      cell: ({ row }) => row.original.id ? <CopyButton value={row.original.id} /> : '—',
    },
    {
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => (
        <Badge variant="outline" className="capitalize">
          {row.original.type}
        </Badge>
      ),
    },
    {
      accessorKey: 'provider',
      header: 'Provider',
      cell: ({ row }) => row.original.provider ?? <span className="text-muted-foreground">—</span>,
    },
    {
      accessorKey: 'email',
      header: 'Email',
      cell: ({ row }) => row.original.email ?? <span className="text-muted-foreground">—</span>,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button
          size="sm"
          variant="destructive"
          onClick={() => row.original.id && deleteIdentity(row.original.id)}
        >
          <Fingerprint className="size-3.5" />
          Remove
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-3">
      {loading && <LoadingState label="Loading identities…" />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && (
        <DataTable
          columns={identityColumns}
          data={identities}
          searchable={false}
          emptyMessage="No linked identities."
        />
      )}
    </div>
  );
}

// ─── grants tab ───────────────────────────────────────────────────────────────

function UserGrantsTab({ user, projectId }: { user: User; projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminUsersByUserIdGrants({
          path: { project_id: projectId, user_id: user.id },
        }),
      ),
    [user.id],
  );
  const grants = data?.data ?? [];

  async function revokeGrant(grantId: string) {
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminUsersByUserIdGrantsByGrantId({
          path: { project_id: projectId, user_id: user.id, grant_id: grantId },
        }),
      );
      toast.success('Grant revoked.');
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to revoke grant');
    }
  }

  const grantColumns: ColumnDef<OAuthGrant, unknown>[] = [
    {
      accessorKey: 'id',
      header: 'Grant ID',
      cell: ({ row }) => row.original.id ? <CopyButton value={row.original.id} /> : '—',
    },
    {
      accessorFn: (g) => g.client?.name ?? g.client?.id ?? '',
      id: 'client',
      header: 'Client',
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.client?.name ?? row.original.client?.id ?? <span className="text-muted-foreground">—</span>}
        </span>
      ),
    },
    {
      accessorFn: (g) => g.scopes?.join(', ') ?? '',
      id: 'scopes',
      header: 'Scopes',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.scopes?.join(', ') ?? '—'}
        </span>
      ),
    },
    {
      accessorFn: (g) => g.granted_at ?? '',
      id: 'granted_at',
      header: 'Granted',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{fmtDate(row.original.granted_at)}</span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button
          size="sm"
          variant="destructive"
          onClick={() => row.original.id && revokeGrant(row.original.id)}
        >
          Revoke
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-3">
      {loading && <LoadingState label="Loading grants…" />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && (
        <DataTable
          columns={grantColumns}
          data={grants}
          searchable={false}
          emptyMessage="No OAuth grants."
        />
      )}
    </div>
  );
}

// ─── edit user dialog ────────────────────────────────────────────────────────

interface EditUserDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  user: User;
  projectId: string;
  onSaved: () => void;
}

function EditUserDialog({ open, onOpenChange, user, projectId, onSaved }: EditUserDialogProps) {
  const [name, setName] = useState(user.profile?.name ?? '');
  const [locale, setLocale] = useState(user.profile?.locale ?? '');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        patchV1ProjectsByProjectIdAdminUsersByUserId({
          path: { project_id: projectId, user_id: user.id },
          body: {
            profile: {
              name: name || null,
              locale: locale || undefined,
            },
          },
        }),
      );
      toast.success('User updated.');
      onOpenChange(false);
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to update user');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit user</DialogTitle>
          <DialogDescription>Update profile fields for this user.</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="eu-name">Display name</Label>
            <Input
              id="eu-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Jane Doe"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="eu-locale">Locale</Label>
            <Input
              id="eu-locale"
              value={locale}
              onChange={(e) => setLocale(e.target.value)}
              placeholder="en"
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Save changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ─── set password dialog ──────────────────────────────────────────────────────

interface SetPasswordDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  user: User;
  projectId: string;
}

function SetPasswordDialog({ open, onOpenChange, user, projectId }: SetPasswordDialogProps) {
  const [password, setPassword] = useState('');
  const [revokeSessions, setRevokeSessions] = useState(true);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdPassword({
          path: { project_id: projectId, user_id: user.id },
          body: { password, revoke_sessions: revokeSessions },
        }),
      );
      toast.success('Password updated.');
      onOpenChange(false);
      setPassword('');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to set password');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set password</DialogTitle>
          <DialogDescription>
            Override the password for <strong>{user.primary_email ?? user.id}</strong>. Existing
            sessions can optionally be revoked.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="sp-password">New password</Label>
            <Input
              id="sp-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoFocus
            />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={revokeSessions}
              onChange={(e) => setRevokeSessions(e.target.checked)}
              className="size-4 rounded border border-input"
            />
            Revoke existing sessions
          </label>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !password}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Set password
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ─── ban dialog ───────────────────────────────────────────────────────────────

interface BanDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  user: User;
  projectId: string;
  onBanned: () => void;
}

function BanDialog({ open, onOpenChange, user, projectId, onBanned }: BanDialogProps) {
  const [reason, setReason] = useState('');
  const [until, setUntil] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdBan({
          path: { project_id: projectId, user_id: user.id },
          body: {
            reason: reason || undefined,
            until: until || undefined,
          },
        }),
      );
      toast.success('User banned.');
      onOpenChange(false);
      setReason('');
      setUntil('');
      onBanned();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to ban user');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Ban user</DialogTitle>
          <DialogDescription>
            Ban <strong>{user.primary_email ?? user.id}</strong>. Leave "Until" blank for a
            permanent ban.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="ban-reason">Reason</Label>
            <Input
              id="ban-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="optional"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ban-until">Until (ISO date-time)</Label>
            <Input
              id="ban-until"
              value={until}
              onChange={(e) => setUntil(e.target.value)}
              placeholder="2026-01-01T00:00:00Z — leave blank for permanent"
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" variant="destructive" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Ban user
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ─── impersonate dialog ───────────────────────────────────────────────────────

interface ImpersonateDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  user: User;
  projectId: string;
}

function ImpersonateDialog({ open, onOpenChange, user, projectId }: ImpersonateDialogProps) {
  const [reason, setReason] = useState('');
  const [durationSeconds, setDurationSeconds] = useState(3600);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [impersonationUrl, setImpersonationUrl] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const result = await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdImpersonate({
          path: { project_id: projectId, user_id: user.id },
          body: { reason, duration_seconds: durationSeconds },
        }),
      );
      setImpersonationUrl(result.impersonation_url ?? null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create impersonation URL');
    } finally {
      setBusy(false);
    }
  }

  function handleOpenChange(v: boolean) {
    if (!v) {
      setReason('');
      setDurationSeconds(3600);
      setImpersonationUrl(null);
      setErr(null);
    }
    onOpenChange(v);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Impersonate user</DialogTitle>
          <DialogDescription>
            Generate a one-time login URL for{' '}
            <strong>{user.primary_email ?? user.id}</strong>. The URL will be shown once — copy it
            immediately.
          </DialogDescription>
        </DialogHeader>

        {impersonationUrl ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
              This URL grants full access as the user. Do not share it insecurely. It expires after
              the requested duration.
            </div>
            <div className="space-y-2">
              <Label>Impersonation URL (one-time)</Label>
              <div className="flex flex-col gap-2">
                <CopyButton value={impersonationUrl} className="w-full" />
              </div>
            </div>
            <DialogFooter showCloseButton />
          </div>
        ) : (
          <form onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="imp-reason">Reason *</Label>
              <Input
                id="imp-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Support ticket #1234"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="imp-duration">Duration (seconds)</Label>
              <Input
                id="imp-duration"
                type="number"
                min={60}
                max={86400}
                value={durationSeconds}
                onChange={(e) => setDurationSeconds(Number(e.target.value))}
              />
            </div>
            {err && <p className="text-sm text-destructive">{err}</p>}
            <DialogFooter>
              <Button type="submit" disabled={busy || !reason}>
                {busy && <Loader2 className="size-4 animate-spin" />}
                Generate URL
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ─── anonymize dialog ─────────────────────────────────────────────────────────

interface AnonymizeDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  user: User;
  projectId: string;
  onDone: () => void;
}

function AnonymizeDialog({ open, onOpenChange, user, projectId, onDone }: AnonymizeDialogProps) {
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminUsersByUserIdAnonymize({
          path: { project_id: projectId, user_id: user.id },
          body: { reason: reason || undefined },
        }),
      );
      toast.success('User anonymized.');
      onOpenChange(false);
      setReason('');
      onDone();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to anonymize user');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Anonymize user</DialogTitle>
          <DialogDescription>
            Permanently scrub PII from <strong>{user.primary_email ?? user.id}</strong>. This
            action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="anon-reason">Reason</Label>
            <Input
              id="anon-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="GDPR erasure request"
              autoFocus
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" variant="destructive" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Anonymize user
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ─── delete user dialog ───────────────────────────────────────────────────────

interface DeleteUserDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  user: User;
  projectId: string;
  onDeleted: () => void;
}

function DeleteUserDialog({ open, onOpenChange, user, projectId, onDeleted }: DeleteUserDialogProps) {
  const [hard, setHard] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function handleDelete() {
    setBusy(true);
    setErr(null);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminUsersByUserId({
          path: { project_id: projectId, user_id: user.id },
          query: { hard },
        }),
      );
      toast.success(hard ? 'User permanently deleted.' : 'User deleted (soft).');
      onOpenChange(false);
      onDeleted();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to delete user');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete user</DialogTitle>
          <DialogDescription>
            Delete <strong>{user.primary_email ?? user.id}</strong>. A soft delete retains the
            record but marks it deactivated. A hard delete is permanent.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={hard}
              onChange={(e) => setHard(e.target.checked)}
              className="size-4 rounded border border-input"
            />
            Hard delete (permanent — cannot be undone)
          </label>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button variant="destructive" disabled={busy} onClick={handleDelete}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              {hard ? 'Permanently delete' : 'Delete user'}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
