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
function ProductCard({ product, selected, onClick }: { product: any; selected: boolean; onClick: () => void }) {
    const isRotating = product.metadata?.service_id?.includes('rotating')
    return (
        <div onClick={onClick} style={{
            background: selected ? 'rgba(230,168,23,.07)' : 'var(--surface)',
            border: '1px solid',
            borderColor: selected ? 'var(--dc-gold)' : 'var(--border)',
            borderRadius: 'var(--radius-xl)',
            padding: '1.35rem',
            cursor: 'pointer',
            transition: 'background .15s, border-color .15s, box-shadow .15s',
            boxShadow: selected ? 'var(--shadow)' : 'var(--shadow-sm)',
            position: 'relative',
        }}>
            {selected && (
                <div style={{ position: 'absolute', top: 12, right: 12 }}>
                    <CheckCircle2 size={18} color="var(--dc-gold)" />
                </div>
            )}

            {/* Type badge */}
            <div style={{ marginBottom: '.65rem' }}>
                <span style={{
                    display: 'inline-block', fontSize: '.7rem', fontWeight: 700,
                    textTransform: 'uppercase', letterSpacing: '.07em',
                    color: selected ? 'var(--dc-gold)' : 'var(--text-muted)',
                    background: selected ? 'rgba(230,168,23,.15)' : 'rgba(255,255,255,.06)',
                    padding: '.2rem .65rem', borderRadius: 'var(--radius-pill)',
                }}>
                    {product.proxy_type} · {product.protocol?.toUpperCase()}
                </span>
            </div>

            {/* Name */}
            <div style={{ fontWeight: 700, fontSize: '1rem', color: 'var(--text-heading)', marginBottom: '.6rem', lineHeight: 1.3, paddingRight: selected ? '1.6rem' : 0 }}>
                {product.name}
            </div>

            {/* Specs */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '.25rem', marginBottom: '.9rem' }}>
                {product.location && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                        <Globe size={11} /> {product.location}
                    </div>
                )}
                {product.duration_days && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                        <Clock size={11} /> {product.duration_days} days
                    </div>
                )}
                {product.bandwidth_gb && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                        <Database size={11} /> {product.bandwidth_gb} GB
                    </div>
                )}
                {!product.duration_days && !product.bandwidth_gb && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                        <Zap size={11} /> Monthly / per GB
                    </div>
                )}
            </div>

            {/* Divider */}
            <div style={{ height: 1, background: 'var(--border-light)', marginBottom: '.9rem' }} />

            {/* Price + CTA */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div>
                    <span style={{ fontWeight: 800, fontSize: '1.2rem', color: selected ? 'var(--dc-gold)' : 'var(--text-heading)' }}>
                        {formatVND(product.base_cost)}
                    </span>
                    <span style={{ fontSize: '.75rem', color: 'var(--text-muted)', marginLeft: '.15rem' }}>
                        {isRotating ? '/GB' : '/tháng'}
                    </span>
                </div>
                <span style={{ fontSize: '.78rem', fontWeight: 600, color: selected ? 'var(--dc-gold)' : 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '.2rem' }}>
                    {selected ? 'Selected' : 'Select'} <ChevronRight size={13} style={{ transform: selected ? 'rotate(90deg)' : 'none', transition: 'transform .15s' }} />
                </span>
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
        setPlacing(true)
        try {
            await proxyAPI.createOrder(product.id, qty, {
                country: country || undefined,
                isp_id: ispId || undefined,
                period_months: isStatic ? periodMonths : undefined,
            })
            success('Order placed! Proxy is being activated...')
            qc.invalidateQueries({ queryKey: ['proxy-orders'] })
            onSuccess()
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Order failed')
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

// ─── Orders Table ─────────────────────────────────────────────────────────────
function OrdersTable({ orders, onCancel }: { orders: any[]; onCancel: (id: string, num: string) => void }) {
    const [revealed, setRevealed] = useState<Record<string, any>>({})
    const { success, error: toastError } = useToast()

    const revealMut = useMutation({
        mutationFn: (id: string) => proxyAPI.getCredentials(id),
        onSuccess: (res, id) => setRevealed(prev => ({ ...prev, [id]: res.data?.data })),
        onError: () => toastError('Failed to load credentials'),
    })

    const handleCopy = async (id: string) => {
        const c = revealed[id]
        if (!c) return
        await navigator.clipboard.writeText(`${c.username}:${c.password}@${c.host}:${c.port}`)
        success('Credentials copied!')
    }

    if (orders.length === 0) return null

    return (
        <div className="card" style={{ padding: 0 }}>
            <div style={{ padding: '1rem 1.5rem', borderBottom: '1px solid var(--border-light)', display: 'flex', alignItems: 'center', gap: '.5rem' }}>
                <Package size={16} color="var(--dc-gold)" />
                <span style={{ fontWeight: 600, color: 'var(--text-heading)' }}>My Orders</span>
                <span className="badge badge-secondary" style={{ marginLeft: '.25rem' }}>{orders.length}</span>
            </div>
            <div className="table-wrapper">
                <table className="data-table">
                    <thead>
                        <tr>
                            <th>Order</th>
                            <th>Type</th>
                            <th>Qty</th>
                            <th>Amount</th>
                            <th>Expires</th>
                            <th>Status</th>
                            <th>Credentials</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {orders.map((o: any) => {
                            const st = STATUS[o.status] ?? { label: o.status, cls: 'badge-secondary' }
                            return (
                                <tr key={o.id}>
                                    <td>
                                        <code style={{ fontSize: '.78rem', background: 'rgba(255,255,255,.06)', padding: '.15rem .45rem', borderRadius: 4, color: 'var(--text-heading)' }}>{o.order_number}</code>
                                        {o.product_name && <div style={{ fontSize: '.75rem', color: 'var(--text-muted)', marginTop: 2 }}>{o.product_name}</div>}
                                    </td>
                                    <td><span className="badge badge-info">{o.proxy_type ?? '—'}</span></td>
                                    <td style={{ fontWeight: 600 }}>{o.quantity}</td>
                                    <td><strong>{formatVND(o.total_amount)}</strong></td>
                                    <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>
                                        {o.expires_at ? new Date(o.expires_at).toLocaleDateString() : '—'}
                                    </td>
                                    <td><span className={`badge ${st.cls}`}>{st.label}</span></td>
                                    <td>
                                        {o.status === 'active' ? (
                                            revealed[o.id] ? (
                                                <div style={{ display: 'flex', gap: '.35rem', alignItems: 'center' }}>
                                                    <code style={{ fontSize: '.73rem', background: 'rgba(255,255,255,.06)', padding: '.15rem .4rem', borderRadius: 4, maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--text-heading)' }}>
                                                        {revealed[o.id].username}:•••@{revealed[o.id].host}
                                                    </code>
                                                    <button className="action-btn blue" onClick={() => handleCopy(o.id)} title="Copy"><Copy size={11} /></button>
                                                    <button className="action-btn gray" onClick={() => setRevealed(p => { const n = { ...p }; delete n[o.id]; return n })}><EyeOff size={11} /></button>
                                                </div>
                                            ) : (
                                                <button className="action-btn purple" onClick={() => revealMut.mutate(o.id)} disabled={revealMut.isPending}>
                                                    <Eye size={12} /> View
                                                </button>
                                            )
                                        ) : o.status === 'processing' ? (
                                            <span style={{ fontSize: '.78rem', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '.3rem' }}>
                                                <div className="pulse" style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--warning)', flexShrink: 0 }} /> Activating...
                                            </span>
                                        ) : '—'}
                                    </td>
                                    <td>
                                        {(o.status === 'active' || o.status === 'pending') && (
                                            <button className="action-btn red" onClick={() => onCancel(o.id, o.order_number)}>
                                                <XCircle size={12} /> Cancel
                                            </button>
                                        )}
                                    </td>
                                </tr>
                            )
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    )
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function ProxyOrdersPage() {
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
        onSuccess: () => { success('Order cancelled'); qc.invalidateQueries({ queryKey: ['proxy-orders'] }) },
        onError: (err: any) => toastError(err?.response?.data?.error?.message ?? 'Cancel failed'),
    })
    const handleCancel = async (id: string, num: string) => {
        const ok = await confirm({ title: 'Cancel Order', message: `Cancel order ${num}? This cannot be undone.`, confirmLabel: 'Cancel Order', variant: 'danger' })
        if (ok) cancelMut.mutate(id)
    }

    const typeFilters = [
        { value: '', label: 'All' },
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
                    <p className="page-subtitle">Select a product to configure and place an order</p>
                </div>
                <button className="topbar-icon-btn" onClick={() => refetch()} title="Refresh">
                    <RefreshCw size={15} />
                </button>
            </div>

            {/* Filter tabs — gold active */}
            <div style={{ display: 'flex', gap: '.4rem', marginBottom: '1.25rem', flexWrap: 'wrap' }}>
                {typeFilters.map(f => (
                    <button key={f.value} onClick={() => { setFilterType(f.value); setSelected(null) }}
                        style={{
                            padding: '.42rem 1.1rem',
                            borderRadius: 'var(--radius-pill)',
                            fontSize: '.82rem',
                            fontWeight: 600,
                            border: filterType === f.value ? '1px solid rgba(230,168,23,.4)' : '1px solid var(--border)',
                            background: filterType === f.value ? 'rgba(230,168,23,.12)' : 'var(--surface)',
                            color: filterType === f.value ? 'var(--dc-gold)' : 'var(--text)',
                            cursor: 'pointer',
                            transition: 'all .15s',
                            boxShadow: filterType === f.value ? '0 0 0 1px rgba(230,168,23,.15)' : 'none',
                        }}>
                        {f.label}
                    </button>
                ))}
            </div>

            {/* Products grid */}
            {prodLoading ? (
                <div className="loading-spinner">Loading products...</div>
            ) : products.length === 0 ? (
                <div className="card">
                    <div className="empty-state">
                        <Globe size={40} opacity={0.25} />
                        <p>No proxy products available</p>
                        <span style={{ fontSize: '.82rem', color: 'var(--text-muted)' }}>Ask your admin to add products</span>
                    </div>
                </div>
            ) : (
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(210px, 1fr))', gap: '1rem', padding: '2px' }}>
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

            {/* Inline Order Panel */}
            {selected && (
                <OrderPanel
                    product={selected}
                    onClose={() => setSelected(null)}
                    onSuccess={() => setSelected(null)}
                />
            )}

            {/* Orders */}
            <div style={{ marginTop: '1.75rem' }}>
                {ordersLoading ? (
                    <div className="loading-spinner">Loading orders...</div>
                ) : (
                    <>
                        <OrdersTable orders={orders} onCancel={handleCancel} />
                        {orders.length > 0 && (
                            <Pagination page={page} totalPages={meta.pages ?? 1} total={meta.total ?? 0} limit={20} onPageChange={setPage} />
                        )}
                        {orders.length === 0 && products.length > 0 && !selected && (
                            <div style={{ textAlign: 'center', color: 'var(--text-muted)', fontSize: '.875rem', padding: '2rem 0' }}>
                                <Package size={28} opacity={0.25} style={{ display: 'block', margin: '0 auto .5rem' }} />
                                No orders yet — select a product above to get started
                            </div>
                        )}
                    </>
                )}
            </div>
        </AppLayout>
    )
}
