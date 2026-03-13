'use client'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { walletAPI, proxyAPI, vpsAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { AppLayout } from '@/components/layout/AppLayout'
import { formatVND } from '@/lib/format'
import { Wallet, Globe, Server, Activity, ArrowUpRight, AlertTriangle, CheckCircle2, Info } from 'lucide-react'

// ─── Stat Card ───────────────────────────────────────────────────────────────
function StatCard({ title, value, sub, icon: Icon, accent }: {
    title: string; value: string | number; sub?: string; icon: any; accent: string
}) {
    return (
        <div style={{
            background: 'var(--surface)', border: '1px solid var(--border)',
            borderTop: `3px solid ${accent}`,
            borderRadius: '0 0 var(--radius-xl) var(--radius-xl)',
            padding: '1.35rem 1.5rem',
            display: 'flex', alignItems: 'flex-start', gap: '1rem',
            boxShadow: 'var(--shadow-sm)',
            transition: 'box-shadow .2s, transform .2s', cursor: 'default',
        }}>
            <div style={{
                width: 44, height: 44, borderRadius: 10,
                background: `${accent}18`, display: 'grid', placeItems: 'center', flexShrink: 0,
            }}>
                <Icon size={20} color={accent} />
            </div>
            <div>
                <p style={{ fontSize: '.75rem', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em', marginBottom: '.25rem' }}>
                    {title}
                </p>
                <p style={{ fontSize: '1.7rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>
                    {value}
                </p>
                {sub && <p style={{ fontSize: '.78rem', color: 'var(--text-muted)', marginTop: '.2rem' }}>{sub}</p>}
            </div>
        </div>
    )
}

// ─── Sample announcements (replace with API later) ────────────────────────────
const ANNOUNCEMENTS = [
    // { type: 'warning', msg: 'Proxy location US-Dallas đang bảo trì, dự kiến khôi phục 14:00 UTC.' },
    // { type: 'info', msg: 'Nâng cấp hệ thống vào 02:00 UTC ngày 20/03, downtime ~10 phút.' },
]

const ICON_MAP: Record<string, any> = { warning: AlertTriangle, info: Info, success: CheckCircle2 }
const COLOR_MAP: Record<string, string> = {
    warning: 'var(--warning)',
    info: 'var(--primary)',
    success: 'var(--success)',
}
const BG_MAP: Record<string, string> = {
    warning: 'rgba(255,159,67,.08)',
    info: 'rgba(115,103,240,.08)',
    success: 'rgba(40,199,111,.08)',
}

export default function DashboardPage() {
    const { user } = useAuthStore()
    const { data: wallet } = useQuery({ queryKey: ['wallet'], queryFn: () => walletAPI.getBalance() })
    const { data: orders } = useQuery({ queryKey: ['proxy-orders'], queryFn: () => proxyAPI.listOrders() })
    const { data: vps } = useQuery({ queryKey: ['vps-instances'], queryFn: () => vpsAPI.listInstances() })
    const { data: txData } = useQuery({ queryKey: ['txs'], queryFn: () => walletAPI.getTransactions() })

    const balance = wallet?.data?.data?.balance ?? '0.00'
    const orderCount = orders?.data?.meta?.total ?? 0
    const vpsCount = vps?.data?.meta?.total ?? 0
    const txs: any[] = (txData?.data?.data ?? []).slice(0, 8)

    const hour = new Date().getHours()
    const greeting = hour < 12 ? 'Good morning' : hour < 18 ? 'Good afternoon' : 'Good evening'
    const firstName = user?.fullName?.split(' ')[0] || user?.email?.split('@')[0] || 'there'

    return (
        <AppLayout breadcrumb={[{ label: 'Home', href: '/dashboard' }, { label: 'Dashboard' }]}>

            {/* ── System Announcements banner ── */}
            <div style={{
                background: 'linear-gradient(135deg, var(--dc-dark) 0%, #2a2a2d 100%)',
                borderRadius: 'var(--radius-xl)',
                padding: '1.6rem 2rem',
                marginBottom: '1.75rem',
                border: '1px solid rgba(230,168,23,.15)',
            }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: ANNOUNCEMENTS.length > 0 ? '1rem' : 0 }}>
                    <div>
                        <p style={{ color: 'rgba(255,255,255,.45)', fontSize: '.82rem', marginBottom: '.25rem' }}>
                            {greeting}, <span style={{ color: 'var(--dc-gold)', fontWeight: 600 }}>{firstName}</span>
                        </p>
                        <h1 style={{ fontSize: '1.15rem', fontWeight: 700, color: '#fff' }}>
                            System Announcements
                        </h1>
                    </div>
                    <span style={{
                        padding: '.25rem .75rem',
                        background: ANNOUNCEMENTS.length > 0 ? 'rgba(255,159,67,.18)' : 'rgba(40,199,111,.15)',
                        color: ANNOUNCEMENTS.length > 0 ? 'var(--warning)' : 'var(--success)',
                        borderRadius: 'var(--radius-pill)', fontSize: '.75rem', fontWeight: 700,
                    }}>
                        {ANNOUNCEMENTS.length > 0 ? `${ANNOUNCEMENTS.length} active` : '✓ All systems operational'}
                    </span>
                </div>

                {ANNOUNCEMENTS.length > 0 ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '.6rem' }}>
                        {ANNOUNCEMENTS.map((a, i) => {
                            const Icon = ICON_MAP[a.type] ?? Info
                            return (
                                <div key={i} style={{
                                    display: 'flex', alignItems: 'flex-start', gap: '.75rem',
                                    background: BG_MAP[a.type],
                                    border: `1px solid ${COLOR_MAP[a.type]}30`,
                                    borderLeft: `3px solid ${COLOR_MAP[a.type]}`,
                                    borderRadius: '6px', padding: '.65rem .9rem',
                                }}>
                                    <Icon size={15} color={COLOR_MAP[a.type]} style={{ flexShrink: 0, marginTop: '.1rem' }} />
                                    <p style={{ fontSize: '.85rem', color: 'rgba(255,255,255,.8)', lineHeight: 1.5 }}>{a.msg}</p>
                                </div>
                            )
                        })}
                    </div>
                ) : (
                    <p style={{ fontSize: '.82rem', color: 'rgba(255,255,255,.32)', marginTop: '.5rem' }}>
                        No active notices at this time. We'll notify you here about maintenance, outages, or important updates.
                    </p>
                )}
            </div>

            {/* ── Stat cards ── */}
            <div className="stats-grid" style={{ marginBottom: '1.75rem' }}>
                <StatCard title="Wallet Balance" value={formatVND(balance)} sub="Số dư khả dụng"
                    icon={Wallet} accent="var(--dc-gold)" />
                <StatCard title="Proxy Orders" value={orderCount} sub="Total orders"
                    icon={Globe} accent="#00CFE8" />
                <StatCard title="VPS Instances" value={vpsCount} sub="Total instances"
                    icon={Server} accent="#28C76F" />
                <StatCard title="Account Status" value="Active" sub="All systems operational"
                    icon={Activity} accent="#FF9F43" />
            </div>

            {/* ── Recent Activities ── */}
            <div>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '.75rem' }}>
                    <p style={{ fontSize: '.72rem', fontWeight: 700, letterSpacing: '.08em', textTransform: 'uppercase', color: 'var(--text-muted)' }}>
                        Recent Activities
                    </p>
                    <Link href="/dashboard/wallet" style={{ fontSize: '.8rem', color: 'var(--dc-gold)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '.2rem', fontWeight: 600 }}>
                        View all <ArrowUpRight size={13} />
                    </Link>
                </div>
                <div className="card" style={{ padding: 0, marginBottom: 0 }}>
                    <RecentActivities txs={txs} />
                </div>
            </div>
        </AppLayout>
    )
}

function RecentActivities({ txs }: { txs: any[] }) {
    if (txs.length === 0) {
        return (
            <div style={{ padding: '3rem', textAlign: 'center' }}>
                <Activity size={32} color="var(--text-muted)" style={{ marginBottom: '.75rem', opacity: .35 }} />
                <p style={{ color: 'var(--text-muted)', fontSize: '.875rem', marginBottom: '.5rem' }}>No recent activity</p>
                <p style={{ color: 'var(--text-muted)', fontSize: '.8rem', opacity: .6 }}>Your transactions and orders will appear here.</p>
            </div>
        )
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
                                <td style={{ fontWeight: 700, color: isCredit ? 'var(--success)' : 'var(--error)', fontVariantNumeric: 'tabular-nums' }}>
                                    {isCredit ? '+' : '-'}{formatVND(tx.amount)}
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
