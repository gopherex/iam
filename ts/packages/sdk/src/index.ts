// @gopherex/iam-sdk
//
// Two layers:
//  - the generated, low-level typed client (one function per API operation plus
//    the request/response types), re-exported from ./gen;
//  - a Supabase-style auth client (createIamClient) that manages the session,
//    refreshes tokens, and syncs across tabs.

export * from './gen';
// The shared fetch client — configure it directly (baseUrl, auth, interceptors)
// for non-auth callers such as an admin panel using the raw operations.
export { client } from './gen/client.gen';

export { createIamClient, IamAuth, IamConfig } from './auth/client';
export { IamAuthError } from './auth/types';
export type {
  AuthChangeEvent,
  AuthData,
  AuthResponse,
  IamClientOptions,
  Session,
  StorageAdapter,
  Subscription,
} from './auth/types';
export { MemoryStorage } from './auth/storage';
export { createFlowController } from './auth/flow';
export type {
  FlowController,
  FlowControllerOptions,
  FlowChangeCallback,
  FlowKind,
} from './auth/flow';
// Note: FlowState is exported from './gen' via the wildcard export above.
