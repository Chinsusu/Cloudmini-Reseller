'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { formatVND } from '@/lib/format'
import { Globe, Plus, X, ToggleLeft, ToggleRight, Info, Pencil } from 'lucide-react'

// Proxy-Cheap service catalog — plans differ per service type
type PCService = { id: string, label: string, plans: { id: string, label: string }[], hasPackages?: boolean, isRotating?: boolean }

const PROXY_CHEAP_SERVICES: PCService[] = [
    { id: 'static-residential-ipv4', label: 'Static Residential IPv4 (ISP)', plans: [{ id: 'basic', label: 'Basic' }, { id: 'standard', label: 'Standard' }, { id: 'premium', label: 'Premium' }] },
    { id: 'static-datacenter-ipv4', label: 'Static Datacenter IPv4', plans: [{ id: 'basic', label: 'Basic' }, { id: 'standard', label: 'Standard' }, { id: 'premium', label: 'Premium' }] },
    { id: 'static-datacenter-ipv6', label: 'Static Datacenter IPv6', plans: [], hasPackages: true },
    { id: 'dedicated-mobile', label: 'Dedicated Mobile (Static)', plans: [{ id: 'dedicated', label: 'Dedicated' }] },
    { id: 'rotating-mobile', label: 'Rotating Mobile', plans: [], isRotating: true },
    { id: 'rotating-residential', label: 'Rotating Residential', plans: [], isRotating: true },
]

// ─── Adapter-specific metadata fields ─────────────────────────────────────────
// Country and ISP are NOT here — clients select them when placing an order
function AdapterMetaFields({ adapterType, meta, setMeta }: {
    adapterType: string
    meta: Record<string, string>
    setMeta: (m: Record<string, string>) => void
}) {
    const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
        setMeta({ ...meta, [k]: e.target.value })

    if (adapterType !== 'proxy_cheap') return null

    const selectedService = PROXY_CHEAP_SERVICES.find(s => s.id === (meta.service_id ?? ''))

    return (
        <div style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border-light)', borderRadius: 'var(--radius)', padding: '.85rem', display: 'flex', flexDirection: 'column', gap: '.65rem' }}>
            <p style={{ fontSize: '.8rem', fontWeight: 600, color: 'var(--primary)', display: 'flex', alignItems: 'center', gap: '.3rem', margin: 0 }}>
                <Info size={13} /> Proxy-Cheap — service &amp; plan config (country/ISP selected by client at order time)
            </p>

            {/* Service */}
            <div className="form-group">
                <label style={{ fontSize: '.78rem' }}>Service</label>
                <select className="input" style={{ fontSize: '.85rem' }} value={meta.service_id ?? ''} onChange={e =>
                    setMeta({ ...meta, service_id: e.target.value, plan_id: '', package_id: '' })
                }>
                    <option value="">— Select service —</option>
                    {PROXY_CHEAP_SERVICES.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}
                </select>
            </div>

            {/* Plan — static services with named plans */}
            {selectedService && !selectedService.isRotating && !selectedService.hasPackages && selectedService.plans.length > 0 && (
                <div className="form-group">
                    <label style={{ fontSize: '.78rem' }}>Plan</label>
                    <select className="input" style={{ fontSize: '.85rem' }} value={meta.plan_id ?? ''} onChange={set('plan_id')}>
                        <option value="">— Select plan —</option>
                        {selectedService.plans.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
                    </select>
                </div>
            )}

            {/* Package — IPv6 datacenter */}
            {selectedService?.hasPackages && (
                <div className="form-group">
                    <label style={{ fontSize: '.78rem' }}>Package size</label>
                    <select className="input" style={{ fontSize: '.85rem' }} value={meta.package_id ?? ''} onChange={set('package_id')}>
                        <option value="">— Select package —</option>
                        <option value="50">50 proxies</option>
                        <option value="150">150 proxies</option>
                        <option value="500">500 proxies</option>
                    </select>
                </div>
            )}

            {/* Traffic GB — rotating */}
            {selectedService?.isRotating && (
                <div className="form-group">
                    <label style={{ fontSize: '.78rem' }}>Traffic GB included</label>
                    <input className="input" style={{ fontSize: '.85rem' }} type="number" min="1" placeholder="e.g. 10" value={meta.traffic_gb ?? ''} onChange={set('traffic_gb')} />
                    <p style={{ fontSize: '.73rem', color: 'var(--text-muted)', marginTop: '.2rem' }}>Rotating proxies are billed per GB.</p>
                </div>
            )}

            {/* Period */}
            {selectedService && (
                <div className="form-group">
                    <label style={{ fontSize: '.78rem' }}>Default period (months)</label>
                    <input className="input" style={{ fontSize: '.85rem' }} type="number" min="1" placeholder="1" value={meta.period_months ?? ''} onChange={set('period_months')} />
                </div>
            )}

            <p style={{ fontSize: '.72rem', color: 'var(--text-muted)', margin: 0 }}>
                Country and ISP will be selected by the client when placing an order.
            </p>
        </div>
    )
}

