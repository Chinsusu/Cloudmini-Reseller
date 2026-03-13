'use client'
import { useState } from 'react'
import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import AuditLog from '@/components/ui/AuditLog'
import { formatVND } from '@/lib/format'
import { Users, Pencil, Trash2, ShieldCheck, X, Save, Globe, Server, Wallet } from 'lucide-react'

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

// ─── Top-Up Modal ─────────────────────────────────────────────────────────────
function TopUpModal({ user, onClose }: { user: User; onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [amount, setAmount] = useState('')
    const [desc, setDesc] = useState('')
    const [loading, setLoading] = useState(false)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        const num = parseInt(amount.replace(/[^0-9]/g, ''), 10)
        if (!num || num <= 0) { toastError('Nhập số tiền hợp lệ'); return }
        setLoading(true)
        try {
            await adminAPI.adminAdjustBalance(user.id, num, desc || 'Admin manual top-up')
            qc.invalidateQueries({ queryKey: ['admin-user-wallet', user.id], refetchType: 'all' })
            success(`Đã nạp ${num.toLocaleString('vi-VN')}₫ cho ${user.full_name || user.email}`)
            onClose()
        } catch (e: any) {
            toastError(e?.response?.data?.error?.message ?? 'Nạp tiền thất bại')
        } finally { setLoading(false) }
    }

    const formatted = amount ? parseInt(amount, 10).toLocaleString('vi-VN') : ''

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 420, width: '95vw' }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}>
                        <Wallet size={16} color="var(--dc-gold)" /> Nạp tiền cho user
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body">
                    {/* User info */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem', marginBottom: '1.25rem', padding: '.75rem', background: 'var(--bg)', borderRadius: 8, border: '1px solid var(--border)' }}>
                        <div style={{ width: 38, height: 38, borderRadius: '50%', background: 'rgba(230,168,23,.15)', border: '1.5px solid rgba(230,168,23,.3)', display: 'grid', placeItems: 'center', color: 'var(--dc-gold)', fontWeight: 700, fontSize: '.85rem', flexShrink: 0 }}>
                            {(user.full_name?.[0] ?? user.email?.[0] ?? '?').toUpperCase()}
                        </div>
                        <div>
                            <p style={{ fontWeight: 600, fontSize: '.875rem' }}>{user.full_name || user.email}</p>
                            <p style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>{user.email}</p>
                        </div>
                    </div>

                    {/* Notice */}
                    <div style={{ padding: '.65rem .875rem', background: 'rgba(230,168,23,.08)', border: '1px solid rgba(230,168,23,.25)', borderRadius: 7, marginBottom: '1rem', fontSize: '.78rem', color: 'var(--text-muted)', lineHeight: 1.5 }}>
                        <span style={{ color: 'var(--dc-gold)', fontWeight: 600 }}>Lưu ý:</span> Khoản nạp này được ghi nhận là <strong>admin adjustment</strong> — <em>không</em> tính vào doanh thu cty.
                    </div>

                    <form onSubmit={handleSubmit}>
                        <div className="form-group">
                            <label>Số tiền (₫)</label>
                            <input
                                className="input"
                                placeholder="Ví dụ: 500000"
                                value={amount}
                                onChange={e => setAmount(e.target.value.replace(/[^0-9]/g, ''))}
                                autoFocus
                                inputMode="numeric"
                            />
                            {formatted && (
                                <p style={{ fontSize: '.82rem', color: 'var(--dc-gold)', marginTop: '.4rem', fontWeight: 600 }}>
                                    = {formatted}₫
                                </p>
                            )}
                        </div>
                        <div className="form-group">
                            <label>Ghi chú <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}>(tùy chọn)</span></label>
                            <input className="input" placeholder="Lý do nạp tiền..." value={desc} onChange={e => setDesc(e.target.value)} />
                        </div>
                        <button type="submit" className="btn-primary" style={{ width: '100%', marginTop: '.25rem' }} disabled={!amount || loading}>
                            <Wallet size={14} />{loading ? 'Đang nạp...' : 'Xác nhận nạp'}
                        </button>
                    </form>
                </div>
            </div>
        </div>
    )
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
                            borderBottom: activeTab === tab ? '2px solid var(--dc-gold)' : '2px solid transparent',
                            color: activeTab === tab ? 'var(--dc-gold)' : 'var(--text-muted)',
                            textTransform: 'capitalize',
                        }}>{tab}</button>
                    ))}
                </div>
                <div className="modal-body">
                    {activeTab === 'info' && (
                        <>
                            {/* User info header */}
                            <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem', marginBottom: '1rem', padding: '.75rem', background: 'var(--bg)', borderRadius: 8, border: '1px solid var(--border)' }}>
                                <div style={{
                                    width: 40, height: 40, borderRadius: '50%',
                                    background: 'linear-gradient(135deg, rgba(230,168,23,.25), rgba(230,168,23,.1))',
                                    border: '1.5px solid rgba(230,168,23,.35)',
                                    color: 'var(--dc-gold)', display: 'grid', placeItems: 'center', fontWeight: 700, fontSize: '.9rem', flexShrink: 0,
                                }}>
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
                                <Save size={14} />{anyPending ? 'Saving...' : 'Save Changes'}
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

// ─── User Row with balance/service data ──────────────────────────────────────
function UserRow({ u, onEdit, onDelete, onDisable2FA, onTopUp }: {
    u: User
    onEdit: (u: User) => void
    onDelete: (u: User) => void
    onDisable2FA: (id: string) => void
    onTopUp: (u: User) => void
}) {

    const walletQ = useQuery({
        queryKey: ['admin-user-wallet', u.id],
        queryFn: () => adminAPI.getUserWallet(u.id),
        staleTime: 0,
        retry: false,
    })
    const proxyQ = useQuery({
        queryKey: ['admin-user-proxy', u.id],
        queryFn: () => adminAPI.getUserProxyOrders(u.id),
        staleTime: 60_000,
        retry: false,
    })
    const vpsQ = useQuery({
        queryKey: ['admin-user-vps', u.id],
        queryFn: () => adminAPI.getUserVPSInstances(u.id),
        staleTime: 60_000,
        retry: false,
    })

    const balance = walletQ.data?.data?.data?.balance ?? walletQ.data?.data?.balance
    const proxyTotal = proxyQ.data?.data?.meta?.total ?? 0
    const vpsTotal = vpsQ.data?.data?.meta?.total ?? 0
    const totalServices = proxyTotal + vpsTotal

    const initials = u.full_name
        ? u.full_name.split(' ').map((n: string) => n[0]).join('').toUpperCase().slice(0, 2)
        : (u.email?.[0] ?? '?').toUpperCase()

    return (
        <tr>
            <td>
                <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                    <div style={{
                        width: 34, height: 34, borderRadius: '50%',
                        background: 'linear-gradient(135deg, rgba(230,168,23,.2), rgba(230,168,23,.08))',
                        border: '1.5px solid rgba(230,168,23,.3)',
                        color: 'var(--dc-gold)', display: 'grid', placeItems: 'center', fontWeight: 700, fontSize: '.78rem', flexShrink: 0,
                    }}>
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
            {/* Balance */}
            <td>
                {walletQ.isLoading ? (
                    <span style={{ color: 'var(--text-muted)', fontSize: '.78rem' }}>…</span>
                ) : balance != null ? (
                    <span style={{ fontWeight: 700, color: 'var(--text-heading)', fontSize: '.875rem' }}>
                        {formatVND(balance)}
                    </span>
                ) : (
                    <span style={{ color: 'var(--text-muted)', fontSize: '.78rem' }}>—</span>
                )}
            </td>
            {/* Total Services */}
            <td>
                {(proxyQ.isLoading || vpsQ.isLoading) ? (
                    <span style={{ color: 'var(--text-muted)', fontSize: '.78rem' }}>…</span>
                ) : (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.5rem', flexWrap: 'wrap' }}>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.25rem', fontSize: '.78rem', color: proxyTotal > 0 ? 'var(--info)' : 'var(--text-muted)' }}>
                            <Globe size={11} /> {proxyTotal}
                        </span>
                        <span style={{ color: 'var(--border)', fontSize: '.7rem' }}>·</span>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.25rem', fontSize: '.78rem', color: vpsTotal > 0 ? 'var(--success)' : 'var(--text-muted)' }}>
                            <Server size={11} /> {vpsTotal}
                        </span>
                        {totalServices > 0 && (
                            <span className="badge badge-secondary" style={{ marginLeft: '.1rem' }}>
                                {totalServices}
                            </span>
                        )}
                    </div>
                )}
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
                    <button className="action-btn" onClick={() => onEdit(u)} title="Edit">
                        <Pencil size={13} /> Edit
                    </button>
                    <button
                        className="action-btn"
                        style={{ color: 'var(--dc-gold)', borderColor: 'rgba(230,168,23,.3)' }}
                        title="Nạp tiền"
                        onClick={() => onTopUp(u)}
                    >
                        <Wallet size={13} />
                    </button>
                    {u.totp_enabled && (
                        <button className="action-btn" style={{ color: 'var(--warning)' }}
                            title="Disable 2FA"
                            onClick={() => onDisable2FA(u.id)}>
                            <ShieldCheck size={13} />
                        </button>
                    )}
                    <button className="action-btn red" onClick={() => onDelete(u)} title="Delete">
                        <Trash2 size={13} />
                    </button>
                </div>
            </td>
        </tr>
    )
}

