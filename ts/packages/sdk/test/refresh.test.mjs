import assert from 'node:assert/strict';
import { test } from 'node:test';
import { setImmediate } from 'node:timers/promises';
import { createIamClient, MemoryStorage } from '../dist/index.js';

const json = (body, status = 200, headers = {}) => new Response(JSON.stringify(body), {
  status, headers: { 'Content-Type': 'application/json', ...headers },
});
const tokens = (suffix = 'new') => ({
  access_token: `access-${suffix}`, refresh_token: `refresh-${suffix}`,
  token_type: 'Bearer', expires_in: 3600,
});
const success = () => json({ session: tokens(), user: { id: 'user' } });

async function setup(t, autoRefresh = false, expiresIn = 600000) {
  t.mock.timers.enable({ apis: ['Date', 'setTimeout'], now: 1800000000000 });
  t.mock.method(Math, 'random', () => 0);
  const storage = new MemoryStorage();
  const session = {
    ...tokens('old'), expires_at: Date.now() + expiresIn, user: { id: 'user' },
  };
  await storage.setItem('iam.session', JSON.stringify(session));
  const iam = createIamClient({ baseUrl: 'https://iam.example', clientId: 'project', storage, autoRefresh, multiTab: false });
  const events = [];
  iam.auth.onAuthStateChange(event => events.push(event));
  await iam.auth.ready();
  await setImmediate();
  return { iam, storage, session, events };
}

for (const [name, reply] of [
  ['500', () => json({ error: { code: 'internal_error' } }, 500)],
  ['503 HTML', () => new Response('<html>Unavailable</html>', { status: 503 })],
  ['429', () => json({ error: { code: 'rate_limited' } }, 429)],
  ['408', () => new Response('', { status: 408 })],
  ['network error', () => { throw new TypeError('fetch failed'); }],
  ['broken JSON', () => new Response('{', { headers: { 'Content-Type': 'application/json' } })],
  ['missing tokens', () => json({})],
  ['missing expiry', () => json({ session: { access_token: 'incomplete' } })],
  ['unrelated 403', () => json({ error: { code: 'forbidden' } }, 403)],
  ['500 with misleading auth code', () => json({ error: 'invalid_grant' }, 500)],
]) {
  test(`refresh preserves the session on ${name} and gates repeated attempts`, async t => {
    const { iam, storage, session, events } = await setup(t);
    const fetch = t.mock.method(globalThis, 'fetch', async () => reply());
    assert.deepEqual(await iam.auth.refreshSession(), session);
    assert.deepEqual(await iam.auth.getSession(), session);
    assert.deepEqual(JSON.parse(await storage.getItem('iam.session')), session);
    assert.deepEqual(events, ['INITIAL_SESSION']);
    await iam.auth.refreshSession();
    assert.equal(fetch.mock.callCount(), 1);
    t.mock.timers.tick(60000);
    await setImmediate();
    assert.equal(fetch.mock.callCount(), 1, 'autoRefresh:false must not start a background retry');
  });
}

for (const [status, body] of [
  [400, { error: 'invalid_grant' }],
  [400, { error: { code: 'invalid_grant' } }],
  [401, { error: { code: 'token_revoked' } }],
  [401, null],
  [403, { error: { code: 'token_revoked' } }],
]) {
  test(`refresh refusal ${status} ${JSON.stringify(body)} clears storage and emits SIGNED_OUT`, async t => {
    const { iam, storage, events } = await setup(t, true);
    const fetch = t.mock.method(globalThis, 'fetch', async () => json(body, status));
    assert.equal(await iam.auth.refreshSession(), null);
    assert.equal(await iam.auth.getSession(), null);
    assert.equal(await storage.getItem('iam.session'), null);
    assert.deepEqual(events, ['INITIAL_SESSION', 'SIGNED_OUT']);
    t.mock.timers.tick(3600000);
    await setImmediate();
    assert.equal(fetch.mock.callCount(), 1);
  });
}

test('background retries back off, respect Retry-After, and persist successful rotation', async t => {
  const { iam, storage, events } = await setup(t, true);
  let calls = 0;
  t.mock.method(globalThis, 'fetch', async request => {
    assert.deepEqual(await request.json(), { refresh_token: 'refresh-old' });
    calls++;
    if (calls === 1) throw new TypeError('offline');
    if (calls === 2) return json({}, 429, { 'Retry-After': '5' });
    return success();
  });
  await iam.auth.refreshSession();
  t.mock.timers.tick(499);
  await setImmediate();
  assert.equal(calls, 1);
  t.mock.timers.tick(1);
  await setImmediate();
  assert.equal(calls, 2);
  t.mock.timers.tick(4999);
  await iam.auth.refreshSession();
  assert.equal(calls, 2);
  t.mock.timers.tick(1);
  await setImmediate();
  assert.equal(calls, 3);
  assert.equal((await iam.auth.getSession()).refresh_token, 'refresh-new');
  assert.equal(JSON.parse(await storage.getItem('iam.session')).access_token, 'access-new');
  assert.deepEqual(events, ['INITIAL_SESSION', 'TOKEN_REFRESHED']);
});

test('automatic refresh of an expired restored session survives a network failure', async t => {
  const { iam, events } = await setup(t, true, -1000);
  let calls = 0;
  t.mock.method(globalThis, 'fetch', async () => {
    if (++calls === 1) throw new TypeError('offline');
    return success();
  });
  t.mock.timers.tick(0);
  await setImmediate();
  assert.equal(calls, 1);
  assert.equal((await iam.auth.getSession()).refresh_token, 'refresh-old');
  t.mock.timers.tick(500);
  await setImmediate();
  assert.equal(calls, 2);
  assert.equal((await iam.auth.getSession()).refresh_token, 'refresh-new');
  assert.deepEqual(events, ['INITIAL_SESSION', 'TOKEN_REFRESHED']);
});

