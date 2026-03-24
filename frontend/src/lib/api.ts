import axios from 'axios'
import Cookies from 'js-cookie'

// Safe UUID generator — crypto.randomUUID() only works in secure contexts (HTTPS/localhost).
// Falls back to a Math.random-based RFC4122 v4 UUID for non-secure origins (e.g. local IP).
function generateUUID(): string {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID()
    }
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = (Math.random() * 16) | 0
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
    })
}

const api = axios.create({
    baseURL: '/api',
    timeout: 30_000,
    headers: { 'Content-Type': 'application/json' },
})

// Request interceptor — attach JWT
api.interceptors.request.use((config) => {
    const token = Cookies.get('pvp_token')
    if (token) config.headers.Authorization = `Bearer ${token}`
    return config
})

// Response interceptor — auto-refresh on 401 (deduplicated to prevent race condition)
let refreshPromise: Promise<string> | null = null

api.interceptors.response.use(
    (res) => res,
    async (err) => {
        const original = err.config
        if (err.response?.status === 401 && !original._retry) {
            original._retry = true
            try {
                // If a refresh is already in flight, wait for it instead of making a new call
                if (!refreshPromise) {
                    const refresh = Cookies.get('pvp_refresh')
                    refreshPromise = axios
                        .post('/api/v1/auth/refresh', { refresh_token: refresh })
                        .then((r) => {
                            const token = r.data.data.access_token
                            Cookies.set('pvp_token', token, { sameSite: 'lax', expires: 1 })
                            // Update refresh token if rotated
                            if (r.data.data.refresh_token) {
                                Cookies.set('pvp_refresh', r.data.data.refresh_token, { sameSite: 'lax', expires: 30 })
                            }
                            return token
                        })
                        .finally(() => { refreshPromise = null })
                }
                const newToken = await refreshPromise
                original.headers.Authorization = `Bearer ${newToken}`
                return api(original)
            } catch {
                refreshPromise = null
                Cookies.remove('pvp_token')
                Cookies.remove('pvp_refresh')
                window.location.href = '/login'
            }
        }
        return Promise.reject(err)
    }
)

// ─── Auth ──────────────────────────────────────────────────────────────────────
export const authAPI = {
    login: (email: string, password: string) =>
        api.post('/v1/auth/login', { email, password }),
    register: (email: string, password: string, fullName: string) =>
        api.post('/v1/auth/register', { email, password, full_name: fullName }),
    logout: () => {
        const refresh = Cookies.get('pvp_refresh')
        // Server requires refresh_token in body to revoke the session.
        // If cookie is missing (already expired), skip the server call — just clear locally.
        return refresh
            ? api.post('/v1/auth/logout', { refresh_token: refresh })
            : Promise.resolve()
    },
    forgotPassword: (email: string) => api.post('/v1/auth/forgot-password', { email }),
    me: () => api.get('/v1/users/me'),
    updateMe: (data: { full_name: string; phone: string }) => api.put('/v1/users/me', data),
    changePassword: (data: { old_password: string; new_password: string }) =>
        api.put('/v1/users/me/password', data),
    // 2FA
    setup2FA: () => api.post('/v1/users/me/2fa/setup'),
    enable2FA: (code: string) => api.post('/v1/users/me/2fa/enable', { code }),
    disable2FA: (code: string) => api.delete('/v1/users/me/2fa', { data: { code } }),
}

// ─── Wallet ────────────────────────────────────────────────────────────────────
export const walletAPI = {
    getBalance: () => api.get('/v1/billing/wallet'),
    getTransactions: (page = 1) => api.get(`/v1/billing/transactions?page=${page}&limit=20`),
    topUp: (amount: string, paymentMethod: string) =>
        api.post('/v1/billing/top-up', { amount, payment_method: paymentMethod }),
}

