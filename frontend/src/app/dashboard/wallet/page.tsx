'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { walletAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
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

    const txs = txData?.data?.data ?? []
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
        ['deposit', 'order_refund', 'hold_release', 'adjustment'].includes(type)

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
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#7367F0,#9e95f5)' }}>
                        <Wallet size={22} />
                    </div>
                    <div>
                        <p className="stat-label">Total Balance</p>
                        <p className="stat-value">${balance.toFixed(2)}</p>
                        <p className="stat-sub">Gross balance</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#28C76F,#48DA89)' }}>
                        <TrendingUp size={22} />
                    </div>
                    <div>
                        <p className="stat-label">Available</p>
                        <p className="stat-value" style={{ color: 'var(--success)' }}>${available.toFixed(2)}</p>
                        <p className="stat-sub">Ready to spend</p>
                    </div>
                </div>
                <div className="stat-card">
                    <div className="stat-icon" style={{ background: 'linear-gradient(135deg,#FF9F43,#FFB976)' }}>
                        <Lock size={22} />
                    </div>
                    <div>
                        <p className="stat-label">On Hold</p>
                        <p className="stat-value" style={{ color: 'var(--warning)' }}>${hold.toFixed(2)}</p>
                        <p className="stat-sub">Reserved for active orders</p>
                    </div>
                </div>
            </div>

            {/* Top-up Form */}
            <div className="card">
                <div className="card-header"><ArrowUpRight size={17} /> Top Up Balance</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: '.75rem', alignItems: 'flex-end' }}>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Amount (USD)</label>
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
                    Minimum top-up: $5.00. Balance is updated upon payment confirmation.
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
                                        return (
                                            <tr key={tx.id}>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.82rem', whiteSpace: 'nowrap' }}>
                                                    {new Date(tx.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                                                </td>
                                                <td>
                                                    <span className={`badge badge-${tx.type}`}>
                                                        {tx.type.replace(/_/g, ' ')}
                                                    </span>
                                                </td>
                                                <td style={{ color: 'var(--text-muted)', fontSize: '.85rem' }}>
                                                    {tx.description || '—'}
                                                </td>
                                                <td style={{ textAlign: 'right', fontWeight: 600, color: credit ? 'var(--success)' : 'var(--error)' }}>
                                                    <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '.2rem' }}>
                                                        {credit ? <TrendingUp size={13} /> : <TrendingDown size={13} />}
                                                        {credit ? '+' : '-'}${parseFloat(tx.amount || '0').toFixed(2)}
                                                    </span>
                                                </td>
                                                <td style={{ fontSize: '.875rem', color: 'var(--text-muted)' }}>
                                                    {tx.balance_after ? `$${parseFloat(tx.balance_after).toFixed(2)}` : '—'}
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
                            totalPages={meta.pages ?? 1}
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
