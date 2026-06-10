import type { EmailProvider, SmsProvider } from '@gopherex/iam-sdk';
import {
  deleteV1ProjectsByProjectIdAdminEmailProvidersById,
  deleteV1ProjectsByProjectIdAdminSmsProvidersById,
  getV1ProjectsByProjectIdAdminEmailProviders,
  getV1ProjectsByProjectIdAdminEmailTemplates,
  getV1ProjectsByProjectIdAdminSmsProviders,
  patchV1ProjectsByProjectIdAdminEmailProvidersById,
  patchV1ProjectsByProjectIdAdminEmailTemplatesById,
  patchV1ProjectsByProjectIdAdminSmsProvidersById,
  postV1ProjectsByProjectIdAdminEmailProviders,
  postV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview,
  postV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest,
  postV1ProjectsByProjectIdAdminSmsProviders,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  Eye,
  Loader2,
  MoreHorizontal,
  Plus,
  Send,
  Trash2,
} from 'lucide-react';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';
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
// Types
// ---------------------------------------------------------------------------

/** A template as returned by the loose { [key: string]: unknown } response. */
type EmailTemplate = {
  id?: string;
  name?: string;
  subject?: string;
  html?: string;
  text?: string;
  locale?: string;
  customized?: boolean;
  [key: string]: unknown;
};

// ---------------------------------------------------------------------------
// Secret config fields that should be rendered as password inputs
// ---------------------------------------------------------------------------

const SECRET_KEYS = new Set(['password', 'api_key', 'secret', 'token', 'access_token', 'auth_token']);

// ---------------------------------------------------------------------------
// Key/value config editor
// ---------------------------------------------------------------------------

type ConfigEntry = { key: string; value: string };

function configToEntries(config?: Record<string, unknown>): ConfigEntry[] {
  if (!config) return [];
  return Object.entries(config).map(([key, value]) => ({
    key,
    value: typeof value === 'string' ? value : JSON.stringify(value),
  }));
}

function entriesToConfig(entries: ConfigEntry[]): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const { key, value } of entries) {
    if (!key.trim()) continue;
    out[key.trim()] = value;
  }
  return out;
}

