'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import {
    Cloud, CheckCircle2, XCircle, Cpu, Plus, Pencil, Trash2,
    ToggleLeft, ToggleRight, X, Save, Globe
} from 'lucide-react'

// Adapter type → human-readable label + color
const ADAPTER_META: Record<string, { label: string; color: string }> = {
    'vpm':          { label: 'VPM',         color: '#E6A817' },
    'proxy_cheap':  { label: 'Proxy-Cheap', color: '#7367F0' },
    'sandbox':      { label: 'Sandbox',     color: '#28C76F' },
}

function adapterMeta(adapterType: string) {
    return ADAPTER_META[adapterType] ?? { label: adapterType ?? 'Unknown', color: '#6c757d' }
}

// ─── Add/Edit Provider Modal ──────────────────────────────────────────────────
function ProviderModal({ provider, onClose, onSaved }: {
    provider?: any, onClose: () => void, onSaved: () => void
}) {
    const { success, error: toastError } = useToast()
    const qc = useQueryClient()
    const isEdit = !!provider

    const [name, setName] = useState(provider?.name ?? '')
    const [displayName, setDisplayName] = useState(provider?.display_name ?? '')
    const [adapterType, setAdapterType] = useState(provider?.adapter_type ?? 'vpm')
    const [baseUrl, setBaseUrl] = useState('')
    const [apiKey, setApiKey] = useState('')
    const [priority, setPriority] = useState(provider?.priority ?? 0)
    const [saving, setSaving] = useState(false)

    const handleSave = async () => {
        if (!name.trim()) { toastError('Name is required'); return }
        setSaving(true)
        try {
            const config: Record<string, string> = {}
            if (adapterType === 'vpm') {
                if (baseUrl.trim()) config.base_url = baseUrl.trim()
                if (apiKey.trim()) config.api_key = apiKey.trim()
            }
            const payload = {
                name: name.trim(),
                display_name: displayName.trim() || name.trim(),
                adapter_type: adapterType,
                config: Object.keys(config).length > 0 ? config : (isEdit ? undefined : {}),
                priority,
                ...(isEdit ? { is_active: provider.is_active } : {}),
            }
            if (isEdit) {
                await adminAPI.updateProxyProvider(provider.id, payload)
                success('Provider updated')
            } else {
                await adminAPI.createProxyProvider(payload)
                success('Provider created! Restart proxy-service to activate.')
            }
            qc.invalidateQueries({ queryKey: ['admin-proxy-providers'] })
            onSaved()
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Failed to save provider')
        } finally {
            setSaving(false)
        }
    }

    return (
        <div style={{
            position: 'fixed', inset: 0, zIndex: 1000,
            background: 'rgba(0,0,0,.5)', display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={onClose}>
            <div className="card" onClick={e => e.stopPropagation()}
                style={{ width: 480, maxWidth: '95vw', padding: '1.5rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '1.25rem' }}>
                    <h3 style={{ margin: 0, color: 'var(--text-heading)' }}>
                        {isEdit ? 'Edit Provider' : 'Add Provider'}
                    </h3>
                    <button onClick={onClose} className="topbar-icon-btn"><X size={16} /></button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '.85rem' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem' }}>
                        <div>
                            <label style={labelStyle}>Name (unique)</label>
                            <input value={name} onChange={e => setName(e.target.value)}
                                placeholder="vpm-cz" style={inputStyle} />
                        </div>
                        <div>
                            <label style={labelStyle}>Display Name</label>
                            <input value={displayName} onChange={e => setDisplayName(e.target.value)}
                                placeholder="VPM Czech Republic" style={inputStyle} />
                        </div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '.75rem' }}>
                        <div>
                            <label style={labelStyle}>Adapter Type</label>
                            <select value={adapterType} onChange={e => setAdapterType(e.target.value)} style={inputStyle}>
                                <option value="vpm">VPM</option>
                                <option value="proxy_cheap">Proxy-Cheap</option>
                                <option value="sandbox">Sandbox</option>
                            </select>
                        </div>
                        <div>
                            <label style={labelStyle}>Priority</label>
                            <input type="number" value={priority} onChange={e => setPriority(+e.target.value)}
                                style={inputStyle} />
                        </div>
                    </div>

                    {adapterType === 'vpm' && (
                        <>
                            <div>
                                <label style={labelStyle}>Base URL</label>
                                <input value={baseUrl} onChange={e => setBaseUrl(e.target.value)}
                                    placeholder="https://cz.resvn.net" style={inputStyle} />
                            </div>
                            <div>
                                <label style={labelStyle}>API Key</label>
                                <input value={apiKey} onChange={e => setApiKey(e.target.value)}
                                    placeholder="vpm_live_xxxxx" style={inputStyle} type="password" />
                            </div>
                        </>
                    )}
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '.5rem', marginTop: '1.25rem' }}>
                    <button onClick={onClose} style={{ ...btnStyle, background: 'var(--surface)', color: 'var(--text)' }}>
                        Cancel
                    </button>
                    <button onClick={handleSave} disabled={saving} className="btn-primary"
                        style={{ display: 'flex', alignItems: 'center', gap: '.35rem' }}>
                        <Save size={14} /> {saving ? 'Saving...' : isEdit ? 'Update' : 'Create'}
                    </button>
                </div>

                {!isEdit && (
                    <p style={{ marginTop: '.75rem', fontSize: '.75rem', color: 'var(--text-muted)', lineHeight: 1.5 }}>
                        ⚠️ After creating, restart <code>proxy-service</code> to activate the adapter.
                    </p>
                )}
            </div>
        </div>
    )
}

