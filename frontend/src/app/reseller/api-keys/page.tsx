'use client'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { Sidebar } from '@/components/layout/Sidebar'
import { Key, Plus, Trash2, Copy } from 'lucide-react'
import { useState } from 'react'

export default function ResellerAPIKeysPage() {
    const qc = useQueryClient()
    const [newKeyResult, setNewKeyResult] = useState<string | null>(null)
    const [name, setName] = useState('')

    const { data } = useQuery({
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
        },
    })

    const revoke = useMutation({
        mutationFn: (id: string) => resellerAPI.revokeAPIKey(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['api-keys'] }),
    })

    return (
        <div className="page-layout">
            <Sidebar />
            <main className="page-main">
                <div className="page-header">
                    <h1 className="page-title">API Keys</h1>
                </div>

                {/* New key banner */}
                {newKeyResult && (
                    <div className="alert alert-success">
                        <p><strong>API Key created! Copy it now — it won't be shown again.</strong></p>
                        <div className="key-display">
                            <code>{newKeyResult}</code>
                            <button onClick={() => navigator.clipboard.writeText(newKeyResult)}>
                                <Copy size={14} />
                            </button>
                        </div>
                        <button className="btn-ghost" onClick={() => setNewKeyResult(null)}>Dismiss</button>
                    </div>
                )}

                {/* Create form */}
                <div className="card">
                    <div className="card-header"><Key size={18} /> Create New API Key</div>
                    <div className="form-row">
                        <input
                            className="input"
                            placeholder="Key name (e.g. Production App)"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                        />
                        <button
                            className="btn-primary"
                            onClick={() => create.mutate()}
                            disabled={!name || create.isPending}
                        >
                            <Plus size={16} /> Generate
                        </button>
                    </div>
                </div>

                {/* Keys list */}
                <div className="card">
                    <div className="table-wrapper">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>Name</th><th>Prefix</th><th>Last Used</th>
                                    <th>Expires</th><th>Status</th><th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {keys.map((k: any) => (
                                    <tr key={k.id}>
                                        <td><strong>{k.name}</strong></td>
                                        <td><code>{k.key_prefix}...</code></td>
                                        <td>{k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : 'Never'}</td>
                                        <td>{k.expires_at ? new Date(k.expires_at).toLocaleDateString() : '∞'}</td>
                                        <td>
                                            <span className={`badge ${k.revoked_at ? 'badge-error' : 'badge-success'}`}>
                                                {k.revoked_at ? 'Revoked' : 'Active'}
                                            </span>
                                        </td>
                                        <td>
                                            {!k.revoked_at && (
                                                <button
                                                    className="action-btn red"
                                                    onClick={() => { if (confirm('Revoke this key?')) revoke.mutate(k.id) }}
                                                    title="Revoke"
                                                >
                                                    <Trash2 size={14} />
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            </main>
        </div>
    )
}
