'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { proxyAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import {
    ShoppingCart, Eye, EyeOff, Copy, XCircle, RefreshCw,
    Plus, X, Globe, Wifi, Clock, Database, ChevronRight
} from 'lucide-react'

const STATUS_COLOR: Record<string, string> = {
    active: 'success', pending: 'warning', processing: 'info',
    cancelled: 'error', failed: 'error', expired: 'secondary', refunded: 'secondary',
}

function OrderModal({ onClose }: { onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [filter, setFilter] = useState({ proxy_type: '', protocol: '', location: '' })
    const [selected, setSelected] = useState<any>(null)
    const [qty, setQty] = useState(1)
    const [buying, setBuying] = useState(false)

    const { data, isLoading } = useQuery({
        queryKey: ['proxy-products', filter],
        queryFn: () => proxyAPI.listProducts(filter.proxy_type, filter.protocol, filter.location),
    })
    const products = data?.data?.data ?? []

    const handleBuy = async () => {
        if (!selected) return
        setBuying(true)
        try {
            await proxyAPI.createOrder(selected.id, qty)
            success('Order placed! Processing...')
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
            onClose()
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Order failed')
        } finally {
            setBuying(false)
        }
    }

    const total = selected ? (parseFloat(selected.base_cost) * qty).toFixed(2) : '0.00'

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 780, width: '95vw', maxHeight: '90vh', overflowY: 'auto' }}
                onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}>
                        <ShoppingCart size={18} /> Buy Proxy
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body">
                    {/* Filters */}
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '.6rem', marginBottom: '1rem' }}>
                        <select className="input" value={filter.proxy_type} onChange={e => setFilter(f => ({ ...f, proxy_type: e.target.value }))}>
                            <option value="">All Types</option>
                            <option value="residential">Residential</option>
                            <option value="datacenter">Datacenter</option>
                            <option value="mobile">Mobile</option>
                        </select>
                        <select className="input" value={filter.protocol} onChange={e => setFilter(f => ({ ...f, protocol: e.target.value }))}>
                            <option value="">All Protocols</option>
                            <option value="http">HTTP</option>
                            <option value="socks5">SOCKS5</option>
                        </select>
                        <input className="input" placeholder="Location (e.g. US, VN)" value={filter.location}
                            onChange={e => setFilter(f => ({ ...f, location: e.target.value }))} />
                    </div>

                    {/* Products grid */}
                    {isLoading ? (
                        <div className="loading-spinner">Loading products...</div>
                    ) : products.length === 0 ? (
                        <div className="empty-state"><Globe size={36} opacity={0.3} /><p>No products available</p></div>
                    ) : (
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill,minmax(220px,1fr))', gap: '.65rem', marginBottom: '1rem' }}>
                            {products.map((p: any) => (
                                <div key={p.id} onClick={() => setSelected(p)}
                                    style={{
                                        border: `2px solid ${selected?.id === p.id ? 'var(--primary)' : 'var(--border-light)'}`,
                                        borderRadius: 10, padding: '1rem', cursor: 'pointer',
                                        background: selected?.id === p.id ? 'var(--primary-light)' : 'var(--surface)',
                                        transition: 'all .15s',
                                    }}>
                                    <div style={{ fontWeight: 700, fontSize: '.92rem', marginBottom: '.3rem' }}>{p.name}</div>
                                    <div style={{ display: 'flex', gap: '.4rem', flexWrap: 'wrap', marginBottom: '.5rem' }}>
                                        <span className="badge badge-info">{p.proxy_type}</span>
                                        <span className="badge badge-secondary">{p.protocol?.toUpperCase()}</span>
                                    </div>
                                    <div style={{ fontSize: '.8rem', color: 'var(--text-muted)', marginBottom: '.4rem' }}>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem' }}>
                                            <Globe size={11} />{p.location}
                                        </span>
                                        {p.duration_days && <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem' }}>
                                            <Clock size={11} />{p.duration_days} days
                                        </span>}
                                        {p.bandwidth_gb && <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem' }}>
                                            <Database size={11} />{p.bandwidth_gb} GB
                                        </span>}
                                    </div>
                                    <div style={{ fontWeight: 800, color: 'var(--primary)', fontSize: '1.05rem' }}>${parseFloat(p.base_cost).toFixed(2)}</div>
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Order summary */}
                    {selected && (
                        <div style={{ background: 'var(--bg)', borderRadius: 8, padding: '1rem', border: '1px solid var(--border-light)' }}>
                            <div style={{ fontWeight: 600, marginBottom: '.75rem' }}>Order Summary — {selected.name}</div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', flexWrap: 'wrap' }}>
                                <div className="form-group" style={{ marginBottom: 0, minWidth: 120 }}>
                                    <label>Quantity</label>
                                    <input className="input" type="number" min={1} max={1000} value={qty}
                                        onChange={e => setQty(Math.max(1, parseInt(e.target.value) || 1))} />
                                </div>
                                <div style={{ flex: 1 }}>
                                    <div style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>Unit price: ${parseFloat(selected.base_cost).toFixed(2)}</div>
                                    <div style={{ fontWeight: 800, fontSize: '1.2rem', color: 'var(--primary)' }}>Total: ${total}</div>
                                </div>
                                <button className="btn-primary" onClick={handleBuy} disabled={buying}>
                                    <ShoppingCart size={15} />
                                    {buying ? 'Placing...' : 'Confirm Order'}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default function ProxyOrdersPage() {
    const [page, setPage] = useState(1)
    const [showModal, setShowModal] = useState(false)
    const [revealed, setRevealed] = useState<Record<string, any>>({})
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()
    const qc = useQueryClient()

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['proxy-orders', page],
        queryFn: () => proxyAPI.listOrders(page),
        refetchInterval: 15000,
    })
    const orders = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const revealMut = useMutation({
        mutationFn: (id: string) => proxyAPI.getCredentials(id),
        onSuccess: (res, id) => setRevealed(prev => ({ ...prev, [id]: res.data?.data })),
        onError: () => toastError('Failed to load credentials'),
    })

    const cancelMut = useMutation({
        mutationFn: (id: string) => proxyAPI.cancelOrder(id),
        onSuccess: () => { success('Order cancelled'); qc.invalidateQueries({ queryKey: ['proxy-orders'] }) },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Cancel failed'),
    })

    const handleCancel = async (id: string, num: string) => {
        const ok = await confirm({ title: 'Cancel Order', message: `Cancel order ${num}? This cannot be undone.`, confirmLabel: 'Cancel Order', variant: 'danger' })
        if (ok) cancelMut.mutate(id)
    }

    const handleCopy = async (id: string) => {
        const creds = revealed[id]
        if (!creds) return
        const text = `${creds.username}:${creds.password}@${creds.host}:${creds.port}`
        await navigator.clipboard.writeText(text)
        success('Credentials copied!')
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Proxy Orders' }]}>
            {confirmDialog}
            {showModal && <OrderModal onClose={() => setShowModal(false)} />}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Proxy Orders</h1>
                    <p className="page-subtitle">{meta.total ?? 0} orders total</p>
                </div>
                <div style={{ display: 'flex', gap: '.6rem' }}>
                    <button className="btn-secondary" onClick={() => refetch()}><RefreshCw size={14} /> Refresh</button>
                    <button className="btn-primary" onClick={() => setShowModal(true)}><Plus size={14} /> Buy Proxy</button>
                </div>
            </div>

            <div className="card" style={{ padding: 0 }}>
                {isLoading ? (
                    <div className="loading-spinner">Loading orders...</div>
                ) : orders.length === 0 ? (
                    <div className="empty-state">
                        <Globe size={44} opacity={0.3} />
                        <p>No proxy orders yet</p>
                        <button className="btn-primary" onClick={() => setShowModal(true)}>
                            <Plus size={14} /> Buy your first proxy
                        </button>
                    </div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Order #</th>
                                        <th>Type</th>
                                        <th>Qty</th>
                                        <th>Amount</th>
                                        <th>Expires</th>
                                        <th>Status</th>
                                        <th>Credentials</th>
                                        <th>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {orders.map((o: any) => (
                                        <tr key={o.id}>
                                            <td><code style={{ fontSize: '.8rem' }}>{o.order_number}</code></td>
                                            <td><span className="badge badge-info">{o.proxy_type ?? '—'}</span></td>
                                            <td>{o.quantity}</td>
                                            <td><strong>${parseFloat(o.total_amount).toFixed(2)}</strong></td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                {o.expires_at ? new Date(o.expires_at).toLocaleDateString() : '—'}
                                            </td>
                                            <td><span className={`badge badge-${STATUS_COLOR[o.status] ?? 'secondary'}`}>{o.status}</span></td>
                                            <td>
                                                {o.status === 'active' ? (
                                                    revealed[o.id] ? (
                                                        <div style={{ display: 'flex', gap: '.4rem', alignItems: 'center' }}>
                                                            <code style={{ fontSize: '.75rem' }}>{revealed[o.id].username}:***</code>
                                                            <button className="action-btn" onClick={() => handleCopy(o.id)} title="Copy"><Copy size={12} /></button>
                                                            <button className="action-btn" onClick={() => setRevealed(p => { const n = { ...p }; delete n[o.id]; return n })}><EyeOff size={12} /></button>
                                                        </div>
                                                    ) : (
                                                        <button className="action-btn" onClick={() => revealMut.mutate(o.id)} disabled={revealMut.isPending}>
                                                            <Eye size={12} /> View
                                                        </button>
                                                    )
                                                ) : '—'}
                                            </td>
                                            <td>
                                                {(o.status === 'active' || o.status === 'pending') && (
                                                    <button className="action-btn red" onClick={() => handleCancel(o.id, o.order_number)}
                                                        disabled={cancelMut.isPending}>
                                                        <XCircle size={12} /> Cancel
                                                    </button>
                                                )}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                        <Pagination page={page} totalPages={meta.pages ?? 1} total={meta.total ?? 0} limit={20} onPageChange={setPage} />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
