import type { Invite, InviteCreated } from '@gopherex/iam-sdk';
import {
  getV1ProjectsByProjectIdAdminInvites,
  postV1ProjectsByProjectIdAdminInvites,
  postV1ProjectsByProjectIdAdminInvitesByInviteIdRevoke,
} from '@gopherex/iam-sdk';
import type { ColumnDef } from '@tanstack/react-table';
import { Check, Copy, Loader2, MoreHorizontal, Plus } from 'lucide-react';
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
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function fmtDate(ts?: string | null): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function statusVariant(status: Invite['status']): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'accepted':
      return 'default';
    case 'revoked':
      return 'destructive';
    case 'pending':
    default:
      return 'secondary';
  }
}

// ---------------------------------------------------------------------------
// Copyable token field
// ---------------------------------------------------------------------------

function CopyableToken({ token }: { token: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      toast.success('Invite token copied to clipboard');
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Failed to copy — select and copy the token manually');
    }
  }

  return (
    <div className="flex gap-2">
      <Input
        readOnly
        value={token}
        className="flex-1 font-mono text-xs"
        onFocus={(e) => e.currentTarget.select()}
      />
      <Button type="button" variant="outline" size="icon-sm" onClick={() => void copy()} title="Copy">
        {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Create invite dialog
// ---------------------------------------------------------------------------

function CreateInviteDialog({
  projectId,
  trigger,
  onSaved,
}: {
  projectId: string;
  trigger: React.ReactElement;
  onSaved: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [redirectTo, setRedirectTo] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [created, setCreated] = useState<InviteCreated | null>(null);

  function reset() {
    setEmail('');
    setExpiresAt('');
    setRedirectTo('');
    setErr(null);
    setCreated(null);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const res = await call(
        postV1ProjectsByProjectIdAdminInvites({
          path: { project_id: projectId },
          body: {
            email: email.trim() || undefined,
            // datetime-local yields "YYYY-MM-DDTHH:mm" (local); convert to RFC3339.
            expires_at: expiresAt ? new Date(expiresAt).toISOString() : undefined,
            redirect_to: redirectTo.trim() || undefined,
          },
        }),
      );
      setCreated(res);
      toast.success(
        email.trim()
          ? `Invitation created — email sent to ${email.trim()}`
          : 'Invitation created',
      );
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create invitation');
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
        {created ? (
          <>
            <DialogHeader>
              <DialogTitle>Invitation created</DialogTitle>
              <DialogDescription>
                Copy the invite token now — it is shown <strong>once</strong> and cannot be retrieved
                again.
                {created.email
                  ? ' An invitation email was also sent to the invitee.'
                  : ''}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2">
              <Label>Invite token</Label>
              <CopyableToken token={created.invite_token} />
            </div>
            <DialogFooter>
              <DialogClose render={<Button />}>Done</DialogClose>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>New invitation</DialogTitle>
              <DialogDescription>
                Create an invitation token. Provide an email to bind the invite to a specific address
                and send a notification email, or leave it blank for an open invite usable by anyone.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={submit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="inv-email">Email (optional)</Label>
                <Input
                  id="inv-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="invitee@example.com"
                  autoFocus
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="inv-expires">Expires at (optional)</Label>
                <Input
                  id="inv-expires"
                  type="datetime-local"
                  value={expiresAt}
                  onChange={(e) => setExpiresAt(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="inv-redirect">Redirect URL (optional)</Label>
                <Input
                  id="inv-redirect"
                  type="url"
                  value={redirectTo}
                  onChange={(e) => setRedirectTo(e.target.value)}
                  placeholder="https://app.example.com/accept"
                />
                <p className="text-xs text-muted-foreground">
                  Base URL for the invitation email link. Its origin must match the project&rsquo;s
                  configured app base URL.
                </p>
              </div>
              {err && <p className="text-sm text-destructive">{err}</p>}
              <DialogFooter>
                <Button type="submit" disabled={busy}>
                  {busy && <Loader2 className="size-4 animate-spin" />}
                  Create invitation
                </Button>
              </DialogFooter>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Confirm revoke dialog
// ---------------------------------------------------------------------------

function ConfirmRevokeDialog({
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
          <DialogTitle>Revoke invitation</DialogTitle>
          <DialogDescription>
            Revoke the invitation for <strong>{label}</strong>? The token will no longer be usable.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button variant="destructive" onClick={onConfirm} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Revoke
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Page root
// ---------------------------------------------------------------------------

export function InvitesPage() {
  const { projectId } = useParams();
  const { data, loading, error, reload } = useApi(
    () => call(getV1ProjectsByProjectIdAdminInvites({ path: { project_id: projectId! } })),
    [projectId],
  );
  const invites = data?.invites ?? [];

  const [revokeTarget, setRevokeTarget] = useState<Invite | null>(null);
  const [revokeBusy, setRevokeBusy] = useState(false);

  async function doRevoke() {
    if (!revokeTarget?.id) return;
    setRevokeBusy(true);
    try {
      await call(
        postV1ProjectsByProjectIdAdminInvitesByInviteIdRevoke({
          path: { project_id: projectId!, invite_id: revokeTarget.id },
        }),
      );
      toast.success('Invitation revoked');
      setRevokeTarget(null);
      reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to revoke invitation');
    } finally {
      setRevokeBusy(false);
    }
  }

  const columns: ColumnDef<Invite>[] = [
    {
      id: 'email',
      header: 'Email',
      accessorFn: (i) => i.email ?? '',
      cell: ({ row }) =>
        row.original.email ? (
          <span className="font-medium">{row.original.email}</span>
        ) : (
          <span className="text-muted-foreground">— (any)</span>
        ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'status',
      cell: ({ row }) => (
        <Badge variant={statusVariant(row.original.status)} className="capitalize">
          {row.original.status}
        </Badge>
      ),
    },
    {
      id: 'created_at',
      header: 'Created',
      accessorKey: 'created_at',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{fmtDate(row.original.created_at)}</span>
      ),
    },
    {
      id: 'expires_at',
      header: 'Expires',
      accessorKey: 'expires_at',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{fmtDate(row.original.expires_at)}</span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const inv = row.original;
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />}>
                <MoreHorizontal className="size-4" />
                <span className="sr-only">Actions</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  variant="destructive"
                  disabled={inv.status !== 'pending'}
                  onClick={() => setRevokeTarget(inv)}
                >
                  Revoke
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
        title="Invitations"
        description="Issue and manage invitation tokens used to onboard users into this project."
        actions={
          <CreateInviteDialog
            projectId={projectId!}
            trigger={
              <Button>
                <Plus className="size-4" />
                New invitation
              </Button>
            }
            onSaved={reload}
          />
        }
      />

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && invites.length === 0 && (
        <EmptyState
          title="No invitations"
          description="Create an invitation to onboard a user. You can bind it to an email or leave it open."
          action={
            <CreateInviteDialog
              projectId={projectId!}
              trigger={
                <Button>
                  <Plus className="size-4" />
                  New invitation
                </Button>
              }
              onSaved={reload}
            />
          }
        />
      )}
      {!loading && !error && invites.length > 0 && (
        <DataTable
          columns={columns}
          data={invites}
          searchPlaceholder="Search invitations…"
          emptyMessage="No invitations match your search."
        />
      )}

      <ConfirmRevokeDialog
        open={Boolean(revokeTarget)}
        onOpenChange={(o) => {
          if (!o) setRevokeTarget(null);
        }}
        label={revokeTarget?.email ?? 'this invitation'}
        onConfirm={() => void doRevoke()}
        busy={revokeBusy}
      />
    </div>
  );
}
