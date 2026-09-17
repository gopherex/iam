import assert from 'node:assert/strict';
const { chromium } = await import(process.env.IAM_PLAYWRIGHT_MODULE);
const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1280, height: 960 } });
page.setDefaultTimeout(10000);
const errors = [];
page.on('pageerror', error => errors.push(error.message));
try {
  if (process.env.IAM_BROWSER_FLOW_TOKEN) {
    const origin = new URL(process.env.IAM_BROWSER_URL).origin;
    const key = `iam.security.flow:${encodeURIComponent(origin)}:${encodeURIComponent(process.env.IAM_BROWSER_PROJECT)}:live`;
    await page.addInitScript(({ key, token }) => sessionStorage.setItem(key, token), { key, token: process.env.IAM_BROWSER_FLOW_TOKEN });
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('WebAuthn.enable');
    await cdp.send('WebAuthn.addVirtualAuthenticator', { options: { protocol: 'ctap2', transport: 'internal', hasResidentKey: true, hasUserVerification: true, isUserVerified: true, automaticPresenceSimulation: true } });
    await page.goto(process.env.IAM_BROWSER_URL);
    await page.getByRole('button', { name: 'Создать новый passkey', exact: true }).click();
    await page.getByRole('status').filter({ hasText: 'Сценарий завершён' }).waitFor();
    assert.deepEqual(errors, []);
  } else {
  await page.goto(process.env.IAM_BROWSER_URL);
  await page.locator('#flow-email').fill(process.env.IAM_BROWSER_EMAIL);
  await page.locator('#flow-password').fill('Sup3rStr0ng!Pass');
  await page.getByRole('button', { name: 'Sign in', exact: true }).last().click();
  await page.getByRole('button', { name: 'Обновить активность', exact: true }).click();
  await page.getByRole('button', { name: 'Проверить', exact: true }).first().click();
  await page.getByLabel('Код подтверждения', { exact: true }).waitFor();
  const code = await (await page.request.get(process.env.IAM_BROWSER_CODE_URL)).text();
  await page.getByLabel('Код подтверждения', { exact: true }).fill(code);
  await page.getByRole('button', { name: 'Подтвердить', exact: true }).click();
  await page.getByRole('button', { name: 'Это не я', exact: true }).waitFor();
  await page.reload();
  const checkbox = page.getByRole('checkbox', { name: 'Завершить все сессии и снять доверие со всех устройств' });
  await checkbox.waitFor();
  assert.equal(await checkbox.isChecked(), false);
  if (process.env.IAM_BROWSER_ALL === 'true') await checkbox.check();
  await page.getByRole('button', { name: 'Это не я', exact: true }).click();
  await page.getByRole('status').filter({ hasText: 'Восстановление доступа не останавливает таймер' }).waitFor();
  await page.getByRole('button', { name: 'Отменить удаление аккаунта', exact: true }).click();
  await page.getByRole('status').filter({ hasText: 'Удаление аккаунта отменено.' }).waitFor();
  await page.getByLabel('Новый пароль', { exact: true }).fill('Restored!Secure123');
  await page.getByRole('button', { name: 'Сохранить новый пароль', exact: true }).click();
  await page.getByRole('status').filter({ hasText: 'Сценарий завершён' }).waitFor();
  assert.equal(await page.getByRole('button', { name: 'Обновить активность', exact: true }).count(), 0);
  assert.deepEqual(errors, []);
  }
  await page.screenshot({ path: `/tmp/iam-security-browser/result-${process.env.IAM_BROWSER_ALL}.png`, fullPage: true });
} catch (error) {
  console.error(await page.locator('body').innerText());
  throw error;
} finally { await browser.close(); }
