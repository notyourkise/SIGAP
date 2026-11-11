import { test, expect, request, type APIRequestContext } from '@playwright/test';
import {
  ADMIN_PASS,
  ADMIN_USER,
  BASE_URL,
  createAuthenticatedApiSession,
  newAuthenticatedBrowserContext,
} from './helpers/auth';

test.describe('SIGAP Ops Regression Suite (#155)', () => {
  let adminApi: APIRequestContext;
  let adminCsrf = '';
  let setupSkipReason = '';

  test.beforeAll(async () => {
    try {
      const session = await createAuthenticatedApiSession(ADMIN_USER, ADMIN_PASS);
      adminApi = session.apiCtx;
      adminCsrf = session.csrfToken;
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg.includes('MUST_CHANGE_PASSWORD_ACTIVE') && !process.env.CI && process.env.E2E_MUTATE !== '1') {
        setupSkipReason = msg;
        return;
      }
      throw err;
    }
  });

  test.afterAll(async () => {
    if (adminApi) {
      await adminApi.dispose();
    }
  });

  test.beforeEach(() => {
    if (setupSkipReason) {
      test.skip(true, setupSkipReason);
    }
  });

  test('smoke test halaman /cctv/logs (filter URL, reload, pencarian, WITA, nol error)', async ({ browser }) => {
    const context = await newAuthenticatedBrowserContext(browser, adminApi);
    const page = await context.newPage();

    const pageErrors: string[] = [];
    const serverErrors: string[] = [];
    page.on('pageerror', (err) => pageErrors.push(String(err)));
    page.on('response', (res) => {
      if (res.status() >= 500) {
        serverErrors.push(`${res.status()} ${res.request().method()} ${res.url()}`);
      }
    });

    try {
      await page.goto('/cctv/logs', { waitUntil: 'domcontentloaded' });
      await expect(page.getByRole('heading', { name: 'Log Status CCTV' })).toBeVisible();

      await expect(page.getByText('Total hasil filter')).toBeVisible();
      await expect(page.getByText('Event online hari ini')).toBeVisible();
      await expect(page.getByText('Event offline hari ini')).toBeVisible();
      await expect(page.getByText(/zona waktu\s+WITA/i)).toBeVisible();

      for (const label of ['Semua Event', 'Online', 'Offline', 'Siaga', 'Tidak Diketahui']) {
        await expect(page.getByRole('button', { name: label, exact: true })).toBeVisible();
      }

      // Filter Offline -> tersinkron ke URL
      const offlineBtn = page.getByRole('button', { name: 'Offline', exact: true });
      await offlineBtn.click();
      await expect(page).toHaveURL(/event_type=offline/);
      await expect(offlineBtn).toHaveAttribute('aria-pressed', 'true');

      // Filter Tidak Diketahui -> persist setelah reload
      const unknownBtn = page.getByRole('button', { name: 'Tidak Diketahui', exact: true });
      await unknownBtn.click();
      await expect(page).toHaveURL(/event_type=unknown/);
      await page.reload({ waitUntil: 'domcontentloaded' });
      await expect(page).toHaveURL(/event_type=unknown/);
      await expect(page.getByRole('button', { name: 'Tidak Diketahui', exact: true })).toHaveAttribute(
        'aria-pressed',
        'true'
      );

      // Pencarian ter-debounce memicu empty state filter
      const searchInput = page.getByLabel('Cari log berdasarkan kode atau nama CCTV');
      await searchInput.fill('OPS-NONE-99999');
      await expect(page).toHaveURL(/search=OPS-NONE-99999/);
      await expect(page.getByText('Tidak ada hasil filter')).toBeVisible();

      // Reset filter membersihkan URL
      await page.getByRole('button', { name: 'Hapus filter' }).click();
      await expect(page).toHaveURL(/\/cctv\/logs$/);

      // Segarkan memuat ulang endpoint /api/cctv-logs tanpa error
      const refreshResponsePromise = page.waitForResponse(
        (res) => res.url().includes('/api/cctv-logs') && res.request().method() === 'GET'
      );
      await page.getByRole('button', { name: 'Segarkan', exact: true }).click();
      const refreshRes = await refreshResponsePromise;
      expect(refreshRes.status()).toBe(200);

      expect(pageErrors, 'Tidak boleh ada runtime pageerror').toEqual([]);
      expect(serverErrors, 'Tidak boleh ada HTTP >= 500').toEqual([]);
    } finally {
      await context.close();
    }
  });

  test('timezone Asia/Makassar, retensi log, dan diagnostik Metro-E pada API', async () => {
    const logsRes = await adminApi.get('/api/cctv-logs?page=1&limit=5');
    expect(logsRes.status()).toBe(200);
    const logsBody = (await logsRes.json()) as {
      status: string;
      data: {
        rows: unknown[];
        summary: {
          timezone: string;
          today_online_events: number;
          today_offline_events: number;
          total_events: number;
        };
      };
    };
    expect(logsBody.status).toBe('success');
    expect(Array.isArray(logsBody.data.rows)).toBeTruthy();
    expect(logsBody.data.summary.timezone).toBe('Asia/Makassar');
    expect(logsBody.data.summary.today_online_events).toBeGreaterThanOrEqual(0);
    expect(logsBody.data.summary.today_offline_events).toBeGreaterThanOrEqual(0);

    // Filter event_type tidak valid wajib ditolak fail-fast 400
    const invalidFilterRes = await adminApi.get('/api/cctv-logs?event_type=invalid_e2e');
    expect(invalidFilterRes.status()).toBe(400);

    // Diagnostik streaming admin menyertakan mode Metro-E, retensi log, & timezone
    const diagRes = await adminApi.get('/api/streams/diagnostics');
    expect(diagRes.status()).toBe(200);
    const diagBody = (await diagRes.json()) as {
      status: string;
      data: {
        metro_e: {
          configured_direct_probe: boolean;
          effective_direct_probe: boolean;
          passive_fallback_active: boolean;
          mode: string;
        };
        summary: {
          false_offline_protected: boolean;
        };
        observability: {
          metrics_endpoint: string;
          stream_log_retention_days: number;
          timezone: string;
        };
      };
    };
    expect(diagBody.status).toBe('success');
    expect(['direct_probe', 'passive_fallback', 'passive_configured']).toContain(diagBody.data.metro_e.mode);
    expect(diagBody.data.summary.false_offline_protected).toBe(!diagBody.data.metro_e.effective_direct_probe);
    expect(diagBody.data.observability.metrics_endpoint).toBe('/api/metrics');
    expect(diagBody.data.observability.timezone).toBe('Asia/Makassar');

    const expectedRetention = Number(process.env.CCTV_STREAM_LOG_RETENTION_DAYS ?? '90');
    expect(diagBody.data.observability.stream_log_retention_days).toBe(expectedRetention);
  });

  test('isolasi multi-site antar teknisi, penolakan cross-site 403, dan SiteSwitcher UI', async ({ browser }) => {
    const suffix = `${Date.now()}`.slice(-6);
    const siteACode = `OPSA${suffix}`;
    const siteBCode = `OPSB${suffix}`;
    const techUsername = `opstech_${suffix}`;
    const techPass = 'OpsTechPass!2026';

    let siteAId = '';
    let siteBId = '';
    let techUserId = '';
    let techApi: APIRequestContext | null = null;
    const browserCtx = await newAuthenticatedBrowserContext(browser, adminApi);

    try {
      // 1. Buat dua site sementara + satu akun teknisi sementara
      const createARes = await adminApi.post('/api/sites', {
        headers: { 'X-CSRF-Token': adminCsrf },
        data: {
          nama: `Ops Regression Site A ${suffix}`,
          kode: siteACode,
          alamat: 'Kawasan Pengujian A',
          is_default: false,
        },
      });
      expect(createARes.status()).toBe(201);
      siteAId = ((await createARes.json()) as { data: { id: string } }).data.id;

      const createBRes = await adminApi.post('/api/sites', {
        headers: { 'X-CSRF-Token': adminCsrf },
        data: {
          nama: `Ops Regression Site B ${suffix}`,
          kode: siteBCode,
          alamat: 'Kawasan Pengujian B',
          is_default: false,
        },
      });
      expect(createBRes.status()).toBe(201);
      siteBId = ((await createBRes.json()) as { data: { id: string } }).data.id;

      const createTechRes = await adminApi.post('/api/auth/users', {
        headers: { 'X-CSRF-Token': adminCsrf },
        data: {
          username: techUsername,
          password: techPass,
          full_name: `Teknisi Ops ${suffix}`,
          email: `opstech_${suffix}@sigap.local`,
          role: 'teknisi',
        },
      });
      expect(createTechRes.status()).toBe(201);
      techUserId = ((await createTechRes.json()) as { data: { id: string } }).data.id;

      // 2. Login teknisi (CreateUser menyetel must_change_password=true -> selesaikan forced change)
      techApi = await request.newContext({ baseURL: BASE_URL, ignoreHTTPSErrors: true });
      const techLoginRes = await techApi.post('/api/auth/login', {
        data: { username: techUsername, password: techPass },
      });
      expect(techLoginRes.status()).toBe(200);
      const techCsrf = ((await techLoginRes.json()) as { data: { csrf_token: string } }).data.csrf_token;
      const techChangeRes = await techApi.post('/api/auth/change-password', {
        headers: { 'X-CSRF-Token': techCsrf },
        data: { new_password: `${techPass}9!` },
      });
      expect(techChangeRes.status()).toBe(200);

      // 3. Sebelum ditugaskan ke site mana pun -> fail closed 403
      const unassignedLogsRes = await techApi.get('/api/cctv-logs');
      expect(unassignedLogsRes.status()).toBe(403);

      // 4. Tugaskan teknisi HANYA ke Site A
      const assignRes = await adminApi.post(`/api/sites/${siteAId}/assign-users`, {
        headers: { 'X-CSRF-Token': adminCsrf },
        data: { user_ids: [techUserId] },
      });
      expect(assignRes.status()).toBe(200);

      // Teknisi boleh mengakses Site A, tetapi DILARANG mengakses Site B atau /streams/diagnostics
      const ownSiteDefaultRes = await techApi.get('/api/cctv-logs');
      expect(ownSiteDefaultRes.status()).toBe(200);

      const ownSiteHeaderRes = await techApi.get('/api/cctv-logs', {
        headers: { 'X-Site-ID': siteAId },
      });
      expect(ownSiteHeaderRes.status()).toBe(200);

      const crossSiteRes = await techApi.get('/api/cctv-logs', {
        headers: { 'X-Site-ID': siteBId },
      });
      expect(crossSiteRes.status()).toBe(403);

      const techDiagRes = await techApi.get('/api/streams/diagnostics');
      expect(techDiagRes.status()).toBe(403);

      const techSitesRes = await techApi.get('/api/sites/all');
      expect(techSitesRes.status()).toBe(200);
      const techSites = ((await techSitesRes.json()) as { data: Array<{ id: string }> }).data;
      expect(techSites.some((s) => s.id === siteAId)).toBeTruthy();
      expect(techSites.some((s) => s.id === siteBId)).toBeFalsy();

      // 5. Verifikasi SiteSwitcher UI oleh Admin pada halaman /cctv/logs
      const page = await browserCtx.newPage();
      await page.goto('/cctv/logs', { waitUntil: 'domcontentloaded' });
      await expect(page.getByRole('heading', { name: 'Log Status CCTV' })).toBeVisible();

      const switcherButton = page.locator('button[aria-label="Ganti site / kawasan aktif"]:visible').last();
      await expect(switcherButton).toBeVisible();
      await expect(switcherButton).not.toHaveText(/Memuat/);
      await switcherButton.click();

      await Promise.all([
        page.waitForEvent('load'),
        page.getByRole('menuitem', { name: new RegExp(siteACode) }).click(),
      ]);
      await expect
        .poll(async () => (await browserCtx.cookies()).find((c) => c.name === 'sigap_site_id')?.value)
        .toBe(siteAId);
      await expect(page.getByRole('heading', { name: 'Log Status CCTV' })).toBeVisible();

      // Kembalikan ke "Semua Site" agar cookie bersih
      const switcherAfter = page.locator('button[aria-label="Ganti site / kawasan aktif"]:visible').last();
      await expect(switcherAfter).not.toHaveText(/Memuat/);
      await switcherAfter.click();
      await Promise.all([
        page.waitForEvent('load'),
        page.getByRole('menuitem', { name: /Semua Site/i }).click(),
      ]);
      await expect
        .poll(async () => (await browserCtx.cookies()).find((c) => c.name === 'sigap_site_id')?.value ?? '')
        .toBe('');
    } finally {
      if (techApi) {
        await techApi.dispose();
      }
      await browserCtx.close();

      // Cleanup entitas uji sementara
      if (techUserId) {
        await adminApi.delete(`/api/users/${techUserId}`, {
          headers: { 'X-CSRF-Token': adminCsrf },
        });
      }
      if (siteAId) {
        await adminApi.delete(`/api/sites/${siteAId}`, {
          headers: { 'X-CSRF-Token': adminCsrf },
        });
      }
      if (siteBId) {
        await adminApi.delete(`/api/sites/${siteBId}`, {
          headers: { 'X-CSRF-Token': adminCsrf },
        });
      }
    }
  });

  test('perlindungan proxy spoofing (X-Forwarded-For / X-Real-IP palsu ditolak pada audit log)', async () => {
    const spoofedIP = '198.51.100.77';
    const probeUsername = `spoof_e2e_${Date.now()}`;

    const unauthApi = await request.newContext({ baseURL: BASE_URL, ignoreHTTPSErrors: true });
    try {
      const spoofRes = await unauthApi.post('/api/auth/login', {
        headers: {
          'X-Forwarded-For': spoofedIP,
          'X-Real-IP': spoofedIP,
        },
        data: {
          username: probeUsername,
          password: 'WrongPassword123!',
        },
      });
      // Hanya mengirim 1 request (tidak memicu kuota 5/menit AuthLimiter)
      expect(spoofRes.status()).toBe(401);
    } finally {
      await unauthApi.dispose();
    }

    // Tunggu goroutine async AuditLog menulis baris LOGIN_FAILED dan pastikan IP palsu tidak tercatat
    await expect
      .poll(
        async () => {
          const auditRes = await adminApi.get(
            `/api/audit-logs?module=AUTH&search=${encodeURIComponent(probeUsername)}`
          );
          if (auditRes.status() !== 200) return null;
          const body = (await auditRes.json()) as {
            data?: { rows?: Array<{ action: string; ip_address: string; description: string }> };
          };
          const row = body.data?.rows?.find((r) => r.description.includes(probeUsername));
          return row ?? null;
        },
        { timeout: 10_000, intervals: [300, 500, 1000] }
      )
      .not.toBeNull();

    const auditRes = await adminApi.get(
      `/api/audit-logs?module=AUTH&search=${encodeURIComponent(probeUsername)}`
    );
    const auditBody = (await auditRes.json()) as {
      data: { rows: Array<{ action: string; ip_address: string; description: string }> };
    };
    const recorded = auditBody.data.rows.find((r) => r.description.includes(probeUsername));
    expect(recorded).toBeDefined();
    expect(recorded?.action).toBe('LOGIN_FAILED');
    expect(recorded?.ip_address).toBeTruthy();
    expect(recorded?.ip_address).not.toBe(spoofedIP);
  });
});
