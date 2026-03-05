'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { vpsAPI } from '@/lib/api'
import { Sidebar } from '@/components/layout/Sidebar'
import { Server, Play, Square, RefreshCw, Trash2, Terminal, Plus } from 'lucide-react'

const statusColor: Record<string, string> = {
    running: '#10b981', pending: '#f59e0b', provisioning: '#6366f1',
    suspended: '#ef4444', terminated: '#6b7280', failed: '#ef4444',
}

export default function VPSPage() {
    const qc = useQueryClient()
    const { data, isLoading } = useQuery({
        queryKey: ['vps-list'],
        queryFn: () => vpsAPI.listInstances(),
        refetchInterval: 10_000, // poll every 10s for status updates
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
        onSuccess: () => qc.invalidateQueries({ queryKey: ['vps-list'] }),
    })

    return (
        <div className="page-layout">
            <Sidebar />
            <main className="page-main">
                <div className="page-header">
                    <h1 className="page-title">VPS Instances</h1>
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
                                        <code>{inst.ip_address || 'Pending...'}</code>
                                    </div>
                                    <div className="vps-detail">
                                        <span className="label">Node</span>
                                        <span>{inst.node_name}</span>
                                    </div>
                                </div>

                                <div className="vps-actions">
                                    <button
                                        className="action-btn green"
                                        onClick={() => action.mutate({ id: inst.id, op: 'start' })}
                                        disabled={inst.status === 'running'}
                                        title="Start"
                                    >
                                        <Play size={14} />
                                    </button>
                                    <button
                                        className="action-btn red"
                                        onClick={() => action.mutate({ id: inst.id, op: 'stop' })}
                                        disabled={inst.status !== 'running'}
                                        title="Stop"
                                    >
                                        <Square size={14} />
                                    </button>
                                    <button
                                        className="action-btn blue"
                                        onClick={() => action.mutate({ id: inst.id, op: 'reboot' })}
                                        disabled={inst.status !== 'running'}
                                        title="Reboot"
                                    >
                                        <RefreshCw size={14} />
                                    </button>
                                    <button
                                        className="action-btn yellow"
                                        onClick={() => window.open(`/dashboard/vps/${inst.id}/console`, '_blank')}
                                        title="Console"
                                    >
                                        <Terminal size={14} />
                                    </button>
                                    <button
                                        className="action-btn gray"
                                        onClick={() => { if (confirm('Terminate this VPS?')) action.mutate({ id: inst.id, op: 'delete' }) }}
                                        title="Terminate"
                                    >
                                        <Trash2 size={14} />
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </main>
        </div>
    )
}
