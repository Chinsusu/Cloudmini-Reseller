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
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--dc-gold)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(230,168,23,.15)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Users size={18} color="var(--dc-gold)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Total Users</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>{totalUsers}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>All registered accounts</p>
                        </div>
                    </div>
                </div>
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--success)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(40,199,111,.12)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <ShieldCheck size={18} color="var(--success)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Resellers</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>{meta.total ?? 0}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Registered resellers</p>
                        </div>
                    </div>
                </div>
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--info)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(0,207,232,.1)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Activity size={18} color="var(--info)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Platform Status</p>
                            <p style={{ fontSize: '1.2rem', fontWeight: 700, color: 'var(--success)', lineHeight: 1.2 }}>Operational</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>All services running</p>
                        </div>
                    </div>
                </div>
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--warning)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(255,159,67,.12)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <AlertCircle size={18} color="var(--warning)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Pending Approval</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--warning)', lineHeight: 1.2 }}>{items.filter((r: any) => r.status === 'pending').length}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Awaiting review</p>
                        </div>
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
