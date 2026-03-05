import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from '@/components/layout/Providers'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
    title: 'Cloudmini — Reseller Platform',
    description: 'ProxyVPS Reseller Platform — manage proxy & VPS services',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
    return (
        <html lang="en" className="dark">
            <body className={inter.className}>
                <Providers>{children}</Providers>
            </body>
        </html>
    )
}
