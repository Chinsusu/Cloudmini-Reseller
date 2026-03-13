# Trang: `/dashboard/wallet` — Ví & Giao dịch

> Role: **user**

---

## Mục đích

User xem số dư ví, nạp tiền, và xem lịch sử giao dịch.

---

## Layout

```
┌───────────────┬───────────────┬──────────────┐
│ Total Balance │   Available   │   On Hold    │
├───────────────┴───────────────┴──────────────┤
│  Top-up Form                                 │
├──────────────────────────────────────────────┤
│  Transaction History Table (paginated)       │
└──────────────────────────────────────────────┘
```

---

## Stat Cards (3 cards)

| Card | Giá trị | Mô tả |
|------|---------|-------|
| Total Balance | `wallet.balance` | Toàn bộ số tiền trong tài khoản |
| Available | `balance - hold_amount` | Số tiền có thể dùng ngay |
| On Hold | `wallet.hold_amount` | Đang bị hold cho orders đang xử lý |

---

## Top-up Form

| Field | Loại | Validation |
|-------|------|-----------|
| Amount ($) | `number` input | Min $5 |
| Payment Method | `select` | Bank Transfer / Crypto (USDT/USDC) / Stripe / VNPay |

Submit → `POST /api/v1/billing/top-up`

Response handling:
- Success → Toast "Top-up request submitted"
- Stripe/VNPay → redirect đến payment gateway URL

---

## Transaction History Table

| Column | Mô tả |
|--------|-------|
| Date | `DD MMM YYYY HH:mm` |
| Type | Badge màu theo loại: deposit (green) / debit (red) / hold (yellow) / refund (blue) / fee (grey) |
| Description | Mô tả giao dịch (vd: "Proxy order #123", "Top-up via Crypto") |
| Amount | `+$X.XX` (credit, green) hoặc `-$X.XX` (debit, red) |
| Balance After | Số dư sau giao dịch |
| Status | Badge: completed / pending / failed |

Pagination: 20 rows/page.

---

## API Calls

```
GET  /api/v1/billing/wallet
GET  /api/v1/billing/transactions?page=N&limit=20
POST /api/v1/billing/top-up   { amount, payment_method }
```

---

## File

`frontend/src/app/dashboard/wallet/page.tsx`
