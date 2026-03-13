'use client'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import Cookies from 'js-cookie'

interface User {
    id: string
    email: string
    fullName: string
    role: 'user' | 'reseller' | 'admin' | 'super_admin'
}

interface AuthState {
    user: User | null
    isAuthenticated: boolean
    setUser: (user: User, token: string, refresh: string) => void
    clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
    persist(
        (set) => ({
            user: null,
            isAuthenticated: false,
            setUser: (user, token, refresh) => {
                Cookies.set('pvp_token', token, { sameSite: 'lax', expires: 1 })
                Cookies.set('pvp_refresh', refresh, { sameSite: 'lax', expires: 30 })
                set({ user, isAuthenticated: true })
            },
            clearAuth: () => {
                Cookies.remove('pvp_token')
                Cookies.remove('pvp_refresh')
                set({ user: null, isAuthenticated: false })
            },
        }),
        { name: 'pvp-auth', partialize: (s) => ({ user: s.user, isAuthenticated: s.isAuthenticated }) }
    )
)