const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: '.78rem', fontWeight: 600,
    color: 'var(--text-muted)', marginBottom: '.3rem',
}
const inputStyle: React.CSSProperties = {
    width: '100%', padding: '.5rem .65rem',
    background: 'var(--surface-raised)', border: '1px solid var(--border)',
    borderRadius: 'var(--radius)', color: 'var(--text-heading)', fontSize: '.88rem',
}
const btnStyle: React.CSSProperties = {
    padding: '.45rem 1rem', borderRadius: 'var(--radius)',
    border: '1px solid var(--border)', fontWeight: 600, fontSize: '.82rem', cursor: 'pointer',
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function AdminProxyProvidersPage() {
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()
    const qc = useQueryClient()
    const [modal, setModal] = useState<'add' | 'edit' | null>(null)
    const [editProvider, setEditProvider] = useState<any>(null)

    const { data, isLoading, isError } = useQuery({
        queryKey: ['admin-proxy-providers'],
        queryFn: () => adminAPI.listProxyProviders(),
        staleTime: 30_000,
    })

    const providers: any[] = data?.data?.data ?? data?.data ?? []

    const toggleMut = useMutation({
        mutationFn: (id: string) => adminAPI.toggleProxyProvider(id),
        onSuccess: () => { success('Provider toggled'); qc.invalidateQueries({ queryKey: ['admin-proxy-providers'] }) },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Toggle failed'),
    })

    const deleteMut = useMutation({
        mutationFn: (id: string) => adminAPI.deleteProxyProvider(id),
        onSuccess: () => { success('Provider deleted'); qc.invalidateQueries({ queryKey: ['admin-proxy-providers'] }) },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Delete failed'),
    })

    const handleDelete = async (p: any) => {
        const ok = await confirm({
            title: 'Delete Provider',
            message: `Delete "${p.display_name || p.name}"? Products using this provider will stop working.`,
            confirmLabel: 'Delete', variant: 'danger',
        })
        if (ok) deleteMut.mutate(p.id)
    }

    return (
        <AppLayout breadcrumb={[
            { label: 'Admin', href: '/admin' },
            { label: 'Proxy Products', href: '/admin/proxy' },
            { label: 'Providers' },
        ]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Proxy Providers</h1>
                    <p className="page-subtitle">
                        {isLoading ? 'Loading…' : `${providers.length} provider${providers.length !== 1 ? 's' : ''}`}
                    </p>
                </div>
                <button className="btn-primary" onClick={() => { setEditProvider(null); setModal('add') }}
                    style={{ display: 'flex', alignItems: 'center', gap: '.35rem' }}>
                    <Plus size={14} /> Add Provider
                </button>
            </div>

            <div className="card" style={{ padding: 0 }}>
                {isLoading ? (
                    <div className="loading-spinner">Loading…</div>
                ) : isError ? (
                    <div className="empty-state">
                        <XCircle size={40} color="var(--error)" opacity={0.5} />
                        <p>Failed to load providers</p>
                    </div>
                ) : providers.length === 0 ? (
                    <div className="empty-state">
                        <Cloud size={40} opacity={0.25} />
                        <p>No providers registered</p>
                        <button className="btn-primary" onClick={() => setModal('add')}>
                            <Plus size={14} /> Add First Provider
                        </button>
                    </div>
                ) : (
                    <div className="table-wrapper">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>Provider</th>
                                    <th>Adapter</th>
                                    <th>Priority</th>
                                    <th>Status</th>
                                    <th>Created</th>
                                    <th style={{ width: 100 }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {providers.map((p: any) => {
                                    const meta = adapterMeta(p.adapter_type)
                                    return (
                                        <tr key={p.id}>
                                            <td>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem' }}>
                                                    <div style={{
                                                        width: 34, height: 34, borderRadius: 8,
                                                        background: meta.color + '22',
                                                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                                                        color: meta.color, flexShrink: 0,
                                                    }}>
                                                        <Globe size={16} />
                                                    </div>
                                                    <div>
                                                        <p style={{ fontWeight: 600, color: 'var(--text-heading)', lineHeight: 1.2 }}>
                                                            {p.display_name || p.name}
                                                        </p>
                                                        <p style={{ fontSize: '.72rem', color: 'var(--text-muted)', fontFamily: 'monospace' }}>
                                                            {p.name}
                                                        </p>
                                                    </div>
                                                </div>
                                            </td>
                                            <td>
                                                <span style={{
                                                    display: 'inline-flex', alignItems: 'center', gap: '.35rem',
                                                    padding: '.2rem .6rem', borderRadius: 6, fontSize: '.8rem', fontWeight: 600,
                                                    background: meta.color + '1a', color: meta.color,
                                                }}>
                                                    <Cpu size={12} />
                                                    {meta.label}
                                                </span>
                                            </td>
                                            <td>
                                                <span className="badge badge-secondary">
                                                    {p.priority ?? 0}
                                                </span>
                                            </td>
                                            <td>
                                                <button onClick={() => toggleMut.mutate(p.id)}
                                                    style={{ background: 'none', border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '.35rem' }}
                                                    title={p.is_active ? 'Click to deactivate' : 'Click to activate'}>
                                                    {p.is_active ? (
                                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', color: 'var(--success)', fontWeight: 500, fontSize: '.85rem' }}>
                                                            <ToggleRight size={20} /> Active
                                                        </span>
                                                    ) : (
                                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', color: 'var(--text-muted)', fontWeight: 500, fontSize: '.85rem' }}>
                                                            <ToggleLeft size={20} /> Inactive
                                                        </span>
                                                    )}
                                                </button>
                                            </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                {p.created_at
                                                    ? new Date(p.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
                                                    : '—'}
                                            </td>
                                            <td>
                                                <div style={{ display: 'flex', gap: '.3rem' }}>
                                                    <button className="topbar-icon-btn" title="Edit"
                                                        onClick={() => { setEditProvider(p); setModal('edit') }}>
                                                        <Pencil size={14} />
                                                    </button>
                                                    <button className="topbar-icon-btn" title="Delete"
                                                        onClick={() => handleDelete(p)}
                                                        style={{ color: 'var(--error)' }}>
                                                        <Trash2 size={14} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    )
                                })}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Info box */}
            <div className="card" style={{ marginTop: '1.25rem', padding: '1rem 1.25rem', background: 'var(--bg-secondary)', border: '1px solid var(--border-light)' }}>
                <p style={{ fontSize: '.82rem', color: 'var(--text-muted)', lineHeight: 1.6 }}>
                    <strong style={{ color: 'var(--text-heading)' }}>Multi-Region VPM:</strong>{' '}
                    Each VPM location (e.g., CZ, VN, US) should be added as a separate provider with its own{' '}
                    <code style={{ background: 'var(--bg-card)', padding: '.1rem .35rem', borderRadius: 4 }}>base_url</code>.
                    All VPM providers share the same adapter type. Restart{' '}
                    <code style={{ background: 'var(--bg-card)', padding: '.1rem .35rem', borderRadius: 4 }}>proxy-service</code>{' '}
                    after adding/editing providers.
                </p>
            </div>

            {/* Modal */}
            {(modal === 'add' || modal === 'edit') && (
                <ProviderModal
                    provider={modal === 'edit' ? editProvider : undefined}
                    onClose={() => setModal(null)}
                    onSaved={() => setModal(null)}
                />
            )}
        </AppLayout>
    )
}
