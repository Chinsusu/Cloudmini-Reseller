# Trang: `/dashboard` — Tổng quan User

> Role: **user** (và admin khi dùng chế độ service)

---

## Mục đích

Trang home của user sau khi đăng nhập. Hiển thị tổng quan nhanh về tài khoản: số dư ví, proxy orders đang có, VPS instances, và giao dịch gần nhất.

---

## Layout

```
┌─────────────────────────────────────────┐
│  Page Header: "Dashboard"               │
├──────────┬──────────┬───────────────────┤
│  Wallet  │  Proxy   │  VPS Instances    │
│  Balance │  Orders  │  count            │
├─────────────────────────────────────────┤
│  Recent Transactions (5 rows)           │
└─────────────────────────────────────────┘
```

---

## Components

### Stat Cards (3 cards)

| Card | Giá trị | Icon | Màu |
|------|---------|------|-----|
| Wallet Balance | `wallet.balance` dạng `$X.XX` | Wallet | Purple |
| Proxy Orders | Số order đang active | Globe | Green |
| VPS Instances | Số VPS đang running | Server | Teal |

### Recent Transactions Table

| Column | Mô tả |
|--------|-------|
| Date | `MMM DD, YYYY` |
| Type | Badge: deposit / debit / hold / refund |
| Description | Text mô tả giao dịch |
| Amount | `+$X.XX` (green credit) / `-$X.XX` (red debit) |
| Status | Badge: completed / pending |

Chỉ hiện **5 rows** gần nhất, không có pagination.  
Link "View all" → `/dashboard/wallet`

---

## API Calls

```
GET /api/v1/billing/wallet               → balance, hold_amount
GET /api/v1/proxy/orders?limit=1         → total count
GET /api/v1/vps/instances?limit=1        → total count
GET /api/v1/billing/transactions?limit=5 → recent transactions
```

---

## Quick Links

- "Buy Proxy" → `/dashboard/proxy`
- "Deploy VPS" → `/dashboard/vps`
- "Top Up" → `/dashboard/wallet`

---

## File

`frontend/src/app/dashboard/page.tsx`
