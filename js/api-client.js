(function (global) {
    const REQUEST_TIMEOUT_MS = 10000;

    function normalizeBase(base) {
        return (base || API_BASE || '').replace(/\/+$/, '');
    }

    function withQuery(path, query) {
        const entries = Object.entries(query || {}).filter(([, value]) => value !== undefined && value !== null && value !== '');
        if (!entries.length) {
            return path;
        }

        const params = new URLSearchParams();
        entries.forEach(([key, value]) => {
            params.append(key, String(value));
        });
        return `${path}?${params.toString()}`;
    }

    function prepareRequest(options) {
        const init = { ...options };
        const body = init.body;
        const requestHeaders = init.headers ? { ...init.headers } : { ...headers };

        if (body instanceof FormData) {
            Object.keys(requestHeaders).forEach((key) => {
                if (key.toLowerCase() === 'content-type') {
                    delete requestHeaders[key];
                }
            });
        } else if (body !== undefined && body !== null && typeof body === 'object' && !(body instanceof Blob)) {
            init.body = JSON.stringify(body);
            if (!Object.keys(requestHeaders).some((key) => key.toLowerCase() === 'content-type')) {
                requestHeaders['Content-Type'] = 'application/json';
            }
        }

        init.headers = requestHeaders;
        return init;
    }

    function request(path, options) {
        const init = prepareRequest(options || {});
        const base = normalizeBase(init.base);
        delete init.base;

        if (!init.signal && typeof AbortController !== 'undefined') {
            const controller = new AbortController();
            const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
            init.signal = controller.signal;

            return fetch(`${base}${path}`, init).finally(() => {
                clearTimeout(timer);
            });
        }

        return fetch(`${base}${path}`, init);
    }

    global.LinkGenieAPI = {
        system: {
            getStatus(options) {
                return request('/api/system/status', options);
            },
            getConfig(options) {
                return request('/api/system/config', options);
            },
            saveConfig(payload, options) {
                return request('/api/system/config', {
                    method: 'POST',
                    body: payload,
                    ...(options || {})
                });
            }
        },
        folders: {
            list(options) {
                return request('/api/folders/', options);
            },
            getBookmarks(folderId, options) {
                return request(`/api/folders/${folderId}/bookmarks`, options);
            },
            create(payload, options) {
                return request('/api/folders/', {
                    method: 'POST',
                    body: payload,
                    ...(options || {})
                });
            },
            update(folderId, payload, options) {
                return request(`/api/folders/${folderId}`, {
                    method: 'PUT',
                    body: payload,
                    ...(options || {})
                });
            },
            remove(folderId, options) {
                return request(`/api/folders/${folderId}`, {
                    method: 'DELETE',
                    ...(options || {})
                });
            },
            addBookmark(bookmarkId, folderIds, options) {
                return request(`/api/bookmarks/${bookmarkId}/folders`, {
                    method: 'POST',
                    body: { folder_ids: folderIds },
                    ...(options || {})
                });
            },
            removeBookmark(bookmarkId, folderId, options) {
                return request(`/api/bookmarks/${bookmarkId}/folders/${folderId}`, {
                    method: 'DELETE',
                    ...(options || {})
                });
            }
        },
        workflows: {
            list(options) {
                return request('/api/workflows/', options);
            },
            create(payload, options) {
                return request('/api/workflows/', {
                    method: 'POST',
                    body: payload,
                    ...(options || {})
                });
            },
            update(workflowId, payload, options) {
                return request(`/api/workflows/${workflowId}`, {
                    method: 'PUT',
                    body: payload,
                    ...(options || {})
                });
            },
            toggle(workflowId, options) {
                return request(`/api/workflows/${workflowId}/toggle`, {
                    method: 'POST',
                    ...(options || {})
                });
            },
            remove(workflowId, options) {
                return request(`/api/workflows/${workflowId}`, {
                    method: 'DELETE',
                    ...(options || {})
                });
            },
            apply(workflowIds, bookmarkIds, options) {
                return request('/api/workflows/apply', {
                    method: 'POST',
                    body: {
                        workflow_ids: workflowIds,
                        bookmark_ids: bookmarkIds
                    },
                    ...(options || {})
                });
            }
        },
        bookmarks: {
            list(query, options) {
                return request(withQuery('/api/bookmarks/', query), options);
            },
            get(bookmarkId, options) {
                return request(`/api/bookmarks/${bookmarkId}/`, options);
            },
            create(payload, options) {
                return request('/api/bookmarks/', {
                    method: 'POST',
                    body: payload,
                    ...(options || {})
                });
            },
            update(bookmarkId, payload, options) {
                return request(`/api/bookmarks/${bookmarkId}/`, {
                    method: 'PUT',
                    body: payload,
                    ...(options || {})
                });
            },
            remove(bookmarkId, options) {
                return request(`/api/bookmarks/${bookmarkId}/`, {
                    method: 'DELETE',
                    ...(options || {})
                });
            },
            enhance(bookmarkId, options) {
                return request(`/api/bookmarks/${bookmarkId}/enhance/`, {
                    method: 'POST',
                    ...(options || {})
                });
            },
            importFile(file, options) {
                const formData = new FormData();
                formData.append('file', file);
                return request('/api/bookmarks/import/', {
                    method: 'POST',
                    body: formData,
                    ...(options || {})
                });
            },
            export(options) {
                return request('/api/bookmarks/export/', options);
            }
        },
        tags: {
            getStats(options) {
                return request('/api/tags/stats', options);
            },
            optimize(payload, options) {
                return request('/api/tags/optimize', {
                    method: 'POST',
                    body: payload,
                    ...(options || {})
                });
            }
        }
    };
})(window);
