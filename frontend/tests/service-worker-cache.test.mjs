import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import vm from 'node:vm';

const serviceWorkerSource = await readFile(
  new URL('../public/sw.js', import.meta.url),
  'utf8'
);

function loadServiceWorker(fetchImpl) {
  const listeners = new Map();
  const calls = { open: 0, match: 0, put: 0 };
  const cache = {
    addAll: async () => undefined,
    match: async () => {
      calls.match += 1;
      return new Response('stale');
    },
    put: async () => {
      calls.put += 1;
    },
  };

  const context = vm.createContext({
    URL,
    Response,
    Promise,
    JSON,
    Error,
    setTimeout,
    fetch: fetchImpl,
    caches: {
      open: async () => {
        calls.open += 1;
        return cache;
      },
      keys: async () => [],
      delete: async () => true,
      match: cache.match,
    },
    self: {
      location: { origin: 'https://sigap.test' },
      addEventListener: (type, handler) => listeners.set(type, handler),
      skipWaiting: async () => undefined,
      clients: {
        claim: async () => undefined,
        matchAll: async () => [],
        openWindow: async () => undefined,
      },
      registration: { showNotification: async () => undefined },
    },
  });

  vm.runInContext(serviceWorkerSource, context, { filename: 'public/sw.js' });
  return { listeners, calls };
}

async function dispatchFetch(handler, request) {
  let responsePromise;
  handler({
    request,
    respondWith: (value) => {
      responsePromise = Promise.resolve(value);
    },
  });
  assert.ok(responsePromise, 'service worker must handle the request');
  return responsePromise;
}

test('QR image requests bypass service-worker runtime cache', async () => {
  const fresh = new Response('fresh-qr', {
    status: 200,
    headers: { 'Content-Type': 'image/png', 'Cache-Control': 'no-store' },
  });
  const { listeners, calls } = loadServiceWorker(async () => fresh.clone());

  const response = await dispatchFetch(listeners.get('fetch'), {
    url: 'https://sigap.test/api/cctv/6eb3fd94-9238-44e6-a71c-3f4bf756fcf8/qrcode',
    method: 'GET',
    mode: 'cors',
    cache: 'default',
  });

  assert.equal(await response.text(), 'fresh-qr');
  assert.deepEqual(calls, { open: 0, match: 0, put: 0 });
});

test('network-first does not store responses marked no-store', async () => {
  const fresh = new Response('{"status":"success"}', {
    status: 200,
    headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' },
  });
  const { listeners, calls } = loadServiceWorker(async () => fresh.clone());

  const response = await dispatchFetch(listeners.get('fetch'), {
    url: 'https://sigap.test/api/profile',
    method: 'GET',
    mode: 'cors',
    cache: 'default',
  });

  assert.equal(response.status, 200);
  assert.equal(calls.open, 1);
  assert.equal(calls.put, 0);
});
