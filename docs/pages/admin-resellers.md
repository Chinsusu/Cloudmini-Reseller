# Trang: `/admin/resellers` — Quản lý Resellers

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Admin xem danh sách đầy đủ tất cả resellers với khả năng filter theo status, approve và suspend resellers.

---

## Layout

```
┌──────────────────────────────────────────────────┐
│  Header: "Resellers"       [Filter: All|Pending] │
├──────────────────────────────────────────────────┤
│  Table: danh sách resellers (paginated)          │
└──────────────────────────────────────────────────┘
```

---

## Filter Bar

Tab/button group:
- **All** — tất cả resellers
- **Pending** — chờ duyệt
- **Approved** — đã active
- **Suspended** — bị suspend

---

## Resellers Table

| Column | Mô tả |
|--------|-------|
| Company | `company_name` + `website` sub-text |
| Email | Email |
| Status | Badge: pending (yellow) / approved (green) / suspended (red) |
| Commission | `commission_pct%` |
| Created | Date `DD MMM YYYY` |
| Actions | Approve / Suspend |

---

## Actions

### Approve Reseller

- Chỉ hiện khi `status = pending`
- `useConfirm({ variant: 'primary', message: 'Approve "${company_name}" as a reseller?' })`
- `PUT /api/v1/admin/resellers/{id}/approve`
- Sau khi approve: reseller nhận email thông báo (billing-service / notification-service)

### Suspend Reseller

- Chỉ hiện khi `status ≠ suspended`
- `useConfirm({ variant: 'danger' })`
- `PUT /api/v1/admin/resellers/{id}/suspend   { reason: 'Suspended by admin' }`
- Sub-accounts của reseller đó mất quyền truy cập

---

## API Calls

```
GET /api/v1/admin/resellers?status=pending|approved|suspended
PUT /api/v1/admin/resellers/{id}/approve
PUT /api/v1/admin/resellers/{id}/suspend   { reason }
```

---

## File

`frontend/src/app/admin/resellers/page.tsx`
