# Trang: `/admin/vps` — Quản lý VPS Plans

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Admin tạo mới và bật/tắt các VPS plans để user có thể deploy VPS.

---

## Layout

```
┌──────────────────────────────────────────────────┐
│  Header: "VPS Plans"              [+ Add Plan]   │
├──────────────────────────────────────────────────┤
│  Table: danh sách VPS plans (paginated)          │
├──────────────────────────────────────────────────┤
│  Pagination                                      │
└──────────────────────────────────────────────────┘
```

---

## Add Plan Modal

| Field | Loại | Validation |
|-------|------|-----------|
| Plan Name | text | Required |
| Slug | text | Lowercase, no spaces (vd: `vps-basic`) |
| CPU Cores | number | Min 1 |
| RAM (MB) | number | Min 512 |
| Disk (GB) | number | Min 10 |
| Monthly Rate ($) | number | Required |
| Hourly Rate ($) | number | Optional (auto-compute nếu trống: `monthly / 730`) |

Submit: `POST /api/v1/admin/vps/plans`

---

## VPS Plans Table

| Column | Mô tả |
|--------|-------|
| Name | Plan name (bold) + slug sub-text |
| CPU | `N cores` |
| RAM | `N GB` (chuyển từ MB: `ram_mb / 1024`) |
| Disk | `N GB` |
| Monthly | `$X.XX/mo` |
| Hourly | `$X.XXXX/hr` |
| Status | Toggle button (Active/Inactive) |

### Status Toggle

- Click → `PUT /api/v1/admin/vps/plans/{id}/toggle`
- Icon: `ToggleRight` (green) / `ToggleLeft` (grey)
- Disabled trong khi call đang pending

---

## API Calls

```
GET  /api/v1/admin/vps/plans?page=N&limit=20
POST /api/v1/admin/vps/plans   { name, slug, cpu_cores, ram_mb, disk_gb, monthly_rate, hourly_rate }
PUT  /api/v1/admin/vps/plans/{id}/toggle
```

---

## File

`frontend/src/app/admin/vps/page.tsx`
