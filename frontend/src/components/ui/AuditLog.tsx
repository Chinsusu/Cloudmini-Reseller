'use client'
import { useQuery } from '@tanstack/react-query'
import { Shield, LogIn, UserCheck, KeyRound, ShieldOff, ShieldCheck, User, RefreshCw } from 'lucide-react'
import api from '@/lib/api'
import { Pagination } from '@/components/ui/Pagination'
import { useState } from 'react'

export interface LogEntry {
    id: string
    service_name: string
    user_id?: string
    actor_type: string
    action: string
    level: string
    resource_type?: string
    resource_id?: string
    message: string
    ip_address?: string
    created_at: string
}

interface AuditLogProps {
    userId?: string       // filter by user (admin passes a target user's ID)
    action?: string       // filter by event type
    pageSize?: number
    title?: string
}

const ACTION_ICON: Record<string, React.ReactNode> = {
    'user.registered': <User size={14} />,
    'user.login': <LogIn size={14} />,
    'user.verified': <UserCheck size={14} />,
    'user.password_changed': <KeyRound size={14} />,
    'user.suspended': <ShieldOff size={14} />,
    'user.2fa_enabled': <ShieldCheck size={14} />,
    'user.2fa_disabled': <ShieldOff size={14} />,
    'user.2fa_admin_disabled': <ShieldOff size={14} />,
    'user.admin_updated': <RefreshCw size={14} />,
}

const LEVEL_COLOR: Record<string, string> = {
    INFO: 'var(--success)',
    WARN: 'var(--warning)',
    ERROR: 'var(--error)',
}

function formatDate(iso: string) {
    const d = new Date(iso)
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
        + ' ' + d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
}

export default function AuditLog({ userId, action, pageSize = 10, title = 'Activity Log' }: AuditLogProps) {
    const [page, setPage] = useState(1)

    const params = new URLSearchParams({ page: String(page), limit: String(pageSize) })
    if (userId) params.set('user_id', userId)
    if (action) params.set('action', action)

    const { data, isLoading, isError } = useQuery({
        queryKey: ['audit-logs', userId, action, page],
        queryFn: () => api.get(`/v1/logs?${params}`).then(r => r.data),
    })

    const entries: LogEntry[] = data?.data ?? []
    const meta = data?.meta ?? {}

    return (
        <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '.5rem', marginBottom: '1rem' }}>
                <Shield size={16} style={{ color: 'var(--primary)' }} />
                <span style={{ fontWeight: 600, fontSize: '.9rem' }}>{title}</span>
            </div>

            {isLoading && (
                <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>
                    Loading activity…
                </div>
            )}

            {isError && (
                <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--error)', fontSize: '.85rem' }}>
                    Failed to load activity log.
                </div>
            )}

            {!isLoading && !isError && entries.length === 0 && (
                <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>
                    No activity recorded yet.
                </div>
            )}

            {!isLoading && entries.length > 0 && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
                    {entries.map((e, idx) => (
                        <div key={e.id ?? idx} style={{
                            display: 'flex', alignItems: 'flex-start', gap: '.75rem',
                            padding: '.65rem 0',
                            borderBottom: idx < entries.length - 1 ? '1px solid var(--border-light)' : 'none',
                        }}>
                            {/* Icon */}
                            <div style={{
                                width: 30, height: 30, borderRadius: '50%', flexShrink: 0,
                                background: 'var(--bg)', border: '1px solid var(--border-light)',
                                display: 'grid', placeItems: 'center',
                                color: LEVEL_COLOR[e.level] ?? 'var(--text-muted)',
                            }}>
                                {ACTION_ICON[e.action] ?? <Shield size={14} />}
                            </div>

                            {/* Content */}
                            <div style={{ flex: 1, minWidth: 0 }}>
                                <div style={{ fontSize: '.85rem', fontWeight: 500 }}>{e.message}</div>
                                <div style={{ display: 'flex', gap: '.5rem', marginTop: 2, flexWrap: 'wrap' }}>
                                    <span style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>
                                        {formatDate(e.created_at)}
                                    </span>
                                    {e.ip_address && (
                                        <span style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>
                                            · {e.ip_address}
                                        </span>
                                    )}
                                    <span style={{
                                        fontSize: '.7rem', fontWeight: 600,
                                        padding: '1px 6px', borderRadius: 4,
                                        background: e.actor_type === 'admin' ? 'rgba(239,68,68,.1)' : 'rgba(var(--primary-rgb),.08)',
                                        color: e.actor_type === 'admin' ? 'var(--error)' : 'var(--primary)',
                                    }}>
                                        {e.actor_type === 'admin' ? 'by admin' : e.actor_type}
                                    </span>
                                </div>
                            </div>

                            {/* Level badge */}
                            <span style={{
                                fontSize: '.7rem', fontWeight: 700, flexShrink: 0, marginTop: 4,
                                padding: '2px 7px', borderRadius: 4,
                                background: e.level === 'INFO'
                                    ? 'rgba(34,197,94,.1)' : e.level === 'WARN'
                                        ? 'rgba(251,191,36,.1)' : 'rgba(239,68,68,.1)',
                                color: LEVEL_COLOR[e.level] ?? 'var(--text-muted)',
                            }}>
                                {e.level}
                            </span>
                        </div>
                    ))}
                </div>
            )}

            {meta.pages > 1 && (
                <div style={{ marginTop: '1rem' }}>
                    <Pagination
                        page={page}
                        totalPages={meta.pages ?? 1}
                        total={meta.total ?? 0}
                        limit={pageSize}
                        onPageChange={setPage}
                    />
                </div>
            )}
        </div>
    )
}
