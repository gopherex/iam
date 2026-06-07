import {
  deleteV1ProjectsByProjectIdAdminSsoConnectionsById,
  deleteV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensByTokenId,
  getV1ProjectsByProjectIdAdminSsoConnections,
  getV1ProjectsByProjectIdAdminSsoConnectionsById,
  getV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokens,
  patchV1ProjectsByProjectIdAdminSsoConnectionsById,
  postV1ProjectsByProjectIdAdminSsoConnections,
  postV1ProjectsByProjectIdAdminSsoConnectionsByIdRotateCertificate,
  postV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokens,
  postV1ProjectsByProjectIdAdminSsoConnectionsByIdTest,
  type SsoConnection,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  ExternalLink,
  Loader2,
  MoreHorizontal,
  Plus,
  RefreshCw,
  ShieldCheck,
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
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ScimTokenRecord = { id?: string; name?: string; created_at?: string };

// ---------------------------------------------------------------------------
// Status badge helper
// ---------------------------------------------------------------------------

function StatusBadge({ status }: { status?: string }) {
  if (!status) return <span className="text-muted-foreground">—</span>;
  const variant =
    status === 'active'
      ? 'default'
      : status === 'inactive'
        ? 'secondary'
        : 'outline';
  return <Badge variant={variant}>{status}</Badge>;
}

// ---------------------------------------------------------------------------
// Type badge helper
// ---------------------------------------------------------------------------

function TypeBadge({ type }: { type?: string }) {
  if (!type) return null;
  return (
    <Badge variant="outline" className="font-mono uppercase text-xs">
      {type}
    </Badge>
  );
}

// ---------------------------------------------------------------------------
// ConnectionsPage (root)
// ---------------------------------------------------------------------------

export function ConnectionsPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminSsoConnections({ path: { project_id: projectId! } })),
    [projectId],
  );
  const connections = data?.data ?? [];

  // Selected connection for the detail drawer/dialog
  const [selected, setSelected] = useState<SsoConnection | null>(null);

  const columns: ColumnDef<SsoConnection, unknown>[] = [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <span className="font-medium">{row.original.name ?? row.original.id ?? '—'}</span>
      ),
    },
    {
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => <TypeBadge type={row.original.type} />,
    },
    {
      accessorKey: 'domains',
      header: 'Domains',
      cell: ({ row }) => {
        const domains = row.original.domains ?? [];
        if (domains.length === 0) return <span className="text-muted-foreground">—</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {domains.slice(0, 3).map((d) => (
              <Badge key={d} variant="secondary" className="font-mono text-xs">
                {d}
              </Badge>
            ))}
            {domains.length > 3 && (
              <Badge variant="secondary">+{domains.length - 3}</Badge>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      id: 'id',
      header: 'ID',
      cell: ({ row }) =>
        row.original.id ? <CopyButton value={row.original.id} /> : null,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <ConnectionRowActions
          connection={row.original}
          projectId={projectId!}
          onEdit={() => setSelected(row.original)}
          onDeleted={reload}
        />
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="SSO Connections"
        description="Manage SAML and OIDC federation connections for this project."
        actions={<NewConnectionDialog projectId={projectId!} onCreated={reload} />}
      />

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && connections.length === 0 && (
        <EmptyState
          title="No SSO connections"
          description="Add a SAML or OIDC connection to enable enterprise single sign-on."
          action={<NewConnectionDialog projectId={projectId!} onCreated={reload} />}
        />
      )}
      {!loading && !error && connections.length > 0 && (
        <DataTable
          columns={columns}
          data={connections}
          searchPlaceholder="Search connections…"
          emptyMessage="No connections match your search."
        />
      )}

      {selected && (
        <ConnectionDetailDialog
          projectId={projectId!}
          connectionId={selected.id!}
          open={!!selected}
          onOpenChange={(o) => { if (!o) setSelected(null); }}
          onUpdated={reload}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Row actions dropdown
// ---------------------------------------------------------------------------

function ConnectionRowActions({
  connection,
  projectId,
  onEdit,
  onDeleted,
}: {
  connection: SsoConnection;
  projectId: string;
  onEdit: () => void;
  onDeleted: () => void;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  async function testConnection() {
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminSsoConnectionsByIdTest({
          path: { project_id: projectId, id: connection.id! },
        }),
      );
      if (res.test_url) {
        window.open(res.test_url, '_blank', 'noopener,noreferrer');
      } else {
        toast.success('Test initiated');
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Test failed');
    }
  }

  async function rotateCert() {
    try {
      await call(
        postV1ProjectsByProjectIdAdminSsoConnectionsByIdRotateCertificate({
          path: { project_id: projectId, id: connection.id! },
        }),
      );
      toast.success('Certificate rotated');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Rotation failed');
    }
  }

  async function doDelete() {
    setBusy(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminSsoConnectionsById({
          path: { project_id: projectId, id: connection.id! },
        }),
      );
      toast.success('Connection deleted');
      setDeleteOpen(false);
      onDeleted();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Delete failed');
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" size="icon" className="size-7" aria-label="Actions" />
          }
        >
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>Actions</DropdownMenuLabel>
          <DropdownMenuItem onClick={onEdit}>Edit / configure</DropdownMenuItem>
          {connection.type === 'saml' && connection.sp_metadata_url && (
            <DropdownMenuItem
              onClick={() => window.open(connection.sp_metadata_url!, '_blank', 'noopener,noreferrer')}
            >
              <ExternalLink className="size-4" />
              SP metadata
            </DropdownMenuItem>
          )}
          <DropdownMenuItem onClick={testConnection}>
            <ShieldCheck className="size-4" />
            Test login
          </DropdownMenuItem>
          {connection.type === 'saml' && (
            <DropdownMenuItem onClick={rotateCert}>
              <RefreshCw className="size-4" />
              Rotate certificate
            </DropdownMenuItem>
          )}
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
            <DialogTitle>Delete connection</DialogTitle>
            <DialogDescription>
              Permanently delete <strong>{connection.name ?? connection.id}</strong>? This cannot be
              undone and will break any active SSO flows using this connection.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={doDelete} disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Delete connection
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ---------------------------------------------------------------------------
// New connection dialog
// ---------------------------------------------------------------------------

function NewConnectionDialog({
  projectId,
  onCreated,
}: {
  projectId: string;
  onCreated: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [type, setType] = useState<'saml' | 'oidc'>('saml');
  const [name, setName] = useState('');
  const [domains, setDomains] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // SAML config fields
  const [samlMetadataUrl, setSamlMetadataUrl] = useState('');
  const [samlIdpCert, setSamlIdpCert] = useState('');
  const [samlSsoUrl, setSamlSsoUrl] = useState('');
  const [samlEntityId, setSamlEntityId] = useState('');

  // OIDC config fields
  const [oidcIssuer, setOidcIssuer] = useState('');
  const [oidcClientId, setOidcClientId] = useState('');
  const [oidcClientSecret, setOidcClientSecret] = useState('');
  const [oidcScopes, setOidcScopes] = useState('openid email profile');

  function buildConfig(): Record<string, unknown> {
    if (type === 'saml') {
      return {
        metadata_url: samlMetadataUrl || undefined,
        idp_cert: samlIdpCert || undefined,
        sso_url: samlSsoUrl || undefined,
        entity_id: samlEntityId || undefined,
      };
    }
    return {
      issuer: oidcIssuer || undefined,
      client_id: oidcClientId || undefined,
      client_secret: oidcClientSecret || undefined,
      scopes: oidcScopes
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean),
    };
  }

  function reset() {
    setName('');
    setDomains('');
    setType('saml');
    setSamlMetadataUrl('');
    setSamlIdpCert('');
    setSamlSsoUrl('');
    setSamlEntityId('');
    setOidcIssuer('');
    setOidcClientId('');
    setOidcClientSecret('');
    setOidcScopes('openid email profile');
    setErr(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminSsoConnections({
          path: { project_id: projectId },
          body: {
            type,
            name,
            domains: domains
              ? domains
                  .split(/[\s,]+/)
                  .map((d) => d.trim())
                  .filter(Boolean)
              : undefined,
            config: buildConfig(),
          },
        }),
      );
      toast.success('Connection created');
      setOpen(false);
      reset();
      onCreated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create connection');
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
        New connection
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>New SSO connection</DialogTitle>
          <DialogDescription>
            Configure a SAML 2.0 or OpenID Connect federation for this project.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          {/* Name + type */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="nc-name">Name</Label>
              <Input
                id="nc-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                autoFocus
                placeholder="Acme SAML"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="nc-type">Type</Label>
              <Select value={type} onValueChange={(v) => setType(v as 'saml' | 'oidc')}>
                <SelectTrigger id="nc-type" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="saml">SAML 2.0</SelectItem>
                  <SelectItem value="oidc">OIDC</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Domains */}
          <div className="space-y-2">
            <Label htmlFor="nc-domains">Domains</Label>
            <Input
              id="nc-domains"
              value={domains}
              onChange={(e) => setDomains(e.target.value)}
              placeholder="acme.com, acme.org"
            />
            <p className="text-xs text-muted-foreground">
              Comma- or space-separated list of email domains to route through this connection.
            </p>
          </div>

          {/* Type-specific config */}
          {type === 'saml' ? (
            <SamlConfigFields
              metadataUrl={samlMetadataUrl}
              setMetadataUrl={setSamlMetadataUrl}
              idpCert={samlIdpCert}
              setIdpCert={setSamlIdpCert}
              ssoUrl={samlSsoUrl}
              setSsoUrl={setSamlSsoUrl}
              entityId={samlEntityId}
              setEntityId={setSamlEntityId}
            />
          ) : (
            <OidcConfigFields
              issuer={oidcIssuer}
              setIssuer={setOidcIssuer}
              clientId={oidcClientId}
              setClientId={setOidcClientId}
              clientSecret={oidcClientSecret}
              setClientSecret={setOidcClientSecret}
              scopes={oidcScopes}
              setScopes={setOidcScopes}
            />
          )}

          {err && <p className="text-sm text-destructive">{err}</p>}

          <DialogFooter>
            <Button type="submit" disabled={busy || !name}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create connection
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// SAML config fields (shared between create + edit)
// ---------------------------------------------------------------------------

function SamlConfigFields({
  metadataUrl,
  setMetadataUrl,
  idpCert,
  setIdpCert,
  ssoUrl,
  setSsoUrl,
  entityId,
  setEntityId,
}: {
  metadataUrl: string;
  setMetadataUrl: (v: string) => void;
  idpCert: string;
  setIdpCert: (v: string) => void;
  ssoUrl: string;
  setSsoUrl: (v: string) => void;
  entityId: string;
  setEntityId: (v: string) => void;
}) {
  return (
    <div className="space-y-3">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        IdP configuration
      </p>
      <div className="space-y-2">
        <Label htmlFor="saml-metadata-url">Metadata URL</Label>
        <Input
          id="saml-metadata-url"
          type="url"
          value={metadataUrl}
          onChange={(e) => setMetadataUrl(e.target.value)}
          placeholder="https://idp.example.com/metadata.xml"
        />
        <p className="text-xs text-muted-foreground">
          Provide the metadata URL or fill in the fields below manually.
        </p>
      </div>
      <div className="space-y-2">
        <Label htmlFor="saml-sso-url">SSO URL</Label>
        <Input
          id="saml-sso-url"
          type="url"
          value={ssoUrl}
          onChange={(e) => setSsoUrl(e.target.value)}
          placeholder="https://idp.example.com/sso"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="saml-entity-id">Entity ID</Label>
        <Input
          id="saml-entity-id"
          value={entityId}
          onChange={(e) => setEntityId(e.target.value)}
          placeholder="https://idp.example.com/entity"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="saml-idp-cert">IdP certificate (PEM)</Label>
        <textarea
          id="saml-idp-cert"
          value={idpCert}
          onChange={(e) => setIdpCert(e.target.value)}
          rows={4}
          placeholder="-----BEGIN CERTIFICATE-----&#10;…&#10;-----END CERTIFICATE-----"
          className="w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1.5 font-mono text-xs transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
        />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// OIDC config fields
// ---------------------------------------------------------------------------

function OidcConfigFields({
  issuer,
  setIssuer,
  clientId,
  setClientId,
  clientSecret,
  setClientSecret,
  scopes,
  setScopes,
}: {
  issuer: string;
  setIssuer: (v: string) => void;
  clientId: string;
  setClientId: (v: string) => void;
  clientSecret: string;
  setClientSecret: (v: string) => void;
  scopes: string;
  setScopes: (v: string) => void;
}) {
  return (
    <div className="space-y-3">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Provider configuration
      </p>
      <div className="space-y-2">
        <Label htmlFor="oidc-issuer">Issuer URL</Label>
        <Input
          id="oidc-issuer"
          type="url"
          value={issuer}
          onChange={(e) => setIssuer(e.target.value)}
          placeholder="https://login.microsoftonline.com/{tenant}/v2.0"
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="oidc-client-id">Client ID</Label>
          <Input
            id="oidc-client-id"
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="oidc-client-secret">Client secret</Label>
          <Input
            id="oidc-client-secret"
            type="password"
            value={clientSecret}
            onChange={(e) => setClientSecret(e.target.value)}
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="oidc-scopes">Scopes</Label>
        <Input
          id="oidc-scopes"
          value={scopes}
          onChange={(e) => setScopes(e.target.value)}
          placeholder="openid email profile"
        />
        <p className="text-xs text-muted-foreground">Space- or comma-separated OAuth scopes.</p>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Connection detail dialog (view / edit + SCIM tokens)
// ---------------------------------------------------------------------------

function ConnectionDetailDialog({
  projectId,
  connectionId,
  open,
  onOpenChange,
  onUpdated,
}: {
  projectId: string;
  connectionId: string;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onUpdated: () => void;
}) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminSsoConnectionsById({
          path: { project_id: projectId, id: connectionId },
        }),
      ),
    [projectId, connectionId],
  );
  const conn = data?.connection;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{conn?.name ?? 'Connection'}</DialogTitle>
          <DialogDescription>
            View and update the connection configuration, or manage SCIM provisioning tokens.
          </DialogDescription>
        </DialogHeader>

        {loading && <LoadingState />}
        {error && <ErrorState error={error} onRetry={reload} />}

        {!loading && !error && conn && (
          <Tabs defaultValue="config">
            <TabsList>
              <TabsTrigger value="config">Configuration</TabsTrigger>
              <TabsTrigger value="scim">SCIM tokens</TabsTrigger>
            </TabsList>

            <TabsContent value="config" className="pt-4">
              <ConnectionEditForm
                projectId={projectId}
                connection={conn}
                onUpdated={() => {
                  reload();
                  onUpdated();
                }}
              />
            </TabsContent>

            <TabsContent value="scim" className="pt-4">
              <ScimTokensPanel projectId={projectId} connectionId={connectionId} />
            </TabsContent>
          </Tabs>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Connection edit form
// ---------------------------------------------------------------------------

function ConnectionEditForm({
  projectId,
  connection,
  onUpdated,
}: {
  projectId: string;
  connection: SsoConnection;
  onUpdated: () => void;
}) {
  const cfg = (connection.config ?? {}) as Record<string, string>;

  const [name, setName] = useState(connection.name ?? '');
  const [domains, setDomains] = useState((connection.domains ?? []).join(', '));

  // SAML
  const [samlMetadataUrl, setSamlMetadataUrl] = useState(cfg['metadata_url'] ?? '');
  const [samlIdpCert, setSamlIdpCert] = useState(cfg['idp_cert'] ?? '');
  const [samlSsoUrl, setSamlSsoUrl] = useState(cfg['sso_url'] ?? '');
  const [samlEntityId, setSamlEntityId] = useState(cfg['entity_id'] ?? '');

  // OIDC
  const [oidcIssuer, setOidcIssuer] = useState(cfg['issuer'] ?? '');
  const [oidcClientId, setOidcClientId] = useState(cfg['client_id'] ?? '');
  const [oidcClientSecret, setOidcClientSecret] = useState('');
  const [oidcScopes, setOidcScopes] = useState(
    Array.isArray(cfg['scopes'])
      ? (cfg['scopes'] as unknown as string[]).join(' ')
      : (cfg['scopes'] ?? 'openid email profile'),
  );

  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function buildConfig(): Record<string, unknown> {
    if (connection.type === 'saml') {
      return {
        metadata_url: samlMetadataUrl || undefined,
        idp_cert: samlIdpCert || undefined,
        sso_url: samlSsoUrl || undefined,
        entity_id: samlEntityId || undefined,
      };
    }
    return {
      issuer: oidcIssuer || undefined,
      client_id: oidcClientId || undefined,
      ...(oidcClientSecret ? { client_secret: oidcClientSecret } : {}),
      scopes: oidcScopes
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean),
    };
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        patchV1ProjectsByProjectIdAdminSsoConnectionsById({
          path: { project_id: projectId, id: connection.id! },
          body: {
            name,
            domains: domains
              ? domains
                  .split(/[\s,]+/)
                  .map((d) => d.trim())
                  .filter(Boolean)
              : [],
            config: buildConfig(),
          },
        }),
      );
      toast.success('Connection updated');
      onUpdated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Update failed');
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      {/* SP metadata link for SAML */}
      {connection.type === 'saml' && connection.sp_metadata_url && (
        <div className="flex items-center justify-between rounded-lg border bg-muted/40 px-3 py-2 text-sm">
          <span className="text-muted-foreground">SP metadata URL</span>
          <div className="flex items-center gap-1.5">
            <CopyButton value={connection.sp_metadata_url} />
            <a
              href={connection.sp_metadata_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground"
            >
              <ExternalLink className="size-3.5" />
            </a>
          </div>
        </div>
      )}

      {/* Name */}
      <div className="space-y-2">
        <Label htmlFor="ed-name">Name</Label>
        <Input
          id="ed-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
      </div>

      {/* Domains */}
      <div className="space-y-2">
        <Label htmlFor="ed-domains">Domains</Label>
        <Input
          id="ed-domains"
          value={domains}
          onChange={(e) => setDomains(e.target.value)}
          placeholder="acme.com, acme.org"
        />
      </div>

      {/* Connection ID (read-only) */}
      {connection.id && (
        <div className="space-y-1">
          <p className="text-xs text-muted-foreground">Connection ID</p>
          <CopyButton value={connection.id} />
        </div>
      )}

      {/* Type-specific config */}
      {connection.type === 'saml' ? (
        <SamlConfigFields
          metadataUrl={samlMetadataUrl}
          setMetadataUrl={setSamlMetadataUrl}
          idpCert={samlIdpCert}
          setIdpCert={setSamlIdpCert}
          ssoUrl={samlSsoUrl}
          setSsoUrl={setSamlSsoUrl}
          entityId={samlEntityId}
          setEntityId={setSamlEntityId}
        />
      ) : (
        <OidcConfigFields
          issuer={oidcIssuer}
          setIssuer={setOidcIssuer}
          clientId={oidcClientId}
          setClientId={setOidcClientId}
          clientSecret={oidcClientSecret}
          setClientSecret={setOidcClientSecret}
          scopes={oidcScopes}
          setScopes={setOidcScopes}
        />
      )}

      {err && <p className="text-sm text-destructive">{err}</p>}

      <div className="flex justify-end">
        <Button type="submit" disabled={busy || !name}>
          {busy && <Loader2 className="size-4 animate-spin" />}
          Save changes
        </Button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------------------
// SCIM tokens panel
// ---------------------------------------------------------------------------

function ScimTokensPanel({
  projectId,
  connectionId,
}: {
  projectId: string;
  connectionId: string;
}) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokens({
          path: { project_id: projectId, id: connectionId },
        }),
      ),
    [projectId, connectionId],
  );
  const tokens = (data?.data ?? []) as ScimTokenRecord[];

  // Minted token shown once after creation
  const [mintedSecret, setMintedSecret] = useState<string | null>(null);

  async function onTokenCreated(secret: string) {
    setMintedSecret(secret);
    reload();
  }

  return (
    <div className="space-y-4">
      {/* Minted secret banner */}
      {mintedSecret && (
        <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-3 text-sm">
          <p className="mb-1.5 font-medium">Token created — copy it now, it won't be shown again.</p>
          <div className="flex items-center gap-2">
            <CopyButton value={mintedSecret} className="flex-1" />
            <Button
              size="sm"
              variant="outline"
              onClick={() => setMintedSecret(null)}
            >
              Dismiss
            </Button>
          </div>
        </div>
      )}

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          SCIM bearer tokens used by your identity provider for directory provisioning.
        </p>
        <NewScimTokenDialog
          projectId={projectId}
          connectionId={connectionId}
          onCreated={onTokenCreated}
        />
      </div>

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}

      {!loading && !error && tokens.length === 0 && (
        <EmptyState
          title="No SCIM tokens"
          description="Create a token to allow your IdP to provision users via SCIM."
          action={
            <NewScimTokenDialog
              projectId={projectId}
              connectionId={connectionId}
              onCreated={onTokenCreated}
            />
          }
        />
      )}

      {!loading && !error && tokens.length > 0 && (
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40">
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">ID</th>
                <th className="w-12 px-4 py-2.5" />
              </tr>
            </thead>
            <tbody>
              {tokens.map((t, i) => (
                <tr key={t.id ?? i} className="border-b last:border-0">
                  <td className="px-4 py-2.5">{t.name ?? '—'}</td>
                  <td className="px-4 py-2.5">
                    {t.id ? <CopyButton value={t.id} /> : '—'}
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    <RevokeScimTokenButton
                      projectId={projectId}
                      connectionId={connectionId}
                      tokenId={t.id!}
                      tokenName={t.name}
                      onRevoked={reload}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// New SCIM token dialog
// ---------------------------------------------------------------------------

function NewScimTokenDialog({
  projectId,
  connectionId,
  onCreated,
}: {
  projectId: string;
  connectionId: string;
  onCreated: (secret: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [tokenName, setTokenName] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokens({
          path: { project_id: projectId, id: connectionId },
          body: {
            name: tokenName,
            expires_at: expiresAt || undefined,
          },
        }),
      );
      toast.success('SCIM token created');
      setOpen(false);
      setTokenName('');
      setExpiresAt('');
      if (res.secret) {
        onCreated(res.secret);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create token');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) {
          setTokenName('');
          setExpiresAt('');
          setErr(null);
        }
      }}
    >
      <DialogTrigger render={<Button size="sm" />}>
        <Plus className="size-3.5" />
        New token
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New SCIM token</DialogTitle>
          <DialogDescription>
            The secret is shown once immediately after creation. Store it securely in your IdP.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="scim-name">Token name</Label>
            <Input
              id="scim-name"
              value={tokenName}
              onChange={(e) => setTokenName(e.target.value)}
              required
              autoFocus
              placeholder="Okta provisioning"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="scim-expires">Expires at (optional)</Label>
            <Input
              id="scim-expires"
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !tokenName}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create token
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Revoke SCIM token button with confirm
// ---------------------------------------------------------------------------

function RevokeScimTokenButton({
  projectId,
  connectionId,
  tokenId,
  tokenName,
  onRevoked,
}: {
  projectId: string;
  connectionId: string;
  tokenId: string;
  tokenName?: string;
  onRevoked: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  async function revoke() {
    setBusy(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminSsoConnectionsByIdScimTokensByTokenId({
          path: { project_id: projectId, id: connectionId, token_id: tokenId },
        }),
      );
      toast.success('Token revoked');
      setOpen(false);
      onRevoked();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Revoke failed');
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Button
        variant="ghost"
        size="icon-sm"
        className="text-destructive hover:bg-destructive/10 hover:text-destructive"
        onClick={() => setOpen(true)}
        title="Revoke token"
      >
        <Trash2 className="size-3.5" />
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke SCIM token</DialogTitle>
            <DialogDescription>
              Revoke <strong>{tokenName ?? tokenId}</strong>? Any IdP using this token will
              immediately lose the ability to provision users.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={revoke} disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Revoke token
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
