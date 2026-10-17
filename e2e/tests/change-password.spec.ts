import { test, expect, request } from '@playwright/test';
import * as fs from 'fs';

const BASE_URL = 'https://localhost';
const USERNAME = process.env.SIGAP_USER ?? 'admin';
const PASSWORD = process.env.SIGAP_PASS ?? 'AdminPass123!';

type Page = import('@playwright/test').Page;
type TestInfo = import('@playwright/test').TestInfo;

// Tempelkan listener + login + tunggu /change-password, lalu kumpulkan diagnostik.
async function loginAndReachChangePassword(page: Page) {
  const apiResponses: string[] = [];
  const consoleErrors: string[] = [];
  const pageErrors: string[] = [];

  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(`[error] ${msg.text()}`);
  });
  page.on('pageerror', (err) => pageErrors.push(String(err)));
  page.on('response', (res) => {
    if (res.url().includes('/api/')) {
      apiResponses.push(`${res.status()} ${res.request().method()} ${res.url()}`);
    }
  });

  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.fill('#username', USERNAME);
  await page.fill('#password', PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/change-password', { timeout: 25_000 });
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(1500);

  return { apiResponses, consoleErrors, pageErrors };
}

test('halaman /change-password tidak loop & mode forced tampil', async ({ page }, testInfo: TestInfo) => {
  const { apiResponses, consoleErrors, pageErrors } = await loginAndReachChangePassword(page);

  const meCalls = apiResponses.filter((r) => r.includes('/api/auth/me'));
  const bannerVisible = await page
    .getByText('You must change your password before continuing')
    .isVisible()
    .catch(() => false);
  const exitButtonVisible = await page
    .getByRole('button', { name: /Exit \/ Back to Login/i })
    .isVisible()
    .catch(() => false);
  const finalUrl = page.url();

  const shot = testInfo.outputPath('change-password.png');
  await page.screenshot({ path: shot, fullPage: true });

  const diagnostics = {
    finalUrl,
    meCalls: meCalls.length,
    meCallList: meCalls,
    bannerVisible,
    exitButtonVisible,
    consoleErrors,
    pageErrors,
    apiResponses,
    screenshot: shot,
  };
  console.log('DIAGNOSTICS_JSON=' + JSON.stringify(diagnostics, null, 2));
  fs.writeFileSync(testInfo.outputPath('diagnostics.json'), JSON.stringify(diagnostics, null, 2));

  // Anti-loop: /auth/me hanya boleh dipanggil segelintir kali, bukan ratusan.
  expect(meCalls.length, 'Terlalu banyak panggilan /auth/me (indikasi loop reload)').toBeLessThanOrEqual(5);
  expect(bannerVisible, 'Banner "You must change your password..." tidak tampil').toBeTruthy();
  expect(exitButtonVisible, 'Tombol Exit / Back to Login tidak tampil').toBeTruthy();
  expect(finalUrl).toContain('/change-password');
});

// Alur mutasi (mengubah sandi admin lalu mengembalikannya). Hanya berjalan bila
// E2E_MUTATE=1 agar tidak merusak state admin default saat suite dijalankan biasa.
test('alur ganti sandi berfungsi normal (forced & non-forced)', async ({ page }, testInfo: TestInfo) => {
  test.skip(process.env.E2E_MUTATE !== '1', 'Set E2E_MUTATE=1 untuk menjalankan alur mutasi sandi.');

  const TEMP_PASSWORD = 'TempPass123!';

  // 1) Login -> dipaksa ke /change-password.
  await loginAndReachChangePassword(page);
  await expect(page).toHaveURL(/\/change-password$/);

  // 2) Ganti sandi (mode forced) -> kembali ke dasbor.
  await page.fill('#new_password', TEMP_PASSWORD);
  await page.fill('#confirm_password', TEMP_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard', { timeout: 25_000 });

  // 3) Buktikan sandi baru benar-benar berlaku (login lewat API terpisah).
  const apiCtx = await request.newContext({ baseURL: BASE_URL, ignoreHTTPSErrors: true });
  const loginWithTemp = await apiCtx.post('/api/auth/login', {
    data: { username: USERNAME, password: TEMP_PASSWORD },
  });
  expect(loginWithTemp.status(), 'Login dengan sandi baru gagal').toBe(200);

  // 4) Mode non-forced: buka /change-password dari sesi berjalan, kembalikan sandi awal.
  await page.goto('/change-password', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1000);
  await expect(page).toHaveURL(/\/change-password$/); // guard salah sudah dihapus -> tidak diblok ke /dashboard
  await page.fill('#new_password', PASSWORD);
  await page.fill('#confirm_password', PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/profile**', { timeout: 25_000 });

  // 5) Buktikan sandi awal sudah kembali.
  const loginBack = await apiCtx.post('/api/auth/login', {
    data: { username: USERNAME, password: PASSWORD },
  });
  expect(loginBack.status(), 'Sandi awal gagal dikembalikan').toBe(200);
  await apiCtx.dispose();

  const shot = testInfo.outputPath('change-password-flow.png');
  await page.screenshot({ path: shot, fullPage: true });
});
