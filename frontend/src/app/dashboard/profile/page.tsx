'use client'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { authAPI } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { useToast } from '@/components/ui/Toast'
import { useConfirm } from '@/components/ui/ConfirmDialog'
import { User, Lock, ShieldCheck, ShieldOff, Eye, EyeOff, X, Copy, Check } from 'lucide-react'

// ─── 2FA Setup Modal ─────────────────────────────────────────────────────────
function TwoFASetupModal({
    secret, otpauthUrl, onClose, onSuccess,
}: {
    secret: string; otpauthUrl: string; onClose: () => void; onSuccess: () => void
}) {
    const { success, error: toastError } = useToast()
    const [code, setCode] = useState('')
    const [copied, setCopied] = useState(false)

    const enableMut = useMutation({
        mutationFn: () => authAPI.enable2FA(code),
        onSuccess: () => { success('2FA enabled!'); onSuccess() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Invalid code'),
    })

    const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(otpauthUrl)}`

    const handleCopy = () => {
        navigator.clipboard.writeText(secret)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 440 }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem' }}>
                        <ShieldCheck size={16} /> Set Up 2FA
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                    <p style={{ fontSize: '.85rem', color: 'var(--text-muted)' }}>
                        Scan the QR code with your authenticator app (Google Authenticator, Authy, etc.)
                    </p>

                    {/* QR Code */}
                    <div style={{ display: 'flex', justifyContent: 'center' }}>
                        <img src={qrUrl} alt="TOTP QR Code" width={200} height={200}
                            style={{ border: '4px solid var(--border-light)', borderRadius: 12 }} />
                    </div>

                    {/* Manual secret */}
                    <div>
                        <p style={{ fontSize: '.78rem', color: 'var(--text-muted)', marginBottom: '.35rem' }}>
                            Or enter this secret manually:
                        </p>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '.5rem', background: 'var(--bg)', borderRadius: 8, padding: '.5rem .75rem', border: '1px solid var(--border-light)' }}>
                            <code style={{ flex: 1, fontSize: '.78rem', fontFamily: 'monospace', wordBreak: 'break-all' }}>{secret}</code>
                            <button style={{ background: 'none', border: 'none', cursor: 'pointer', color: copied ? 'var(--success)' : 'var(--text-muted)', padding: '0 .25rem' }} onClick={handleCopy}>
                                {copied ? <Check size={14} /> : <Copy size={14} />}
                            </button>
                        </div>
                    </div>

                    {/* Verify */}
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>Enter 6-digit code to verify</label>
                        <input className="input" placeholder="000000" maxLength={6} value={code}
                            onChange={e => setCode(e.target.value.replace(/\D/g, ''))}
                            onKeyDown={e => e.key === 'Enter' && code.length === 6 && enableMut.mutate()} />
                    </div>

                    <button className="btn-primary" disabled={code.length !== 6 || enableMut.isPending}
                        onClick={() => enableMut.mutate()}>
                        <ShieldCheck size={15} />
                        {enableMut.isPending ? 'Verifying...' : 'Enable 2FA'}
                    </button>
                </div>
            </div>
        </div>
    )
}

// ─── Disable 2FA Modal ────────────────────────────────────────────────────────
function DisableTwoFAModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) {
    const { success, error: toastError } = useToast()
    const [code, setCode] = useState('')

    const disableMut = useMutation({
        mutationFn: () => authAPI.disable2FA(code),
        onSuccess: () => { success('2FA disabled'); onSuccess() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Invalid code'),
    })

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 400 }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem' }}>
                        <ShieldOff size={16} /> Disable 2FA
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                    <p style={{ fontSize: '.85rem', color: 'var(--text-muted)' }}>
                        Enter your current authenticator code to confirm disabling 2FA.
                    </p>
                    <div className="form-group" style={{ marginBottom: 0 }}>
                        <label>6-digit authenticator code</label>
                        <input className="input" placeholder="000000" maxLength={6} value={code}
                            onChange={e => setCode(e.target.value.replace(/\D/g, ''))} />
                    </div>
                    <button className="btn-primary" style={{ background: 'var(--error)' }}
                        disabled={code.length !== 6 || disableMut.isPending}
                        onClick={() => disableMut.mutate()}>
                        <ShieldOff size={15} />
                        {disableMut.isPending ? 'Disabling...' : 'Disable 2FA'}
                    </button>
                </div>
            </div>
        </div>
    )
}

// ─── Change Password Modal ────────────────────────────────────────────────────
function ChangePasswordModal({ onClose }: { onClose: () => void }) {
    const { success, error: toastError } = useToast()
    const [form, setForm] = useState({ old_password: '', new_password: '', confirm: '' })
    const [show, setShow] = useState(false)

    const changeMut = useMutation({
        mutationFn: () => authAPI.changePassword({ old_password: form.old_password, new_password: form.new_password }),
        onSuccess: () => { success('Password changed'); onClose() },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed'),
    })

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" style={{ maxWidth: 420 }} onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '.5rem' }}>
                        <Lock size={16} /> Change Password
                    </span>
                    <button className="modal-close" onClick={onClose}><X size={18} /></button>
                </div>
                <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: '.75rem' }}>
                    {(['old_password', 'new_password', 'confirm'] as const).map(field => (
                        <div className="form-group" key={field} style={{ marginBottom: 0, position: 'relative' }}>
                            <label>{field === 'old_password' ? 'Current Password' : field === 'new_password' ? 'New Password' : 'Confirm New Password'}</label>
                            <input className="input" type={show ? 'text' : 'password'}
                                value={form[field]} onChange={e => setForm(f => ({ ...f, [field]: e.target.value }))} />
                        </div>
                    ))}
                    <button style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '.78rem', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '.35rem', alignSelf: 'flex-start' }}
                        onClick={() => setShow(s => !s)}>
                        {show ? <EyeOff size={12} /> : <Eye size={12} />} {show ? 'Hide' : 'Show'} passwords
                    </button>
                    {form.new_password && form.confirm && form.new_password !== form.confirm && (
                        <p style={{ fontSize: '.78rem', color: 'var(--error)' }}>Passwords do not match</p>
                    )}
                    <button className="btn-primary"
                        disabled={!form.old_password || !form.new_password || form.new_password !== form.confirm || changeMut.isPending}
                        onClick={() => changeMut.mutate()}>
                        {changeMut.isPending ? 'Saving...' : 'Change Password'}
                    </button>
                </div>
            </div>
        </div>
    )
}

// ─── Page ─────────────────────────────────────────────────────────────────────
export default function ProfilePage() {
    const qc = useQueryClient()
    const { success, error: toastError } = useToast()
    const { confirm, dialog: confirmDialog } = useConfirm()

    const [showSetup, setShowSetup] = useState(false)
    const [setupData, setSetupData] = useState<{ secret: string; otpauth_url: string } | null>(null)
    const [showDisable, setShowDisable] = useState(false)
    const [showChangePwd, setShowChangePwd] = useState(false)
    const [editMode, setEditMode] = useState(false)
    const [form, setForm] = useState({ full_name: '', phone: '' })

    const { data, isLoading } = useQuery({
        queryKey: ['me'],
        queryFn: () => authAPI.me(),
        select: (d) => (d.data as any)?.data ?? d.data,
    })

    const user = data

    const updateMut = useMutation({
        mutationFn: () => authAPI.updateMe(form),
        onSuccess: () => { qc.invalidateQueries({ queryKey: ['me'] }); success('Profile updated'); setEditMode(false) },
        onError: (e: any) => toastError(e?.response?.data?.error?.message ?? 'Failed'),
    })

    const handleStartEdit = () => {
        setForm({ full_name: user?.full_name ?? '', phone: user?.phone ?? '' })
        setEditMode(true)
    }

    const handleSetup2FA = async () => {
        try {
            const res = await authAPI.setup2FA()
            setSetupData(res.data)
            setShowSetup(true)
        } catch (e: any) { toastError(e?.response?.data?.error?.message ?? 'Failed') }
    }

    if (isLoading) return <AppLayout><div className="loading-spinner">Loading...</div></AppLayout>

    const twoFAEnabled = user?.totp_enabled ?? false

    return (
        <AppLayout breadcrumb={[{ label: 'Profile' }]}>
            {confirmDialog}
            {showSetup && setupData && (
                <TwoFASetupModal
                    secret={setupData.secret}
                    otpauthUrl={setupData.otpauth_url}
                    onClose={() => setShowSetup(false)}
                    onSuccess={() => { setShowSetup(false); qc.invalidateQueries({ queryKey: ['me'] }) }}
                />
            )}
            {showDisable && (
                <DisableTwoFAModal
                    onClose={() => setShowDisable(false)}
                    onSuccess={() => { setShowDisable(false); qc.invalidateQueries({ queryKey: ['me'] }) }}
                />
            )}
            {showChangePwd && <ChangePasswordModal onClose={() => setShowChangePwd(false)} />}

            <div className="page-header">
                <h1 className="page-title">Profile</h1>
                <p className="page-subtitle">Manage your account information and security</p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
                {/* Profile Info Card */}
                <div className="card">
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.25rem' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem', fontWeight: 700 }}>
                            <User size={17} /> Account Info
                        </div>
                        {!editMode && (
                            <button className="action-btn" onClick={handleStartEdit}>Edit</button>
                        )}
                    </div>

                    {/* Avatar */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1.5rem' }}>
                        <div style={{
                            width: 52, height: 52, borderRadius: '50%',
                            background: 'linear-gradient(135deg, rgba(230,168,23,.25), rgba(230,168,23,.1))',
                            border: '2px solid rgba(230,168,23,.35)',
                            display: 'grid', placeItems: 'center',
                            fontWeight: 800, fontSize: '1.2rem', color: 'var(--dc-gold)',
                        }}>
                            {user?.email?.[0]?.toUpperCase() ?? '?'}
                        </div>
                        <div>
                            <p style={{ fontWeight: 700, color: 'var(--text-heading)', fontSize: '1rem' }}>{user?.full_name || '—'}</p>
                            <p style={{ fontSize: '.82rem', color: 'var(--text-muted)', marginTop: '.1rem' }}>{user?.email}</p>
                            <span className="badge badge-secondary" style={{ marginTop: '.35rem' }}>{user?.role}</span>
                        </div>
                    </div>

                    {editMode ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '.75rem' }}>
                            <div className="form-group" style={{ marginBottom: 0 }}>
                                <label>Full Name</label>
                                <input className="input" value={form.full_name}
                                    onChange={e => setForm(f => ({ ...f, full_name: e.target.value }))} />
                            </div>
                            <div className="form-group" style={{ marginBottom: 0 }}>
                                <label>Phone</label>
                                <input className="input" value={form.phone}
                                    onChange={e => setForm(f => ({ ...f, phone: e.target.value }))} placeholder="+84..." />
                            </div>
                            <div style={{ display: 'flex', gap: '.5rem' }}>
                                <button className="btn-primary" onClick={() => updateMut.mutate()} disabled={updateMut.isPending}>
                                    {updateMut.isPending ? 'Saving...' : 'Save'}
                                </button>
                                <button className="action-btn" onClick={() => setEditMode(false)}>Cancel</button>
                            </div>
                        </div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
                            {[
                                ['Email', user?.email],
                                ['Full Name', user?.full_name || '—'],
                                ['Phone', user?.phone || '—'],
                                ['Status', user?.status],
                            ].map(([label, value]) => (
                                <div key={label} style={{ display: 'flex', justifyContent: 'space-between', fontSize: '.875rem', borderBottom: '1px solid var(--border-light)', padding: '.625rem 0' }}>
                                    <span style={{ color: 'var(--text-muted)', fontWeight: 500 }}>{label}</span>
                                    <span style={{ fontWeight: 600, color: 'var(--text-heading)' }}>{value}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Security Card */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
                    {/* 2FA Card */}
                    <div className="card">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem', fontWeight: 700, marginBottom: '1rem' }}>
                            <ShieldCheck size={17} /> Two-Factor Authentication
                        </div>
                        <div style={{
                            display: 'flex', alignItems: 'center', gap: '1rem', padding: '.875rem',
                            background: twoFAEnabled ? 'rgba(40,199,111,.06)' : 'rgba(255,255,255,.03)',
                            borderRadius: 10,
                            border: `1px solid ${twoFAEnabled ? 'rgba(40,199,111,.2)' : 'var(--border)'}`,
                            marginBottom: '1rem',
                        }}>
                            <div style={{ width: 40, height: 40, borderRadius: '50%', background: twoFAEnabled ? 'rgba(34,197,94,.15)' : 'var(--border-light)', display: 'grid', placeItems: 'center' }}>
                                {twoFAEnabled ? <ShieldCheck size={18} style={{ color: 'var(--success)' }} /> : <ShieldOff size={18} style={{ color: 'var(--text-muted)' }} />}
                            </div>
                            <div>
                                <p style={{ fontWeight: 600, color: twoFAEnabled ? 'var(--success)' : 'var(--text-heading)', fontSize: '.875rem' }}>
                                    {twoFAEnabled ? '2FA is Enabled' : '2FA is Disabled'}
                                </p>
                                <p style={{ fontSize: '.78rem', color: 'var(--text-muted)' }}>
                                    {twoFAEnabled
                                        ? 'Your account is protected with TOTP authentication'
                                        : 'Enable 2FA to add an extra layer of security'}
                                </p>
                            </div>
                        </div>

                        {twoFAEnabled ? (
                            <button className="action-btn red" style={{ width: '100%', justifyContent: 'center', padding: '.6rem' }}
                                onClick={() => setShowDisable(true)}>
                                <ShieldOff size={15} /> Disable 2FA
                            </button>
                        ) : (
                            <button className="btn-primary" style={{ width: '100%' }}
                                onClick={handleSetup2FA}>
                                <ShieldCheck size={15} /> Enable 2FA
                            </button>
                        )}
                    </div>

                    {/* Change Password Card */}
                    <div className="card">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '.6rem', fontWeight: 700, marginBottom: '1rem' }}>
                            <Lock size={17} /> Password
                        </div>
                        <p style={{ fontSize: '.85rem', color: 'var(--text-muted)', marginBottom: '1rem' }}>
                            It&apos;s a good idea to use a strong password that you don&apos;t use elsewhere.
                        </p>
                        <button className="action-btn" style={{ width: '100%', justifyContent: 'center', padding: '.6rem', color: 'var(--dc-gold)', border: '1px solid rgba(230,168,23,.3)', background: 'rgba(230,168,23,.06)' }}
                            onClick={() => setShowChangePwd(true)}>
                            <Lock size={14} /> Change Password
                        </button>
                    </div>
                </div>
            </div>
        </AppLayout>
    )
}
