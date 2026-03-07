'use client'
import { useQuery } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { Sidebar } from '@/components/layout/Sidebar'

export default function AdminUsersPage() {
    const { data, isLoading } = useQuery({
        queryKey: ['admin-users'],
        queryFn: () => adminAPI.listUsers(),
    })
    const users = data?.data?.data ?? []
    const total = data?.data?.meta?.total ?? 0

    return (
        <div className="page-layout">
            <Sidebar />
            <main className="page-main">
                <div className="page-header">
                    <h1 className="page-title">Users</h1>
                    <p className="page-subtitle">{total} registered accounts</p>
                </div>

                <section className="section">
                    {isLoading ? (
                        <div className="empty-state">Loading...</div>
                    ) : users.length === 0 ? (
                        <div className="empty-state">No users found</div>
                    ) : (
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Email</th>
                                        <th>Full Name</th>
                                        <th>Role</th>
                                        <th>Status</th>
                                        <th>Created</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {users.map((u: any) => (
                                        <tr key={u.id}>
                                            <td>{u.email}</td>
                                            <td>{u.full_name || '—'}</td>
                                            <td><span className={`badge badge-${u.role}`}>{u.role}</span></td>
                                            <td><span className={`badge badge-${u.status}`}>{u.status}</span></td>
                                            <td>{new Date(u.created_at).toLocaleDateString()}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </section>
            </main>
        </div>
    )
}
