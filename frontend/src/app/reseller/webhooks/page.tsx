'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { Webhook, Plus, Trash2, Link } from 'lucide-react'

export default function ResellerWebhooksPage() {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const [url, setUrl] = useState('')
    const [secret, setSecret] = useState('')
    const [events, setEvents] = useState<string[]>(['order.created', 'order.status_changed'])

    const AVAILABLE_EVENTS = [
        'order.created', 'order.status_changed', 'order.cancelled',
        'payment.completed', 'payment.failed',
        'vps.created', 'vps.status_changed',
        'reseller.approved', 'reseller.suspended',
    ]

    const { data, isLoading } = useQuery({
        queryKey: ['reseller-webhooks'],
        queryFn: () => resellerAPI.listWebhooks(),
    })
    const webhooks = data?.data?.data ?? []

    const createMut = useMutation({
        mutationFn: () => resellerAPI.createWebhook(url, secret, events),
        onSuccess: () => {
            setUrl('')
            setSecret('')
            setEvents(['order.created', 'order.status_changed'])
            qc.invalidateQueries({ queryKey: ['reseller-webhooks'] })
            success('Webhook created')
        },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Failed to create webhook'),
    })

    const deleteMut = useMutation({
        mutationFn: (id: string) => resellerAPI.deleteWebhook(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['reseller-webhooks'] })
            success('Webhook deleted')
        },
        onError: () => toastError('Failed to delete webhook'),
    })

    const handleDelete = async (id: string, webhookUrl: string) => {
        const ok = await confirm({
            title: 'Delete Webhook',
            message: `Delete webhook endpoint "${webhookUrl}"?`,
            confirmLabel: 'Delete',
            variant: 'danger',
        })
        if (ok) deleteMut.mutate(id)
    }

    const toggleEvent = (evt: string) => {
        setEvents(prev => prev.includes(evt) ? prev.filter(e => e !== evt) : [...prev, evt])
    }

    return (
        <AppLayout breadcrumb={[
            { label: 'Reseller', href: '/reseller' },
            { label: 'Webhooks' },
        ]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Webhooks</h1>
                    <p className="page-subtitle">Receive real-time event notifications via HTTP POST</p>
                </div>
            </div>

            {/* Create webhook */}
            <div className="card">
                <div className="card-header"><Plus size={17} /> Add Webhook Endpoint</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem', marginBottom: '.75rem' }}>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Endpoint URL</label>
                        <input
                            className="input"
                            type="url"
                            placeholder="https://your-app.com/webhook"
                            value={url}
                            onChange={e => setUrl(e.target.value)}
                        />
                    </div>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Signing Secret (HMAC-SHA256)</label>
                        <input
                            className="input"
                            type="password"
                            placeholder="Optional secret for request signing"
                            value={secret}
                            onChange={e => setSecret(e.target.value)}
                        />
                    </div>
                </div>
                <div className="form-group" style={{ marginBottom: '1rem' }}>
                    <label>Events to subscribe</label>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '.4rem', marginTop: '.5rem' }}>
                        {AVAILABLE_EVENTS.map(evt => (
                            <button
                                key={evt}
                                onClick={() => toggleEvent(evt)}
                                style={{
                                    padding: '.25rem .65rem',
                                    borderRadius: 20,
                                    fontSize: '.78rem',
                                    fontWeight: 500,
                                    border: '1px solid',
                                    cursor: 'pointer',
                                    transition: 'all .15s',
                                    background: events.includes(evt) ? 'rgba(230,168,23,.15)' : 'transparent',
                                    color: events.includes(evt) ? 'var(--dc-gold)' : 'var(--text-muted)',
                                    borderColor: events.includes(evt) ? 'rgba(230,168,23,.5)' : 'var(--border-light)',
                                }}
                            >
                                {evt}
                            </button>
                        ))}
                    </div>
                </div>
                <button
                    className="btn-primary"
                    onClick={() => createMut.mutate()}
                    disabled={!url || events.length === 0 || createMut.isPending}
                >
                    <Plus size={15} />{createMut.isPending ? 'Creating...' : 'Create Webhook'}
                </button>
            </div>

            {/* Webhooks list */}
            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
                    <Webhook size={17} /> Endpoints ({webhooks.length})
                </div>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : webhooks.length === 0 ? (
                    <div className="empty-state">
                        <Webhook size={40} opacity={0.3} />
                        <p>No webhooks configured</p>
                        <p style={{ fontSize: '.8rem' }}>Add an endpoint above to start receiving events</p>
                    </div>
                ) : (
                    <div className="table-wrapper">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>Endpoint URL</th>
                                    <th>Status</th>
                                    <th>Created</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {webhooks.map((wh: any) => (
                                    <tr key={wh.id}>
                                        <td>
                                            <span style={{ display: 'flex', alignItems: 'center', gap: '.4rem', fontSize: '.875rem' }}>
                                                <Link size={13} color="var(--text-muted)" />
                                                <span style={{ fontFamily: 'monospace', color: 'var(--text-heading)' }}>{wh.url}</span>
                                            </span>
                                        </td>
                                        <td>
                                            <span className={`badge badge-${wh.is_active ? 'success' : 'error'}`}>
                                                {wh.is_active ? 'Active' : 'Inactive'}
                                            </span>
                                        </td>
                                        <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                            {new Date(wh.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                                        </td>
                                        <td>
                                            <button
                                                className="action-btn red"
                                                onClick={() => handleDelete(wh.id, wh.url)}
                                                disabled={deleteMut.isPending}
                                                title="Delete webhook"
                                            >
                                                <Trash2 size={13} /> Delete
                                            </button>
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
