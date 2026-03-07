'use client'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import Cookies from 'js-cookie'
import { authAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { Loader2 } from 'lucide-react'

const schema = z.object({
    email: z.string().email('Invalid email'),
    password: z.string().min(8, 'Min 8 characters'),
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
            // Store tokens first so /me call can use them
            Cookies.set('pvp_token', access_token, { sameSite: 'strict' })
            Cookies.set('pvp_refresh', refresh_token, { sameSite: 'strict' })
            // Fetch user profile
            let email = data.email
            let fullName = ''
            try {
                const meRes = await authAPI.me()
                email = meRes.data.data?.email || email
                fullName = meRes.data.data?.full_name || ''
            } catch { }
            setUser(
                { id: user_id, email, fullName, role },
                access_token,
                refresh_token,
            )
            if (role === 'admin' || role === 'super_admin') {
                router.push('/admin')
            } else if (role === 'reseller') {
                router.push('/reseller')
            } else {
                router.push('/dashboard')
            }
        } catch (err: any) {
            setError(err.response?.data?.error?.message || err.response?.data?.message || 'Login failed')
        }
    }

    return (
        <div className="auth-page">
            <div className="auth-card">
                {/* Logo */}
                <div className="auth-logo">
                    <div className="auth-logo-icon">☁</div>
                    <h1>Cloudmini</h1>
                    <p>Sign in to your account</p>
                </div>

                <form onSubmit={handleSubmit(onSubmit)} className="auth-form">
                    <div className="form-group">
                        <label>Email</label>
                        <input
                            {...register('email')}
                            type="email"
                            className={`input ${errors.email ? 'input-error' : ''}`}
                            placeholder="you@example.com"
                            autoComplete="email"
                        />
                        {errors.email && <span className="error-msg">{errors.email.message}</span>}
                    </div>

                    <div className="form-group">
                        <label>Password</label>
                        <input
                            {...register('password')}
                            type="password"
                            className={`input ${errors.password ? 'input-error' : ''}`}
                            placeholder="••••••••"
                            autoComplete="current-password"
                        />
                        {errors.password && <span className="error-msg">{errors.password.message}</span>}
                    </div>

                    {error && <div className="alert alert-error">{error}</div>}

                    <button type="submit" className="btn-primary w-full" disabled={isSubmitting}>
                        {isSubmitting ? <Loader2 size={16} className="spin" /> : 'Sign In'}
                    </button>
                </form>

                <p className="auth-footer">
                    Don't have an account?{' '}
                    <a href="/register" className="link">Sign up</a>
                </p>
            </div>
        </div>
    )
}