function ConfigEditor({
  entries,
  onChange,
}: {
  entries: ConfigEntry[];
  onChange: (entries: ConfigEntry[]) => void;
}) {
  function setEntry(index: number, field: 'key' | 'value', val: string) {
    const next = entries.map((e, i) => (i === index ? { ...e, [field]: val } : e));
    onChange(next);
  }

  function removeEntry(index: number) {
    onChange(entries.filter((_, i) => i !== index));
  }

  function addEntry() {
    onChange([...entries, { key: '', value: '' }]);
  }

  return (
    <div className="space-y-2">
      {entries.map((entry, i) => {
        const isSecret = SECRET_KEYS.has(entry.key.toLowerCase());
        return (
          <div key={i} className="flex gap-2">
            <Input
              placeholder="key"
              value={entry.key}
              onChange={(e) => setEntry(i, 'key', e.target.value)}
              className="flex-1 font-mono text-xs"
            />
            <Input
              placeholder="value"
              type={isSecret ? 'password' : 'text'}
              value={entry.value}
              onChange={(e) => setEntry(i, 'value', e.target.value)}
              className="flex-1 font-mono text-xs"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              onClick={() => removeEntry(i)}
              title="Remove"
            >
              <Trash2 className="size-3.5" />
            </Button>
          </div>
        );
      })}
      <Button type="button" variant="outline" size="sm" onClick={addEntry}>
        <Plus className="size-3.5" />
        Add field
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Confirm delete dialog
// ---------------------------------------------------------------------------

function ConfirmDeleteDialog({
  open,
  onOpenChange,
  label,
  onConfirm,
  busy,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  label: string;
  onConfirm: () => void;
  busy: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Delete provider</DialogTitle>
          <DialogDescription>
            Permanently delete <strong>{label}</strong>? This cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button variant="destructive" onClick={onConfirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Email provider form dialog (create + edit)
// ---------------------------------------------------------------------------

function EmailProviderDialog({
  projectId,
  provider,
  trigger,
  onSaved,
}: {
  projectId: string;
  provider?: EmailProvider;
  trigger: React.ReactElement;
  onSaved: () => void;
}) {
  const isEdit = Boolean(provider?.id);
  const [open, setOpen] = useState(false);
  const [type, setType] = useState(provider?.type ?? 'smtp');
  const [enabled, setEnabled] = useState(provider?.enabled ?? true);
  const [entries, setEntries] = useState<ConfigEntry[]>(() =>
    configToEntries(provider?.config as Record<string, unknown> | undefined),
  );
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function reset() {
    setType(provider?.type ?? 'smtp');
    setEnabled(provider?.enabled ?? true);
    setEntries(configToEntries(provider?.config as Record<string, unknown> | undefined));
    setErr(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const body: EmailProvider = { type, enabled, config: entriesToConfig(entries) };
      if (isEdit) {
        await call(
          patchV1ProjectsByProjectIdAdminEmailProvidersById({
            path: { project_id: projectId, id: provider!.id! },
            body,
          }),
        );
        toast.success('Email provider updated');
      } else {
        await call(
          postV1ProjectsByProjectIdAdminEmailProviders({
            path: { project_id: projectId },
            body,
          }),
        );
        toast.success('Email provider created');
      }
      setOpen(false);
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to save provider');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) reset();
      }}
    >
      <DialogTrigger render={trigger} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit email provider' : 'New email provider'}</DialogTitle>
          <DialogDescription>
            Configure an SMTP server or transactional email API for sending messages.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="ep-type">Type</Label>
            <Input
              id="ep-type"
              value={type}
              onChange={(e) => setType(e.target.value)}
              placeholder="e.g. smtp, sendgrid, mailgun"
              required
              autoFocus={!isEdit}
            />
          </div>
          <div className="space-y-2">
            <Label>Configuration</Label>
            <p className="text-xs text-muted-foreground">
              Common keys: <code>host</code>, <code>port</code>, <code>from</code>,{' '}
              <code>username</code>, <code>password</code>, <code>api_key</code>. Secret fields are
              masked.
            </p>
            <ConfigEditor entries={entries} onChange={setEntries} />
          </div>
          <div className="flex items-center gap-2">
            <Switch
              checked={enabled}
              onCheckedChange={setEnabled}
              id="ep-enabled"
            />
            <Label htmlFor="ep-enabled">Enabled</Label>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !type}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              {isEdit ? 'Save changes' : 'Create provider'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// SMS provider form dialog (create + edit)
// ---------------------------------------------------------------------------

function SmsProviderDialog({
  projectId,
  provider,
  trigger,
  onSaved,
}: {
  projectId: string;
  provider?: SmsProvider;
  trigger: React.ReactElement;
  onSaved: () => void;
}) {
  const isEdit = Boolean(provider?.id);
  const [open, setOpen] = useState(false);
  const [type, setType] = useState(provider?.type ?? 'twilio');
  const [enabled, setEnabled] = useState(provider?.enabled ?? true);
  const [entries, setEntries] = useState<ConfigEntry[]>(() =>
    configToEntries(provider?.config as Record<string, unknown> | undefined),
  );
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function reset() {
    setType(provider?.type ?? 'twilio');
    setEnabled(provider?.enabled ?? true);
    setEntries(configToEntries(provider?.config as Record<string, unknown> | undefined));
    setErr(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const body: SmsProvider = { type, enabled, config: entriesToConfig(entries) };
      if (isEdit) {
        await call(
          patchV1ProjectsByProjectIdAdminSmsProvidersById({
            path: { project_id: projectId, id: provider!.id! },
            body,
          }),
        );
        toast.success('SMS provider updated');
      } else {
        await call(
          postV1ProjectsByProjectIdAdminSmsProviders({
            path: { project_id: projectId },
            body,
          }),
        );
        toast.success('SMS provider created');
      }
      setOpen(false);
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to save provider');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) reset();
      }}
    >
      <DialogTrigger render={trigger} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit SMS provider' : 'New SMS provider'}</DialogTitle>
          <DialogDescription>
            Configure a telephony API for sending OTP and notification messages via SMS.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="sp-type">Type</Label>
            <Input
              id="sp-type"
              value={type}
              onChange={(e) => setType(e.target.value)}
              placeholder="e.g. twilio, vonage"
              required
              autoFocus={!isEdit}
            />
          </div>
          <div className="space-y-2">
            <Label>Configuration</Label>
            <p className="text-xs text-muted-foreground">
              Common keys: <code>account_sid</code>, <code>api_key</code>, <code>from</code>.
              Secret fields are masked.
            </p>
            <ConfigEditor entries={entries} onChange={setEntries} />
          </div>
          <div className="flex items-center gap-2">
            <Switch checked={enabled} onCheckedChange={setEnabled} id="sp-enabled" />
            <Label htmlFor="sp-enabled">Enabled</Label>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !type}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              {isEdit ? 'Save changes' : 'Create provider'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Email providers tab
// ---------------------------------------------------------------------------

function EmailProvidersTab({ projectId }: { projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminEmailProviders({ path: { project_id: projectId } })),
    [projectId],
  );
  const providers = data?.data ?? [];

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<EmailProvider | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [smtpTestOpen, setSmtpTestOpen] = useState(false);

  async function doDelete() {
    if (!deleteTarget?.id) return;
    setDeleteBusy(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminEmailProvidersById({
          path: { project_id: projectId, id: deleteTarget.id },
        }),
      );
      toast.success('Provider deleted');
      setDeleteTarget(null);
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete provider');
    } finally {
      setDeleteBusy(false);
    }
  }

  async function toggleEnabled(p: EmailProvider) {
    if (!p.id) return;
    try {
      await call(
        patchV1ProjectsByProjectIdAdminEmailProvidersById({
          path: { project_id: projectId, id: p.id },
          body: { ...p, enabled: !p.enabled },
        }),
      );
      toast.success(p.enabled ? 'Provider disabled' : 'Provider enabled');
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to update provider');
    }
  }

  const columns: ColumnDef<EmailProvider>[] = [
    {
      id: 'type',
      header: 'Type',
      accessorKey: 'type',
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.type}</span>
      ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'enabled',
      cell: ({ row }) => (
        <Badge variant={row.original.enabled ? 'default' : 'outline'}>
          {row.original.enabled ? 'Enabled' : 'Disabled'}
        </Badge>
      ),
    },
    {
      id: 'id',
      header: 'ID',
      accessorKey: 'id',
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground">{row.original.id ?? '—'}</span>
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
                <EmailProviderDialog
                  projectId={projectId}
                  provider={p}
                  trigger={<DropdownMenuItem onSelect={(e) => e.preventDefault()}>Edit</DropdownMenuItem>}
                  onSaved={reload}
                />
                <DropdownMenuItem onClick={() => void toggleEnabled(p)}>
                  {p.enabled ? 'Disable' : 'Enable'}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setSmtpTestOpen(true)}>
                  <Send className="size-4" />
                  Send test
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => setDeleteTarget(p)}
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
    <div className="space-y-4">
      <div className="flex justify-end">
        <EmailProviderDialog
          projectId={projectId}
          trigger={
            <Button>
              <Plus className="size-4" />
              New provider
            </Button>
          }
          onSaved={reload}
        />
      </div>

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && providers.length === 0 && (
        <EmptyState
          title="No email providers"
          description="Add an SMTP server or transactional email API to send messages."
          action={
            <EmailProviderDialog
              projectId={projectId}
              trigger={
                <Button>
                  <Plus className="size-4" />
                  New provider
                </Button>
              }
              onSaved={reload}
            />
          }
        />
      )}
      {!loading && !error && providers.length > 0 && (
        <DataTable
          columns={columns}
          data={providers}
          searchPlaceholder="Search providers…"
          emptyMessage="No providers match your search."
        />
      )}

      <ConfirmDeleteDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}
        label={deleteTarget?.type ?? 'this provider'}
        onConfirm={() => void doDelete()}
        busy={deleteBusy}
      />

      <SendTestDialog
        projectId={projectId}
        template={{ id: 'email_verification', name: 'SMTP connectivity test' }}
        open={smtpTestOpen}
        onOpenChange={setSmtpTestOpen}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// SMS providers tab
// ---------------------------------------------------------------------------

function SmsProvidersTab({ projectId }: { projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminSmsProviders({ path: { project_id: projectId } })),
    [projectId],
  );
  const providers = data?.data ?? [];

  const [deleteTarget, setDeleteTarget] = useState<SmsProvider | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);

  async function doDelete() {
    if (!deleteTarget?.id) return;
    setDeleteBusy(true);
    try {
      await call(
        deleteV1ProjectsByProjectIdAdminSmsProvidersById({
          path: { project_id: projectId, id: deleteTarget.id },
        }),
      );
      toast.success('Provider deleted');
      setDeleteTarget(null);
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete provider');
    } finally {
      setDeleteBusy(false);
    }
  }

  async function toggleEnabled(p: SmsProvider) {
    if (!p.id) return;
    try {
      await call(
        patchV1ProjectsByProjectIdAdminSmsProvidersById({
          path: { project_id: projectId, id: p.id },
          body: { ...p, enabled: !p.enabled },
        }),
      );
      toast.success(p.enabled ? 'Provider disabled' : 'Provider enabled');
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to update provider');
    }
  }

  const columns: ColumnDef<SmsProvider>[] = [
    {
      id: 'type',
      header: 'Type',
      accessorKey: 'type',
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.type}</span>
      ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'enabled',
      cell: ({ row }) => (
        <Badge variant={row.original.enabled ? 'default' : 'outline'}>
          {row.original.enabled ? 'Enabled' : 'Disabled'}
        </Badge>
      ),
    },
    {
      id: 'id',
      header: 'ID',
      accessorKey: 'id',
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground">{row.original.id ?? '—'}</span>
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
                <SmsProviderDialog
                  projectId={projectId}
                  provider={p}
                  trigger={<DropdownMenuItem onSelect={(e) => e.preventDefault()}>Edit</DropdownMenuItem>}
                  onSaved={reload}
                />
                <DropdownMenuItem onClick={() => void toggleEnabled(p)}>
                  {p.enabled ? 'Disable' : 'Enable'}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => setDeleteTarget(p)}
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
    <div className="space-y-4">
      <div className="flex justify-end">
        <SmsProviderDialog
          projectId={projectId}
          trigger={
            <Button>
              <Plus className="size-4" />
              New provider
            </Button>
          }
          onSaved={reload}
        />
      </div>

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && providers.length === 0 && (
        <EmptyState
          title="No SMS providers"
          description="Add a telephony API such as Twilio or Vonage to send OTP codes via SMS."
          action={
            <SmsProviderDialog
              projectId={projectId}
              trigger={
                <Button>
                  <Plus className="size-4" />
                  New provider
                </Button>
              }
              onSaved={reload}
            />
          }
        />
      )}
      {!loading && !error && providers.length > 0 && (
        <DataTable
          columns={columns}
          data={providers}
          searchPlaceholder="Search providers…"
          emptyMessage="No providers match your search."
        />
      )}

      <ConfirmDeleteDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}
        label={deleteTarget?.type ?? 'this provider'}
        onConfirm={() => void doDelete()}
        busy={deleteBusy}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Template edit dialog