// ─── Proxy Orders ──────────────────────────────────────────────────────────────
export const proxyAPI = {
    listProducts: (proxyType = '', protocol = '', location = '') =>
        api.get(`/v1/proxy/products?proxy_type=${proxyType}&protocol=${protocol}&location=${location}`),
    listOrders: (page = 1, limit = 20) => api.get(`/v1/proxy/orders?page=${page}&limit=${limit}`),
    createOrder: async (productId: string, qty: number, meta?: { country?: string; isp_id?: string; period_months?: number; protocol?: string }) => {
        // Create qty individual orders SEQUENTIALLY to avoid billing wallet row-lock race condition
        const count = Math.max(1, qty)
        let lastResult: any
        for (let i = 0; i < count; i++) {
            lastResult = await api.post('/v1/proxy/orders', {
                product_id: productId, quantity: 1, metadata: meta ?? {}, idempotency_key: generateUUID()
            })
        }
        return lastResult
    },
    getOrder: (id: string) => api.get(`/v1/proxy/orders/${id}`),
    cancelOrder: (id: string) => api.delete(`/v1/proxy/orders/${id}`),
    patchOrder: (id: string, data: { custom_price?: string; custom_expires_at?: string; admin_note?: string }) =>
        api.patch(`/v1/proxy/orders/${id}`, data),
    getCredentials: (id: string) => api.get(`/v1/proxy/orders/${id}/credentials`),
    getOrderEvents: (id: string) => api.get(`/v1/proxy/orders/${id}/events`),
    renewOrder: (id: string) => api.post(`/v1/proxy/orders/${id}/renew`),
    serviceOptions: (serviceId: string, planId?: string) =>
        api.get(`/v1/proxy/service-options?service_id=${serviceId}${planId ? `&plan_id=${planId}` : ''}`),
}

// ─── VPS ───────────────────────────────────────────────────────────────────────
export const vpsAPI = {
    listPlans: () => api.get('/v1/vps/plans'),
    listInstances: (page = 1) => api.get(`/v1/vps/instances?page=${page}&limit=20`),
    getInstance: (id: string) => api.get(`/v1/vps/instances/${id}`),
    createVPS: (planId: string, hostname: string) =>
        api.post('/v1/vps/orders', { plan_id: planId, hostname, idempotency_key: generateUUID() }),
    startInstance: (id: string) => api.post(`/v1/vps/instances/${id}/start`),
    stopInstance: (id: string) => api.post(`/v1/vps/instances/${id}/stop`),
    rebootInstance: (id: string) => api.post(`/v1/vps/instances/${id}/reboot`),
    terminateInstance: (id: string) => api.delete(`/v1/vps/instances/${id}`),
    getConsole: (id: string) => api.get(`/v1/vps/instances/${id}/console`),
    listSnapshots: (id: string) => api.get(`/v1/vps/instances/${id}/snapshots`),
    createSnapshot: (id: string, name: string) =>
        api.post(`/v1/vps/instances/${id}/snapshots`, { name }),
}

