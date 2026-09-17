import { useEffect, useState } from 'react';
import { useStore } from '@nanostores/react';
import { getAccountDeletionPolicy, putAccountDeletionPolicy, getAdminAccountDeletion, cancelAdminAccountDeletion } from '@gopherex/iam-sdk';
import { toast } from 'sonner';
import { $environment } from '@/stores/auth';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { ErrorState, LoadingState } from '@/components/states';

export function AccountDeletionPolicyCard({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(() => call(getAccountDeletionPolicy({ path: { project_id: projectId }, headers: { 'X-Environment': env } })), [projectId, env]);
  const [days, setDays] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => { setDays(data ? String(data.grace_days) : ''); }, [data, projectId, env]);
  async function save(event: React.FormEvent) {
    event.preventDefault();
    const value = Number(days);
    if (!Number.isInteger(value) || value < 1 || value > 365) return;
    setBusy(true);
    try { await call(putAccountDeletionPolicy({ path: { project_id: projectId }, headers: { 'X-Environment': env }, body: { grace_days: value } })); toast.success('Deletion policy saved'); reload(); }
    catch (error) { toast.error(error instanceof Error ? error.message : 'Failed to save deletion policy'); }
    finally { setBusy(false); }
  }
  return <Card><CardHeader><CardTitle>Account deletion</CardTitle><CardDescription>Users retain full access during the cancellation period. Signing in does not cancel deletion. Existing deadlines do not change when this setting is updated.</CardDescription></CardHeader><CardContent>
    {loading ? <LoadingState /> : error ? <ErrorState error={error} onRetry={reload} /> : <form onSubmit={save} className="space-y-4"><Label htmlFor="deletion-grace-days">Cancellation period (days)</Label><Input id="deletion-grace-days" type="number" min={1} max={365} step={1} required value={days} onChange={event => setDays(event.target.value)} /><Button type="submit" disabled={busy || !data}>Save deletion policy</Button></form>}
  </CardContent></Card>;
}

export function UserDeletionDetails({ projectId, userId, onChanged }: { projectId: string; userId: string; onChanged?: () => void }) {
  const env = useStore($environment);
  const { data, loading, error, reload } = useApi(() => call(getAdminAccountDeletion({ path: { project_id: projectId, user_id: userId }, headers: { 'X-Environment': env } })), [projectId, userId, env]);
  const [now, setNow] = useState(Date.now());
  useEffect(() => { const timer = setInterval(() => setNow(Date.now()), 60000); return () => clearInterval(timer); }, []);
  if (loading) return <LoadingState />;
  if (error) return <ErrorState error={error} onRetry={reload} />;
  if (!data) return null;
  const deletion = data.deletion;
  const remaining = deletion.delete_at ? Math.max(0, Math.ceil((Date.parse(deletion.delete_at) - now) / 86400000)) : 0;
  return <div className="space-y-5 p-1"><div className="space-y-2"><h3 className="font-medium">Account deletion</h3><p>Status: {deletion.status === 'pending' ? remaining > 0 ? 'Scheduled' : 'Awaiting cleanup — access ended' : deletion.status}</p>{deletion.requested_at && <p>Requested: {new Date(deletion.requested_at).toLocaleString()}</p>}{deletion.delete_at && <p>Deletion date: {new Date(deletion.delete_at).toLocaleString()}</p>}{deletion.status === 'pending' && remaining > 0 && <p>{remaining} days remaining. The account retains full access until the deadline.</p>}{deletion.cancelled_at && <p>Cancelled: {new Date(deletion.cancelled_at).toLocaleString()}</p>}</div><div className="space-y-2"><h3 className="font-medium">Deletion history</h3>{data.events.length === 0 ? <p className="text-muted-foreground">No deletion requests.</p> : data.events.map((event, index) => <div key={`${event.request_id}:${event.type}:${index}`} className="border-b py-2 text-sm"><p>{event.type}</p>{event.actor_id && <p>By: {event.actor_id}</p>}{event.reason && <p>{event.reason}</p>}<time className="text-muted-foreground">{new Date(event.at).toLocaleString()}</time><p className="font-mono text-xs text-muted-foreground">{event.request_id}</p></div>)}</div><div className="flex gap-2"><Button variant="outline" onClick={reload}>Refresh</Button>{deletion.status === 'pending' && remaining > 0 && deletion.request_id && <CancelDeletionDialog key={`${projectId}:${userId}:${env}:${deletion.request_id}`} projectId={projectId} userId={userId} env={env} requestId={deletion.request_id} onCancelled={() => { reload(); onChanged?.(); }} />}</div></div>;
}

function CancelDeletionDialog({ projectId, userId, env, requestId, onCancelled }: {
  projectId: string; userId: string; env: string; requestId: string; onCancelled: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!reason.trim()) return;
    setBusy(true); setError('');
    try {
      await call(cancelAdminAccountDeletion({ path: { project_id: projectId, user_id: userId }, headers: { 'X-Environment': env }, body: { request_id: requestId, reason: reason.trim() } }));
      toast.success('Account deletion cancelled'); setOpen(false); onCancelled();
    } catch (error) { setError(error instanceof Error ? error.message : 'Cancellation failed'); }
    finally { setBusy(false); }
  }
  return <Dialog open={open} onOpenChange={value => { if (!busy) { setOpen(value); setError(''); if (value) setReason(''); } }}>
    <DialogTrigger render={<Button />}>Cancel scheduled deletion</DialogTrigger>
    <DialogContent><DialogHeader><DialogTitle>Cancel account deletion</DialogTitle><DialogDescription>The account will no longer be scheduled for deletion. Existing account restrictions remain in effect. Your identity and reason will be recorded, and IAM will queue a notification for the user.</DialogDescription></DialogHeader>
      <form onSubmit={submit} className="space-y-4">
        <Label htmlFor="deletion-cancel-reason">Reason for cancellation</Label><Input id="deletion-cancel-reason" value={reason} onChange={event => setReason(event.target.value)} required maxLength={1024} />
        {error && <p role="alert" className="text-destructive">{error}</p>}
        <Button type="submit" disabled={busy || !reason.trim()}>Confirm cancellation</Button>
      </form>
    </DialogContent>
  </Dialog>;
}
