'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import AuditLog from '@/components/ui/AuditLog'
import { Users, Pencil, Trash2, ShieldCheck, X } from 'lucide-react'

const ROLES = ['user', 'reseller', 'admin']
const STATUSES = ['active', 'suspended', 'banned']

const ROLE_COLOR: Record<string, string> = {
    super_admin: 'primary', admin: 'info', reseller: 'warning', user: 'secondary',
}
const STATUS_COLOR: Record<string, string> = {
    active: 'success', suspended: 'warning', banned: 'error',
}

type User = {
    id: string; email: string; full_name: string; phone: string
    role: string; status: string; created_at: string; last_login_at?: string
    totp_enabled: boolean
}

// ─── Edit Modal ───────────────────────────────────────────────────────────────
function EditModal({ user, onClose }: { user: User; onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [form, setForm] = useState({ full_name: user.full_name, phone: user.phone })
    const [role, setRole] = useState(user.role)
    const [status, setStatus] = useState(user.status)
    const [activeTab, setActiveTab] = useState<'info' | 'activity'>('info')

    const profileMut = useMutation({
        mutationFn: () => adminAPI.updateUserProfile(user.id, form),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); success('Profile updated') },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed'),
    })
    const roleMut = useMutation({
        mutationFn: () => adminAPI.updateUserRole(user.id, role),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); success('Role updated') },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed'),
    })
    const statusMut = useMutation({
        mutationFn: () => adminAPI.updateUserStatus(user.id, status),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); success('Status updated') },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed'),
    })

    const anyPending = profileMut.isPending || roleMut.isPending || statusMut.isPending

    const handleSaveAll = async () => {
        const tasks: Promise<any>[] = []
        if (form.full_name !== user.full_name || form.phone !== user.phone)
            tasks.push(adminAPI.updateUserProfile(user.id, form))
        if (role !== user.role)
            tasks.push(adminAPI.updateUserRole(user.id, role))
        if (status !== user.status)
            tasks.push(adminAPI.updateUserStatus(user.id, status))
        if (!tasks.length) { onClose(); return }
        try {
            await Promise.all(tasks)
            qc.invalidateQueries({ queryKey: ['admin-users'] })
            success('User updated')
            onClose()
        } catch (e: any) { toastError(e?.response?.data?.error?.message ?? 'Some updates failed') }
    }

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 480, width: '95vw' }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}>
                        <Pencil size={16} /> Edit User
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>

                {/* Tabs */}
                <div style={{ display: 'flex', borderBottom: '1px solid var(--border-light)', padding: '0 1.25rem' }}>
                    {(['info', 'activity'] as const).map(tab => (
                        <button key={tab} onClick={() => setActiveTab(tab)} style={{
                            padding: '.6rem 1rem', fontWeight: 600, fontSize: '.82rem',
                            background: 'none', border: 'none', cursor: 'pointer',
                            borderBottom: activeTab === tab ? '2px solid var(--primary)' : '2px solid transparent',
                            color: activeTab === tab ? 'var(--primary)' : 'var(--text-muted)',
                            textTransform: 'capitalize',
                        }}>{tab}</button>
                    ))}
                </div>
                <div className="modal-body">
                    {activeTab === 'info' && (
                        <>
                            {/* User info header */}
                            <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem', marginBottom: '1rem', padding: '.75rem', background: 'var(--bg)', borderRadius: 8 }}>
                                <div style={{ width: 40, height: 40, borderRadius: '50%', background: 'var(--primary-light)', color: 'var(--primary)', display: 'grid', placeItems: 'center', fontWeight: 700, fontSize: '.9rem', flexShrink: 0 }}>
                                    {user.email?.[0]?.toUpperCase() ?? '?'}
                                </div>
                                <div>
                                    <div style={{ fontWeight: 600 }}>{user.full_name || user.email}</div>
                                    <div style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>{user.email}</div>
                                </div>
                            </div>

                            <div style={{ display: 'flex', flexDirection: 'column', gap: '.75rem' }}>
                                <div className="form-group" style={{ marginBottom: 0 }}>
                                    <label>Full Name</label>
                                    <input className="input" value={form.full_name}
                                        onChange={e => setForm(f => ({ ...f, full_name: e.target.value }))} />
                                </div>
                                <div className="form-group" style={{ marginBottom: 0 }}>
                                    <label>Phone</label>
                                    <input className="input" value={form.phone}
                                        onChange={e => setForm(f => ({ ...f, phone: e.target.value }))} placeholder="+84..." />
                                </div>
                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem' }}>
                                    <div className="form-group" style={{ marginBottom: 0 }}>
                                        <label>Role</label>
                                        <select className="input" value={role} onChange={e => setRole(e.target.value)}>
                                            {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
                                        </select>
                                    </div>
                                    <div className="form-group" style={{ marginBottom: 0 }}>
                                        <label>Status</label>
                                        <select className="input" value={status} onChange={e => setStatus(e.target.value)}>
                                            {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
                                        </select>
                                    </div>
                                </div>
                            </div>

                            <button className="btn-primary" style={{ width: '100%', marginTop: '1rem' }}
                                onClick={handleSaveAll} disabled={anyPending}>
                                💾&nbsp;{anyPending ? 'Saving...' : 'Save Changes'}
                            </button>
                        </>
                    )}

                    {activeTab === 'activity' && (
                        <AuditLog userId={user.id} pageSize={8} title="User Activity" />
                    )}
                </div>
            </div>
        </div>
    )
}

