'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { resellerAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import { formatVND } from '@/lib/format'
import { Tag, Save, Globe, Server } from 'lucide-react'

type PriceRow = {
    id: string
    product_name?: string
    proxy_type?: string
    protocol?: string
    location?: string
    floor_price: string
    sell_price: string
    base_cost?: string
    editValue?: string
}

export default function ResellerPricingPage() {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [editing, setEditing] = useState<Record<string, string>>({})

    const { data, isLoading } = useQuery({
        queryKey: ['reseller-pricing'],
        queryFn: () => resellerAPI.listPricing(),
    })
    const rows: PriceRow[] = data?.data?.data ?? []

    const saveMut = useMutation({
        mutationFn: ({ productId, sellPrice }: { productId: string; sellPrice: string }) =>
            resellerAPI.setPricing(productId, sellPrice),
        onSuccess: (_data, variables) => {
            qc.invalidateQueries({ queryKey: ['reseller-pricing'] })
            setEditing(prev => {
                const copy = { ...prev }
                delete copy[variables.productId]
                return copy
            })
            success('Price updated successfully')
        },
        onError: () => toastError('Failed to update price'),
    })

    const startEdit = (id: string, currentVal: string) => {
        setEditing(prev => ({ ...prev, [id]: currentVal }))
    }

    const calcMarkup = (floor: string, sell: string) => {
        const f = parseFloat(floor || '0')
        const s = parseFloat(sell || '0')
        if (f === 0) return '—'
        const markup = ((s - f) / f) * 100
        return markup >= 0 ? `+${markup.toFixed(1)}%` : `${markup.toFixed(1)}%`
    }

    return (
        <AppLayout breadcrumb={[
            { label: 'Reseller', href: '/reseller' },
            { label: 'Pricing' },
        ]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Pricing Management</h1>
                    <p className="page-subtitle">Set custom sell prices for all proxy products</p>
                </div>
            </div>

            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
                    <Tag size={17} /> Products & Pricing ({rows.length})
                </div>

                {isLoading ? (
                    <div className="loading-spinner">Loading pricing...</div>
                ) : rows.length === 0 ? (
                    <div className="empty-state">
                        <Tag size={40} opacity={0.3} />
                        <p>No products available</p>
                        <p style={{ fontSize: '.8rem' }}>Contact admin to set up proxy products</p>
                    </div>
                ) : (
                    <div className="table-wrapper">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>Product</th>
                                    <th>Type</th>
                                    <th>Location</th>
                                    <th>Floor Price</th>
                                    <th>Your Sell Price</th>
                                    <th>Markup</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {rows.map((row) => {
                                    const isEditing = editing[row.id] !== undefined
                                    const editVal = editing[row.id] ?? row.sell_price
                                    const markup = calcMarkup(row.floor_price, isEditing ? editVal : row.sell_price)
                                    const markupColor = parseFloat(isEditing ? editVal : row.sell_price) < parseFloat(row.floor_price)
                                        ? 'var(--error)'
                                        : 'var(--success)'

                                    return (
                                        <tr key={row.id}>
                                            <td>
                                                <p style={{ fontWeight: 600, color: 'var(--text-heading)', fontSize: '.875rem' }}>
                                                    {row.product_name || `Product ${row.id.slice(0, 6)}`}
                                                </p>
                                            </td>
                                             <td>
                                                <span className="badge badge-secondary" style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem' }}>
                                                    {row.proxy_type === 'datacenter' ? <Server size={10} /> : <Globe size={10} />}
                                                    {row.proxy_type ?? '—'}
                                                </span>
                                             </td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.875rem' }}>
                                                {row.location ?? '—'}
                                            </td>
                                            <td>
                                                <span style={{ fontWeight: 500, color: 'var(--error)', fontSize: '.875rem' }}>
                                                    {formatVND(row.floor_price)}
                                                </span>
                                                <span style={{ fontSize: '.75rem', color: 'var(--text-muted)', marginLeft: '.3rem' }}>min</span>
                                            </td>
                                            <td>
                                                {isEditing ? (
                                                    <input
                                                        className="input"
                                                        type="number"
                                                        step="0.01"
                                                        min={row.floor_price}
                                                        value={editVal}
                                                        onChange={(e) => setEditing(prev => ({ ...prev, [row.id]: e.target.value }))}
                                                        style={{ maxWidth: 110, padding: '.4rem .625rem', fontSize: '.875rem' }}
                                                        autoFocus
                                                    />
                                                ) : (
                                                    <span
                                                        style={{ fontWeight: 600, color: 'var(--success)', fontSize: '.875rem', cursor: 'pointer' }}
                                                        onClick={() => startEdit(row.id, row.sell_price)}
                                                        title="Click to edit"
                                                    >
                                                        {formatVND(row.sell_price)}
                                                    </span>
                                                )}
                                            </td>
                                            <td>
                                                <span style={{ fontWeight: 600, color: markupColor, fontSize: '.875rem' }}>
                                                    {markup}
                                                </span>
                                            </td>
                                            <td>
                                                <div className="action-group">
                                                    {isEditing ? (
                                                        <>
                                                            <button
                                                                className="action-btn green"
                                                                onClick={() => saveMut.mutate({ productId: row.id, sellPrice: editVal })}
                                                                disabled={saveMut.isPending || parseFloat(editVal) < parseFloat(row.floor_price)}
                                                                title="Save"
                                                            >
                                                                <Save size={13} /> Save
                                                            </button>
                                                            <button
                                                                className="action-btn gray"
                                                                onClick={() => setEditing(prev => { const c = { ...prev }; delete c[row.id]; return c })}
                                                                title="Cancel"
                                                            >
                                                                Cancel
                                                            </button>
                                                        </>
                                                    ) : (
                                                        <button
                                                            className="action-btn purple"
                                                            onClick={() => startEdit(row.id, row.sell_price)}
                                                            title="Edit price"
                                                        >
                                                            Edit
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
                )}
            </div>

            <div className="alert alert-info" style={{ marginTop: '1rem' }}>
                <strong>Note:</strong> Sell price must be ≥ floor price. Selling below floor price is not allowed.
                Customers will see your sell price, not the floor price.
            </div>
        </AppLayout>
    )
}
