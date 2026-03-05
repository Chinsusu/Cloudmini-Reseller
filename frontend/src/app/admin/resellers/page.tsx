'use client'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { Sidebar } from '@/components/layout/Sidebar'
import { CheckCircle, XCircle, Users } from 'lucide-react'

export default function AdminResellersPage() {
    const qc = useQueryClient()
    const { data } = useQuery({
        queryKey: ['admin-resellers'],
        queryFn: () => adminAPI.listResellers(),
    })
    const resellers = data?.data?.data ?? []

    const approve = useMutation({
        mutationFn: (id: string) => adminAPI.approveReseller(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-resellers'] }),
    })
    const suspend = useMutation({
        mutationFn: ({ id, reason }: { id: string; reason: string }) =>
            adminAPI.suspendReseller(id, reason),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-resellers'] }),
    })

    return (
        <div className="page-layout">
            <Sidebar />
            <main className="page-main">
                <div className="page-header">
                    <h1 className="page-title">Reseller Management</h1>
                </div>

                <div className="card">
                    <div className="card-header">
                        <Users size={18} />
                        <span>Resellers ({resellers.length})</span>
                    </div>
                    <div className="table-wrapper">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>Company</th><th>Email</th><th>Status</th>
                                    <th>Commission</th><th>Created</th><th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {resellers.map((r: any) => (
                                    <tr key={r.id}>
                                        <td><strong>{r.company_name}</strong></td>
                                        <td>{r.email}</td>
                                        <td>
                                            <span className={`badge badge-${r.status}`}>{r.status}</span>
                                        </td>
                                        <td>{r.commission_pct}%</td>
                                        <td>{new Date(r.created_at).toLocaleDateString()}</td>
                                        <td>
                                            <div className="action-group">
                                                {r.status === 'pending' && (
                                                    <button
                                                        className="action-btn green"
                                                        onClick={() => approve.mutate(r.id)}
                                                        title="Approve"
                                                    >
                                                        <CheckCircle size={14} /> Approve
                                                    </button>
                                                )}
                                                {r.status === 'approved' && (
                                                    <button
                                                        className="action-btn red"
                                                        onClick={() => {
                                                            const reason = prompt('Suspension reason:')
                                                            if (reason) suspend.mutate({ id: r.id, reason })
                                                        }}
                                                        title="Suspend"
                                                    >
                                                        <XCircle size={14} /> Suspend
                                                    </button>
                                                )}
                                            </div>
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
