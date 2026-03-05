'use client'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
    LayoutDashboard, Globe, Server, Wallet, Settings,
    Users, Key, Webhook, LogOut, ShieldCheck, ChevronRight
} from 'lucide-react'
import { useAuthStore } from '@/lib/store'
import { authAPI } from '@/lib/api'
import { useRouter } from 'next/navigation'
import clsx from 'clsx'

const userNav = [
    { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/dashboard/proxy', label: 'Proxy Orders', icon: Globe },
    { href: '/dashboard/vps', label: 'VPS Instances', icon: Server },
    { href: '/dashboard/wallet', label: 'Wallet', icon: Wallet },
    { href: '/dashboard/settings', label: 'Settings', icon: Settings },
]

const adminNav = [
    { href: '/admin', label: 'Overview', icon: LayoutDashboard },
    { href: '/admin/users', label: 'Users', icon: Users },
    { href: '/admin/resellers', label: 'Resellers', icon: ShieldCheck },
]

const resellerNav = [
    { href: '/reseller', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/reseller/accounts', label: 'Sub-Accounts', icon: Users },
    { href: '/reseller/pricing', label: 'Pricing', icon: Wallet },
    { href: '/reseller/api-keys', label: 'API Keys', icon: Key },
    { href: '/reseller/webhooks', label: 'Webhooks', icon: Webhook },
]

export function Sidebar() {
    const pathname = usePathname()
    const { user, clearAuth } = useAuthStore()
    const router = useRouter()

    const nav = user?.role === 'admin' || user?.role === 'super_admin'
        ? adminNav
        : user?.role === 'reseller'
            ? resellerNav
            : userNav

    const handleLogout = async () => {
        try { await authAPI.logout() } catch { }
        clearAuth()
        router.push('/login')
    }

    return (
        <aside className="sidebar">
            {/* Logo */}
            <div className="sidebar-logo">
                <div className="logo-icon">☁</div>
                <span className="logo-text">Cloudmini</span>
            </div>

            {/* User info */}
            {user && (
                <div className="sidebar-user">
                    <div className="user-avatar">{user.email[0].toUpperCase()}</div>
                    <div className="user-info">
                        <p className="user-name">{user.fullName || user.email}</p>
                        <span className="user-role">{user.role}</span>
                    </div>
                </div>
            )}

            {/* Navigation */}
            <nav className="sidebar-nav">
                {nav.map(({ href, label, icon: Icon }) => (
                    <Link
                        key={href}
                        href={href}
                        className={clsx('nav-item', pathname === href && 'nav-item-active')}
                    >
                        <Icon size={18} />
                        <span>{label}</span>
                        {pathname === href && <ChevronRight size={14} className="ml-auto" />}
                    </Link>
                ))}
            </nav>

            {/* Logout */}
            <button onClick={handleLogout} className="sidebar-logout">
                <LogOut size={18} />
                <span>Log Out</span>
            </button>
        </aside>
    )
}
