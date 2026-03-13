# Cloudmini — Tài liệu Thiết kế Frontend

> Phiên bản: v0.9.x · Cập nhật: 2026-03-13

---

## Hệ thống Design

| Thuộc tính | Giá trị |
|-----------|---------|
| Framework | Next.js 14 App Router |
| Font | Public Sans (Google Fonts) |
| State | Zustand (`useAuthStore`) + React Query |
| Notifications | `useToast()` / `useConfirm()` |
| Pagination | `<Pagination>` component |
| Icons | `lucide-react` |

---

## Dark Theme — Hetzner-inspired

> Tham khảo: Hetzner Console dark mode

### Hierarchy màu sắc

```
┌─────────────────── BACKGROUND ───────────────────────┐  #111316  (tối nhất - canvas)
│  ┌─── SIDEBAR / TOPBAR / CARD ─────────────────────┐ │  #1C1D21  (panel nổi trên bg)
│  │  ┌── RAISED (modal, dropdown, hover) ─────────┐ │ │  #252629  (nổi nhất)
│  │  └────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
```

### CSS Variables — Layout & Color

| Variable | Hex | Dùng cho |
|---|---|---|
| `--bg` | `#111316` | Canvas background — tối nhất |
| `--surface` | `#1C1D21` | Card, sidebar, topbar, panel |
| `--surface-raised` | `#252629` | Modal, dropdown, hover state |
| `--sidebar-bg` | `#1C1D21` | Sidebar (= surface) |
| `--border` | `rgba(255,255,255,.09)` | Viền phân cách |
| `--border-light` | `rgba(255,255,255,.05)` | Viền nhẹ hơn |
| `--text` | `#ADADB8` | Body text |
| `--text-heading` | `#E4E4E8` | Heading, số liệu quan trọng |
| `--text-muted` | `rgba(255,255,255,.38)` | Placeholder, label, muted |
| `--shadow-sm` | `0 2px 6px rgba(0,0,0,.45)` | Card shadow |
| `--shadow` | `0 4px 18px rgba(0,0,0,.55)` | Elevated shadow |

### CSS Variables — Brand & Accent

| Variable | Hex | Dùng cho |
|---|---|---|
| `--dc-gold` | `#E6A817` | CTA chính, active nav, avatar, link |
| `--dc-gold-dark` | `#C98F10` | Hover gold |
| `--dc-gold-text` | `#1C1C1E` | Text trên nền gold |
| `--primary` | `#7367F0` | Accent phụ (filter active, focus ring) |
| `--success` | `#28C76F` | |
| `--warning` | `#FF9F43` | |
| `--error` | `#EA5455` | |
| `--info` | `#00CFE8` | |

> **Lưu ý:** `--success-light`, `--error-light`, v.v. là `rgba()` để phù hợp dark bg.

---

## Layout — AppLayout

```
┌──────────┬────────────────────────────────────┐
│          │  Topbar (--surface, border-bottom)  │
│ Sidebar  ├────────────────────────────────────┤
│(260px)   │                                    │
│(--sidebar│   .page-main (--bg, padding 2rem)  │
│  -bg)    │                                    │
└──────────┴────────────────────────────────────┘
```

- `.page-layout`: `display: flex; min-height: 100vh`
- `.page-main`: `flex: 1; background: var(--bg); padding: 1.75rem 2rem`
- Sidebar: `position: sticky; top: 0; height: 100vh; overflow-y: auto`

---

## Sidebar Navigation

> File: `frontend/src/components/layout/Sidebar.tsx`
> Xem thêm: `docs/pages/sidebar-navigation.md`

### Style

| Element | Style |
|---|---|
| Background | `var(--sidebar-bg)` = `#1C1D21` |
| Logo icon | 34×34px, `bg: var(--dc-gold)`, radius 8px |
| Logo text | `color: #fff`, weight 800 |
| Section label | `.67rem`, uppercase, `color: rgba(white,.28)` |
| Nav item | `color: rgba(white,.62)`, hover: `rgba(white,.07)` bg |
| **Active item** | `bg: rgba(gold,.12)` + left border `3px var(--dc-gold)` + `color: var(--dc-gold)` |
| User avatar | 34×34 circle, `bg: var(--dc-gold)` |
| Logout hover | red `#ea5455` |

### Menu theo Role

