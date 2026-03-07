# Admin Proxy Products — `/admin/proxy`

## Tổng quan

**URL:** `/admin/proxy`  
**Role yêu cầu:** `admin`, `super_admin`  
**Layout:** `<AppLayout>` với breadcrumb `[Admin > Proxy Products]`

Trang quản lý danh mục proxy products. Admin tạo products mới, bật/tắt từng product. Products được hiển thị cho user khi đặt lệnh mua proxy tại `/dashboard/proxy`.

---

## Layout

```
┌─────────────────────────────────────────────────────────┐
│  Page Header                                            │
│  "Proxy Products"  [N products]        [+ Add Product]  │
├─────────────────────────────────────────────────────────┤
│  Products Table                                         │
│  Name | Type | Protocol | Location | Duration | BW |   │
│       | Cost | Status                                   │
│  ─────────────────────────────────────────────────────  │
│  row...                                                 │
├─────────────────────────────────────────────────────────┤
│  Pagination  (20 per page)                              │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────── Modal ───────────────────────┐
│  Add Proxy Product                                   [X] │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Product Name (full width)                       │   │
│  │ ─────────────────────────────────────────────── │   │
│  │ Proxy Type [select]  │  Protocol [select]        │   │
│  │ Location             │  Base Cost ($)            │   │
│  │ Duration (days, opt) │  Bandwidth GB (opt)       │   │
│  │ Provider ID (UUID, full width)                  │   │
│  └─────────────────────────────────────────────────┘   │
│  [          Create Product          ]                   │
└─────────────────────────────────────────────────────────┘
```

---

## Components

### Products Table

**Columns:**

| Column | Type | Mô tả |
|--------|------|-------|
| Name | text bold | `product.name` |
| Type | badge info | `product.proxy_type` (residential / datacenter / mobile) |
| Protocol | badge secondary | `product.protocol` UPPERCASE (HTTP / SOCKS5) |
| Location | text | `product.location` (US, VN, EU...) |
| Duration | text muted | `product.duration_days + "d"` hoặc `—` nếu null |
| Bandwidth | text muted | `product.bandwidth_gb + " GB"` hoặc `—` nếu null |
| Cost | text bold | `$parseFloat(product.base_cost).toFixed(2)` |
| Status | toggle button | Active/Inactive (green/grey), click để toggle |

**Status Toggle Button:**
- Icon: `<ToggleRight>` (green) khi active, `<ToggleLeft>` (grey) khi inactive
- Text: "Active" / "Inactive"
- Click → `PUT /api/v1/admin/proxy/products/{id}/toggle` → refetch

### Add Product Modal

**Fields:**

| Field | Type | Validation |
|-------|------|-----------|
| Product Name | text input | required |
| Proxy Type | select | residential / datacenter / mobile |
| Protocol | select | http / socks5 |
| Location | text input | e.g. "US", "VN", "EU" |
| Base Cost ($) | number (step 0.01) | required, min 0 |
| Duration (days) | number | optional, min 1 |
| Bandwidth (GB) | number (step 0.1) | optional, min 0 |
| Provider ID | text (UUID format) | UUID của provider backend |

**Grid layout:** 2 columns `1fr 1fr`, Product Name và Provider ID chiếm full width (`gridColumn: '1/-1'`)

**Submit button:**
- `disabled` khi `!form.name || !form.base_cost || isPending`
- Text: "Create Product" / "Creating..."

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/admin/proxy/products?page=N&limit=20` | List all products (active + inactive) |
| `POST` | `/api/v1/admin/proxy/products` | Tạo product mới |
| `PUT` | `/api/v1/admin/proxy/products/{id}/toggle` | Toggle is_active |

**Request body — Create Product:**
```json
{
  "provider_id": "uuid",
  "name": "US Residential HTTP 30 Days",
  "proxy_type": "residential",
  "protocol": "http",
  "location": "US",
  "duration_days": 30,
  "bandwidth_gb": null,
  "base_cost": "5.00"
}
```

**Response — List:**
```json
{
  "data": [
    {
      "id": "uuid",
      "provider_id": "uuid",
      "name": "US Residential HTTP",
      "proxy_type": "residential",
      "protocol": "http",
      "location": "US",
      "duration_days": 30,
      "bandwidth_gb": null,
      "base_cost": "5.00",
      "is_active": true,
      "created_at": "2026-01-15T08:00:00Z"
    }
  ],
  "meta": { "total": 42, "page": 1, "limit": 20, "pages": 3 }
}
```

---

## User Flows

### Tạo Product Mới
```
Click "Add Product"
→ Modal mở
→ Điền form (name, type, protocol, location, cost, provider_id)
→ Click "Create Product"
→ POST /admin/proxy/products
→ Toast "Product created" + close modal + refetch
```

### Toggle Active/Inactive
```
Click Toggle button trên row
→ PUT /admin/proxy/products/{id}/toggle
→ Toast "Updated" + refetch
→ Icon/color đổi ngay sau khi refetch
```

---

## States

| State | Hiển thị |
|-------|---------|
| Loading | Loading spinner |
| Empty | Empty state + "Add first product" button |
| Creating | Button disabled + "Creating..." text |
| Toggle pending | disabled trong lúc mutation chạy |

---

## Business Rules

1. Product `is_active = false` sẽ **không hiện** cho user khi Buy Proxy
2. `base_cost` là giá gốc từ provider — billing engine sẽ tính markup thêm nếu có reseller
3. `provider_id` phải là UUID hợp lệ của một provider đã đăng ký trong DB
4. Một product có thể không có `duration_days` (unlimited) hoặc không có `bandwidth_gb` (unlimited)

---

## Navigation
**Sidebar link:** MANAGEMENT → Proxy Products  
**Breadcrumb:** Admin > Proxy Products
