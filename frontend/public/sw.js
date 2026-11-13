/* ==========================================
   SIGAP — Service Worker (Task 8.1 PWA + Task 8.3 Push)
   ==========================================
   Scope: installability + offline shell + notifikasi push browser.
   Strategi cache (HARDENING KEAMANAN):
     - Shell statis (CSS, JS chunk, ikon/font): cache-first (di-hash Next.js).
     - Aset statis  : cache-first dengan pembaruan latar belakang.
     - Navigasi HTML: network-only; bila offline tampilkan halaman offline
                      generik. HTML terautentikasi TIDAK pernah di-cache.
     - API calls    : SELALU network-only. Respons API terautentikasi TIDAK
                      di-cache (mencegah data pengguna bocor lintas sesi/user
                      pada perangkat bersama). Bila offline, kembalikan JSON 503.
     - Request `no-store` & gambar QR: network-only.
   Push (Task 8.3): menangani event `push` (Web Push/VAPID) dan menampilkan
   notifikasi OS via showNotification, serta membuka halaman terkait saat diklik.
   BUKAN full offline (tanpa IndexedDB) — data tetap butuh jaringan/API.
   ========================================== */

// Nama cache versi-mu — naikkan saat konten shell berubah agar cache lama dibuang.
const CACHE_VERSION = 'sigap-shell-v5';
const SHELL_CACHE = CACHE_VERSION;
const IS_DEVELOPMENT = new URL(self.location.href).searchParams.has('dev');

// Aset inti shell yang di-pre-cache saat install (hanya aset statis publik;
// TIDAK ada Halaman HTML terautentikasi agar tidak bocor ke perangkat bersama).
const PRECACHE_URLS = [
  '/manifest.json',
  '/icons/icon-192x192.png',
  '/icons/icon-512x512.png',
];

// Batas waktu (detik) sebelum permintaan jaringan dianggap gagal pada
// strategi network-first (mencegah layar kosong saat koneksi lambat).
const NETWORK_TIMEOUT = 5000;

self.addEventListener('install', (event) => {
  if (IS_DEVELOPMENT) {
    event.waitUntil(self.skipWaiting());
    return;
  }
  event.waitUntil(
    caches
      .open(SHELL_CACHE)
      .then((cache) => cache.addAll(PRECACHE_URLS))
      // Langsung aktif tanpa menunggu tab lama ditutup.
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  if (IS_DEVELOPMENT) {
    event.waitUntil(
      caches.keys().then((keys) =>
        Promise.all(keys.filter((key) => key.startsWith('sigap-shell-')).map((key) => caches.delete(key)))
      ).then(() => self.clients.claim())
    );
    return;
  }
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((key) => key !== SHELL_CACHE)
            .map((key) => caches.delete(key))
        )
      )
      // Segera menguasai halaman yang sudah terbuka.
      .then(() => self.clients.claim())
  );
});

// ---- Helper: apakah ini navigasi halaman (request HTML utama) ----
function isNavigationRequest(request) {
  return request.mode === 'navigate';
}

// ---- Helper: apakah ini panggilan API ----
function isApiRequest(url) {
  return url.pathname.startsWith('/api/');
}

// QR hasil rotasi tidak boleh memiliki fallback cache.
function isQRCodeImageRequest(url) {
  return /^\/api\/cctv\/[^/]+\/qrcode\/?$/.test(url.pathname);
}

// Hormati kebijakan cache dari caller maupun server untuk data operasional
// yang harus selalu segar.
function isRuntimeCacheAllowed(request, response) {
  if (request.cache === 'no-store') return false;
  const cacheControl = response.headers.get('Cache-Control') || '';
  return !cacheControl.toLowerCase().includes('no-store');
}

// ---- Helper: apakah ini aset statis yang aman di-cache-first ----
function isStaticAsset(url) {
  const path = url.pathname;
  // CSS/JS chunk (biasanya ber-hash), gambar, font, dll.
  return /\.(?:css|js|png|jpg|jpeg|webp|svg|ico|woff2?|ttf|eot|json)$/i.test(path);
}

// ---- Strategi: network-only dengan batas waktu (navigasi & API) ----
async function networkOnly(request) {
  const timeoutPromise = new Promise((_, reject) =>
    setTimeout(() => reject(new Error('timeout')), NETWORK_TIMEOUT)
  );
  const response = await Promise.race([fetch(request), timeoutPromise]);
  if (!response) {
    throw new Error('response kosong');
  }
  return response;
}

// ---- Strategi: cache-first (untuk aset statis shell saja) ----
async function cacheFirst(request) {
  const cache = await caches.open(SHELL_CACHE);
  const cached = await cache.match(request);
  if (cached) {
    // Pembaruan latar belakang tanpa memblokir respons cache.
    fetch(request)
      .then((resp) => {
        if (resp && resp.ok && isRuntimeCacheAllowed(request, resp)) {
          cache.put(request, resp);
        }
      })
      .catch(() => {});
    return cached;
  }
  const response = await fetch(request);
  if (response && response.ok && isRuntimeCacheAllowed(request, response)) {
    cache.put(request, response.clone());
  }
  return response;
}

