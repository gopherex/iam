/**
 * useFlow — React hook wrapping FlowController for the /flow page.
 *
 * - Creates (or reuses) a singleton FlowController on mount.
 * - Resumes any in-progress flow on mount:
 *     1. If `?flow=<token>` is present, resumes that token (deep-link).
 *     2. Otherwise calls resume() which tries the HttpOnly iam_flow cookie,
 *        then falls back to localStorage.
 * - Subscribes to onChange and re-renders on every state change.
 * - Exposes `start`, `submit`, `resend`, `abandon` bound to the controller.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  createFlowController,
  type FlowController,
  type FlowState,
  type FlowKind,
} from '@gopherex/iam-sdk';
import { IamAuthError } from '@gopherex/iam-sdk';

export interface UseFlowReturn {
  state: FlowState | null;
  error: IamAuthError | null;
  loading: boolean;
  start(params: {
    kind: FlowKind;
    email?: string;
    password?: string;
    name?: string;
    captchaToken?: string;
  }): Promise<void>;
  submit(action: string, payload?: Record<string, unknown>): Promise<void>;
  resend(): Promise<void>;
  abandon(): Promise<void>;
}

// Singleton controller — persists across React StrictMode double-mounts.
let _controller: FlowController | null = null;

function getController(): FlowController {
  if (!_controller) {
    _controller = createFlowController({
      // Same-origin in production; dev server proxies /v1 to the API.
      baseUrl: '',
      // The flow API is public (no X-Client-Id enforcement from the admin app
      // side), but the controller sends it for consistency.  The web/sdk.ts
      // master-key interceptor only adds Authorization, not X-Client-Id, so we
      // pass an empty string. The backend will use the project from the cookie.
      clientId: '',
    });
  }
  return _controller;
}

export function useFlow(opts?: { flowToken?: string }): UseFlowReturn {
  const controller = getController();
  const [state, setState] = useState<FlowState | null>(controller.currentState);
  const [error, setError] = useState<IamAuthError | null>(null);
  const [loading, setLoading] = useState(true);
  const resumedRef = useRef(false);

  // Subscribe to controller changes.
  useEffect(() => {
    const unsub = controller.onChange((nextState, nextError) => {
      setState(nextState);
      setError(nextError);
    });
    return unsub;
  }, [controller]);

  // Resume on mount (once per mount, not per StrictMode re-run).
  useEffect(() => {
    if (resumedRef.current) return;
    resumedRef.current = true;

    void (async () => {
      setLoading(true);
      try {
        if (opts?.flowToken) {
          await controller.resumeByToken(opts.flowToken);
        } else {
          await controller.resume();
        }
      } finally {
        setLoading(false);
      }
    })();
  }, [controller, opts?.flowToken]);

  const start = useCallback(
    async (params: {
      kind: FlowKind;
      email?: string;
      password?: string;
      name?: string;
      captchaToken?: string;
      locale?: string;
    }) => {
      setLoading(true);
      try {
        // Default flow-email language to the browser's, so verification/continue
        // emails match what the user sees. Apps may override via params.locale.
        await controller.start({
          ...params,
          locale: params.locale ?? (typeof navigator !== 'undefined' ? navigator.language : undefined),
        });
      } finally {
        setLoading(false);
      }
    },
    [controller],
  );

  const submit = useCallback(
    async (action: string, payload?: Record<string, unknown>) => {
      setLoading(true);
      try {
        await controller.submit(action, payload);
      } finally {
        setLoading(false);
      }
    },
    [controller],
  );

  const resend = useCallback(async () => {
    setLoading(true);
    try {
      await controller.resend();
    } finally {
      setLoading(false);
    }
  }, [controller]);

  const abandon = useCallback(async () => {
    setLoading(true);
    try {
      await controller.abandon();
    } finally {
      setLoading(false);
    }
  }, [controller]);

  return { state, error, loading, start, submit, resend, abandon };
}
