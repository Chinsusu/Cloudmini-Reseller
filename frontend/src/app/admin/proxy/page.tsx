'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { Globe, Plus, X, ToggleLeft, ToggleRight } from 'lucide-react'

function AddProductModal({ onClose }: { onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [form, setForm] = useState({
        name: '', proxy_type: 'residential', protocol: 'http',
        location: '', duration_days: '', bandwidth_gb: '', base_cost: '', provider_id: '',
    })

    const mut = useMutation({
        mutationFn: () => adminAPI.createProxyProduct(form),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-proxy-products'] }); success('Product created'); onClose() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed to create product'),
    })

    const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
        setForm(f => ({ ...f, [k]: e.target.value }))

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 560, width: '95vw' }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}><Plus size={17} /> Add Proxy Product</span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body">
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem' }}>
                        <div className="form-group" style={{ gridColumn: '1/-1' }}>
                            <label>Product Name</label>
                            <input className="input" placeholder="e.g. US Residential HTTP" value={form.name} onChange={set('name')} />
                        </div>
                        <div className="form-group">
                            <label>Proxy Type</label>
                            <select className="input" value={form.proxy_type} onChange={set('proxy_type')}>
                                <option value="residential">Residential</option>
                                <option value="datacenter">Datacenter</option>
                                <option value="mobile">Mobile</option>
                            </select>
                        </div>
                        <div className="form-group">
                            <label>Protocol</label>
                            <select className="input" value={form.protocol} onChange={set('protocol')}>
                                <option value="http">HTTP</option>
                                <option value="socks5">SOCKS5</option>
                            </select>
                        </div>
                        <div className="form-group">
                            <label>Location</label>
                            <input className="input" placeholder="US, VN, EU..." value={form.location} onChange={set('location')} />
                        </div>
                        <div className="form-group">
                            <label>Base Cost ($)</label>
                            <input className="input" type="number" step="0.01" min="0" placeholder="5.00" value={form.base_cost} onChange={set('base_cost')} />
                        </div>
                        <div className="form-group">
                            <label>Duration (days, optional)</label>
                            <input className="input" type="number" min="1" placeholder="30" value={form.duration_days} onChange={set('duration_days')} />
                        </div>
                        <div className="form-group">
                            <label>Bandwidth GB (optional)</label>
                            <input className="input" type="number" step="0.1" min="0" placeholder="10" value={form.bandwidth_gb} onChange={set('bandwidth_gb')} />
                        </div>
                        <div className="form-group" style={{ gridColumn: '1/-1' }}>
                            <label>Provider ID (UUID)</label>
                            <input className="input" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" value={form.provider_id} onChange={set('provider_id')} />
                        </div>
                    </div>
                    <button className="btn-primary" style={{ width: '100%', marginTop: '.5rem' }}
                        onClick={() => mut.mutate()} disabled={!form.name || !form.base_cost || mut.isPending}>
                        {mut.isPending ? 'Creating...' : 'Create Product'}
                    </button>
                </div>
            </div>
        </div>
    )
}

export default function AdminProxyPage() {
    const [page, setPage] = useState(1)
    const [showModal, setShowModal] = useState(false)
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()

    const { data, isLoading } = useQuery({
        queryKey: ['admin-proxy-products', page],
        queryFn: () => adminAPI.listProxyProducts(page),
    })
    const products = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const toggleMut = useMutation({
        mutationFn: (id: string) => adminAPI.toggleProxyProduct(id),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-proxy-products'] }); success('Updated') },
        onError: () => toastError('Failed to update'),
    })

    return (
        <AppLayout breadcrumb={[{ label: 'Admin', href: '/admin' }, { label: 'Proxy Products' }]}>
            {showModal && <AddProductModal onClose={() => setShowModal(false)} />}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Proxy Products</h1>
                    <p className="page-subtitle">{meta.total ?? 0} products</p>
                </div>
                <button className="btn-primary" onClick={() => setShowModal(true)}>
                    <Plus size={14} /> Add Product
                </button>
            </div>

            <div className="card" style={{ padding: 0 }}>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : products.length === 0 ? (
                    <div className="empty-state">
                        <Globe size={40} opacity={0.3} />
                        <p>No proxy products yet</p>
                        <button className="btn-primary" onClick={() => setShowModal(true)}><Plus size={14} /> Add first product</button>
                    </div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Name</th>
                                        <th>Type</th>
                                        <th>Protocol</th>
                                        <th>Location</th>
                                        <th>Duration</th>
                                        <th>Bandwidth</th>
                                        <th>Cost</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {products.map((p: any) => (
                                        <tr key={p.id}>
                                            <td><strong>{p.name}</strong></td>
                                            <td><span className="badge badge-info">{p.proxy_type}</span></td>
                                            <td><span className="badge badge-secondary">{p.protocol?.toUpperCase()}</span></td>
                                            <td>{p.location}</td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>
                                                {p.duration_days ? `${p.duration_days}d` : '—'}
                                            </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>
                                                {p.bandwidth_gb ? `${p.bandwidth_gb} GB` : '—'}
                                            </td>
                                            <td><strong>${parseFloat(p.base_cost).toFixed(2)}</strong></td>
                                            <td>
                                                <button
                                                    className="action-btn"
                                                    onClick={() => toggleMut.mutate(p.id)}
                                                    disabled={toggleMut.isPending}
                                                    style={{ color: p.is_active ? 'var(--success)' : 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '.3rem' }}
                                                    title={p.is_active ? 'Click to deactivate' : 'Click to activate'}
                                                >
                                                    {p.is_active ? <ToggleRight size={18} /> : <ToggleLeft size={18} />}
                                                    {p.is_active ? 'Active' : 'Inactive'}
                                                </button>
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
