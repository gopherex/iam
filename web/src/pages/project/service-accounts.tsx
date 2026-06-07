import type { ServiceAccount } from '@gopherex/iam-sdk';
import {
  deleteV1ProjectsByProjectIdAdminServiceAccountsBySaId,
  deleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretId,
  getV1ProjectsByProjectIdAdminServiceAccounts,
  getV1ProjectsByProjectIdAdminServiceAccountsBySaId,
  patchV1ProjectsByProjectIdAdminServiceAccountsBySaId,
  postV1ProjectsByProjectIdAdminServiceAccounts,
  postV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecrets,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  AlertTriangle,
  KeyRound,
  Loader2,
  MoreHorizontal,
  Plus,
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
// Page
// ---------------------------------------------------------------------------

export function ServiceAccountsPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminServiceAccounts({
          path: { project_id: projectId! },
        }),
      ),
    [projectId],
  );
  const accounts = data?.data ?? [];

  // State for detail/edit dialog
  const [editTarget, setEditTarget] = useState<ServiceAccount | null>(null);
  const [editOpen, setEditOpen] = useState(false);

  // State for delete confirm dialog
  const [deleteTarget, setDeleteTarget] = useState<ServiceAccount | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  // State for add-secret dialog (keyed by service account)
  const [secretTarget, setSecretTarget] = useState<ServiceAccount | null>(null);
  const [secretOpen, setSecretOpen] = useState(false);

  const columns: ColumnDef<ServiceAccount, unknown>[] = [
    {
      accessorKey: 'id',
      header: 'ID',
      cell: ({ row }) =>
        row.original.id ? <CopyButton value={row.original.id} /> : <span className="text-muted-foreground">—</span>,
    },
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <span className="font-medium">{row.original.name ?? '—'}</span>
      ),
    },
    {
      accessorKey: 'scopes',
      header: 'Scopes',
      cell: ({ row }) => {
        const scopes = row.original.scopes ?? [];
        if (scopes.length === 0) return <span className="text-muted-foreground text-xs">none</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {scopes.slice(0, 3).map((s) => (
              <Badge key={s} variant="secondary">
                {s}
              </Badge>
            ))}
            {scopes.length > 3 && (
              <Badge variant="outline">+{scopes.length - 3}</Badge>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: 'disabled',
      header: 'Status',
      cell: ({ row }) =>
        row.original.disabled ? (
          <Badge variant="destructive">Disabled</Badge>
        ) : (
          <Badge variant="secondary">Active</Badge>
        ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const sa = row.original;
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-7" />}>
                <MoreHorizontal className="size-4" />
                <span className="sr-only">Open menu</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={() => {
                    setEditTarget(sa);
                    setEditOpen(true);
                  }}
                >
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    setSecretTarget(sa);
                    setSecretOpen(true);
                  }}
                >
                  <KeyRound className="size-4" />
                  Add secret
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => {
                    setDeleteTarget(sa);
                    setDeleteOpen(true);
                  }}
                >
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
    <div>
      <PageHeader
        title="Service Accounts"
        description="Machine identities that authenticate via client credentials."
        actions={<CreateServiceAccountDialog projectId={projectId!} onCreated={reload} />}
      />
      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && accounts.length === 0 && (
        <EmptyState
          title="No service accounts"
          description="Create a service account to grant machine access to your project."
          action={<CreateServiceAccountDialog projectId={projectId!} onCreated={reload} />}
        />
      )}
      {!loading && !error && accounts.length > 0 && (
        <DataTable
          columns={columns}
          data={accounts}
          searchPlaceholder="Search service accounts…"
        />
      )}

      {/* Edit / view dialog */}
      {editTarget && (
        <EditServiceAccountDialog
          projectId={projectId!}
          sa={editTarget}
          open={editOpen}
          onOpenChange={(v) => {
            setEditOpen(v);
            if (!v) setEditTarget(null);
          }}
          onUpdated={reload}
        />
      )}

      {/* Delete confirm dialog */}
      {deleteTarget && (
        <DeleteServiceAccountDialog
          projectId={projectId!}
          sa={deleteTarget}
          open={deleteOpen}
          onOpenChange={(v) => {
            setDeleteOpen(v);
            if (!v) setDeleteTarget(null);
          }}
          onDeleted={reload}
        />
      )}

      {/* Add secret dialog */}
      {secretTarget && (
        <AddSecretDialog
          projectId={projectId!}
          sa={secretTarget}
          open={secretOpen}
          onOpenChange={(v) => {
            setSecretOpen(v);
            if (!v) setSecretTarget(null);
          }}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Create dialog
// ---------------------------------------------------------------------------

function CreateServiceAccountDialog({
  projectId,
  onCreated,
}: {
  projectId: string;
  onCreated: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [scopesRaw, setScopesRaw] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const scopes = scopesRaw
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      await call(
        postV1ProjectsByProjectIdAdminServiceAccounts({
          path: { project_id: projectId },
          body: { name, scopes: scopes.length ? scopes : undefined },
        }),
      );
      toast.success('Service account created');
      setOpen(false);
      setName('');
      setScopesRaw('');
      onCreated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create service account');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <Plus className="size-4" />
        New service account
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New service account</DialogTitle>
          <DialogDescription>
            A service account authenticates machines via client credentials (client_id / client_secret).
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="sa-name">Name</Label>
            <Input
              id="sa-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. data-pipeline"
              required
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="sa-scopes">
              Scopes <span className="text-muted-foreground font-normal">(optional, space or comma separated)</span>
            </Label>
            <Input
              id="sa-scopes"
              value={scopesRaw}
              onChange={(e) => setScopesRaw(e.target.value)}
              placeholder="read:users write:events"
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !name.trim()}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Edit dialog  (also shows secrets list with delete)
// ---------------------------------------------------------------------------

function EditServiceAccountDialog({
  projectId,
  sa: initialSa,
  open,
  onOpenChange,
  onUpdated,
}: {
  projectId: string;
  sa: ServiceAccount;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onUpdated: () => void;
}) {
  // Fetch fresh data when dialog opens
  const { data } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminServiceAccountsBySaId({
          path: { project_id: projectId, sa_id: initialSa.id! },
        }),
      ),
    [projectId, initialSa.id, open],
  );
  const sa = data?.service_account ?? initialSa;

  const [scopesRaw, setScopesRaw] = useState<string | null>(null);
  const [disabled, setDisabled] = useState<boolean | undefined>(undefined);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // Lazy-initialise editable fields from fetched data
  const currentScopes = scopesRaw ?? (sa.scopes ?? []).join(' ');
  const currentDisabled = disabled ?? (sa.disabled ?? false);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const scopes = currentScopes
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      await call(
        patchV1ProjectsByProjectIdAdminServiceAccountsBySaId({
          path: { project_id: projectId, sa_id: sa.id! },
          body: { scopes, disabled: currentDisabled },
        }),
      );
      toast.success('Service account updated');
      onOpenChange(false);
      onUpdated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to update service account');
    } finally {
      setBusy(false);
    }
  }

  // Inline add-secret state inside edit dialog
  const [addSecretOpen, setAddSecretOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit service account</DialogTitle>
          <DialogDescription>
            {sa.name ?? sa.id}
            {sa.id && (
              <span className="block mt-1">
                <CopyButton value={sa.id} />
              </span>
            )}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-scopes">
              Scopes <span className="text-muted-foreground font-normal">(space or comma separated)</span>
            </Label>
            <Input
              id="edit-scopes"
              value={currentScopes}
              onChange={(e) => setScopesRaw(e.target.value)}
              placeholder="read:users write:events"
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="edit-disabled">Disabled</Label>
              <p className="text-xs text-muted-foreground">Prevent this account from authenticating.</p>
            </div>
            <Switch
              id="edit-disabled"
              checked={currentDisabled}
              onCheckedChange={(checked) => setDisabled(checked)}
            />
          </div>

          {/* Secrets section */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Client secrets</Label>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setAddSecretOpen(true)}
              >
                <Plus className="size-3.5" />
                Add secret
              </Button>
            </div>
            {/* Secrets are not returned by GET — surface add + revoke-by-ID flows. */}
            <RevokeSecretByIdForm projectId={projectId} saId={sa.id!} />
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

      {/* Inline add-secret dialog nested in edit */}
      {sa.id && (
        <AddSecretDialog
          projectId={projectId}
          sa={sa}
          open={addSecretOpen}
          onOpenChange={setAddSecretOpen}
        />
      )}
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Delete confirm dialog
// ---------------------------------------------------------------------------

function DeleteServiceAccountDialog({
  projectId,
  sa,
  open,
  onOpenChange,
  onDeleted,
}: {
  projectId: string;
  sa: ServiceAccount;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onDeleted: () => void;
}) {
  const [busy, setBusy] = useState(false);

  async function confirm() {
    setBusy(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminServiceAccountsBySaId({
          path: { project_id: projectId, sa_id: sa.id! },
        }),
      );
      toast.success('Service account deleted');
      onOpenChange(false);
      onDeleted();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete service account');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete service account?</DialogTitle>
          <DialogDescription>
            <span className="font-medium text-foreground">{sa.name ?? sa.id}</span> will be permanently removed. Any
            active sessions using its credentials will stop working immediately.
          </DialogDescription>
        </DialogHeader>
        {sa.id && (
          <div className="flex items-center gap-2 rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
            <span>ID:</span>
            <CopyButton value={sa.id} />
          </div>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={confirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Add secret dialog — minted secret is shown once with CopyButton + warning
// ---------------------------------------------------------------------------

type MintedSecret = {
  secret_id?: string;
  client_id?: string;
  client_secret?: string;
};

function AddSecretDialog({
  projectId,
  sa,
  open,
  onOpenChange,
}: {
  projectId: string;
  sa: ServiceAccount;
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const [name, setName] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [minted, setMinted] = useState<MintedSecret | null>(null);

  function reset() {
    setName('');
    setExpiresAt('');
    setErr(null);
    setMinted(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const result = await call(
        postV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecrets({
          path: { project_id: projectId, sa_id: sa.id! },
          body: {
            name,
            expires_at: expiresAt || undefined,
          },
        }),
      );
      setMinted(result);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create secret');
    } finally {
      setBusy(false);
    }
  }

  function handleOpenChange(v: boolean) {
    if (!v) reset();
    onOpenChange(v);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add client secret</DialogTitle>
          <DialogDescription>
            For service account <span className="font-medium text-foreground">{sa.name ?? sa.id}</span>.
          </DialogDescription>
        </DialogHeader>

        {minted ? (
          /* ---- Reveal screen ---- */
          <div className="space-y-4">
            <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2.5 text-sm text-amber-800 dark:border-amber-800/50 dark:bg-amber-950/30 dark:text-amber-400">
              <AlertTriangle className="mt-0.5 size-4 shrink-0" />
              <span>
                Copy the client secret now — it will <strong>not</strong> be shown again after you close this dialog.
              </span>
            </div>
            <div className="space-y-3">
              {minted.client_id && (
                <SecretField label="Client ID" value={minted.client_id} />
              )}
              {minted.client_secret && (
                <SecretField label="Client secret" value={minted.client_secret} sensitive />
              )}
              {minted.secret_id && (
                <SecretField label="Secret ID" value={minted.secret_id} />
              )}
            </div>
            <DialogFooter>
              <Button onClick={() => handleOpenChange(false)}>Done — I've saved the secret</Button>
            </DialogFooter>
          </div>
        ) : (
          /* ---- Create form ---- */
          <form onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="secret-name">Secret name</Label>
              <Input
                id="secret-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. production-2025"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="secret-expires">
                Expires at <span className="text-muted-foreground font-normal">(optional, ISO 8601)</span>
              </Label>
              <Input
                id="secret-expires"
                type="datetime-local"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
            </div>
            {err && <p className="text-sm text-destructive">{err}</p>}
            <DialogFooter>
              <Button type="submit" disabled={busy || !name.trim()}>
                {busy && <Loader2 className="size-4 animate-spin" />}
                Generate secret
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Revoke secret by ID (inline form inside edit dialog)
// The API does not expose a GET /secrets list, so the user must supply the
// secret_id they received at mint time.
// ---------------------------------------------------------------------------

function RevokeSecretByIdForm({
  projectId,
  saId,
}: {
  projectId: string;
  saId: string;
}) {
  const [secretId, setSecretId] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function revoke(e: React.FormEvent) {
    e.preventDefault();
    if (!secretId.trim()) return;
    setBusy(true);
    setErr(null);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretId({
          path: { project_id: projectId, sa_id: saId, secret_id: secretId.trim() },
        }),
      );
      toast.success('Secret revoked');
      setSecretId('');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to revoke secret');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-md border border-dashed px-3 py-2.5 space-y-2">
      <p className="text-xs text-muted-foreground">
        Secrets cannot be listed — paste a <span className="font-mono">secret_id</span> to revoke it.
      </p>
      <form onSubmit={revoke} className="flex gap-2">
        <Input
          value={secretId}
          onChange={(e) => setSecretId(e.target.value)}
          placeholder="secret_…"
          className="h-7 text-xs font-mono"
        />
        <Button type="submit" size="sm" variant="destructive" disabled={busy || !secretId.trim()}>
          {busy ? <Loader2 className="size-3.5 animate-spin" /> : <Trash2 className="size-3.5" />}
          Revoke
        </Button>
      </form>
      {err && <p className="text-xs text-destructive">{err}</p>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helper: labelled copyable field for minted credentials
// ---------------------------------------------------------------------------

function SecretField({
  label,
  value,
  sensitive = false,
}: {
  label: string;
  value: string;
  sensitive?: boolean;
}) {
  return (
    <div className="space-y-1">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <CopyButton
        value={value}
        className={sensitive ? 'w-full font-mono text-xs' : undefined}
      />
    </div>
  );
}
