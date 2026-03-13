/**
 * Format a numeric value as Vietnamese Dong (VND)
 * e.g. 120000 → "120.000đ"
 */
export function formatVND(value: number | string | null | undefined): string {
    if (value === null || value === undefined || value === '') return '0đ'
    const n = Math.round(parseFloat(String(value)))
    if (isNaN(n)) return '0đ'
    return n.toLocaleString('vi-VN') + 'đ'
}
