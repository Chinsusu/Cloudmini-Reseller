'use client'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
    LayoutDashboard, Globe, Server, Wallet, Settings,
    Users, Key, Webhook, LogOut, ShieldCheck, BarChart3,
    Cloud, ChevronRight
} from 'lucide-react'
import { useAuthStore } from '@/lib/store'
import { authAPI } from '@/lib/api'
import { useRouter } from 'next/navigation'
import clsx from 'clsx'

const userNav = [
    {
        group: 'MAIN',
        items: [
            { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
        ]
    },
    {
        group: 'SERVICES',
        items: [
            { href: '/dashboard/proxy', label: 'Proxy Orders', icon: Globe },
            { href: '/dashboard/vps', label: 'VPS Instances', icon: Server },
        ]
    },
    {
        group: 'ACCOUNT',
        items: [
            { href: '/dashboard/wallet', label: 'Wallet', icon: Wallet },
            { href: '/dashboard/settings', label: 'Settings', icon: Settings },
        ]
    },
]

const adminNav = [
    {
        group: 'OVERVIEW',
        items: [
            { href: '/admin', label: 'Dashboard', icon: LayoutDashboard },
        ]
    },
    {
        group: 'MANAGEMENT',
        items: [
            { href: '/admin/users', label: 'Users', icon: Users },
            { href: '/admin/resellers', label: 'Resellers', icon: ShieldCheck },
        ]
    },
]

const resellerNav = [
    {
        group: 'OVERVIEW',
        items: [
            { href: '/reseller', label: 'Dashboard', icon: BarChart3 },
        ]
    },
    {
        group: 'MANAGEMENT',
        items: [
            { href: '/reseller/accounts', label: 'Sub-Accounts', icon: Users },
            { href: '/reseller/pricing', label: 'Pricing', icon: Wallet },
        ]
    },
    {
        group: 'DEVELOPER',
        items: [
            { href: '/reseller/api-keys', label: 'API Keys', icon: Key },
            { href: '/reseller/webhooks', label: 'Webhooks', icon: Webhook },
        ]
    },
]

export function Sidebar({ mobileOpen = false, onClose }: { mobileOpen?: boolean; onClose?: () => void }) {
    const pathname = usePathname()
    const { user, clearAuth } = useAuthStore()
    const router = useRouter()

    const navGroups = user?.role === 'admin' || user?.role === 'super_admin'
        ? adminNav
        : user?.role === 'reseller'
            ? resellerNav
            : userNav

    const handleLogout = async () => {
        try { await authAPI.logout() } catch { }
        clearAuth()
        router.push('/login')
    }

    const initials = user?.fullName
        ? user.fullName.split(' ').map((n: string) => n[0]).join('').toUpperCase().slice(0, 2)
        : user?.email?.[0]?.toUpperCase() ?? 'U'

    return (
        <aside className={clsx('sidebar', mobileOpen && 'mobile-open')}>
            {/* Logo */}
            <div className="sidebar-logo">
                <div className="logo-icon">
                    <Cloud size={18} />
                </div>
                <span className="logo-text">Cloudmini</span>
            </div>

            {/* Navigation groups */}
            {navGroups.map(({ group, items }) => (
                <div key={group}>
                    <p className="nav-group-label">{group}</p>
                    <nav className="sidebar-nav">
                        {items.map(({ href, label, icon: Icon }) => {
                            const active = pathname === href || (href !== '/' && pathname?.startsWith(href + '/'))
                            return (
                                <Link
                                    key={href}
                                    href={href}
                                    className={clsx('nav-item', active && 'nav-item-active')}
                                >
                                    <Icon size={17} />
                                    <span>{label}</span>
                                    {active && <ChevronRight size={14} style={{ marginLeft: 'auto', opacity: .7 }} />}
                                </Link>
                            )
                        })}
                    </nav>
                </div>
            ))}

            {/* User + Logout at bottom — single row */}
            <div className="sidebar-bottom">
                {user && (
                    <div className="sidebar-user" style={{ justifyContent: 'space-between' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '.75rem' }}>
                            <div className="user-avatar">{initials}</div>
                            <div>
                                <p className="user-name">{user.fullName || user.email}</p>
                                <span className="user-role">{user.role}</span>
                            </div>
                        </div>
                        <button
                            onClick={handleLogout}
                            title="Log out"
                            style={{
                                background: 'none', border: 'none', cursor: 'pointer',
                                color: 'var(--text-muted)', padding: '.35rem',
                                borderRadius: 'var(--radius)', display: 'flex',
                                alignItems: 'center', transition: 'color .15s',
                                flexShrink: 0
                            }}
                            onMouseEnter={e => (e.currentTarget.style.color = 'var(--error)')}
                            onMouseLeave={e => (e.currentTarget.style.color = 'var(--text-muted)')}
                        >
                            <LogOut size={17} />
                        </button>
                    </div>
                )}
            </div>
        </aside>
    )
}
