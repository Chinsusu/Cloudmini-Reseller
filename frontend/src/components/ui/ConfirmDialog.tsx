'use client'
import { ReactNode } from 'react'

interface ConfirmDialogProps {
    open: boolean
    title: string
    message: string
    confirmLabel?: string
    cancelLabel?: string
    variant?: 'danger' | 'warning' | 'primary'
    onConfirm: () => void
    onCancel: () => void
}

export function ConfirmDialog({
    open, title, message,
    confirmLabel = 'Confirm', cancelLabel = 'Cancel',
    variant = 'danger',
    onConfirm, onCancel
}: ConfirmDialogProps) {
    if (!open) return null

    return (
        <div className="modal-overlay" onClick={onCancel}>
            <div className="modal-card" onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <h3 className="modal-title">{title}</h3>
                </div>
                <div className="modal-body">
                    <p>{message}</p>
                </div>
                <div className="modal-footer">
                    <button className="btn-ghost" onClick={onCancel}>{cancelLabel}</button>
                    <button
                        className={variant === 'danger' ? 'btn-danger' : 'btn-primary'}
                        onClick={onConfirm}
                    >
                        {confirmLabel}
                    </button>
                </div>
            </div>
        </div>
    )
}

// Hook for easy usage
import { useState, useCallback } from 'react'

export function useConfirm() {
    const [state, setState] = useState<{
        open: boolean; title: string; message: string;
        confirmLabel?: string; variant?: 'danger' | 'warning' | 'primary';
        resolve?: (val: boolean) => void
    }>({ open: false, title: '', message: '' })

    const confirm = useCallback((opts: {
        title: string; message: string;
        confirmLabel?: string; variant?: 'danger' | 'warning' | 'primary'
    }): Promise<boolean> => {
        return new Promise(resolve => {
            setState({ ...opts, open: true, resolve })
        })
    }, [])

    const handleConfirm = useCallback(() => {
        state.resolve?.(true)
        setState(p => ({ ...p, open: false }))
    }, [state])

    const handleCancel = useCallback(() => {
        state.resolve?.(false)
        setState(p => ({ ...p, open: false }))
    }, [state])

    const dialog = (
        <ConfirmDialog
            open={state.open}
            title={state.title}
            message={state.message}
            confirmLabel={state.confirmLabel}
            variant={state.variant}
            onConfirm={handleConfirm}
            onCancel={handleCancel}
        />
    )

    return { confirm, dialog }
}