// ─── Add Product Modal ─────────────────────────────────────────────────────────
function AddProductModal({ onClose }: { onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [form, setForm] = useState({ name: '', proxy_type: 'residential', location: '', duration_days: '', bandwidth_gb: '', base_cost: '', provider_ids: [] as string[] })
    const [protocols, setProtocols] = useState<string[]>(['http', 'socks5'])
    const [selectedGroups, setSelectedGroups] = useState<string[]>([])
    const [meta, setMeta] = useState<Record<string, string>>({})
    const { data: providersData } = useQuery({ queryKey: ['admin-proxy-providers'], queryFn: () => adminAPI.listProxyProviders(), staleTime: 60_000 })
    const providers: any[] = providersData?.data?.data ?? providersData?.data ?? []
    const selectedProvider = form.provider_ids.length > 0 ? providers.find(p => p.id === form.provider_ids[0]) : undefined;

    // Fetch groups when VPM provider selected (based on first sub-provider)
    const { data: groupsData, isLoading: groupsLoading } = useQuery({
        queryKey: ['provider-groups', form.provider_ids[0]],
        queryFn: () => adminAPI.getProviderGroups(form.provider_ids[0]),
        enabled: form.provider_ids.length > 0 && selectedProvider?.adapter_type === 'vpm',
        staleTime: 60_000,
    })
    const groups: any[] = groupsData?.data?.data ?? groupsData?.data ?? []

    const toggleProtocol = (p: string) => {
        setProtocols(prev => prev.includes(p) ? prev.filter(x => x !== p) : [...prev, p])
    }
    const toggleGroup = (gid: string) => {
        setSelectedGroups(prev => prev.includes(gid) ? prev.filter(x => x !== gid) : [...prev, gid])
    }

    const mut = useMutation({
        mutationFn: () => {
            const payload: Record<string, any> = {
                name: form.name, proxy_type: form.proxy_type,
                protocol: protocols.join(','),
                location: form.location,
                base_cost: form.base_cost, provider_ids: form.provider_ids,
            }
            if (form.duration_days !== '') payload.duration_days = parseInt(form.duration_days, 10)
            if (form.bandwidth_gb !== '') payload.bandwidth_gb = form.bandwidth_gb
            // Store selected group IDs in metadata
            const m: Record<string, any> = Object.fromEntries(Object.entries(meta).filter(([k, v]) => v !== '' && !k.startsWith('_')))
            if (selectedGroups.length > 0) m.group_ids = selectedGroups.join(',')
            if (Object.keys(m).length > 0) payload.metadata = m
            return adminAPI.createProxyProduct(payload)
        },
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-proxy-products'] }); success('Product created'); onClose() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed to create product'),
    })
    const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setForm(f => ({ ...f, [k]: e.target.value }))

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 580, width: '95vw' }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}><Plus size={17} /> Add Proxy Product</span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body" style={{ maxHeight: '80vh', overflowY: 'auto' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem' }}>
                        <div className="form-group" style={{ gridColumn: '1/-1' }}><label>Product Name</label><input className="input" placeholder="e.g. US Static Residential Basic" value={form.name} onChange={set('name')} /></div>
                        <div className="form-group"><label>Proxy Type</label>
                            <select className="input" value={form.proxy_type} onChange={set('proxy_type')}>
                                <option value="residential">Residential</option><option value="datacenter">Datacenter</option><option value="mobile">Mobile</option>
                            </select>
                        </div>

                        {/* Protocol — checkboxes */}
                        <div className="form-group">
                            <label>Protocol</label>
                            <div style={{ display: 'flex', gap: '.75rem', padding: '.5rem 0' }}>
                                {['http', 'socks5'].map(p => (
                                    <label key={p} style={{ display: 'flex', alignItems: 'center', gap: '.35rem', cursor: 'pointer', fontSize: '.88rem', fontWeight: 500 }}>
                                        <input type="checkbox" checked={protocols.includes(p)} onChange={() => toggleProtocol(p)}
                                            style={{ width: 16, height: 16, accentColor: 'var(--primary)' }} />
                                        {p.toUpperCase()}
                                    </label>
                                ))}
                            </div>
                        </div>

                        <div className="form-group"><label>Location</label><input className="input" placeholder="CZ-HCM-FPT, US, EU..." value={form.location} onChange={set('location')} /></div>
                        <div className="form-group"><label>Base Cost ($)</label><input className="input" type="number" step="0.01" min="0" placeholder="5.00" value={form.base_cost} onChange={set('base_cost')} /></div>
                        <div className="form-group"><label>Duration (days, optional)</label><input className="input" type="number" min="1" placeholder="30" value={form.duration_days} onChange={set('duration_days')} /></div>
                        <div className="form-group"><label>Bandwidth GB (optional)</label><input className="input" type="number" step="0.1" min="0" placeholder="10" value={form.bandwidth_gb} onChange={set('bandwidth_gb')} /></div>

                        {/* Provider */}
                        <div className="form-group" style={{ gridColumn: '1/-1' }}>
                            <label>Providers (Select one or more for load balancing)</label>
                            {providers.length === 0
                                ? <div style={{ padding: '.6rem .75rem', borderRadius: 'var(--radius)', background: 'var(--bg-secondary)', border: '1px solid var(--border-light)', fontSize: '.85rem', color: 'var(--text-muted)' }}>No providers — add one in <strong>Proxy Providers</strong> first.</div>
                                : <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem', maxHeight: '150px', overflowY: 'auto', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: '0.5rem' }}>
                                    {providers.map((p: any) => (
                                        <label key={p.id} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.85rem' }}>
                                            <input type="checkbox" checked={form.provider_ids.includes(p.id)} onChange={(e) => {
                                                if (e.target.checked) setForm(f => ({...f, provider_ids: [...f.provider_ids, p.id]}));
                                                else setForm(f => ({...f, provider_ids: f.provider_ids.filter(id => id !== p.id)}));
                                                setMeta({}); setSelectedGroups([]);
                                            }} />
                                            {p.display_name || p.name} ({p.adapter_type})
                                        </label>
                                    ))}
                                </div>
                            }
                        </div>

                        {/* Groups — VPM providers */}
                        {selectedProvider?.adapter_type === 'vpm' && (
                            <div className="form-group" style={{ gridColumn: '1/-1' }}>
                                <label>Groups</label>
                                {groupsLoading ? (
                                    <div style={{ padding: '.5rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>Loading groups...</div>
                                ) : groups.length === 0 ? (
                                    <div style={{ padding: '.5rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>No groups available</div>
                                ) : (
                                    <div style={{
                                        background: 'var(--bg-secondary)', border: '1px solid var(--border-light)',
                                        borderRadius: 'var(--radius)', padding: '.6rem .75rem',
                                        display: 'flex', flexWrap: 'wrap', gap: '.5rem',
                                    }}>
                                        {groups.map((g: any) => (
                                            <label key={g.id} style={{
                                                display: 'flex', alignItems: 'center', gap: '.35rem', cursor: 'pointer',
                                                padding: '.35rem .65rem', borderRadius: 6, fontSize: '.82rem', fontWeight: 500,
                                                background: selectedGroups.includes(g.id) ? 'var(--primary)' : 'var(--surface-raised)',
                                                color: selectedGroups.includes(g.id) ? '#fff' : 'var(--text)',
                                                border: '1px solid ' + (selectedGroups.includes(g.id) ? 'var(--primary)' : 'var(--border)'),
                                                transition: 'all .15s',
                                            }}>
                                                <input type="checkbox" checked={selectedGroups.includes(g.id)} onChange={() => toggleGroup(g.id)}
                                                    style={{ display: 'none' }} />
                                                {g.name}
                                                {g.available_ips > 0 && <span style={{ fontSize: '.7rem', opacity: .7 }}>({g.available_ips} IPs)</span>}
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        )}

                        {/* Provider Plan Configuration */}
                        {selectedProvider && (
                            <div style={{ gridColumn: '1/-1' }}>
                                <label style={{ marginBottom: '.4rem', display: 'block' }}>Provider Plan Configuration</label>
                                <AdapterMetaFields adapterType={selectedProvider.adapter_type} meta={meta} setMeta={setMeta} />
                            </div>
                        )}
                    </div>
                    <button className="btn-primary" style={{ width: '100%', marginTop: '.75rem' }}
                        onClick={() => mut.mutate()} disabled={!form.name || !form.base_cost || form.provider_ids.length === 0 || protocols.length === 0 || mut.isPending}>
                        {mut.isPending ? 'Creating...' : 'Create Product'}
                    </button>
                </div>
            </div>
        </div>
    )
}


// ─── Edit Product Modal ────────────────────────────────────────────────────────
function EditProductModal({ product, providers, onClose }: { product: any, providers: any[], onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()

    // Pre-populate metadata from product
    const initialMeta: Record<string, string> = (() => {
        try { return typeof product.metadata === 'object' && product.metadata ? Object.fromEntries(Object.entries(product.metadata).map(([k, v]) => [k, String(v)])) : {} }
        catch { return {} }
    })()

    const [form, setForm] = useState({
        name: product.name ?? '', proxy_type: product.proxy_type ?? 'residential',
        location: product.location ?? '', duration_days: product.duration_days?.toString() ?? '',
        bandwidth_gb: product.bandwidth_gb?.toString() ?? '', base_cost: parseFloat(product.base_cost).toFixed(2),
        provider_ids: Array.isArray(product.provider_ids) ? product.provider_ids : (product.provider_id ? [product.provider_id] : []) as string[],
    })
    const [protocols, setProtocols] = useState<string[]>(() => {
        const p = product.protocol ?? 'http,socks5'
        return p.split(',').map((s: string) => s.trim()).filter(Boolean)
    })
    const [selectedGroups, setSelectedGroups] = useState<string[]>(() => {
        const gids = initialMeta.group_ids ?? ''
        return gids ? gids.split(',').filter(Boolean) : []
    })
    const [meta, setMeta] = useState<Record<string, string>>(initialMeta)

    const selectedProvider = form.provider_ids.length > 0 ? providers.find((p: any) => p.id === form.provider_ids[0]) : undefined

    // Fetch groups only for VPM providers
    const { data: groupsData, isLoading: groupsLoading } = useQuery({
        queryKey: ['provider-groups', form.provider_ids[0]],
        queryFn: () => adminAPI.getProviderGroups(form.provider_ids[0]),
        enabled: form.provider_ids.length > 0 && selectedProvider?.adapter_type === 'vpm',
        staleTime: 60_000,
    })
    const groups: any[] = groupsData?.data?.data ?? groupsData?.data ?? []

    const toggleProtocol = (p: string) => {
        setProtocols(prev => prev.includes(p) ? prev.filter(x => x !== p) : [...prev, p])
    }
    const toggleGroup = (gid: string) => {
        setSelectedGroups(prev => prev.includes(gid) ? prev.filter(x => x !== gid) : [...prev, gid])
    }

    const mut = useMutation({
        mutationFn: () => {
            const payload: Record<string, any> = {
                name: form.name, proxy_type: form.proxy_type,
                protocol: protocols.join(','),
                location: form.location,
                base_cost: form.base_cost, provider_ids: form.provider_ids,
            }
            if (form.duration_days !== '') payload.duration_days = parseInt(form.duration_days, 10)
            if (form.bandwidth_gb !== '') payload.bandwidth_gb = form.bandwidth_gb
            const m: Record<string, any> = Object.fromEntries(Object.entries(meta).filter(([k, v]) => v !== '' && !k.startsWith('_')))
            if (selectedGroups.length > 0) m.group_ids = selectedGroups.join(',')
            if (Object.keys(m).length > 0) payload.metadata = m
            return adminAPI.updateProxyProduct(product.id, payload)
        },
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-proxy-products'] }); success('Product updated'); onClose() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed to update product'),
    })
    const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setForm(f => ({ ...f, [k]: e.target.value }))

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 580, width: '95vw' }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}><Pencil size={17} /> Edit Product</span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body" style={{ maxHeight: '80vh', overflowY: 'auto' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem' }}>
                        <div className="form-group" style={{ gridColumn: '1/-1' }}><label>Product Name</label><input className="input" value={form.name} onChange={set('name')} /></div>
                        <div className="form-group"><label>Proxy Type</label>
                            <select className="input" value={form.proxy_type} onChange={set('proxy_type')}>
                                <option value="residential">Residential</option><option value="datacenter">Datacenter</option><option value="mobile">Mobile</option>
                            </select>
                        </div>

                        {/* Protocol — checkboxes */}
                        <div className="form-group">
                            <label>Protocol</label>
                            <div style={{ display: 'flex', gap: '.75rem', padding: '.5rem 0' }}>
                                {['http', 'socks5'].map(p => (
                                    <label key={p} style={{ display: 'flex', alignItems: 'center', gap: '.35rem', cursor: 'pointer', fontSize: '.88rem', fontWeight: 500 }}>
                                        <input type="checkbox" checked={protocols.includes(p)} onChange={() => toggleProtocol(p)}
                                            style={{ width: 16, height: 16, accentColor: 'var(--primary)' }} />
                                        {p.toUpperCase()}
                                    </label>
                                ))}
                            </div>
                        </div>

                        <div className="form-group"><label>Location</label><input className="input" placeholder="CZ-HCM-FPT, US, EU..." value={form.location} onChange={set('location')} /></div>
                        <div className="form-group"><label>Base Cost ($)</label><input className="input" type="number" step="0.01" min="0" value={form.base_cost} onChange={set('base_cost')} /></div>
                        <div className="form-group"><label>Duration (days)</label><input className="input" type="number" min="1" placeholder="—" value={form.duration_days} onChange={set('duration_days')} /></div>
                        <div className="form-group"><label>Bandwidth GB</label><input className="input" type="number" step="0.1" min="0" placeholder="—" value={form.bandwidth_gb} onChange={set('bandwidth_gb')} /></div>

                        {/* Provider */}
                        <div className="form-group" style={{ gridColumn: '1/-1' }}>
                            <label>Providers (Select one or more for load balancing)</label>
                            {providers.length === 0
                                ? <div style={{ padding: '.6rem .75rem', borderRadius: 'var(--radius)', background: 'var(--bg-secondary)', border: '1px solid var(--border-light)', fontSize: '.85rem', color: 'var(--text-muted)' }}>No providers — add one in <strong>Proxy Providers</strong> first.</div>
                                : <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem', maxHeight: '150px', overflowY: 'auto', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: '0.5rem' }}>
                                    {providers.map((p: any) => (
                                        <label key={p.id} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.85rem' }}>
                                            <input type="checkbox" checked={form.provider_ids.includes(p.id)} onChange={(e) => {
                                                if (e.target.checked) setForm(f => ({...f, provider_ids: [...f.provider_ids, p.id]}));
                                                else setForm(f => ({...f, provider_ids: f.provider_ids.filter(id => id !== p.id)}));
                                                setMeta({}); setSelectedGroups([]);
                                            }} />
                                            {p.display_name || p.name} ({p.adapter_type})
                                        </label>
                                    ))}
                                </div>
                            }
                        </div>

                        {/* Groups — VPM providers only */}
                        {selectedProvider?.adapter_type === 'vpm' && (
                            <div className="form-group" style={{ gridColumn: '1/-1' }}>
                                <label>Groups</label>
                                {groupsLoading ? (
                                    <div style={{ padding: '.5rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>Loading groups...</div>
                                ) : groups.length === 0 ? (
                                    <div style={{ padding: '.5rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>No groups available</div>
                                ) : (
                                    <div style={{
                                        background: 'var(--bg-secondary)', border: '1px solid var(--border-light)',
                                        borderRadius: 'var(--radius)', padding: '.6rem .75rem',
                                        display: 'flex', flexWrap: 'wrap', gap: '.5rem',
                                    }}>
                                        {groups.map((g: any) => (
                                            <label key={g.id} style={{
                                                display: 'flex', alignItems: 'center', gap: '.35rem', cursor: 'pointer',
                                                padding: '.35rem .65rem', borderRadius: 6, fontSize: '.82rem', fontWeight: 500,
                                                background: selectedGroups.includes(g.id) ? 'var(--primary)' : 'var(--surface-raised)',
                                                color: selectedGroups.includes(g.id) ? '#fff' : 'var(--text)',
                                                border: '1px solid ' + (selectedGroups.includes(g.id) ? 'var(--primary)' : 'var(--border)'),
                                                transition: 'all .15s',
                                            }}>
                                                <input type="checkbox" checked={selectedGroups.includes(g.id)} onChange={() => toggleGroup(g.id)}
                                                    style={{ display: 'none' }} />
                                                {g.name}
                                                {g.available_ips > 0 && <span style={{ fontSize: '.7rem', opacity: .7 }}>({g.available_ips} IPs)</span>}
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        )}

                        {/* Provider Plan Configuration */}
                        {selectedProvider && (
                            <div style={{ gridColumn: '1/-1' }}>
                                <label style={{ marginBottom: '.4rem', display: 'block' }}>Provider Plan Configuration</label>
                                <AdapterMetaFields adapterType={selectedProvider.adapter_type} meta={meta} setMeta={setMeta} />
                            </div>
                        )}
                    </div>
                    <button className="btn-primary" style={{ width: '100%', marginTop: '.75rem' }}
                        onClick={() => mut.mutate()} disabled={!form.name || !form.base_cost || form.provider_ids.length === 0 || protocols.length === 0 || mut.isPending}>
                        {mut.isPending ? 'Saving...' : 'Save Changes'}
                    </button>
                </div>
            </div>
        </div>
    )
}


// ─── Main Page ─────────────────────────────────────────────────────────────────
export default function AdminProxyPage() {
    const [page, setPage] = useState(1)
    const [showAdd, setShowAdd] = useState(false)
    const [editProduct, setEditProduct] = useState<any>(null)
    const [filter, setFilter] = useState<'available' | 'hidden'>('available')
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()

    const { data, isLoading } = useQuery({ queryKey: ['admin-proxy-products', page], queryFn: () => adminAPI.listProxyProducts(page) })
    const allProducts = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    // Client-side filter by is_active
    const products = allProducts.filter((p: any) => filter === 'available' ? p.is_active : !p.is_active)
    const availableCount = allProducts.filter((p: any) => p.is_active).length
    const hiddenCount = allProducts.filter((p: any) => !p.is_active).length

    const { data: providersData } = useQuery({ queryKey: ['admin-proxy-providers'], queryFn: () => adminAPI.listProxyProviders(), staleTime: 60_000 })
    const providers: any[] = providersData?.data?.data ?? providersData?.data ?? []
    const providerMap = Object.fromEntries(providers.map((p: any) => [p.id, p]))

    const toggleMut = useMutation({
        mutationFn: (id: string) => adminAPI.toggleProxyProduct(id),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-proxy-products'] }); success('Status updated') },
        onError: () => toastError('Failed to update'),
    })

    return (
        <AppLayout breadcrumb={[{ label: 'Admin', href: '/admin' }, { label: 'Proxy Products' }]}>
            {showAdd && <AddProductModal onClose={() => setShowAdd(false)} />}
            {editProduct && <EditProductModal product={editProduct} providers={providers} onClose={() => setEditProduct(null)} />}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Proxy Products</h1>
                    <p className="page-subtitle">{meta.total ?? 0} products</p>
                </div>
                <button className="btn-primary" onClick={() => setShowAdd(true)}><Plus size={14} /> Add Product</button>
            </div>

            {/* Filter tabs */}
            <div style={{ display: 'flex', gap: '.4rem', marginBottom: '1rem' }}>
                <button
                    onClick={() => setFilter('available')}
                    style={{
                        padding: '.45rem .9rem', borderRadius: 'var(--radius)', border: '1px solid var(--border)',
                        fontWeight: 600, fontSize: '.82rem', cursor: 'pointer',
                        background: filter === 'available' ? 'var(--primary)' : 'var(--surface)',
                        color: filter === 'available' ? '#fff' : 'var(--text)',
                    }}>
                    Available ({availableCount})
                </button>
                <button
                    onClick={() => setFilter('hidden')}
                    style={{
                        padding: '.45rem .9rem', borderRadius: 'var(--radius)', border: '1px solid var(--border)',
                        fontWeight: 600, fontSize: '.82rem', cursor: 'pointer',
                        background: filter === 'hidden' ? 'var(--text-muted)' : 'var(--surface)',
                        color: filter === 'hidden' ? '#fff' : 'var(--text)',
                    }}>
                    Hidden ({hiddenCount})
                </button>
            </div>

            <div className="card" style={{ padding: 0 }}>
                {isLoading ? <div className="loading-spinner">Loading...</div>
                    : products.length === 0 ? (
                        <div className="empty-state">
                            <Globe size={40} opacity={0.3} />
                            <p>{filter === 'available' ? 'No active products' : 'No hidden products'}</p>
                            {filter === 'available' && (
                                <button className="btn-primary" onClick={() => setShowAdd(true)}><Plus size={14} /> Add first product</button>
                            )}
                        </div>
                    ) : (
                        <>
                            <div className="table-wrapper">
                                <table className="data-table">
                                    <thead>
                                        <tr>
                                            <th>Name</th>
                                            <th>Provider</th>
                                            <th>Type</th>
                                            <th>Protocol</th>
                                            <th>Location</th>
                                            <th>Duration</th>
                                            <th>Bandwidth</th>
                                            <th>Cost</th>
                                            <th>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {products.map((p: any) => {
                                            const provider = providerMap[p.provider_id]
                                            return (
                                                <tr key={p.id} style={{ opacity: p.is_active ? 1 : 0.55 }}>
                                                    <td><strong>{p.name}</strong></td>
                                                    <td>
                                                        {provider ? (
                                                            <span style={{ fontSize: '.82rem' }}>
                                                                <strong>{provider.display_name || provider.name}</strong>
                                                                <br /><span style={{ color: 'var(--text-muted)', fontFamily: 'monospace', fontSize: '.75rem' }}>{provider.adapter_type}</span>
                                                            </span>
                                                        ) : <span style={{ color: 'var(--text-muted)', fontSize: '.8rem', fontFamily: 'monospace' }}>{p.provider_id?.slice(0, 8)}…</span>}
                                                    </td>
                                                    <td><span className="badge badge-info">{p.proxy_type}</span></td>
                                                    <td><span className="badge badge-secondary">{p.protocol?.toUpperCase()}</span></td>
                                                    <td>{p.location || '—'}</td>
                                                    <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>{p.duration_days ? `${p.duration_days}d` : '—'}</td>
                                                    <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>{p.bandwidth_gb ? `${p.bandwidth_gb} GB` : '—'}</td>
                                                    <td><strong>{formatVND(p.base_cost)}</strong></td>
                                                    <td>
                                                        <div style={{ display: 'flex', gap: '.3rem' }}>
                                                            <button className="action-btn" onClick={() => setEditProduct(p)} title="Edit" style={{ color: 'var(--primary)' }}><Pencil size={15} /></button>
                                                            <button className="action-btn" onClick={() => toggleMut.mutate(p.id)}
                                                                disabled={toggleMut.isPending}
                                                                title={p.is_active ? 'Hide product' : 'Show product'}
                                                                style={{ color: p.is_active ? 'var(--text-muted)' : 'var(--success)', display: 'flex', alignItems: 'center', gap: '.25rem', fontSize: '.8rem' }}>
                                                                {p.is_active ? <><ToggleLeft size={16} /> Hide</> : <><ToggleRight size={16} /> Show</>}
                                                            </button>
                                                        </div>
                                                    </td>
                                                </tr>
                                            )
                                        })}
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

