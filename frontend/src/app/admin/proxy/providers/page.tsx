'use client'
import { useQuery } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Cloud, CheckCircle2, XCircle, Cpu } from 'lucide-react'

// Adapter type → human-readable label + color
const ADAPTER_META: Record<string, { label: string; color: string }> = {
    'proxy_cheap':  { label: 'Proxy-Cheap',  color: '#7367F0' },
    'sandbox':      { label: 'Sandbox',       color: '#28C76F' },
    'luminati':     { label: 'Luminati',      color: '#00CFE8' },
    'brightdata':   { label: 'BrightData',    color: '#FF9F43' },
}

function adapterMeta(adapterType: string) {
    return ADAPTER_META[adapterType] ?? { label: adapterType ?? 'Unknown', color: '#6c757d' }
}

export default function AdminProxyProvidersPage() {
    const { data, isLoading, isError } = useQuery({
        queryKey: ['admin-proxy-providers'],
        queryFn: () => adminAPI.listProxyProviders(),
        staleTime: 30_000,
    })

    const providers: any[] = data?.data?.data ?? data?.data ?? []

    return (
        <AppLayout breadcrumb={[
            { label: 'Admin', href: '/admin' },
            { label: 'Proxy Products', href: '/admin/proxy' },
            { label: 'Providers' },
        ]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Proxy Providers</h1>
                    <p className="page-subtitle">
                        {isLoading ? 'Loading…' : `${providers.length} active provider${providers.length !== 1 ? 's' : ''}`}
                    </p>
                </div>
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
                        <p>No active providers registered</p>
                        <p style={{ fontSize: '.82rem', color: 'var(--text-muted)', maxWidth: 340, textAlign: 'center' }}>
                            Insert a row into <code>proxy.providers</code> and restart the service with the appropriate env vars.
                        </p>
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
                                                        <Cloud size={16} />
                                                    </div>
                                                    <div>
                                                        <p style={{ fontWeight: 600, color: 'var(--text-heading)', lineHeight: 1.2 }}>
                                                            {p.name || meta.label}
                                                        </p>
                                                        <p style={{ fontSize: '.75rem', color: 'var(--text-muted)', fontFamily: 'monospace' }}>
                                                            {p.id}
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
                                                {p.is_active ? (
                                                    <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', color: 'var(--success)', fontWeight: 500, fontSize: '.85rem' }}>
                                                        <CheckCircle2 size={15} /> Active
                                                    </span>
                                                ) : (
                                                    <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', color: 'var(--text-muted)', fontWeight: 500, fontSize: '.85rem' }}>
                                                        <XCircle size={15} /> Inactive
                                                    </span>
                                                )}
                                            </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                {p.created_at
                                                    ? new Date(p.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
                                                    : '—'}
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
                    <strong style={{ color: 'var(--text-heading)' }}>How to add a provider:</strong>{' '}
                    Insert a row into <code style={{ background: 'var(--bg-card)', padding: '.1rem .35rem', borderRadius: 4 }}>proxy.providers</code> with the correct{' '}
                    <code style={{ background: 'var(--bg-card)', padding: '.1rem .35rem', borderRadius: 4 }}>adapter_type</code>, then set the corresponding env vars
                    (e.g. <code style={{ background: 'var(--bg-card)', padding: '.1rem .35rem', borderRadius: 4 }}>PROXY_CHEAP_API_KEY</code>) and restart proxy-service.
                </p>
            </div>
        </AppLayout>
    )
}
