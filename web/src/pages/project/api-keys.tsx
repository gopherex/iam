import type { ApiKey } from '@gopherex/iam-sdk';
import {
  deleteV1ProjectsByProjectIdAdminApiKeysByKeyId,
  getV1ProjectsByProjectIdAdminApiKeys,
  patchV1ProjectsByProjectIdAdminApiKeysByKeyId,
  postV1ProjectsByProjectIdAdminApiKeys,
  postV1ProjectsByProjectIdAdminApiKeysByKeyIdRotate,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  AlertTriangle,
  Loader2,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Trash2,
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
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatExpiry(ts?: string): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleDateString(undefined, { dateStyle: 'medium' });
}

function isExpired(ts?: string): boolean {
  if (!ts) return false;
  return new Date(ts) < new Date();
}

// ---------------------------------------------------------------------------
// Secret reveal dialog (shown once after create or rotate)
// ---------------------------------------------------------------------------

function SecretRevealDialog({
  open,
  onOpenChange,
  secret,
  keyName,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  secret: string;
  keyName?: string;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Save your secret key</DialogTitle>
          <DialogDescription>
            The secret for <strong>{keyName ?? 'this key'}</strong> is shown only once. Copy it now
            — you won't be able to retrieve it again.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 dark:border-amber-900 dark:bg-amber-950/40">
            <AlertTriangle className="size-4 shrink-0 text-amber-600 dark:text-amber-500" />
            <p className="text-xs text-amber-700 dark:text-amber-400">
              Store it in a secrets manager — this dialog will not appear again.
            </p>
          </div>
          <CopyButton value={secret} className="w-full justify-between font-mono text-xs" />
        </div>
        <DialogFooter>
          <DialogClose render={<Button />}>Done, I've copied it</DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Create dialog
// ---------------------------------------------------------------------------

function CreateApiKeyDialog({ projectId, onCreated }: { projectId: string; onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [scopes, setScopes] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // After creation, show the secret reveal dialog
  const [revealOpen, setRevealOpen] = useState(false);
  const [revealSecret, setRevealSecret] = useState('');
  const [revealName, setRevealName] = useState('');

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const scopeList = scopes
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      const res = await call(
        postV1ProjectsByProjectIdAdminApiKeys({
          path: { project_id: projectId },
          body: {
            name,
            scopes: scopeList,
            expires_at: expiresAt || undefined,
          },
        }),
      );
      setOpen(false);
      setName('');
      setScopes('');
      setExpiresAt('');
      onCreated();
      if (res.secret) {
        setRevealSecret(res.secret);
        setRevealName(res.api_key?.name ?? name);
        setRevealOpen(true);
      } else {
        toast.success('API key created');
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create API key');
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger render={<Button />}>
          <Plus className="size-4" />
          New API key
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New API key</DialogTitle>
            <DialogDescription>
              Create a long-lived credential scoped to this project's admin API.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="ak-name">Name</Label>
              <Input
                id="ak-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. CI pipeline key"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ak-scopes">Scopes</Label>
              <Input
                id="ak-scopes"
                value={scopes}
                onChange={(e) => setScopes(e.target.value)}
                placeholder="e.g. users:read users:write"
              />
              <p className="text-xs text-muted-foreground">Space or comma-separated list of scopes.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="ak-expires">Expiry date</Label>
              <Input
                id="ak-expires"
                type="date"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">Leave blank for a non-expiring key.</p>
            </div>
            {err && <p className="text-sm text-destructive">{err}</p>}
            <DialogFooter>
              <Button type="submit" disabled={busy || !name}>
                {busy && <Loader2 className="size-4 animate-spin" />}
                Create key
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <SecretRevealDialog
        open={revealOpen}
        onOpenChange={setRevealOpen}
        secret={revealSecret}
        keyName={revealName}
      />
    </>
  );
}

// ---------------------------------------------------------------------------
// Edit dialog
// ---------------------------------------------------------------------------

function EditApiKeyDialog({
  projectId,
  apiKey,
  onUpdated,
  open,
  onOpenChange,
}: {
  projectId: string;
  apiKey: ApiKey;
  onUpdated: () => void;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const [name, setName] = useState(apiKey.name ?? '');
  const [scopes, setScopes] = useState((apiKey.scopes ?? []).join(' '));
  const [disabled, setDisabled] = useState(apiKey.disabled ?? false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const scopeList = scopes
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      await call(
        patchV1ProjectsByProjectIdAdminApiKeysByKeyId({
          path: { project_id: projectId, key_id: apiKey.id! },
          body: { name, scopes: scopeList, disabled },
        }),
      );
      onOpenChange(false);
      onUpdated();
      toast.success('API key updated');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to update API key');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit API key</DialogTitle>
          <DialogDescription>Update name, scopes, or status for this key.</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="ek-name">Name</Label>
            <Input
              id="ek-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ek-scopes">Scopes</Label>
            <Input
              id="ek-scopes"
              value={scopes}
              onChange={(e) => setScopes(e.target.value)}
              placeholder="e.g. users:read users:write"
            />
            <p className="text-xs text-muted-foreground">Space or comma-separated list of scopes.</p>
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="ek-disabled">Disabled</Label>
            <Switch
              id="ek-disabled"
              checked={disabled}
              onCheckedChange={(checked) => setDisabled(checked)}
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !name}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Save changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Rotate confirm dialog
// ---------------------------------------------------------------------------

function RotateDialog({
  projectId,
  apiKey,
  onRotated,
  open,
  onOpenChange,
}: {
  projectId: string;
  apiKey: ApiKey;
  onRotated: () => void;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // After rotation, show secret reveal
  const [revealOpen, setRevealOpen] = useState(false);
  const [revealSecret, setRevealSecret] = useState('');

  async function confirm() {
    setBusy(true);
    setErr(null);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminApiKeysByKeyIdRotate({
          path: { project_id: projectId, key_id: apiKey.id! },
        }),
      );
      onOpenChange(false);
      onRotated();
      if (res.secret) {
        setRevealSecret(res.secret);
        setRevealOpen(true);
      } else {
        toast.success('API key rotated');
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to rotate API key');
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rotate API key</DialogTitle>
            <DialogDescription>
              This will immediately invalidate the current secret for{' '}
              <strong>{apiKey.name}</strong> and issue a new one. Any integrations using
              the old secret will stop working.
            </DialogDescription>
          </DialogHeader>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter showCloseButton>
            <Button variant="destructive" onClick={confirm} disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Rotate secret
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SecretRevealDialog
        open={revealOpen}
        onOpenChange={setRevealOpen}
        secret={revealSecret}
        keyName={apiKey.name}
      />
    </>
  );
}

// ---------------------------------------------------------------------------
// Delete confirm dialog
// ---------------------------------------------------------------------------

function DeleteDialog({
  projectId,
  apiKey,
  onDeleted,
  open,
  onOpenChange,
}: {
  projectId: string;
  apiKey: ApiKey;
  onDeleted: () => void;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function confirm() {
    setBusy(true);
    setErr(null);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminApiKeysByKeyId({
          path: { project_id: projectId, key_id: apiKey.id! },
        }),
      );
      onOpenChange(false);
      onDeleted();
      toast.success('API key deleted');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to delete API key');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete API key</DialogTitle>
          <DialogDescription>
            Permanently delete <strong>{apiKey.name}</strong>? This cannot be undone.
            Any integrations using this key will immediately lose access.
          </DialogDescription>
        </DialogHeader>
        {err && <p className="text-sm text-destructive">{err}</p>}
        <DialogFooter showCloseButton>
          <Button variant="destructive" onClick={confirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            <Trash2 className="size-4" />
            Delete key
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Per-row action menu
// ---------------------------------------------------------------------------

function ApiKeyActions({
  projectId,
  apiKey,
  onRefresh,
}: {
  projectId: string;
  apiKey: ApiKey;
  onRefresh: () => void;
}) {
  const [editOpen, setEditOpen] = useState(false);
  const [rotateOpen, setRotateOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" aria-label="Key actions" />}>
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => setEditOpen(true)}>
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setRotateOpen(true)}>
            <RefreshCw className="size-4" />
            Rotate secret
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="size-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {editOpen && (
        <EditApiKeyDialog
          projectId={projectId}
          apiKey={apiKey}
          onUpdated={onRefresh}
          open={editOpen}
          onOpenChange={setEditOpen}
        />
      )}
      {rotateOpen && (
        <RotateDialog
          projectId={projectId}
          apiKey={apiKey}
          onRotated={onRefresh}
          open={rotateOpen}
          onOpenChange={setRotateOpen}
        />
      )}
      {deleteOpen && (
        <DeleteDialog
          projectId={projectId}
          apiKey={apiKey}
          onDeleted={onRefresh}
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
        />
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// Column definitions
// ---------------------------------------------------------------------------

function buildColumns(projectId: string, reload: () => void): ColumnDef<ApiKey, unknown>[] {
  return [
    {
      id: 'name',
      header: 'Name',
      accessorFn: (row) => row.name ?? '',
      cell: ({ row }) => {
        const k = row.original;
        return (
          <div className="flex flex-col gap-0.5">
            <span className="font-medium">{k.name}</span>
            {k.id && <CopyButton value={k.id} />}
          </div>
        );
      },
    },
    {
      id: 'prefix',
      header: 'Prefix',
      accessorFn: (row) => row.prefix ?? '',
      cell: ({ row }) =>
        row.original.prefix ? (
          <CopyButton value={row.original.prefix} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      id: 'scopes',
      header: 'Scopes',
      accessorFn: (row) => (row.scopes ?? []).join(' '),
      cell: ({ row }) => {
        const scopes = row.original.scopes ?? [];
        if (scopes.length === 0)
          return <span className="text-muted-foreground">—</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {scopes.map((s) => (
              <Badge key={s} variant="secondary">
                {s}
              </Badge>
            ))}
          </div>
        );
      },
    },
    {
      id: 'status',
      header: 'Status',
      accessorFn: (row) => (row.disabled ? 'disabled' : 'active'),
      cell: ({ row }) => {
        const k = row.original;
        if (k.disabled)
          return <Badge variant="outline">Disabled</Badge>;
        if (isExpired(k.expires_at))
          return <Badge variant="destructive">Expired</Badge>;
        return <Badge variant="secondary">Active</Badge>;
      },
    },
    {
      id: 'expires_at',
      header: 'Expires',
      accessorFn: (row) => row.expires_at ?? '',
      cell: ({ row }) => (
        <span
          className={
            isExpired(row.original.expires_at)
              ? 'text-destructive'
              : 'text-muted-foreground'
          }
        >
          {formatExpiry(row.original.expires_at)}
        </span>
      ),
    },
    {
      id: 'actions',
      header: '',
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex justify-end">
          <ApiKeyActions
            projectId={projectId}
            apiKey={row.original}
            onRefresh={reload}
          />
        </div>
      ),
    },
  ];
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function ApiKeysPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminApiKeys({
          path: { project_id: projectId! },
        }),
      ),
    [projectId],
  );

  const keys = data?.data ?? [];
  const columns = buildColumns(projectId!, reload);

  return (
    <div>
      <PageHeader
        title="API Keys"
        description="Long-lived credentials for machine access to this project's admin API."
        actions={<CreateApiKeyDialog projectId={projectId!} onCreated={reload} />}
      />
      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && keys.length === 0 && (
        <EmptyState
          title="No API keys"
          description="Create a key to grant programmatic admin access."
          action={<CreateApiKeyDialog projectId={projectId!} onCreated={reload} />}
        />
      )}
      {!loading && !error && keys.length > 0 && (
        <DataTable
          columns={columns}
          data={keys}
          searchPlaceholder="Search keys…"
          emptyMessage="No keys match your search."
        />
      )}
    </div>
  );
}
