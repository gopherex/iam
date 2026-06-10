/**
 * /flow — Server-side resumable auth flow page.
 *
 * Public route (no RequireAuth). Renders one step-component per FlowState.step,
 * driving the FlowController via the useFlow hook.
 *
 * Deep-link support:
 *   ?flow=<token>  — resume a specific flow token (cross-device link)
 *   ?kind=<kind>   — pre-select the flow kind on the credential form (default: signin)
 *
 * Cookie resume: on mount, if no ?flow param is present, useFlow calls
 * resume() which first tries GET /v1/auth/flows/current (HttpOnly cookie path)
 * then falls back to the token in localStorage.
 */

import { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  KeyRound,
  Loader2,
  Mail,
  Phone,
  ShieldCheck,
  AlertCircle,
  Clock,
  RefreshCw,
  CheckCircle,
} from 'lucide-react';
import type { FlowState, FlowKind } from '@gopherex/iam-sdk';

// ConsentRef is the shape of a consent doc reference in FlowState.consents_required
// (the generated element type is loose; this keeps the consent UI typed).
type ConsentRef = { key?: string | null; version?: string | null; url?: string | null };
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
import { useFlow } from '@/lib/use-flow';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function ErrorBanner({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
      <AlertCircle className="mt-0.5 size-4 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function ResendCountdown({
  resendAt,
  onResend,
  loading,
}: {
  resendAt: string | undefined;
  onResend: () => void;
  loading: boolean;
}) {
  const [secondsLeft, setSecondsLeft] = useState(0);

  useEffect(() => {
    if (!resendAt) {
      setSecondsLeft(0);
      return;
    }
    const target = new Date(resendAt).getTime();
    const tick = () => {
      const diff = Math.max(0, Math.ceil((target - Date.now()) / 1000));
      setSecondsLeft(diff);
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [resendAt]);

  if (secondsLeft > 0) {
    return (
      <p className="flex items-center gap-1 text-sm text-muted-foreground">
        <Clock className="size-3.5" />
        Resend available in {secondsLeft}s
      </p>
    );
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="gap-1.5"
      disabled={loading}
      onClick={onResend}
    >
      <RefreshCw className="size-3.5" />
      Resend code
    </Button>
  );
}

// ---------------------------------------------------------------------------
// Step components
// ---------------------------------------------------------------------------

/** collect_credentials: a form to start a new flow */
function CollectCredentials({
  defaultKind,
  loading,
  onStart,
  error,
}: {
  defaultKind: FlowKind;
  loading: boolean;
  onStart: (params: { kind: FlowKind; email?: string; password?: string; name?: string }) => void;
  error: string | null;
}) {
  const [kind, setKind] = useState<FlowKind>(defaultKind);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');

  const isSignup = kind === 'signup';
  const isRecovery = kind === 'recovery';

  function submit(e: React.FormEvent) {
    e.preventDefault();
    onStart({ kind, email: email || undefined, password: password || undefined, name: name || undefined });
  }

  const kindLabel: Record<FlowKind, string> = {
    signup: 'Create account',
    signin: 'Sign in',
    recovery: 'Reset password',
    email_change: 'Change email',
  };

  return (
    <form onSubmit={submit} className="space-y-4">
      {/* Kind selector */}
      <div className="flex rounded-lg border border-border overflow-hidden text-sm">
        {(['signin', 'signup', 'recovery'] as FlowKind[]).map((k) => (
          <button
            key={k}
            type="button"
            onClick={() => setKind(k)}
            className={`flex-1 py-1.5 transition-colors ${
              kind === k
                ? 'bg-primary text-primary-foreground font-medium'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted'
            }`}
          >
            {k === 'signin' ? 'Sign in' : k === 'signup' ? 'Sign up' : 'Recovery'}
          </button>
        ))}
      </div>

      {isSignup && (
        <div className="space-y-2">
          <Label htmlFor="flow-name">Full name</Label>
          <Input
            id="flow-name"
            type="text"
            autoComplete="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Your name"
          />
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="flow-email">Email</Label>
        <div className="relative">
          <Mail className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="flow-email"
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className="pl-8"
          />
        </div>
      </div>

      {!isRecovery && (
        <div className="space-y-2">
          <Label htmlFor="flow-password">Password</Label>
          <div className="relative">
            <KeyRound className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              id="flow-password"
              type="password"
              autoComplete={isSignup ? 'new-password' : 'current-password'}
              required={!isRecovery}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="pl-8"
            />
          </div>
        </div>
      )}

      {error && <ErrorBanner message={error} />}

      <Button type="submit" className="w-full" disabled={loading || !email}>
        {loading && <Loader2 className="size-4 animate-spin" />}
        {kindLabel[kind]}
      </Button>
    </form>
  );
}

/** verify_email / verify_phone: code entry + resend */
function VerifyCode({
  state,
  channel,
  loading,
  onSubmit,
  onResend,
  error,
}: {
  state: FlowState;
  channel: 'email' | 'phone';
  loading: boolean;
  onSubmit: (code: string) => void;
  onResend: () => void;
  error: string | null;
}) {
  const [code, setCode] = useState('');
  const contact =
    channel === 'email' ? state.contact?.email_masked : state.contact?.phone_masked;
  const attemptsLeft = state.challenge?.attempts_left;
  const resendAt = state.challenge?.resend_at as string | undefined;

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (code.trim()) onSubmit(code.trim());
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {channel === 'email' ? (
          <>
            <Mail className="mr-1 inline-block size-3.5" />
            Code sent to <strong>{contact ?? 'your email'}</strong>
          </>
        ) : (
          <>
            <Phone className="mr-1 inline-block size-3.5" />
            Code sent to <strong>{contact ?? 'your phone'}</strong>
          </>
        )}
      </p>

      <div className="space-y-2">
        <Label htmlFor="flow-code">Verification code</Label>
        <Input
          id="flow-code"
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          autoFocus
          required
          maxLength={8}
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
          placeholder="123456"
          className="font-mono tracking-widest text-center text-lg"
        />
      </div>

      {attemptsLeft !== undefined && attemptsLeft <= 3 && (
        <p className="text-sm text-amber-600 dark:text-amber-400">
          {attemptsLeft} attempt{attemptsLeft !== 1 ? 's' : ''} remaining
        </p>
      )}

      {error && <ErrorBanner message={error} />}

      <Button type="submit" className="w-full" disabled={loading || code.length < 4}>
        {loading && <Loader2 className="size-4 animate-spin" />}
        Verify
      </Button>

      <div className="flex justify-center">
        <ResendCountdown resendAt={resendAt} onResend={onResend} loading={loading} />
      </div>
    </form>
  );
}

/** set_password: new password for recovery */
function SetPassword({
  loading,
  onSubmit,
  error,
}: {
  loading: boolean;
  onSubmit: (password: string) => void;
  error: string | null;
}) {
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const mismatch = confirm.length > 0 && password !== confirm;

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (password && password === confirm) onSubmit(password);
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="flow-new-pw">New password</Label>
        <div className="relative">
          <KeyRound className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="flow-new-pw"
            type="password"
            autoComplete="new-password"
            autoFocus
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            className="pl-8"
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="flow-confirm-pw">Confirm password</Label>
        <div className="relative">
          <KeyRound className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="flow-confirm-pw"
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder="••••••••"
            className="pl-8"
            aria-invalid={mismatch || undefined}
          />
        </div>
        {mismatch && <p className="text-sm text-destructive">Passwords do not match</p>}
      </div>

      {error && <ErrorBanner message={error} />}

      <Button
        type="submit"
        className="w-full"
        disabled={loading || !password || password !== confirm}
      >
        {loading && <Loader2 className="size-4 animate-spin" />}
        Set new password
      </Button>
    </form>
  );
}

/** mfa_required: TOTP/OTP code entry */
function MfaRequired({
  state,
  loading,
  onSubmit,
  onResend,
  error,
}: {
  state: FlowState;
  loading: boolean;
  onSubmit: (code: string) => void;
  onResend: () => void;
  error: string | null;
}) {
  const [code, setCode] = useState('');
  const resendAt = state.challenge?.resend_at as string | undefined;
  const hasChallengeChannel = !!state.challenge?.channel;

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (code.trim()) onSubmit(code.trim());
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Enter the code from your authenticator app or the one-time code sent to you.
      </p>

      <div className="space-y-2">
        <Label htmlFor="flow-mfa-code">Authentication code</Label>
        <Input
          id="flow-mfa-code"
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          autoFocus
          required
          maxLength={8}
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
          placeholder="123456"
          className="font-mono tracking-widest text-center text-lg"
        />
      </div>

      {error && <ErrorBanner message={error} />}

      <Button type="submit" className="w-full" disabled={loading || code.length < 4}>
        {loading && <Loader2 className="size-4 animate-spin" />}
        Verify
      </Button>

      {hasChallengeChannel && (
        <div className="flex justify-center">
          <ResendCountdown resendAt={resendAt} onResend={onResend} loading={loading} />
        </div>
      )}
    </form>
  );
}

/** accept_consents: show consent docs and accept */
function AcceptConsents({
  state,
  loading,
  onSubmit,
  error,
}: {
  state: FlowState;
  loading: boolean;
  onSubmit: (keys: string[]) => void;
  error: string | null;
}) {
  const docs = (state.consents_required ?? []) as ConsentRef[];
  const [accepted, setAccepted] = useState<Set<string>>(new Set());

  function toggle(key: string) {
    setAccepted((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  const allAccepted = docs.length > 0 && docs.every((d) => d.key && accepted.has(d.key));

  function submit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit([...accepted]);
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Please review and accept the following before continuing.
      </p>

      <div className="space-y-3">
        {docs.map((doc) => {
          const key = doc.key ?? '';
          return (
            <label key={key} className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                className="mt-0.5 size-4 rounded border-border accent-primary"
                checked={accepted.has(key)}
                onChange={() => toggle(key)}
              />
              <span className="text-sm">
                I accept the{' '}
                {doc.url ? (
                  <a
                    href={doc.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:text-foreground"
                  >
                    {key} {doc.version && `v${doc.version}`}
                  </a>
                ) : (
                  <strong>
                    {key} {doc.version && `v${doc.version}`}
                  </strong>
                )}
              </span>
            </label>
          );
        })}
      </div>

      {error && <ErrorBanner message={error} />}

      <Button type="submit" className="w-full" disabled={loading || !allAccepted}>
        {loading && <Loader2 className="size-4 animate-spin" />}
        Accept and continue
      </Button>
    </form>
  );
}

/** request_access: reason / message form */
function RequestAccess({
  loading,
  onSubmit,
  error,
}: {
  loading: boolean;
  onSubmit: (message: string) => void;
  error: string | null;
}) {
  const [message, setMessage] = useState('');

  function submit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit(message);
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-muted-foreground">
        This product requires approval. Describe why you need access.
      </p>

      <div className="space-y-2">
        <Label htmlFor="flow-reason">Reason (optional)</Label>
        <textarea
          id="flow-reason"
          rows={4}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="Briefly describe your use case…"
          className="w-full resize-none rounded-lg border border-border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        />
      </div>

      {error && <ErrorBanner message={error} />}

      <Button type="submit" className="w-full" disabled={loading}>
        {loading && <Loader2 className="size-4 animate-spin" />}
        Request access
      </Button>
    </form>
  );
}

/** awaiting_approval: informational notice */
function AwaitingApproval() {
  return (
    <div className="space-y-3 text-center">
      <p className="text-sm text-muted-foreground">
        Your request is pending review. You'll receive an email when it's approved.
      </p>
    </div>
  );
}

/** blocked: error notice */
function Blocked({ code }: { code: string | undefined }) {
  const messages: Record<string, string> = {
    closed: 'Registrations are currently closed.',
    invite_only: 'This product is invite-only.',
  };
  return (
    <div className="space-y-3 text-center">
      <ErrorBanner message={messages[code ?? ''] ?? 'Access is not available at this time.'} />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function FlowPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const flowToken = searchParams.get('flow') ?? undefined;
  const kindParam = (searchParams.get('kind') ?? 'signin') as FlowKind;

  const { state, error, loading, start, submit, resend, abandon } = useFlow({ flowToken });

  // Redirect to / on completion (session is already set by the controller).
  const completedRef = useRef(false);
  useEffect(() => {
    if (state?.status === 'completed' && !completedRef.current) {
      completedRef.current = true;
      navigate('/', { replace: true });
    }
  }, [state?.status, navigate]);

  // Reset completed flag if a new flow starts.
  useEffect(() => {
    if (state?.status === 'pending') completedRef.current = false;
  }, [state?.status]);

  // ---- title / icon per step ----
  const stepMeta: Record<
    string,
    { title: string; description: string }
  > = {
    collect_credentials: { title: 'Welcome', description: 'Sign in or create your account.' },
    verify_email: { title: 'Check your email', description: 'Enter the code we sent you.' },
    verify_phone: { title: 'Verify your phone', description: 'Enter the code we sent you.' },
    set_password: { title: 'Set a new password', description: 'Choose a strong password.' },
    mfa_required: { title: 'Two-factor authentication', description: 'Enter your authentication code.' },
    accept_consents: { title: 'Review & accept', description: 'Read and accept the required documents.' },
    request_access: { title: 'Request access', description: 'Submit a request to join.' },
    awaiting_approval: { title: 'Pending approval', description: 'Your request is under review.' },
    completed: { title: 'All done!', description: 'Redirecting…' },
    blocked: { title: 'Access denied', description: '' },
  };

  const step = state?.step ?? 'collect_credentials';
  const meta = stepMeta[step] ?? { title: 'Auth flow', description: '' };

  // ---- inline error from FlowState.error (doesn't reset the flow) ----
  const inlineError = state?.error?.message ?? error?.message ?? null;

  // ---- Submit helpers that resolve the action name from next_actions ----
  function firstAction(...preferred: string[]): string {
    for (const a of preferred) {
      if (state?.next_actions?.includes(a)) return a;
    }
    return state?.next_actions?.[0] ?? 'submit';
  }

  return (
    <div className="relative flex min-h-svh items-center justify-center bg-muted/30 p-4">
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>

      <Card className="w-full max-w-sm">
        <CardHeader className="items-center text-center">
          <div className="mb-2 flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            {step === 'completed' ? (
              <CheckCircle className="size-6" />
            ) : (
              <ShieldCheck className="size-6" />
            )}
          </div>
          <CardTitle className="font-heading text-xl">{meta.title}</CardTitle>
          {meta.description && <CardDescription>{meta.description}</CardDescription>}
        </CardHeader>

        <CardContent>
          {/* Loading spinner while resuming */}
          {loading && !state && (
            <div className="flex justify-center py-6">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          )}

          {/* expired / aborted: show restart prompt */}
          {(state?.status === 'expired' || state?.status === 'aborted') && (
            <div className="space-y-4">
              <ErrorBanner
                message={
                  state.status === 'expired'
                    ? 'Your session expired. Please start again.'
                    : 'The flow was cancelled. Please start again.'
                }
              />
              <Button
                className="w-full"
                onClick={() => start({ kind: kindParam })}
              >
                Start over
              </Button>
            </div>
          )}

          {/* Active steps */}
          {state?.status === 'pending' && (
            <>
              {step === 'collect_credentials' && (
                <CollectCredentials
                  defaultKind={kindParam}
                  loading={loading}
                  error={inlineError}
                  onStart={({ kind, email, password, name }) =>
                    start({ kind, email, password, name })
                  }
                />
              )}

              {step === 'verify_email' && (
                <VerifyCode
                  state={state}
                  channel="email"
                  loading={loading}
                  error={inlineError}
                  onSubmit={(code) =>
                    submit(firstAction('verify_email', 'verify', 'submit'), { code })
                  }
                  onResend={resend}
                />
              )}

              {step === 'verify_phone' && (
                <VerifyCode
                  state={state}
                  channel="phone"
                  loading={loading}
                  error={inlineError}
                  onSubmit={(code) =>
                    submit(firstAction('verify_phone', 'verify', 'submit'), { code })
                  }
                  onResend={resend}
                />
              )}

              {step === 'set_password' && (
                <SetPassword
                  loading={loading}
                  error={inlineError}
                  onSubmit={(password) =>
                    submit(firstAction('set_password', 'submit'), { password })
                  }
                />
              )}

              {step === 'mfa_required' && (
                <MfaRequired
                  state={state}
                  loading={loading}
                  error={inlineError}
                  onSubmit={(code) => {
                    const action = firstAction('verify_mfa', 'mfa', 'submit');
                    submit(action, { code });
                  }}
                  onResend={resend}
                />
              )}

              {step === 'accept_consents' && (
                <AcceptConsents
                  state={state}
                  loading={loading}
                  error={inlineError}
                  onSubmit={(keys) =>
                    submit(firstAction('accept_consents', 'accept', 'submit'), {
                      consents: keys.map((key) => {
                        const doc = (state.consents_required as ConsentRef[] | undefined)?.find((d) => d.key === key);
                        return { key, version: doc?.version ?? '' };
                      }),
                    })
                  }
                />
              )}

              {step === 'request_access' && (
                <RequestAccess
                  loading={loading}
                  error={inlineError}
                  onSubmit={(message) =>
                    submit(firstAction('request_access', 'submit'), { message })
                  }
                />
              )}

              {step === 'awaiting_approval' && <AwaitingApproval />}

              {step === 'blocked' && <Blocked code={state.error?.code} />}

              {/* Abandon link — always available while pending */}
              {!['awaiting_approval', 'blocked', 'collect_credentials'].includes(step) && (
                <div className="mt-4 flex justify-center">
                  <button
                    type="button"
                    onClick={abandon}
                    className="text-xs text-muted-foreground underline-offset-2 hover:underline"
                  >
                    Cancel and start over
                  </button>
                </div>
              )}
            </>
          )}

          {/* Completed step */}
          {step === 'completed' && (
            <div className="flex justify-center py-4">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
