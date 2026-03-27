'use client'
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { proxyAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import {
    Globe, CheckCircle2, ChevronRight, Loader2,
    Zap, ArrowLeft, ShoppingCart
} from 'lucide-react'
import { formatVND } from '@/lib/format'
import { useRouter } from 'next/navigation'

// ─── Proxy-type metadata ──────────────────────────────────────────────────────
const PROXY_TYPE_META: Record<string, { color: string; bg: string; label: string }> = {
    residential: { color: '#4ade80', bg: 'rgba(74,222,128,.12)', label: 'Residential' },
    datacenter: { color: '#60a5fa', bg: 'rgba(96,165,250,.12)', label: 'Datacenter' },
    mobile: { color: '#f472b6', bg: 'rgba(244,114,182,.12)', label: 'Mobile' },
    isp: { color: '#a78bfa', bg: 'rgba(167,139,250,.12)', label: 'ISP' },
}

// ─── Product Card ─────────────────────────────────────────────────────────────
function ProductCard({ product, selected, onClick }: { product: any; selected: boolean; onClick: () => void }) {
    const isRotating = product.metadata?.service_id?.includes('rotating')
    const meta = PROXY_TYPE_META[product.proxy_type?.toLowerCase()] ?? {
        color: 'var(--dc-gold)', bg: 'rgba(230,168,23,.12)', label: product.proxy_type ?? 'Proxy',
    }

    return (
        <div
            onClick={onClick}
            style={{
                background: selected ? 'rgba(230,168,23,.05)' : 'var(--surface)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-xl)',
                padding: '1.35rem',
                cursor: 'pointer',
                display: 'flex',
                flexDirection: 'column',
                gap: '.75rem',
                transition: 'transform .18s ease, box-shadow .18s ease',
                boxShadow: selected
                    ? '0 0 0 2px var(--dc-gold), 0 8px 24px rgba(230,168,23,.15)'
                    : '0 2px 8px rgba(0,0,0,.15)',
                transform: selected ? 'translateY(-2px)' : undefined,
            }}
            onMouseEnter={e => {
                if (!selected) (e.currentTarget as HTMLDivElement).style.transform = 'translateY(-3px)'
                if (!selected) (e.currentTarget as HTMLDivElement).style.boxShadow = '0 8px 24px rgba(0,0,0,.25)'
            }}
            onMouseLeave={e => {
                if (!selected) (e.currentTarget as HTMLDivElement).style.transform = ''
                if (!selected) (e.currentTarget as HTMLDivElement).style.boxShadow = '0 2px 8px rgba(0,0,0,.15)'
            }}
        >
            {/* Type badge + checkmark */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span style={{
                    display: 'inline-flex', alignItems: 'center', gap: '.3rem',
                    fontSize: '.7rem', fontWeight: 700, textTransform: 'uppercase',
                    letterSpacing: '.07em',
                    color: meta.color,
                    background: meta.bg,
                    padding: '.2rem .65rem',
                    borderRadius: 'var(--radius-pill)',
                }}>
                    {meta.label}
                    {product.protocol && <span style={{ opacity: .65 }}>· {product.protocol.toUpperCase()}</span>}
                </span>
                {selected && (
                    <div style={{
                        width: 20, height: 20, borderRadius: '50%',
                        background: 'var(--dc-gold)',
                        display: 'grid', placeItems: 'center', flexShrink: 0,
                    }}>
                        <CheckCircle2 size={12} color="#000" strokeWidth={3} />
                    </div>
                )}
            </div>

            {/* Name */}
            <div style={{ fontWeight: 800, fontSize: '1.05rem', color: 'var(--text-heading)', lineHeight: 1.25 }}>
                {product.name}
            </div>

            {/* Feature pills */}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '.3rem' }}>
                {product.location && (
                    <span style={{
                        display: 'inline-flex', alignItems: 'center', gap: '.2rem',
                        padding: '.15rem .5rem', borderRadius: '100px', fontSize: '.7rem',
                        background: 'var(--surface-raised)', color: 'var(--text)',
                    }}>📍 {product.location}</span>
                )}
                {product.protocol && (
                    <span style={{
                        display: 'inline-flex', alignItems: 'center', gap: '.2rem',
                        padding: '.15rem .5rem', borderRadius: '100px', fontSize: '.7rem',
                        background: 'var(--surface-raised)', color: 'var(--text)',
                    }}>🔌 {product.protocol.toUpperCase()}</span>
                )}
                {isRotating && (
                    <span style={{
                        display: 'inline-flex', alignItems: 'center', gap: '.2rem',
                        padding: '.15rem .5rem', borderRadius: '100px', fontSize: '.7rem',
                        background: 'rgba(230,168,23,.08)', color: 'var(--dc-gold)',
                    }}>🔄 Rotating</span>
                )}
            </div>

            {/* Price + CTA */}
            <div style={{ marginTop: 'auto', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div>
                    <span style={{ fontWeight: 900, fontSize: '1.15rem', color: 'var(--dc-gold)' }}>
                        {formatVND(parseFloat(product.base_cost))}
                    </span>
                    <span style={{ fontSize: '.72rem', color: 'var(--text-muted)', marginLeft: '.2rem' }}>
                        {isRotating ? '/GB' : '/tháng'}
                    </span>
                </div>

                <button style={{
                    display: 'inline-flex', alignItems: 'center', gap: '.3rem',
                    padding: '.42rem .9rem',
                    background: selected ? 'var(--dc-gold)' : 'transparent',
                    color: selected ? 'var(--dc-gold-text)' : meta.color,
                    border: `1px solid ${selected ? 'var(--dc-gold)' : meta.color}`,
                    borderRadius: 'var(--radius)',
                    fontWeight: 700, fontSize: '.8rem',
                    cursor: 'pointer',
                    transition: 'all .15s',
                    flexShrink: 0,
                }}>
                    {selected ? <><CheckCircle2 size={13} /> Đã chọn</> : <>Chọn <ChevronRight size={13} /></>}
                </button>
            </div>
        </div>
    )
}

// ─── Order Panel ──────────────────────────────────────────────────────────────
function OrderPanel({ product, onClose, onSuccess }: { product: any; onClose: () => void; onSuccess: () => void }) {
    const { success, error: toastError } = useToast()
    const qc = useQueryClient()
    const [qty, setQty] = useState(1)
    const [country, setCountry] = useState('')
    const [ispId, setIspId] = useState('')
    const [periodMonths, setPeriodMonths] = useState(1)
    const [protocol, setProtocol] = useState('default')
    const [groupId, setGroupId] = useState('')
    const [placing, setPlacing] = useState(false)

    const meta = product.metadata ?? {}
    const serviceId = meta.service_id ?? ''
    const planId = meta.plan_id ?? ''
    const isRotating = serviceId.includes('rotating')
    const isStatic = !isRotating && serviceId !== ''
    const unitLabel = isRotating ? '/GB' : '/tháng'
    const total = formatVND(parseFloat(product.base_cost) * qty * (isStatic ? periodMonths : 1))

    const { data: optData, isLoading: optLoading } = useQuery({
        queryKey: ['service-options', serviceId, planId],
        queryFn: () => proxyAPI.serviceOptions(serviceId, planId),
        enabled: !!serviceId && isStatic,
        staleTime: 300_000,
    })
    const countries: string[] = optData?.data?.data?.countries ?? []
    const isps: { id: string; name: string }[] = country && optData?.data?.data?.isps
        ? (optData.data.data.isps[country] ?? [])
        : []

    // Fetch groups for this product (VPM only)
    const { data: groupsData } = useQuery({
        queryKey: ['product-groups', product.id],
        queryFn: () => proxyAPI.getProductGroups(product.id),
        staleTime: 120_000,
    })
    const groups: any[] = groupsData?.data?.data ?? groupsData?.data ?? []

    const handleOrder = async () => {
        if (isStatic && countries.length > 0 && !country) {
            toastError('Please select a country')
            return
        }
        if (groups.length > 0 && !groupId) {
            toastError('Vui lòng chọn khu vực')
            return
        }
        const payload = {
            product_id: product.id,
            quantity: qty,
            metadata: {
                ...(serviceId ? { service_id: serviceId } : {}),
                ...(planId ? { plan_id: planId } : {}),
                ...(country ? { country_code: country } : {}),
                ...(ispId ? { isp_id: ispId } : {}),
                ...(isStatic ? { period_months: String(periodMonths) } : {}),
                ...(groupId ? { group_id: groupId } : {}),
                protocol,
            },
        }
        setPlacing(true)
        try {
            await proxyAPI.createOrder(payload.product_id, payload.quantity, payload.metadata)
            success('Đặt hàng thành công! 🎉')
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
            onSuccess()
        } catch (err: any) {
            console.error('[OrderPanel] createOrder error full:', err)
            toastError(err?.response?.data?.error?.message ?? err?.message ?? 'Đặt hàng thất bại')
        } finally {
            setPlacing(false)
        }
    }

    const pmeta = PROXY_TYPE_META[product.proxy_type?.toLowerCase()] ?? { color: 'var(--dc-gold)', bg: 'rgba(230,168,23,.12)', label: 'Proxy' }

    return (
        <div className="card fade-in" style={{ marginTop: '1.25rem', padding: '1.5rem' }}>
            {/* Header */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.25rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem' }}>
                    <button onClick={onClose} className="topbar-icon-btn"><ArrowLeft size={16} /></button>
                    <h3 style={{ margin: 0, fontSize: '1rem', color: 'var(--text-heading)' }}>Đặt hàng: {product.name}</h3>
                </div>
                <span style={{
                    padding: '.2rem .65rem', borderRadius: 'var(--radius-pill)',
                    background: pmeta.bg, color: pmeta.color,
                    fontSize: '.7rem', fontWeight: 700, textTransform: 'uppercase',
                }}>{pmeta.label}</span>
            </div>

            {/* Protocol selector */}
            <div style={{ marginBottom: '1rem' }}>
                <label style={{ display: 'block', fontSize: '.78rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.45rem' }}>
                    Protocol
                </label>
                <div style={{ display: 'flex', gap: '.4rem', flexWrap: 'wrap' }}>
                    {[
                        { value: 'default', label: '⚡ Default', desc: 'HTTP + SOCKS5' },
                        { value: 'http', label: '🌐 HTTP', desc: 'HTTP only' },
                        { value: 'socks5', label: '🔌 SOCKS5', desc: 'SOCKS5 only' },
                    ].map(opt => (
                        <button key={opt.value} onClick={() => setProtocol(opt.value)}
                            style={{
                                display: 'flex', flexDirection: 'column', alignItems: 'center',
                                padding: '.5rem 1rem',
                                borderRadius: 'var(--radius)',
                                border: protocol === opt.value ? '1.5px solid var(--dc-gold)' : '1px solid var(--border)',
                                background: protocol === opt.value ? 'rgba(230,168,23,.08)' : 'var(--surface)',
                                color: protocol === opt.value ? 'var(--dc-gold)' : 'var(--text)',
                                cursor: 'pointer', transition: 'all .15s', gap: '.15rem',
                            }}>
                            <span style={{ fontWeight: 700, fontSize: '.82rem' }}>{opt.label}</span>
                            <span style={{ fontSize: '.65rem', color: 'var(--text-muted)' }}>{opt.desc}</span>
                        </button>
                    ))}
                </div>
            </div>

            {/* Group selector — VPM products */}
            {groups.length > 0 && (
                <div style={{ marginBottom: '1rem' }}>
                    <label style={{ display: 'block', fontSize: '.78rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.45rem' }}>
                        Khu vực
                    </label>
                    <div style={{ display: 'flex', gap: '.4rem', flexWrap: 'wrap' }}>
                        {groups.map((g: any) => (
                            <button key={g.id} onClick={() => setGroupId(g.id)}
                                style={{
                                    display: 'flex', flexDirection: 'column', alignItems: 'center',
                                    padding: '.45rem .9rem',
                                    borderRadius: 'var(--radius)',
                                    border: groupId === g.id ? '1.5px solid var(--dc-gold)' : '1px solid var(--border)',
                                    background: groupId === g.id ? 'rgba(230,168,23,.08)' : 'var(--surface)',
                                    color: groupId === g.id ? 'var(--dc-gold)' : 'var(--text)',
                                    cursor: 'pointer', transition: 'all .15s', gap: '.1rem',
                                }}>
                                <span style={{ fontWeight: 700, fontSize: '.82rem' }}>📍 {g.name}</span>
                                {g.available_ips > 0 && (
                                    <span style={{ fontSize: '.62rem', color: 'var(--text-muted)' }}>{g.available_ips} IPs</span>
                                )}
                            </button>
                        ))}
                    </div>
                </div>
            )}

            {/* Config form */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: '.75rem', marginBottom: '1rem' }}>
                <div>
                    <label style={{ display: 'block', fontSize: '.78rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.35rem' }}>Số lượng</label>
                    <input type="number" min={1} max={100} value={qty} onChange={e => setQty(Math.max(1, +e.target.value))}
                        style={{
                            width: '100%', padding: '.5rem .65rem',
                            background: 'var(--surface-raised)', border: '1px solid var(--border)',
                            borderRadius: 'var(--radius)', color: 'var(--text-heading)',
                            fontSize: '.88rem',
                        }} />
                </div>
                {isStatic && (
                    <div>
                        <label style={{ display: 'block', fontSize: '.78rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.35rem' }}>Thời gian</label>
                        <select value={periodMonths} onChange={e => setPeriodMonths(+e.target.value)}
                            style={{
                                width: '100%', padding: '.5rem .65rem',
                                background: 'var(--surface-raised)', border: '1px solid var(--border)',
                                borderRadius: 'var(--radius)', color: 'var(--text-heading)',
                                fontSize: '.88rem',
                            }}>
                            {[1, 3, 6, 12].map(m => <option key={m} value={m}>{m} tháng</option>)}
                        </select>
                    </div>
                )}
                {isStatic && countries.length > 0 && (
                    <div>
                        <label style={{ display: 'block', fontSize: '.78rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.35rem' }}>Quốc gia</label>
                        {optLoading ? <Loader2 size={16} className="spin" /> : (
                            <select value={country} onChange={e => { setCountry(e.target.value); setIspId('') }}
                                style={{
                                    width: '100%', padding: '.5rem .65rem',
                                    background: 'var(--surface-raised)', border: '1px solid var(--border)',
                                    borderRadius: 'var(--radius)', color: 'var(--text-heading)',
                                    fontSize: '.88rem',
                                }}>
                                <option value="">— Chọn —</option>
                                {countries.map(c => <option key={c} value={c}>{c}</option>)}
                            </select>
                        )}
                    </div>
                )}
                {isStatic && isps.length > 0 && (
                    <div>
                        <label style={{ display: 'block', fontSize: '.78rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.35rem' }}>ISP</label>
                        <select value={ispId} onChange={e => setIspId(e.target.value)}
                            style={{
                                width: '100%', padding: '.5rem .65rem',
                                background: 'var(--surface-raised)', border: '1px solid var(--border)',
                                borderRadius: 'var(--radius)', color: 'var(--text-heading)',
                                fontSize: '.88rem',
                            }}>
                            <option value="">— Tự động —</option>
                            {isps.map(i => <option key={i.id} value={i.id}>{i.name}</option>)}
                        </select>
                    </div>
                )}
            </div>

            {/* Order summary + CTA */}
            <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '.85rem 1rem',
                background: 'rgba(230,168,23,.04)',
                borderRadius: 'var(--radius)',
                border: '1px solid rgba(230,168,23,.15)',
            }}>
                <div>
                    <div style={{ fontSize: '.72rem', color: 'var(--text-muted)' }}>Tổng thanh toán</div>
                    <div style={{ fontWeight: 900, fontSize: '1.2rem', color: 'var(--dc-gold)' }}>{total}</div>
                    <div style={{ fontSize: '.68rem', color: 'var(--text-muted)' }}>
                        {qty} × {formatVND(parseFloat(product.base_cost))}{unitLabel}
                        {isStatic && periodMonths > 1 ? ` × ${periodMonths} tháng` : ''}
                    </div>
                </div>
                <button onClick={handleOrder} disabled={placing}
                    style={{
                        display: 'flex', alignItems: 'center', gap: '.4rem',
                        padding: '.55rem 1.5rem',
                        background: 'var(--dc-gold)',
                        color: 'var(--dc-gold-text)',
                        border: 'none',
                        borderRadius: 'var(--radius)',
                        fontWeight: 700, fontSize: '.88rem',
                        cursor: placing ? 'wait' : 'pointer',
                        opacity: placing ? .7 : 1,
                    }}>
                    {placing ? <><Loader2 size={14} className="spin" /> Đang xử lý...</> : <><Zap size={14} /> Đặt hàng</>}
                </button>
            </div>
        </div>
    )
}


