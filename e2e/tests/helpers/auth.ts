import { request, type APIRequestContext, type Browser, type BrowserContext } from '@playwright/test';

export const BASE_URL = process.env.SIGAP_BASE_URL ?? 'https://localhost';
export const ADMIN_USER = process.env.SIGAP_USER ?? 'admin';
export const ADMIN_PASS = process.env.SIGAP_PASS ?? 'AdminPass123!';

export interface LoggedInUser {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: string;
  must_change_password: boolean;
}

export interface AuthenticatedApiSession {
  apiCtx: APIRequestContext;
  csrfToken: string;
  user: LoggedInUser;
}

const TEMP_RECOVERY_PASS = 'TempOpsPass!2026';

function isForcedPasswordClearAllowed(): boolean {
  return process.env.CI === 'true' || process.env.CI === '1' || process.env.E2E_MUTATE === '1';
}

/**
 * Membuka sesi API terautentikasi ke backend SIGAP melalui Nginx TLS.
 * Jika pengguna masih berstatus `must_change_password` pada DB segar CI (atau saat
 * E2E_MUTATE=1), sandi disiklus sementara lalu dikembalikan ke nilai semula
 * agar endpoint operasional tidak tertahan 403.
 */
export async function createAuthenticatedApiSession(
  username = ADMIN_USER,
  password = ADMIN_PASS
): Promise<AuthenticatedApiSession> {
  const apiCtx = await request.newContext({
    baseURL: BASE_URL,
    ignoreHTTPSErrors: true,
  });

  const loginRes = await apiCtx.post('/api/auth/login', {
    data: { username, password },
  });
  if (loginRes.status() !== 200) {
    const bodyText = await loginRes.text().catch(() => '');
    await apiCtx.dispose();
    throw new Error(`Login API gagal untuk ${username}: status=${loginRes.status()} body=${bodyText}`);
  }

  const loginJson = (await loginRes.json()) as {
    data?: { csrf_token?: string; user?: LoggedInUser };
  };
  let csrfToken = loginJson.data?.csrf_token ?? '';
  let user = loginJson.data?.user;
  if (!user) {
    await apiCtx.dispose();
    throw new Error('Respons login tidak menyertakan objek user.');
  }

  if (user.must_change_password) {
    if (!isForcedPasswordClearAllowed()) {
      await apiCtx.dispose();
      throw new Error(
        `MUST_CHANGE_PASSWORD_ACTIVE: akun ${username} masih berstatus must_change_password=true (set CI=true atau E2E_MUTATE=1 untuk menyiklus otomatis).`
      );
    }

    const tempPassword = password === TEMP_RECOVERY_PASS ? 'TempOpsPass!2027' : TEMP_RECOVERY_PASS;

    const firstChange = await apiCtx.post('/api/auth/change-password', {
      headers: { 'X-CSRF-Token': csrfToken },
      data: { new_password: tempPassword },
    });
    if (firstChange.status() !== 200) {
      const text = await firstChange.text().catch(() => '');
      await apiCtx.dispose();
      throw new Error(`Gagal menyelesaikan forced password change pertama: ${firstChange.status()} ${text}`);
    }
    const firstJson = (await firstChange.json()) as {
      data?: { csrf_token?: string; user?: LoggedInUser };
    };
    csrfToken = firstJson.data?.csrf_token ?? csrfToken;

    const restoreChange = await apiCtx.post('/api/auth/change-password', {
      headers: { 'X-CSRF-Token': csrfToken },
      data: { old_password: tempPassword, new_password: password },
    });
    if (restoreChange.status() !== 200) {
      const text = await restoreChange.text().catch(() => '');
      await apiCtx.dispose();
      throw new Error(`Gagal mengembalikan sandi awal setelah forced change: ${restoreChange.status()} ${text}`);
    }
    const restoreJson = (await restoreChange.json()) as {
      data?: { csrf_token?: string; user?: LoggedInUser };
    };
    csrfToken = restoreJson.data?.csrf_token ?? csrfToken;
    if (restoreJson.data?.user) {
      user = restoreJson.data.user;
    }
  }

  return { apiCtx, csrfToken, user };
}

/**
 * Membuat BrowserContext baru dengan cookie sesi & CSRF dari API session
 * tanpa mengonsumsi kuota AuthLimiter `/api/auth/login`.
 */
export async function newAuthenticatedBrowserContext(
  browser: Browser,
  apiCtx: APIRequestContext
): Promise<BrowserContext> {
  const storageState = await apiCtx.storageState();
  return browser.newContext({
    baseURL: BASE_URL,
    ignoreHTTPSErrors: true,
    storageState,
  });
}
