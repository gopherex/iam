import type {
  AuthConfig,
  ConsentConfig,
  ConsentDocument,
  MfaPolicy,
  PasswordPolicy,
  RateLimitRule,
  RateLimits,
  RegistrationConfig,
  SessionPolicy,
} from '@gopherex/iam-sdk';
import {
  getV1ProjectsByProjectIdAdminConfigAuth,
  getV1ProjectsByProjectIdAdminConfigMfaPolicy,
  getV1ProjectsByProjectIdAdminConfigPasswordPolicy,
  getV1ProjectsByProjectIdAdminConfigRateLimits,
  getV1ProjectsByProjectIdAdminConfigSessionPolicy,
  getV1ProjectsByProjectIdAdminConsents,
  patchV1ProjectsByProjectIdAdminConfigAuth,
  patchV1ProjectsByProjectIdAdminConfigMfaPolicy,
  patchV1ProjectsByProjectIdAdminConfigPasswordPolicy,
  patchV1ProjectsByProjectIdAdminConfigRateLimits,
  patchV1ProjectsByProjectIdAdminConfigSessionPolicy,
  putV1ProjectsByProjectIdAdminConsents,
} from '@gopherex/iam-sdk';
import { Loader2, Plus, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';
import { PageHeader } from '@/components/page-header';
import { ErrorState, LoadingState } from '@/components/states';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

// ---------------------------------------------------------------------------
// Shared constants
// ---------------------------------------------------------------------------

const DEFAULT_ENV = 'live';

const ENV_HEADER = (env: string) => ({ 'X-Environment': env } as const);

// ---------------------------------------------------------------------------
// Auth & Methods tab
// ---------------------------------------------------------------------------

const KNOWN_AUTH_METHODS = ['email', 'phone', 'username', 'oauth', 'magic_link', 'passkey'];
const REGISTRATION_MODES = ['open', 'invite_only', 'request_access', 'closed'] as const;

function AuthTab({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminConfigAuth({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
        }),
      ),
    [projectId, env],
  );

  const [form, setForm] = useState<AuthConfig | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  function toggleMethod(method: string) {
    setForm((prev) => {
      if (!prev) return prev;
      const methods = prev.methods ?? [];
      return {
        ...prev,
        methods: methods.includes(method) ? methods.filter((m) => m !== method) : [...methods, method],
      };
    });
  }

  function setRegistrationMode(mode: string) {
    setForm((prev) =>
      prev
        ? { ...prev, registration: { ...prev.registration, mode: mode as RegistrationConfig['mode'] } }
        : prev,
    );
  }

  function setPasswordStrategy(strategy: string) {
    setForm((prev) =>
      prev
        ? {
            ...prev,
            registration: {
              ...prev.registration,
              password_strategy: strategy as RegistrationConfig['password_strategy'],
            },
          }
        : prev,
    );
  }

  async function save() {
    if (!form) return;
    setBusy(true);
    try {
      const updated = await call(
        patchV1ProjectsByProjectIdAdminConfigAuth({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
          body: form,
        }),
      );
      setForm(updated);
      toast.success('Auth config saved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to save auth config');
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <LoadingState />;
  if (error) return <ErrorState error={error} onRetry={reload} />;
  if (!form) return null;

  const enabledMethods = form.methods ?? [];

  return (
    <div className="space-y-4">
      {/* Auth Methods */}
      <Card>
        <CardHeader>
          <CardTitle>Authentication methods</CardTitle>
          <CardDescription>Enable the sign-in channels available to users in this environment.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {KNOWN_AUTH_METHODS.map((method) => (
            <div key={method} className="flex items-center justify-between">
              <Label htmlFor={`method-${method}`} className="capitalize">
                {method.replace('_', ' ')}
              </Label>
              <Switch
                id={`method-${method}`}
                checked={enabledMethods.includes(method)}
                onCheckedChange={() => toggleMethod(method)}
              />
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Application */}
      <Card>
        <CardHeader>
          <CardTitle>Application</CardTitle>
          <CardDescription>
            Public base URL of this project&rsquo;s hosted auth UI. Used to build the cross-device
            &ldquo;continue on another device&rdquo; deep-link (<code>&lt;base&gt;/continue?flow=…</code>).
            Leave empty to disable that email.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-1.5">
            <Label htmlFor="app-base-url">App base URL</Label>
            <Input
              id="app-base-url"
              type="url"
              placeholder="https://auth.example.com"
              value={form.app_base_url ?? ''}
              onChange={(e) =>
                setForm((p) => (p ? { ...p, app_base_url: e.target.value || undefined } : p))
              }
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={save} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Save
          </Button>
        </CardFooter>
      </Card>

      {/* Registration */}
      <Card>
        <CardHeader>
          <CardTitle>Registration</CardTitle>
          <CardDescription>Control how new users can sign up for this project.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <Label>Registration mode</Label>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {REGISTRATION_MODES.map((mode) => (
                <button
                  key={mode}
                  type="button"
                  onClick={() => setRegistrationMode(mode)}
                  className={[
                    'rounded-lg border px-3 py-2 text-sm transition-colors',
                    form.registration?.mode === mode
                      ? 'border-primary bg-primary/5 text-foreground font-medium'
                      : 'border-border bg-transparent text-muted-foreground hover:border-primary/40 hover:text-foreground',
                  ].join(' ')}
                >
                  {mode.replace('_', ' ')}
                </button>
              ))}
            </div>
          </div>
          <div className="mt-4 space-y-2">
            <Label>Password collection</Label>
            <p className="text-xs text-muted-foreground">
              When to ask for the password during signup.
            </p>
            <div className="grid grid-cols-2 gap-2">
              {([
                ['password_first', 'At signup'],
                ['after_verify', 'After email verified'],
              ] as const).map(([value, label]) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setPasswordStrategy(value)}
                  className={[
                    'rounded-lg border px-3 py-2 text-sm transition-colors',
                    (form.registration?.password_strategy ?? 'password_first') === value
                      ? 'border-primary bg-primary/5 text-foreground font-medium'
                      : 'border-border bg-transparent text-muted-foreground hover:border-primary/40 hover:text-foreground',
                  ].join(' ')}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={save} disabled={busy}>
            {busy && <Loader2 className="size-4 animate-spin" />}
            Save
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Policies tab (password + session + MFA)
// ---------------------------------------------------------------------------

function NumberField({
  label,
  description,
  value,
  onChange,
  min,
  max,
  placeholder,
}: {
  label: string;
  description?: string;
  value: number | undefined;
  onChange: (v: number | undefined) => void;
  min?: number;
  max?: number;
  placeholder?: string;
}) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {description && <p className="text-xs text-muted-foreground">{description}</p>}
      <Input
        type="number"
        min={min}
        max={max}
        placeholder={placeholder ?? '—'}
        value={value ?? ''}
        onChange={(e) => {
          const v = e.target.value === '' ? undefined : Number(e.target.value);
          onChange(v);
        }}
        className="w-36"
      />
    </div>
  );
}

function SwitchField({
  label,
  description,
  id,
  checked,
  onCheckedChange,
}: {
  label: string;
  description?: string;
  id: string;
  checked: boolean | undefined;
  onCheckedChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="space-y-0.5">
        <Label htmlFor={id}>{label}</Label>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </div>
      <Switch id={id} checked={!!checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}

function PasswordPolicyCard({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminConfigPasswordPolicy({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
        }),
      ),
    [projectId, env],
  );

  const [form, setForm] = useState<PasswordPolicy | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  async function save() {
    if (!form) return;
    setBusy(true);
    try {
      const updated = await call(
        patchV1ProjectsByProjectIdAdminConfigPasswordPolicy({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
          body: form,
        }),
      );
      setForm(updated);
      toast.success('Password policy saved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to save password policy');
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <LoadingState label="Loading password policy…" />;
  if (error) return <ErrorState error={error} onRetry={reload} />;
  if (!form) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Password policy</CardTitle>
        <CardDescription>Rules enforced at sign-up and password change.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <NumberField
            label="Minimum length"
            description="Characters required (8–128)."
            value={form.min_length}
            onChange={(v) => setForm((p) => p ? { ...p, min_length: v } : p)}
            min={6}
            max={128}
            placeholder="8"
          />
          <NumberField
            label="History"
            description="Prevent reuse of the last N passwords."
            value={form.history}
            onChange={(v) => setForm((p) => p ? { ...p, history: v } : p)}
            min={0}
            max={24}
            placeholder="0"
          />
          <NumberField
            label="zxcvbn minimum score"
            description="Strength score 0–4 (4 = very strong)."
            value={form.zxcvbn_min_score}
            onChange={(v) => setForm((p) => p ? { ...p, zxcvbn_min_score: v } : p)}
            min={0}
            max={4}
            placeholder="0"
          />
        </div>
        <Separator />
        <SwitchField
          id="pw-breached"
          label="Breached password check"
          description="Reject passwords found in HaveIBeenPwned."
          checked={form.breached_check}
          onCheckedChange={(v) => setForm((p) => p ? { ...p, breached_check: v } : p)}
        />
      </CardContent>
      <CardFooter className="justify-end">
        <Button onClick={save} disabled={busy}>
          {busy && <Loader2 className="size-4 animate-spin" />}
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

function SessionPolicyCard({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminConfigSessionPolicy({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
        }),
      ),
    [projectId, env],
  );

  const [form, setForm] = useState<SessionPolicy | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  async function save() {
    if (!form) return;
    setBusy(true);
    try {
      const updated = await call(
        patchV1ProjectsByProjectIdAdminConfigSessionPolicy({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
          body: form,
        }),
      );
      setForm(updated);
      toast.success('Session policy saved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to save session policy');
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <LoadingState label="Loading session policy…" />;
  if (error) return <ErrorState error={error} onRetry={reload} />;
  if (!form) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Session policy</CardTitle>
        <CardDescription>Token lifetimes and rotation behaviour. Durations in seconds.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <NumberField
            label="Access token TTL"
            description="Seconds until access tokens expire."
            value={form.access_ttl}
            onChange={(v) => setForm((p) => p ? { ...p, access_ttl: v } : p)}
            min={60}
            placeholder="3600"
          />
          <NumberField
            label="Refresh token TTL"
            description="Seconds until refresh tokens expire."
            value={form.refresh_ttl}
            onChange={(v) => setForm((p) => p ? { ...p, refresh_ttl: v } : p)}
            min={60}
            placeholder="2592000"
          />
          <NumberField
            label="Idle timeout"
            description="Expire session after inactivity (seconds)."
            value={form.idle_timeout}
            onChange={(v) => setForm((p) => p ? { ...p, idle_timeout: v } : p)}
            min={0}
            placeholder="unset"
          />
          <NumberField
            label="Absolute timeout"
            description="Hard cap on session lifetime (seconds)."
            value={form.absolute_timeout}
            onChange={(v) => setForm((p) => p ? { ...p, absolute_timeout: v } : p)}
            min={0}
            placeholder="unset"
          />
        </div>
        <Separator />
        <SwitchField
          id="sess-reuse"
          label="Refresh token reuse detection"
          description="Revoke all tokens in the family when a used refresh token is replayed."
          checked={form.reuse_detection}
          onCheckedChange={(v) => setForm((p) => p ? { ...p, reuse_detection: v } : p)}
        />
      </CardContent>
      <CardFooter className="justify-end">
        <Button onClick={save} disabled={busy}>
          {busy && <Loader2 className="size-4 animate-spin" />}
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

const KNOWN_MFA_FACTORS = ['totp', 'sms', 'email_otp', 'webauthn', 'backup_codes'];

function MfaPolicyCard({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminConfigMfaPolicy({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
        }),
      ),
    [projectId, env],
  );

  const [form, setForm] = useState<MfaPolicy | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  function toggleFactor(factor: string) {
    setForm((prev) => {
      if (!prev) return prev;
      const factors = prev.allowed_factors ?? [];
      return {
        ...prev,
        allowed_factors: factors.includes(factor)
          ? factors.filter((f) => f !== factor)
          : [...factors, factor],
      };
    });
  }

  async function save() {
    if (!form) return;
    setBusy(true);
    try {
      const updated = await call(
        patchV1ProjectsByProjectIdAdminConfigMfaPolicy({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
          body: form,
        }),
      );
      setForm(updated);
      toast.success('MFA policy saved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to save MFA policy');
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <LoadingState label="Loading MFA policy…" />;
  if (error) return <ErrorState error={error} onRetry={reload} />;
  if (!form) return null;

  const allowedFactors = form.allowed_factors ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>MFA policy</CardTitle>
        <CardDescription>Multi-factor authentication settings.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label>Allowed factors</Label>
          <p className="text-xs text-muted-foreground">Enable the second factors users may enrol.</p>
          <div className="mt-2 space-y-2">
            {KNOWN_MFA_FACTORS.map((factor) => (
              <div key={factor} className="flex items-center justify-between">
                <Label htmlFor={`factor-${factor}`} className="font-normal capitalize">
                  {factor.replace('_', ' ')}
                </Label>
                <Switch
                  id={`factor-${factor}`}
                  checked={allowedFactors.includes(factor)}
                  onCheckedChange={() => toggleFactor(factor)}
                />
              </div>
            ))}
          </div>
        </div>
        <Separator />
        <SwitchField
          id="mfa-admins"
          label="Required for admins"
          description="Enforce MFA for all admin accounts."
          checked={form.required_for_admins}
          onCheckedChange={(v) => setForm((p) => p ? { ...p, required_for_admins: v } : p)}
        />
        <SwitchField
          id="mfa-remember"
          label="Remember device"
          description="Allow users to trust a device for 30 days."
          checked={form.remember_device}
          onCheckedChange={(v) => setForm((p) => p ? { ...p, remember_device: v } : p)}
        />
      </CardContent>
      <CardFooter className="justify-end">
        <Button onClick={save} disabled={busy}>
          {busy && <Loader2 className="size-4 animate-spin" />}
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

function PoliciesTab({ projectId, env }: { projectId: string; env: string }) {
  return (
    <div className="space-y-4">
      <PasswordPolicyCard projectId={projectId} env={env} />
      <SessionPolicyCard projectId={projectId} env={env} />
      <MfaPolicyCard projectId={projectId} env={env} />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Rate limits tab
// ---------------------------------------------------------------------------

const BLANK_RULE: RateLimitRule = {
  endpoint: '',
  action: '',
  limit: 100,
  window_seconds: 60,
  by: 'ip',
};

function RateLimitsTab({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminConfigRateLimits({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
        }),
      ),
    [projectId, env],
  );

  const [rules, setRules] = useState<RateLimitRule[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) setRules(data.rules ?? []);
  }, [data]);

  function updateRule(idx: number, patch: Partial<RateLimitRule>) {
    setRules((prev) => prev.map((r, i) => (i === idx ? { ...r, ...patch } : r)));
  }

  function addRule() {
    setRules((prev) => [...prev, { ...BLANK_RULE }]);
  }

  function removeRule(idx: number) {
    setRules((prev) => prev.filter((_, i) => i !== idx));
  }

  async function save() {
    setBusy(true);
    const body: RateLimits = { rules };
    try {
      const updated = await call(
        patchV1ProjectsByProjectIdAdminConfigRateLimits({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
          body,
        }),
      );
      setRules(updated.rules ?? []);
      toast.success('Rate limits saved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to save rate limits');
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <LoadingState />;
  if (error) return <ErrorState error={error} onRetry={reload} />;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Rate limits</CardTitle>
        <CardDescription>
          Per-endpoint throttling rules. Each rule matches an endpoint and action, counting requests
          by the specified key within the time window.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {rules.length === 0 && (
          <p className="text-sm text-muted-foreground">
            No rules configured. Click &ldquo;Add rule&rdquo; to create one.
          </p>
        )}
        {rules.map((rule, idx) => (
          <div key={idx} className="rounded-lg border bg-muted/30 p-4">
            <div className="mb-3 flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Rule {idx + 1}
              </span>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => removeRule(idx)}
                title="Remove rule"
              >
                <Trash2 className="size-3.5 text-destructive" />
              </Button>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <div className="space-y-1.5">
                <Label>Endpoint</Label>
                <Input
                  placeholder="/v1/auth/sign-in"
                  value={rule.endpoint ?? ''}
                  onChange={(e) => updateRule(idx, { endpoint: e.target.value })}
                />
              </div>
              <div className="space-y-1.5">
                <Label>Action</Label>
                <Input
                  placeholder="sign_in"
                  value={rule.action ?? ''}
                  onChange={(e) => updateRule(idx, { action: e.target.value })}
                />
              </div>
              <div className="space-y-1.5">
                <Label>Count by</Label>
                <Input
                  placeholder="ip"
                  value={rule.by ?? ''}
                  onChange={(e) => updateRule(idx, { by: e.target.value })}
                />
              </div>
              <div className="space-y-1.5">
                <Label>Limit</Label>
                <Input
                  type="number"
                  min={1}
                  value={rule.limit ?? ''}
                  onChange={(e) => updateRule(idx, { limit: e.target.value === '' ? undefined : Number(e.target.value) })}
                  className="w-28"
                />
              </div>
              <div className="space-y-1.5">
                <Label>Window (seconds)</Label>
                <Input
                  type="number"
                  min={1}
                  value={rule.window_seconds ?? ''}
                  onChange={(e) => updateRule(idx, { window_seconds: e.target.value === '' ? undefined : Number(e.target.value) })}
                  className="w-28"
                />
              </div>
            </div>
          </div>
        ))}
        <Button variant="outline" size="sm" onClick={addRule}>
          <Plus className="size-3.5" />
          Add rule
        </Button>
      </CardContent>
      <CardFooter className="justify-end">
        <Button onClick={save} disabled={busy}>
          {busy && <Loader2 className="size-4 animate-spin" />}
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// Terms / consents tab
// ---------------------------------------------------------------------------

function emptyConsent(): ConsentDocument {
  return { key: 'tos', version: new Date().toISOString().slice(0, 10), required: true, locale: 'en' };
}

function TermsTab({ projectId, env }: { projectId: string; env: string }) {
  const { data, loading, error, reload } = useApi(
    () =>
      call(
        getV1ProjectsByProjectIdAdminConsents({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
        }),
      ),
    [projectId, env],
  );

  const [form, setForm] = useState<ConsentConfig>({ documents: [] });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) setForm({ documents: data.documents ?? [] });
  }, [data]);

  function updateDoc(index: number, patch: Partial<ConsentDocument>) {
    setForm((prev) => ({
      documents: (prev.documents ?? []).map((doc, i) => (i === index ? { ...doc, ...patch } : doc)),
    }));
  }

  function addDoc() {
    setForm((prev) => ({ documents: [...(prev.documents ?? []), emptyConsent()] }));
  }

  function removeDoc(index: number) {
    setForm((prev) => ({ documents: (prev.documents ?? []).filter((_, i) => i !== index) }));
  }

  async function save() {
    setBusy(true);
    try {
      const updated = await call(
        putV1ProjectsByProjectIdAdminConsents({
          path: { project_id: projectId },
          headers: ENV_HEADER(env),
          body: form,
        }),
      );
      setForm(updated);
      toast.success('Terms saved');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to save terms');
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <LoadingState label="Loading terms…" />;
  if (error) return <ErrorState error={error} onRetry={reload} />;

  const docs = form.documents ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Terms and consent documents</CardTitle>
        <CardDescription>Documents returned in the public SDK config and accepted as user consents.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {docs.map((doc, idx) => (
          <div key={`${doc.key}-${doc.version}-${idx}`} className="space-y-3 rounded-lg border p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="grid flex-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <div className="space-y-1.5">
                  <Label>Key</Label>
                  <Input value={doc.key} onChange={(e) => updateDoc(idx, { key: e.target.value })} placeholder="tos" />
                </div>
                <div className="space-y-1.5">
                  <Label>Version</Label>
                  <Input value={doc.version} onChange={(e) => updateDoc(idx, { version: e.target.value })} placeholder="2026-06-08" />
                </div>
                <div className="space-y-1.5">
                  <Label>Locale</Label>
                  <Input value={doc.locale ?? ''} onChange={(e) => updateDoc(idx, { locale: e.target.value })} placeholder="en" />
                </div>
                <div className="flex items-end gap-2">
                  <Switch
                    id={`consent-required-${idx}`}
                    checked={doc.required ?? true}
                    onCheckedChange={(required) => updateDoc(idx, { required })}
                  />
                  <Label htmlFor={`consent-required-${idx}`} className="pb-1.5">Required</Label>
                </div>
              </div>
              <Button variant="ghost" size="icon-sm" onClick={() => removeDoc(idx)} title="Remove document">
                <Trash2 className="size-3.5 text-destructive" />
              </Button>
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label>Title</Label>
                <Input value={doc.title ?? ''} onChange={(e) => updateDoc(idx, { title: e.target.value })} placeholder="Terms of Service" />
              </div>
              <div className="space-y-1.5">
                <Label>URL</Label>
                <Input value={doc.url ?? ''} onChange={(e) => updateDoc(idx, { url: e.target.value })} placeholder="https://example.com/terms" />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label>Body</Label>
              <textarea
                value={doc.body ?? ''}
                onChange={(e) => updateDoc(idx, { body: e.target.value })}
                rows={5}
                className="w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                placeholder="Localized terms text shown by your application."
              />
            </div>
          </div>
        ))}
        <Button variant="outline" size="sm" onClick={addDoc}>
          <Plus className="size-3.5" />
          Add document
        </Button>
      </CardContent>
      <CardFooter className="justify-end">
        <Button onClick={save} disabled={busy}>
          {busy && <Loader2 className="size-4 animate-spin" />}
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// Environment selector
// ---------------------------------------------------------------------------

const COMMON_ENVS = ['live', 'staging', 'dev'];

function EnvPicker({ env, onChange }: { env: string; onChange: (e: string) => void }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-muted-foreground">Environment:</span>
      <div className="flex rounded-lg border border-border bg-muted p-[3px]">
        {COMMON_ENVS.map((e) => (
          <button
            key={e}
            type="button"
            onClick={() => onChange(e)}
            className={[
              'rounded-md px-2.5 py-0.5 text-sm transition-colors',
              e === env
                ? 'bg-background font-medium text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            ].join(' ')}
          >
            {e}
          </button>
        ))}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page root
// ---------------------------------------------------------------------------

export function ConfigPage() {
  const { projectId } = useParams();
  const [env, setEnv] = useState(DEFAULT_ENV);

  return (
    <div>
      <PageHeader
        title="Configuration"
        description="Auth methods, security policies, and rate limits — per environment."
        actions={<EnvPicker env={env} onChange={setEnv} />}
      />

      <Tabs defaultValue="auth">
        <TabsList className="mb-4">
          <TabsTrigger value="auth">Auth &amp; methods</TabsTrigger>
          <TabsTrigger value="policies">Policies</TabsTrigger>
          <TabsTrigger value="terms">Terms</TabsTrigger>
          <TabsTrigger value="rate-limits">Rate limits</TabsTrigger>
        </TabsList>

        <TabsContent value="auth">
          <AuthTab projectId={projectId!} env={env} />
        </TabsContent>

        <TabsContent value="policies">
          <PoliciesTab projectId={projectId!} env={env} />
        </TabsContent>

        <TabsContent value="terms">
          <TermsTab projectId={projectId!} env={env} />
        </TabsContent>

        <TabsContent value="rate-limits">
          <RateLimitsTab projectId={projectId!} env={env} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