// ─── Main Page ────────────────────────────────────────────────────────────────
export default function OrderProxyPage() {
    const [selected, setSelected] = useState<any>(null)
    const [filterType, setFilterType] = useState('')
    const router = useRouter()

    const { data: prodData, isLoading: prodLoading } = useQuery({
        queryKey: ['proxy-products', filterType],
        queryFn: () => proxyAPI.listProducts(filterType, '', ''),
        staleTime: 60_000,
    })
    const products: any[] = prodData?.data?.data ?? []

    const typeFilters = [
        { value: '', label: 'Tất cả' },
        { value: 'residential', label: 'Residential' },
        { value: 'datacenter', label: 'Datacenter' },
        { value: 'mobile', label: 'Mobile' },
    ]

    return (
        <AppLayout breadcrumb={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Proxy', href: '/dashboard/proxy' }, { label: 'Order Proxy' }]}>
            {/* Header */}
            <div className="page-header">
                <div>
                    <h1 className="page-title">Order Proxy</h1>
                    <p className="page-subtitle">Chọn sản phẩm proxy và đặt hàng</p>
                </div>
            </div>

            {/* Filter chips */}
            <div style={{ display: 'flex', gap: '.4rem', marginBottom: '1.25rem', flexWrap: 'wrap' }}>
                {typeFilters.map(f => (
                    <button key={f.value} onClick={() => { setFilterType(f.value); setSelected(null) }}
                        style={{
                            padding: '.38rem 1rem',
                            borderRadius: 'var(--radius-pill)',
                            fontSize: '.82rem', fontWeight: 600,
                            border: filterType === f.value ? '1px solid rgba(230,168,23,.4)' : '1px solid var(--border)',
                            background: filterType === f.value ? 'rgba(230,168,23,.12)' : 'var(--surface)',
                            color: filterType === f.value ? 'var(--dc-gold)' : 'var(--text)',
                            cursor: 'pointer', transition: 'all .15s',
                        }}>
                        {f.label}
                    </button>
                ))}
            </div>

            {/* Products grid */}
            {prodLoading ? (
                <div className="loading-spinner">Đang tải sản phẩm...</div>
            ) : products.length === 0 ? (
                <div className="card">
                    <div className="empty-state">
                        <Globe size={40} opacity={0.25} />
                        <p>Chưa có sản phẩm proxy nào</p>
                        <span style={{ fontSize: '.82rem', color: 'var(--text-muted)' }}>Liên hệ admin để thêm sản phẩm</span>
                    </div>
                </div>
            ) : (
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1.25rem', padding: '2px' }}>
                    {products.map((p: any) => (
                        <ProductCard
                            key={p.id}
                            product={p}
                            selected={selected?.id === p.id}
                            onClick={() => setSelected(selected?.id === p.id ? null : p)}
                        />
                    ))}
                </div>
            )}

            {/* Order panel */}
            {selected && (
                <OrderPanel
                    product={selected}
                    onClose={() => setSelected(null)}
                    onSuccess={() => { setSelected(null); router.push('/dashboard/proxy') }}
                />
            )}
        </AppLayout>
    )
}
