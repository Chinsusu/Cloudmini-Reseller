'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { Users, ShieldCheck, Activity, AlertCircle, CheckCircle2, XCircle } from 'lucide-react'

export default function AdminPage() {
    const [page, setPage] = useState(1)
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const { data: resellers, isLoading } = useQuery({
        queryKey: ['admin-resellers', page],
        queryFn: () => adminAPI.listResellers(),
    })
    const { data: users } = useQuery({
        queryKey: ['admin-users-count'],
        queryFn: () => adminAPI.listUsers(1),
    })

    const items = resellers?.data?.data ?? []
    const meta = resellers?.data?.meta ?? {}
    const totalUsers = users?.data?.meta?.total ?? 0

    const approveMut = useMutation({
        mutationFn: (id: string) => adminAPI.approveReseller(id),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-resellers'] }); success('Reseller approved') },
        onError: () => toastError('Failed to approve reseller'),
    })
    const suspendMut = useMutation({
        mutationFn: (id: string) => adminAPI.suspendReseller(id, 'Suspended by admin'),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-resellers'] }); success('Reseller suspended') },
        onError: () => toastError('Failed to suspend reseller'),
    })

    const handleApprove = async (id: string, company: string) => {
        const ok = await confirm({ title: 'Approve Reseller', message: `Approve "${company}"?`, confirmLabel: 'Approve', variant: 'primary' })
        if (ok) approveMut.mutate(id)
    }
    const handleSuspend = async (id: string, company: string) => {
        const ok = await confirm({ title: 'Suspend Reseller', message: `Suspend "${company}"? They will lose access.`, confirmLabel: 'Suspend', variant: 'danger' })
        if (ok) suspendMut.mutate(id)
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Admin' }, { label: 'Overview' }]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Admin Dashboard</h1>
                    <p className="page-subtitle">Platform overview and management</p>
                </div>
            </div>

            {/* Stats */}
            <div className="stats-grid">
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#7367F0,#9e95f5)' }}><Users size={22} /></div>
                    <div>
                        <p className="stat-label">Total Users</p>
                        <p className="stat-value">{totalUsers}</p>
                        <p className="stat-sub">All registered accounts</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#28C76F,#48DA89)' }}><ShieldCheck size={22} /></div>
                    <div>
                        <p className="stat-label">Resellers</p>
                        <p className="stat-value">{meta.total ?? 0}</p>
                        <p className="stat-sub">Registered resellers</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#00CFE8,#1EDEC5)' }}><Activity size={22} /></div>
                    <div>
                        <p className="stat-label">Platform Status</p>
                        <p className="stat-value" style={{ fontSize: '1.2rem', color: 'var(--success)' }}>Operational</p>
                        <p className="stat-sub">All services running</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#FF9F43,#FFB976)' }}><AlertCircle size={22} /></div>
                    <div>
                        <p className="stat-label">Pending Approval</p>
                        <p className="stat-value">{items.filter((r: any) => r.status === 'pending').length}</p>
                        <p className="stat-sub">Awaiting review</p>
                    </div>
                </div>
            </div>

            {/* Resellers Table */}
            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
                    <ShieldCheck size={17} /> Resellers
                </div>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : items.length === 0 ? (
                    <div className="empty-state"><ShieldCheck size={40} opacity={0.3} /><p>No resellers yet</p></div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Company</th>
                                        <th>Email</th>
                                        <th>Status</th>
                                        <th>Commission</th>
                                        <th>Created</th>
                                        <th>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {items.map((r: any) => (
                                        <tr key={r.id}>
                                            <td>
                                                <p style={{ fontWeight: 600, color: 'var(--text-heading)' }}>{r.company_name}</p>
                                                <p style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>{r.website || '—'}</p>
                                            </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.875rem' }}>{r.email || '—'}</td>
                                            <td><span className={`badge badge-${r.status}`}>{r.status}</span></td>
                                            <td style={{ fontSize: '.875rem' }}>{r.commission_pct ?? 0}%</td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                {new Date(r.created_at).toLocaleDateString()}
                                            </td>
                                            <td>
                                                <div className="action-group">
                                                    {r.status === 'pending' && (
                                                        <button className="action-btn green" onClick={() => handleApprove(r.id, r.company_name)} title="Approve">
                                                            <CheckCircle2 size={14} />
                                                        </button>
                                                    )}
                                                    {r.status !== 'suspended' && (
                                                        <button className="action-btn red" onClick={() => handleSuspend(r.id, r.company_name)} title="Suspend">
                                                            <XCircle size={14} />
                                                        </button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                        <Pagination
                            page={page}
                            totalPages={meta.pages ?? 1}
                            total={meta.total ?? 0}
                            limit={10}
                            onPageChange={setPage}
                        />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
