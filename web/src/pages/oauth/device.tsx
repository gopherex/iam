/**
 * /oauth/device — the screen a person opens on their phone or laptop to approve
 * a device that cannot show a browser (RFC 8628).
 *
 * Two states: enter the user_code the device is displaying, then confirm or
 * refuse the application behind it. Approving requires a signed-in browser
 * session, so an unauthenticated visitor is sent through the same sign-in flow
 * the interaction page uses, in cookie mode.
 */

import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, CheckCircle, Loader2, MonitorSmartphone } from 'lucide-react';
import { createIamOidc } from '@gopherex/iam-sdk';

import { FlowSteps } from '@/components/flow-steps';
import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { interpolate, resolveLocale, scopeLabel, translator, type T } from '@/lib/i18n';
import { providerErrorKey } from '@/lib/provider-api';
import { IamAuthError } from '@gopherex/iam-sdk';
import { useFlow } from '@/lib/use-flow';

interface DeviceContext {
  client?: Record<string, unknown>;
  scopes?: string[];
  expires_at?: string;
}

type Screen = 'code' | 'confirm' | 'signin' | 'approved' | 'denied' | 'error';

export function DevicePage() {
  const [searchParams] = useSearchParams();

  // The device may print a complete verification URI carrying the code.
  const [code, setCode] = useState(searchParams.get('user_code') ?? '');
  const projectId = searchParams.get('client_id') ?? searchParams.get('project_id') ?? '';

  const [ctx, setCtx] = useState<DeviceContext | null>(null);
  const [screen, setScreen] = useState<Screen>('code');
  const [errorKey, setErrorKey] = useState('provider.error.generic');
  const [busy, setBusy] = useState(false);

  // The device screen is reached without an interaction, so there is no project
  // locale to read until the code resolves; fall back to the browser's.
  const t: T = useMemo(() => translator(resolveLocale(undefined, undefined)), []);

  const flow = useFlow();
  const appName = String((ctx?.client as { name?: string })?.name ?? '');

  // The SDK namespace owns the cookie-mode CSRF handshake.
  const oidc = useMemo(
    () => (projectId ? createIamOidc({ baseUrl: '', clientId: projectId }) : null),
    [projectId],
  );

  const fail = useCallback((err: unknown) => {
    const apiErr = err as IamAuthError;
    setErrorKey(apiErr?.code === 'not_found' ? 'device.expired' : providerErrorKey(apiErr?.code));
    setScreen('error');
  }, []);

  async function lookup(e: React.FormEvent) {
    e.preventDefault();
    if (!code.trim() || !projectId) {
      setErrorKey('provider.error.generic');
      setScreen('error');
      return;
    }
    setBusy(true);
    try {
      const { data, error } = await oidc!.getDevice(code.trim());
      if (error) throw error;
      setCtx((data ?? {}) as DeviceContext);
      setScreen('confirm');
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  async function decide(approve: boolean) {
    if (!oidc) return;
    setBusy(true);
    try {
      const { error } = approve
        ? await oidc.approveDevice(code.trim())
        : await oidc.denyDevice(code.trim());
      if (error) throw error;
      setScreen(approve ? 'approved' : 'denied');
    } catch (err) {
      // Approving needs a signed-in browser; send an anonymous visitor through
      // the same sign-in flow rather than a dead end.
      if ((err as IamAuthError)?.status === 401) {
        setScreen('signin');
        return;
      }
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  const heading = (() => {
    switch (screen) {
      case 'confirm':
        return {
          title: t('device.confirmTitle', 'Confirm this device'),
          description: interpolate(
            t('device.confirmDescription', '{app} is asking to use your account on a device.'),
            { app: appName },
          ),
        };
      case 'approved':
        return { title: t('device.confirmTitle', 'Confirm this device'), description: '' };
      case 'denied':
      case 'error':
        return { title: t('flow.title.blocked', 'Access denied'), description: '' };
      default:
        return {
          title: t('device.title', 'Connect a device'),
          description: t('device.description', 'Enter the code shown on your device.'),
        };
    }
  })();

  return (
    <div className="relative flex min-h-svh items-center justify-center bg-muted/30 p-4">
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>

      <Card className="w-full max-w-sm">
        <CardHeader className="items-center text-center">
          <div className="mb-2 flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            {screen === 'approved' ? (
              <CheckCircle className="size-6" />
            ) : screen === 'denied' || screen === 'error' ? (
              <AlertCircle className="size-6" />
            ) : (
              <MonitorSmartphone className="size-6" />
            )}
          </div>
          <CardTitle className="font-heading text-xl">{heading.title}</CardTitle>
          {heading.description && <CardDescription>{heading.description}</CardDescription>}
        </CardHeader>

        <CardContent>
          {screen === 'code' && (
            <form onSubmit={lookup} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="user-code">{t('device.codeLabel', 'Code')}</Label>
                <Input
                  id="user-code"
                  autoFocus
                  value={code}
                  onChange={(e) => setCode(e.target.value.toUpperCase())}
                  placeholder="ABCD-EFGH"
                  className="text-center font-mono tracking-widest"
                />
              </div>
              <Button type="submit" className="w-full" disabled={busy || !code.trim()}>
                {busy && <Loader2 className="size-4 animate-spin" />}
                {t('device.submit', 'Continue')}
              </Button>
            </form>
          )}

          {screen === 'signin' && (
            <FlowSteps flow={flow} kind="signin" t={t} startOptions={{ cookieMode: true }} />
          )}

          {screen === 'confirm' && (
            <div className="space-y-4">
              <ul className="space-y-2 text-sm">
                {(ctx?.scopes ?? []).map((scope) => (
                  <li key={scope} className="flex items-start gap-2">
                    <CheckCircle className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                    <span>{scopeLabel(t, scope)}</span>
                  </li>
                ))}
              </ul>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  className="flex-1"
                  disabled={busy}
                  onClick={() => void decide(false)}
                >
                  {t('device.deny', 'Refuse')}
                </Button>
                <Button className="flex-1" disabled={busy} onClick={() => void decide(true)}>
                  {busy && <Loader2 className="size-4 animate-spin" />}
                  {t('device.approve', 'Approve')}
                </Button>
              </div>
            </div>
          )}

          {screen === 'approved' && (
            <p className="text-center text-sm text-muted-foreground">
              {t('device.approved', 'The device is connected. You can go back to it.')}
            </p>
          )}

          {screen === 'denied' && (
            <p className="text-center text-sm text-muted-foreground">
              {t('device.denied', 'The request was refused.')}
            </p>
          )}

          {screen === 'error' && (
            <p className="text-center text-sm text-muted-foreground">
              {t(errorKey, 'Something went wrong.')}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
