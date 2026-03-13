'use client'
import { useQuery } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import Link from 'next/link'
import { BarChart3, Users, Key, Tag, ArrowRight, Webhook } from 'lucide-react'

function QuickLink({ href, icon: Icon, label, desc, color }: any) {
    return (
        <Link href={href} style={{ textDecoration: 'none' }}>
            <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: 0, cursor: 'pointer', transition: 'all .2s', border: '1px solid var(--border)' }}
                onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = 'rgba(230,168,23,.4)'; (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)' }}
                onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'; (e.currentTarget as HTMLElement).style.transform = '' }}
            >
                <div style={{
                    width: 44, height: 44, borderRadius: 10,
                    background: color, display: 'grid', placeItems: 'center',
                    flexShrink: 0,
                }}>
                    <Icon size={20} color="var(--dc-gold)" />
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
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--dc-gold)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(230,168,23,.15)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Users size={18} color="var(--dc-gold)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Sub-Accounts</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>{subCount}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Managed accounts</p>
                        </div>
                    </div>
                </div>
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--success)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(40,199,111,.12)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Tag size={18} color="var(--success)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Pricing Rules</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>{pricingCount}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Custom pricing overrides</p>
                        </div>
                    </div>
                </div>
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--warning)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(255,159,67,.12)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Key size={18} color="var(--warning)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>API Keys</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>{keysCount}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Active integrations</p>
                        </div>
                    </div>
                </div>
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--info)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(0,207,232,.1)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <BarChart3 size={18} color="var(--info)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Commission</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--text-heading)', lineHeight: 1.2 }}>{dash.commission_pct ?? '—'}{dash.commission_pct ? '%' : ''}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Your commission rate</p>
                        </div>
                    </div>
                </div>
            </div>

            <div>
                <h2 className="section-title" style={{ marginBottom: '1rem' }}>Quick Access</h2>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1rem' }}>
                    <QuickLink href="/reseller/pricing" icon={Tag} label="Pricing Management" desc="Set custom sell prices per product" color="rgba(230,168,23,.12)" />
                    <QuickLink href="/reseller/api-keys" icon={Key} label="API Keys" desc="Manage programmatic access keys" color="rgba(40,199,111,.1)" />
                    <QuickLink href="/reseller/webhooks" icon={Webhook} label="Webhooks" desc="Configure event delivery endpoints" color="rgba(255,159,67,.1)" />
                    <QuickLink href="/reseller/accounts" icon={Users} label="Sub-Accounts" desc="Manage users under your account" color="rgba(0,207,232,.1)" />
                </div>
            </div>
        </AppLayout>
    )
}
