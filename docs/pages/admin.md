# Trang: `/admin` — Admin Dashboard

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Tổng quan nền tảng và quản lý resellers (approve/suspend). Trang landing đầu tiên khi admin đăng nhập.

---

## Layout

```
┌──────────┬──────────┬──────────┬──────────┐
│  Total   │Resellers │ Platform │ Pending  │
│  Users   │ count    │  Status  │ Approval │
├──────────┴──────────┴──────────┴──────────┤
│  Resellers Table (paginated)              │
└───────────────────────────────────────────┘
```

---

## Stat Cards (4 cards)

| Card | Giá trị | Icon |
|------|---------|------|
| Total Users | `users.meta.total` | Users |
| Resellers | `resellers.meta.total` | ShieldCheck |
| Platform Status | "Operational" (hardcoded) | Activity |
| Pending Approval | Số reseller `status = pending` | AlertCircle |

---

## Resellers Table

| Column | Mô tả |
|--------|-------|
| Company | `company_name` + `website` sub-text |
| Email | Email |
| Status | Badge: pending (yellow) / approved (green) / suspended (red) |
| Commission | `commission_pct%` |
| Created | Date |
| Actions | Approve / Suspend buttons |

### Approve

- Điều kiện: `status = pending`
- `useConfirm({ title: 'Approve Reseller', variant: 'primary' })`
- `PUT /api/v1/admin/resellers/{id}/approve`

### Suspend

- Điều kiện: `status ≠ suspended`
- `useConfirm({ title: 'Suspend Reseller', variant: 'danger' })`
- `PUT /api/v1/admin/resellers/{id}/suspend   { reason: 'Suspended by admin' }`

---

## API Calls

```
GET /api/v1/admin/users?page=1&limit=1   → total count
GET /api/v1/admin/resellers              → list
PUT /api/v1/admin/resellers/{id}/approve
PUT /api/v1/admin/resellers/{id}/suspend { reason }
```

---

## File

`frontend/src/app/admin/page.tsx`
