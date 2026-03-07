# Admin Dashboard — `/admin`

## Tổng quan

**URL:** `/admin`  
**Role yêu cầu:** `admin`, `super_admin`  
**Layout:** `<AppLayout>` với breadcrumb `[Dashboard]`

Trang tổng quan của admin, hiện thị stats toàn hệ thống và quản lý resellers đang chờ duyệt.

---

## Layout

```
┌─────────────────────────────────────────────────────────┐
│  Stat Cards (2 cards)                                   │
│  ┌──────────────┐  ┌──────────────┐                    │
│  │ Total Users  │  │ Total        │                    │
│  │     1,234    │  │ Resellers 56 │                    │
│  └──────────────┘  └──────────────┘                    │
├─────────────────────────────────────────────────────────┤
│  Reseller Management Table                              │
│  Company | Email | Status | Commission | Created |Actions│
│  ─────────────────────────────────────────────────────  │
│  row...                                                 │
├─────────────────────────────────────────────────────────┤
│  Pagination                                             │
└─────────────────────────────────────────────────────────┘
```

---

## Components

### Stat Cards
| Card | Value | Icon |
|------|-------|------|
| Total Users | `stats.total_users` | Users icon |
| Total Resellers | `stats.total_resellers` | ShieldCheck icon |

Style: white card, 24px rounded, icon left + number + label layout.

### Reseller Table

**Columns:**

| Column | Type | Mô tả |
|--------|------|-------|
| Company | text | `reseller.company_name` |
| Email | text | `reseller.email` (từ user account) |
| Status | badge | pending/approved/suspended |
| Commission | text | `reseller.commission_pct%` |
| Created | date | `created_at` formatted `MMM DD, YYYY` |
| Actions | buttons | Tuỳ theo status |

**Status badges:**
| Status | Color |
|--------|-------|
| pending | warning (yellow) |
| approved | success (green) |
| suspended | error (red) |

**Action buttons:**
| Button | Hiện khi | Behavior |
|--------|----------|---------|
| ✅ Approve | `status = pending` | `PUT /api/v1/admin/resellers/{id}/approve` |
| 🚫 Suspend | `status = approved` | → `useConfirm` với reason input → `PUT /api/v1/admin/resellers/{id}/suspend` |

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/admin/stats` | Lấy system stats |
| `GET` | `/api/v1/admin/resellers?page=N` | Danh sách resellers |
| `PUT` | `/api/v1/admin/resellers/{id}/approve` | Duyệt reseller |
| `PUT` | `/api/v1/admin/resellers/{id}/suspend` | Suspend reseller |

**Request body — Suspend:**
```json
{ "reason": "Violation of terms of service" }
```

**Response — Stats:**
```json
{
  "total_users": 1234,
  "total_resellers": 56
}
```

---

## User Flows

### Approve Reseller
```
Admin click "Approve" → PUT /admin/resellers/{id}/approve
→ Toast "Reseller approved" → refetch table
```

### Suspend Reseller
```
Admin click "Suspend" → useConfirm opens (với textarea reason)
→ Admin nhập reason → confirm
→ PUT /admin/resellers/{id}/suspend {reason}
→ Toast "Reseller suspended" → refetch table
```

---

## States

| State | Hiển thị |
|-------|---------|
| Loading | Skeleton / spinner |
| Empty (no resellers) | Empty state icon + message |
| Error | Toast error |

---

## Navigation
**Sidebar link:** OVERVIEW → Dashboard  
**Breadcrumb:** Dashboard
