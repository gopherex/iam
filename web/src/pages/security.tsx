import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { createIamClient, type SecurityFlowState, type SecurityIncident, type SecurityDevice, type SecurityFlowResult } from '@gopherex/iam-sdk';
import { AccountDeletionCard } from '@/components/account-deletion';
import { useFlow } from '@/lib/use-flow';
import { FlowSteps } from '@/components/flow-steps';
import { translator } from '@/lib/i18n';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';

/** Runnable reference application. All decisions and transitions belong to IAM. */
export function SecurityPage() {
  const { projectId = '', environment = 'live' } = useParams();
  const iam = useMemo(() => createIamClient({ baseUrl: window.location.origin, clientId: projectId, environment, storageKey: `iam.example:${projectId}:${environment}`, autoRefresh: false, multiTab: false }), [projectId, environment]);
  const login = useFlow({ controller: iam.flow });
  const [selectedSession, setSelectedSession] = useState('');
  const [clock, setClock] = useState(Date.now());
  const [state, setState] = useState<SecurityFlowState | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [contact, setContact] = useState('');
  const [message, setMessage] = useState('');
  const [all, setAll] = useState(false);
  const [factor, setFactor] = useState('');
  const [incidents, setIncidents] = useState<SecurityIncident[]>([]);
  const [devices, setDevices] = useState<SecurityDevice[]>([]);
  const [incidentCursor, setIncidentCursor] = useState('');
  const [deviceCursor, setDeviceCursor] = useState('');
  const [deviceNames, setDeviceNames] = useState<Record<string, string>>({});
  const [signedIn, setSignedIn] = useState(false);
  const [continuation] = useState(() => window.location.href.includes('#iam_security=') ? window.location.href : '');
  useEffect(() => {
    const off = iam.security.flow.onChange((value, failure) => { setState(value); setError(failure?.message ?? ''); });
    const auth = iam.auth.onAuthStateChange((_event, session) => setSignedIn(!!session));
    void iam.security.flow.resume();
    return () => { off(); auth.unsubscribe(); };
  }, [iam]);
  useEffect(() => {
    const token = login.error?.details?.security_flow_token;
    if (typeof token === 'string') void iam.security.flow.resumeByToken(token);
  }, [login.error, iam]);
  useEffect(() => { const tick = setInterval(() => setClock(Date.now()), 1000); return () => clearInterval(tick); }, []);
  async function run(action: () => Promise<unknown>) {
    setBusy(true); setError('');
    try { const result = await action() as { error?: Error | null }; if (result?.error) setError(result.error.message); }
    catch (e) { setError(e instanceof Error ? e.message : 'Не удалось выполнить запрос'); }
    finally { setBusy(false); }
  }
  async function refreshActivity() {
    const [events, devices] = await Promise.all([iam.security.listIncidents(), iam.security.listDevices()]);
    if (events.error) throw events.error;
    if (devices.error) throw devices.error;
    setIncidents(events.data?.data ?? []); setDevices(devices.data?.data ?? []);
    setIncidentCursor(events.data?.next_cursor ?? ""); setDeviceCursor(devices.data?.next_cursor ?? "");
  }
  async function loadMore(kind: 'incidents' | 'devices') {
    if (kind === 'incidents') {
      const result = await iam.security.listIncidents({ cursor: incidentCursor });
      if (result.error) throw result.error;
      setIncidents(items => [...items, ...(result.data?.data ?? [])]);
      setIncidentCursor(result.data?.next_cursor ?? '');
    } else {
      const result = await iam.security.listDevices({ cursor: deviceCursor });
      if (result.error) throw result.error;
      setDevices(items => [...items, ...(result.data?.data ?? [])]);
      setDeviceCursor(result.data?.next_cursor ?? '');
    }
  }
  async function updateDevice(id: string, action: 'rename' | 'untrust' | 'revoke') {
    const result = await iam.security.updateDevice(id, { action, name: deviceNames[id] });
    if (result.error) throw result.error;
    if (result.data) setDevices(items => items.map(device => device.id === id ? result.data! : device));
  }
  useEffect(() => {
    if (state?.step === 'review') { setAll(state.all_sessions); setSelectedSession(''); }
  }, [state?.step, state?.incident?.id, state?.case?.id]);
  const pending = state?.status === 'pending';
  const can = (action: string) => pending && state.next_actions.includes(action);
  const step = state?.step;
  const perform = (f: () => Promise<SecurityFlowResult>) => void run(f);
  return <main className="mx-auto min-h-screen max-w-3xl space-y-6 px-5 py-10">
    <div><h1 className="text-2xl font-semibold">Безопасность аккаунта</h1><p className="mt-2 text-muted-foreground">Проверьте активность и восстановите доступ, если что-то пошло не так.</p></div>
    {error && <div role="alert" className="rounded border border-destructive p-3 text-destructive">{error}</div>}
    {continuation && !state && <Card><CardHeader><CardTitle>Проверить уведомление</CardTitle><CardDescription>Открытие уведомления не завершает сессии и не выполняет вход.</CardDescription></CardHeader><CardContent><Button disabled={busy} onClick={() => perform(() => iam.security.flow.continueFromURL(continuation))}>Продолжить</Button></CardContent></Card>}
    {!pending && <Card><CardHeader><CardTitle>{signedIn ? 'Ваш аккаунт' : 'Вход'}</CardTitle></CardHeader><CardContent className="space-y-3">
      {!signedIn ? <FlowSteps fixedKind flow={login} kind="signin" t={translator('ru')} /> : <div className="flex flex-wrap gap-2"><Button disabled={busy} onClick={() => void run(refreshActivity)}>Обновить активность</Button><Button variant="outline" disabled={busy} onClick={() => void run(() => iam.security.registerDevice(navigator.platform))}>Запомнить это устройство</Button><Button variant="outline" onClick={() => void run(() => iam.auth.signOut())}>Выйти</Button></div>}
    </CardContent></Card>}
    {state?.status === 'completed' && <div role="status" className="rounded border p-4">{state.outcome === 'deletion_authorized' ? 'Личность подтверждена. Завершите выбранное действие в разделе удаления аккаунта.' : state.outcome === 'activity_confirmed' ? 'Активность подтверждена.' : state.outcome === 'support_rejected' ? 'Поддержка отклонила обращение. Подробности указаны в сообщениях.' : 'Сценарий завершён. Для доступа к приложению войдите обычным способом.'}</div>}
    {pending && <Card><CardHeader><CardTitle>{can('authorize_deletion') ? 'Подтвердите действие с аккаунтом' : step === 'review' ? 'Это были вы?' : step === 'restore_access' ? 'Восстановите доступ' : step === 'awaiting_support' ? 'Обращение в поддержку' : 'Подтвердите запрос'}</CardTitle></CardHeader><CardContent className="space-y-4">
      {state.incident && <div className="space-y-2"><p>{state.incident.reasons.join(', ')}</p>{state.incident.events.map(e => <div key={e.id} className="text-sm"><time>{new Date(e.at).toLocaleString()}</time><p>{e.outcome === 'blocked' ? 'Попытка остановлена' : 'Операция выполнена'} · {e.user_agent || 'Устройство неизвестно'} · {e.ip || 'Адрес неизвестен'}</p></div>)}</div>}
      {(step === 'verify_contact' || step === 'verify_identity') && <form className="space-y-3" onSubmit={e => { e.preventDefault(); perform(() => iam.security.flow.submit('verify_code', { code, message })); setCode(''); }}>
        <p>Код отправлен на {state.contact_masked}. Осталось попыток: {state.attempts_left}.</p>
        <Input aria-label="Код подтверждения" autoComplete="one-time-code" value={code} onChange={e => setCode(e.target.value)} required />
        <div className="flex gap-2"><Button type="submit" disabled={busy}>Подтвердить</Button><Button type="button" variant="outline" disabled={busy || !!state.resend_at && Date.parse(state.resend_at) > clock} onClick={() => perform(() => iam.security.flow.resend())}>Отправить снова</Button></div>
      </form>}
      {step === 'verify_mfa' && <div className="space-y-3"><p>Подтвердите ранее настроенным фактором или кодом восстановления.</p><select aria-label="Фактор подтверждения" className="w-full rounded border bg-background p-2" value={factor} onChange={e => setFactor(e.target.value)}><option value="">Выберите фактор</option>{state.factors?.map(f => <option key={f.id} value={f.id}>{f.type} {f.hint}</option>)}</select><Button variant="outline" disabled={busy || !factor || !!state.resend_at && Date.parse(state.resend_at) > clock} onClick={() => perform(() => iam.security.flow.submit('select_factor', { factor_id: factor }))}>Подготовить проверку</Button>{state.contact_masked && <p>Код: {state.contact_masked}</p>}<Input aria-label="Код фактора" value={code} onChange={e => setCode(e.target.value)} /><div className="flex gap-2"><Button disabled={busy || !factor} onClick={() => perform(() => factor === 'passkey' ? iam.security.flow.verifyPasskey() : iam.security.flow.submit('verify_mfa', { factor_id: factor, code }))}>Подтвердить</Button><Button disabled={busy} variant="outline" onClick={() => perform(() => iam.security.flow.submit('verify_recovery_code', { code }))}>Код восстановления</Button></div></div>}
      {step === 'review' && !can('trust_device') && !can('authorize_deletion') && <div className="space-y-4">{!all && state.sessions && state.sessions.length > 0 && <label className="block space-y-2"><span>Сессия, которую нужно завершить</span><select aria-label="Подозрительная сессия" className="w-full rounded border bg-background p-2" value={selectedSession} onChange={e => setSelectedSession(e.target.value)}><option value="">Выберите сессию</option>{state.sessions.map(session => <option key={session.id} value={session.id}>{session.name || session.id}</option>)}</select></label>}<label className="flex items-start gap-3"><input type="checkbox" checked={all} onChange={e => setAll(e.target.checked)} /><span>Завершить все сессии и снять доверие со всех устройств</span></label><p className="text-sm text-muted-foreground">Без галочки будет завершена сессия, связанная с событием. Доверие к устройствам не включается при подтверждении активности.</p><div className="flex gap-2">{can('confirm_activity') && <Button variant="outline" disabled={busy} onClick={() => perform(() => iam.security.flow.confirmActivity())}>Это я</Button>}<Button disabled={busy} onClick={() => perform(() => iam.security.flow.reportActivity({ allSessions: all, sessionId: selectedSession }))}>Это не я</Button></div></div>}
      {state.deletion && <div role="status" className="space-y-3 rounded border p-3">
        <p>{state.deletion.status === 'cancelled' ? 'Удаление аккаунта отменено.' : `Удаление аккаунта назначено на ${state.deletion.delete_at ? new Date(state.deletion.delete_at).toLocaleString() : 'неизвестную дату'}. Восстановление доступа не останавливает таймер.`}</p>
        {can('cancel_deletion') && <Button variant="outline" disabled={busy} onClick={() => perform(() => iam.security.flow.cancelDeletion(state.deletion!.request_id!))}>Отменить удаление аккаунта</Button>}
      </div>}
      {can('authorize_deletion') && <Button disabled={busy} onClick={() => perform(() => iam.security.flow.submit('authorize_deletion'))}>Подтвердить действие</Button>}
      {can('trust_device') && <div className="space-y-3"><p>Добавить доверие к этому устройству?</p><Button disabled={busy} onClick={() => perform(() => iam.security.flow.submit('trust_device'))}>Доверять устройству</Button></div>}
      {step === 'restore_access' && can('set_password') && <form className="space-y-3" onSubmit={e => { e.preventDefault(); perform(() => iam.security.flow.setPassword(password)); setPassword(''); }}><p>Завершено сессий: {state.revoked_session_ids.length}. Выбранный масштаб защиты сохранён.</p><Input aria-label="Новый пароль" type="password" autoComplete="new-password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Новый пароль" required /><Button type="submit" disabled={busy}>Сохранить новый пароль</Button></form>}
      {can('begin_passkey') && <Button disabled={busy} onClick={() => perform(() => iam.security.flow.registerRecoveryPasskey())}>Создать новый passkey</Button>}
      {can('complete_recovery') && <Button disabled={busy} onClick={() => perform(() => iam.security.flow.submit('complete_recovery'))}>Завершить восстановление</Button>}
      {can('set_phone') && <div className="space-y-3"><Input aria-label="Телефон для входа" type="tel" value={contact} onChange={e => setContact(e.target.value)} placeholder="+79991234567" /><Button disabled={busy} onClick={() => perform(() => iam.security.flow.submit('set_phone', { contact }))}>Подтвердить телефон для входа</Button></div>}
      {can('verify_phone') && <div className="space-y-3"><p>Код отправлен на {state.contact_masked}.</p><Input aria-label="Код нового телефона" value={code} onChange={e => setCode(e.target.value)} /><Button disabled={busy} onClick={() => perform(() => iam.security.flow.submit('verify_phone', { code }))}>Сохранить телефон</Button></div>}
      {state.case && <div className="space-y-2"><p>Номер обращения: {state.case.id}</p><p>Статус: {state.case.status}</p>{state.case.messages.map((m, i) => <p key={i} className="rounded bg-muted p-3">{m.author === 'support' ? 'Поддержка' : 'Вы'}: {m.message}</p>)}<Button variant="outline" disabled={busy} onClick={() => perform(() => iam.security.flow.resume())}>Обновить статус</Button></div>}
      {can('support_message') && <form className="space-y-3" onSubmit={e => { e.preventDefault(); perform(() => iam.security.flow.submit('support_message', { message })); setMessage(''); }}><Input aria-label="Сообщение поддержке" value={message} onChange={e => setMessage(e.target.value)} required /><Button type="submit" disabled={busy}>Отправить сообщение</Button></form>}
      {can('abandon') && <Button variant="ghost" disabled={busy} onClick={() => perform(() => iam.security.flow.abandon())}>Закрыть сценарий</Button>}
    </CardContent></Card>}
    {(!pending || can('request_support')) && <Card><CardHeader><CardTitle>Не получается подтвердить личность?</CardTitle><CardDescription>Подтвердите доступный вам адрес для связи с поддержкой. Само обращение не блокирует аккаунт.</CardDescription></CardHeader><CardContent><form className="space-y-3" onSubmit={e => { e.preventDefault(); perform(() => pending ? iam.security.flow.requestSupport({ identifier: email, contact, message }) : iam.security.flow.recover({ identifier: email, contact })); }}><Input aria-label="Аккаунт для восстановления" value={email} onChange={e => setEmail(e.target.value)} placeholder="Email или телефон аккаунта" required /><Input aria-label="Email для связи" type="email" value={contact} onChange={e => setContact(e.target.value)} placeholder="Доступный email для связи" required /><Input aria-label="Описание проблемы" value={message} onChange={e => setMessage(e.target.value)} placeholder="Что произошло" /><Button type="submit" disabled={busy}>Обратиться за помощью</Button></form></CardContent></Card>}
    {signedIn && !pending && <AccountDeletionCard iam={iam} proof={state} />}
    {signedIn && !pending && <div className="space-y-4"><h2 className="text-lg font-semibold">Активность</h2>{incidents.length === 0 && <p className="text-muted-foreground">Нет событий для отображения.</p>}{incidents.map(i => <Card key={i.id}><CardContent className="flex items-center justify-between gap-3 pt-5"><div><p>{i.reasons.join(', ')}</p><p className="text-sm text-muted-foreground">{new Date(i.created_at).toLocaleString()} · {i.status}</p></div><Button disabled={busy || i.status === 'resolved'} onClick={() => perform(() => iam.security.flow.start({ incidentId: i.id }))}>Проверить</Button></CardContent></Card>)}{incidentCursor && <Button disabled={busy} variant="outline" onClick={() => void run(() => loadMore('incidents'))}>Ещё события</Button>}<h2 className="text-lg font-semibold">Устройства</h2>{devices.map(d => <div key={d.id} className="flex flex-wrap items-center justify-between gap-3 rounded border p-3"><div><Input aria-label="Название устройства" value={deviceNames[d.id] ?? d.name ?? ''} onChange={e => setDeviceNames(names => ({ ...names, [d.id]: e.target.value }))} /><Button variant="ghost" disabled={busy || deviceNames[d.id] === undefined} onClick={() => void run(() => updateDevice(d.id, 'rename'))}>Переименовать</Button><p className="text-sm text-muted-foreground">{d.revoked ? 'Доступ завершён' : d.trusted_until ? 'Доверенное' : 'Распознано'} · {new Date(d.last_seen_at).toLocaleString()}</p></div>{d.trusted_until && !d.revoked && <Button variant="outline" disabled={busy} onClick={() => void run(() => updateDevice(d.id, 'untrust'))}>Снять доверие</Button>}{d.current && !d.revoked && <Button variant="outline" disabled={busy} onClick={() => perform(() => iam.security.flow.start({ deviceId: d.id }))}>Настроить доверие</Button>}<Button variant="outline" disabled={busy || d.revoked} onClick={() => void run(() => updateDevice(d.id, 'revoke'))}>Завершить сессии</Button></div>)}{deviceCursor && <Button disabled={busy} variant="outline" onClick={() => void run(() => loadMore('devices'))}>Ещё устройства</Button>}</div>}
  </main>;
}
