import type { SigningKey, TokenProfile } from '@gopherex/iam-sdk';
import {
  deleteV1ProjectsByProjectIdAdminJwksByKeyId,
  deleteV1ProjectsByProjectIdAdminTokenProfilesById,
  getV1ProjectsByProjectIdAdminJwks,
  getV1ProjectsByProjectIdAdminTokenProfiles,
  patchV1ProjectsByProjectIdAdminTokenProfilesById,
  postV1ProjectsByProjectIdAdminJwksByKeyIdActivate,
  postV1ProjectsByProjectIdAdminJwksRotate,
  postV1ProjectsByProjectIdAdminTokenProfiles,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  KeyRound,
  Loader2,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Trash2,
  Zap,
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
  DialogClose,
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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type KeyStatus = SigningKey['status'];

function statusVariant(status: KeyStatus): 'default' | 'secondary' | 'outline' | 'destructive' {
  switch (status) {
    case 'active':
      return 'default';
    case 'inactive':
      return 'outline';
    case 'retired':
      return 'destructive';
    default:
      return 'secondary';
  }
}

function fmtTtl(seconds?: number): string {
  if (seconds == null) return '—';
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h`;
  return `${Math.round(seconds / 86400)}d`;
}

// ---------------------------------------------------------------------------
// Dialogs — Signing Keys
// ---------------------------------------------------------------------------

function RotateKeyDialog({
  projectId,
  onRotated,
}: {
  projectId: string;
  onRotated: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [activate, setActivate] = useState(true);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminJwksRotate({
          path: { project_id: projectId },
          body: { activate },
        }),
      );
      setOpen(false);
      onRotated();
      toast.success('New signing key generated' + (activate ? ' and activated' : ''));
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to rotate signing key');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <RefreshCw className="size-4" />
        Rotate key
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generate new signing key</DialogTitle>
          <DialogDescription>
            A new asymmetric key pair will be generated. Existing tokens signed with the current key
            remain valid until the old key is retired.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
            <div className="space-y-0.5">
              <p className="text-sm font-medium">Activate immediately</p>
              <p className="text-xs text-muted-foreground">
                New tokens will be signed with this key right away.
              </p>
            </div>
            <Switch
              checked={activate}
              onCheckedChange={setActivate}
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Generate key
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmDeleteKeyDialog({
  projectId,
  signingKey,
  open,
  onOpenChange,
  onDeleted,
}: {
  projectId: string;
  signingKey: SigningKey;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onDeleted: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function confirm() {
    setBusy(true);
    setErr(null);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminJwksByKeyId({
          path: { project_id: projectId, key_id: signingKey.kid! },
        }),
      );
      onOpenChange(false);
      onDeleted();
      toast.success('Signing key retired');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to retire key');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Retire signing key?</DialogTitle>
          <DialogDescription>
            Key <strong>{signingKey.kid}</strong> will be retired. Tokens already signed with it
            become unverifiable. This cannot be undone.
          </DialogDescription>
        </DialogHeader>
        {err && <p className="text-sm text-destructive">{err}</p>}
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button variant="destructive" onClick={confirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Retire key
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Dialogs — Token Profiles
// ---------------------------------------------------------------------------

function CreateProfileDialog({
  projectId,
  onCreated,
}: {
  projectId: string;
  onCreated: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [audience, setAudience] = useState('');
  const [accessTtl, setAccessTtl] = useState('');
  const [refreshTtl, setRefreshTtl] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function reset() {
    setName('');
    setAudience('');
    setAccessTtl('');
    setRefreshTtl('');
    setErr(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminTokenProfiles({
          path: { project_id: projectId },
          body: {
            name: name || undefined,
            audience: audience || undefined,
            access_ttl: accessTtl ? Number(accessTtl) : undefined,
            refresh_ttl: refreshTtl ? Number(refreshTtl) : undefined,
          },
        }),
      );
      setOpen(false);
      reset();
      onCreated();
      toast.success('Token profile created');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create profile');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) reset();
      }}
    >
      <DialogTrigger render={<Button />}>
        <Plus className="size-4" />
        New profile
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New token profile</DialogTitle>
          <DialogDescription>
            A token profile controls audience, TTLs, and optional claim templates for issued JWTs.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="tp-name">Name</Label>
            <Input
              id="tp-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. mobile-app"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tp-audience">Audience</Label>
            <Input
              id="tp-audience"
              value={audience}
              onChange={(e) => setAudience(e.target.value)}
              placeholder="e.g. https://api.example.com"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="tp-access">Access TTL (s)</Label>
              <Input
                id="tp-access"
                type="number"
                min={0}
                value={accessTtl}
                onChange={(e) => setAccessTtl(e.target.value)}
                placeholder="e.g. 3600"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tp-refresh">Refresh TTL (s)</Label>
              <Input
                id="tp-refresh"
                type="number"
                min={0}
                value={refreshTtl}
                onChange={(e) => setRefreshTtl(e.target.value)}
                placeholder="e.g. 86400"
              />
            </div>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create profile
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditProfileDialog({
  projectId,
  profile,
  open,
  onOpenChange,
  onUpdated,
}: {
  projectId: string;
  profile: TokenProfile;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onUpdated: () => void;
}) {
  const [name, setName] = useState(profile.name ?? '');
  const [audience, setAudience] = useState(profile.audience ?? '');
  const [accessTtl, setAccessTtl] = useState(profile.access_ttl != null ? String(profile.access_ttl) : '');
  const [refreshTtl, setRefreshTtl] = useState(profile.refresh_ttl != null ? String(profile.refresh_ttl) : '');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        patchV1ProjectsByProjectIdAdminTokenProfilesById({
          path: { project_id: projectId, id: profile.id! },
          body: {
            name: name || undefined,
            audience: audience || undefined,
            access_ttl: accessTtl ? Number(accessTtl) : undefined,
            refresh_ttl: refreshTtl ? Number(refreshTtl) : undefined,
          },
        }),
      );
      onOpenChange(false);
      onUpdated();
      toast.success('Token profile updated');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to update profile');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit token profile</DialogTitle>
          <DialogDescription>Update audience, TTLs, or name.</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="etp-name">Name</Label>
            <Input
              id="etp-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="etp-audience">Audience</Label>
            <Input
              id="etp-audience"
              value={audience}
              onChange={(e) => setAudience(e.target.value)}
              placeholder="e.g. https://api.example.com"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="etp-access">Access TTL (s)</Label>
              <Input
                id="etp-access"
                type="number"
                min={0}
                value={accessTtl}
                onChange={(e) => setAccessTtl(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="etp-refresh">Refresh TTL (s)</Label>
              <Input
                id="etp-refresh"
                type="number"
                min={0}
                value={refreshTtl}
                onChange={(e) => setRefreshTtl(e.target.value)}
              />
            </div>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
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

function ConfirmDeleteProfileDialog({
  projectId,
  profile,
  open,
  onOpenChange,
  onDeleted,
}: {
  projectId: string;
  profile: TokenProfile;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onDeleted: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function confirm() {
    setBusy(true);
    setErr(null);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminTokenProfilesById({
          path: { project_id: projectId, id: profile.id! },
        }),
      );
      onOpenChange(false);
      onDeleted();
      toast.success('Token profile deleted');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to delete profile');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete token profile?</DialogTitle>
          <DialogDescription>
            Profile <strong>{profile.name ?? profile.id}</strong> will be permanently deleted. Any
            apps using it will fall back to the project default.
          </DialogDescription>
        </DialogHeader>
        {err && <p className="text-sm text-destructive">{err}</p>}
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button variant="destructive" onClick={confirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Delete profile
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function TokenProfilesTab({ projectId }: { projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminTokenProfiles({ path: { project_id: projectId } })),
    [projectId],
  );
  const profiles = data?.data ?? [];

  const [editProfile, setEditProfile] = useState<TokenProfile | null>(null);
  const [deleteProfile, setDeleteProfile] = useState<TokenProfile | null>(null);

  const columns: ColumnDef<TokenProfile, unknown>[] = [
    {
      accessorKey: 'id',
      header: 'ID',
      cell: ({ row }) => {
        const id = row.original.id;
        return id ? <CopyButton value={id} /> : <span className="text-muted-foreground">—</span>;
      },
    },
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ getValue }) => (
        <span className="font-medium">{(getValue() as string) ?? '—'}</span>
      ),
    },
    {
      accessorKey: 'audience',
      header: 'Audience',
      cell: ({ getValue }) => (
        <span className="max-w-xs truncate text-sm text-muted-foreground">
          {(getValue() as string) ?? '—'}
        </span>
      ),
    },
    {
      accessorKey: 'access_ttl',
      header: 'Access TTL',
      cell: ({ row }) => (
        <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
          {fmtTtl(row.original.access_ttl)}
        </code>
      ),
    },
    {
      accessorKey: 'refresh_ttl',
      header: 'Refresh TTL',
      cell: ({ row }) => (
        <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
          {fmtTtl(row.original.refresh_ttl)}
        </code>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const p = row.original;
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />}>
                <MoreHorizontal className="size-4" />
                <span className="sr-only">Actions</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setEditProfile(p)}>
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem variant="destructive" onClick={() => setDeleteProfile(p)}>
                  <Trash2 className="size-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <CreateProfileDialog projectId={projectId} onCreated={reload} />
      </div>

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && profiles.length === 0 && (
        <EmptyState
          title="No token profiles"
          description="Create a profile to customise audience, TTLs, and claim templates for issued JWTs."
          action={<CreateProfileDialog projectId={projectId} onCreated={reload} />}
        />
      )}
      {!loading && !error && profiles.length > 0 && (
        <DataTable
          columns={columns}
          data={profiles}
          searchable
          searchPlaceholder="Filter by name, audience…"
          emptyMessage="No profiles match the filter."
        />
      )}

      {editProfile && (
        <EditProfileDialog
          projectId={projectId}
          profile={editProfile}
          open={!!editProfile}
          onOpenChange={(o) => { if (!o) setEditProfile(null); }}
          onUpdated={reload}
        />
      )}

      {deleteProfile && (
        <ConfirmDeleteProfileDialog
          projectId={projectId}
          profile={deleteProfile}
          open={!!deleteProfile}
          onOpenChange={(o) => { if (!o) setDeleteProfile(null); }}
          onDeleted={reload}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page root
// ---------------------------------------------------------------------------

export function SigningKeysPage() {
  const { projectId } = useParams();

  return (
    <div>
      <PageHeader
        title="Signing Keys"
        description="Manage JWT signing keys and token profiles for this project."
      />

      <Tabs defaultValue="keys">
        <TabsList>
          <TabsTrigger value="keys">Signing Keys</TabsTrigger>
          <TabsTrigger value="profiles">Token Profiles</TabsTrigger>
        </TabsList>

        <TabsContent value="keys" className="mt-4">
          {projectId ? <SigningKeysTab projectId={projectId} /> : null}
        </TabsContent>

        <TabsContent value="profiles" className="mt-4">
          {projectId ? <TokenProfilesTab projectId={projectId} /> : null}
        </TabsContent>
      </Tabs>
    </div>
  );
}

function SigningKeysTab({ projectId }: { projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminJwks({ path: { project_id: projectId } })),
    [projectId],
  );
  const keys = data?.data ?? [];

  const [deleteKey, setDeleteKey] = useState<SigningKey | null>(null);
  const [activating, setActivating] = useState<string | null>(null);

  async function activate(kid: string) {
    setActivating(kid);
    try {
      await call(
        postV1ProjectsByProjectIdAdminJwksByKeyIdActivate({
          path: { project_id: projectId, key_id: kid },
        }),
      );
      reload();
      toast.success('Key activated');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to activate key');
    } finally {
      setActivating(null);
    }
  }

  const columns: ColumnDef<SigningKey, unknown>[] = [
    {
      accessorKey: 'kid',
      header: 'Key ID',
      cell: ({ row }) => {
        const kid = row.original.kid;
        return kid ? <CopyButton value={kid} /> : <span className="text-muted-foreground">—</span>;
      },
    },
    {
      accessorKey: 'alg',
      header: 'Algorithm',
      cell: ({ getValue }) => (
        <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{(getValue() as string) ?? '—'}</code>
      ),
    },
    {
      accessorKey: 'use',
      header: 'Use',
      cell: ({ getValue }) => (
        <span className="text-sm text-muted-foreground">{(getValue() as string) ?? '—'}</span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => {
        const s = row.original.status;
        return <Badge variant={statusVariant(s)}>{s ?? 'unknown'}</Badge>;
      },
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const k = row.original;
        const kid = k.kid!;
        const isActive = k.status === 'active';
        const isRetired = k.status === 'retired';
        const isActivating = activating === kid;
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />}>
                <MoreHorizontal className="size-4" />
                <span className="sr-only">Actions</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {!isActive && !isRetired && (
                  <DropdownMenuItem onClick={() => activate(kid)} disabled={isActivating}>
                    {isActivating ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Zap className="size-4" />
                    )}
                    Activate
                  </DropdownMenuItem>
                )}
                {!isActive && !isRetired && <DropdownMenuSeparator />}
                <DropdownMenuItem
                  variant="destructive"
                  disabled={isActive || isRetired}
                  onClick={() => setDeleteKey(k)}
                >
                  <Trash2 className="size-4" />
                  Retire key
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
    },
  ];

  return (
    <div className="space-y-4">
      {/* Public JWKS discovery hint */}
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
        <KeyRound className="size-4 shrink-0" />
        <span>Public JWKS:</span>
        <code className="rounded bg-muted px-1.5 py-0.5 text-xs text-foreground">
          /p/{projectId}/e/{'{env}'}/.well-known/jwks.json
        </code>
        <div className="ml-auto flex items-center gap-2">
          <RotateKeyDialog projectId={projectId} onRotated={reload} />
        </div>
      </div>

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && keys.length === 0 && (
        <EmptyState
          title="No signing keys"
          description="Generate the first key pair. Private key material is stored server-side and is never exposed."
          action={<RotateKeyDialog projectId={projectId} onRotated={reload} />}
        />
      )}
      {!loading && !error && keys.length > 0 && (
        <DataTable
          columns={columns}
          data={keys}
          searchable
          searchPlaceholder="Filter by kid, alg…"
          emptyMessage="No keys match the filter."
        />
      )}

      {deleteKey && (
        <ConfirmDeleteKeyDialog
          projectId={projectId}
          signingKey={deleteKey}
          open={!!deleteKey}
          onOpenChange={(o) => { if (!o) setDeleteKey(null); }}
          onDeleted={reload}
        />
      )}
    </div>
  );
}
