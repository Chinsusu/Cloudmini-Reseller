'use client'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { Key, Plus, Trash2, Copy } from 'lucide-react'
import { useState } from 'react'

export default function ResellerAPIKeysPage() {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()
    const [newKeyResult, setNewKeyResult] = useState<string | null>(null)
    const [name, setName] = useState('')

    const { data, isLoading } = useQuery({
        queryKey: ['api-keys'],
        queryFn: () => resellerAPI.listAPIKeys(),
    })
    const keys = data?.data?.data ?? []

    const create = useMutation({
        mutationFn: () => resellerAPI.createAPIKey(name, ['read', 'write']),
        onSuccess: (res) => {
            setNewKeyResult(res.data.data.key)
            setName('')
            qc.invalidateQueries({ queryKey: ['api-keys'] })
            success('API key created — copy it now!')
        },
        onError: () => toastError('Failed to create API key'),
    })

    const revoke = useMutation({
        mutationFn: (id: string) => resellerAPI.revokeAPIKey(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['api-keys'] })
            success('API key revoked')
        },
        onError: () => toastError('Failed to revoke key'),
    })

    const handleRevoke = async (id: string, keyName: string) => {
        const ok = await confirm({
            title: 'Revoke API Key',
            message: `Revoke "${keyName}"? This action cannot be undone.`,
            confirmLabel: 'Revoke',
            variant: 'danger',
        })
        if (ok) revoke.mutate(id)
    }

    const handleCopy = (text: string) => {
        navigator.clipboard.writeText(text)
        success('Copied to clipboard')
    }

    return (
        <AppLayout breadcrumb={[
            { label: 'Reseller', href: '/reseller' },
            { label: 'API Keys' },
        ]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">API Keys</h1>
                    <p className="page-subtitle">Manage programmatic access to the platform</p>
                </div>
            </div>

            {/* New key created banner */}
            {newKeyResult && (
                <div style={{ marginBottom: '1.5rem', padding: '1rem 1.25rem', background: 'var(--surface)', border: '1px solid var(--border)', borderLeft: '4px solid var(--dc-gold)', borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-sm)' }}>
                    <p style={{ fontWeight: 700, color: 'var(--dc-gold)', marginBottom: '.5rem', fontSize: '.875rem' }}>API Key created! Copy it now — it won't be shown again.</p>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem', background: 'var(--bg)', borderRadius: 8, padding: '.5rem .75rem', border: '1px solid var(--border)' }}>
                        <code style={{ flex: 1, fontSize: '.78rem', fontFamily: 'monospace', wordBreak: 'break-all', color: 'var(--text-heading)' }}>{newKeyResult}</code>
                        <button className="action-btn" style={{ color: 'var(--dc-gold)' }} onClick={() => handleCopy(newKeyResult)} title="Copy">
                            <Copy size={13} /> Copy
                        </button>
                    </div>
                    <button style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: '.78rem', marginTop: '.5rem', padding: 0 }} onClick={() => setNewKeyResult(null)}>
                        Dismiss
                    </button>
                </div>
            )}

            {/* Create form */}
            <div className="card">
                <div className="card-header"><Key size={17} /> Create New API Key</div>
                <div className="form-row">
                    <input
                        className="input"
                        placeholder="Key name (e.g. Production App)"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && name && create.mutate()}
                    />
                    <button
                        className="btn-primary"
                        onClick={() => create.mutate()}
                        disabled={!name || create.isPending}
                    >
                        <Plus size={16} />{create.isPending ? 'Generating...' : 'Generate'}
                    </button>
                </div>
            </div>

            {/* Keys list */}
            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
                    <Key size={17} /> Your API Keys ({keys.length})
                </div>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : keys.length === 0 ? (
                    <div className="empty-state">
                        <Key size={40} opacity={0.3} />
                        <p>No API keys yet</p>
                        <p style={{ fontSize: '.8rem' }}>Create your first key to integrate with the platform</p>
                    </div>
                ) : (
                    <div className="table-wrapper">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Prefix</th>
                                    <th>Scopes</th>
                                    <th>Last Used</th>
                                    <th>Expires</th>
                                    <th>Status</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {keys.map((k: any) => (
                                    <tr key={k.id}>
                                        <td style={{ fontWeight: 600, color: 'var(--text-heading)' }}>{k.name}</td>
                                        <td><code className="font-mono">{k.key_prefix}...</code></td>
                                        <td>
                                            {(k.scopes ?? []).map((s: string) => (
                                                <span key={s} className="badge badge-secondary" style={{ marginRight: 4 }}>{s}</span>
                                            ))}
                                        </td>
                                        <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                            {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : 'Never'}
                                        </td>
                                        <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                            {k.expires_at ? new Date(k.expires_at).toLocaleDateString() : '∞'}
                                        </td>
                                        <td>
                                            <span className={`badge ${k.revoked_at ? 'badge-error' : 'badge-success'}`}>
                                                {k.revoked_at ? 'Revoked' : 'Active'}
                                            </span>
                                        </td>
                                        <td>
                                            {!k.revoked_at && (
                                                <button
                                                    className="action-btn red"
                                                    onClick={() => handleRevoke(k.id, k.name)}
                                                    disabled={revoke.isPending}
                                                    title="Revoke"
                                                >
                                                    <Trash2 size={13} /> Revoke
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </AppLayout>
    )
}
