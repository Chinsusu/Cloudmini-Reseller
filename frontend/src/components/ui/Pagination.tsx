'use client'

interface PaginationProps {
    page: number
    totalPages: number
    total: number
    limit: number
    onPageChange: (page: number) => void
}

export function Pagination({ page, totalPages, total, limit, onPageChange }: PaginationProps) {
    if (total === 0) return null

    const from = Math.min((page - 1) * limit + 1, total)
    const to   = Math.min(page * limit, total)

    // Build page numbers with ellipsis
    const pages: (number | '...')[] = []
    if (totalPages <= 7) {
        for (let i = 1; i <= totalPages; i++) pages.push(i)
    } else {
        pages.push(1)
        if (page > 3) pages.push('...')
        for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) pages.push(i)
        if (page < totalPages - 2) pages.push('...')
        pages.push(totalPages)
    }

    return (
        <div className="pagination">
            {/* Left: entries info */}
            <span className="pagination-info">
                Hiển thị {from}–{to} của {total} mục
            </span>

            {/* Right: page buttons */}
            <div className="pagination-controls">
                <button className="pg-btn" disabled={page === 1} onClick={() => onPageChange(1)} title="First">«</button>
                <button className="pg-btn" disabled={page === 1} onClick={() => onPageChange(page - 1)}>‹</button>

                {pages.map((p, i) =>
                    p === '...'
                        ? <span key={`e${i}`} className="pg-ellipsis">…</span>
                        : <button
                            key={p}
                            className={`pg-btn ${p === page ? 'pg-active' : ''}`}
                            onClick={() => onPageChange(p)}
                        >{p}</button>
                )}

                <button className="pg-btn" disabled={page === totalPages || totalPages <= 1} onClick={() => onPageChange(page + 1)}>›</button>
                <button className="pg-btn" disabled={page === totalPages || totalPages <= 1} onClick={() => onPageChange(totalPages)} title="Last">»</button>
            </div>
        </div>
    )
}
