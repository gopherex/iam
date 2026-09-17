import assert from 'node:assert/strict';
import { test, afterEach } from 'node:test';
import { createIamClient, MemoryStorage } from '../dist/index.js';

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });
const state = (overrides = {}) => ({ flow_token: 'sft_first', kind: 'security_review', status: 'pending', step: 'verify_identity', version: 1, next_actions: ['verify_code', 'request_support'], expires_at: new Date(Date.now() + 600000).toISOString(), attempts_left: 5, all_sessions: false, revoked_session_ids: [], ...overrides });
const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const options = (extra = {}) => ({ baseUrl: 'https://iam.example', clientId: 'project', autoRefresh: false, multiTab: false, storage: new MemoryStorage(), securityStorage: new MemoryStorage(), ...extra });

test('security proof is isolated from ordinary credentials and rotates persisted capability', async () => {
  const requests = [];
  globalThis.fetch = async request => {
    requests.push(request);
    if (request.method === 'GET') return json(state());
    assert.equal(request.headers.get('X-Security-Token'), 'sft_first');
    assert.equal(request.headers.get('Authorization'), null);
    const body = await request.json();
    assert.equal(body.version, 1);
    assert.equal(body.code, '123456');
    return json(state({ flow_token: 'sft_second', version: 2, step: 'review', next_actions: ['report_activity'] }));
  };
  const opts = options();
  const iam = createIamClient(opts);
  await iam.security.flow.resumeByToken('sft_first');
  const result = await iam.security.flow.verifyCode('123456');
  assert.equal(result.error, null);
  assert.equal(result.state.flow_token, 'sft_second');
  globalThis.fetch = async request => {
    assert.equal(request.headers.get('X-Security-Token'), 'sft_second');
    return json(result.state);
  };
  const reloaded = createIamClient(opts);
  assert.equal((await reloaded.security.flow.resume()).state.version, 2);
  assert.equal(await iam.auth.getSession(), null);
  assert.equal(requests.length, 2);
});

test('tenant and environment flow storage do not collide', async () => {
  const opts = options();
  globalThis.fetch = async () => json(state());
  await createIamClient(opts).security.flow.resumeByToken('sft_first');
  let requested = false;
  globalThis.fetch = async () => { requested = true; throw new Error('unexpected request'); };
  assert.equal((await createIamClient({ ...opts, environment: 'test' }).security.flow.resume()).state, null);
  assert.equal((await createIamClient({ ...opts, clientId: 'other' }).security.flow.resume()).state, null);
  assert.equal(requested, false);
});

test('revoking the current session keeps recovery alive and completing it does not sign in', async () => {
  const opts = options();
  const claims = Buffer.from(JSON.stringify({ sid: 'session-1' })).toString('base64url');
  await opts.storage.setItem('iam.session', JSON.stringify({ access_token: `header.${claims}.signature`, refresh_token: 'refresh', token_type: 'Bearer', expires_at: Date.now() + 600000, user: { id: 'user' } }));
  const iam = createIamClient(opts);
  await iam.auth.ready();
  globalThis.fetch = async () => json(state({ step: 'review', next_actions: ['report_activity'] }));
  await iam.security.flow.resumeByToken('sft_first');
  globalThis.fetch = async request => {
    const body = await request.json();
    assert.equal(body.all_sessions, false);
    return json(state({ flow_token: 'sft_recover', version: 2, step: 'restore_access', revoked_session_ids: ['session-1'], next_actions: ['set_password'] }));
  };
  await iam.security.flow.reportActivity();
  assert.equal(await iam.auth.getSession(), null);
  assert.equal(iam.security.flow.currentState.step, 'restore_access');
  globalThis.fetch = async request => {
    assert.equal(request.headers.get('Authorization'), null);
    return json(state({ status: 'completed', step: 'completed', version: 3, outcome: 'account_secured', next_actions: [] }));
  };
  await iam.security.flow.setPassword('strong password');
  assert.equal(await iam.auth.getSession(), null);
  assert.equal((await createIamClient(opts).security.flow.resume()).state, null);
});

test('completed unfamiliar-device proof is attached to an explicit login retry', async () => {
  const iam = createIamClient(options());
  globalThis.fetch = async () => json(state({ status: 'completed', step: 'completed', outcome: 'sign_in_approved', next_actions: [] }));
  await iam.security.flow.resumeByToken('sft_first');
  globalThis.fetch = async request => {
    assert.equal(request.headers.get('X-Security-Proof'), 'sft_first');
    return json({ error: { code: 'invalid_credentials', message: 'Invalid credentials' } }, 401);
  };
  assert.equal((await iam.auth.signInWithPassword({ email: 'user@example.com', password: 'bad' })).error.code, 'invalid_credentials');
});