```
user:     MAIN(Dashboard) → SERVICES(Proxy Orders, VPS Instances) → ACCOUNT(Wallet, Profile)
reseller: OVERVIEW(Dashboard) → MANAGEMENT(Sub-Accounts, Pricing) → DEVELOPER(API Keys, Webhooks)
admin:    OVERVIEW(Dashboard) → MANAGEMENT(Users, Resellers, Proxy Products, Proxy Providers, VPS Plans, Audit Logs) → SERVICES(Proxy, VPS) → MY ACCOUNT(Wallet, Profile)
```

---

## Auth Pages

> File: `docs/pages/auth-pages.md` (chi tiết đầy đủ)

### Style chung

```
[BG: /public/datacenter-bg.jpg + rgba(0,0,0,.45) blur]

      ┌────────────────────────┐
      │  WHITE   (card top)    │   max-width: 420px
      │  Title / Form / Button │   border-radius: 4px
      ├────────────────────────┤
      │  DARK    (card bottom) │   bg: var(--dc-dark) = #1A1B1E
      │  links / register btn  │
      └────────────────────────┘
```

| Element | Style |
|---|---|
| Form background | `#fff` |
| CTA button | `var(--dc-gold)`, text `#1C1C1E`, weight 800 |
| Input focus | border `var(--dc-gold)`, shadow `rgba(gold,.15)` |
| Card bottom | `var(--dc-dark)` = `#1A1B1E` |

### Routes

| Route | Mô tả |
|---|---|
| `/login` | Email + password → redirect theo role |
| `/register` | Full name + email + password + confirm |
| `/forgot-password` | Email → gửi link reset → success state |

---

## Components dùng chung

### Cards

```css
.card {
  background: var(--surface);    /* #1C1D21 */
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);  /* 16px */
  box-shadow: var(--shadow-sm);
}
```

### Stat Cards

- Top colored border `3px solid {accent}`
- Icon: 44×44, `bg: {accent}18` (18 opacity)
- Label: `.75rem` uppercase muted
- Value: `1.7rem` weight 800, `var(--text-heading)`

### Buttons

| Class | Dùng cho |
|---|---|
| `.btn-primary` | Purple primary |
| `.btn-secondary` | Muted secondary |
| `.btn-success` / `.btn-danger` | Status actions |
| `.dc-btn-primary` | **Auth pages** — gold CTA |
| `.dc-btn-outline` | **Auth pages** — viền trắng trên dark bg |

### Badges

| Class | Color |
|---|---|
| `.badge-success` | Green tint |
| `.badge-warning` | Orange tint |
| `.badge-danger` | Red tint |
| `.badge-info` | Cyan tint |
| Tất cả | `rgba()` bg để readable trên dark surface |

### Forms

```css
.input {
  background: var(--surface);   /* dark bg */
  border: 1px solid var(--border);
  color: var(--text-heading);
}
.input:focus { border-color: var(--primary); }
```

### Topbar

- `background: var(--surface)` — cùng tone với sidebar
- `border-bottom: 1px solid var(--border)`
- Search box: `bg: var(--bg)` (tối hơn topbar)
- Icon buttons: `bg: var(--bg)`, hover → primary accent

---

## User Pages (role: `user`)

### `/dashboard`

**Components:**
- System Announcements banner (dark gradient, `var(--dc-dark)`, gold border)
- 4 stat cards: Wallet Balance · Proxy Orders · VPS Instances · Account Status
- Recent Activities table (8 rows, View all link)

**APIs:**
```
GET /api/v1/billing/wallet
GET /api/v1/proxy/orders
GET /api/v1/vps/instances
GET /api/v1/billing/transactions?limit=8
```

---

### `/dashboard/proxy`

**Layout:** One-page catalog — chọn product → inline order panel

**Buy Proxy Flow:**
- Filter tabs (All / Residential / Datacenter / Mobile) — không có icon
- Product cards grid (Vuexy card style, selected: outline + bg tint)
- Order panel (slide in khi chọn): Country, ISP, Period (months), Quantity
- Submit → `POST /api/v1/proxy/orders`

**APIs:**
```
GET    /api/v1/proxy/products?proxy_type=
GET    /api/v1/proxy/orders?page=N&limit=20
POST   /api/v1/proxy/orders
GET    /api/v1/proxy/orders/{id}/credentials
DELETE /api/v1/proxy/orders/{id}
```

---

### `/dashboard/vps`

**Layout:** Page header (title + icon refresh **+** Deploy CTA) → instances table

