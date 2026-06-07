import type { ReactNode } from 'react';
import type { AccessRequest } from '@gopherex/iam-sdk';
import {
  getV1ProjectsByProjectIdAdminAccessRequests,
  postV1ProjectsByProjectIdAdminAccessRequestsByIdApprove,
  postV1ProjectsByProjectIdAdminAccessRequestsByIdDeny,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import { CheckCircle, Loader2, MoreHorizontal, XCircle } from 'lucide-react';
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
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Label } from '@/components/ui/label';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDate(ts?: string): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleDateString(undefined, { dateStyle: 'medium' });
}

function StatusBadge({ status }: { status?: AccessRequest['status'] }) {
  if (status === 'approved') {
    return <Badge variant="secondary" className="bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400">Approved</Badge>;
  }
  if (status === 'denied') {
    return <Badge variant="destructive">Denied</Badge>;
  }
  return <Badge variant="outline" className="border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-400">Pending</Badge>;
}

// ---------------------------------------------------------------------------
// Detail sheet / dialog
// ---------------------------------------------------------------------------

function DetailDialog({
  request,
  open,
  onOpenChange,
}: {
  request: AccessRequest;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Access Request</DialogTitle>
          <DialogDescription>
            Full details for the request from <strong>{request.email ?? request.id}</strong>.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 text-sm">
          <Row label="ID">
            {request.id ? <CopyButton value={request.id} /> : '—'}
          </Row>
          <Row label="Email">{request.email ?? '—'}</Row>
          <Row label="Status">
            <StatusBadge status={request.status} />
          </Row>
          <Row label="Requested at">{formatDate(request.created_at)}</Row>
          {request.reason && (
            <div className="flex flex-col gap-1">
              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Reason</span>
              <p className="rounded-md bg-muted px-3 py-2 text-sm">{request.reason}</p>
            </div>
          )}
          {request.fields && Object.keys(request.fields).length > 0 && (
            <div className="flex flex-col gap-1">
              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Additional fields</span>
              <pre className="overflow-auto rounded-md bg-muted px-3 py-2 text-xs">
                {JSON.stringify(request.fields, null, 2)}
              </pre>
            </div>
          )}
        </div>

        <DialogFooter showCloseButton />
      </DialogContent>
    </Dialog>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="shrink-0 text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className="text-right">{children}</span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Deny dialog
// ---------------------------------------------------------------------------

function DenyDialog({
  projectId,
  request,
  open,
  onOpenChange,
  onDenied,
}: {
  projectId: string;
  request: AccessRequest;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onDenied: () => void;
}) {
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function confirm() {
    setBusy(true);
    setErr(null);
    try {
      await call(
        postV1ProjectsByProjectIdAdminAccessRequestsByIdDeny({
          path: { project_id: projectId, id: request.id! },
          body: reason ? { reason } : undefined,
        }),
      );
      onOpenChange(false);
      setReason('');
      onDenied();
      toast.success('Request denied');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to deny request');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Deny access request</DialogTitle>
          <DialogDescription>
            Deny the request from <strong>{request.email ?? request.id}</strong>. You may optionally
            provide a reason that will be included in the notification.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="deny-reason">Reason (optional)</Label>
          <textarea
            id="deny-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Explain why this request is being denied…"
            rows={3}
            className="w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1.5 text-sm transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:opacity-50 resize-none"
          />
        </div>

        {err && <p className="text-sm text-destructive">{err}</p>}

        <DialogFooter showCloseButton>
          <Button variant="destructive" onClick={confirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            <XCircle className="size-4" />
            Deny request
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Per-row action menu
// ---------------------------------------------------------------------------

function AccessRequestActions({
  projectId,
  request,
  onRefresh,
}: {
  projectId: string;
  request: AccessRequest;
  onRefresh: () => void;
}) {
  const [detailOpen, setDetailOpen] = useState(false);
  const [denyOpen, setDenyOpen] = useState(false);
  const [approveBusy, setApproveBusy] = useState(false);

  const isPending = request.status === 'pending';

  async function approve() {
    setApproveBusy(true);
    try {
      await call(
        postV1ProjectsByProjectIdAdminAccessRequestsByIdApprove({
          path: { project_id: projectId, id: request.id! },
        }),
      );
      onRefresh();
      toast.success('Request approved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to approve request');
    } finally {
      setApproveBusy(false);
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={<Button variant="ghost" size="icon-sm" aria-label="Request actions" />}
        >
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => setDetailOpen(true)}>View details</DropdownMenuItem>
          {isPending && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={approve} disabled={approveBusy}>
                <CheckCircle className="size-4 text-emerald-600" />
                Approve
              </DropdownMenuItem>
              <DropdownMenuItem variant="destructive" onClick={() => setDenyOpen(true)}>
                <XCircle className="size-4" />
                Deny
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      {detailOpen && (
        <DetailDialog request={request} open={detailOpen} onOpenChange={setDetailOpen} />
      )}
      {denyOpen && (
        <DenyDialog
          projectId={projectId}
          request={request}
          open={denyOpen}
          onOpenChange={setDenyOpen}
          onDenied={onRefresh}
        />
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// Inline pending actions (shown directly in the row)
// ---------------------------------------------------------------------------

function InlinePendingActions({
  projectId,
  request,
  onRefresh,
}: {
  projectId: string;
  request: AccessRequest;
  onRefresh: () => void;
}) {
  const [denyOpen, setDenyOpen] = useState(false);
  const [approveBusy, setApproveBusy] = useState(false);

  async function approve() {
    setApproveBusy(true);
    try {
      await call(
        postV1ProjectsByProjectIdAdminAccessRequestsByIdApprove({
          path: { project_id: projectId, id: request.id! },
        }),
      );
      onRefresh();
      toast.success('Request approved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to approve request');
    } finally {
      setApproveBusy(false);
    }
  }

  return (
    <>
      <div className="flex items-center gap-1.5">
        <Button size="xs" onClick={approve} disabled={approveBusy} title="Approve">
          {approveBusy ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <CheckCircle className="size-3" />
          )}
          Approve
        </Button>
        <Button size="xs" variant="outline" onClick={() => setDenyOpen(true)} title="Deny">
          <XCircle className="size-3" />
          Deny
        </Button>
      </div>

      {denyOpen && (
        <DenyDialog
          projectId={projectId}
          request={request}
          open={denyOpen}
          onOpenChange={setDenyOpen}
          onDenied={onRefresh}
        />
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// Column definitions
// ---------------------------------------------------------------------------

function buildColumns(projectId: string, reload: () => void): ColumnDef<AccessRequest, unknown>[] {
  return [
    {
      id: 'subject',
      header: 'Subject',
      accessorFn: (row) => `${row.email ?? ''} ${row.id ?? ''}`,
      cell: ({ row }) => {
        const r = row.original;
        return (
          <div className="flex flex-col gap-0.5">
            <span className="font-medium">{r.email ?? '—'}</span>
            {r.id && <CopyButton value={r.id} />}
          </div>
        );
      },
    },
    {
      id: 'status',
      header: 'Status',
      accessorFn: (row) => row.status ?? 'pending',
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      id: 'requested_at',
      header: 'Requested at',
      accessorFn: (row) => row.created_at ?? '',
      cell: ({ row }) => (
        <span className="text-muted-foreground">{formatDate(row.original.created_at)}</span>
      ),
    },
    {
      id: 'reason',
      header: 'Reason',
      accessorFn: (row) => row.reason ?? '',
      cell: ({ row }) =>
        row.original.reason ? (
          <span className="line-clamp-2 max-w-xs text-sm text-muted-foreground">
            {row.original.reason}
          </span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      id: 'inline_actions',
      header: '',
      enableSorting: false,
      cell: ({ row }) => {
        if (row.original.status !== 'pending') return null;
        return (
          <InlinePendingActions projectId={projectId} request={row.original} onRefresh={reload} />
        );
      },
    },
    {
      id: 'actions',
      header: '',
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex justify-end">
          <AccessRequestActions
            projectId={projectId}
            request={row.original}
            onRefresh={reload}
          />
        </div>
      ),
    },
  ];
}

// ---------------------------------------------------------------------------
// Status filter tabs
// ---------------------------------------------------------------------------

const STATUS_FILTERS = [
  { label: 'All', value: '' },
  { label: 'Pending', value: 'pending' },
  { label: 'Approved', value: 'approved' },
  { label: 'Denied', value: 'denied' },
] as const;

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function AccessRequestsPage() {
  const { projectId } = useParams();
  const [statusFilter, setStatusFilter] = useState<string>('');

  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminAccessRequests({
          path: { project_id: projectId! },
          query: statusFilter ? { status: statusFilter } : undefined,
        }),
      ),
    [projectId, statusFilter],
  );

  const requests = data?.data ?? [];
  const columns = buildColumns(projectId!, reload);

  return (
    <div>
      <PageHeader
        title="Access Requests"
        description="Review and act on user access requests for this project."
      />

      {/* Status filter */}
      <div className="mb-4 flex items-center gap-1 rounded-lg border bg-muted/40 p-1 w-fit">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => setStatusFilter(f.value)}
            className={[
              'rounded-md px-3 py-1 text-sm font-medium transition-colors',
              statusFilter === f.value
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            ].join(' ')}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && requests.length === 0 && (
        <EmptyState
          title="No access requests"
          description={
            statusFilter
              ? `No ${statusFilter} requests found.`
              : 'No access requests have been submitted yet.'
          }
        />
      )}
      {!loading && !error && requests.length > 0 && (
        <DataTable
          columns={columns}
          data={requests}
          searchPlaceholder="Search by email or ID…"
          emptyMessage="No requests match your search."
        />
      )}
    </div>
  );
}
