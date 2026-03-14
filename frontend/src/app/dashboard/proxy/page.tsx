'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { proxyAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import {
    ShoppingCart, Eye, EyeOff, Copy, XCircle, RefreshCw,
    Globe, Clock, Database, ChevronRight, CheckCircle2,
    Package, Zap, ArrowLeft, Loader2
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
    const [price, setPrice] = useState(order.custom_price ?? order.unit_price ?? '')
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

// ─── Orders Table ─────────────────────────────────────────────────────────────
function OrdersTable({ orders, onCancel, onRefresh }: { orders: any[]; onCancel: (id: string, num: string) => void; onRefresh: () => void }) {
    const [revealed, setRevealed] = useState<Record<string, any>>({})
    const [editOrder, setEditOrder] = useState<any>(null)
    const { success, error: toastError } = useToast()

    const revealMut = useMutation({
        mutationFn: (id: string) => proxyAPI.getCredentials(id),
        onSuccess: (res, id) => setRevealed(prev => ({ ...prev, [id]: res.data?.data })),
        onError: () => toastError('Failed to load credentials'),
    })

    const handleCopy = async (text: string) => {
        await navigator.clipboard.writeText(text)
        success('Đã sao chép!')
    }

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
            <div className="card" style={{ padding: 0 }}>
                <div style={{ padding: '1rem 1.5rem', borderBottom: '1px solid var(--border-light)', display: 'flex', alignItems: 'center', gap: '.5rem' }}>
                    <Package size={16} color="var(--dc-gold)" />
                    <span style={{ fontWeight: 600, color: 'var(--text-heading)' }}>My Orders</span>
                    <span className="badge badge-secondary" style={{ marginLeft: '.25rem' }}>{orders.length}</span>
                </div>
                <div className="table-wrapper" style={{ overflowX: 'auto' }}>
                    <table className="data-table" style={{ minWidth: 900 }}>
                        <thead>
                            <tr>
                                <th>Order</th>
                                <th>Status</th>
                                <th>Qty</th>
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
                                const isFailed = o.status === 'failed'
                                const effectivePrice = o.custom_price ?? o.unit_price
                                return (
                                    <tr key={o.id} style={{ opacity: isFailed ? .6 : 1 }}>
                                        <td>
                                            <code style={{ fontSize: '.78rem', background: 'rgba(255,255,255,.06)', padding: '.15rem .45rem', borderRadius: 4, color: 'var(--text-heading)' }}>{o.order_number}</code>
                                            {o.admin_note && <div style={{ fontSize: '.72rem', color: 'var(--warning)', marginTop: 2 }}>📝 {o.admin_note}</div>}
                                        </td>
                                        <td><span className={`badge ${st.cls}`}>{st.label}</span></td>
                                        <td style={{ fontWeight: 600 }}>{o.quantity}</td>
                                        <td>
                                            <strong>{Number(effectivePrice).toLocaleString('vi-VN')}đ</strong>
                                            {o.custom_price && <div style={{ fontSize: '.7rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>{Number(o.unit_price).toLocaleString('vi-VN')}đ</div>}
                                        </td>

                                        {/* Host:Port */}
                                        <td>
                                            {isActive && creds ? (
                                                <div style={{ display: 'flex', gap: '.3rem', alignItems: 'center' }}>
                                                    <code style={{ fontSize: '.76rem', background: 'rgba(255,255,255,.06)', padding: '.1rem .35rem', borderRadius: 3, color: 'var(--text-heading)' }}>
                                                        {creds.host}:{creds.port}
                                                    </code>
                                                    <button className="action-btn blue" onClick={() => handleCopy(`${creds.host}:${creds.port}`)} title="Copy"><Copy size={10} /></button>
                                                </div>
                                            ) : isActive ? (
                                                <button className="action-btn purple" onClick={() => revealMut.mutate(o.id)} disabled={revealMut.isPending} style={{ fontSize: '.72rem' }}>
                                                    <Eye size={11} /> Show
                                                </button>
                                            ) : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                                        </td>

                                        {/* Username */}
                                        <td>
                                            {isActive && creds ? (
                                                <div style={{ display: 'flex', gap: '.3rem', alignItems: 'center' }}>
                                                    <code style={{ fontSize: '.76rem', background: 'rgba(255,255,255,.06)', padding: '.1rem .35rem', borderRadius: 3, color: 'var(--text-heading)', maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{creds.username}</code>
                                                    <button className="action-btn blue" onClick={() => handleCopy(creds.username)} title="Copy"><Copy size={10} /></button>
                                                </div>
                                            ) : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                                        </td>

                                        {/* Password */}
                                        <td>
                                            {isActive && creds ? (
                                                <div style={{ display: 'flex', gap: '.3rem', alignItems: 'center' }}>
                                                    <code style={{ fontSize: '.76rem', background: 'rgba(255,255,255,.06)', padding: '.1rem .35rem', borderRadius: 3, color: 'var(--text-heading)', maxWidth: 90, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>••••••</code>
                                                    <button className="action-btn blue" onClick={() => handleCopy(creds.password)} title="Copy password"><Copy size={10} /></button>
                                                </div>
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
                                                {!isFailed && (
                                                    <button className="action-btn" style={{ background: 'rgba(139,92,246,.15)', color: '#a78bfa', border: '1px solid rgba(139,92,246,.3)' }}
                                                        onClick={() => setEditOrder(o)} title="Sửa">
                                                        ✏️
                                                    </button>
                                                )}
                                                {isActive && creds && (
                                                    <button className="action-btn gray" onClick={() => setRevealed(p => { const n = { ...p }; delete n[o.id]; return n })} title="Ẩn">
                                                        <EyeOff size={11} />
                                                    </button>
                                                )}
                                                {(o.status === 'active' || o.status === 'pending') && (
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
    const [tab, setTab] = useState<'buy' | 'orders'>('buy')
    const [page, setPage] = useState(1)
    const [selected, setSelected] = useState<any>(null)
    const [filterType, setFilterType] = useState('')
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()
    const qc = useQueryClient()

    const { data: prodData, isLoading: prodLoading } = useQuery({
        queryKey: ['proxy-products', filterType],
        queryFn: () => proxyAPI.listProducts(filterType, '', ''),
        staleTime: 60_000,
    })
    const products: any[] = prodData?.data?.data ?? []

    const { data: ordersData, isLoading: ordersLoading, refetch } = useQuery({
        queryKey: ['proxy-orders', page],
        queryFn: () => proxyAPI.listOrders(page),
        refetchInterval: 15000,
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

    const typeFilters = [
        { value: '', label: 'Tất cả' },
        { value: 'residential', label: 'Residential' },
        { value: 'datacenter', label: 'Datacenter' },
        { value: 'mobile', label: 'Mobile' },
    ]

    return (
        <AppLayout breadcrumb={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Proxy' }]}>
            {confirmDialog}

            {/* Header */}
            <div className="page-header">
                <div>
                    <h1 className="page-title">Proxy Services</h1>
                    <p className="page-subtitle">Mua và quản lý proxy của bạn</p>
                </div>
                <button className="topbar-icon-btn" onClick={() => refetch()} title="Refresh">
                    <RefreshCw size={15} />
                </button>
            </div>

            {/* ─── Tab Navigation ─── */}
            <div style={{
                display: 'flex', borderBottom: '1px solid var(--border)',
                marginBottom: '1.5rem', gap: '.25rem',
            }}>
                <TabBtn label="🛒 Mua Proxy" active={tab === 'buy'} onClick={() => setTab('buy')} />
                <TabBtn label="📦 Proxy của tôi" active={tab === 'orders'} onClick={() => setTab('orders')} count={orders.length || undefined} />
            </div>

            {/* ─── Tab: Mua Proxy ─── */}
            {tab === 'buy' && (
                <div className="fade-in">
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

                    {/* Order panel (slide-in below selected card) */}
                    {selected && (
                        <OrderPanel
                            product={selected}
                            onClose={() => setSelected(null)}
                            onSuccess={() => { setSelected(null); setTab('orders') }}
                        />
                    )}
                </div>
            )}

            {/* ─── Tab: Proxy của tôi ─── */}
            {tab === 'orders' && (
                <div className="fade-in">
                    {ordersLoading ? (
                        <div className="loading-spinner">Đang tải đơn hàng...</div>
                    ) : orders.length === 0 ? (
                        <div className="card">
                            <div className="empty-state">
                                <Package size={44} opacity={0.25} />
                                <p>Bạn chưa có đơn proxy nào</p>
                                <button className="btn-primary" onClick={() => setTab('buy')}>
                                    <ShoppingCart size={14} /> Mua Proxy ngay
                                </button>
                            </div>
                        </div>
                    ) : (
                        <>
                            <OrdersTable orders={orders} onCancel={handleCancel} onRefresh={() => qc.invalidateQueries({ queryKey: ['proxy-orders'] })} />
                            {orders.length > 0 && (
                                <Pagination page={page} totalPages={meta.pages ?? 1} total={meta.total ?? 0} limit={20} onPageChange={setPage} />
                            )}
                        </>
                    )}
                </div>
            )}
        </AppLayout>
    )
}
