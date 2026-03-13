'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { formatVND } from '@/lib/format'
import { Users, UserPlus } from 'lucide-react'

export default function ResellerAccountsPage() {
    const [page, setPage] = useState(1)
    const [newUserID, setNewUserID] = useState('')
    const [creditLimit, setCreditLimit] = useState('0')
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()

    const { data, isLoading } = useQuery({
        queryKey: ['reseller-subs', page],
        queryFn: () => resellerAPI.listSubAccounts(page),
    })
    const subs = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const addMut = useMutation({
        mutationFn: () => resellerAPI.createSubAccount(newUserID, creditLimit),
        onSuccess: () => {
            setNewUserID('')
            setCreditLimit('0')
            qc.invalidateQueries({ queryKey: ['reseller-subs'] })
            success('Sub-account added')
        },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Failed to add sub-account'),
    })

    return (
        <AppLayout breadcrumb={[
            { label: 'Reseller', href: '/reseller' },
            { label: 'Sub-Accounts' },
        ]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Sub-Accounts</h1>
                    <p className="page-subtitle">{meta.total ?? 0} managed accounts</p>
                </div>
            </div>

            {/* Add sub-account form */}
            <div className="card">
                <div className="card-header"><UserPlus size={17} /> Add Sub-Account</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 160px auto', gap: '.75rem', alignItems: 'flex-end' }}>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>User ID</label>
                        <input
                            className="input"
                            placeholder="User UUID to add"
                            value={newUserID}
                            onChange={e => setNewUserID(e.target.value)}
                        />
                    </div>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Hạn mức tín dụng (VND)</label>
                        <input
                            className="input"
                            type="number"
                            min="0"
                            step="1"
                            value={creditLimit}
                            onChange={e => setCreditLimit(e.target.value)}
                        />
                    </div>
                    <button
                        className="btn-primary"
                        onClick={() => addMut.mutate()}
                        disabled={!newUserID || addMut.isPending}
                    >
                        <UserPlus size={15} />
                        {addMut.isPending ? 'Adding...' : 'Add'}
                    </button>
                </div>
            </div>

            {/* Sub-accounts table */}
            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
                    <Users size={17} /> Sub-Accounts ({subs.length})
                </div>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : subs.length === 0 ? (
                    <div className="empty-state">
                        <Users size={40} opacity={0.3} />
                        <p>No sub-accounts yet</p>
                        <p style={{ fontSize: '.8rem' }}>Add a user's UUID above to grant them access under your account</p>
                    </div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>User ID</th>
                                        <th>Credit Limit</th>
                                        <th>Added</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {subs.map((s: any) => (
                                        <tr key={s.id}>
                                            <td><code className="font-mono" style={{ fontSize: '.8rem' }}>{s.user_id}</code></td>
                                            <td>
                                                <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem', fontWeight: 600 }}>
                                                    {formatVND(s.credit_limit)}
                                                </span>
                                            </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                {new Date(s.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
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
                            limit={20}
                            onPageChange={setPage}
                        />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