test('illegal transitions are rejected locally and server proof failures preserve retry state', async () => {
  const iam = createIamClient(options());
  globalThis.fetch = async () => json(state());
  await iam.security.flow.resumeByToken('sft_first');
  assert.equal((await iam.security.flow.reportActivity()).error.code, 'invalid_flow_action');
  globalThis.fetch = async () => json(state({ flow_token: 'sft_retry', version: 2, attempts_left: 4, error_code: 'invalid_code' }));
  const result = await iam.security.flow.verifyCode('wrong');
  assert.equal(result.error.code, 'invalid_code');
  assert.equal(result.state.attempts_left, 4);
  assert.equal(result.state.flow_token, 'sft_retry');
});


test('revoking the current device clears its local session without another request', async () => {
  const opts = options();
  const claims = Buffer.from(JSON.stringify({ sid: 'old-session' })).toString('base64url');
  await opts.storage.setItem('iam.session', JSON.stringify({ access_token: `header.${claims}.signature`, refresh_token: 'refresh', token_type: 'Bearer', expires_at: Date.now() + 600000, user: { id: 'user' } }));
  const iam = createIamClient(opts);
  await iam.auth.ready();
  let requests = 0;
  globalThis.fetch = async request => {
    requests++;
    assert.ok(request.url.endsWith('/v1/security/devices/device-1'));
    return json({ id: 'device-1', current: true, revoked: true, session_ids: ['recent-session'] });
  };
  assert.equal((await iam.security.updateDevice('device-1', { action: 'revoke' })).error, null);
  assert.equal(await iam.auth.getSession(), null);
  assert.equal(requests, 1);
});


test('lost continuation response resumes with the same secret after reload', async () => {
  const opts = options();
  const iam = createIamClient(opts);
  await iam.auth.ready();
  const raw = 'https://app.example/security#iam_security=one-time&tab=activity';
  globalThis.window = { location: { href: raw }, history: { state: null, replaceState(_s, _t, path) { window.location.href = 'https://app.example' + path; } } };
  let attempt;
  try {
    globalThis.fetch = async request => {
      attempt = await request.json();
      assert.ok(attempt.exchange_key.length >= 32);
      throw new Error('response lost after commit');
    };
    assert.ok((await iam.security.flow.continueFromURL(raw)).error);
    assert.equal(window.location.href, 'https://app.example/security#tab=activity');
  } finally { delete globalThis.window; }
  globalThis.fetch = async request => {
    assert.deepEqual(await request.json(), attempt);
    return json(state());
  };
  const reloaded = createIamClient(opts);
  assert.equal((await reloaded.security.flow.resume()).error, null);
  globalThis.fetch = async request => {
    assert.equal(request.method, 'GET');
    return json(state());
  };
  assert.equal((await createIamClient(opts).security.flow.resume()).state.flow_token, 'sft_first');
});

test('failed persistence keeps the continuation in the URL and does not exchange it', async () => {
  const iam = createIamClient(options({ securityStorage: { getItem: async () => null, setItem: async () => {}, removeItem: async () => {} } }));
  await iam.auth.ready();
  const raw = 'https://app.example/security#iam_security=one-time';
  globalThis.window = { location: { href: raw }, history: { state: null, replaceState() { assert.fail('must keep fragment'); } } };
  try {
    globalThis.fetch = async () => { assert.fail('must persist before request'); };
    assert.equal((await iam.security.flow.continueFromURL(raw)).error.code, 'storage_error');
    assert.equal(window.location.href, raw);
  } finally { delete globalThis.window; }
});

test('verified recovery cancels the advertised deletion without an ordinary session', async () => {
  const iam = createIamClient(options());
  const deletion = { ok: true, request_id: 'request-1', status: 'pending', grace_days: 7 };
  globalThis.fetch = async () => json(state({ step: 'restore_access', next_actions: ['cancel_deletion', 'set_password'], deletion }));
  await iam.security.flow.resumeByToken('sft_first');
  globalThis.fetch = async request => {
    assert.equal(request.headers.get('Authorization'), null);
    assert.equal(request.headers.get('X-Security-Token'), 'sft_first');
    assert.deepEqual(await request.json(), { request_id: 'request-1', action: 'cancel_deletion', version: 1 });
    return json(state({ flow_token: 'sft_second', version: 2, step: 'restore_access', next_actions: ['set_password'], deletion: { ...deletion, status: 'cancelled' } }));
  };
  assert.equal((await iam.security.flow.cancelDeletion('request-1')).state.deletion.status, 'cancelled');
  assert.equal(await iam.auth.getSession(), null);
});
