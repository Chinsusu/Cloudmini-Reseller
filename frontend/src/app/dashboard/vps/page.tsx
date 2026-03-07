'use client'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { vpsAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { Server, Play, Square, RefreshCw, Trash2, Terminal, Plus } from 'lucide-react'

const statusColor: Record<string, string> = {
    running: '#28C76F', pending: '#FF9F43', provisioning: '#7367F0',
    booting: '#00CFE8', suspended: '#EA5455', terminated: '#A8AAAE', failed: '#EA5455',
}

export default function VPSPage() {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const { data, isLoading } = useQuery({
        queryKey: ['vps-list'],
        queryFn: () => vpsAPI.listInstances(),
        refetchInterval: 10_000,
    })
    const instances = data?.data?.data ?? []

    const action = useMutation({
        mutationFn: ({ id, op }: { id: string; op: string }) => {
            const ops: Record<string, (id: string) => Promise<any>> = {
                start: vpsAPI.startVPS, stop: vpsAPI.stopVPS,
                reboot: vpsAPI.rebootVPS, delete: vpsAPI.deleteVPS,
            }
            return ops[op](id)
        },
        onSuccess: (_data, variables) => {
            qc.invalidateQueries({ queryKey: ['vps-list'] })
            success(`VPS ${variables.op} action sent`)
        },
        onError: () => toastError('Action failed, please try again'),
    })

    const handleDelete = async (id: string, hostname: string) => {
        const ok = await confirm({
            title: 'Terminate VPS',
            message: `Permanently terminate "${hostname}"? This cannot be undone.`,
            confirmLabel: 'Terminate',
            variant: 'danger',
        })
        if (ok) action.mutate({ id, op: 'delete' })
    }

    return (
        <AppLayout breadcrumb={[
            { label: 'Dashboard', href: '/dashboard' },
            { label: 'VPS Instances' },
        ]}>
            {confirmDialog}

            <div className="page-header">
                <div>
                    <h1 className="page-title">VPS Instances</h1>
                    <p className="page-subtitle">{instances.length} instance{instances.length !== 1 ? 's' : ''}</p>
                </div>
                <button className="btn-primary">
                    <Plus size={16} /> New Instance
                </button>
            </div>

            {isLoading ? (
                <div className="loading-spinner">Loading...</div>
            ) : instances.length === 0 ? (
                <div className="empty-state">
                    <Server size={48} opacity={0.3} />
                    <p>No VPS instances yet</p>
                    <button className="btn-primary">Create your first VPS</button>
                </div>
            ) : (
                <div className="vps-grid">
                    {instances.map((inst: any) => (
                        <div key={inst.id} className="vps-card">
                            <div className="vps-card-header">
                                <div className="vps-name">
                                    <span className="status-dot" style={{ background: statusColor[inst.status] }} />
                                    <span>{inst.hostname}</span>
                                </div>
                                <span className={`badge badge-${inst.status}`}>{inst.status}</span>
                            </div>

                            <div className="vps-details">
                                <div className="vps-detail">
                                    <span className="label">IP</span>
                                    <code className="font-mono">{inst.ip_address || 'Pending...'}</code>
                                </div>
                                <div className="vps-detail">
                                    <span className="label">Node</span>
                                    <span>{inst.node_name || '—'}</span>
                                </div>
                                <div className="vps-detail">
                                    <span className="label">Plan</span>
                                    <span>{inst.plan_name || '—'}</span>
                                </div>
                            </div>

                            <div className="vps-actions">
                                <button
                                    className="action-btn green" title="Start"
                                    onClick={() => action.mutate({ id: inst.id, op: 'start' })}
                                    disabled={inst.status === 'running' || action.isPending}
                                ><Play size={14} /></button>
                                <button
                                    className="action-btn red" title="Stop"
                                    onClick={() => action.mutate({ id: inst.id, op: 'stop' })}
                                    disabled={inst.status !== 'running' || action.isPending}
                                ><Square size={14} /></button>
                                <button
                                    className="action-btn blue" title="Reboot"
                                    onClick={() => action.mutate({ id: inst.id, op: 'reboot' })}
                                    disabled={inst.status !== 'running' || action.isPending}
                                ><RefreshCw size={14} /></button>
                                <button
                                    className="action-btn yellow" title="Console"
                                    onClick={() => window.open(`/dashboard/vps/${inst.id}/console`, '_blank')}
                                ><Terminal size={14} /></button>
                                <button
                                    className="action-btn gray" title="Terminate"
                                    onClick={() => handleDelete(inst.id, inst.hostname)}
                                    disabled={inst.status === 'terminated' || action.isPending}
                                ><Trash2 size={14} /></button>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </AppLayout>
    )
}
