const CACHE_NAME = 'bookmark-app-v6';
const RUNTIME_CACHE = 'runtime-cache-v1';
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
    '/js/app.js'
];

// 安装 Service Worker
self.addEventListener('install', (event) => {
    console.log('[SW] Installing Service Worker v6...');
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
    console.log('[SW] Activating Service Worker v6...');
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
            fetch(request)
                .then((response) => {
                    // 只缓存成功的响应
                    if (response.ok) {
                        const clonedResponse = response.clone();
                        caches.open(RUNTIME_CACHE).then((cache) => {
                            cache.put(request, clonedResponse);
                        });
                    }
                    return response;
                })
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

    // 静态资源 - 缓存优先,网络降级
    event.respondWith(
        caches.match(request)
            .then((cachedResponse) => {
                if (cachedResponse) {
                    return cachedResponse;
                }

                return fetch(request).then((response) => {
                    // 缓存新的静态资源
                    if (response.ok) {
                        const clonedResponse = response.clone();
                        caches.open(CACHE_NAME).then((cache) => {
                            cache.put(request, clonedResponse);
                        });
                    }
                    return response;
                });
            })
    );
});
