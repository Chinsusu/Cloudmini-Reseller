# Admin VPS Plans — `/admin/vps`

## Tổng quan

**URL:** `/admin/vps`  
**Role yêu cầu:** `admin`, `super_admin`  
**Layout:** `<AppLayout>` với breadcrumb `[Admin > VPS Plans]`

Trang quản lý VPS plans. Admin tạo plans với cấu hình CPU/RAM/Disk và giá tiền. Plans được hiển thị cho user khi deploy VPS tại `/dashboard/vps`.

---

## Layout

```
┌─────────────────────────────────────────────────────────┐
│  Page Header                                            │
│  "VPS Plans"  [Manage VPS configurations]  [+ Add Plan] │
├─────────────────────────────────────────────────────────┤
│  Plans Table                                            │
│  Name/Slug | CPU | RAM | Disk | Monthly | Hourly | Status│
│  ─────────────────────────────────────────────────────  │
│  row...                                                 │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────── Modal ───────────────────────┐
│  Add VPS Plan                                        [X] │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Plan Name            │  Slug                    │   │
│  │ CPU Cores            │  RAM (MB)                │   │
│  │ Disk (GB)            │  Monthly Rate ($)        │   │
│  │ Hourly Rate ($)      │                          │   │
│  └─────────────────────────────────────────────────┘   │
│  [             Create Plan              ]               │
└─────────────────────────────────────────────────────────┘
```

---

## Components

### Plans Table

**Columns:**

| Column | Type | Mô tả |
|--------|------|-------|
| Name | text bold + slug sub | `plan.name` (bold) + `plan.slug` (muted, 0.75rem below) |
| CPU | icon + text | `<Cpu size={12}> {plan.cpu_cores} vCPU` |
| RAM | icon + text | `<MemoryStick size={12}> {plan.ram_mb / 1024} GB` |
| Disk | icon + text | `<HardDrive size={12}> {plan.disk_gb} GB` |
| Monthly | text bold | `$parseFloat(plan.monthly_rate).toFixed(2)` |
| Hourly | text muted | `$parseFloat(plan.hourly_rate).toFixed(4)` |
| Status | toggle button | Active/Inactive (green/grey) |

**Status Toggle Button:**
- Icon: `<ToggleRight>` (success green) khi active, `<ToggleLeft>` (text-muted) khi inactive
- Click → `PUT /api/v1/admin/vps/plans/{id}/toggle` → Toast "Updated" → refetch

### Add Plan Modal

**Fields:**

| Field | Type | Validation |
|-------|------|-----------|
| Plan Name | text input | required |
| Slug | text input | required, lowercase-dash |
| CPU Cores | number (min 1) | required |
| RAM (MB) | number (min 512) | required, step 512 |
| Disk (GB) | number (min 10) | required |
| Monthly Rate ($) | number (step 0.01) | required |
| Hourly Rate ($) | number (step 0.0001) | required |

**Grid layout:** 2 columns `1fr 1fr`

**Submit button:**
- `disabled` khi `!name || !cpu_cores || !ram_mb || !disk_gb || isPending`
- Text: "Create Plan" / "Creating..."

**RAM helper:** Placeholder gợi ý `2048` (= 2 GB). Frontend hiển thị `ram_mb / 1024` GB trong bảng.

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/admin/vps/plans` | List all VPS plans |
| `POST` | `/api/v1/admin/vps/plans` | Tạo plan mới |
| `PUT` | `/api/v1/admin/vps/plans/{id}/toggle` | Toggle is_active |

> **Lưu ý:** `GET /api/v1/vps/plans` (không có `/admin`) chỉ trả active plans — dùng cho user khi deploy VPS.
> Admin endpoint `/admin/vps/plans` trả cả inactive.

**Request body — Create Plan:**
```json
{
  "name": "Starter 2",
  "slug": "starter-2",
  "cpu_cores": 2,
  "ram_mb": 2048,
  "disk_gb": 40,
  "hourly_rate": "0.0277",
  "monthly_rate": "19.99"
}
```

**Response — List (array):**
```json
[
  {
    "id": "uuid",
    "name": "Starter 2",
    "slug": "starter-2",
    "cpu_cores": 2,
    "ram_mb": 2048,
    "disk_gb": 40,
    "bandwidth_gb": 0,
    "hourly_rate": "0.0277",
    "monthly_rate": "19.99",
    "is_active": true,
    "created_at": "2026-01-01T00:00:00Z"
  }
]
```

---

## User Flows

### Tạo Plan Mới
```
Click "Add Plan"
→ Modal mở
→ Điền form (name, slug, CPU, RAM, Disk, rates)
→ Click "Create Plan"
→ POST /admin/vps/plans
→ Toast "Plan created" + close modal + refetch
```

### Toggle Active/Inactive
```
Click Toggle button
→ PUT /admin/vps/plans/{id}/toggle
→ Toast "Updated" + refetch
→ Nếu deactivate → plan không hiện cho user khi Deploy VPS
```

---

## States

| State | Hiển thị |
|-------|---------|
| Loading | Loading spinner |
| Empty | Empty state icon + "Add first plan" button |
| Creating | Button disabled + "Creating..." text |
| Toggle pending | button disabled trong lúc mutation |

---

## Business Rules

1. Plan `is_active = false` → **không hiện** trong Deploy VPS modal của user
2. `slug` phải unique — dùng để reference trong provisioning scripts
3. `ram_mb` lưu trong DB dưới đơn vị MB, frontend convert `/1024` khi hiện thị (GB)
4. `hourly_rate` dùng cho billing khi user bị charge theo giờ (chưa thanh toán tháng)
5. `monthly_rate` dùng cho hiển thị giá trong Deploy VPS modal và billing định kỳ
6. `bandwidth_gb = 0` có nghĩa là unlimited (không có giới hạn bandwidth)

---

## Pricing Reference

Các mức giá tham khảo:

| Tier | CPU | RAM | Disk | Monthly |
|------|-----|-----|------|---------|
| Nano | 1 vCPU | 512 MB | 10 GB | ~$3 |
| Micro | 1 vCPU | 1 GB | 20 GB | ~$5 |
| Starter | 2 vCPU | 2 GB | 40 GB | ~$10 |
| Standard | 4 vCPU | 8 GB | 80 GB | ~$24 |
| Pro | 8 vCPU | 16 GB | 160 GB | ~$48 |

---

## Navigation
**Sidebar link:** MANAGEMENT → VPS Plans  
**Breadcrumb:** Admin > VPS Plans
