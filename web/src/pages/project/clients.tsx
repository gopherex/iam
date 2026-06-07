import type { ColumnDef } from '@tanstack/react-table';
import {
  deleteV1ProjectsByProjectIdAdminAppsByAppId,
  deleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretId,
  getV1ProjectsByProjectIdAdminApps,
  getV1ProjectsByProjectIdAdminAppsByAppId,
  patchV1ProjectsByProjectIdAdminAppsByAppId,
  postV1ProjectsByProjectIdAdminApps,
  postV1ProjectsByProjectIdAdminAppsByAppIdSecrets,
} from '@gopherex/iam-sdk';
import type { AppClient } from '@gopherex/iam-sdk';
import {
  AlertTriangle,
  Eye,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ── Type helpers ─────────────────────────────────────────────────────────────

type AppType = 'spa' | 'native' | 'web' | 'machine';

const APP_TYPES: { value: AppType; label: string }[] = [
  { value: 'spa', label: 'Single-Page App (SPA)' },
  { value: 'web', label: 'Web (server-side)' },
  { value: 'native', label: 'Native / Mobile' },
  { value: 'machine', label: 'Machine (M2M)' },
];

function appTypeLabel(type?: string) {
  return APP_TYPES.find((t) => t.value === type)?.label ?? type ?? '—';
}

function typeVariant(type?: string): 'default' | 'secondary' | 'outline' {
  if (type === 'machine') return 'default';
  if (type === 'web') return 'secondary';
  return 'outline';
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function ClientsPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminApps({ path: { project_id: projectId! } })),
    [projectId],
  );
  const clients = data?.data ?? [];

  // detail/edit dialog state
  const [detailClient, setDetailClient] = useState<AppClient | null>(null);

  const columns: ColumnDef<AppClient, unknown>[] = [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <span className="font-medium">{row.original.name ?? '—'}</span>
      ),
    },
    {
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => (
        <Badge variant={typeVariant(row.original.type)}>
          {appTypeLabel(row.original.type)}
        </Badge>
      ),
    },
    {
      accessorKey: 'environment',
      header: 'Environment',
      cell: ({ row }) =>
        row.original.environment ? (
          <Badge variant="outline">{row.original.environment}</Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      accessorKey: 'id',
      header: 'Client ID',
      cell: ({ row }) =>
        row.original.id ? (
          <CopyButton value={row.original.id} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      id: 'redirect_uris',
      header: 'Redirect URIs',
      cell: ({ row }) => {
        const uris = row.original.redirect_uris ?? [];
        if (uris.length === 0) return <span className="text-muted-foreground">—</span>;
        return (
          <span className="text-sm text-muted-foreground">
            {uris.length === 1 ? uris[0] : `${uris[0]} +${uris.length - 1}`}
          </span>
        );
      },
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex justify-end">
          <ClientRowActions
            client={row.original}
            projectId={projectId!}
            onEdit={() => setDetailClient(row.original)}
            onDeleted={reload}
          />
        </div>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="App Clients"
        description="OAuth / OIDC application clients registered in this project."
        actions={<CreateClientDialog projectId={projectId!} onCreated={reload} />}
      />

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}

      {!loading && !error && clients.length === 0 && (
        <EmptyState
          title="No app clients yet"
          description="Register your first OAuth / OIDC client to let applications authenticate."
          action={<CreateClientDialog projectId={projectId!} onCreated={reload} />}
        />
      )}

      {!loading && !error && clients.length > 0 && (
        <DataTable
          columns={columns}
          data={clients}
          searchPlaceholder="Search clients…"
        />
      )}

      {detailClient && (
        <ClientDetailDialog
          projectId={projectId!}
          client={detailClient}
          open={!!detailClient}
          onOpenChange={(open) => { if (!open) setDetailClient(null); }}
          onSaved={() => { setDetailClient(null); reload(); }}
        />
      )}
    </div>
  );
}

// ── Row actions ───────────────────────────────────────────────────────────────

