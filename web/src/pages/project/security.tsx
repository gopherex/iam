import { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useStore } from '@nanostores/react';
import {
  getSecurityPolicy, putSecurityPolicy, adminListSecurityIncidents,
  listSecurityCases, decideSecurityCase, listSecurityDeliveries, retrySecurityDelivery,
  type SecurityPolicy, type SecurityCase, type SecurityIncident, type SecurityDelivery,
} from '@gopherex/iam-sdk';
import { call } from '@/lib/sdk';
import { $environment } from '@/stores/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';

export function AccountSecurityPage() {
  const { projectId = '' } = useParams();
  const environment = useStore($environment);
  const [policy, setPolicy] = useState<SecurityPolicy | null>(null);
  const [cases, setCases] = useState<SecurityCase[]>([]);
  const [incidents, setIncidents] = useState<SecurityIncident[]>([]);
  const [deliveries, setDeliveries] = useState<SecurityDelivery[]>([]);
  const [selected, setSelected] = useState<SecurityCase | null>(null);
  const [message, setMessage] = useState('');
  const [evidence, setEvidence] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState<'policy' | 'incidents' | 'support' | 'deliveries'>('incidents');
  const [cursors, setCursors] = useState<Record<string, string | undefined>>({});
  const scopeRef = useRef('');
  const scopeKey = `${projectId}:${environment}`; scopeRef.current = scopeKey;
  const options = { path: { project_id: projectId }, headers: { 'X-Environment': environment } };
  async function load(reset = true) {
    if (!reset) {
      if (tab === 'incidents') { const page = await call(adminListSecurityIncidents({ ...options, query: { cursor: cursors.incidents } })); if (scopeRef.current !== scopeKey) return; setIncidents(old => [...old, ...page.data]); setCursors(old => ({ ...old, incidents: page.next_cursor })); }
      if (tab === 'support') { const page = await call(listSecurityCases({ ...options, query: { cursor: cursors.support } })); if (scopeRef.current !== scopeKey) return; setCases(old => [...old, ...page.data]); setCursors(old => ({ ...old, support: page.next_cursor })); }
      if (tab === 'deliveries') { const page = await call(listSecurityDeliveries({ ...options, query: { cursor: cursors.deliveries } })); if (scopeRef.current !== scopeKey) return; setDeliveries(old => [...old, ...page.data]); setCursors(old => ({ ...old, deliveries: page.next_cursor })); }
      return;
    }
    const [p, i, c, d] = await Promise.all([
      call(getSecurityPolicy(options)),
      call(adminListSecurityIncidents({ ...options, query: { cursor: reset ? undefined : cursors.incidents } })),
      call(listSecurityCases({ ...options, query: { cursor: reset ? undefined : cursors.support } })),
      call(listSecurityDeliveries({ ...options, query: { cursor: reset ? undefined : cursors.deliveries } })),
    ]);
    if (scopeRef.current !== scopeKey) return;
    setPolicy(p); setIncidents(old => reset ? i.data : [...old, ...i.data]); setCases(old => reset ? c.data : [...old, ...c.data]); setDeliveries(old => reset ? d.data : [...old, ...d.data]);
    setCursors({ incidents: i.next_cursor, support: c.next_cursor, deliveries: d.next_cursor });
  }
  async function run(action: () => Promise<unknown>) {
    setBusy(true); setError('');
    try { await action(); } catch (e) { setError(e instanceof Error ? e.message : 'Request failed'); } finally { setBusy(false); }
  }
  useEffect(() => { setSelected(null); void run(() => load()); }, [projectId, environment]);
  async function decide(action: string) {
    if (!selected) return;
    const c = await call(decideSecurityCase({ path: { project_id: projectId, case_id: selected.id }, headers: options.headers, body: { action, message, evidence } }));
    if (scopeRef.current !== scopeKey) return;
    setSelected(c); setMessage(''); setEvidence(''); await load();
  }
  return <div className="space-y-6">
    <div className="flex items-start justify-between"><div><h1 className="text-2xl font-semibold">Account security</h1><p className="mt-1 text-muted-foreground">Incidents, protection and account recovery in {environment}.</p></div><Button variant="outline" disabled={busy} onClick={() => void run(() => load())}>Refresh</Button></div>
    <nav className="flex gap-2" aria-label="Security sections">{(['incidents', 'support', 'deliveries', 'policy'] as const).map(t => <Button key={t} variant={tab === t ? 'default' : 'outline'} onClick={() => setTab(t)}>{t[0].toUpperCase() + t.slice(1)}</Button>)}</nav>
    {error && <div role="alert" className="rounded border border-destructive p-3 text-destructive">{error}</div>}
    {tab === 'policy' && policy && <Card><CardHeader><CardTitle>Security policy</CardTitle><CardDescription>Observe records events. Enforce also applies configured checks and sends enabled notifications.</CardDescription></CardHeader><CardContent><form className="space-y-4" onSubmit={e => { e.preventDefault(); void run(async () => { setPolicy(await call(putSecurityPolicy({ ...options, body: policy }))); }); }}>
      <label className="block space-y-1"><span>Mode</span><select className="block w-full rounded border bg-background p-2" value={policy.mode} onChange={e => setPolicy({ ...policy, mode: e.target.value })}><option value="disabled">Disabled</option><option value="observe">Observe</option><option value="enforce">Enforce</option></select></label>
      <label className="block space-y-1"><span>Application continuation URL</span><Input type="url" value={policy.continue_url} onChange={e => setPolicy({ ...policy, continue_url: e.target.value })} placeholder={`https://app.example.com/security/${projectId}/${environment}`} /></label>
      <p className="text-sm text-muted-foreground">The application renders the recovery flow. Only registered HTTPS addresses are allowed; localhost HTTP is supported for development.</p>
      <label className="flex gap-2"><input type="checkbox" checked={policy.notify} onChange={e => setPolicy({ ...policy, notify: e.target.checked })} />Send security notifications</label>
      <label className="flex gap-2"><input type="checkbox" checked={policy.require_new_device_proof} onChange={e => setPolicy({ ...policy, require_new_device_proof: e.target.checked })} />Require additional proof on unfamiliar devices</label>
      <div className="grid gap-4 sm:grid-cols-2">{([
        ['flow_ttl_seconds', 'Flow lifetime (seconds)'], ['trust_ttl_seconds', 'Device trust lifetime (seconds)'],
        ['retention_days', 'Retention (days)'], ['failure_threshold', 'Failure threshold'],
        ['failure_window_seconds', 'Failure window (seconds)'], ['notification_cooldown_seconds', 'Repeated notification interval (seconds)'],
      ] as const).map(([key, label]) => <label key={key} className="space-y-1"><span>{label}</span><Input type="number" min="1" value={policy[key]} onChange={e => setPolicy({ ...policy, [key]: Number(e.target.value) })} /></label>)}</div>
      <label className="block space-y-1"><span>Client continuation URLs (JSON)</span><textarea className="w-full rounded border bg-background p-2 font-mono" key={JSON.stringify(policy.client_urls)} defaultValue={JSON.stringify(policy.client_urls, null, 2)} onBlur={e => { try { const urls = JSON.parse(e.target.value); if (!urls || Array.isArray(urls) || typeof urls !== 'object' || Object.values(urls).some(url => typeof url !== 'string')) throw new Error('Use an object mapping client IDs to URLs'); setPolicy({ ...policy, client_urls: urls }); setError(''); } catch (e) { setError(e instanceof Error ? e.message : 'Invalid JSON'); } }} /></label>
      <Button type="submit" disabled={busy || !!error}>Save policy</Button>
    </form></CardContent></Card>}
    {tab === 'incidents' && <div className="space-y-3">{incidents.length === 0 && <p className="text-muted-foreground">No incidents in this environment.</p>}{incidents.map(i => <Card key={i.id}><CardHeader><CardTitle className="text-base">{i.reasons.join(', ')}</CardTitle><CardDescription>{i.account_id} · {i.status} · {i.severity} · {new Date(i.created_at).toLocaleString()}</CardDescription></CardHeader><CardContent className="space-y-2">{i.events.map(e => <p key={e.id} className="text-sm">{e.type} · {e.outcome} · {e.ip || 'Unknown address'} · {e.user_agent || 'Unknown device'}</p>)}{i.resolution && <p>Resolution: {i.resolution}</p>}<p className="text-sm text-muted-foreground">Revoked sessions: {i.revoked_session_ids.length}</p></CardContent></Card>)}</div>}
    {tab === 'support' && <div className="grid gap-4 lg:grid-cols-2"><div className="space-y-2">{cases.length === 0 && <p>No recovery requests.</p>}{cases.map(c => <button key={c.id} type="button" className={`w-full rounded border p-4 text-left ${selected?.id === c.id ? 'border-primary' : ''}`} onClick={() => { setSelected(c); setMessage(''); setEvidence(''); }}><p>{c.contact_masked}</p><p className="text-sm text-muted-foreground">{c.status} · {new Date(c.created_at).toLocaleString()}</p><p className="break-all text-xs">{c.id}</p></button>)}</div>{selected && <Card><CardHeader><CardTitle>Recovery request</CardTitle><CardDescription>Account: {selected.account_id || 'Not identified'}</CardDescription></CardHeader><CardContent className="space-y-4">{selected.messages.map((m, i) => <div className="rounded bg-muted p-3" key={i}><p className="text-xs text-muted-foreground">{m.author} · {new Date(m.at).toLocaleString()}</p><p>{m.message}</p></div>)}{['pending', 'needs_information'].includes(selected.status) && <><label className="block space-y-1"><span>Message to the user</span><textarea className="w-full rounded border bg-background p-2" value={message} onChange={e => setMessage(e.target.value)} maxLength={4096} /></label><label className="block space-y-1"><span>Verification evidence / internal reference</span><textarea className="w-full rounded border bg-background p-2" value={evidence} onChange={e => setEvidence(e.target.value)} maxLength={4096} /></label><p className="text-sm text-muted-foreground">Approval requires independent ownership verification and security:recovery permission. It grants a restricted recovery flow, not a login session.</p><div className="flex flex-wrap gap-2"><Button disabled={busy || !message.trim() || !evidence.trim()} onClick={() => void run(() => decide('approve'))}>Approve recovery</Button><Button variant="outline" disabled={busy || !message.trim() || !evidence.trim()} onClick={() => void run(() => decide('request_information'))}>Request information</Button><Button variant="destructive" disabled={busy || !message.trim() || !evidence.trim()} onClick={() => void run(() => decide('reject'))}>Reject</Button></div></>}</CardContent></Card>}</div>}
    {tab === 'deliveries' && <div className="space-y-2">{deliveries.length === 0 && <p>No security deliveries.</p>}{deliveries.map(d => <div key={d.id} className="flex items-center justify-between rounded border p-4"><div><p>{d.recipient_masked} · {d.channel} · {d.status}</p><p className="text-sm text-muted-foreground">Attempts: {d.attempts} · {d.last_error || 'No error'}</p></div><Button variant="outline" disabled={busy || !['failed', 'blocked'].includes(d.status)} onClick={() => void run(async () => { await call(retrySecurityDelivery({ path: { project_id: projectId, delivery_id: d.id }, headers: options.headers })); await load(); })}>Retry</Button></div>)}<p className="text-sm text-muted-foreground">Accepted means the provider accepted the message; it does not confirm receipt by the user.</p></div>}
    {cursors[tab] && <Button variant="outline" disabled={busy} onClick={() => void run(() => load(false))}>Load more</Button>}
  </div>;
}
