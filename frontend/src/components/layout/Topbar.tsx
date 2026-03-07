'use client'
import { useState } from 'react'
import { Search, Bell, Menu, X } from 'lucide-react'
import { useAuthStore } from '@/lib/store'

interface TopbarProps {
    title?: string
    breadcrumb?: { label: string; href?: string }[]
    onMobileMenuToggle?: () => void
    mobileMenuOpen?: boolean
}

export function Topbar({ title, breadcrumb, onMobileMenuToggle, mobileMenuOpen }: TopbarProps) {
    const { user } = useAuthStore()
    const [searchFocused, setSearchFocused] = useState(false)

    const initials = user?.fullName
        ? user.fullName.split(' ').map((n: string) => n[0]).join('').toUpperCase().slice(0, 2)
        : user?.email?.[0]?.toUpperCase() ?? 'U'

    return (
        <header className="topbar">
            {/* LEFT: Mobile hamburger + breadcrumb */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                <button
                    className="mobile-menu-btn"
                    onClick={onMobileMenuToggle}
                    aria-label="Toggle sidebar"
                >
                    {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
                </button>

                {breadcrumb && breadcrumb.length > 0 && (
                    <nav className="breadcrumb topbar-breadcrumb">
                        {breadcrumb.map((item, i) => (
                            <span key={i} style={{ display: 'flex', alignItems: 'center', gap: '.4rem' }}>
                                {i > 0 && <span className="breadcrumb-sep">/</span>}
                                {item.href
                                    ? <a href={item.href} className="breadcrumb-link">{item.label}</a>
                                    : <span className="breadcrumb-current">{item.label}</span>
                                }
                            </span>
                        ))}
                    </nav>
                )}
            </div>

            {/* RIGHT: Search + Notification + Avatar */}
            <div className="topbar-actions">
                {/* Search */}
                <div className={`topbar-search-wrap ${searchFocused ? 'focused' : ''}`}>
                    <Search size={15} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
                    <input
                        className="topbar-search-input"
                        placeholder="Search... (Ctrl+K)"
                        onFocus={() => setSearchFocused(true)}
                        onBlur={() => setSearchFocused(false)}
                    />
                </div>

                {/* Notification bell */}
                <button className="topbar-icon-btn" title="Notifications" style={{ position: 'relative' }}>
                    <Bell size={18} />
                    <span className="notif-dot" />
                </button>

                {/* User avatar */}
                <div className="topbar-avatar" title={user?.email}>
                    {initials}
                </div>
            </div>
        </header>
    )
}
