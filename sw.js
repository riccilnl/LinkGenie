const CACHE_NAME = 'bookmark-app-v7';
const RUNTIME_CACHE = 'runtime-cache-v2';
const ASSETS_TO_CACHE = [
    '/',
    '/index.html',
    '/icon.svg',
    '/apple-touch-icon.png',
    '/manifest.json',
    '/css/base.css',
    '/css/layout.css',
    '/css/components.css',
    '/css/mobile.css',
    '/js/config.js',
    '/js/ui.js',
    '/js/store.js',
    '/js/api-client.js',
    '/js/app.js'
];

function shouldUseNetworkFirst(request, url) {
    if (request.mode === 'navigate') {
        return true;
    }

    if (url.pathname === '/' || url.pathname === '/index.html' || url.pathname === '/manifest.json') {
        return true;
    }

    if (url.pathname.startsWith('/js/') || url.pathname.startsWith('/css/')) {
        return true;
    }

    return false;
}

async function networkFirst(request, cacheName) {
    try {
        const response = await fetch(request);
        if (response.ok) {
            const cache = await caches.open(cacheName);
            cache.put(request, response.clone());
        }
        return response;
    } catch (error) {
        const cachedResponse = await caches.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        throw error;
    }
}

async function cacheFirst(request, cacheName) {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
        return cachedResponse;
    }

    const response = await fetch(request);
    if (response.ok) {
        const cache = await caches.open(cacheName);
        cache.put(request, response.clone());
    }
    return response;
}

// 安装 Service Worker
self.addEventListener('install', (event) => {
    console.log('[SW] Installing Service Worker v7...');
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => {
                console.log('[SW] Caching static assets');
                return cache.addAll(ASSETS_TO_CACHE);
            })
            .then(() => self.skipWaiting())
    );
});

// 激活并清理旧缓存
self.addEventListener('activate', (event) => {
    console.log('[SW] Activating Service Worker v7...');
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames
                    .filter((name) => name !== CACHE_NAME && name !== RUNTIME_CACHE)
                    .map((name) => {
                        console.log('[SW] Deleting old cache:', name);
                        return caches.delete(name);
                    })
            );
        }).then(() => self.clients.claim())
    );
});

// 拦截网络请求
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // 跳过非 GET 请求
    if (request.method !== 'GET') {
        return;
    }

    // API 请求 - 网络优先,缓存降级
    if (url.pathname.startsWith('/api/')) {
        event.respondWith(
            networkFirst(request, RUNTIME_CACHE)
                .catch(() => {
                    // 网络失败,尝试从缓存读取
                    console.log('[SW] Network failed, using cache for:', url.pathname);
                    return caches.match(request).then((cachedResponse) => {
                        if (cachedResponse) {
                            return cachedResponse;
                        }
                        // 返回离线页面或错误响应
                        return new Response(
                            JSON.stringify({ error: 'Offline', message: '网络连接失败,请稍后重试' }),
                            {
                                status: 503,
                                headers: { 'Content-Type': 'application/json' }
                            }
                        );
                    });
                })
        );
        return;
    }

    // 前端壳资源 - 网络优先,缓存降级,避免长期吃旧缓存
    if (shouldUseNetworkFirst(request, url)) {
        event.respondWith(
            networkFirst(request, CACHE_NAME).catch(() => {
                return caches.match(request).then((cachedResponse) => {
                    if (cachedResponse) {
                        return cachedResponse;
                    }
                    return caches.match('/index.html');
                });
            })
        );
        return;
    }

    // 其他静态资源 - 缓存优先,网络降级
    event.respondWith(
        cacheFirst(request, CACHE_NAME)
    );
});
