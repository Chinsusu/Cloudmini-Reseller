'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { vpsAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { formatVND } from '@/lib/format'
import {
    Server, Play, Square, RotateCcw, Terminal, Trash2,
    Plus, X, Cpu, MemoryStick, HardDrive, RefreshCw, Activity
} from 'lucide-react'

const STATUS_COLOR: Record<string, string> = {
    running: 'success', booting: 'info', provisioning: 'info',
    stopped: 'secondary', suspended: 'warning',
    pending: 'warning', terminated: 'error',
}

function DeployModal({ onClose }: { onClose: () => void }) {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const [selectedPlan, setSelectedPlan] = useState<any>(null)
    const [hostname, setHostname] = useState('')
    const [deploying, setDeploying] = useState(false)

    const { data, isLoading } = useQuery({
        queryKey: ['vps-plans'],
        queryFn: () => vpsAPI.listPlans(),
    })
    const plans = data?.data?.data ?? data?.data ?? []

    const hostnameValid = /^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$/.test(hostname)

    const handleDeploy = async () => {
        if (!selectedPlan || !hostname) return
        setDeploying(true)
        try {
            await vpsAPI.createVPS(selectedPlan.id, hostname)
            success('VPS is being provisioned! It will be ready shortly.')
            qc.invalidateQueries({ queryKey: ['vps-instances'] })
            onClose()
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Deployment failed')
        } finally {
            setDeploying(false)
        }
    }

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 720, width: '95vw', maxHeight: '90vh', overflowY: 'auto' }}
                onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem', fontWeight: 700 }}>
                        <Server size={18} /> Deploy VPS
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body">
                    <p style={{ color: 'var(--text-muted)', marginBottom: '1rem', fontSize: '.88rem' }}>
                        Choose a plan and hostname. Your VPS will be provisioned within 2–5 minutes.
                    </p>

                    {/* Plans grid */}
                    {isLoading ? (
                        <div className="loading-spinner">Loading plans...</div>
                    ) : plans.length === 0 ? (
                        <div className="empty-state"><Server size={36} opacity={0.3} /><p>No plans available</p></div>
                    ) : (
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill,minmax(200px,1fr))', gap: '.65rem', marginBottom: '1.25rem' }}>
                            {plans.map((plan: any) => (
                                <div key={plan.id} onClick={() => setSelectedPlan(plan)}
                                    style={{
                                        border: '1px solid',
                                        borderColor: selectedPlan?.id === plan.id ? 'var(--dc-gold)' : 'var(--border)',
                                        borderRadius: 10, padding: '1rem', cursor: 'pointer',
                                        background: selectedPlan?.id === plan.id ? 'rgba(230,168,23,.07)' : 'var(--surface)',
                                        transition: 'all .15s',
                                        boxShadow: selectedPlan?.id === plan.id ? 'var(--shadow)' : 'var(--shadow-sm)',
                                    }}>
                                    <div style={{ fontWeight: 700, marginBottom: '.5rem' }}>{plan.name}</div>
                                    <div style={{ fontSize: '.82rem', color: 'var(--text)', display: 'flex', flexDirection: 'column', gap: '.2rem', marginBottom: '.6rem' }}>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem' }}><Cpu size={12} />{plan.cpu_cores} vCPU</span>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem' }}><MemoryStick size={12} />{plan.ram_mb / 1024} GB RAM</span>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.3rem' }}><HardDrive size={12} />{plan.disk_gb} GB SSD</span>
                                    </div>
                                    <div style={{ fontWeight: 800, color: selectedPlan?.id === plan.id ? 'var(--dc-gold)' : 'var(--text-heading)' }}>
                                        {formatVND(plan.monthly_rate)}<span style={{ fontWeight: 400, fontSize: '.78rem', color: 'var(--text-muted)' }}>/tháng</span>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Hostname */}
                    {selectedPlan && (
                        <div>
                            <div className="form-group">
                                <label>Hostname</label>
                                <input className="input" placeholder="my-server-01" value={hostname}
                                    onChange={e => setHostname(e.target.value.toLowerCase())} />
                                {hostname && !hostnameValid && (
                                    <small style={{ color: 'var(--error)' }}>Lowercase letters, numbers, hyphens only (3-63 chars)</small>
                                )}
                            </div>
                            <div style={{ background: 'var(--surface-raised)', borderRadius: 8, padding: '.8rem 1rem', marginBottom: '1rem', fontSize: '.85rem', border: '1px solid var(--border)' }}>
                                <strong style={{ color: 'var(--text-heading)' }}>{selectedPlan.name}</strong> · {selectedPlan.cpu_cores} vCPU · {selectedPlan.ram_mb / 1024}GB RAM · {selectedPlan.disk_gb}GB SSD<br />
                                <span style={{ color: 'var(--dc-gold)', fontWeight: 700 }}>
                                    {formatVND(selectedPlan.monthly_rate)}/tháng
                                    &nbsp;({formatVND(selectedPlan.hourly_rate)}/giờ)
                                </span>
                            </div>
                            <button className="btn-primary" onClick={handleDeploy}
                                disabled={!hostname || !hostnameValid || deploying} style={{ width: '100%' }}>
                                <Server size={15} />
                                {deploying ? 'Deploying...' : 'Deploy VPS'}
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default function VPSPage() {
    const [page, setPage] = useState(1)
    const [showModal, setShowModal] = useState(false)
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['vps-instances', page],
        queryFn: () => vpsAPI.listInstances(page),
        refetchInterval: 10000,
    })
    const instances = data?.data?.data ?? []
    const meta = data?.data?.meta ?? {}

    const action = (fn: () => Promise<any>, label: string) => async () => {
        try { await fn(); success(`${label} successful`); refetch() }
        catch (e: any) { toastError(e?.response?.data?.error?.message ?? `${label} failed`) }
    }

    const handleTerminate = async (id: string, hostname: string) => {
        const ok = await confirm({ title: 'Terminate VPS', message: `Terminate "${hostname}"? All data will be lost.`, confirmLabel: 'Terminate', variant: 'danger' })
        if (!ok) return
        try { await vpsAPI.terminateInstance(id); success('Terminated'); qc.invalidateQueries({ queryKey: ['vps-instances'] }) }
        catch (e: any) { toastError(e?.response?.data?.error?.message ?? 'Failed to terminate') }
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'VPS Instances' }]}>
            {confirmDialog}
            {showModal && <DeployModal onClose={() => setShowModal(false)} />}

            <div className="page-header">
                <div>
                    <h1 className="page-title">VPS Instances</h1>
                    <p className="page-subtitle">{meta.total ?? 0} instances · auto-refreshes every 10s</p>
                </div>
                <div style={{ display: 'flex', gap: '.6rem' }}>
                    <button className="topbar-icon-btn" onClick={() => refetch()} title="Refresh"><RefreshCw size={15} /></button>
                    <button className="btn-primary" onClick={() => setShowModal(true)}><Plus size={14} /> Deploy VPS</button>
                </div>
            </div>

            <div className="card" style={{ padding: 0 }}>
                {isLoading ? (
                    <div className="loading-spinner">Loading instances...</div>
                ) : instances.length === 0 ? (
                    <div className="empty-state">
                        <Server size={44} opacity={0.3} />
                        <p>No VPS instances yet</p>
                        <button className="btn-primary" onClick={() => setShowModal(true)}><Plus size={14} /> Deploy your first VPS</button>
                    </div>
                ) : (
                    <>
                        <div className="table-wrapper">
                            <table className="data-table">
                                <thead>
                                    <tr>
                                        <th>Hostname</th>
                                        <th>Plan</th>
                                        <th>IP Address</th>
                                        <th>Status</th>
                                        <th>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {instances.map((inst: any) => (
                                        <tr key={inst.id}>
                                            <td>
                                                <div style={{ fontWeight: 600 }}>{inst.hostname}</div>
                                                <div style={{ fontSize: '.75rem', color: 'var(--text-muted)' }}>VMID: {inst.vmid}</div>
                                            </td>
                                            <td style={{ fontSize: '.85rem' }}>{inst.plan_id ?? '—'}</td>
                                            <td><code style={{ fontSize: '.82rem' }}>{inst.ip_address ?? 'pending...'}</code></td>
                                            <td>
                                                <span className={`badge badge-${STATUS_COLOR[inst.status] ?? 'secondary'}`} style={{ display: 'inline-flex', alignItems: 'center', gap: '.3rem' }}>
                                                    {inst.status === 'running' && <span style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--success)', display: 'inline-block', flexShrink: 0 }} className="pulse" />}
                                                    {inst.status}
                                                </span>
                                            </td>
                                            <td>
                                                <div style={{ display: 'flex', gap: '.3rem', flexWrap: 'wrap' }}>
                                                    {inst.status === 'stopped' && (
                                                        <button className="action-btn green" onClick={action(() => vpsAPI.startInstance(inst.id), 'Start')} title="Start">
                                                            <Play size={12} /> Start
                                                        </button>
                                                    )}
                                                    {inst.status === 'running' && (<>
                                                        <button className="action-btn" onClick={action(() => vpsAPI.stopInstance(inst.id), 'Stop')} title="Stop">
                                                            <Square size={12} /> Stop
                                                        </button>
                                                        <button className="action-btn" onClick={action(() => vpsAPI.rebootInstance(inst.id), 'Reboot')} title="Reboot">
                                                            <RotateCcw size={12} /> Reboot
                                                        </button>
                                                    </>)}
                                                    {(inst.status === 'running' || inst.status === 'stopped') && (
                                                        <button className="action-btn" onClick={async () => {
                                                            const res = await vpsAPI.getConsole(inst.id)
                                                            window.open(res.data?.data?.console_url, '_blank')
                                                        }} title="Console"><Terminal size={12} /> Console</button>
                                                    )}
                                                    {inst.status !== 'terminated' && (
                                                        <button className="action-btn red" onClick={() => handleTerminate(inst.id, inst.hostname)} title="Terminate">
                                                            <Trash2 size={12} /> Terminate
                                                        </button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                        <Pagination page={page} totalPages={meta.pages ?? 1} total={meta.total ?? 0} limit={15} onPageChange={setPage} />
                    </>
                )}
            </div>
        </AppLayout>
    )
}
