'use client'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { authAPI } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import Cookies from 'js-cookie'
import { Loader2, Mail, Lock, User } from 'lucide-react'

const schema = z.object({
    fullName: z.string().min(2, 'Min 2 characters'),
    email: z.string().email('Invalid email'),
    password: z.string().min(8, 'Min 8 characters'),
    confirm: z.string(),
}).refine(d => d.password === d.confirm, { message: 'Passwords do not match', path: ['confirm'] })
type FormData = z.infer<typeof schema>

export default function RegisterPage() {
    const router = useRouter()
    const { setUser } = useAuthStore()
    const [error, setError] = useState('')
    const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
        resolver: zodResolver(schema),
    })

    const onSubmit = async (data: FormData) => {
        setError('')
        try {
            const res = await authAPI.register(data.email, data.password, data.fullName)
            const { access_token, refresh_token, role, user_id } = res.data.data
            Cookies.set('pvp_token', access_token, { sameSite: 'lax', expires: 1 })
            Cookies.set('pvp_refresh', refresh_token, { sameSite: 'lax', expires: 30 })
            setUser({ id: user_id, email: data.email, fullName: data.fullName, role: role || 'user' }, access_token, refresh_token)
            router.push('/dashboard')
        } catch (err: any) {
            setError(err.response?.data?.error?.message || err.response?.data?.message || 'Registration failed')
        }
    }

    return (
        <div className="dc-auth-page">
            <div className="dc-card">
                <div className="dc-card-top">
                    <h1 className="dc-title">REGISTER</h1>

                    {error && <div className="dc-error">{error}</div>}

                    <form onSubmit={handleSubmit(onSubmit)}>
                        <div className="dc-field">
                            <User size={16} className="dc-field-icon" />
                            <input {...register('fullName')} type="text" placeholder="Full name" autoComplete="name"
                                className={`dc-input ${errors.fullName ? 'dc-input-err' : ''}`} />
                        </div>
                        {errors.fullName && <p className="dc-field-error">{errors.fullName.message}</p>}

                        <div className="dc-field" style={{ marginTop: '.75rem' }}>
                            <Mail size={16} className="dc-field-icon" />
                            <input {...register('email')} type="email" placeholder="Email" autoComplete="email"
                                className={`dc-input ${errors.email ? 'dc-input-err' : ''}`} />
                        </div>
                        {errors.email && <p className="dc-field-error">{errors.email.message}</p>}

                        <div className="dc-field" style={{ marginTop: '.75rem' }}>
                            <Lock size={16} className="dc-field-icon" />
                            <input {...register('password')} type="password" placeholder="Password" autoComplete="new-password"
                                className={`dc-input ${errors.password ? 'dc-input-err' : ''}`} />
                        </div>
                        {errors.password && <p className="dc-field-error">{errors.password.message}</p>}

                        <div className="dc-field" style={{ marginTop: '.75rem' }}>
                            <Lock size={16} className="dc-field-icon" />
                            <input {...register('confirm')} type="password" placeholder="Confirm password" autoComplete="new-password"
                                className={`dc-input ${errors.confirm ? 'dc-input-err' : ''}`} />
                        </div>
                        {errors.confirm && <p className="dc-field-error">{errors.confirm.message}</p>}

                        <button type="submit" className="dc-btn-primary" disabled={isSubmitting}>
                            {isSubmitting ? <Loader2 size={17} className="spin" /> : 'CREATE ACCOUNT'}
                        </button>
                    </form>
                </div>

                <div className="dc-card-bottom">
                    <p className="dc-sub-text">Already have an account?</p>
                    <a href="/login" className="dc-btn-outline">SIGN IN</a>
                </div>
            </div>
        </div>
    )
}
