'use client'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { walletAPI, proxyAPI, vpsAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { AppLayout } from '@/components/layout/AppLayout'
import { Wallet, Globe, Server, Activity } from 'lucide-react'

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
        <AppLayout breadcrumb={[
            { label: 'Home', href: '/dashboard' },
            { label: 'Dashboard' },
        ]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Welcome back, {user?.fullName || user?.email}</h1>
                    <p className="page-subtitle">Overview of your account</p>
                </div>
            </div>

            <div className="stats-grid">
                <StatCard
                    title="Wallet Balance" value={`$${balance}`} sub="Available credit"
                    icon={Wallet} color="linear-gradient(135deg,#7367F0,#9e95f5)"
                />
                <StatCard
                    title="Proxy Orders" value={orderCount} sub="Total orders"
                    icon={Globe} color="linear-gradient(135deg,#00CFE8,#1EDEC5)"
                />
                <StatCard
                    title="VPS Instances" value={vpsCount} sub="Total instances"
                    icon={Server} color="linear-gradient(135deg,#28C76F,#48DA89)"
                />
                <StatCard
                    title="Account Status" value="Active" sub="All systems operational"
                    icon={Activity} color="linear-gradient(135deg,#FF9F43,#FFB976)"
                />
            </div>

            <section className="section">
                <h2 className="section-title">Recent Transactions</h2>
                <div className="card" style={{ padding: 0, marginBottom: 0 }}>
                    <RecentTransactions />
                </div>
            </section>
        </AppLayout>
    )
}

function RecentTransactions() {
    const { data } = useQuery({ queryKey: ['txs'], queryFn: () => walletAPI.getTransactions() })
    const txs = data?.data?.data ?? []

    if (txs.length === 0) {
        return <div className="empty-state"><p>No transactions yet</p></div>
    }

    return (
        <div className="table-wrapper">
            <table className="data-table">
                <thead>
                    <tr>
                        <th>Date</th>
                        <th>Type</th>
                        <th>Description</th>
                        <th>Amount</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    {txs.map((tx: any) => {
                        const isCredit = ['deposit', 'refund', 'hold_release', 'adjustment'].includes(tx.type)
                        return (
                            <tr key={tx.id}>
                                <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                    {new Date(tx.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                                </td>
                                <td><span className={`badge badge-${tx.type}`}>{tx.type.replace(/_/g, ' ')}</span></td>
                                <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>{tx.description || '—'}</td>
                                <td style={{ fontWeight: 600, color: isCredit ? 'var(--success)' : 'var(--error)' }}>
                                    {isCredit ? '+' : '-'}${tx.amount}
                                </td>
                                <td><span className="badge badge-success">{tx.status}</span></td>
                            </tr>
                        )
                    })}
                </tbody>
            </table>
        </div>
    )
}
