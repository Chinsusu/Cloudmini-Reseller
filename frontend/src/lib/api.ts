import axios from 'axios'
import Cookies from 'js-cookie'

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
                            Cookies.set('pvp_token', token, { secure: true, sameSite: 'strict' })
                            // Update refresh token if rotated
                            if (r.data.data.refresh_token) {
                                Cookies.set('pvp_refresh', r.data.data.refresh_token, { secure: true, sameSite: 'strict' })
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
    logout: () => api.post('/v1/auth/logout'),
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
    listOrders: (page = 1) => api.get(`/v1/proxy/orders?page=${page}&limit=20`),
    createOrder: (productId: string, qty: number) =>
        api.post('/v1/proxy/orders', { product_id: productId, quantity: qty, idempotency_key: crypto.randomUUID() }),
    getOrder: (id: string) => api.get(`/v1/proxy/orders/${id}`),
    cancelOrder: (id: string) => api.delete(`/v1/proxy/orders/${id}`),
    getCredentials: (id: string) => api.get(`/v1/proxy/orders/${id}/credentials`),
}

// ─── VPS ───────────────────────────────────────────────────────────────────────
export const vpsAPI = {
    listPlans: () => api.get('/v1/vps/plans'),
    listInstances: (page = 1) => api.get(`/v1/vps/instances?page=${page}&limit=20`),
    getInstance: (id: string) => api.get(`/v1/vps/instances/${id}`),
    createVPS: (planId: string, hostname: string) =>
        api.post('/v1/vps/orders', { plan_id: planId, hostname, idempotency_key: crypto.randomUUID() }),
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
    toggleProxyProduct: (id: string) => api.put(`/v1/admin/proxy/products/${id}/toggle`),
    // VPS plan management
    listVPSPlans: (page = 1) => api.get(`/v1/admin/vps/plans?page=${page}&limit=20`),
    createVPSPlan: (data: Record<string, any>) => api.post('/v1/admin/vps/plans', data),
    toggleVPSPlan: (id: string) => api.put(`/v1/admin/vps/plans/${id}/toggle`),
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
