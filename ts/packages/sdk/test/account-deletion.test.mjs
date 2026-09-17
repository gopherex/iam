import assert from 'node:assert/strict';
import { test, afterEach } from 'node:test';
import { createIamClient, MemoryStorage } from '../dist/index.js';

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });
const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
async function signedIn() {
  const storage = new MemoryStorage();
  await storage.setItem('iam.session', JSON.stringify({ access_token: 'access-token', refresh_token: 'refresh-token', token_type: 'Bearer', expires_at: Date.now() + 600000, user: { id: 'user' } }));
  const iam = createIamClient({ baseUrl: 'https://iam.example', clientId: 'project', environment: 'staging', storage, autoRefresh: false, multiTab: false });
  await iam.auth.ready();
  return iam;
}

test('deletion request, status and cancellation retain the ordinary session', async () => {
  const iam = await signedIn();
  const pending = { ok: true, status: 'pending', request_id: 'request', grace_days: 7, delete_at: '2026-09-24T12:00:00Z' };
  let calls = 0;
  globalThis.fetch = async request => {
    assert.equal(request.headers.get('Authorization'), 'Bearer access-token');
    assert.equal(request.headers.get('X-Environment'), 'staging');
    calls++;
    if (request.method === 'GET') {
      assert.ok(request.url.endsWith('/v1/users/me/deletion'));
      return json(pending);
    }
    assert.deepEqual(await request.json(), { password: 'current-password', proof_token: 'sft_authorized' });
    if (request.method === 'DELETE') {
      assert.ok(request.url.endsWith('/v1/users/me'));
      return json(pending);
    }
    assert.ok(request.url.endsWith('/v1/users/me/deletion/cancel'));
    return json({ ...pending, status: 'cancelled' });
  };
  const input = { password: 'current-password', proofToken: 'sft_authorized' };
  assert.equal((await iam.account.deletion.request(input)).data.delete_at, pending.delete_at);
  assert.equal((await iam.account.deletion.get()).data.request_id, 'request');
  assert.equal((await iam.account.deletion.cancel(input)).data.status, 'cancelled');
  assert.equal((await iam.auth.getSession()).access_token, 'access-token');
  assert.equal(calls, 3);
});

test('proof-required errors expose the isolated capability without signing out', async () => {
  const iam = await signedIn();
  globalThis.fetch = async () => json({ error: { code: 'step_up_required', message: 'Confirm identity', details: { security_flow_token: 'sft_proof', purpose: 'deletion_request' } } }, 403);
  const result = await iam.account.deleteAccount({ password: 'current-password' });
  assert.equal(result.data, null);
  assert.equal(result.error.code, 'step_up_required');
  assert.equal(result.error.details.security_flow_token, 'sft_proof');
  assert.equal((await iam.auth.getSession()).refresh_token, 'refresh-token');
});
