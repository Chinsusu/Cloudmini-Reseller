'use client'
import { useQuery } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { Sidebar } from '@/components/layout/Sidebar'
import { Users, ShieldCheck, TrendingUp, Activity } from 'lucide-react'

function StatCard({ title, value, sub, icon: Icon, color }: any) {
    return (
        <div className="stat-card">
            <div className="stat-icon" style={{ background: color }}>
                <Icon size={22} />
            </div>
            <div>
                <p className="stat-label">{title}</p>
                <p className="stat-value">{value}</p>
                {sub && <p className="stat-sub">{sub}</p>}
            </div>
        </div>
    )
}

export default function AdminPage() {
    const { user } = useAuthStore()
    const { data: users } = useQuery({ queryKey: ['admin-users'], queryFn: () => adminAPI.listUsers() })
    const { data: resellers } = useQuery({ queryKey: ['admin-resellers'], queryFn: () => adminAPI.listResellers() })

    const totalUsers = users?.data?.meta?.total ?? '—'
    const totalResellers = resellers?.data?.data?.length ?? '—'

    return (
        <div className="page-layout">
            <Sidebar />
            <main className="page-main">
                <div className="page-header">
                    <h1 className="page-title">Admin Overview 🛡️</h1>
                    <p className="page-subtitle">Welcome back, {user?.email}</p>
                </div>

                <div className="stats-grid">
                    <StatCard
                        title="Total Users"
                        value={totalUsers}
                        sub="Registered accounts"
                        icon={Users}
                        color="linear-gradient(135deg,#6366f1,#8b5cf6)"
                    />
                    <StatCard
                        title="Resellers"
                        value={totalResellers}
                        sub="Active resellers"
                        icon={ShieldCheck}
                        color="linear-gradient(135deg,#06b6d4,#0ea5e9)"
                    />
                    <StatCard
                        title="Platform"
                        value="Online"
                        sub="All services running"
                        icon={Activity}
                        color="linear-gradient(135deg,#10b981,#059669)"
                    />
                    <StatCard
                        title="Role"
                        value={user?.role ?? 'admin'}
                        sub="Access level"
                        icon={TrendingUp}
                        color="linear-gradient(135deg,#f59e0b,#ef4444)"
                    />
                </div>

                <section className="section">
                    <h2 className="section-title">Recent Resellers</h2>
                    <ResellerTable data={resellers?.data?.data ?? []} />
                </section>
            </main>
        </div>
    )
}

function ResellerTable({ data }: { data: any[] }) {
    if (data.length === 0) {
        return <div className="empty-state">No resellers yet</div>
    }
    return (
        <div className="table-wrapper">
            <table className="data-table">
                <thead>
                    <tr>
                        <th>Email</th><th>Status</th><th>Created</th>
                    </tr>
                </thead>
                <tbody>
                    {data.map((r: any) => (
                        <tr key={r.id}>
                            <td>{r.email}</td>
                            <td><span className={`badge badge-${r.status}`}>{r.status}</span></td>
                            <td>{new Date(r.created_at).toLocaleDateString()}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    )
}
