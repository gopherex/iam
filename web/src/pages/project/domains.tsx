import type { Domain } from '@gopherex/iam-sdk';
import {
  deleteV1ProjectsByProjectIdAdminDomainsByDomainId,
  getV1ProjectsByProjectIdAdminDomains,
  postV1ProjectsByProjectIdAdminDomains,
  postV1ProjectsByProjectIdAdminDomainsByDomainIdVerify,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import {
  CheckCircle2,
  Clock,
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
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// DNS verification record dialog (shown after domain registration)
// ---------------------------------------------------------------------------

type VerificationRecord = {
  type?: string;
  name?: string;
  value?: string;
};

function DnsRecordDialog({
  open,
  onOpenChange,
  domain,
  record,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  domain?: string;
  record: VerificationRecord;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add DNS verification record</DialogTitle>
          <DialogDescription>
            Add the following TXT record to your DNS provider for{' '}
            <strong>{domain}</strong>. Once propagated, use{' '}
            <em>Verify</em> in the domain menu to confirm ownership.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="rounded-lg border bg-muted/40 p-4 space-y-3 text-sm">
            <RecordField label="Type" value={record.type ?? ''} />
            <RecordField label="Name" value={record.name ?? ''} />
            <RecordField label="Value" value={record.value ?? ''} />
          </div>
          <p className="text-xs text-muted-foreground">
            DNS changes can take up to 48 hours to propagate. You can trigger a
            re-check at any time from the domain row actions.
          </p>
        </div>
        <DialogFooter>
          <DialogClose render={<Button />}>Got it</DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RecordField({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </span>
      {value ? (
        <CopyButton value={value} className="w-full justify-between" />
      ) : (
        <span className="text-muted-foreground">—</span>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Register domain dialog
// ---------------------------------------------------------------------------

function RegisterDomainDialog({
  projectId,
  onCreated,
}: {
  projectId: string;
  onCreated: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [domain, setDomain] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // After creation — show DNS record dialog
  const [dnsOpen, setDnsOpen] = useState(false);
  const [dnsRecord, setDnsRecord] = useState<VerificationRecord>({});
  const [createdDomain, setCreatedDomain] = useState<string | undefined>();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminDomains({
          path: { project_id: projectId },
          body: { domain },
        }),
      );
      setOpen(false);
      setDomain('');
      onCreated();
      if (res.verification_record) {
        setCreatedDomain(res.domain?.domain);
        setDnsRecord(res.verification_record);
        setDnsOpen(true);
      } else {
        toast.success('Domain registered');
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to register domain');
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger render={<Button />}>
          <Plus className="size-4" />
          Register domain
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Register domain</DialogTitle>
            <DialogDescription>
              Add a verified email or SSO domain for this project. You will receive a
              DNS TXT record to prove ownership.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="domain-name">Domain</Label>
              <Input
                id="domain-name"
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
                placeholder="e.g. example.com"
                required
                autoFocus
              />
              <p className="text-xs text-muted-foreground">
                Enter a bare domain without protocol or path.
              </p>
            </div>
            {err && <p className="text-sm text-destructive">{err}</p>}
            <DialogFooter>
              <Button type="submit" disabled={busy || !domain.trim()}>
                {busy && <Loader2 className="size-4 animate-spin" />}
                Register
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <DnsRecordDialog
        open={dnsOpen}
        onOpenChange={setDnsOpen}
        domain={createdDomain}
        record={dnsRecord}
      />
    </>
  );
}

// ---------------------------------------------------------------------------
// Delete confirm dialog
// ---------------------------------------------------------------------------

function DeleteDomainDialog({
  projectId,
  domain,
  onDeleted,
  open,
  onOpenChange,
}: {
  projectId: string;
  domain: Domain;
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
        deleteV1ProjectsByProjectIdAdminDomainsByDomainId({
          path: { project_id: projectId, domain_id: domain.id! },
        }),
      );
      onOpenChange(false);
      onDeleted();
      toast.success('Domain removed');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to delete domain');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Remove domain</DialogTitle>
          <DialogDescription>
            Permanently remove <strong>{domain.domain}</strong> from this project? This
            cannot be undone. Any SSO connections linked to this domain may be affected.
          </DialogDescription>
        </DialogHeader>
        {err && <p className="text-sm text-destructive">{err}</p>}
        <DialogFooter showCloseButton>
          <Button variant="destructive" onClick={confirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            <Trash2 className="size-4" />
            Remove domain
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Per-row action menu
// ---------------------------------------------------------------------------

function DomainActions({
  projectId,
  domain,
  onRefresh,
}: {
  projectId: string;
  domain: Domain;
  onRefresh: () => void;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [verifying, setVerifying] = useState(false);

  async function verify() {
    setVerifying(true);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminDomainsByDomainIdVerify({
          path: { project_id: projectId, domain_id: domain.id! },
        }),
      );
      onRefresh();
      if (res.domain?.status === 'verified') {
        toast.success(`${domain.domain} is now verified`);
      } else {
        toast.info('Verification check complete — DNS record not yet detected.');
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Verification failed');
    } finally {
      setVerifying(false);
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={<Button variant="ghost" size="icon-sm" aria-label="Domain actions" />}
        >
          {verifying ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <MoreHorizontal className="size-4" />
          )}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={verify} disabled={verifying}>
            <RefreshCw className="size-4" />
            Verify now
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="size-4" />
            Remove
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {deleteOpen && (
        <DeleteDomainDialog
          projectId={projectId}
          domain={domain}
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

function buildColumns(projectId: string, reload: () => void): ColumnDef<Domain, unknown>[] {
  return [
    {
      id: 'domain',
      header: 'Domain',
      accessorFn: (row) => row.domain ?? '',
      cell: ({ row }) => {
        const d = row.original;
        return (
          <div className="flex flex-col gap-0.5">
            <span className="font-medium">{d.domain}</span>
            {d.id && <CopyButton value={d.id} />}
          </div>
        );
      },
    },
    {
      id: 'status',
      header: 'Status',
      accessorFn: (row) => row.status ?? 'pending',
      cell: ({ row }) => {
        const status = row.original.status;
        if (status === 'verified') {
          return (
            <Badge variant="secondary" className="gap-1.5">
              <CheckCircle2 className="size-3 text-green-600" />
              Verified
            </Badge>
          );
        }
        return (
          <Badge variant="outline" className="gap-1.5">
            <Clock className="size-3 text-amber-500" />
            Pending
          </Badge>
        );
      },
    },
    {
      id: 'verified_at',
      header: 'Verified at',
      accessorFn: (row) => row.verified_at ?? '',
      cell: ({ row }) => {
        const ts = row.original.verified_at;
        if (!ts) return <span className="text-muted-foreground">—</span>;
        return (
          <span className="text-sm text-muted-foreground">
            {new Date(ts as string).toLocaleDateString(undefined, { dateStyle: 'medium' })}
          </span>
        );
      },
    },
    {
      id: 'connection_id',
      header: 'Connection',
      accessorFn: (row) => row.connection_id ?? '',
      cell: ({ row }) => {
        const cid = row.original.connection_id;
        if (!cid) return <span className="text-muted-foreground">—</span>;
        return <CopyButton value={cid} />;
      },
    },
    {
      id: 'actions',
      header: '',
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex justify-end">
          <DomainActions
            projectId={projectId}
            domain={row.original}
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

export function DomainsPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminDomains({
          path: { project_id: projectId! },
        }),
      ),
    [projectId],
  );

  const domains = data?.data ?? [];
  const columns = buildColumns(projectId!, reload);

  return (
    <div>
      <PageHeader
        title="Domains"
        description="Verified email and SSO domains for this project. DNS ownership must be confirmed before a domain can be used."
        actions={<RegisterDomainDialog projectId={projectId!} onCreated={reload} />}
      />
      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && domains.length === 0 && (
        <EmptyState
          title="No domains registered"
          description="Register a domain to enable email or SSO sign-in for users on that domain."
          action={<RegisterDomainDialog projectId={projectId!} onCreated={reload} />}
        />
      )}
      {!loading && !error && domains.length > 0 && (
        <DataTable
          columns={columns}
          data={domains}
          searchPlaceholder="Search domains…"
          emptyMessage="No domains match your search."
        />
      )}
    </div>
  );
}
