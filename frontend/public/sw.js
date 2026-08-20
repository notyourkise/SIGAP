/* ==========================================
   SIGAP — Service Worker (Task 8.1 PWA + Task 8.3 Push)
   ==========================================
   Scope: installability + offline shell + notifikasi push browser.
   Strategi cache:
     - Shell (HTML navigasi, CSS, JS chunks, ikon/font): disimpan di cache.
     - Navigasi HTML   : network-first, fallback ke cache (offline shell).
     - Aset statis     : cache-first dengan pembaruan latar belakang
                         (hashed oleh Next.js sehingga aman di-cache).
     - API calls       : network-first, fallback ke cache (bila ada), selain itu
                         respons JSON ringan "offline". Request `no-store` dan
                         gambar QR selalu network-only agar data lama tidak dipakai.
   Push (Task 8.3): menangani event `push` dari server (Web Push/VAPID) dan
   menampilkan notifikasi OS via showNotification, serta membuka halaman yang
   relevan saat notifikasi diklik.
   BUKAN full offline (tanpa IndexedDB) — data tetap butuh jaringan/API.
   ========================================== */

// Nama cache versi-mu — naikkan saat konten shell berubah agar cache lama dibuang.
const CACHE_VERSION = 'sigap-shell-v3';
const SHELL_CACHE = CACHE_VERSION;

// Aset inti shell yang di-pre-cache saat install (diperbarui saat activate).
const PRECACHE_URLS = [
  '/',
  '/dashboard',
  '/login',
  '/manifest.json',
  '/icons/icon-192x192.png',
  '/icons/icon-512x512.png',
];

// Batas waktu (detik) sebelum permintaan jaringan dianggap gagal pada
// strategi network-first (mencegah layar kosong saat koneksi lambat).
const NETWORK_TIMEOUT = 5000;

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(SHELL_CACHE)
      .then((cache) => cache.addAll(PRECACHE_URLS))
      // Langsung aktif tanpa menunggu tab lama ditutup.
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
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

// QR hasil rotasi tidak boleh memiliki fallback cache: gambar lama yang masih
// tersedia offline lebih berbahaya daripada menampilkan kegagalan jaringan.
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

// ---- Strategi: network-first (untuk navigasi & API) ----
async function networkFirst(request) {
  const cache = await caches.open(SHELL_CACHE);
  try {
    // Beri batas waktu agar tidak menggantung saat offline.
    const timeoutPromise = new Promise((_, reject) =>
      setTimeout(() => reject(new Error('timeout')), NETWORK_TIMEOUT)
    );
    const fetchPromise = fetch(request);
    const response = await Promise.race([fetchPromise, timeoutPromise]);
    if (response && response.ok) {
      // Simpan salinan untuk offline / fallback.
      if (isRuntimeCacheAllowed(request, response)) {
        cache.put(request, response.clone());
      }
      return response;
    }
    throw new Error('response not ok');
  } catch (err) {
    const cached = await cache.match(request);
    if (cached) return cached;
    // Fallback khusus navigasi ke shell utama (halaman dashboard).
    if (isNavigationRequest(request)) {
      const index = await cache.match('/dashboard');
      if (index) return index;
    }
    throw err;
  }
}

// ---- Strategi: cache-first (untuk aset statis shell) ----
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

// ---- Fallback kecil bila permintaan API gagal & tidak ada cache ----
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

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Jangan tangani permintaan lintas-origin (hanya same-origin untuk shell & API).
  if (url.origin !== self.location.origin) return;

  // Permintaan bukan GET tidak di-cache (biarkan jaringan normal).
  if (request.method !== 'GET') return;

  // Request no-store dan gambar QR harus network-only. Jangan pernah fallback
  // ke label lama setelah rotasi ketika jaringan lambat atau offline.
  if (request.cache === 'no-store' || isQRCodeImageRequest(url)) {
    event.respondWith(fetch(request));
    return;
  }

  // API calls: network-first (data selalu segar), fallback cache/offline.
  if (isApiRequest(url)) {
    event.respondWith(networkFirst(request).catch(() => offlineApiResponse()));
    return;
  }

  // Navigasi halaman (HTML shell): network-first, fallback cache offline.
  if (isNavigationRequest(request)) {
    event.respondWith(networkFirst(request).catch(() => caches.match(request)));
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
  return '/dashboard';
}
