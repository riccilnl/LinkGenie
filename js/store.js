(function (global) {
    function replaceObject(target, next) {
        Object.keys(target).forEach(key => {
            if (!Object.prototype.hasOwnProperty.call(next, key)) {
                delete target[key];
            }
        });
        Object.assign(target, next);
    }

    function replaceArray(target, next) {
        target.splice(0, target.length, ...next);
    }

    const state = {
        systemConfigState: {
            ai_api_key_set: false,
            ai_runtime_available: false
        },
        folders: [],
        currentFolderId: null,
        bookmarkUrls: new Set(),
        allBookmarksData: [],
        aiProcessingBookmarks: new Set(),
        workflows: [],
        currentWorkflowId: null,
        workflowTriggers: [],
        workflowActions: [],
        currentCategory: 'all',
        currentBookmarkId: null,
        deleteBookmarkId: null
    };

    function rebuildBookmarkUrls() {
        state.bookmarkUrls.clear();
        state.allBookmarksData.forEach(bookmark => {
            if (bookmark && bookmark.url) {
                state.bookmarkUrls.add(bookmark.url);
            }
        });
    }

    global.LinkGenieStore = {
        getSystemConfigState() {
            return state.systemConfigState;
        },
        setSystemConfigState(next) {
            replaceObject(state.systemConfigState, next || {});
            return state.systemConfigState;
        },
        getFolders() {
            return state.folders;
        },
        replaceFolders(next) {
            replaceArray(state.folders, Array.isArray(next) ? next : []);
            return state.folders;
        },
        getCurrentFolderId() {
            return state.currentFolderId;
        },
        setCurrentFolderId(next) {
            state.currentFolderId = next == null ? null : Number(next);
            return state.currentFolderId;
        },
        getBookmarkUrls() {
            return state.bookmarkUrls;
        },
        getAllBookmarksData() {
            return state.allBookmarksData;
        },
        replaceAllBookmarksData(next) {
            replaceArray(state.allBookmarksData, Array.isArray(next) ? next : []);
            rebuildBookmarkUrls();
            return state.allBookmarksData;
        },
        upsertBookmark(bookmark) {
            const index = state.allBookmarksData.findIndex(item => item.id === bookmark.id);
            if (index === -1) {
                state.allBookmarksData.push(bookmark);
            } else {
                state.allBookmarksData[index] = bookmark;
            }
            rebuildBookmarkUrls();
            return state.allBookmarksData;
        },
        getAIProcessingBookmarks() {
            return state.aiProcessingBookmarks;
        },
        getWorkflows() {
            return state.workflows;
        },
        replaceWorkflows(next) {
            replaceArray(state.workflows, Array.isArray(next) ? next : []);
            return state.workflows;
        },
        getCurrentWorkflowId() {
            return state.currentWorkflowId;
        },
        setCurrentWorkflowId(next) {
            state.currentWorkflowId = next == null ? null : Number(next);
            return state.currentWorkflowId;
        },
        getWorkflowTriggers() {
            return state.workflowTriggers;
        },
        replaceWorkflowTriggers(next) {
            replaceArray(state.workflowTriggers, Array.isArray(next) ? next : []);
            return state.workflowTriggers;
        },
        getWorkflowActions() {
            return state.workflowActions;
        },
        replaceWorkflowActions(next) {
            replaceArray(state.workflowActions, Array.isArray(next) ? next : []);
            return state.workflowActions;
        },
        getCurrentCategory() {
            return state.currentCategory;
        },
        setCurrentCategory(next) {
            state.currentCategory = next || 'all';
            return state.currentCategory;
        },
        getCurrentBookmarkId() {
            return state.currentBookmarkId;
        },
        setCurrentBookmarkId(next) {
            state.currentBookmarkId = next == null ? null : Number(next);
            return state.currentBookmarkId;
        },
        getDeleteBookmarkId() {
            return state.deleteBookmarkId;
        },
        setDeleteBookmarkId(next) {
            state.deleteBookmarkId = next == null ? null : Number(next);
            return state.deleteBookmarkId;
        }
    };
})(window);