test('explicit refresh waits for initial session loading', async t => {
  const storage = new MemoryStorage();
  await storage.setItem('iam.session', JSON.stringify({
    ...tokens('old'), expires_at: Date.now() + 600000, user: { id: 'user' },
  }));
  const fetch = t.mock.method(globalThis, 'fetch', async () => success());
  const iam = createIamClient({ baseUrl: 'https://iam.example', clientId: 'project', storage, autoRefresh: false, multiTab: false });
  assert.equal((await iam.auth.refreshSession()).refresh_token, 'refresh-new');
  assert.equal(fetch.mock.callCount(), 1);
});

test('exponential retry delay is capped and resets after a successful refresh', async t => {
  const { iam } = await setup(t, true);
  let recover = false;
  const fetch = t.mock.method(globalThis, 'fetch', async () => recover ? success() : json({}, 503));
  await iam.auth.refreshSession();
  let calls = 1;
  for (const delay of [500, 1000, 2000, 4000, 8000, 15000, 15000]) {
    t.mock.timers.tick(delay - 1);
    await setImmediate();
    assert.equal(fetch.mock.callCount(), calls);
    t.mock.timers.tick(1);
    await setImmediate();
    assert.equal(fetch.mock.callCount(), ++calls);
  }
  recover = true;
  t.mock.timers.tick(15000);
  await setImmediate();
  recover = false;
  await iam.auth.refreshSession();
  calls = fetch.mock.callCount();
  t.mock.timers.tick(500);
  await setImmediate();
  assert.equal(fetch.mock.callCount(), calls + 1);
});

test('Retry-After accepts an HTTP date', async t => {
  const { iam } = await setup(t, true);
  const until = new Date(Date.now() + 10000).toUTCString();
  const fetch = t.mock.method(globalThis, 'fetch', async () => json({}, 503, { 'Retry-After': until }));
  await iam.auth.refreshSession();
  t.mock.timers.tick(9999);
  await setImmediate();
  assert.equal(fetch.mock.callCount(), 1);
  t.mock.timers.tick(1);
  await setImmediate();
  assert.equal(fetch.mock.callCount(), 2);
});

test('concurrent refresh calls share one request', async t => {
  const { iam } = await setup(t);
  let resolve;
  const fetch = t.mock.method(globalThis, 'fetch', () => new Promise(r => { resolve = r; }));
  const pending = [iam.auth.refreshSession(), iam.auth.refreshSession(), iam.auth.refreshSession()];
  await setImmediate();
  assert.equal(fetch.mock.callCount(), 1);
  resolve(success());
  for (const session of await Promise.all(pending)) assert.equal(session.access_token, 'access-new');
});

for (const refuse of [false, true]) {
  test(`a stale ${refuse ? 'refusal' : 'success'} cannot change a new login`, async t => {
    const { iam, events } = await setup(t);
    let resolve;
    t.mock.method(globalThis, 'fetch', request => request.url.endsWith('/token/refresh')
      ? new Promise(r => { resolve = r; })
      : Promise.resolve(json({ result_type: 'authenticated', session: tokens('login'), user: { id: 'other-user' } })));
    const pending = iam.auth.refreshSession();
    await setImmediate();
    await iam.auth.signInWithPassword({ email: 'user@example.com', password: 'password' });
    resolve(refuse ? json({ error: 'invalid_grant' }, 400) : success());
    await pending;
    assert.equal((await iam.auth.getSession()).access_token, 'access-login');
    assert.deepEqual(events, ['INITIAL_SESSION', 'SIGNED_IN']);
  });
}

test('a pending refresh cannot restore a signed-out session', async t => {
  const { iam } = await setup(t, true);
  let resolve;
  t.mock.method(globalThis, 'fetch', request => request.url.endsWith('/token/refresh')
    ? new Promise(r => { resolve = r; }) : Promise.resolve(json({ ok: true })));
  const pending = iam.auth.refreshSession();
  await setImmediate();
  await iam.auth.signOut();
  resolve(success());
  assert.equal(await pending, null);
  assert.equal(await iam.auth.getSession(), null);
});

test('401 interception retains session on refresh outage and shares the backoff', async t => {
  const { iam, events } = await setup(t);
  let refreshes = 0;
  t.mock.method(globalThis, 'fetch', async request => {
    if (request.url.endsWith('/token/refresh')) { refreshes++; return json({}, 503); }
    return json({ error: { code: 'invalid_token' } }, 401);
  });
  for (let i = 0; i < 3; i++) {
    const result = await iam.client.get({ url: '/v1/users/me', security: [{ type: 'http', scheme: 'bearer' }] });
    assert.equal(result.response.status, 401);
  }
  assert.equal(refreshes, 1);
  assert.equal((await iam.auth.getSession()).access_token, 'access-old');
  assert.deepEqual(events, ['INITIAL_SESSION']);
});

test('401 interception retries the API call once with refreshed credentials', async t => {
  const { iam } = await setup(t);
  let calls = 0;
  t.mock.method(globalThis, 'fetch', async request => {
    calls++;
    if (request.url.endsWith('/token/refresh')) return success();
    if (request.headers.get('X-IAM-Retry') === '1') {
      assert.equal(request.headers.get('Authorization'), 'Bearer access-new');
      return json({ user: { id: 'user' } });
    }
    return json({}, 401);
  });
  const result = await iam.client.get({ url: '/v1/users/me' });
  assert.equal(result.response.status, 200);
  assert.equal(calls, 3);
});
