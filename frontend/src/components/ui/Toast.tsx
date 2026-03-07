'use client'
import { useState, useCallback, createContext, useContext, ReactNode } from 'react'
import { CheckCircle, XCircle, AlertCircle, Info, X } from 'lucide-react'

type ToastType = 'success' | 'error' | 'warning' | 'info'

interface Toast {
    id: string
    type: ToastType
    message: string
}

interface ToastContextValue {
    toast: (type: ToastType, message: string) => void
    success: (message: string) => void
    error: (message: string) => void
    warning: (message: string) => void
    info: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
    const [toasts, setToasts] = useState<Toast[]>([])

    const dismiss = useCallback((id: string) => {
        setToasts(p => p.filter(t => t.id !== id))
    }, [])

    const toast = useCallback((type: ToastType, message: string) => {
        const id = Math.random().toString(36).slice(2)
        setToasts(p => [...p, { id, type, message }])
        setTimeout(() => dismiss(id), 4000)
    }, [dismiss])

    const success = useCallback((m: string) => toast('success', m), [toast])
    const error = useCallback((m: string) => toast('error', m), [toast])
    const warning = useCallback((m: string) => toast('warning', m), [toast])
    const info = useCallback((m: string) => toast('info', m), [toast])

    const icons = {
        success: <CheckCircle size={18} />,
        error: <XCircle size={18} />,
        warning: <AlertCircle size={18} />,
        info: <Info size={18} />,
    }

    return (
        <ToastContext.Provider value={{ toast, success, error, warning, info }}>
            {children}
            {/* Toast container */}
            <div className="toast-container">
                {toasts.map(t => (
                    <div key={t.id} className={`toast toast-${t.type}`}>
                        <span className="toast-icon">{icons[t.type]}</span>
                        <span className="toast-msg">{t.message}</span>
                        <button className="toast-close" onClick={() => dismiss(t.id)}>
                            <X size={14} />
                        </button>
                    </div>
                ))}
            </div>
        </ToastContext.Provider>
    )
}

export function useToast() {
    const ctx = useContext(ToastContext)
    if (!ctx) throw new Error('useToast must be used within ToastProvider')
    return ctx
}
