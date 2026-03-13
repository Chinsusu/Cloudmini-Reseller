# Trang: `/dashboard/proxy` — Quản lý Proxy Orders

> Role: **user**

---

## Mục đích

User xem, mua và quản lý proxy orders. Hỗ trợ xem credentials trực tiếp trong trang với toggle show/hide.

---

## Layout

```
┌──────────────────────────────────────────┐
│  Header: "Proxy Orders"  [Refresh][Buy]  │
├──────────────────────────────────────────┤
│  Table: danh sách orders (paginated)     │
├──────────────────────────────────────────┤
│  Pagination                              │
└──────────────────────────────────────────┘
```

---

## Table Columns

| Column | Mô tả |
|--------|-------|
| Order # | Order number, sub: product name |
| Qty | Số lượng proxy |
| Amount | `$X.XX` |
| Expires | Ngày hết hạn (hoặc "—") |
| Status | Badge với màu theo status |
| Credentials | Toggle show/hide + Copy button |
| Actions | Renew (khi gần hết hạn) + Cancel button |

### Status Colors

| Status | Màu |
|--------|-----|
| active | Green |
| processing | Yellow (async provider đang activate) |
| pending | Yellow |
| expired | Grey |
| cancelled / failed | Red |

---

## Buy Proxy Modal

### Bước 1: Chọn sản phẩm

- Filter bar: Proxy Type · Protocol
- Product cards grid:
  - Name + Provider name
  - Type badge (residential/datacenter/mobile)
  - Protocol badge (HTTP/SOCKS5)
  - Location label
  - Duration / Bandwidth
  - Price: `$X.XX` (billing đơn vị theo provider — Proxy-Cheap: per month)

### Bước 2: Cấu hình order (Provider-specific)

Các field phụ thuộc vào adapter type của provider gắn với product.

#### Proxy-Cheap — orders tính theo tháng

> ⚠️ **Đơn vị: THÁNG**. Order và gia hạn đều tính theo tháng.

| Field | Loại | Mô tả |
|-------|------|-------|
| Country | select | Danh sách countries từ `GET /api/v1/proxy/service-options?service_id=X&plan_id=Y` |
| ISP | select | ISPs theo country đã chọn (từ API) — hoặc "All ISPs" (client không giới hạn ISP) |
| Period (months) | number | Số tháng muốn mua (1–12) |
| Quantity | number | Số lượng proxy (integer ≥ 1) |

- Total preview: `quantity × price × months`
- ISP options: load real-time từ proxy-cheap API theo service + country

#### Providers khác

- Có thể order theo ngày / GB / proxy count — tùy provider
- Field sẽ vary theo adapter_type

### Bước 3: Xác nhận đơn hàng

- Summary: product, config, total cost
- Balance check: disabled nếu không đủ
- Nút "Place Order" → `POST /api/v1/proxy/orders`

### Submit

```
POST /api/v1/proxy/orders
Body: {
  product_id: "<uuid>",
  quantity: 1,
  metadata: { country: "US", isp_id: "<uuid>|null", period_months: 1 },
  idempotency_key: crypto.randomUUID()
}
```

Response:
- `status = processing` → Toast "Order submitted, proxy is being activated" (Proxy-Cheap async)
- `status = active` → Toast "Proxy activated!" + hiện credentials
- Error → Toast error message

---

## Credentials Feature

- Click "View" → `GET /api/v1/proxy/orders/{id}/credentials`
- Hiện: `username:password@host:port` trong mono font
- Copy button: copy toàn bộ string vào clipboard → Toast "Copied!"
- "Hide" → ẩn lại

> Nếu order đang ở trạng thái `processing`, nút "View" bị disabled với tooltip "Proxy is being activated".

---

## Renew Order (Gia hạn)

> Áp dụng cho providers tính theo tháng (vd: Proxy-Cheap).

- Hiển thị badge "Expiring soon" khi còn ≤ 7 ngày
- Click "Renew" → modal xác nhận:
  - Proxy info (IP, location, ISP)
  - Period: `N months`
  - Cost preview
  - Nút "Renew" → `POST /api/v1/proxy/orders/{id}/renew`
- Sau renew: `ExpiresAt` được extend thêm N tháng

> **Proxy-Cheap**: gia hạn qua `POST /proxies/:proxyId/extend-period` với `{ periodInMonths: N }`.

---

## Cancel Order

- Điều kiện: `status ∈ { active, pending, processing }`
- Click Cancel → `useConfirm({ variant: 'danger' })`
- Confirm: `DELETE /api/v1/proxy/orders/{id}`
- Refund tự động qua billing service

---

## Auto-refresh

`refetchInterval: 15000` (15 giây) — để detect khi order processing → active.

---

## API Calls

```
GET    /api/v1/proxy/products?proxy_type=&protocol=
GET    /api/v1/proxy/service-options?service_id=X&plan_id=Y   (countries + ISPs cho order form)
GET    /api/v1/proxy/orders?page=N&limit=20
POST   /api/v1/proxy/orders
POST   /api/v1/proxy/orders/{id}/renew
GET    /api/v1/proxy/orders/{id}/credentials
DELETE /api/v1/proxy/orders/{id}
```

---

## File

`frontend/src/app/dashboard/proxy/page.tsx`
