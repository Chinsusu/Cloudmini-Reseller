'use client'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import Cookies from 'js-cookie'
import { authAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { Loader2, Mail, Lock } from 'lucide-react'

const schema = z.object({
    email: z.string().email('Invalid email'),
    password: z.string().min(6, 'Min 6 characters'),
})
type FormData = z.infer<typeof schema>

export default function LoginPage() {
    const router = useRouter()
    const { setUser } = useAuthStore()
    const [error, setError] = useState('')
    const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
        resolver: zodResolver(schema),
    })

    const onSubmit = async (data: FormData) => {
        setError('')
        try {
            const res = await authAPI.login(data.email, data.password)
            const { access_token, refresh_token, role, user_id } = res.data.data
            Cookies.set('pvp_token', access_token, { sameSite: 'lax', expires: 1 })
            Cookies.set('pvp_refresh', refresh_token, { sameSite: 'lax', expires: 30 })
            let email = data.email
            let fullName = ''
            try {
                const meRes = await authAPI.me()
                email = meRes.data.data?.email || email
                fullName = meRes.data.data?.full_name || ''
            } catch { }
            setUser({ id: user_id, email, fullName, role }, access_token, refresh_token)
            if (role === 'admin' || role === 'super_admin') router.push('/admin')
            else if (role === 'reseller') router.push('/reseller')
            else router.push('/dashboard')
        } catch (err: any) {
            setError(err.response?.data?.error?.message || err.response?.data?.message || 'Login failed')
        }
    }

    return (
        <div className="dc-auth-page">
            <div className="dc-card">
                {/* White top section */}
                <div className="dc-card-top">
                    <h1 className="dc-title">LOGIN</h1>

                    {error && <div className="dc-error">{error}</div>}

                    <form onSubmit={handleSubmit(onSubmit)}>
                        <div className="dc-field">
                            <Mail size={16} className="dc-field-icon" />
                            <input
                                {...register('email')}
                                type="email"
                                placeholder="Email or login"
                                autoComplete="email"
                                className={`dc-input ${errors.email ? 'dc-input-err' : ''}`}
                            />
                        </div>
                        {errors.email && <p className="dc-field-error">{errors.email.message}</p>}

                        <div className="dc-field" style={{ marginTop: '.75rem' }}>
                            <Lock size={16} className="dc-field-icon" />
                            <input
                                {...register('password')}
                                type="password"
                                placeholder="Password"
                                autoComplete="current-password"
                                className={`dc-input ${errors.password ? 'dc-input-err' : ''}`}
                            />
                        </div>
                        {errors.password && <p className="dc-field-error">{errors.password.message}</p>}

                        <button type="submit" className="dc-btn-primary" disabled={isSubmitting}>
                            {isSubmitting ? <Loader2 size={17} className="spin" /> : 'LOGIN'}
                        </button>
                    </form>
                </div>

                {/* Dark bottom section */}
                <div className="dc-card-bottom">
                    <a href="/forgot-password" className="dc-link-forgot">Forgot password?</a>
                    <div className="dc-divider" />
                    <p className="dc-sub-text">Do you not have an account?</p>
                    <a href="/register" className="dc-btn-outline">REGISTER NOW</a>
                </div>
            </div>
        </div>
    )
}
