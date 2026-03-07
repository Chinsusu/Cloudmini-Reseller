'use client'
import { useState } from 'react'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'

interface AppLayoutProps {
    children: React.ReactNode
    breadcrumb?: { label: string; href?: string }[]
}

export function AppLayout({ children, breadcrumb }: AppLayoutProps) {
    const [mobileOpen, setMobileOpen] = useState(false)

    return (
        <div className="page-layout">
            {/* Mobile overlay */}
            <div
                className={`sidebar-overlay ${mobileOpen ? 'visible' : ''}`}
                onClick={() => setMobileOpen(false)}
            />
            {/* Sidebar  */}
            <Sidebar mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />

            {/* Main content */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                <Topbar
                    breadcrumb={breadcrumb}
                    onMobileMenuToggle={() => setMobileOpen(v => !v)}
                    mobileMenuOpen={mobileOpen}
                />
                <main className="page-main">
                    {children}
                </main>
            </div>
        </div>
    )
}
