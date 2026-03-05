'use client'
import { useQuery } from '@tanstack/react-query'
import { walletAPI, proxyAPI, vpsAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { Sidebar } from '@/components/layout/Sidebar'
import { Wallet, Globe, Server, TrendingUp } from 'lucide-react'

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

export default function DashboardPage() {
    const { user } = useAuthStore()
    const { data: wallet } = useQuery({ queryKey: ['wallet'], queryFn: () => walletAPI.getBalance() })
    const { data: orders } = useQuery({ queryKey: ['proxy-orders'], queryFn: () => proxyAPI.listOrders() })
    const { data: vps } = useQuery({ queryKey: ['vps-instances'], queryFn: () => vpsAPI.listInstances() })

    const balance = wallet?.data?.data?.balance ?? '0.00'
    const orderCount = orders?.data?.meta?.total ?? 0
    const vpsCount = vps?.data?.meta?.total ?? 0

    return (
        <div className="page-layout">
            <Sidebar />
            <main className="page-main">
                <div className="page-header">
                    <h1 className="page-title">Welcome back, {user?.fullName || user?.email} 👋</h1>
                    <p className="page-subtitle">Here's an overview of your account</p>
                </div>

                <div className="stats-grid">
                    <StatCard
                        title="Wallet Balance"
                        value={`$${balance}`}
                        sub="Available credit"
                        icon={Wallet}
                        color="linear-gradient(135deg,#6366f1,#8b5cf6)"
                    />
                    <StatCard
                        title="Proxy Orders"
                        value={orderCount}
                        sub="Total orders"
                        icon={Globe}
                        color="linear-gradient(135deg,#06b6d4,#0ea5e9)"
                    />
                    <StatCard
                        title="VPS Instances"
                        value={vpsCount}
                        sub="Total instances"
                        icon={Server}
                        color="linear-gradient(135deg,#10b981,#059669)"
                    />
                    <StatCard
                        title="This Month"
                        value="Active"
                        sub="Account status"
                        icon={TrendingUp}
                        color="linear-gradient(135deg,#f59e0b,#ef4444)"
                    />
                </div>

                {/* Recent Transactions */}
                <section className="section">
                    <h2 className="section-title">Recent Transactions</h2>
                    <RecentTransactions />
                </section>
            </main>
        </div>
    )
}

function RecentTransactions() {
    const { data } = useQuery({ queryKey: ['txs'], queryFn: () => walletAPI.getTransactions() })
    const txs = data?.data?.data ?? []

    if (txs.length === 0) {
        return <div className="empty-state">No transactions yet</div>
    }

    return (
        <div className="table-wrapper">
            <table className="data-table">
                <thead>
                    <tr>
                        <th>Date</th><th>Type</th><th>Amount</th><th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    {txs.map((tx: any) => (
                        <tr key={tx.id}>
                            <td>{new Date(tx.created_at).toLocaleDateString()}</td>
                            <td><span className={`badge badge-${tx.type}`}>{tx.type}</span></td>
                            <td className={tx.type === 'deduct' ? 'text-red' : 'text-green'}>${tx.amount}</td>
                            <td><span className="badge badge-success">{tx.status}</span></td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    )
}