// ---- Fallback JSON bila permintaan API gagal (tanpa cache) ----
function offlineApiResponse() {
  return new Response(
    JSON.stringify({
      status: 'error',
      message: 'Anda sedang offline. Periksa koneksi internet untuk memuat data terbaru.',
      errors: [],
    }),
    {
      status: 503,
      headers: { 'Content-Type': 'application/json; charset=utf-8' },
    }
  );
}

// ---- Fallback HTML statis generik bila navigasi gagal saat offline ----
function offlineShellResponse() {
  return new Response(
    '<!doctype html><html lang="id"><head><meta charset="utf-8">' +
      '<meta name="viewport" content="width=device-width, initial-scale=1">' +
      '<title>SIGAP — Offline</title></head>' +
      '<body style="font-family:system-ui,Segoe UI,Roboto,sans-serif;margin:0;' +
      'display:flex;min-height:100vh;align-items:center;justify-content:center;' +
      'background:#0f172a;color:#e2e8f0;text-align:center">' +
      '<div style="max-width:24rem;padding:2rem">' +
      '<h1 style="font-size:1.25rem;margin:0 0 .5rem">Tidak ada koneksi</h1>' +
      '<p style="margin:0;opacity:.8">Halaman tidak dapat dimuat saat offline. ' +
      'Sambungkan kembali ke jaringan lalu muat ulang.</p></div></body></html>',
    {
      status: 503,
      headers: { 'Content-Type': 'text/html; charset=utf-8' },
    }
  );
}

self.addEventListener('fetch', (event) => {
  // Development hanya membutuhkan registrasi untuk PushManager. Semua request
  // dilepas langsung ke browser agar chunk Turbopack tidak pernah tercache.
  if (IS_DEVELOPMENT) return;
  const { request } = event;
  const url = new URL(request.url);

  // Jangan tangani permintaan lintas-origin (hanya same-origin untuk shell & API).
  if (url.origin !== self.location.origin) return;

  // Permintaan bukan GET tidak di-cache (biarkan jaringan normal).
  if (request.method !== 'GET') return;

  // Request no-store dan gambar QR harus network-only.
  if (request.cache === 'no-store' || isQRCodeImageRequest(url)) {
    event.respondWith(fetch(request));
    return;
  }

  // API calls: SELALU network-only — respons terautentikasi tidak boleh di-cache.
  if (isApiRequest(url)) {
    event.respondWith(networkOnly(request).catch(() => offlineApiResponse()));
    return;
  }

  // Navigasi halaman (HTML): network-only; offline → halaman offline generik.
  if (isNavigationRequest(request)) {
    event.respondWith(networkOnly(request).catch(() => offlineShellResponse()));
    return;
  }

  // Aset statis shell: cache-first dengan pembaruan latar belakang.
  if (isStaticAsset(url)) {
    event.respondWith(cacheFirst(request));
    return;
  }

  // Lainnya: biarkan jaringan normal.
});

// ==========================================
// Notifikasi Push (Task 8.3)
// ==========================================
// Server mengirim payload JSON (data.push) berisi { title, body,
// notification_type, reference_id, reference_type }.

self.addEventListener('push', (event) => {
  let title = 'SIGAP';
  let options = {
    body: 'Anda memiliki pembaruan dari SIGAP.',
    icon: '/icons/icon-192x192.png',
    badge: '/icons/icon-192x192.png',
    data: { url: '/dashboard' },
  };

  if (event.data) {
    try {
      const payload = event.data.json();
      const push = payload && payload.push ? payload.push : payload;

      title = push.title || title;
      options.body = push.body || options.body;
      options.data = {
        url: buildReferenceUrl(push),
      };
    } catch (err) {
      // Payload bukan JSON — tetap tampilkan notifikasi default.
    }
  }

  event.waitUntil(self.registration.showNotification(title, options));
});

// Saat notifikasi diklik: fokus tab yang ada atau buka tab baru ke halaman tujuan.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();

  const targetUrl = (event.notification.data && event.notification.data.url) || '/dashboard';

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if ('focus' in client) {
          client.navigate(targetUrl);
          return client.focus();
        }
      }
      return self.clients.openWindow(targetUrl);
    })
  );
});

// Susun URL tujuan berdasarkan tipe referensi notifikasi.
function buildReferenceUrl(push) {
  const type = push.reference_type;
  const refId = push.reference_id;

  if (type === 'maintenance-log' && refId) return '/maintenance';
  if (type === 'field-maintenance' && refId) return '/field-maintenance';
  if (type === 'daily-digest' && refId) return '/ringkasan';
  if (type === 'zone' && refId) return '/sebaran-cctv';
  if (type === 'maintenance-schedule' && refId) return '/dashboard';
  return '/dashboard';
}
