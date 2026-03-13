# Admin Proxy Products — `/admin/proxy`

## Tổng quan

**URL:** `/admin/proxy`  
**Role yêu cầu:** `admin`, `super_admin`  
**Layout:** `<AppLayout>` với breadcrumb `[Admin > Proxy Products]`

Trang quản lý danh mục proxy products. Admin tạo, chỉnh sửa, bật/tắt và xóa products. Products được hiển thị cho user khi đặt lệnh mua proxy tại `/dashboard/proxy`.

> Products inactive (`is_active = false`) vẫn hiển thị trong admin list (opacity giảm), nhưng không hiện cho user.

---

## Layout

```
┌─────────────────────────────────────────────────────────┐
│  Page Header                                            │
│  "Proxy Products"  [N products]        [+ Add Product]  │
├─────────────────────────────────────────────────────────┤
│  Products Table (active + inactive, paginated 20/page)  │
│  Name | Provider | Type | Proto | Location | Dur | BW  │
│       | Cost | Status | Actions                         │
├─────────────────────────────────────────────────────────┤
│  Pagination                                             │
└─────────────────────────────────────────────────────────┘
```

---

## Components

### Products Table

| Column | Mô tả |
|--------|-------|
| Name | `product.name` (bold) |
| Provider | `provider.display_name` + `adapter_type` sub-text |
| Type | Badge: residential / datacenter / mobile |
| Protocol | Badge: HTTP / SOCKS5 |
| Location | `product.location` hoặc "—" |
| Duration | `Nd` hoặc "—" |
| Bandwidth | `N GB` hoặc "—" |
| Cost | `$X.XX` |
| Status | Toggle button: `ToggleRight` (green) / `ToggleLeft` (grey) |
| Actions | Pencil (Edit) + Trash (Delete) |

Inactive rows: `opacity: 0.5`

---

### Add Product Modal

#### Fields cơ bản (2-column grid)

| Field | Type | Validation |
|-------|------|-----------|
| Product Name | text | Required |
| Proxy Type | select | residential / datacenter / mobile |
| Protocol | select | HTTP / SOCKS5 |
| Location label | text | Optional — vd: "Global", "US, VN" |
| Base Cost ($) | number step=0.01 | Required |
| Duration (days) | number | Optional |
| Bandwidth GB | number step=0.1 | Optional |
| Provider | select | Load từ `GET /api/v1/admin/proxy/providers` |

#### Provider Plan Configuration (dynamic)

Hiển thị khi đã chọn Provider. Fields phụ thuộc `adapter_type`.

**`proxy_cheap` — Proxy-Cheap Adapter:**

> ⚠️ Order và gia hạn tính theo **THÁNG**, không phải ngày.  
> Country và ISP **không** cấu hình ở đây — client chọn khi order.

| Field | Hiển thị khi | Mô tả |
|-------|-------------|-------|
| Service | Luôn | Loại proxy (6 loại) |
| Plan | Service có named plans | basic / standard / premium / dedicated |
| Package size | `static-datacenter-ipv6` | 50 / 150 / 500 proxies |
| Traffic GB | Rotating services | GB per purchase |
| Default period (months) | Service được chọn | Thường là 1 |

**Submit button:** disabled khi `!name || !base_cost || !provider_id || isPending`

---

### Edit Product Modal

Tương tự Add modal, ngoại trừ:
- Provider hiển thị **read-only** (không đổi được)
- Provider Plan Configuration **pre-populate** từ `product.metadata`
- Submit: `PUT /api/v1/admin/proxy/products/{id}`

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/admin/proxy/products?page=N&limit=20` | List all (active + inactive) |
| `POST` | `/api/v1/admin/proxy/products` | Tạo product |
| `PUT` | `/api/v1/admin/proxy/products/{id}` | Cập nhật product |
| `PUT` | `/api/v1/admin/proxy/products/{id}/toggle` | Toggle is_active |
| `DELETE` | `/api/v1/admin/proxy/products/{id}` | Xóa product |
| `GET` | `/api/v1/admin/proxy/providers` | Danh sách providers (cho dropdown) |
| `GET` | `/api/v1/admin/proxy/service-options?service_id=X&plan_id=Y` | Countries + ISPs từ proxy-cheap |

**Request body — Create/Update:**
```json
{
  "provider_id": "uuid",
  "name": "US Static Residential Basic",
  "proxy_type": "residential",
  "protocol": "http",
  "location": "Global",
  "duration_days": null,
  "bandwidth_gb": null,
  "base_cost": "5.00",
  "metadata": {
    "service_id":    "static-residential-ipv4",
    "plan_id":       "basic",
    "period_months": "1"
  }
}
```

**Response — List (với metadata):**
```json
{
  "data": [
    {
      "id": "uuid",
      "provider_id": "uuid",
      "name": "US Static Residential Basic",
      "proxy_type": "residential",
      "protocol": "http",
      "location": "Global",
      "duration_days": null,
      "bandwidth_gb": null,
      "base_cost": "5.00",
      "is_active": true,
      "metadata": {
        "service_id": "static-residential-ipv4",
        "plan_id": "basic",
        "period_months": "1"
      },
      "created_at": "2026-03-13T00:00:00Z"
    }
  ],
  "meta": { "total": 5, "page": 1, "limit": 20, "pages": 1 }
}
```

---

## User Flows

### Tạo Product mới
```
Click "Add Product"
→ Modal mở, chọn Provider từ dropdown
→ Provider Plan Configuration hiện (nếu adapter_type = proxy_cheap: chọn Service → Plan)
→ Điền các field cơ bản (name, cost, ...)
→ Click "Create Product"
→ POST /api/v1/admin/proxy/products   { ..., metadata: { service_id, plan_id, period_months } }
→ Toast "Product created" + close modal + refetch
```

### Chỉnh sửa Product
```
Click pencil icon trên row
→ Edit modal mở, pre-populate tất cả fields + metadata
→ Chỉnh sửa (Provider không đổi được)
→ Click "Save Changes"
→ PUT /api/v1/admin/proxy/products/{id}
→ Toast "Product updated" + close modal + refetch
```

### Toggle Active/Inactive
```
Click toggle button
→ PUT /api/v1/admin/proxy/products/{id}/toggle
→ Toast "Status updated" + refetch (row vẫn hiển thị, chỉ opacity đổi)
```

### Xóa Product
```
Click trash icon
→ ConfirmDialog hiện (variant: danger)
→ Confirm: DELETE /api/v1/admin/proxy/products/{id}
→ Toast "Product deleted" + refetch
```

---

## Business Rules

1. `is_active = false` → vẫn hiện trong admin, nhưng không hiện cho user khi Buy Proxy
2. `base_cost` = giá gốc provider — billing engine tính markup cho reseller
3. `provider_id` phải là UUID hợp lệ của provider đã đăng ký
4. `metadata` lưu JSONB trong DB — map tới config của từng adapter
5. Với proxy_cheap: `metadata.service_id` là bắt buộc, `metadata.plan_id` bắt buộc với static services có plan
6. Country và ISP **không** thuộc product — client chọn khi order

---

## Navigation

**Sidebar link:** MANAGEMENT → Proxy Products  
**Breadcrumb:** Admin > Proxy Products  
**Related:** `/admin/proxy/providers` (quản lý providers)
