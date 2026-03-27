'use client'
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { vpsAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { Pagination } from '@/components/ui/Pagination'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { formatVND } from '@/lib/format'
import {
    Server, Play, Square, RotateCcw, Terminal, Trash2,
    Plus, X, Cpu, MemoryStick, HardDrive, RefreshCw, Rocket
} from 'lucide-react'

const STATUS_COLOR: Record<string, string> = {
    running: 'success', booting: 'info', provisioning: 'info',
    stopped: 'secondary', suspended: 'warning',
    pending: 'warning', terminated: 'error',
}

// ─── Tab Button ───────────────────────────────────────────────────────────────
function TabBtn({ label, active, onClick, count }: { label: string; active: boolean; onClick: () => void; count?: number }) {
    return (
        <button onClick={onClick} style={{
            display: 'inline-flex', alignItems: 'center', gap: '.4rem',
            padding: '.5rem 1.2rem',
            background: 'transparent',
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
                    background: active ? 'rgba(230,168,23,.15)' : 'var(--surface-raised)',
                    color: active ? 'var(--dc-gold)' : 'var(--text-muted)',
                    fontSize: '.72rem', fontWeight: 700,
                }}>{count}</span>
            )}
        </button>
    )
}

// ─── Deploy Tab ───────────────────────────────────────────────────────────────
function DeployTab({ onDeployed }: { onDeployed: () => void }) {
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
            success('VPS đang được khởi tạo! Sẽ sẵn sàng trong 2–5 phút.')
            qc.invalidateQueries({ queryKey: ['vps-instances'] })
            onDeployed()
        } catch (err: any) {
            toastError(err?.response?.data?.error?.message ?? 'Deploy thất bại')
        } finally {
            setDeploying(false)
        }
    }

    return (
        <div className="fade-in">
            <div className="card" style={{ marginBottom: '1.5rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem', marginBottom: '1.25rem' }}>
                    <div style={{ width: 36, height: 36, borderRadius: 'var(--radius)', background: 'rgba(115,103,240,.15)', display: 'grid', placeItems: 'center' }}>
                        <Rocket size={18} color="var(--primary)" />
                    </div>
                    <div>
                        <div style={{ fontWeight: 700, color: 'var(--text-heading)' }}>Chọn gói VPS</div>
                        <div style={{ fontSize: '.82rem', color: 'var(--text-muted)' }}>Nhấn vào một gói để chọn</div>
                    </div>
                </div>

                {isLoading ? (
                    <div className="loading-spinner">Đang tải gói VPS...</div>
                ) : plans.length === 0 ? (
                    <div className="empty-state"><Server size={36} opacity={0.3} /><p>Chưa có gói VPS nào</p></div>
                ) : (
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '.75rem' }}>
                        {plans.map((plan: any) => {
                            const isSel = selectedPlan?.id === plan.id
                            return (
                                <div key={plan.id} onClick={() => setSelectedPlan(plan)} style={{
                                    border: '1px solid var(--border)',
                                    borderRadius: 'var(--radius-xl)',
                                    padding: '1.1rem',
                                    cursor: 'pointer',
                                    transition: 'transform .15s, box-shadow .15s',
                                    boxShadow: isSel
                                        ? '0 0 0 2px var(--dc-gold), 0 4px 16px rgba(230,168,23,.15)'
                                        : '0 2px 6px rgba(0,0,0,.12)',
                                    background: isSel ? 'rgba(230,168,23,.04)' : 'var(--surface)',
                                    transform: isSel ? 'translateY(-2px)' : undefined,
                                }}
                                    onMouseEnter={e => { if (!isSel) (e.currentTarget as HTMLDivElement).style.transform = 'translateY(-3px)' }}
                                    onMouseLeave={e => { if (!isSel) (e.currentTarget as HTMLDivElement).style.transform = '' }}
                                >
                                    <div style={{ fontWeight: 700, marginBottom: '.6rem', color: isSel ? 'var(--dc-gold)' : 'var(--text-heading)' }}>{plan.name}</div>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: '.25rem', marginBottom: '.75rem' }}>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                                            <Cpu size={12} />{plan.cpu_cores} vCPU
                                        </span>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                                            <MemoryStick size={12} />{plan.ram_mb / 1024} GB RAM
                                        </span>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '.35rem', fontSize: '.8rem', color: 'var(--text-muted)' }}>
                                            <HardDrive size={12} />{plan.disk_gb} GB SSD
                                        </span>
                                    </div>
                                    <div style={{ fontWeight: 800, fontSize: '1.05rem', color: isSel ? 'var(--dc-gold)' : 'var(--text-heading)' }}>
                                        {formatVND(plan.monthly_rate)}
                                        <span style={{ fontSize: '.72rem', fontWeight: 400, color: 'var(--text-muted)', marginLeft: '.2rem' }}>/tháng</span>
                                    </div>
                                </div>
                            )
                        })}
                    </div>
                )}
            </div>

            {/* Hostname + Deploy */}
            {selectedPlan && (
                <div className="card fade-in" style={{ maxWidth: 540 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem', marginBottom: '1.25rem' }}>
                        <div style={{ width: 36, height: 36, borderRadius: 'var(--radius)', background: 'rgba(230,168,23,.12)', display: 'grid', placeItems: 'center' }}>
                            <Server size={18} color="var(--dc-gold)" />
                        </div>
                        <div>
                            <div style={{ fontWeight: 700, color: 'var(--dc-gold)' }}>{selectedPlan.name}</div>
                            <div style={{ fontSize: '.8rem', color: 'var(--text-muted)' }}>
                                {selectedPlan.cpu_cores} vCPU · {selectedPlan.ram_mb / 1024}GB RAM · {selectedPlan.disk_gb}GB SSD · {formatVND(selectedPlan.monthly_rate)}/tháng
                            </div>
                        </div>
                    </div>
                    <div className="form-group">
                        <label>Hostname</label>
                        <input className="input" placeholder="my-server-01" value={hostname}
                            onChange={e => setHostname(e.target.value.toLowerCase())} />
                        {hostname && !hostnameValid && (
                            <small style={{ color: 'var(--error)' }}>Chỉ dùng chữ thường, số, dấu gạch ngang (3–63 ký tự)</small>
                        )}
                    </div>
                    <button className="btn-primary" onClick={handleDeploy}
                        disabled={!hostname || !hostnameValid || deploying} style={{ width: '100%' }}>
                        <Rocket size={15} />
                        {deploying ? 'Đang khởi tạo...' : 'Deploy VPS'}
                    </button>
                </div>
            )}
        </div>
    )
}