**Deploy Modal:**
- Plan cards grid (`repeat(auto-fill, minmax(200px,1fr))`)
- **Selected plan card:** `border: 1px solid var(--dc-gold)` + `background: rgba(230,168,23,.07)` + shadow
- Price: `var(--dc-gold)` khi selected, `var(--text-heading)` khi không
- Summary box: `var(--surface-raised)` + `1px solid var(--border)`
- Hostname validate: lowercase/numbers/hyphens, 3-63 chars

**Instances table:**
- Status badge: `badge-{STATUS_COLOR[status]}`
  - `running` → `badge-success` + **pulse dot** (7px green circle)
  - `stopped` → `badge-secondary`
  - `provisioning/booting` → `badge-info`
  - `terminated` → `badge-terminated`
- Actions: Start (green) · Stop · Reboot · Console · Terminate (red)
- Auto-refresh: `refetchInterval: 10000`
- Refresh button: `topbar-icon-btn` (icon-only)

---

### `/dashboard/wallet`

**Layout:** 3 stat cards + top-up form + transaction table

**Stat cards — dark+gold pattern:**
| Card | `borderTop` | Icon color | Value color |
|---|---|---|---|
| Tổng số dư | `var(--dc-gold)` | `var(--dc-gold)` | `var(--dc-gold)` |
| Khả dụng | `var(--success)` | `var(--success)` | `var(--success)` |
| Tạm giữ | `var(--warning)` | `var(--warning)` | `var(--warning)` |

Card structure:
```
border-top: 3px solid {accent}
border-radius: 0 0 var(--radius-xl) var(--radius-xl)
font-size: 1.6rem · weight 800
```

**Top-up form:** Card với card-header. Amount input nhập VND. Payment method select.

**Transaction table:**
- Amount: `+`(green) / `-`(red) + `formatVND()`
- Balance After: `formatVND()`
- Status badge: `badge-success`

---

### `/dashboard/profile`

**Layout:** 2-column grid — Account Info card (left) + Security column (right)

**Account Info card:**
- Avatar: 52px circle, `linear-gradient(135deg, rgba(230,168,23,.25), rgba(230,168,23,.1))`, gold border + gold initial letter
- Avatar bên dưới hiển thị: full name + email + `badge-secondary` role tag
- Info rows: label (`--text-muted`) / value (`--text-heading` weight 600) separated by `border-bottom: 1px solid var(--border-light)`
- Edit mode: inline form fields

**Security column (2 cards):**

1. **Two-Factor Authentication card:**
   - Status box: `rgba(40,199,111,.06)` + green border (enabled) / dark tint + standard border (disabled)
   - Shield icon: green (enabled) / muted (disabled)
   - Disable button: `action-btn red`
   - Enable button: `btn-primary`

2. **Password card:**
   - Change Password button: gold outline (`rgba(230,168,23,.06)` bg, `rgba(230,168,23,.3)` border, `var(--dc-gold)` text)

---

## Reseller Pages (role: `reseller`)

> Stat cards dùng dark+gold `borderTop` pattern. QuickLink cards hover → gold border. Webhook event pills active → gold.

### `/reseller` — Dashboard

**Stat cards:**
| Card | `borderTop` |
|---|---|
| Sub-Accounts | `var(--dc-gold)` |
| Pricing Rules | `var(--success)` |
| API Keys | `var(--warning)` |
| Commission | `var(--info)` |

**QuickLink cards:** `border: 1px solid var(--border)`, hover → `borderColor: rgba(230,168,23,.4)` + `translateY(-2px)`. Icon box: `rgba(color,.1-.15)` tint background, icon = `var(--dc-gold)`.

---

### `/reseller/accounts` — Sub-Accounts

Add by User ID + Hạn mức tín dụng (VND). Credit limit: `formatVND()`. Yêu cầu reseller status = `approved`.

---

### `/reseller/pricing` — Pricing

Bảng inline edit. Validation: `sell_price >= floor_price`. Pháp: `formatVND()` cho floor + sell price.

---

### `/reseller/api-keys` — API Keys

- New key banner: `border-left: 4px solid var(--dc-gold)` gold alert box
- Key code: dark box `var(--bg)` + `var(--border)` border
- Copy button: `action-btn` + `color: var(--dc-gold)`
- Scope badges: `badge-secondary`
- Revoke button: `action-btn red`

---

### `/reseller/webhooks` — Webhooks

- Event pills toggle: active → `rgba(230,168,23,.15)` bg + `var(--dc-gold)` text + `rgba(230,168,23,.5)` border
- HMAC-SHA256 signing secret (password field)

---

---

## Admin Pages (role: `admin` / `super_admin`)

