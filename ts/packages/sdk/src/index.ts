// @gopherex/iam-sdk
//
// Two layers:
//  - the generated, low-level typed client (one function per API operation plus
//    the request/response types), re-exported from ./gen;
//  - a Supabase-style auth client (createIamClient) that manages the session,
//    refreshes tokens, and syncs across tabs.

export * from './gen';

export { createIamClient, IamAuth } from './auth/client';
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
