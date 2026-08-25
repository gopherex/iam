/**
 * /oauth/interaction/:id — the hosted screen an OIDC authorization request lands
 * on.
 *
 * /oauth2/authorize redirects the browser here with nothing but the interaction
 * id, so this page fetches the context (which application is asking, for what,
 * in whose project, in which language) and then drives one of three states:
 *
 *   1. already signed in — an existing browser session is offered for reuse.
 *      This is the state that makes it single sign-on: without it a person
 *      re-types their password once per relying party.
 *   2. sign in — the same resumable flow engine and the same step components as
 *      the console's /flow page, started in cookie mode so completing it leaves
 *      an HttpOnly session cookie behind.
 *   3. consent — the application, the scopes it asks for in plain words, and the
 *      two decisions.
 *
 * Every state-changing call is cookie-mode with a CSRF token; the page never
 * holds a token itself.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { AlertCircle, CheckCircle, Loader2, ShieldCheck, UserRound } from 'lucide-react';
import { createIamOidc, getV1AuthSession, getV1OauthInteractionByInteractionId } from '@gopherex/iam-sdk';

import { FlowSteps, flowStepMeta } from '@/components/flow-steps';
import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { interpolate, resolveLocale, scopeLabel, translator, type T } from '@/lib/i18n';
import { providerErrorKey } from '@/lib/provider-api';
import { IamAuthError } from '@gopherex/iam-sdk';
import { call } from '@/lib/sdk';
import { useFlow } from '@/lib/use-flow';

interface InteractionContext {
  stage?: 'login' | 'consent';
  client?: { id?: string; name?: string; type?: string };
  project_id?: string;
  environment?: string;
  default_locale?: string;
  supported_locales?: string[];
  requested_scopes?: string[];
}

/** Screen is what the page is currently showing. */
type Screen = 'loading' | 'signin' | 'continue' | 'consent' | 'redirecting' | 'error';

