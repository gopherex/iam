import type { Environment } from '@gopherex/iam-sdk';
import {
  deleteMgmtV1ProjectsByProjectIdEnvironmentsByEnv,
  getMgmtV1ProjectsByProjectIdEnvironments,
  postMgmtV1ProjectsByProjectIdEnvironments,
} from '@gopherex/iam-sdk';
import { Loader2, Plus, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';
import { PageHeader } from '@/components/page-header';
import { ErrorState, LoadingState } from '@/components/states';
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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { loadEnvironments } from '@/lib/environments';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

export function EnvironmentsPage() {
  const { projectId } = useParams();
  const pid = projectId!;
  const { data, loading, error, reload } = useApi(
    () => call(getMgmtV1ProjectsByProjectIdEnvironments({ path: { project_id: pid } })),
    [pid],
  );
  const envs: Environment[] = data?.data ?? [];

  const [deleteTarget, setDeleteTarget] = useState<Environment | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);

  async function refresh() {
    reload();
    await loadEnvironments(pid); // keep the header switcher in sync
  }

  async function doDelete() {
    if (!deleteTarget?.name) return;
    setDeleteBusy(true);
    try {
      await call(
        deleteMgmtV1ProjectsByProjectIdEnvironmentsByEnv({
          path: { project_id: pid, env: deleteTarget.name },
        }),
      );
      toast.success(`Environment "${deleteTarget.name}" deleted`);
      setDeleteTarget(null);
      await refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete environment');
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Environments"
        description="Isolated test / live / staging data spaces. Clients pick one via X-Environment; each has its own users, sessions, and signing keys."
        actions={<CreateEnvironmentDialog projectId={pid} existing={envs} onCreated={refresh} />}
      />

      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && (
        <div className="space-y-2">
          {envs.map((e) => (
            <div
              key={e.name}
              className="flex items-center justify-between rounded-lg border bg-card px-4 py-3"
            >
              <div className="min-w-0">
                <div className="font-medium">{e.name}</div>
                <div className="truncate text-xs text-muted-foreground">{e.issuer ?? '—'}</div>
              </div>
              {e.name !== 'live' ? (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => setDeleteTarget(e)}
                  title="Delete environment"
                >
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              ) : (
                <span className="text-xs text-muted-foreground">default</span>
              )}
            </div>
          ))}
        </div>
      )}

      <Dialog open={!!deleteTarget} onOpenChange={(o) => !o && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Delete environment</DialogTitle>
            <DialogDescription>
              Permanently delete <strong>{deleteTarget?.name}</strong> and its isolated data. This
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={doDelete} disabled={deleteBusy}>
              {deleteBusy && <Loader2 className="size-4 animate-spin" />}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function CreateEnvironmentDialog({
  projectId,
  existing,
  onCreated,
}: {
  projectId: string;
  existing: Environment[];
  onCreated: () => void | Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [cloneFrom, setCloneFrom] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postMgmtV1ProjectsByProjectIdEnvironments({
          path: { project_id: projectId },
          body: { name: name.trim(), clone_config_from: cloneFrom || undefined },
        }),
      );
      toast.success(`Environment "${name.trim()}" created`);
      setOpen(false);
      setName('');
      setCloneFrom('');
      await onCreated();
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : 'Failed to create environment');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { setOpen(o); if (!o) setErr(null); }}>
      <DialogTrigger render={<Button><Plus className="size-4" />New environment</Button>} />
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>New environment</DialogTitle>
          <DialogDescription>
            Create an isolated environment (e.g. dev, staging). It gets its own data and signing
            keys; clients reach it with the X-Environment header.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={create} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="env-name">Name</Label>
            <Input
              id="env-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="dev"
              required
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="env-clone">Clone config from (optional)</Label>
            <select
              id="env-clone"
              value={cloneFrom}
              onChange={(e) => setCloneFrom(e.target.value)}
              className="w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              <option value="">— none —</option>
              {existing.map((e) => (
                <option key={e.name} value={e.name ?? ''}>
                  {e.name}
                </option>
              ))}
            </select>
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
