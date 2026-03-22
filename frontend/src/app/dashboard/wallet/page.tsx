'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { walletAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { formatVND } from '@/lib/format'
import { Wallet, TrendingUp, TrendingDown, Lock, CreditCard, ArrowUpRight } from 'lucide-react'

export default function WalletPage() {
    const [page, setPage] = useState(1)
    const [amount, setAmount] = useState('')
    const [paymentMethod, setPaymentMethod] = useState('bank_transfer')
    const { success, error: toastError } = useToast()
    const qc = useQueryClient()

    const { data: walletData } = useQuery({
        queryKey: ['wallet'],
        queryFn: () => walletAPI.getBalance(),
    })
    const { data: txData, isLoading: txLoading } = useQuery({
        queryKey: ['transactions', page],
        queryFn: () => walletAPI.getTransactions(page),
    })

    const wallet = walletData?.data?.data ?? {}
    const balance = parseFloat(wallet.balance ?? '0')
    const hold = parseFloat(wallet.hold_amount ?? '0')
    const available = balance - hold

    // Filter out intermediate 'hold' entries — hold_confirm represents the actual charge
    const txs = (txData?.data?.data ?? []).filter((t: any) => t.type !== 'hold')
    const meta = txData?.data?.meta ?? {}

    const topUpMut = useMutation({
        mutationFn: () => walletAPI.topUp(amount, paymentMethod),
        onSuccess: () => {
            setAmount('')
            qc.invalidateQueries({ queryKey: ['wallet'] })
            qc.invalidateQueries({ queryKey: ['transactions'] })
            success('Top-up request submitted! Your balance will be updated after confirmation.')
        },
        onError: () => toastError('Failed to submit top-up request'),
    })

    const isCredit = (type: string) =>
        ['deposit', 'order_refund', 'hold_release', 'adjustment', 'refund'].includes(type)

    const TX_LABEL: Record<string, string> = {
        deposit: 'Nạp tiền', refund: 'Hoàn tiền', hold_release: 'Hoàn giữ',
        adjustment: 'Điều chỉnh', hold_confirm: 'Thanh toán', hold: 'Giữ tiền', debit: 'Trừ tiền',
    }

    return (
        <AppLayout breadcrumb={[
            { label: 'Dashboard', href: '/dashboard' },
            { label: 'Wallet' },
        ]}>
            <div className="page-header">
                <div>
                    <h1 className="page-title">Wallet</h1>
                    <p className="page-subtitle">Manage your balance and transactions</p>
                </div>
            </div>

            {/* Balance Cards */}
            <div className="stats-grid" style={{ marginBottom: '1.75rem' }}>
                {/* Balance */}
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--dc-gold)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(230,168,23,.15)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Wallet size={18} color="var(--dc-gold)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Tổng số dư</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--dc-gold)', lineHeight: 1.2 }}>{formatVND(balance)}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Số dư gộp</p>
                        </div>
                    </div>
                </div>
                {/* Available */}
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--success)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(40,199,111,.12)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <TrendingUp size={18} color="var(--success)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Khả dụng</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--success)', lineHeight: 1.2 }}>{formatVND(available)}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Sẵn sàng chi tiêu</p>
                        </div>
                    </div>
                </div>
                {/* On Hold */}
                <div style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderTop: '3px solid var(--warning)', borderRadius: '0 0 var(--radius-xl) var(--radius-xl)', padding: '1.25rem 1.5rem', boxShadow: 'var(--shadow-sm)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                        <div style={{ width: 40, height: 40, borderRadius: 10, background: 'rgba(255,159,67,.12)', display: 'grid', placeItems: 'center', flexShrink: 0 }}>
                            <Lock size={18} color="var(--warning)" />
                        </div>
                        <div>
                            <p style={{ fontSize: '.72rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '.07em' }}>Tạm giữ</p>
                            <p style={{ fontSize: '1.6rem', fontWeight: 800, color: 'var(--warning)', lineHeight: 1.2 }}>{formatVND(hold)}</p>
                            <p style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>Dành cho đơn hàng</p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Top-up Form */}
            <div className="card">
                <div className="card-header"><ArrowUpRight size={17} /> Top Up Balance</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: '.75rem', alignItems: 'flex-end' }}>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Số tiền (VND)</label>
                        <input
                            className="input"
                            type="number"
                            min="5"
                            step="1"
                            placeholder="e.g. 50"
                            value={amount}
                            onChange={(e) => setAmount(e.target.value)}
                        />
                    </div>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Payment Method</label>
                        <select
                            className="input"
                            value={paymentMethod}
                            onChange={(e) => setPaymentMethod(e.target.value)}
                            style={{ cursor: 'pointer' }}
                        >
                            <option value="bank_transfer">Bank Transfer</option>
                            <option value="crypto">Crypto (USDT/USDC)</option>
                            <option value="stripe">Stripe (Card)</option>
                            <option value="vnpay">VNPay</option>
                        </select>
                    </div>
                    <button
                        className="btn-primary"
                        onClick={() => topUpMut.mutate()}
                        disabled={!amount || parseFloat(amount) < 5 || topUpMut.isPending}
                    >
                        <CreditCard size={16} />
                        {topUpMut.isPending ? 'Submitting...' : 'Request Top-Up'}
                    </button>
                </div>
                <p style={{ fontSize: '.8rem', color: 'var(--text-muted)', marginTop: '.75rem' }}>
                    Nạp tối thiểu: 50.000đ. Số dư sẽ cập nhật sau khi xác nhận thanh toán.
                </p>
            </div>

            {/* Transaction History */}
            <div className="card" style={{ padding: 0 }}>
                <div className="card-header" style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border-light)', marginBottom: 0 }}>
                    <TrendingDown size={17} /> Transaction History
                </div>
                {txLoading ? (
                    <div className="loading-spinner">Loading transactions...</div>
                ) : txs.length === 0 ? (
                    <div className="empty-state">
                        <Wallet size={40} opacity={0.3} />
                        <p>No transactions yet</p>
                    </div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Date</th>
                                        <th>Type</th>
                                        <th>Description</th>
                                        <th style={{ textAlign: 'right' }}>Amount</th>
                                        <th>Balance After</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {txs.map((tx: any) => {
                                        const credit = isCredit(tx.type)
                                        const label = TX_LABEL[tx.type] ?? tx.type.replace(/_/g, ' ')
                                        return (
                                            <tr key={tx.id}>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.82rem', whiteSpace: 'nowrap' }}>
                                                    {(() => { const d = new Date(tx.created_at); return `${String(d.getDate()).padStart(2,'0')}/${String(d.getMonth()+1).padStart(2,'0')}/${d.getFullYear()}` })()}
                                                </td>
                                                <td>
                                                    <span className={`badge badge-${tx.type}`}>{label}</span>
                                                </td>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>
                                                    {tx.description || '—'}
                                                </td>
                                                <td style={{ textAlign: 'right', fontWeight: 600, color: credit ? 'var(--success)' : 'var(--error)' }}>
                                                    <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '.2rem' }}>
                                                        {credit ? <TrendingUp size={13} /> : <TrendingDown size={13} />}
                                                        {credit ? '+' : '-'}{formatVND(Math.abs(tx.amount))}
                                                    </span>
                                                </td>
                                                <td style={{ fontSize: '.875rem', color: 'var(--text-muted)' }}>
                                                    {tx.balance_after ? formatVND(tx.balance_after) : '—'}
                                                </td>
                                                <td>
                                                    <span className="badge badge-success">{tx.status || 'completed'}</span>
                                                </td>
                                            </tr>
                                        )
                                    })}
                                </tbody>
                            </table>
                        </div>
                        <Pagination
                            page={page}
                            totalPages={meta.total_pages ?? 1}
                            total={meta.total ?? 0}
                            limit={20}
                            onPageChange={setPage}
                        />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
