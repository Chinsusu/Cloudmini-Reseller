# Trang: `/admin/proxy` — Quản lý Proxy Products

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Admin tạo, chỉnh sửa, bật/tắt và xóa các sản phẩm proxy. Mỗi product gắn với một provider cụ thể và cấu hình plan tương ứng của provider đó. User/reseller đặt hàng dựa trên danh sách products này.

---

## Layout

```
┌──────────────────────────────────────────────────────┐
│  Header: "Proxy Products"            [+ Add Product] │
│  Subtitle: "N products"                              │
├──────────────────────────────────────────────────────┤
│  Table: danh sách tất cả products (active + inactive)│
├──────────────────────────────────────────────────────┤
│  Pagination                                          │
└──────────────────────────────────────────────────────┘
```

> Products inactive được hiển thị với `opacity: 0.5` — không bị ẩn khỏi list.

---

## Add Product Modal

### Fields cơ bản

| Field | Loại | Validation |
|-------|------|-----------|
| Product Name | text | Required — vd: "US Static Residential Basic" |
| Proxy Type | select | `residential` / `datacenter` / `mobile` |
| Protocol | select | `HTTP` / `SOCKS5` |
| Location label | text | Optional — vd: "Global", "US, VN, UK" |
| Base Cost ($) | number | Required — giá nhập hàng từ provider |
| Duration (days) | number | Optional — `null` nếu không giới hạn |
| Bandwidth GB | number | Optional — `null` nếu không giới hạn |
| Provider | select | Dropdown → `GET /api/v1/admin/proxy/providers` |

### Provider Plan Configuration (dynamic, per adapter)

Hiển thị sau khi chọn Provider. Các field phụ thuộc vào `adapter_type` của provider.

#### `proxy_cheap` — Proxy-Cheap Adapter

> ⚠️ **Đơn vị thời gian: THÁNG**. Mọi order và gia hạn của Proxy-Cheap đều tính theo tháng, không phải ngày.

| Field | Loại | Mô tả |
|-------|------|-------|
| Service | select | Loại proxy (xem bảng bên dưới) |
| Plan | select | Chỉ hiển thị với service có plans: `basic` / `standard` / `premium` / `dedicated` |
| Package size | select | Chỉ hiển thị với `static-datacenter-ipv6`: 50 / 150 / 500 proxies |
| Traffic GB included | number | Chỉ hiển thị với rotating services (tính theo GB traffic) |
| Default period (months) | number | Số tháng mặc định khi đặt hàng (thường là 1) |

**Services & cấu trúc plan:**

| Service ID | Label | Plan / Billing |
|-----------|-------|----------------|
| `static-residential-ipv4` | Static Residential IPv4 (ISP) | basic / standard / premium × tháng |
| `static-datacenter-ipv4` | Static Datacenter IPv4 | basic / standard / premium × tháng |
| `static-datacenter-ipv6` | Static Datacenter IPv6 | package (50/150/500 proxies) × tháng |
| `dedicated-mobile` | Dedicated Mobile (Static) | dedicated × tháng |
| `rotating-mobile` | Rotating Mobile | per GB traffic |
| `rotating-residential` | Rotating Residential | per GB traffic |

> **Lưu ý:** Country và ISP **không** cấu hình ở đây — client chọn khi đặt hàng.

Submit: `POST /api/v1/admin/proxy/products`

---

## Products Table

| Column | Mô tả |
|--------|-------|
| Name | Tên sản phẩm |
| Provider | Display name + adapter_type (vd: "Proxy-Cheap (proxy_cheap)") |
| Type | Badge: residential (green) / datacenter (blue) / mobile (orange) |
| Protocol | Badge: HTTP / SOCKS5 |
| Location | Text label hoặc "—" |
| Duration | `Nd` hoặc "—" |
| Bandwidth | `N GB` hoặc "—" |
| Cost | `$X.XX` |
| Status | Toggle button (Active ↔ Disabled) |
| Actions | Edit (pencil) + Delete (trash) |

### Status Toggle

- Click → `PUT /api/v1/admin/proxy/products/{id}/toggle`
- Inactive products vẫn hiển thị trong admin list (opacity giảm)
- Icons: `ToggleRight` (green, Active) ↔ `ToggleLeft` (grey, Disabled)

### Edit Product

- Click pencil icon → mở Edit modal
- Tương tự Add modal nhưng:
  - Provider hiển thị read-only (không đổi được sau khi tạo)
  - Provider Plan Configuration pre-populate từ `product.metadata`
- Submit: `PUT /api/v1/admin/proxy/products/{id}`

### Delete Product

- Click trash icon → `useConfirm({ variant: 'danger' })` xác nhận
- Submit: `DELETE /api/v1/admin/proxy/products/{id}`

---

## API Calls

```
GET    /api/v1/admin/proxy/products?page=N&limit=20
POST   /api/v1/admin/proxy/products     { name, proxy_type, protocol, location?, base_cost, duration_days?, bandwidth_gb?, provider_id, metadata? }
PUT    /api/v1/admin/proxy/products/{id}
PUT    /api/v1/admin/proxy/products/{id}/toggle
DELETE /api/v1/admin/proxy/products/{id}
GET    /api/v1/admin/proxy/providers    (để populate Provider dropdown)
GET    /api/v1/admin/proxy/service-options?service_id=X&plan_id=Y  (trả countries + ISPs từ proxy-cheap)
```

### Payload `metadata` (cho proxy_cheap)

```json
{
  "service_id":     "static-residential-ipv4",
  "plan_id":        "basic",
  "period_months":  "1"
}
```

Với IPv6 datacenter:
```json
{ "service_id": "static-datacenter-ipv6", "package_id": "50", "period_months": "1" }
```

Với rotating:
```json
{ "service_id": "rotating-residential", "traffic_gb": "10" }
```

---

## Lưu ý thiết kế

- **Separation of concerns:** Admin định nghĩa product (service + plan). Khi **client order**, client mới chọn country và ISP dựa trên product đã chọn.
- `metadata` của product lưu JSONB trong DB và được trả về API (`json:"metadata,omitempty"`) để Edit modal đọc được.
- Product không bị xóa khỏi list khi toggle — chỉ đổi status. Inactive products hiển thị mờ.

---

## File

`frontend/src/app/admin/proxy/page.tsx`
