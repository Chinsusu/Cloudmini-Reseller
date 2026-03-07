# Cloudmini — Tài liệu Thiết kế Frontend

> Phiên bản: v0.6.x · Cập nhật: 2026-03-07

---

## Hệ thống Design

| Thuộc tính | Giá trị |
|-----------|---------|
| Framework | Next.js 14 App Router |
| Font | Public Sans |
| Primary color | `#7367F0` (purple) |
| Layout | `<AppLayout>` (Sidebar + Topbar) |
| State | Zustand (`useAuthStore`) + React Query |
| Notifications | `useToast()` / `useConfirm()` |
| Pagination | `<Pagination>` component |

---

## Auth Pages

### `/login`
**Mục đích:** Đăng nhập tài khoản

| Mục | Chi tiết |
|-----|---------|
| Inputs | Email, Password |
| Submit | `POST /api/v1/auth/login` → nhận `access_token` + `refresh_token` |
| Success | Redirect theo role: `admin` → `/admin`, `reseller` → `/reseller`, `user` → `/dashboard` |
| Lưu trữ | Token lưu vào Zustand `useAuthStore` + localStorage |

---

### `/register`
**Mục đích:** Tạo tài khoản mới (role mặc định: `user`)

| Mục | Chi tiết |
|-----|---------|
| Inputs | Full name, Email, Password |
| Submit | `POST /api/v1/auth/register` |
| Success | Toast + redirect về `/login` |

---

## User Pages (role: `user`)

### `/dashboard`
**Mục đích:** Tổng quan tài khoản

**Components:**
- 3 stat cards: Wallet Balance · Proxy Orders · VPS Instances
- Recent Transactions table (5 rows, không pagination)

**APIs:**
```
GET /api/v1/billing/wallet      → balance
GET /api/v1/proxy/orders        → count
GET /api/v1/vps/instances       → count
GET /api/v1/billing/transactions?limit=5
```

---

### `/dashboard/proxy`
**Mục đích:** Quản lý Proxy Orders + mua proxy mới

**Layout:** Header với 2 nút (Refresh, Buy Proxy) + Bảng + Pagination

**Buy Proxy Modal:**
- Filter bar: proxy_type (residential/datacenter/mobile) · protocol (http/socks5) · location
- Product cards grid: name · type badge · protocol badge · location · duration · bandwidth · price
- Order summary: chọn product → nhập quantity → total preview
- Submit: `POST /api/v1/proxy/orders` với `idempotency_key = crypto.randomUUID()`

**Columns bảng:** Order # · Qty · Amount · Expires · Status badge · Credentials · Actions

**Tính năng:**
- **Credentials toggle:** Click "View" → `GET /api/v1/proxy/orders/{id}/credentials` → hiện `username:***` + Copy button. Click Hide để ẩn.
- **Copy:** Copy `user:pass@host:port` vào clipboard → Toast
- **Cancel:** chỉ khi `status = active/pending` → `useConfirm` → `DELETE` → refetch
- **Auto-refresh:** `refetchInterval: 15000`

**APIs:**
```
GET    /api/v1/proxy/products?proxy_type=&protocol=&location=
GET    /api/v1/proxy/orders?page=N&limit=20
POST   /api/v1/proxy/orders   {product_id, quantity, idempotency_key}
GET    /api/v1/proxy/orders/{id}/credentials
DELETE /api/v1/proxy/orders/{id}
```

**Status colors:** active → green · pending/processing → yellow · cancelled/failed → red · expired → grey

---

### `/dashboard/vps`
**Mục đích:** Quản lý VPS Instances + deploy VPS mới

**Layout:** Header với Deploy VPS button + Bảng + Pagination

**Deploy VPS Modal:**
- Plans grid: mỗi card hiển thị name · CPU · RAM · Disk · monthly_rate
- Hostname input: lowercase + alphanumeric + dash (3-63 chars), client-side regex validation
- Price summary: monthly + hourly breakdown
- Submit: `POST /api/v1/vps/orders` → response 202 → Toast "VPS is being provisioned"

**Columns bảng:** Hostname (+ VMID sub) · Plan ID · IP Address · Status

**Actions per instance:**
| Action | Condition | API |
|--------|-----------|-----|
| Start | status = stopped | `POST /vps/instances/{id}/start` |
| Stop | status = running | `POST /vps/instances/{id}/stop` |
| Reboot | status = running | `POST /vps/instances/{id}/reboot` |
| Console | running/stopped | `GET /vps/instances/{id}/console` → open URL |
| Terminate | not terminated | `useConfirm` → `DELETE /vps/instances/{id}` |

**Auto-refresh:** `refetchInterval: 10000` (10s)

