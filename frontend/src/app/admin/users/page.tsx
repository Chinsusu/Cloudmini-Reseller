'use client'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { Users } from 'lucide-react'

export default function UsersPage() {
    const [page, setPage] = useState(1)

    const { data, isLoading } = useQuery({
        queryKey: ['admin-users', page],
        queryFn: () => adminAPI.listUsers(page),
    })

    const users = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    return (
        <AppLayout breadcrumb={[
            { label: 'Admin', href: '/admin' },
            { label: 'Users' }
        ]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Users</h1>
                    <p className="page-subtitle">{meta.total ?? 0} total accounts</p>
                </div>
            </div>

            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
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
                                        <th>Joined</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {users.map((u: any) => {
                                        const initials = u.full_name
                                            ? u.full_name.split(' ').map((n: string) => n[0]).join('').toUpperCase().slice(0, 2)
                                            : (u.email?.[0] ?? '?').toUpperCase()
                                        return (
                                            <tr key={u.id}>
                                                <td>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                                                        <div style={{
                                                            width: 34, height: 34, borderRadius: '50%',
                                                            background: 'var(--primary-light)', color: 'var(--primary)',
                                                            display: 'grid', placeItems: 'center',
                                                            fontWeight: 700, fontSize: '.78rem', flexShrink: 0
                                                        }}>{initials}</div>
                                                        <div>
                                                            <p style={{ fontWeight: 600, color: 'var(--text-heading)', fontSize: '.875rem' }}>
                                                                {u.full_name || '—'}
                                                            </p>
                                                            <p style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>{u.email}</p>
                                                        </div>
                                                    </div>
                                                </td>
                                                <td><span className={`badge badge-${u.role}`}>{u.role}</span></td>
                                                <td><span className={`badge badge-${u.status}`}>{u.status}</span></td>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                                    {new Date(u.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
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
