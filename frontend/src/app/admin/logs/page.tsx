'use client'
import { useState } from 'react'
import { AppLayout } from '@/components/layout/AppLayout'
import AuditLog from '@/components/ui/AuditLog'

const ACTIONS = [
    { value: '', label: 'All Events' },
    { value: 'user.registered', label: 'Registered' },
    { value: 'user.login', label: 'Login' },
    { value: 'user.password_changed', label: 'Password Changed' },
    { value: 'user.suspended', label: 'Suspended' },
    { value: 'user.2fa_enabled', label: '2FA Enabled' },
    { value: 'user.2fa_disabled', label: '2FA Disabled' },
    { value: 'user.2fa_admin_disabled', label: '2FA Force-Disabled' },
    { value: 'user.admin_updated', label: 'Admin Updated' },
]

export default function AdminLogsPage() {
    const [filterAction, setFilterAction] = useState('')
    const [filterUser, setFilterUser] = useState('')
    const [appliedUser, setAppliedUser] = useState('')

    return (
        <AppLayout>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.5rem', flexWrap: 'wrap', gap: '1rem' }}>
                <div>
                    <h1 className="page-title" style={{ margin: 0 }}>Audit Logs</h1>
                    <p style={{ margin: 0, fontSize: '.85rem', color: 'var(--text-muted)' }}>System-wide activity trail</p>
                </div>

                {/* Filters */}
                <div style={{ display: 'flex', gap: '.75rem', flexWrap: 'wrap' }}>
                    <select
                        className="input"
                        style={{ width: 'auto', minWidth: 160 }}
                        value={filterAction}
                        onChange={e => setFilterAction(e.target.value)}
                    >
                        {ACTIONS.map(a => (
                            <option key={a.value} value={a.value}>{a.label}</option>
                        ))}
                    </select>

                    <div style={{ display: 'flex', gap: '.5rem' }}>
                        <input
                            className="input"
                            style={{ width: 220 }}
                            placeholder="Filter by User ID…"
                            value={filterUser}
                            onChange={e => setFilterUser(e.target.value)}
                            onKeyDown={e => e.key === 'Enter' && setAppliedUser(filterUser)}
                        />
                        <button className="btn btn-secondary" onClick={() => setAppliedUser(filterUser)}>
                            Filter
                        </button>
                        {appliedUser && (
                            <button className="btn btn-secondary" onClick={() => { setFilterUser(''); setAppliedUser('') }}>
                                Clear
                            </button>
                        )}
                    </div>
                </div>
            </div>

            <div className="card" style={{ padding: '1.25rem' }}>
                <AuditLog
                    userId={appliedUser || undefined}
                    action={filterAction || undefined}
                    pageSize={20}
                    title=""
                />
            </div>
        </AppLayout>
    )
}