**APIs:**
```
GET  /api/v1/vps/plans
GET  /api/v1/vps/instances?page=N
POST /api/v1/vps/orders   {plan_id, hostname, idempotency_key}
```

---

### `/dashboard/wallet`
**Mục đích:** Quản lý ví và nạp tiền

**Layout:**
1. 3 stat cards: Total Balance · Available · On Hold
2. Top-up form
3. Transaction history table

**Stat cards:**
- **Total:** `wallet.balance`
- **Available:** `balance - hold_amount`
- **On Hold:** `wallet.hold_amount` (reserved for active orders)

**Top-up form:**
- Amount input (min $5)
- Payment method select: Bank Transfer / Crypto (USDT/USDC) / Stripe / VNPay
- Submit → `POST /api/v1/billing/wallet/topup` → toast success

**Transaction table:**
| Column | Chi tiết |
|--------|---------|
| Date | Formatted `MMM DD, YYYY` |
| Type | Badge (deposit / order_refund / hold / debit...) |
| Description | Free text |
| Amount | + green (credit) / − red (debit) |
| Balance After | Running balance |
| Status | Badge |

**APIs:**
```
GET  /api/v1/billing/wallet
GET  /api/v1/billing/transactions?page=N&limit=20
POST /api/v1/billing/wallet/topup
```

---

### `/dashboard/profile`
**Mục đích:** Cài đặt tài khoản cá nhân

**Layout:** 3 cards xếp dọc

**Card 1 — Account Information:**
- Avatar circle (initials, gradient purple)
- Full name, Email, Role badge, Status badge
- Member since date

**Card 2 — Change Password:**
- 3 inputs ngang: Current · New · Confirm
- Client-side validation: match check, min 8 chars
- Submit → `POST /api/v1/auth/password` → `useToast`

**Card 3 — Active Sessions:**
- Bảng: Device/UA · IP · Last Active · Revoke button
- Revoke → `useConfirm` → `DELETE /api/v1/auth/sessions/{id}`

**APIs:**
```
GET    /api/v1/auth/me
GET    /api/v1/auth/sessions
POST   /api/v1/auth/password   {old_password, new_password}
DELETE /api/v1/auth/sessions/{id}
```

---

## Reseller Pages (role: `reseller`)

### `/reseller`
**Mục đích:** Dashboard tổng quan reseller

**Layout:** Stats grid + Quick links grid

**Stats (4 cards):**
- Sub-Accounts count
- Pricing Rules count (custom overrides)
- API Keys count
- Commission % (từ reseller account)

**Quick Links (4 cards):**
- Pricing Management → `/reseller/pricing`
- API Keys → `/reseller/api-keys`
- Webhooks → `/reseller/webhooks`
- Sub-Accounts → `/reseller/accounts`

**APIs:**
```
GET /api/v1/reseller/dashboard
GET /api/v1/reseller/users
GET /api/v1/reseller/pricing
GET /api/v1/reseller/api-keys
```

---

### `/reseller/accounts`
**Mục đích:** Quản lý sub-accounts (user accounts nằm dưới reseller)

**Layout:** Add form + bảng

**Add Sub-Account form:**
- User ID input (UUID của user cần thêm)
- Credit Limit input ($)
- Submit → `POST /api/v1/reseller/users`

**Bảng:** User ID · Credit Limit · Added date

> **Note:** Reseller phải ở trạng thái `approved` mới tạo được sub-account.

**APIs:**
```
GET  /api/v1/reseller/users?page=N&limit=20
POST /api/v1/reseller/users  {user_id, credit_limit}
```

---

### `/reseller/pricing`
**Mục đích:** Đặt giá bán riêng cho từng proxy product

**Layout:** Bảng với inline edit

**Columns:**
| Column | Chi tiết |
|--------|---------|
| Product | Tên sản phẩm |
| Type | Badge: datacenter / residential |
| Location | Country/region |
| Floor Price | Giá tối thiểu (không thể bán dưới mức này) — màu đỏ |
| Sell Price | Giá bán ra — click để edit inline |
| Markup | `(sell - floor) / floor × 100%` — tự tính, màu xanh/đỏ |
| Actions | Edit / Save / Cancel |

**Inline edit flow:**
1. Click "Edit" hoặc click trực tiếp vào số
2. Input xuất hiện in-place
3. Validation: `sell_price >= floor_price`
4. Click "Save" → `PUT /api/v1/reseller/pricing/{product_id}`
5. Toast success/error

**APIs:**
```
GET /api/v1/reseller/pricing
PUT /api/v1/reseller/pricing/{product_id}  {sell_price}
```

---

### `/reseller/api-keys`
**Mục đích:** Tạo và quản lý API keys cho tích hợp programmatic

**Layout:** Create form + bảng

