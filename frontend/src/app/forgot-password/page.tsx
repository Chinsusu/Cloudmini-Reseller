'use client'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { authAPI } from '@/lib/api'
import { Loader2, Mail, ArrowLeft, CheckCircle2 } from 'lucide-react'

const schema = z.object({ email: z.string().email('Invalid email') })
type FormData = z.infer<typeof schema>

export default function ForgotPasswordPage() {
    const [sent, setSent] = useState(false)
    const [error, setError] = useState('')
    const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
        resolver: zodResolver(schema),
    })

    const onSubmit = async (data: FormData) => {
        setError('')
        try {
            await authAPI.forgotPassword(data.email)
            setSent(true)
        } catch (err: any) {
            setError(err.response?.data?.error?.message || 'Request failed. Please try again.')
        }
    }

    return (
        <div className="dc-auth-page">
            <div className="dc-card">
                <div className="dc-card-top">
                    {sent ? (
                        <div style={{ textAlign: 'center', padding: '.5rem 0 1rem' }}>
                            <CheckCircle2 size={40} color="var(--dc-gold)" style={{ marginBottom: '1rem' }} />
                            <h1 className="dc-title" style={{ marginBottom: '.5rem' }}>CHECK YOUR EMAIL</h1>
                            <p style={{ color: 'var(--dc-text-muted)', fontSize: '.875rem', lineHeight: 1.6 }}>
                                We sent a password reset link to your email address.
                            </p>
                        </div>
                    ) : (
                        <>
                            <h1 className="dc-title">FORGOT PASSWORD</h1>
                            <p style={{ color: 'var(--dc-text-muted)', fontSize: '.85rem', marginBottom: '1.25rem', textAlign: 'center', lineHeight: 1.5 }}>
                                Enter your email and we'll send you a reset link.
                            </p>

                            {error && <div className="dc-error">{error}</div>}

                            <form onSubmit={handleSubmit(onSubmit)}>
                                <div className="dc-field">
                                    <Mail size={16} className="dc-field-icon" />
                                    <input {...register('email')} type="email" placeholder="Email address" autoComplete="email"
                                        className={`dc-input ${errors.email ? 'dc-input-err' : ''}`} />
                                </div>
                                {errors.email && <p className="dc-field-error">{errors.email.message}</p>}

                                <button type="submit" className="dc-btn-primary" disabled={isSubmitting}>
                                    {isSubmitting ? <Loader2 size={17} className="spin" /> : 'SEND RESET LINK'}
                                </button>
                            </form>
                        </>
                    )}
                </div>

                <div className="dc-card-bottom">
                    <a href="/login" className="dc-link-forgot" style={{ display: 'flex', alignItems: 'center', gap: '.4rem', justifyContent: 'center' }}>
                        <ArrowLeft size={14} /> Back to login
                    </a>
                </div>
            </div>
        </div>
    )
}
