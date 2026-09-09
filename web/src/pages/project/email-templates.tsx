import {
  getV1ProjectsByProjectIdAdminEmailTemplates,
  patchV1ProjectsByProjectIdAdminEmailTemplatesById,
  postV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview,
  postV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import { Eye, Loader2, Mail, MoreHorizontal, Pencil } from 'lucide-react';
import { useEffect, useState } from 'react';
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
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Types (the admin API returns a loose map id → template row)
// ---------------------------------------------------------------------------

type TemplateRow = {
  id: string;
  name?: string;
  locale?: string;
  subject?: string;
  text?: string;
  html?: string;
  customized?: boolean;
};

type Preview = { subject?: string; text?: string; html?: string };

// ---------------------------------------------------------------------------
// Shared bits
// ---------------------------------------------------------------------------

// The ui kit has no textarea; mirror Input's classes for a multi-line field.
function Textarea({
  className,
  ...props
}: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        'w-full min-h-24 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 font-mono text-xs md:text-sm dark:bg-input/30',
        className,
      )}
      {...props}
    />
  );
}

function templateId(row: TemplateRow): string {
  // List ids are "key" or "key:locale"; the PATCH path takes that exact id.
  return row.locale ? `${row.id}:${row.locale}` : row.id;
}

// ---------------------------------------------------------------------------
// Edit dialog (with live draft preview)
// ---------------------------------------------------------------------------

const SAMPLE_DATA_DEFAULT = `{
  "code": "123456",
  "link": "https://example.test/auth/callback?token=sample-token",
  "email": "user@example.com",
  "invite_token": "inv_sample",
  "reason": "sample reason"
}`;

