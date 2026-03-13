# Trang: `/reseller` — Reseller Dashboard

> Role: **reseller** (đã được approve)

---

## Mục đích

Tổng quan cho reseller: xem thống kê nhanh và truy cập các tính năng quản lý.

---

## Layout

```
┌──────────┬──────────┬──────────┬──────────┐
│  Sub-    │  Pricing │  API     │ Commission│
│ Accounts │  Rules   │  Keys    │    %      │
├──────────┴──────────┴──────────┴──────────┤
│  Quick Links Grid (4 cards)               │
└───────────────────────────────────────────┘
```

---

## Stat Cards (4 cards)

| Card | Giá trị lấy từ | Icon |
|------|---------------|------|
| Sub-Accounts | `reseller.users.count` | Users |
| Pricing Rules | `reseller.pricing.count` | DollarSign |
| API Keys | `reseller.api_keys.count` | Key |
| Commission % | `reseller.commission_pct` | Percent |

---

## Quick Links

4 cards lớn với icon, title, mô tả ngắn và link:

| Card | Link |
|------|------|
| Pricing Management | `/reseller/pricing` |
| Sub-Account Management | `/reseller/accounts` |
| API Keys | `/reseller/api-keys` |
| Webhooks | `/reseller/webhooks` |

---

## Trạng thái không được approve

Nếu `reseller.status = pending`: Hiện banner vàng "Your reseller account is pending approval". Không khóa UI nhưng các API calls sẽ trả 403 nếu cố thực hiện action.

---

## API Calls

```
GET /api/v1/reseller/dashboard    → { sub_accounts_count, pricing_count, commission_pct }
GET /api/v1/reseller/users        → count
GET /api/v1/reseller/pricing      → count
GET /api/v1/reseller/api-keys     → count
```

---

## File

`frontend/src/app/reseller/page.tsx`