// ─── Main Page ─────────────────────────────────────────────────────────────────
export default function VPSPage() {
    const [tab, setTab] = useState<'instances' | 'deploy'>('instances')
    const [page, setPage] = useState(1)
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
        try { await fn(); success(`${label} thành công`); refetch() }
        catch (e: any) { toastError(e?.response?.data?.error?.message ?? `${label} thất bại`) }
    }

    const handleTerminate = async (id: string, hostname: string) => {
        const ok = await confirm({ title: 'Xóa VPS', message: `Xóa "${hostname}"? Toàn bộ dữ liệu sẽ mất vĩnh viễn.`, confirmLabel: 'Xóa', variant: 'danger' })
        if (!ok) return
        try { await vpsAPI.terminateInstance(id); success('Đã xóa VPS'); qc.invalidateQueries({ queryKey: ['vps-instances'] }) }
        catch (e: any) { toastError(e?.response?.data?.error?.message ?? 'Xóa thất bại') }
    }

    return (
        <AppLayout breadcrumb={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'VPS Instances' }]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">VPS Instances</h1>
                    <p className="page-subtitle">Quản lý và triển khai máy chủ ảo</p>
                </div>
                <button className="topbar-icon-btn" onClick={() => refetch()} title="Refresh"><RefreshCw size={15} /></button>
            </div>

            {/* ─── Tab Navigation ─── */}
            <div style={{ display: 'flex', borderBottom: '1px solid var(--border)', marginBottom: '1.5rem', gap: '.25rem' }}>
                <TabBtn label="🖥 VPS của tôi" active={tab === 'instances'} onClick={() => setTab('instances')} count={instances.length || undefined} />
                <TabBtn label="🚀 Deploy VPS" active={tab === 'deploy'} onClick={() => setTab('deploy')} />
            </div>

            {/* ─── Tab: VPS của tôi ─── */}
            {tab === 'instances' && (
                <div className="fade-in">
                    <div className="card" style={{ padding: 0 }}>
                        {isLoading ? (
                            <div className="loading-spinner">Đang tải instances...</div>
                        ) : instances.length === 0 ? (
                            <div className="empty-state">
                                <Server size={44} opacity={0.3} />
                                <p>Chưa có VPS nào</p>
                                <button className="btn-primary" onClick={() => setTab('deploy')}>
                                    <Rocket size={14} /> Deploy VPS đầu tiên
                                </button>
                            </div>
                        ) : (
                            <>
                                <div className="table-wrapper">
                                    <table className="data-table">
                                        <thead>
                                            <tr>
                                                <th>Hostname</th>
                                                <th>Gói</th>
                                                <th>IP Address</th>
                                                <th>Trạng thái</th>
                                                <th>Thao tác</th>
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
                                                            {inst.status === 'running' && <span style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--success)', display: 'inline-block' }} className="pulse" />}
                                                            {inst.status}
                                                        </span>
                                                    </td>
                                                    <td>
                                                        <div style={{ display: 'flex', gap: '.3rem', flexWrap: 'wrap' }}>
                                                            {inst.status === 'stopped' && (
                                                                <button className="action-btn green" onClick={action(() => vpsAPI.startInstance(inst.id), 'Khởi động')}><Play size={12} /> Start</button>
                                                            )}
                                                            {inst.status === 'running' && (<>
                                                                <button className="action-btn" onClick={action(() => vpsAPI.stopInstance(inst.id), 'Dừng')}><Square size={12} /> Stop</button>
                                                                <button className="action-btn" onClick={action(() => vpsAPI.rebootInstance(inst.id), 'Khởi động lại')}><RotateCcw size={12} /> Reboot</button>
                                                            </>)}
                                                            {(inst.status === 'running' || inst.status === 'stopped') && (
                                                                <button className="action-btn" onClick={async () => {
                                                                    const res = await vpsAPI.getConsole(inst.id)
                                                                    window.open(res.data?.data?.console_url, '_blank')
                                                                }}><Terminal size={12} /> Console</button>
                                                            )}
                                                            {inst.status !== 'terminated' && (
                                                                <button className="action-btn red" onClick={() => handleTerminate(inst.id, inst.hostname)}><Trash2 size={12} /> Xóa</button>
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
                </div>
            )}

            {/* ─── Tab: Deploy VPS ─── */}
            {tab === 'deploy' && (
                <DeployTab onDeployed={() => setTab('instances')} />
            )}
        </AppLayout>
    )
}

