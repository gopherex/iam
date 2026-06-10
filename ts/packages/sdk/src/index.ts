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

// Ergonomic authenticated namespaces (exposed on createIamClient as
// iam.account / iam.mfa / iam.webauthn).
export { IamAccount } from './auth/account';
export type {
  AccountProfile,
  AccountCapabilities,
  AccountActivity,
  ExportJob,
  ExportStatus,
  ConsentRecord,
  IdentityList,
  MergeStartResult,
  MergeConfirmResult,
  SessionList,
  CurrentSessionResult,
  SessionResult,
  RevokeAllResult,
} from './auth/account';
export { IamMfa } from './auth/mfa';
export type {
  TotpEnrollResult,
  TotpVerifyResult,
  SmsEnrollResult,
  EmailEnrollResult,
  WebAuthnEnrollOptionsResult,
  WebAuthnEnrollVerifyResult,
  RecoveryCodesResult,
} from './auth/mfa';
export { IamWebAuthn } from './auth/webauthn';
export type {
  RegisterOptionsResult,
  RegisterVerifyResult,
  CredentialList,
  CredentialResult,
} from './auth/webauthn';
export { IamTokens } from './auth/tokens';
export { IamOidc } from './auth/oidc';