**Create flow:**
1. Nhập key name
2. Click "Generate" → `POST /api/v1/reseller/api-keys`
3. **Banner one-time:** Hiện plaintext key với nút Copy. Banner tự ẩn sau khi dismiss. Key không bao giờ hiện lại.

**Bảng:** Name · Prefix (8 chars) · Scopes · Last Used · Expires · Status · Revoke

**Revoke:** `useConfirm` → `DELETE /api/v1/reseller/api-keys/{id}`

**APIs:**
```
GET    /api/v1/reseller/api-keys
POST   /api/v1/reseller/api-keys  {name, scopes}
DELETE /api/v1/reseller/api-keys/{id}
```

---

### `/reseller/webhooks`
**Mục đích:** Cấu hình HTTP endpoints nhận event notifications

**Layout:** Create form + bảng

**Create form:**
- Endpoint URL input
- Signing Secret (HMAC-SHA256, optional)
- Event picker: toggle buttons cho 9 event types:
  - `order.created`, `order.status_changed`, `order.cancelled`
  - `payment.completed`, `payment.failed`
  - `vps.created`, `vps.status_changed`
  - `reseller.approved`, `reseller.suspended`

**Bảng:** Endpoint URL · Status (Active/Inactive) · Created · Delete

**APIs:**
```
GET    /api/v1/reseller/webhooks
POST   /api/v1/reseller/webhooks  {url, secret, events[]}
DELETE /api/v1/reseller/webhooks/{id}
```

---

## Admin Pages (role: `admin` / `super_admin`)

### `/admin`
**Mục đích:** Admin dashboard + quản lý resellers

**Layout:** Stats + Reseller management table

**Stats (2 cards):** Total Users · Total Resellers

**Reseller table:** Company · Email · Status · Commission · Created · Actions

**Actions:**
- Approve (chỉ hiện khi `status = pending`) → `PUT /api/v1/admin/resellers/{id}/approve`
- Suspend (chỉ hiện khi `status = approved`) → `useConfirm` với reason input → `PUT /api/v1/admin/resellers/{id}/suspend`

**APIs:**
```
GET /api/v1/admin/stats
GET /api/v1/admin/resellers?status=pending
PUT /api/v1/admin/resellers/{id}/approve
PUT /api/v1/admin/resellers/{id}/suspend
```

---

### `/admin/users`
**Mục đích:** Danh sách tất cả users trong hệ thống

**Layout:** Search + bảng paginated

**Columns:** Email · Full Name · Role badge · Status badge · Created · Actions

**APIs:**
```
GET /api/v1/admin/users?page=N&search=...
```

---

### `/admin/proxy`
**Mục đích:** Quản lý proxy products (admin tạo và bật/tắt sản phẩm)

**Layout:** Header với Add Product button + Bảng

**Add Product Modal:**
- Name, proxy_type (select), protocol (select), location
- base_cost, duration_days (optional), bandwidth_gb (optional)
- Provider ID (UUID)

**Columns bảng:** Name · Type badge · Protocol badge · Location · Duration · Bandwidth · Cost · Status toggle

**Status toggle:** Click Toggle button → `PUT /api/v1/admin/proxy/products/{id}/toggle` → icon đổi màu (green/grey)

**APIs:**
```
GET  /api/v1/admin/proxy/products?page=N&limit=20
POST /api/v1/admin/proxy/products   {name, proxy_type, protocol, location, base_cost, ...}
PUT  /api/v1/admin/proxy/products/{id}/toggle
```

---

### `/admin/vps`
**Mục đích:** Quản lý VPS plans (admin tạo và bật/tắt plan)

**Layout:** Header với Add Plan button + Bảng

**Add Plan Modal:**
- Name, slug, cpu_cores, ram_mb, disk_gb
- monthly_rate, hourly_rate

**Columns bảng:** Name (+ slug sub) · CPU · RAM · Disk · Monthly · Hourly · Status toggle

**RAM hiển thị:** `ram_mb / 1024` GB

**APIs:**
```
GET  /api/v1/admin/vps/plans
POST /api/v1/admin/vps/plans   {name, slug, cpu_cores, ram_mb, disk_gb, hourly_rate, monthly_rate}
PUT  /api/v1/admin/vps/plans/{id}/toggle
```

---

## Navigation theo Role

```
admin:    OVERVIEW(Dashboard) → MANAGEMENT(Users, Resellers, Proxy Products, VPS Plans) → SERVICES(Proxy, VPS) → MY ACCOUNT(Wallet, Profile)
reseller: OVERVIEW(Dashboard) → MANAGEMENT(Sub-Accounts, Pricing) → DEVELOPER(API Keys, Webhooks)
user:     MAIN(Dashboard) → SERVICES(Proxy Orders, VPS Instances) → ACCOUNT(Wallet, Profile)
```
