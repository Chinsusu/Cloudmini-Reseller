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

// Response interceptor — auto-refresh on 401
api.interceptors.response.use(
    (res) => res,
    async (err) => {
        const original = err.config
        if (err.response?.status === 401 && !original._retry) {
            original._retry = true
            try {
                const refresh = Cookies.get('pvp_refresh')
                const { data } = await axios.post('/api/v1/auth/refresh', { refresh_token: refresh })
                Cookies.set('pvp_token', data.data.access_token, { secure: true, sameSite: 'strict' })
                original.headers.Authorization = `Bearer ${data.data.access_token}`
                return api(original)
            } catch {
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
    me: () => api.get('/v1/me'),
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
    listProducts: () => api.get('/v1/proxy/products'),
    listOrders: (page = 1) => api.get(`/v1/proxy/orders?page=${page}&limit=20`),
    createOrder: (planId: string, qty: number, country: string) =>
        api.post('/v1/proxy/orders', { plan_id: planId, quantity: qty, country }),
    getOrder: (id: string) => api.get(`/v1/proxy/orders/${id}`),
    cancelOrder: (id: string) => api.delete(`/v1/proxy/orders/${id}`),
    getCredentials: (id: string) => api.get(`/v1/proxy/orders/${id}/credentials`),
}

// ─── VPS ───────────────────────────────────────────────────────────────────────
export const vpsAPI = {
    listPlans: () => api.get('/v1/vps/plans'),
    listInstances: (page = 1) => api.get(`/v1/vps/instances?page=${page}&limit=20`),
    getInstance: (id: string) => api.get(`/v1/vps/instances/${id}`),
    createVPS: (planId: string, hostname: string, idempotencyKey: string) =>
        api.post('/v1/vps/orders', { plan_id: planId, hostname, idempotency_key: idempotencyKey }),
    startVPS: (id: string) => api.post(`/v1/vps/instances/${id}/start`),
    stopVPS: (id: string) => api.post(`/v1/vps/instances/${id}/stop`),
    rebootVPS: (id: string) => api.post(`/v1/vps/instances/${id}/reboot`),
    deleteVPS: (id: string) => api.delete(`/v1/vps/instances/${id}`),
    getConsole: (id: string) => api.get(`/v1/vps/instances/${id}/console`),
    listSnapshots: (id: string) => api.get(`/v1/vps/instances/${id}/snapshots`),
    createSnapshot: (id: string, name: string) =>
        api.post(`/v1/vps/instances/${id}/snapshots`, { name }),
}

// ─── Admin ─────────────────────────────────────────────────────────────────────
export const adminAPI = {
    listUsers: (page = 1) => api.get(`/v1/admin/users?page=${page}&limit=20`),
    listResellers: (status?: string) =>
        api.get(`/v1/admin/resellers${status ? `?status=${status}` : ''}`),
    approveReseller: (id: string) => api.put(`/v1/admin/resellers/${id}/approve`),
    suspendReseller: (id: string, reason: string) =>
        api.put(`/v1/admin/resellers/${id}/suspend`, { reason }),
}

// ─── Reseller ──────────────────────────────────────────────────────────────────
export const resellerAPI = {
    getDashboard: () => api.get('/v1/reseller/dashboard'),
    listSubAccounts: () => api.get('/v1/reseller/users'),
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
}

export default api