function EditTemplateDialog({
  projectId,
  row,
  open,
  onOpenChange,
  onSaved,
}: {
  projectId: string;
  row: TemplateRow | null;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onSaved: () => void;
}) {
  const [subject, setSubject] = useState('');
  const [text, setText] = useState('');
  const [html, setHtml] = useState('');
  const [locale, setLocale] = useState('');
  const [sampleData, setSampleData] = useState(SAMPLE_DATA_DEFAULT);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [previewErr, setPreviewErr] = useState<string | null>(null);

  // Populate fields each time a different row opens the dialog.
  if (row && loadedFor !== templateId(row)) {
    setLoadedFor(templateId(row));
    setSubject(row.subject ?? '');
    setText(row.text ?? '');
    setHtml(row.html ?? '');
    setLocale(row.locale ?? '');
    setSampleData(SAMPLE_DATA_DEFAULT);
    setErr(null);
    setPreview(null);
    setPreviewErr(null);
  }

  // Live preview: re-render the draft (debounced) whenever the editor body,
  // locale, or sample data changes. Failures surface next to the result, they
  // must not block editing.
  useEffect(() => {
    if (!open || !row) return;
    const timer = window.setTimeout(() => {
      void (async () => {
        try {
          const res = await call(
            postV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview({
              path: { project_id: projectId, id: templateId(row) },
              body: {
                locale: locale.trim() || undefined,
                subject: subject.trim() || undefined,
                text: text || undefined,
                html: html || undefined,
                data: JSON.parse(sampleData || '{}') as Record<string, unknown>,
              },
            }),
          );
          setPreview(res as Preview);
          setPreviewErr(null);
        } catch (e) {
          setPreview(null);
          setPreviewErr(e instanceof Error ? e.message : 'Failed to render preview');
        }
      })();
    }, 600);
    return () => window.clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, subject, text, html, locale, sampleData]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!row) return;
    setBusy(true);
    setErr(null);
    try {
      await call(
        patchV1ProjectsByProjectIdAdminEmailTemplatesById({
          path: { project_id: projectId, id: templateId(row) },
          body: {
            subject: subject.trim(),
            text,
            html,
            locale: locale.trim() || undefined,
          },
        }),
      );
      toast.success('Template saved');
      onSaved();
      onOpenChange(false);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to save template');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Edit template — {row?.name ?? row?.id}</DialogTitle>
          <DialogDescription>
            Go text/template syntax: {'{{.code}}'}, {'{{.link}}'}, etc. The preview below re-renders
            as you type; save writes a project override.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="grid grid-cols-[1fr_160px] gap-4">
            <div className="space-y-2">
              <Label htmlFor="tpl-subject">Subject</Label>
              <Input
                id="tpl-subject"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="Subject line"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tpl-locale">Locale</Label>
              <Input
                id="tpl-locale"
                value={locale}
                onChange={(e) => setLocale(e.target.value)}
                placeholder="en, ru, …"
              />
            </div>
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="tpl-text">Plain text body</Label>
                <Textarea
                  id="tpl-text"
                  value={text}
                  onChange={(e) => setText(e.target.value)}
                  rows={5}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tpl-html">HTML body</Label>
                <Textarea
                  id="tpl-html"
                  value={html}
                  onChange={(e) => setHtml(e.target.value)}
                  rows={8}
                />
              </div>
            </div>
            <div className="space-y-3">
              <div className="space-y-2">
                <Label htmlFor="tpl-data">Sample data (JSON)</Label>
                <Textarea
                  id="tpl-data"
                  value={sampleData}
                  onChange={(e) => setSampleData(e.target.value)}
                  rows={5}
                  className="min-h-20"
                />
              </div>
              {previewErr && <p className="text-sm text-destructive">{previewErr}</p>}
              {preview && (
                <div className="space-y-3">
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">
                      Subject (rendered)
                    </p>
                    <p className="text-sm">{preview.subject || '—'}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">
                      Text (rendered)
                    </p>
                    <pre className="rounded-lg border bg-muted/40 p-3 text-xs whitespace-pre-wrap">
                      {preview.text || '—'}
                    </pre>
                  </div>
                  {preview.html && (
                    <div>
                      <p className="text-xs font-medium text-muted-foreground mb-1">
                        HTML (rendered)
                      </p>
                      <iframe
                        title="Template HTML preview"
                        srcDoc={preview.html}
                        sandbox=""
                        className="w-full h-40 rounded-lg border bg-white"
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>
              Cancel
            </DialogClose>
            <Button type="submit" disabled={busy}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Save
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

function PreviewDialog({
  projectId,
  row,
  open,
  onOpenChange,
}: {
  projectId: string;
  row: TemplateRow | null;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const [locale, setLocale] = useState('');
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function run() {
    if (!row) return;
    setBusy(true);
    setErr(null);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview({
          path: { project_id: projectId, id: templateId(row) },
          body: { locale: locale.trim() || undefined },
        }),
      );
      setPreview(res as Preview);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to render preview');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Preview — {row?.name ?? row?.id}</DialogTitle>
          <DialogDescription>
            Rendered with sample data. The response is capped at 1024 characters per part.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="flex gap-2 items-end">
            <div className="space-y-2 flex-1">
              <Label htmlFor="pv-locale">Locale</Label>
              <Input
                id="pv-locale"
                value={locale}
                onChange={(e) => setLocale(e.target.value)}
                placeholder="en, ru, …"
              />
            </div>
            <Button type="button" variant="outline" onClick={() => void run()} disabled={busy}>
              {busy ? <Loader2 className="size-4 animate-spin" /> : <Eye className="size-4" />}
              Render
            </Button>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          {preview && (
            <div className="space-y-3">
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Subject</p>
                <p className="text-sm">{preview.subject || '—'}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Text</p>
                <pre className="rounded-lg border bg-muted/40 p-3 text-xs whitespace-pre-wrap">
                  {preview.text || '—'}
                </pre>
              </div>
              {preview.html && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1">HTML</p>
                  <pre className="rounded-lg border bg-muted/40 p-3 text-xs whitespace-pre-wrap">
                    {preview.html}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <DialogClose render={<Button />}>Done</DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Send-test dialog
// ---------------------------------------------------------------------------

function SendTestDialog({
  projectId,
  row,
  open,
  onOpenChange,
}: {
  projectId: string;
  row: TemplateRow | null;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const [to, setTo] = useState('');
  const [locale, setLocale] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!row || !to.trim()) return;
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest({
          path: { project_id: projectId, id: templateId(row) },
          body: { to: to.trim(), locale: locale.trim() || undefined },
        }),
      );
      toast.success(`Test email sent to ${to.trim()}`);
      onOpenChange(false);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to send test email');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Send test — {row?.name ?? row?.id}</DialogTitle>
          <DialogDescription>
            Sends this template to the address below through the project's configured SMTP provider.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="st-to">To</Label>
            <Input
              id="st-to"
              type="email"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              placeholder="you@example.com"
              autoFocus
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="st-locale">Locale</Label>
            <Input
              id="st-locale"
              value={locale}
              onChange={(e) => setLocale(e.target.value)}
              placeholder="en, ru, …"
            />
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={busy || !to.trim()}>
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
// Page root
// ---------------------------------------------------------------------------

export function EmailTemplatesPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminEmailTemplates({ path: { project_id: projectId! } })),
    [projectId],
  );
  const rows = Object.entries((data ?? {}) as Record<string, TemplateRow>)
    .map(([key, value]) => ({ ...value, id: value.id ?? key }))
    .sort((a, b) => a.id.localeCompare(b.id));

  const [editRow, setEditRow] = useState<TemplateRow | null>(null);
  const [previewRow, setPreviewRow] = useState<TemplateRow | null>(null);
  const [testRow, setTestRow] = useState<TemplateRow | null>(null);

  const columns: ColumnDef<TemplateRow>[] = [
    {
      id: 'id',
      header: 'Key',
      accessorKey: 'id',
      cell: ({ row }) => <span className="font-mono text-xs">{templateId(row.original)}</span>,
    },
    {
      id: 'name',
      header: 'Name',
      accessorFn: (t) => t.name ?? '',
      cell: ({ row }) => <span className="font-medium">{row.original.name ?? '—'}</span>,
    },
    {
      id: 'subject',
      header: 'Subject',
      accessorFn: (t) => t.subject ?? '',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground truncate max-w-72 inline-block align-bottom">
          {row.original.subject ?? '—'}
        </span>
      ),
    },
    {
      id: 'customized',
      header: 'State',
      accessorFn: (t) => (t.customized ? 'customized' : 'builtin'),
      cell: ({ row }) => (
        <Badge variant={row.original.customized ? 'default' : 'secondary'}>
          {row.original.customized ? 'customized' : 'built-in'}
        </Badge>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex justify-end">
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />}>
              <MoreHorizontal className="size-4" />
              <span className="sr-only">Actions</span>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setEditRow(row.original)}>
                <Pencil className="size-4" /> Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setPreviewRow(row.original)}>
                <Eye className="size-4" /> Preview
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setTestRow(row.original)}>
                <Mail className="size-4" /> Send test
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Email Templates"
        description="System email copy for this project: built-in defaults and per-project overrides with live preview and test sends."
      />

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && rows.length === 0 && (
        <EmptyState
          title="No templates"
          description="Built-in templates appear here once the notification layer knows about them."
        />
      )}
      {!loading && !error && rows.length > 0 && (
        <DataTable
          columns={columns}
          data={rows}
          searchPlaceholder="Search templates…"
          emptyMessage="No templates match your search."
        />
      )}

      <EditTemplateDialog
        projectId={projectId!}
        row={editRow}
        open={Boolean(editRow)}
        onOpenChange={(o) => {
          if (!o) setEditRow(null);
        }}
        onSaved={reload}
      />
      <PreviewDialog
        projectId={projectId!}
        row={previewRow}
        open={Boolean(previewRow)}
        onOpenChange={(o) => {
          if (!o) setPreviewRow(null);
        }}
      />
      <SendTestDialog
        projectId={projectId!}
        row={testRow}
        open={Boolean(testRow)}
        onOpenChange={(o) => {
          if (!o) setTestRow(null);
        }}
      />
    </div>
  );
}
