'use client'
import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { proxyAPI, adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import {
    ShoppingCart, Eye, EyeOff, Copy, XCircle, RefreshCw,
    Globe, Clock, Database, ChevronRight, CheckCircle2,
    Package, Zap, ArrowLeft, Loader2, History
} from 'lucide-react'
import { formatVND } from '@/lib/format'

// ─── Status config ────────────────────────────────────────────────────────────
const STATUS: Record<string, { label: string; cls: string }> = {
    active:     { label: 'Active',      cls: 'badge-active' },
    pending:    { label: 'Pending',     cls: 'badge-pending' },
    processing: { label: 'Processing', cls: 'badge-processing' },
    cancelled:  { label: 'Cancelled',  cls: 'badge-error' },
    failed:     { label: 'Failed',      cls: 'badge-error' },
    expired:    { label: 'Expired',     cls: 'badge-terminated' },
    refunded:   { label: 'Refunded',    cls: 'badge-terminated' },
    suspended:  { label: '🔒 Suspended', cls: 'badge-error' },
}

// ─── Product Card ─────────────────────────────────────────────────────────────
const PROXY_TYPE_META: Record<string, { color: string; bg: string; label: string }> = {
    residential: { color: '#4ade80', bg: 'rgba(74,222,128,.12)',  label: 'Residential' },
    datacenter:  { color: '#60a5fa', bg: 'rgba(96,165,250,.12)', label: 'Datacenter'  },
    mobile:      { color: '#f472b6', bg: 'rgba(244,114,182,.12)', label: 'Mobile'     },
    isp:         { color: '#a78bfa', bg: 'rgba(167,139,250,.12)', label: 'ISP'        },
}

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
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem', fontSize: '.75rem', fontWeight: 500, color: 'var(--text-muted)', background: 'rgba(255,255,255,.06)', border: '1px solid var(--border-light)', padding: '.2rem .55rem', borderRadius: 'var(--radius-pill)' }}>
                        <Globe size={10} /> {product.location}
                    </span>
                )}
                {product.duration_days && (
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem', fontSize: '.75rem', fontWeight: 500, color: 'var(--text-muted)', background: 'rgba(255,255,255,.06)', border: '1px solid var(--border-light)', padding: '.2rem .55rem', borderRadius: 'var(--radius-pill)' }}>
                        <Clock size={10} /> {product.duration_days} ngày
                    </span>
                )}
                {product.bandwidth_gb && (
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem', fontSize: '.75rem', fontWeight: 500, color: 'var(--text-muted)', background: 'rgba(255,255,255,.06)', border: '1px solid var(--border-light)', padding: '.2rem .55rem', borderRadius: 'var(--radius-pill)' }}>
                        <Database size={10} /> {product.bandwidth_gb} GB
                    </span>
                )}
                {isRotating && (
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem', fontSize: '.75rem', fontWeight: 600, color: meta.color, background: meta.bg, border: '1px solid transparent', padding: '.2rem .55rem', borderRadius: 'var(--radius-pill)' }}>
                        <Zap size={10} /> Rotating
                    </span>
                )}
                {!product.duration_days && !product.bandwidth_gb && !isRotating && (
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem', fontSize: '.75rem', fontWeight: 500, color: 'var(--text-muted)', background: 'rgba(255,255,255,.04)', border: '1px solid var(--border-light)', padding: '.2rem .55rem', borderRadius: 'var(--radius-pill)' }}>
                        <Zap size={10} /> Pay per GB
                    </span>
                )}
            </div>

            {/* Divider */}
            <div style={{ height: 1, background: 'var(--border-light)' }} />

            {/* Pricing footer */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div>
                    <div style={{ fontSize: '.68rem', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.06em', marginBottom: '.15rem' }}>
                        {isRotating ? 'Giá / GB' : 'Giá / tháng'}
                    </div>
                    <div style={{ fontWeight: 800, fontSize: '1.3rem', color: selected ? 'var(--dc-gold)' : 'var(--text-heading)', lineHeight: 1, letterSpacing: '-.02em' }}>
                        {formatVND(product.base_cost)}
                        <span style={{ fontSize: '.72rem', fontWeight: 400, color: 'var(--text-muted)', marginLeft: '.2rem' }}>
                            {isRotating ? '/GB' : '/tháng'}
                        </span>
                    </div>
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

    const handleOrder = async () => {
        if (isStatic && countries.length > 0 && !country) {
            toastError('Please select a country')
            return
        }
        const payload = {
            product_id: product.id,
            qty,
            metadata: {
                country: country || undefined,
                isp_id: ispId || undefined,
                period_months: isStatic ? periodMonths : undefined,
            }
        }
        console.log('[OrderPanel] handleOrder payload:', payload)
        setPlacing(true)
        try {
            const res = await proxyAPI.createOrder(product.id, qty, {
                country: country || undefined,
                isp_id: ispId || undefined,
                period_months: isStatic ? periodMonths : undefined,
                protocol,
            })
            console.log('[OrderPanel] createOrder success:', res?.data)
            success('Order placed! Proxy is being activated...')
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
            onSuccess()
        } catch (err: any) {
            console.error('[OrderPanel] createOrder error full:', err)
            console.error('[OrderPanel] response status:', err?.response?.status)
            console.error('[OrderPanel] response data:', err?.response?.data)
            const msg = err?.response?.data?.error?.message ?? err?.message ?? 'Order failed'
            toastError(msg)
        } finally {
            setPlacing(false)
        }
    }

    return (
        <div className="fade-in" style={{
            background: 'var(--surface-raised)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-xl)',
            marginTop: '1rem',
            overflow: 'hidden',
            boxShadow: 'var(--shadow)',
        }}>
            {/* Header */}
            <div style={{
                background: 'rgba(230,168,23,.08)',
                borderBottom: '1px solid rgba(230,168,23,.15)',
                padding: '.875rem 1.5rem',
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem' }}>
                    <ShoppingCart size={16} color="var(--dc-gold)" />
                    <span style={{ fontWeight: 700, fontSize: '.9rem', color: 'var(--text-heading)' }}>
                        Configure Order — <span style={{ color: 'var(--dc-gold)' }}>{product.name}</span>
                    </span>
                </div>
                <button onClick={onClose} style={{
                    background: 'rgba(255,255,255,.06)', border: '1px solid var(--border)',
                    borderRadius: 6, color: 'var(--text-muted)', cursor: 'pointer',
                    padding: '.3rem .65rem', display: 'flex', alignItems: 'center', gap: '.3rem',
                    fontSize: '.78rem', fontWeight: 600, transition: 'color .12s',
                }}
                    onMouseEnter={e => (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-heading)'}
                    onMouseLeave={e => (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)'}
                >
                    <ArrowLeft size={12} /> Deselect
                </button>
            </div>

            <div style={{ padding: '1.25rem 1.5rem' }}>
                {/* Product meta badges */}
                <div style={{ display: 'flex', gap: '.4rem', flexWrap: 'wrap', marginBottom: '1.25rem' }}>
                    <span className="badge badge-secondary">{product.proxy_type?.charAt(0).toUpperCase() + product.proxy_type?.slice(1)}</span>
                    <span className="badge badge-secondary">{product.protocol?.toUpperCase()}</span>
                    {serviceId && <span className="badge badge-secondary">{serviceId}</span>}
                    {planId && <span className="badge badge-secondary">Plan: {planId}</span>}
                </div>

                {/* Options grid */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '1rem', marginBottom: '1.25rem' }}>
                    {/* Country */}
                    {isStatic && (
                        <div className="form-group" style={{ marginBottom: 0 }}>
                            <label>Country {countries.length > 0 && <span style={{ color: 'var(--error)' }}>*</span>}</label>
                            {optLoading ? (
                                <div style={{ display: 'flex', alignItems: 'center', gap: '.4rem', color: 'var(--text-muted)', fontSize: '.85rem' }}>
                                    <Loader2 size={14} className="spin" /> Loading...
                                </div>
                            ) : countries.length > 0 ? (
                                <select className="input" value={country} onChange={e => { setCountry(e.target.value); setIspId('') }}>
                                    <option value="">— Select country —</option>
                                    {countries.map((c: string) => <option key={c} value={c}>{c}</option>)}
                                </select>
                            ) : (
                                <input className="input" placeholder="e.g. US" value={country}
                                    onChange={e => setCountry(e.target.value)} />
                            )}
                        </div>
                    )}

                    {/* ISP */}
                    {isStatic && (
                        <div className="form-group" style={{ marginBottom: 0 }}>
                            <label>ISP <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}>(optional)</span></label>
                            {isps.length > 0 ? (
                                <select className="input" value={ispId} onChange={e => setIspId(e.target.value)}>
                                    <option value="">Any ISP</option>
                                    {isps.map((isp: any) => <option key={isp.id} value={isp.id}>{isp.name}</option>)}
                                </select>
                            ) : (
                                <select className="input" disabled>
                                    <option>{country ? 'No ISPs available' : 'Select country first'}</option>
                                </select>
                            )}
                        </div>
                    )}

                    {/* Period */}
                    {isStatic && (
                        <div className="form-group" style={{ marginBottom: 0 }}>
                            <label>Period (months)</label>
                            <select className="input" value={periodMonths} onChange={e => setPeriodMonths(Number(e.target.value))}>
                                {[1, 2, 3, 6, 12].map(n => <option key={n} value={n}>{n} month{n > 1 ? 's' : ''}</option>)}
                            </select>
                        </div>
                    )}

                    {/* Quantity */}
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Quantity {isRotating ? '(GB)' : '(proxies)'}</label>
                        <div style={{ display: 'flex', gap: '.4rem' }}>
                            <button onClick={() => setQty(q => Math.max(1, q - 1))}
                                style={{ width: 34, height: 38, border: '1px solid var(--border)', borderRadius: 'var(--radius)', background: 'rgba(255,255,255,.04)', color: 'var(--text-heading)', cursor: 'pointer', fontWeight: 700 }}>−</button>
                            <input className="input" type="number" min={1} value={qty}
                                onChange={e => setQty(Math.max(1, parseInt(e.target.value) || 1))}
                                style={{ textAlign: 'center' }} />
                            <button onClick={() => setQty(q => q + 1)}
                                style={{ width: 34, height: 38, border: '1px solid var(--border)', borderRadius: 'var(--radius)', background: 'rgba(255,255,255,.04)', color: 'var(--text-heading)', cursor: 'pointer', fontWeight: 700 }}>+</button>
                        </div>
                    </div>
                </div>

                {/* Protocol selector */}
                <div style={{ marginBottom: '1.25rem' }}>
                    <label style={{ display: 'block', fontSize: '.82rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '.5rem', textTransform: 'uppercase', letterSpacing: '.04em' }}>Protocol</label>
                    <div style={{ display: 'flex', gap: '.5rem', flexWrap: 'wrap' }}>
                        {([
                            { value: 'default', label: '⚡ Default', desc: 'HTTP + SOCKS5' },
                            { value: 'http',    label: '🌐 HTTP',    desc: 'HTTP only' },
                            { value: 'socks5',  label: '🔌 SOCKS5',  desc: 'SOCKS5 only' },
                        ] as const).map(opt => (
                            <button key={opt.value} onClick={() => setProtocol(opt.value)} style={{
                                padding: '.45rem .9rem',
                                border: `1px solid ${protocol === opt.value ? 'var(--dc-gold)' : 'var(--border)'}`,
                                borderRadius: 'var(--radius)',
                                background: protocol === opt.value ? 'rgba(230,168,23,.12)' : 'rgba(255,255,255,.03)',
                                color: protocol === opt.value ? 'var(--dc-gold)' : 'var(--text-muted)',
                                cursor: 'pointer',
                                fontSize: '.82rem',
                                fontWeight: protocol === opt.value ? 700 : 500,
                                transition: 'all .12s',
                                display: 'flex',
                                flexDirection: 'column',
                                alignItems: 'flex-start',
                                gap: '.1rem',
                            }}>
                                <span>{opt.label}</span>
                                <span style={{ fontSize: '.72rem', opacity: .7 }}>{opt.desc}</span>
                            </button>
                        ))}
                    </div>
                </div>

                {/* Total + CTA */}
                <div style={{
                    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                    paddingTop: '1rem', borderTop: '1px solid var(--border-light)',
                    gap: '1rem', flexWrap: 'wrap',
                }}>
                    <div>
                        <div style={{ fontSize: '.8rem', color: 'var(--text-muted)' }}>
                            {qty} proxy{qty > 1 ? 'ies' : ''} × {formatVND(product.base_cost)}{unitLabel}
                            {isStatic && periodMonths > 1 && ` × ${periodMonths} tháng`}
                        </div>
                        <div style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--dc-gold)', lineHeight: 1.2 }}>
                            {total}
                        </div>
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '.3rem' }}>
                        <button onClick={handleOrder} disabled={placing} style={{
                            display: 'flex', alignItems: 'center', gap: '.5rem',
                            padding: '.7rem 1.75rem', fontSize: '.95rem', fontWeight: 700,
                            background: placing ? 'rgba(230,168,23,.5)' : 'var(--dc-gold)',
                            color: 'var(--dc-gold-text)', border: 'none', borderRadius: 'var(--radius)',
                            cursor: placing ? 'not-allowed' : 'pointer',
                            transition: 'background .15s',
                        }}>
                            {placing ? <><Loader2 size={15} className="spin" /> Placing...</> : <><ShoppingCart size={15} /> Place Order</>}
                        </button>
                        <span style={{ fontSize: '.72rem', color: 'var(--text-muted)' }}>Proxy activation may take 1–3 min</span>
                    </div>
                </div>
            </div>
        </div>
    )
}

// ─── Edit Order Modal ──────────────────────────────────────────────────────────
function EditOrderModal({ order, onClose, onSaved }: { order: any; onClose: () => void; onSaved: () => void }) {
    const { success, error: toastError } = useToast()
    const [price, setPrice] = useState(order.custom_price ?? '')  // do NOT fall back to unit_price
    const [expiry, setExpiry] = useState(
        order.custom_expires_at
            ? new Date(order.custom_expires_at).toISOString().slice(0, 16)
            : order.expires_at
                ? new Date(order.expires_at).toISOString().slice(0, 16)
                : ''
    )
    const [note, setNote] = useState(order.admin_note ?? '')
    const [saving, setSaving] = useState(false)

    const handleSave = async () => {
        setSaving(true)
        try {
            await proxyAPI.patchOrder(order.id, {
                custom_price: price ? String(price) : undefined,
                custom_expires_at: expiry ? new Date(expiry).toISOString() : undefined,
                admin_note: note || undefined,
            })
            success('Đã cập nhật đơn hàng')
            onSaved()
            onClose()
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Cập nhật thất bại')
        } finally {
            setSaving(false)
        }
    }

    return (
        <div style={{
            position: 'fixed', inset: 0, zIndex: 1000,
            background: 'rgba(0,0,0,.65)', display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={onClose}>
            <div style={{
                background: 'var(--surface-raised)', border: '1px solid var(--border)',
                borderRadius: 'var(--radius-xl)', padding: '1.5rem', width: 400, maxWidth: '95vw',
            }} onClick={e => e.stopPropagation()}>
                <div style={{ fontWeight: 700, fontSize: '1rem', color: 'var(--text-heading)', marginBottom: '1.25rem' }}>
                    ✏️ Sửa đơn hàng
                    <div style={{ fontSize: '.75rem', color: 'var(--text-muted)', fontWeight: 400, marginTop: 2 }}>{order.order_number}</div>
                </div>

                <div className="form-group">
                    <label>Giá tuỳ chỉnh (VND) <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}>(áp dụng khi gia hạn)</span></label>
                    <input className="input" type="number" min={0} step={1000}
                        value={price} onChange={e => setPrice(e.target.value)}
                        placeholder={`Mặc định: ${Number(order.unit_price).toLocaleString('vi-VN')}đ`} />
                </div>

                <div className="form-group">
                    <label>Ngày hết hạn tuỳ chỉnh</label>
                    <input className="input" type="datetime-local"
                        value={expiry} onChange={e => setExpiry(e.target.value)} />
                </div>

                <div className="form-group" style={{ marginBottom: '1.25rem' }}>
                    <label>Ghi chú</label>
                    <input className="input" value={note} onChange={e => setNote(e.target.value)} placeholder="Ghi chú tuỳ chọn..." />
                </div>

                <div style={{ display: 'flex', gap: '.6rem', justifyContent: 'flex-end' }}>
                    <button onClick={onClose} style={{
                        padding: '.5rem 1.1rem', border: '1px solid var(--border)', borderRadius: 'var(--radius)',
                        background: 'transparent', color: 'var(--text-muted)', cursor: 'pointer', fontWeight: 600,
                    }}>Huỷ</button>
                    <button onClick={handleSave} disabled={saving} style={{
                        padding: '.5rem 1.25rem', border: 'none', borderRadius: 'var(--radius)',
                        background: 'var(--dc-gold)', color: 'var(--dc-gold-text)', cursor: saving ? 'not-allowed' : 'pointer', fontWeight: 700,
                    }}>
                        {saving ? 'Đang lưu...' : 'Lưu'}
                    </button>
                </div>
            </div>
        </div>
    )
}

// ─── Time remaining helper ─────────────────────────────────────────────────────
function timeRemaining(dateStr: string): { text: string; urgent: boolean } {
    const diff = new Date(dateStr).getTime() - Date.now()
    if (diff <= 0) return { text: 'Expired', urgent: true }
    const days = Math.floor(diff / 86400000)
    const hrs  = Math.floor((diff % 86400000) / 3600000)
    if (days > 0) return { text: `${days}d ${hrs}h`, urgent: days <= 3 }
    return { text: `${hrs}h`, urgent: true }
}

// ─── Order Event Modal ─────────────────────────────────────────────────────────
const EVT_CONFIG: Record<string, { icon: string; color: string; label: string }> = {
    'order.created':   { icon: '📦', color: 'var(--dc-gold)',  label: 'Tạo đơn' },
    'order.activated': { icon: '✅', color: 'var(--success)',  label: 'Kích hoạt' },
    'order.cancelled': { icon: '🚫', color: 'var(--error)',    label: 'Huỷ đơn' },
    'order.patched':   { icon: '✏️', color: '#a78bfa',         label: 'Chỉnh sửa' },
    'order.failed':    { icon: '❌', color: 'var(--error)',    label: 'Thất bại' },
}

function OrderEventModal({ order, onClose }: { order: any; onClose: () => void }) {
    const { data, isLoading } = useQuery({
        queryKey: ['order-events', order.id],
        queryFn: () => proxyAPI.getOrderEvents(order.id),
        staleTime: 10_000,
    })
    const events: any[] = data?.data?.data ?? data?.data ?? []

    const fmt = (ts: string) => {
        const d = new Date(ts)
        return `${String(d.getDate()).padStart(2,'0')}/${String(d.getMonth()+1).padStart(2,'0')}/${d.getFullYear()} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
    }

    return (
        <div style={{
            position: 'fixed', inset: 0, zIndex: 1100,
            background: 'rgba(0,0,0,.7)', display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={onClose}>
            <div style={{
                background: 'var(--surface-raised)', border: '1px solid var(--border)',
                borderRadius: 'var(--radius-xl)', padding: '1.5rem', width: 480, maxWidth: '95vw', maxHeight: '80vh',
                display: 'flex', flexDirection: 'column',
            }} onClick={e => e.stopPropagation()}>
                {/* Header */}
                <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem', marginBottom: '1.25rem' }}>
                    <History size={18} color="var(--dc-gold)" />
                    <div>
                        <div style={{ fontWeight: 700, fontSize: '.95rem', color: 'var(--text-heading)' }}>Lịch sử đơn hàng</div>
                        <div style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>{order.order_number}</div>
                    </div>
                    <button onClick={onClose} style={{ marginLeft: 'auto', background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: '1.1rem' }}>✕</button>
                </div>

                {/* Timeline */}
                <div style={{ overflowY: 'auto', flex: 1 }}>
                    {isLoading ? (
                        <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)' }}>Đang tải...</div>
                    ) : events.length === 0 ? (
                        <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)' }}>Chưa có lịch sử</div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
                            {events.map((ev: any, i: number) => {
                                const cfg = EVT_CONFIG[ev.event_type] ?? { icon: '🔹', color: 'var(--text-muted)', label: ev.event_type }
                                const payload = ev.payload && typeof ev.payload === 'object' ? ev.payload : {}
                                const isLast = i === events.length - 1
                                return (
                                    <div key={ev.id} style={{ display: 'flex', gap: '.75rem', paddingBottom: isLast ? 0 : '.75rem' }}>
                                        {/* Dot + line */}
                                        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: 28, flexShrink: 0 }}>
                                            <div style={{
                                                width: 28, height: 28, borderRadius: '50%', display: 'grid', placeItems: 'center',
                                                background: cfg.color + '22', border: `2px solid ${cfg.color}`, fontSize: '.85rem', flexShrink: 0,
                                            }}>{cfg.icon}</div>
                                            {!isLast && <div style={{ width: 2, flex: 1, background: 'var(--border-light)', marginTop: 4 }} />}
                                        </div>
                                        {/* Content */}
                                        <div style={{ paddingBottom: '.75rem', flex: 1 }}>
                                            <div style={{ display: 'flex', alignItems: 'baseline', gap: '.5rem' }}>
                                                <span style={{ fontWeight: 700, color: cfg.color, fontSize: '.88rem' }}>{cfg.label}</span>
                                                <span style={{ fontSize: '.72rem', color: 'var(--text-muted)' }}>{fmt(ev.created_at)}</span>
                                            </div>
                                            {Object.entries(payload).filter(([,v]) => v).map(([k, v]) => (
                                                <div key={k} style={{ fontSize: '.75rem', color: 'var(--text-muted)', marginTop: 2 }}>
                                                    <span style={{ textTransform: 'capitalize' }}>{k.replace(/_/g,' ')}</span>: <span style={{ color: 'var(--text-heading)', fontFamily: 'monospace' }}>{String(v)}</span>
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

// ─── Orders Table ───────────────────────────────────────────────────────────────────
const LIMIT_OPTIONS = [10, 20, 50, 100, 9999]
function OrdersTable({ orders, onCancel, onRenew, onLock, onUnlock, onRefresh, limit, onLimitChange }: {
    orders: any[]; onCancel: (id: string, num: string) => void
    onRenew: (id: string, num: string) => void
    onLock?: (id: string, num: string) => void
    onUnlock?: (id: string, num: string) => void
    onRefresh: () => void
    limit: number; onLimitChange: (n: number) => void
}) {
    const [revealed, setRevealed] = useState<Record<string, any>>({})
    const [editOrder, setEditOrder] = useState<any>(null)
    const [historyOrder, setHistoryOrder] = useState<any>(null)
    const [renewingId, setRenewingId] = useState<string | null>(null)
    const [lockingId, setLockingId] = useState<string | null>(null)
    const { success, error: toastError } = useToast()

    // Auto-load credentials for all active orders
    const activeOrders = orders.filter(o => o.status === 'active')
    useEffect(() => {
        activeOrders.forEach(o => {
            if (!revealed[o.id]) {
                proxyAPI.getCredentials(o.id)
                    .then(res => {
                        const data = res.data?.data
                        if (data) setRevealed(prev => ({ ...prev, [o.id]: data }))
                    })
                    .catch(() => {}) // silent — creds may not exist yet
            }
        })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [orders.length])

    const handleCopy = async (text: string) => {
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(text)
            } else {
                // Fallback for HTTP / non-secure contexts
                const el = document.createElement('textarea')
                el.value = text
                el.style.position = 'fixed'
                el.style.opacity = '0'
                document.body.appendChild(el)
                el.select()
                document.execCommand('copy')
                document.body.removeChild(el)
            }
            success('Đã sao chép!')
        } catch {
            toastError('Không thể sao chép')
        }
    }

    const Chip = ({ value, title }: { value: string; title?: string }) => (
        <span
            title={title ?? `Click to copy: ${value}`}
            onClick={() => handleCopy(value)}
            style={{
                display: 'inline-block', cursor: 'pointer',
                fontSize: '.76rem', background: 'rgba(255,255,255,.06)',
                padding: '.1rem .4rem', borderRadius: 3,
                color: 'var(--text-heading)', fontFamily: 'monospace',
                border: '1px solid transparent', transition: 'border-color .12s',
                whiteSpace: 'nowrap',
            }}
            onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'rgba(230,168,23,.4)'}
            onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'transparent'}
        >
            {value}
        </span>
    )

    if (orders.length === 0) return null

    // effective expiry: custom_expires_at overrides expires_at
    const effectiveExpiry = (o: any) => o.custom_expires_at || o.expires_at

    return (
        <>
            {editOrder && (
                <EditOrderModal
                    order={editOrder}
                    onClose={() => setEditOrder(null)}
                    onSaved={onRefresh}
                />
            )}
            {historyOrder && (
                <OrderEventModal
                    order={historyOrder}
                    onClose={() => setHistoryOrder(null)}
                />
            )}
            <div className="card" style={{ padding: 0 }}>
                <div style={{ padding: '1rem 1.5rem', borderBottom: '1px solid var(--border-light)', display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                    {/* Show [N▼] selector — left side */}
                    <span style={{ fontSize: '.85rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>Hiện</span>
                    <select
                        value={limit}
                        onChange={e => onLimitChange(Number(e.target.value))}
                        style={{
                            background: 'var(--surface-raised)', border: '1px solid var(--border)',
                            borderRadius: 'var(--radius)', color: 'var(--text-heading)',
                            padding: '.25rem .6rem', fontSize: '.85rem', cursor: 'pointer', fontWeight: 600,
                        }}
                    >
                        {LIMIT_OPTIONS.map(o => (
                            <option key={o} value={o}>{o === 9999 ? 'Tất cả' : o}</option>
                        ))}
                    </select>
                    <Package size={15} color="var(--dc-gold)" style={{ marginLeft: 'auto' }} />
                    <span style={{ fontWeight: 600, color: 'var(--text-heading)' }}>Proxy của tôi</span>
                    <span className="badge badge-secondary">{orders.length}</span>
                </div>
                <div className="table-wrapper" style={{ overflowX: 'auto' }}>
                    <table className="data-table" style={{ minWidth: 900 }}>
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Status</th>
                                <th>Amount</th>
                                <th>Host:Port</th>
                                <th>User</th>
                                <th>Password</th>
                                <th>Expires At</th>
                                <th>Time Left</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {orders.map((o: any) => {
                                const st = STATUS[o.status] ?? { label: o.status, cls: 'badge-secondary' }
                                const exp = effectiveExpiry(o)
                                const tr = exp ? timeRemaining(exp) : null
                                const creds = revealed[o.id]
                                const isActive = o.status === 'active'
                                const isExpired = o.status === 'expired'
                                const isSuspended = o.status === 'suspended'
                                const isFailed = o.status === 'failed'
                                const effectivePrice = o.custom_price ?? o.unit_price
                                return (
                                    <tr key={o.id} style={{ opacity: isFailed ? .6 : 1 }}>
                                        <td>
                                            <code style={{ fontSize: '.78rem', background: 'rgba(255,255,255,.06)', padding: '.15rem .45rem', borderRadius: 4, color: 'var(--text-heading)' }}>{o.order_number}</code>
                                            {o.admin_note && <div style={{ fontSize: '.72rem', color: 'var(--warning)', marginTop: 2 }}>📝 {o.admin_note}</div>}
                                        </td>
                                        <td><span className={`badge ${st.cls}`}>{st.label}</span></td>
                                        <td>
                                            <strong>{Number(effectivePrice).toLocaleString('vi-VN')}đ</strong>
                                            {o.custom_price && <div style={{ fontSize: '.7rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>{Number(o.unit_price).toLocaleString('vi-VN')}đ</div>}
                                        </td>

                                        {/* Host:Port */}
                                        <td>
                                            {isActive && creds ? (
                                                <div style={{ display: 'flex', gap: '.3rem', alignItems: 'center' }}>
                                                    <Chip value={`${creds.host}:${creds.port}`} title="Click to copy host:port" />
                                                </div>
                                            ) : isActive ? (
                                                <span style={{ color: 'var(--text-muted)', fontSize: '.75rem' }}>Loading...</span>
                                            ) : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                                        </td>

                                        {/* Username */}
                                        <td>
                                            {isActive && creds
                                                ? <Chip value={creds.username} title="Click to copy username" />
                                                : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                                        </td>

                                        {/* Password */}
                                        <td>
                                            {isActive && creds ? (
                                                <Chip value={creds.password} title="Click to copy password (hidden)" />
                                            ) : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                                        </td>

                                        {/* Expires At */}
                                        <td style={{ color: 'var(--text-muted)', fontSize: '.82rem', whiteSpace: 'nowrap' }}>
                                            {exp
                                                ? <>{new Date(exp).toLocaleDateString('vi-VN')}<br /><span style={{ fontSize: '.72rem' }}>{new Date(exp).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' })}</span></>
                                                : '—'}
                                            {o.custom_expires_at && o.expires_at && o.custom_expires_at !== o.expires_at && (
                                                <div style={{ fontSize: '.68rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>{new Date(o.expires_at).toLocaleDateString('vi-VN')}</div>
                                            )}
                                        </td>

                                        {/* Time Left */}
                                        <td style={{ whiteSpace: 'nowrap' }}>
                                            {tr ? (
                                                <span style={{ fontSize: '.8rem', fontWeight: 600, color: tr.urgent ? 'var(--error)' : 'var(--success)' }}>
                                                    {tr.urgent && '⚠️ '}{tr.text}
                                                </span>
                                            ) : '—'}
                                        </td>

                                        {/* Actions */}
                                        <td>
                                            <div style={{ display: 'flex', gap: '.3rem', flexWrap: 'wrap' }}>
                                                <button className="action-btn" style={{ background: 'rgba(230,168,23,.12)', color: 'var(--dc-gold)', border: '1px solid rgba(230,168,23,.3)' }}
                                                    onClick={() => setHistoryOrder(o)} title="Lịch sử">
                                                    <History size={11} />
                                                </button>
                                                {!isFailed && (
                                                    <button className="action-btn" style={{ background: 'rgba(139,92,246,.15)', color: '#a78bfa', border: '1px solid rgba(139,92,246,.3)' }}
                                                        onClick={() => setEditOrder(o)} title="Sửa">
                                                        ✏️
                                                    </button>
                                                )}
                                                {isActive && creds && (
                                                    <button className="action-btn blue" style={{ fontSize: '.72rem' }}
                                                        onClick={() => handleCopy(`${creds.host}:${creds.port}@${creds.username}:${creds.password}`)} title="Copy all (host:port@user:pass)">
                                                        <Copy size={10} /> All
                                                    </button>
                                                )}
                                                {isExpired && (
                                                    <button
                                                        className="action-btn"
                                                        style={{ background: 'rgba(34,197,94,.15)', color: '#4ade80', border: '1px solid rgba(34,197,94,.3)', opacity: renewingId === o.id ? .6 : 1 }}
                                                        disabled={renewingId === o.id}
                                                        onClick={() => onRenew(o.id, o.order_number)}
                                                        title="Gia hạn"
                                                    >
                                                        {renewingId === o.id ? '...' : '🔄 Gia hạn'}
                                                    </button>
                                                )}
                                                {/* Admin lock/unlock */}
                                                {onLock && (isActive || isExpired) && (
                                                    <button
                                                        className="action-btn"
                                                        style={{ background: 'rgba(239,68,68,.12)', color: '#f87171', border: '1px solid rgba(239,68,68,.3)', fontSize: '.72rem', opacity: lockingId === o.id ? .5 : 1 }}
                                                        disabled={lockingId === o.id}
                                                        onClick={async () => { setLockingId(o.id); try { await onLock(o.id, o.order_number) } finally { setLockingId(null) } }}
                                                        title="Khóa proxy (admin)"
                                                    >
                                                        {lockingId === o.id ? '...' : '🔒 Lock'}
                                                    </button>
                                                )}
                                                {onUnlock && isSuspended && (
                                                    <button
                                                        className="action-btn"
                                                        style={{ background: 'rgba(34,197,94,.12)', color: '#4ade80', border: '1px solid rgba(34,197,94,.3)', fontSize: '.72rem', opacity: lockingId === o.id ? .5 : 1 }}
                                                        disabled={lockingId === o.id}
                                                        onClick={async () => { setLockingId(o.id); try { await onUnlock(o.id, o.order_number) } finally { setLockingId(null) } }}
                                                        title="Mở khóa proxy (admin)"
                                                    >
                                                        {lockingId === o.id ? '...' : '🔓 Unlock'}
                                                    </button>
                                                )}
                                                {(o.status === 'active' || o.status === 'pending' || o.status === 'expired') && (
                                                    <button className="action-btn red" onClick={() => onCancel(o.id, o.order_number)}>
                                                        <XCircle size={12} /> Cancel
                                                    </button>
                                                )}
                                            </div>
                                        </td>
                                    </tr>
                                )
                            })}
                        </tbody>
                    </table>
                </div>
            </div>
        </>
    )
}

// ─── Tab Button ───────────────────────────────────────────────────────────────
function TabBtn({ label, active, onClick, count }: { label: string; active: boolean; onClick: () => void; count?: number }) {
    return (
        <button onClick={onClick} style={{
            display: 'inline-flex', alignItems: 'center', gap: '.4rem',
            padding: '.5rem 1.2rem',
            background: active ? 'var(--surface)' : 'transparent',
            color: active ? 'var(--text-heading)' : 'var(--text-muted)',
            border: 'none',
            borderBottom: active ? '2px solid var(--dc-gold)' : '2px solid transparent',
            fontWeight: active ? 700 : 500,
            fontSize: '.875rem',
            cursor: 'pointer',
            transition: 'all .15s',
            marginBottom: '-1px',
        }}>
            {label}
            {count !== undefined && (
                <span style={{
                    padding: '.1rem .5rem', borderRadius: '100px',
                    background: active ? 'rgba(230,168,23,.15)' : 'rgba(255,255,255,.08)',
                    color: active ? 'var(--dc-gold)' : 'var(--text-muted)',
                    fontSize: '.72rem', fontWeight: 700,
                }}>{count}</span>
            )}
        </button>
    )
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function ProxyOrdersPage() {
    const [page, setPage] = useState(1)
    const [limit, setLimit] = useState(10)
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()
    const qc = useQueryClient()

    const { data: ordersData, isLoading: ordersLoading, refetch } = useQuery({
        queryKey: ['proxy-orders', page, limit],
        queryFn: () => proxyAPI.listOrders(page, limit === 9999 ? 9999 : limit),
        refetchInterval: 30000,
        refetchIntervalInBackground: false,
    })
    const orders: any[] = ordersData?.data?.data ?? []
    const meta = ordersData?.data?.meta ?? {}

    const cancelMut = useMutation({
        mutationFn: (id: string) => proxyAPI.cancelOrder(id),
        onSuccess: () => { success('Đã hủy đơn'); qc.invalidateQueries({ queryKey: ['proxy-orders'] }) },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Hủy thất bại'),
    })
    const handleCancel = async (id: string, num: string) => {
        const ok = await confirm({ title: 'Hủy đơn hàng', message: `Hủy đơn ${num}? Thao tác này không thể hoàn tác.`, confirmLabel: 'Hủy đơn', variant: 'danger' })
        if (ok) cancelMut.mutate(id)
    }

    const handleRenew = async (id: string, num: string) => {
        const ok = await confirm({ title: 'Gia hạn proxy', message: `Gia hạn ${num}? Số dư ví sẽ bị trừ theo giá gốc.`, confirmLabel: 'Gia hạn', variant: 'primary' })
        if (!ok) return
        try {
            await proxyAPI.renewOrder(id)
            success(`Đã gia hạn ${num} thành công!`)
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Gia hạn thất bại')
        }
    }

    const handleLockOrder = async (id: string, num: string) => {
        const ok = await confirm({ title: '🔒 Khóa proxy', message: `Lock ${num}? Proxy sẽ bị tạm ngưng, user không thể sử dụng.`, confirmLabel: 'Lock', variant: 'danger' })
        if (!ok) return
        try {
            await adminAPI.orderAction(id, 'lock')
            success(`Đã lock ${num}`)
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Lock thất bại')
        }
    }

    const handleUnlockOrder = async (id: string, num: string) => {
        const ok = await confirm({ title: '🔓 Mở khóa proxy', message: `Unlock ${num}? Proxy sẽ được khôi phục.`, confirmLabel: 'Unlock', variant: 'primary' })
        if (!ok) return
        try {
            await adminAPI.orderAction(id, 'unlock')
            success(`Đã unlock ${num}`)
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Unlock thất bại')
        }
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Proxy' }]}>
            {confirmDialog}

            {/* Header */}
            <div className="page-header">
                <div>
                    <h1 className="page-title">My Proxies</h1>
                    <p className="page-subtitle">Quản lý proxy của bạn</p>
                </div>
                <div style={{ display: 'flex', gap: '.5rem' }}>
                    <a href="/dashboard/proxy/order" className="btn-primary" style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '.35rem' }}>
                        <ShoppingCart size={14} /> Order Proxy
                    </a>
                    <button className="topbar-icon-btn" onClick={() => refetch()} title="Refresh">
                        <RefreshCw size={15} />
                    </button>
                </div>
            </div>

            {/* Orders list */}
            <div className="fade-in">
                {ordersLoading ? (
                    <div className="loading-spinner">Đang tải đơn hàng...</div>
                ) : orders.length === 0 ? (
                    <div className="card">
                        <div className="empty-state">
                            <Package size={44} opacity={0.25} />
                            <p>Bạn chưa có đơn proxy nào</p>
                            <a href="/dashboard/proxy/order" className="btn-primary" style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '.35rem' }}>
                                <ShoppingCart size={14} /> Order Proxy ngay
                            </a>
                        </div>
                    </div>
                ) : (
                    <>
                        <OrdersTable orders={orders} onCancel={handleCancel} onRenew={handleRenew} onLock={handleLockOrder} onUnlock={handleUnlockOrder} onRefresh={() => qc.invalidateQueries({ queryKey: ['proxy-orders'] })} limit={limit} onLimitChange={l => { setLimit(l); setPage(1) }} />
                        <Pagination
                            page={page}
                            totalPages={meta.total_pages ?? 1}
                            total={meta.total ?? orders.length}
                            limit={limit === 9999 ? (meta.total ?? orders.length) : limit}
                            onPageChange={p => setPage(p)}
                        />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