// ─── Page ─────────────────────────────────────────────────────────────────────
export default function UsersPage() {
    const [page, setPage] = useState(1)
    const [editUser, setEditUser] = useState<User | null>(null)
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const { data, isLoading } = useQuery({
        queryKey: ['admin-users', page],
        queryFn: () => adminAPI.listUsers(page),
    })
    const users: User[] = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const deleteMut = useMutation({
        mutationFn: (id: string) => adminAPI.deleteUser(id),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); success('User deleted') },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Delete failed'),
    })

    const handleDelete = async (u: User) => {
        const ok = await confirm({
            title: 'Delete User',
            message: `Permanently delete "${u.full_name || u.email}"? This action cannot be undone.`,
            confirmLabel: 'Delete',
            variant: 'danger',
        })
        if (ok) deleteMut.mutate(u.id)
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Admin', href: '/admin' }, { label: 'Users' }]}>
            {confirmDialog}
            {editUser && <EditModal user={editUser} onClose={() => setEditUser(null)} />}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Users</h1>
                    <p className="page-subtitle">{meta.total ?? 0} total accounts</p>
                </div>
            </div>

            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.1rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0, display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 600 }}>
                    <Users size={17} /> All Users
                </div>

                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : users.length === 0 ? (
                    <div className="empty-state"><Users size={40} opacity={0.3} /><p>No users found</p></div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>User</th>
                                        <th>Role</th>
                                        <th>Status</th>
                                        <th>2FA</th>
                                        <th>Last Login</th>
                                        <th>Joined</th>
                                        <th>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {users.map((u) => {
                                        const initials = u.full_name
                                            ? u.full_name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
                                            : (u.email?.[0] ?? '?').toUpperCase()
                                        return (
                                            <tr key={u.id}>
                                                <td>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                                                        <div style={{ width: 34, height: 34, borderRadius: '50%', background: 'var(--primary-light)', color: 'var(--primary)', display: 'grid', placeItems: 'center', fontWeight: 700, fontSize: '.78rem', flexShrink: 0 }}>
                                                            {initials}
                                                        </div>
                                                        <div>
                                                            <p style={{ fontWeight: 600, color: 'var(--text-heading)', fontSize: '.875rem' }}>
                                                                {u.full_name || '—'}
                                                            </p>
                                                            <p style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>{u.email}</p>
                                                        </div>
                                                    </div>
                                                </td>
                                                <td>
                                                    <span className={`badge badge-${ROLE_COLOR[u.role] ?? 'secondary'}`}>
                                                        {u.role}
                                                    </span>
                                                </td>
                                                <td>
                                                    <span className={`badge badge-${STATUS_COLOR[u.status] ?? 'secondary'}`}>
                                                        {u.status}
                                                    </span>
                                                </td>
                                                <td>
                                                    <span className={`badge badge-${u.totp_enabled ? 'success' : 'secondary'}`}>
                                                        {u.totp_enabled ? '2FA On' : '2FA Off'}
                                                    </span>
                                                </td>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                    {u.last_login_at
                                                        ? new Date(u.last_login_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
                                                        : '—'}
                                                </td>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                    {new Date(u.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                                                </td>
                                                <td>
                                                    <div style={{ display: 'flex', gap: '.35rem' }}>
                                                        <button className="action-btn" onClick={() => setEditUser(u)} title="Edit">
                                                            <Pencil size={13} /> Edit
                                                        </button>
                                                        {u.totp_enabled && (
                                                            <button className="action-btn" style={{ color: 'var(--warning)' }}
                                                                title="Disable 2FA"
                                                                onClick={async () => {
                                                                    try {
                                                                        await adminAPI.adminDisable2FA(u.id)
                                                                        qc.invalidateQueries({ queryKey: ['admin-users'] })
                                                                        success('2FA disabled')
                                                                    } catch { toastError('Failed') }
                                                                }}>
                                                                <ShieldCheck size={13} />
                                                            </button>
                                                        )}
                                                        <button className="action-btn red" onClick={() => handleDelete(u)}
                                                            disabled={deleteMut.isPending} title="Delete">
                                                            <Trash2 size={13} />
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        )
                                    })}
                                </tbody>
                            </table>
                        </div>
                        <Pagination
                            page={page}
                            totalPages={meta.pages ?? 1}
                            total={meta.total ?? 0}
                            limit={15}
                            onPageChange={setPage}
                        />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