// ─── Page ─────────────────────────────────────────────────────────────────────
export default function UsersPage() {
    const [page, setPage] = useState(1)
    const [editUser, setEditUser] = useState<User | null>(null)
    const [topUpUser, setTopUpUser] = useState<User | null>(null)
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

    const handleDisable2FA = async (id: string) => {
        try {
            await adminAPI.adminDisable2FA(id)
            qc.invalidateQueries({ queryKey: ['admin-users'] })
            success('2FA disabled')
        } catch { toastError('Failed') }
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Admin', href: '/admin' }, { label: 'Users' }]}>
            {confirmDialog}
            {editUser && <EditModal user={editUser} onClose={() => setEditUser(null)} />}
            {topUpUser && <TopUpModal user={topUpUser} onClose={() => setTopUpUser(null)} />}

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
                                        <th>Balance</th>
                                        <th>Services <span style={{ fontSize: '.7rem', fontWeight: 400, color: 'var(--text-muted)' }}>(proxy · vps)</span></th>
                                        <th>Last Login</th>
                                        <th>Joined</th>
                                        <th>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {users.map((u) => (
                                        <UserRow
                                            key={u.id}
                                            u={u}
                                            onEdit={setEditUser}
                                            onDelete={handleDelete}
                                            onDisable2FA={handleDisable2FA}
                                            onTopUp={setTopUpUser}
                                        />
                                    ))}
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
