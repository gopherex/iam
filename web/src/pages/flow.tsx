/**
 * /flow — Server-side resumable auth flow page (admin console surface).
 *
 * Public route (no RequireAuth). The step rendering itself lives in
 * components/flow-steps and is shared with the hosted OIDC provider login
 * screen, so both drive one engine and one set of components.
 *
 * Deep-link support:
 *   ?flow=<token>  — resume a specific flow token (cross-device link)
 *   ?kind=<kind>   — pre-select the flow kind on the credential form (default: signin)
 *
 * Cookie resume: on mount, if no ?flow param is present, useFlow calls
 * resume() which first tries GET /v1/auth/flows/current (HttpOnly cookie path)
 * then falls back to the token in localStorage.
 */

import { useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { CheckCircle, Loader2, ShieldCheck } from 'lucide-react';
import type { FlowKind } from '@gopherex/iam-sdk';

import { FlowSteps, flowStepMeta } from '@/components/flow-steps';
import { ThemeToggle } from '@/components/theme-toggle';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { translator } from '@/lib/i18n';
import { useFlow } from '@/lib/use-flow';

export function FlowPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const flowToken = searchParams.get('flow') ?? undefined;
  const kindParam = (searchParams.get('kind') ?? 'signin') as FlowKind;

  const flow = useFlow({ flowToken });
  const { state, loading } = flow;

  // The console has no project context of its own, so it speaks the source
  // language; the provider pages resolve the project's locale instead.
  const t = translator('en');

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

  const step = state?.step ?? 'collect_credentials';
  const meta = flowStepMeta(t, step);

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
          {loading && !state && (
            <div className="flex justify-center py-6">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          )}
          <FlowSteps flow={flow} kind={kindParam} t={t} />
        </CardContent>
      </Card>
    </div>
  );
}