export function InteractionPage() {
  const { id = '' } = useParams<{ id: string }>();

  const [ctx, setCtx] = useState<InteractionContext | null>(null);
  const [screen, setScreen] = useState<Screen>('loading');
  const [errorKey, setErrorKey] = useState<string>('provider.error.generic');
  const [sessionEmail, setSessionEmail] = useState<string | null>(null);
  const [remember, setRemember] = useState(true);
  const [busy, setBusy] = useState(false);

  const t: T = useMemo(
    () => translator(resolveLocale(ctx?.default_locale, ctx?.supported_locales)),
    [ctx?.default_locale, ctx?.supported_locales],
  );

  const appName = ctx?.client?.name || ctx?.client?.id || '';
  const flow = useFlow();

  // The SDK namespace owns the cookie-mode CSRF handshake; it can only be built
  // once the interaction has told us which project we are in.
  const oidc = useMemo(
    () =>
      ctx?.project_id
        ? createIamOidc({ baseUrl: '', clientId: ctx.project_id, environment: ctx.environment })
        : null,
    [ctx?.project_id, ctx?.environment],
  );

  const fail = useCallback((err: unknown) => {
    setErrorKey(providerErrorKey((err as IamAuthError)?.code));
    setScreen('error');
  }, []);

  // ---- load the interaction, then decide which screen to show ----
  const load = useCallback(async () => {
    if (!id) return;
    try {
      const got = (await call(
        getV1OauthInteractionByInteractionId({ path: { interaction_id: id } }),
      )) as InteractionContext;
      setCtx(got);

      if (got.stage === 'consent') {
        setScreen('consent');
        return;
      }

      // Nobody has claimed the interaction yet. If this browser already holds a
      // valid IAM session, offer it instead of asking for a password again.
      try {
        const sess = (await call(getV1AuthSession())) as { user?: { email?: string } } | undefined;
        setSessionEmail(sess?.user?.email ?? null);
        setScreen('continue');
      } catch {
        setScreen('signin');
      }
    } catch (err) {
      fail(err);
    }
  }, [id, fail]);

  useEffect(() => {
    void load();
  }, [load]);

  // ---- claim the interaction with the current browser session ----
  const claim = useCallback(async () => {
    if (!oidc) return;
    setBusy(true);
    try {
      const { error } = await oidc.loginInteraction(id);
      if (error) throw error;
      await load();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }, [oidc, id, load, fail]);

  // Once the embedded sign-in flow completes it has left a session cookie
  // behind, so the interaction can be claimed exactly like an existing session.
  useEffect(() => {
    if (screen === 'signin' && flow.state?.status === 'completed') {
      void claim();
    }
  }, [screen, flow.state?.status, claim]);

  function leave(url: string) {
    setScreen('redirecting');
    window.location.assign(url);
  }

  async function decide(allow: boolean) {
    if (!oidc) return;
    setBusy(true);
    try {
      const { data, error } = allow
        ? await oidc.consentInteraction(id, {
            grantedScopes: ctx?.requested_scopes ?? [],
            remember,
          })
        : await oidc.rejectInteraction(id, { error: 'access_denied' });

      if (error) throw error;

      const to = (data as { redirect_to?: string } | undefined)?.redirect_to;
      if (to) {
        leave(to);
        return;
      }
      setErrorKey(allow ? 'provider.error.generic' : 'provider.error.denied');
      setScreen('error');
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  // ---- chrome ----
  const heading = (() => {
    switch (screen) {
      case 'consent':
        return {
          title: t('provider.consentTitle', 'Allow access'),
          description: interpolate(
            t('provider.consentDescription', '{app} wants to access your account.'),
            { app: appName },
          ),
        };
      case 'continue':
        return {
          title: appName
            ? interpolate(t('provider.signInTo', 'Sign in to {app}'), { app: appName })
            : t('provider.signInTo', 'Sign in'),
          description: t('provider.continueDescription', 'You are already signed in to this account.'),
        };
      case 'signin': {
        const meta = flowStepMeta(t, flow.state?.step ?? 'collect_credentials');
        return {
          title: appName
            ? interpolate(t('provider.signInTo', 'Sign in to {app}'), { app: appName })
            : meta.title,
          description: meta.description,
        };
      }
      case 'error':
        return { title: t('flow.title.blocked', 'Access denied'), description: '' };
      default:
        return { title: t('provider.redirecting', 'Redirecting…'), description: '' };
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
            {screen === 'error' ? (
              <AlertCircle className="size-6" />
            ) : screen === 'continue' ? (
              <UserRound className="size-6" />
            ) : screen === 'redirecting' ? (
              <CheckCircle className="size-6" />
            ) : (
              <ShieldCheck className="size-6" />
            )}
          </div>
          <CardTitle className="font-heading text-xl">{heading.title}</CardTitle>
          {heading.description && <CardDescription>{heading.description}</CardDescription>}
        </CardHeader>

        <CardContent>
          {(screen === 'loading' || screen === 'redirecting') && (
            <div className="flex justify-center py-6">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          )}

          {screen === 'error' && (
            <p className="text-center text-sm text-muted-foreground">
              {t(errorKey, 'Something went wrong.')}
            </p>
          )}

          {screen === 'continue' && (
            <div className="space-y-4">
              {sessionEmail && (
                <p className="text-center text-sm font-medium">{sessionEmail}</p>
              )}
              <Button className="w-full" disabled={busy} onClick={() => void claim()}>
                {busy && <Loader2 className="size-4 animate-spin" />}
                {sessionEmail
                  ? interpolate(t('provider.continueAs', 'Continue as {email}'), {
                      email: sessionEmail,
                    })
                  : t('provider.continue', 'Continue')}
              </Button>
              <button
                type="button"
                className="w-full text-xs text-muted-foreground underline-offset-2 hover:underline"
                onClick={() => setScreen('signin')}
              >
                {t('provider.useAnotherAccount', 'Use another account')}
              </button>
            </div>
          )}

          {screen === 'signin' && (
            <FlowSteps flow={flow} kind="signin" t={t} startOptions={{ cookieMode: true }} />
          )}

          {screen === 'consent' && (
            <div className="space-y-4">
              <ul className="space-y-2 text-sm">
                {(ctx?.requested_scopes ?? []).map((scope) => (
                  <li key={scope} className="flex items-start gap-2">
                    <CheckCircle className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                    <span>{scopeLabel(t, scope)}</span>
                  </li>
                ))}
              </ul>

              <div className="flex items-center gap-2">
                <Checkbox
                  id="remember-consent"
                  checked={remember}
                  onCheckedChange={(v) => setRemember(v === true)}
                />
                <Label htmlFor="remember-consent" className="text-sm font-normal">
                  {t('provider.remember', 'Remember this decision')}
                </Label>
              </div>

              <div className="flex gap-2">
                <Button
                  variant="outline"
                  className="flex-1"
                  disabled={busy}
                  onClick={() => void decide(false)}
                >
                  {t('provider.deny', 'Deny')}
                </Button>
                <Button className="flex-1" disabled={busy} onClick={() => void decide(true)}>
                  {busy && <Loader2 className="size-4 animate-spin" />}
                  {t('provider.allow', 'Allow')}
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