function ClientRowActions({
  client,
  projectId,
  onEdit,
  onDeleted,
}: {
  client: AppClient;
  projectId: string;
  onEdit: () => void;
  onDeleted: () => void;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function handleDelete() {
    if (!client.id) return;
    setDeleting(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminAppsByAppId({
          path: { project_id: projectId, app_id: client.id },
        }),
      );
      toast.success('App client deleted');
      setDeleteOpen(false);
      onDeleted();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete client');
    } finally {
      setDeleting(false);
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" />}>
          <MoreHorizontal className="size-4" />
          <span className="sr-only">Actions</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={onEdit}>
            <Eye className="size-4" />
            View / Edit
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="size-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Delete confirmation */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete app client</DialogTitle>
            <DialogDescription>
              This permanently removes <strong>{client.name ?? client.id}</strong> and revokes all
              its tokens. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting && <Loader2 className="size-4 animate-spin" />}
              Delete client
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ── Create dialog ─────────────────────────────────────────────────────────────

function CreateClientDialog({
  projectId,
  onCreated,
}: {
  projectId: string;
  onCreated: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [type, setType] = useState<AppType>('spa');
  const [environment, setEnvironment] = useState('');
  const [redirectUris, setRedirectUris] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // secret reveal after create
  const [newSecret, setNewSecret] = useState<{ secretId?: string; value: string } | null>(null);

  function reset() {
    setName('');
    setType('spa');
    setEnvironment('');
    setRedirectUris('');
    setErr(null);
    setBusy(false);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const uris = redirectUris
        .split('\n')
        .map((u) => u.trim())
        .filter(Boolean);
      const result = await call(
        postV1ProjectsByProjectIdAdminApps({
          path: { project_id: projectId },
          body: {
            name,
            type,
            environment: environment || undefined,
            redirect_uris: uris.length ? uris : undefined,
          },
        }),
      );
      if (result.client_secret) {
        setNewSecret({ value: result.client_secret });
      } else {
        toast.success('App client created');
        setOpen(false);
        reset();
        onCreated();
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create client');
    } finally {
      setBusy(false);
    }
  }

  function closeAfterSecret() {
    setNewSecret(null);
    setOpen(false);
    reset();
    onCreated();
  }

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => { setOpen(o); if (!o) { reset(); setNewSecret(null); } }}>
        <DialogTrigger render={<Button />}>
          <Plus className="size-4" />
          New client
        </DialogTrigger>
        <DialogContent className="sm:max-w-md">
          {newSecret ? (
            <SecretRevealPanel secret={newSecret.value} onDone={closeAfterSecret} />
          ) : (
            <>
              <DialogHeader>
                <DialogTitle>New app client</DialogTitle>
                <DialogDescription>
                  Register an OAuth / OIDC application. For confidential types (web, machine) a
                  client secret will be shown once after creation.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={submit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="c-name">Name</Label>
                  <Input
                    id="c-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    autoFocus
                    placeholder="My Web App"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Type</Label>
                    <Select value={type} onValueChange={(v) => setType(v as AppType)}>
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {APP_TYPES.map((t) => (
                          <SelectItem key={t.value} value={t.value}>
                            {t.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="c-env">Environment</Label>
                    <Input
                      id="c-env"
                      value={environment}
                      onChange={(e) => setEnvironment(e.target.value)}
                      placeholder="production"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="c-uris">Redirect URIs</Label>
                  <textarea
                    id="c-uris"
                    value={redirectUris}
                    onChange={(e) => setRedirectUris(e.target.value)}
                    rows={3}
                    placeholder="https://app.example.com/callback&#10;http://localhost:3000/callback"
                    className="h-auto w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1.5 text-sm placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                  <p className="text-xs text-muted-foreground">One URI per line.</p>
                </div>
                {err && <p className="text-sm text-destructive">{err}</p>}
                <DialogFooter>
                  <Button type="submit" disabled={busy || !name}>
                    {busy && <Loader2 className="size-4 animate-spin" />}
                    Create client
                  </Button>
                </DialogFooter>
              </form>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

// ── Secret reveal panel ───────────────────────────────────────────────────────

function SecretRevealPanel({ secret, onDone }: { secret: string; onDone: () => void }) {
  return (
    <>
      <DialogHeader>
        <DialogTitle>Client secret — store it now</DialogTitle>
        <DialogDescription>
          This secret will <strong>not</strong> be shown again. Copy it and store it securely before
          closing.
        </DialogDescription>
      </DialogHeader>
      <div className="space-y-3">
        <div className="flex items-center gap-2 rounded-lg border border-amber-300 bg-amber-50 p-3 text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
          <AlertTriangle className="size-4 shrink-0" />
          <span className="text-xs">This is the only time the secret is displayed.</span>
        </div>
        <div className="space-y-1">
          <Label>Client secret</Label>
          <CopyButton value={secret} className="w-full max-w-full font-mono text-xs" />
        </div>
      </div>
      <DialogFooter>
        <Button onClick={onDone}>I've saved the secret</Button>
      </DialogFooter>
    </>
  );
}

// ── Detail / Edit dialog ──────────────────────────────────────────────────────

function ClientDetailDialog({
  projectId,
  client: initialClient,
  open,
  onOpenChange,
  onSaved,
}: {
  projectId: string;
  client: AppClient;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const appId = initialClient.id!;

  // fetch the latest version when dialog opens
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminAppsByAppId({ path: { project_id: projectId, app_id: appId } })),
    [projectId, appId],
  );
  const client = data?.app ?? initialClient;

  // edit form state
  const [name, setName] = useState(initialClient.name ?? '');
  const [type, setType] = useState<AppType>((initialClient.type as AppType) ?? 'spa');
  const [redirectUris, setRedirectUris] = useState(
    (initialClient.redirect_uris ?? []).join('\n'),
  );
  const [loginUri, setLoginUri] = useState(initialClient.login_uri ?? '');
  const [defaultRedirectUri, setDefaultRedirectUri] = useState(
    initialClient.default_redirect_uri ?? '',
  );
  const [saving, setSaving] = useState(false);
  const [saveErr, setSaveErr] = useState<string | null>(null);

  // secret management
  const [newSecret, setNewSecret] = useState<string | null>(null);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setSaveErr(null);
    try {
      const uris = redirectUris
        .split('\n')
        .map((u) => u.trim())
        .filter(Boolean);
      await call(
        patchV1ProjectsByProjectIdAdminAppsByAppId({
          path: { project_id: projectId, app_id: appId },
          body: {
            name,
            type,
            redirect_uris: uris,
            login_uri: loginUri || null,
            default_redirect_uri: defaultRedirectUri || null,
          },
        }),
      );
      toast.success('App client updated');
      onSaved();
    } catch (e) {
      setSaveErr(e instanceof Error ? e.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        {newSecret ? (
          <SecretRevealPanel secret={newSecret} onDone={() => { setNewSecret(null); reload(); }} />
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>App client</DialogTitle>
              <DialogDescription>
                {client.id && <CopyButton value={client.id} />}
              </DialogDescription>
            </DialogHeader>

            {loading && <LoadingState />}
            {error && <ErrorState error={error} onRetry={reload} />}

            {!loading && !error && (
              <form onSubmit={handleSave} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="d-name">Name</Label>
                  <Input
                    id="d-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label>Type</Label>
                  <Select value={type} onValueChange={(v) => setType(v as AppType)}>
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {APP_TYPES.map((t) => (
                        <SelectItem key={t.value} value={t.value}>
                          {t.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="d-uris">Redirect URIs</Label>
                  <textarea
                    id="d-uris"
                    value={redirectUris}
                    onChange={(e) => setRedirectUris(e.target.value)}
                    rows={3}
                    placeholder="https://app.example.com/callback"
                    className="h-auto w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1.5 text-sm placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                  <p className="text-xs text-muted-foreground">One URI per line.</p>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="d-login-uri">Login URI</Label>
                    <Input
                      id="d-login-uri"
                      value={loginUri}
                      onChange={(e) => setLoginUri(e.target.value)}
                      placeholder="optional"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="d-default-uri">Default redirect URI</Label>
                    <Input
                      id="d-default-uri"
                      value={defaultRedirectUri}
                      onChange={(e) => setDefaultRedirectUri(e.target.value)}
                      placeholder="optional"
                    />
                  </div>
                </div>

                {saveErr && <p className="text-sm text-destructive">{saveErr}</p>}

                <DialogFooter className="items-center">
                  <AddSecretDialog
                    projectId={projectId}
                    appId={appId}
                    onSecretCreated={(s) => setNewSecret(s)}
                  />
                  <Button type="submit" disabled={saving}>
                    {saving && <Loader2 className="size-4 animate-spin" />}
                    Save changes
                  </Button>
                </DialogFooter>
              </form>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ── Add secret dialog ─────────────────────────────────────────────────────────

function AddSecretDialog({
  projectId,
  appId,
  onSecretCreated,
}: {
  projectId: string;
  appId: string;
  onSecretCreated: (secret: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [secretName, setSecretName] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const result = await call(
        postV1ProjectsByProjectIdAdminAppsByAppIdSecrets({
          path: { project_id: projectId, app_id: appId },
          body: {
            name: secretName,
            expires_at: expiresAt || undefined,
          },
        }),
      );
      toast.success('Client secret created');
      setOpen(false);
      setSecretName('');
      setExpiresAt('');
      if (result.client_secret) {
        onSecretCreated(result.client_secret);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create secret');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" type="button" />}>
        <KeyRound className="size-4" />
        Add secret
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add client secret</DialogTitle>
          <DialogDescription>
            Create a new secret for this app client. The secret value is shown once immediately
            after creation.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="s-name">Secret name</Label>
            <Input
              id="s-name"
              value={secretName}
              onChange={(e) => setSecretName(e.target.value)}
              required
              autoFocus
              placeholder="e.g. prod-2025"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="s-expires">Expires at</Label>
            <Input
              id="s-expires"
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Leave blank for no expiry.</p>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>Cancel</DialogClose>
            <Button type="submit" disabled={busy || !secretName}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create secret
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ── Delete secret dialog (standalone, for future secrets list) ────────────────

export function DeleteSecretDialog({
  projectId,
  appId,
  secretId,
  secretName,
  onDeleted,
}: {
  projectId: string;
  appId: string;
  secretId: string;
  secretName?: string;
  onDeleted: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function handleDelete() {
    setDeleting(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretId({
          path: { project_id: projectId, app_id: appId, secret_id: secretId },
        }),
      );
      toast.success('Secret revoked');
      setOpen(false);
      onDeleted();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to revoke secret');
    } finally {
      setDeleting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="ghost" size="icon-sm" />}>
        <Trash2 className="size-3.5" />
        <span className="sr-only">Revoke secret</span>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Revoke secret</DialogTitle>
          <DialogDescription>
            Revoking <strong>{secretName ?? secretId}</strong> will immediately invalidate any
            tokens obtained with it. This cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
            {deleting && <Loader2 className="size-4 animate-spin" />}
            Revoke secret
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
