'use client'
import { useQuery } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import Link from 'next/link'
import { BarChart3, Users, DollarSign, Key, Tag, ArrowRight, Webhook } from 'lucide-react'

function QuickLink({ href, icon: Icon, label, desc, color }: any) {
    return (
        <Link href={href} style={{ textDecoration: 'none' }}>
            <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: 0, cursor: 'pointer', transition: 'all .2s' }}
                onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)' }}
                onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = '' }}
            >
                <div style={{
                    width: 48, height: 48, borderRadius: 10,
                    background: color, display: 'grid', placeItems: 'center',
                    color: 'white', flexShrink: 0,
                }}>
                    <Icon size={20} />
                </div>
                <div style={{ flex: 1 }}>
                    <p style={{ fontWeight: 600, color: 'var(--text-heading)', fontSize: '.9375rem' }}>{label}</p>
                    <p style={{ fontSize: '.8rem', color: 'var(--text-muted)' }}>{desc}</p>
                </div>
                <ArrowRight size={16} color="var(--text-muted)" />
            </div>
        </Link>
    )
}

export default function ResellerDashboardPage() {
    const { data: dashData } = useQuery({
        queryKey: ['reseller-dashboard'],
        queryFn: () => resellerAPI.getDashboard(),
    })
    const { data: subData } = useQuery({
        queryKey: ['reseller-subaccounts'],
        queryFn: () => resellerAPI.listSubAccounts(),
    })
    const { data: pricingData } = useQuery({
        queryKey: ['reseller-pricing'],
        queryFn: () => resellerAPI.listPricing(),
    })
    const { data: keysData } = useQuery({
        queryKey: ['api-keys'],
        queryFn: () => resellerAPI.listAPIKeys(),
    })

    const dash = dashData?.data?.data ?? {}
    const subCount = subData?.data?.meta?.total ?? subData?.data?.data?.length ?? 0
    const pricingCount = pricingData?.data?.data?.length ?? 0
    const keysCount = keysData?.data?.data?.length ?? 0

    return (
        <AppLayout breadcrumb={[{ label: 'Reseller Dashboard' }]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Reseller Dashboard</h1>
                    <p className="page-subtitle">Overview of your reseller account</p>
                </div>
            </div>

            {/* Stats */}
            <div className="stats-grid">
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#7367F0,#9e95f5)' }}>
                        <Users size={22} />
                    </div>
                    <div>
                        <p className="stat-label">Sub-Accounts</p>
                        <p className="stat-value">{subCount}</p>
                        <p className="stat-sub">Managed accounts</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#28C76F,#48DA89)' }}>
                        <Tag size={22} />
                    </div>
                    <div>
                        <p className="stat-label">Pricing Rules</p>
                        <p className="stat-value">{pricingCount}</p>
                        <p className="stat-sub">Custom pricing overrides</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#FF9F43,#FFB976)' }}>
                        <Key size={22} />
                    </div>
                    <div>
                        <p className="stat-label">API Keys</p>
                        <p className="stat-value">{keysCount}</p>
                        <p className="stat-sub">Active integrations</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#00CFE8,#1EDEC5)' }}>
                        <DollarSign size={22} />
                    </div>
                    <div>
                        <p className="stat-label">Commission</p>
                        <p className="stat-value">{dash.commission_pct ?? '—'}{dash.commission_pct ? '%' : ''}</p>
                        <p className="stat-sub">Your commission rate</p>
                    </div>
                </div>
            </div>

            {/* Quick Links */}
            <div>
                <h2 className="section-title" style={{ marginBottom: '1rem' }}>Quick Access</h2>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1rem' }}>
                    <QuickLink
                        href="/reseller/pricing"
                        icon={Tag}
                        label="Pricing Management"
                        desc="Set custom sell prices per product"
                        color="linear-gradient(135deg,#7367F0,#9e95f5)"
                    />
                    <QuickLink
                        href="/reseller/api-keys"
                        icon={Key}
                        label="API Keys"
                        desc="Manage programmatic access keys"
                        color="linear-gradient(135deg,#28C76F,#48DA89)"
                    />
                    <QuickLink
                        href="/reseller/webhooks"
                        icon={Webhook}
                        label="Webhooks"
                        desc="Configure event delivery endpoints"
                        color="linear-gradient(135deg,#FF9F43,#FFB976)"
                    />
                    <QuickLink
                        href="/reseller/accounts"
                        icon={Users}
                        label="Sub-Accounts"
                        desc="Manage users under your account"
                        color="linear-gradient(135deg,#00CFE8,#1EDEC5)"
                    />
                </div>
            </div>
        </AppLayout>
    )
}
