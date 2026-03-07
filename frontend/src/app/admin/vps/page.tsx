'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { Server, Plus, X, ToggleLeft, ToggleRight, Cpu, MemoryStick, HardDrive } from 'lucide-react'

function AddPlanModal({ onClose }: { onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [form, setForm] = useState({
        name: '', slug: '', cpu_cores: '', ram_mb: '', disk_gb: '',
        hourly_rate: '', monthly_rate: '',
    })

    const mut = useMutation({
        mutationFn: () => adminAPI.createVPSPlan(form),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-vps-plans'] }); success('Plan created'); onClose() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed to create plan'),
    })

    const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement>) => setForm(f => ({ ...f, [k]: e.target.value }))

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 520, width: '95vw' }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}><Plus size={17} /> Add VPS Plan</span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body">
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '.75rem' }}>
                        <div className="form-group">
                            <label>Plan Name</label>
                            <input className="input" placeholder="Starter 1" value={form.name} onChange={set('name')} />
                        </div>
                        <div className="form-group">
                            <label>Slug</label>
                            <input className="input" placeholder="starter-1" value={form.slug} onChange={set('slug')} />
                        </div>
                        <div className="form-group">
                            <label>CPU Cores</label>
                            <input className="input" type="number" min="1" placeholder="2" value={form.cpu_cores} onChange={set('cpu_cores')} />
                        </div>
                        <div className="form-group">
                            <label>RAM (MB)</label>
                            <input className="input" type="number" min="512" placeholder="2048" value={form.ram_mb} onChange={set('ram_mb')} />
                        </div>
                        <div className="form-group">
                            <label>Disk (GB)</label>
                            <input className="input" type="number" min="10" placeholder="40" value={form.disk_gb} onChange={set('disk_gb')} />
                        </div>
                        <div className="form-group">
                            <label>Monthly Rate ($)</label>
                            <input className="input" type="number" step="0.01" min="0" placeholder="19.99" value={form.monthly_rate} onChange={set('monthly_rate')} />
                        </div>
                        <div className="form-group">
                            <label>Hourly Rate ($)</label>
                            <input className="input" type="number" step="0.0001" min="0" placeholder="0.0277" value={form.hourly_rate} onChange={set('hourly_rate')} />
                        </div>
                    </div>
                    <button className="btn-primary" style={{ width: '100%', marginTop: '.5rem' }}
                        onClick={() => mut.mutate()}
                        disabled={!form.name || !form.cpu_cores || !form.ram_mb || !form.disk_gb || mut.isPending}>
                        {mut.isPending ? 'Creating...' : 'Create Plan'}
                    </button>
                </div>
            </div>
        </div>
    )
}

export default function AdminVPSPage() {
    const [page, setPage] = useState(1)
    const [showModal, setShowModal] = useState(false)
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()

    const { data, isLoading } = useQuery({
        queryKey: ['admin-vps-plans', page],
        queryFn: () => adminAPI.listVPSPlans(page),
    })
    const plans = data?.data?.data ?? data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const toggleMut = useMutation({
        mutationFn: (id: string) => adminAPI.toggleVPSPlan(id),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-vps-plans'] }); success('Updated') },
        onError: () => toastError('Failed to update'),
    })

    return (
        <AppLayout breadcrumb={[{ label: 'Admin', href: '/admin' }, { label: 'VPS Plans' }]}>
            {showModal && <AddPlanModal onClose={() => setShowModal(false)} />}

            <div className="page-header">
                <div>
                    <h1 className="page-title">VPS Plans</h1>
                    <p className="page-subtitle">Manage available VPS configurations</p>
                </div>
                <button className="btn-primary" onClick={() => setShowModal(true)}>
                    <Plus size={14} /> Add Plan
                </button>
            </div>

            <div className="card" style={{ padding: 0 }}>
                {isLoading ? (
                    <div className="loading-spinner">Loading...</div>
                ) : plans.length === 0 ? (
                    <div className="empty-state">
                        <Server size={40} opacity={0.3} />
                        <p>No VPS plans yet</p>
                        <button className="btn-primary" onClick={() => setShowModal(true)}><Plus size={14} /> Add first plan</button>
                    </div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Name</th>
                                        <th>CPU</th>
                                        <th>RAM</th>
                                        <th>Disk</th>
                                        <th>Monthly</th>
                                        <th>Hourly</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {plans.map((p: any) => (
                                        <tr key={p.id}>
                                            <td>
                                                <div style={{ fontWeight: 600 }}>{p.name}</div>
                                                <div style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>{p.slug}</div>
                                            </td>
                                            <td>
                                                <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem', fontSize: '.85rem' }}>
                                                    <Cpu size={12} /> {p.cpu_cores} vCPU
                                                </span>
                                            </td>
                                            <td>
                                                <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem', fontSize: '.85rem' }}>
                                                    <MemoryStick size={12} /> {p.ram_mb / 1024} GB
                                                </span>
                                            </td>
                                            <td>
                                                <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem', fontSize: '.85rem' }}>
                                                    <HardDrive size={12} /> {p.disk_gb} GB
                                                </span>
                                            </td>
                                            <td><strong>${parseFloat(p.monthly_rate ?? 0).toFixed(2)}</strong></td>
                                            <td style={{ color: 'var(--text-muted)', fontSize: '.82rem' }}>${parseFloat(p.hourly_rate ?? 0).toFixed(4)}</td>
                                            <td>
                                                <button
                                                    className="action-btn"
                                                    onClick={() => toggleMut.mutate(p.id)}
                                                    disabled={toggleMut.isPending}
                                                    style={{ color: p.is_active ? 'var(--success)' : 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '.3rem' }}
                                                    title={p.is_active ? 'Click to deactivate' : 'Click to activate'}
                                                >
                                                    {p.is_active ? <ToggleRight size={18} /> : <ToggleLeft size={18} />}
                                                    {p.is_active ? 'Active' : 'Inactive'}
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                        <Pagination page={page} totalPages={meta.pages ?? 1} total={meta.total ?? 0} limit={20} onPageChange={setPage} />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
