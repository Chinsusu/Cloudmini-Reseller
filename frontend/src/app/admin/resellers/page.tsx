'use client'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { CheckCircle2, XCircle, Users } from 'lucide-react'
import { useState } from 'react'

const STATUS_COLOR: Record<string, string> = {
    approved: 'success', pending: 'warning', suspended: 'error',
}

export default function AdminResellersPage() {
    const qc = useQueryClient()
    const [page, setPage] = useState(1)
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const { data, isLoading } = useQuery({
        queryKey: ['admin-resellers', page],
        queryFn: () => adminAPI.listResellers(),
    })
    const resellers = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const approve = useMutation({
        mutationFn: (id: string) => adminAPI.approveReseller(id),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-resellers'] }); success('Reseller approved') },
        onError: () => toastError('Failed to approve reseller'),
    })
    const suspend = useMutation({
        mutationFn: ({ id, reason }: { id: string; reason: string }) => adminAPI.suspendReseller(id, reason),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-resellers'] }); success('Reseller suspended') },
        onError: () => toastError('Failed to suspend reseller'),
    })

    const handleApprove = async (id: string, company: string) => {
        const ok = await confirm({ title: 'Approve Reseller', message: `Approve "${company}"?`, confirmLabel: 'Approve', variant: 'primary' })
        if (ok) approve.mutate(id)
    }
    const handleSuspend = async (id: string, company: string) => {
        const ok = await confirm({ title: 'Suspend Reseller', message: `Suspend "${company}"? They will lose access.`, confirmLabel: 'Suspend', variant: 'danger' })
        if (ok) suspend.mutate({ id, reason: 'Suspended by admin' })
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Admin', href: '/admin' }, { label: 'Resellers' }]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">Reseller Management</h1>
                    <p className="page-subtitle">{meta.total ?? resellers.length} resellers registered</p>
                </div>
            </div>

            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.1rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0, display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 600 }}>
                    <Users size={17} /> Resellers
                </div>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : resellers.length === 0 ? (
                    <div className="empty-state"><Users size={40} opacity={0.3} /><p>No resellers yet</p></div>
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
                                    {resellers.map((r: any) => (
                                        <tr key={r.id}>
                                            <td>
                                                <p style={{ fontWeight: 600, color: 'var(--text-heading)' }}>{r.company_name}</p>
                                                <p style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>{r.website || '—'}</p>
                                            </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.875rem' }}>{r.email || '—'}</td>
                                            <td>
                                                <span className={`badge badge-${STATUS_COLOR[r.status] ?? 'secondary'}`}>{r.status}</span>
                                            </td>
                                            <td style={{ fontSize: '.875rem', fontWeight: 600 }}>{r.commission_pct ?? 0}%</td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                {new Date(r.created_at).toLocaleDateString()}
                                            </td>
                                            <td>
                                                <div style={{ display: 'flex', gap: '.35rem' }}>
                                                    {r.status === 'pending' && (
                                                        <button className="action-btn green" onClick={() => handleApprove(r.id, r.company_name)} title="Approve">
                                                            <CheckCircle2 size={13} /> Approve
                                                        </button>
                                                    )}
                                                    {r.status !== 'suspended' && (
                                                        <button className="action-btn red" onClick={() => handleSuspend(r.id, r.company_name)} title="Suspend">
                                                            <XCircle size={13} /> Suspend
                                                        </button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                        <Pagination page={page} totalPages={meta.pages ?? 1} total={meta.total ?? resellers.length} limit={20} onPageChange={setPage} />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
