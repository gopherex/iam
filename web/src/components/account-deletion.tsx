import { useEffect, useState } from 'react';
import { type createIamClient, type AccountDeletion, type SecurityFlowState } from '@gopherex/iam-sdk';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export function AccountDeletionCard({ iam, proof }: { iam: ReturnType<typeof createIamClient>; proof: SecurityFlowState | null }) {
  const [deletion, setDeletion] = useState<AccountDeletion | null>(null);
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    void iam.account.deletion.get().then(result => {
      if (active) { setDeletion(result.data); setError(result.error?.message ?? ''); }
    }).catch(error => { if (active) setError(String(error)); });
    return () => { active = false; };
  }, [iam]);
  const scheduled = deletion?.status === 'pending';
  async function submit(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setError('');
    try {
      const input = { password, proofToken: proof?.outcome === 'deletion_authorized' ? proof.flow_token : undefined };
      const result = await (scheduled ? iam.account.deletion.cancel(input) : iam.account.deletion.request(input));
      const token = result.error?.details?.security_flow_token;
      if (typeof token === 'string') {
        const next = await iam.security.flow.resumeByToken(token);
        if (next.error) setError(next.error.message);
      } else if (result.error) setError(result.error.message);
      else { setDeletion(result.data); setPassword(''); }
    } catch (error) { setError(error instanceof Error ? error.message : 'Не удалось выполнить запрос'); }
    finally { setBusy(false); }
  }
  return <Card><CardHeader><CardTitle>Удаление аккаунта</CardTitle><CardDescription>В течение срока ожидания можно пользоваться аккаунтом и отменить удаление. Вход в аккаунт сам по себе не отменяет заявку.</CardDescription></CardHeader><CardContent className="space-y-4">
    {error && <p role="alert" className="text-destructive">{error}</p>}
    {scheduled && <p role="status">Удаление назначено на {new Date(deletion.delete_at!).toLocaleString()}. После этой даты доступ закончится.</p>}
    {deletion?.status === 'cancelled' && <p role="status">Удаление отменено.</p>}
    {deletion && <form className="space-y-3" onSubmit={submit}>
      {!scheduled && <p>{deletion.status === 'none' ? `Аккаунт будет удалён через ${deletion.grace_days} дн.` : 'Срок новой заявки определяется текущими настройками проекта.'} Подтверждение запустит срок ожидания.</p>}
      <Label htmlFor="deletion-password">Текущий пароль</Label><Input id="deletion-password" type="password" autoComplete="current-password" value={password} onChange={event => setPassword(event.target.value)} />
      <p className="text-sm text-muted-foreground">Если у аккаунта нет пароля, оставьте поле пустым: IAM предложит доступный способ подтверждения.</p>
      {proof?.outcome === 'deletion_authorized' && <p>Проверка завершена. Повторно подтвердите выбранное действие.</p>}
      <Button type="submit" variant={scheduled ? 'outline' : 'destructive'} disabled={busy}>{scheduled ? 'Отменить удаление' : 'Запланировать удаление'}</Button>
    </form>}
  </CardContent></Card>;
}
