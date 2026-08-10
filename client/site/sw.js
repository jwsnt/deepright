const MESSAGE_IMAGE_CACHE = 'deepright-message-images-v1';

self.addEventListener('install', event => {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    const keys = await caches.keys();
    await Promise.all(
      keys
        .filter(key => key.startsWith('deepright-message-images-') && key !== MESSAGE_IMAGE_CACHE)
        .map(key => caches.delete(key))
    );
    await self.clients.claim();
  })());
});

function isCacheableImageRequest(request) {
  if (!request || request.method !== 'GET' || request.destination !== 'image') return false;
  try {
    const url = new URL(request.url);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

self.addEventListener('fetch', event => {
  const request = event.request;
  if (!isCacheableImageRequest(request)) return;
  event.respondWith((async () => {
    const cache = await caches.open(MESSAGE_IMAGE_CACHE);
    const cached = await cache.match(request, { ignoreVary: true });
    if (cached) return cached;
    const response = await fetch(request);
    if (response && (response.ok || response.type === 'opaque')) {
      cache.put(request, response.clone()).catch(() => {});
    }
    return response;
  })());
});