// ─── Admin ─────────────────────────────────────────────────────────────────────
export const adminAPI = {
    listUsers: (page = 1) => api.get(`/v1/admin/users?page=${page}&limit=15`),
    getUser: (id: string) => api.get(`/v1/admin/users/${id}`),
    updateUserRole: (id: string, role: string) => api.put(`/v1/admin/users/${id}/role`, { role }),
    updateUserStatus: (id: string, status: string) => api.put(`/v1/admin/users/${id}/status`, { status }),
    updateUserProfile: (id: string, data: { full_name: string; phone: string }) =>
        api.put(`/v1/admin/users/${id}/profile`, data),
    deleteUser: (id: string) => api.delete(`/v1/admin/users/${id}`),
    adminDisable2FA: (id: string) => api.put(`/v1/admin/users/${id}/2fa/disable`),
    listResellers: (status?: string) =>
        api.get(`/v1/admin/resellers${status ? `?status=${status}` : ''}`),
    approveReseller: (id: string) => api.put(`/v1/admin/resellers/${id}/approve`),
    suspendReseller: (id: string, reason: string) =>
        api.put(`/v1/admin/resellers/${id}/suspend`, { reason }),
    // Proxy product management
    listProxyProducts: (page = 1) => api.get(`/v1/admin/proxy/products?page=${page}&limit=20`),
    createProxyProduct: (data: Record<string, any>) => api.post('/v1/admin/proxy/products', data),
    updateProxyProduct: (id: string, data: Record<string, any>) => api.put(`/v1/admin/proxy/products/${id}`, data),
    deleteProxyProduct: (id: string) => api.delete(`/v1/admin/proxy/products/${id}`),
    toggleProxyProduct: (id: string) => api.put(`/v1/admin/proxy/products/${id}/toggle`),
    // Proxy provider management
    listProxyProviders: () => api.get('/v1/admin/proxy/providers'),
    getProxyServiceOptions: (serviceId: string, planId?: string) =>
        api.get(`/v1/admin/proxy/service-options?service_id=${serviceId}${planId ? `&plan_id=${planId}` : ''}`),
    // VPS plan management
    listVPSPlans: (page = 1) => api.get(`/v1/admin/vps/plans?page=${page}&limit=20`),
    createVPSPlan: (data: Record<string, any>) => api.post('/v1/admin/vps/plans', data),
    toggleVPSPlan: (id: string) => api.put(`/v1/admin/vps/plans/${id}/toggle`),
    // Per-user admin queries
    getUserWallet: (userId: string) => api.get(`/v1/admin/billing/wallet?user_id=${userId}`),
    getUserProxyOrders: (userId: string) => api.get(`/v1/admin/proxy/user-orders?user_id=${userId}&limit=1`),
    getUserVPSInstances: (userId: string) => api.get(`/v1/admin/vps/user-instances?user_id=${userId}&limit=1`),
    // Admin proxy order actions: lock/unlock
    orderAction: (orderId: string, action: 'lock' | 'unlock', reason?: string) =>
        api.put(`/v1/admin/proxy/orders/${orderId}/action`, { action, reason }),
    // Admin manual balance adjustment (reference_type="adjustment" — not counted as revenue)
    adminAdjustBalance: (userId: string, amount: number, description: string) =>
        api.post('/v1/admin/billing/adjustment', { user_id: userId, amount, description }),
}

// ─── Reseller ──────────────────────────────────────────────────────────────────
export const resellerAPI = {
    getDashboard: () => api.get('/v1/reseller/dashboard'),
    listSubAccounts: (page = 1) => api.get(`/v1/reseller/users?page=${page}&limit=20`),
    createSubAccount: (userId: string, creditLimit: string) =>
        api.post('/v1/reseller/users', { user_id: userId, credit_limit: creditLimit }),
    listPricing: () => api.get('/v1/reseller/pricing'),
    setPricing: (productId: string, sellPrice: string) =>
        api.put(`/v1/reseller/pricing/${productId}`, { sell_price: sellPrice }),
    listAPIKeys: () => api.get('/v1/reseller/api-keys'),
    createAPIKey: (name: string, scopes: string[]) =>
        api.post('/v1/reseller/api-keys', { name, scopes }),
    revokeAPIKey: (id: string) => api.delete(`/v1/reseller/api-keys/${id}`),
    listWebhooks: () => api.get('/v1/reseller/webhooks'),
    createWebhook: (url: string, secret: string, events: string[]) =>
        api.post('/v1/reseller/webhooks', { url, secret, events }),
    deleteWebhook: (id: string) => api.delete(`/v1/reseller/webhooks/${id}`),
}

// ─── Logs / Audit ──────────────────────────────────────────────────────────────
export const logsAPI = {
    list: (params: { user_id?: string; action?: string; service?: string; page?: number; limit?: number } = {}) => {
        const q = new URLSearchParams()
        if (params.user_id) q.set('user_id', params.user_id)
        if (params.action) q.set('action', params.action)
        if (params.service) q.set('service', params.service)
        if (params.page) q.set('page', String(params.page))
        if (params.limit) q.set('limit', String(params.limit))
        return api.get(`/v1/logs?${q}`)
    },
}

export default api