> Stat cards dùng dark+gold `borderTop` pattern (không dùng gradient icon). Icon bài trong ô màu trà. Avatar user dùng gold gradient circle.

### `/admin` — Admin Dashboard

**Stat cards — dark+gold pattern:**
| Card | `borderTop` | Icon color |
|---|---|---|
| Total Users | `var(--dc-gold)` | `var(--dc-gold)` |
| Resellers | `var(--success)` | `var(--success)` |
| Platform Status | `var(--info)` | `var(--info)` |
| Pending Approval | `var(--warning)` | `var(--warning)` |

**Resellers table:** Company/website 2-line cell · Status badge (`badge-approved/pending/suspended`) · Approve (green) + Suspend (red) action buttons.

---

### `/admin/users` — Users

**User table columns:** User · Role · Status · 2FA · Balance · Services · Last Login · Joined · Actions

**User cell:**
- Avatar: 34px gold gradient circle (`rgba(230,168,23,.2)` bg + `rgba(230,168,23,.3)` border), initials in `var(--dc-gold)`
- Full name (`var(--text-heading)` 600) + email (`var(--text-muted)` .78rem)

**Role badge:** `badge-{primary/info/warning/secondary}`

**Status badge:** `badge-{success/warning/error}`

**2FA badge:** `badge-success` / `badge-secondary`

**Balance column:**
- Loading: `…` in `var(--text-muted)`
- Value: `formatVND(balance)` — fontWeight 700, `var(--text-heading)`, .875rem
- No wallet: `—` in `var(--text-muted)`

**Services column (proxy · vps):**
- 🌐 proxy count — `var(--info)` if > 0, else muted
- · separator
- 🖥 vps count — `var(--success)` if > 0, else muted
- Combined badge `badge-secondary` if total > 0

**Row hover:** `background: rgba(255,255,255,.03)` — very subtle, Hetzner-style

**Actions:** Edit · Wallet (gold, `var(--dc-gold)`) · 2FA-off (warning, shown only if enabled) · Delete (red)

**Edit Modal:**
- Tabs (Info / Activity): active tab underline `var(--dc-gold)`, color `var(--dc-gold)`
- User header: gold avatar + `var(--bg)` box with `var(--border)` line
- Save button: `btn-primary` + `<Save>` icon

**TopUp Modal (`TopUpModal`):**
- Triggered by Wallet icon button (gold) in Actions column
- User info card: gold avatar + name/email in `var(--bg)` + `var(--border)` box
- Notice: `rgba(230,168,23,.08)` bg + `rgba(230,168,23,.25)` border — ghi rõ "admin adjustment — không tính vào doanh thu"
- Amount input: raw digits (`inputMode="numeric"`), formatted preview `= X₫` below in `var(--dc-gold)` 600
- Description field: optional
- Submit: `btn-primary` full-width

---

### `/admin/resellers` — Reseller Management

**Components:** `AppLayout` + `page-header` + paginated table

Status badge: `badge-success` (approved) / `badge-warning` (pending) / `badge-error` (suspended)

Action buttons: Approve `action-btn green` · Suspend `action-btn red` (with `useConfirm` dialog)

---

### `/admin/proxy` — Proxy Products

Add product modal + status toggle ToggleLeft/Right. Base cost: `formatVND()`.

---

### `/admin/proxy/providers` — Proxy Providers

Read-only. Adapter tag: colored inline badge (`{color}1a` bg). Status inline (CheckCircle2 green / XCircle muted). Info box: `var(--bg)` + `var(--border-light)` card.

---

### `/admin/vps` — VPS Plans

Add plan modal + ToggleLeft/Right. Monthly/hourly rate: `formatVND()`.

---

### `/admin/logs` — Audit Logs

**Layout:** `page-header` (title left, filter controls right)

Filters: Event type `select.input` + User ID `input` + `btn-secondary` Filter/Clear buttons.

Audit table: `<AuditLog>` component. Realtime stream via WebSocket.

---

---

## Token & Auth

| Cookie | Expires | Flag |
|---|---|---|
| `pvp_token` | 1 ngày | `sameSite: lax` — KHÔNG `secure` trên HTTP |
| `pvp_refresh` | 30 ngày | `sameSite: lax` |

Auto-refresh: axios interceptor gọi `POST /v1/auth/refresh` khi nhận 401. Fail → redirect `/login`.

Zustand persist key: `pvp-auth` (localStorage). Lưu `user + isAuthenticated`, token trong cookie.