// ---------------------------------------------------------------------------

function EditTemplateDialog({
  projectId,
  template,
  onSaved,
}: {
  projectId: string;
  template: EmailTemplate;
  onSaved: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [subject, setSubject] = useState(template.subject ?? '');
  const [html, setHtml] = useState(template.html ?? '');
  const [text, setText] = useState(template.text ?? '');
  const [locale, setLocale] = useState(template.locale ?? 'en');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function reset() {
    setSubject(template.subject ?? '');
    setHtml(template.html ?? '');
    setText(template.text ?? '');
    setLocale(template.locale ?? 'en');
    setErr(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!template.id) return;
    setBusy(true);
    setErr(null);
    try {
      await call(
        patchV1ProjectsByProjectIdAdminEmailTemplatesById({
          path: { project_id: projectId, id: template.id },
          body: { subject, html, text, locale },
        }),
      );
      toast.success('Template saved');
      setOpen(false);
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to save template');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) reset();
      }}
    >
      <DialogTrigger render={<DropdownMenuItem onSelect={(e) => e.preventDefault()} />}>
        Edit
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit template — {template.name ?? template.id}</DialogTitle>
          <DialogDescription>
            Modify the subject, HTML body, and plain-text fallback for this email template.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="tpl-subject">Subject</Label>
            <Input
              id="tpl-subject"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Your subject line"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tpl-locale">Locale</Label>
            <Input
              id="tpl-locale"
              value={locale}
              onChange={(e) => setLocale(e.target.value)}
              placeholder="en"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tpl-html">HTML body</Label>
            <textarea
              id="tpl-html"
              value={html}
              onChange={(e) => setHtml(e.target.value)}
              rows={8}
              className="w-full rounded-lg border border-input bg-transparent px-3 py-2 font-mono text-xs outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
              placeholder="<html>…</html>"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tpl-text">Plain-text fallback</Label>
            <textarea
              id="tpl-text"
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={4}
              className="w-full rounded-lg border border-input bg-transparent px-3 py-2 font-mono text-xs outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3"
              placeholder="Plain text version…"
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Save template
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Preview dialog
// ---------------------------------------------------------------------------

function PreviewTemplateDialog({
  projectId,
  template,
}: {
  projectId: string;
  template: EmailTemplate;
}) {
  const [open, setOpen] = useState(false);
  const [locale, setLocale] = useState('');
  const [preview, setPreview] = useState<{ subject?: string; html?: string; text?: string } | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function fetchPreview() {
    if (!template.id) return;
    setPreviewLoading(true);
    setErr(null);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview({
          path: { project_id: projectId, id: template.id },
          body: { locale: locale || undefined },
        }),
      );
      setPreview(res);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Preview failed');
    } finally {
      setPreviewLoading(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setPreview(null);
          setErr(null);
        }
      }}
    >
      <DialogTrigger render={<DropdownMenuItem onSelect={(e) => e.preventDefault()} />}>
        <Eye className="size-4" />
        Preview
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Preview — {template.name ?? template.id}</DialogTitle>
          <DialogDescription>
            Render the template with optional locale. The server substitutes sample values.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="flex gap-2">
            <Input
              value={locale}
              onChange={(e) => setLocale(e.target.value)}
              placeholder="Locale (e.g. en, fr) — optional"
              className="flex-1"
            />
            <Button type="button" onClick={() => void fetchPreview()} disabled={previewLoading}>
              {previewLoading && <Loader2 className="size-4 animate-spin" />}
              Render
            </Button>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          {preview && (
            <div className="space-y-3 rounded-lg border p-4">
              {preview.subject && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Subject</p>
                  <p className="text-sm">{preview.subject}</p>
                </div>
              )}
              {preview.html && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground">HTML</p>
                  <iframe
                    srcDoc={preview.html}
                    className="h-64 w-full rounded border bg-white"
                    sandbox="allow-same-origin"
                    title="HTML preview"
                  />
                </div>
              )}
              {preview.text && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Plain text</p>
                  <pre className="rounded bg-muted px-3 py-2 text-xs whitespace-pre-wrap">
                    {preview.text}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
        <DialogFooter showCloseButton />
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Send-test dialog
// ---------------------------------------------------------------------------

// SendTestDialog is CONTROLLED by the parent (open/onOpenChange) and rendered
// OUTSIDE the row DropdownMenu. Nesting a Dialog trigger inside a Base UI Menu
// makes the menu's close dismiss the dialog immediately, so the trigger is a
// plain menu item that lifts state instead.
function SendTestDialog({
  projectId,
  template,
  open,
  onOpenChange,
}: {
  projectId: string;
  template: EmailTemplate | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [to, setTo] = useState('');
  const [locale, setLocale] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function send(e: React.FormEvent) {
    e.preventDefault();
    if (!template?.id) return;
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest({
          path: { project_id: projectId, id: template.id },
          body: { to, locale: locale || undefined },
        }),
      );
      toast.success(`Test email sent to ${to}`);
      onOpenChange(false);
      setTo('');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to send test email');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { onOpenChange(o); if (!o) setErr(null); }}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Send test email</DialogTitle>
          <DialogDescription>
            Send a rendered copy of <strong>{template?.name ?? template?.id}</strong> to a recipient
            address.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={send} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="st-to">Recipient</Label>
            <Input
              id="st-to"
              type="email"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              placeholder="you@example.com"
              required
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="st-locale">Locale</Label>
            <Input
              id="st-locale"
              value={locale}
              onChange={(e) => setLocale(e.target.value)}
              placeholder="e.g. en (optional)"
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !to}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Send
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Email templates tab
// ---------------------------------------------------------------------------

function EmailTemplatesTab({ projectId }: { projectId: string }) {
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminEmailTemplates({ path: { project_id: projectId } })),
    [projectId],
  );
  const [testTemplate, setTestTemplate] = useState<EmailTemplate | null>(null);

  // The response is { [key: string]: unknown } — coerce to a list.
  const templates: EmailTemplate[] = (() => {
    if (!data) return [];
    if (Array.isArray(data)) return data as EmailTemplate[];
    const d = data as Record<string, unknown>;
    if (Array.isArray(d['data'])) return d['data'] as EmailTemplate[];
    return Object.entries(d).map(([id, val]) =>
      typeof val === 'object' && val !== null ? { id, ...(val as EmailTemplate) } : { id },
    );
  })();

  const columns: ColumnDef<EmailTemplate>[] = [
    {
      id: 'name',
      header: 'Name',
      accessorFn: (t) => t.name ?? t.id ?? '',
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="font-medium">{row.original.name ?? row.original.id ?? '—'}</span>
          <Badge variant={row.original.customized ? 'secondary' : 'outline'}>
            {row.original.customized ? 'Customized' : 'Default'}
          </Badge>
        </div>
      ),
    },
    {
      id: 'subject',
      header: 'Subject',
      accessorKey: 'subject',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.subject ? String(row.original.subject).slice(0, 60) : '—'}
        </span>
      ),
    },
    {
      id: 'locale',
      header: 'Locale',
      accessorKey: 'locale',
      cell: ({ row }) =>
        row.original.locale ? (
          <Badge variant="secondary">{String(row.original.locale)}</Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const t = row.original;
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />}>
                <MoreHorizontal className="size-4" />
                <span className="sr-only">Actions</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <EditTemplateDialog projectId={projectId} template={t} onSaved={reload} />
                <PreviewTemplateDialog projectId={projectId} template={t} />
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setTestTemplate(t)}>
                  <Send className="size-4" />
                  Send test
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
      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && templates.length === 0 && (
        <EmptyState
          title="No email templates"
          description="Email templates are managed by the system. They will appear here once configured."
        />
      )}
      {!loading && !error && templates.length > 0 && (
        <DataTable
          columns={columns}
          data={templates}
          searchPlaceholder="Search templates…"
          emptyMessage="No templates match your search."
        />
      )}

      <SendTestDialog
        projectId={projectId}
        template={testTemplate}
        open={Boolean(testTemplate)}
        onOpenChange={(o) => { if (!o) setTestTemplate(null); }}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page root
// ---------------------------------------------------------------------------

export function ProvidersPage() {
  const { projectId } = useParams();

  return (
    <div>
      <PageHeader
        title="Providers"
        description="Email and SMS delivery providers used to send authentication messages and notifications."
      />
      <Tabs defaultValue="email-providers">
        <TabsList>
          <TabsTrigger value="email-providers">Email providers</TabsTrigger>
          <TabsTrigger value="sms-providers">SMS providers</TabsTrigger>
          <TabsTrigger value="email-templates">Email templates</TabsTrigger>
        </TabsList>
        <TabsContent value="email-providers" className="pt-4">
          <EmailProvidersTab projectId={projectId!} />
        </TabsContent>
        <TabsContent value="sms-providers" className="pt-4">
          <SmsProvidersTab projectId={projectId!} />
        </TabsContent>
        <TabsContent value="email-templates" className="pt-4">
          <EmailTemplatesTab projectId={projectId!} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
