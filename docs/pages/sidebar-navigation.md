# Sidebar Navigation

> Component dùng chung cho tất cả logged-in pages (user / reseller / admin)

**File:** `frontend/src/components/layout/Sidebar.tsx`

---

## Design Theme — Dark + Gold (Hetzner-inspired)

| CSS Variable | Giá trị | Dùng cho |
|---|---|---|
| `--dc-dark` | `#1C1C1E` | Sidebar background |
| `--dc-gold` | `#E6A817` | Logo icon, active item, avatar, role badge |
| `--dc-gold-text` | `#1C1C1E` | Text trên nền gold |

### Visual Structure

```
┌──────────────────────────┐  bg: #1C1C1E
│ ☁  Cloudmini             │  logo-icon: gold
├──────────────────────────┤  separator: rgba(255,255,255,.07)
│  MAIN          ← label   │  font: .67rem uppercase, mờ 28%
│    Dashboard             │  nav-item: rgba(white,.62)
│                          │
│  SERVICES                │
│  ▌ Proxy Orders ←active  │  left border 3px gold + bg gold 12%
│    VPS Instances         │
│                          │
│  ACCOUNT                 │
│    Wallet                │
│    Profile               │
│                          │
│        (space)           │
├──────────────────────────┤  separator: rgba(255,255,255,.07)
│ [A] An        [logout→]  │  avatar: gold bg, name: white, role: gold
└──────────────────────────┘
```

---

## CSS Classes

| Class | Mô tả |
|---|---|
| `.sidebar` | Container chính, `bg: var(--dc-dark)`, `box-shadow: 2px 0 20px rgba(0,0,0,.35)` |
| `.sidebar-logo` | Logo row, border-bottom mờ |
| `.logo-icon` | 34×34, `bg: var(--dc-gold)`, border-radius 8px |
| `.logo-text` | `color: #fff`, weight 800 |
| `.nav-group-label` | Section header: `.67rem`, uppercase, `color: rgba(white,.28)` |
| `.sidebar-nav` | `display: flex; flex-direction: column; gap: 1px` |
| `.nav-item` | `color: rgba(white,.62)`, hover: `rgba(white,.07)` bg |
| `.nav-item-active` | `bg: rgba(gold,.12)`, `color: var(--dc-gold)`, `::before` left border 3px gold |
| `.sidebar-bottom` | `margin-top: auto`, border-top mờ |
| `.sidebar-user` | User row: `bg: rgba(white,.06)`, hover `rgba(white,.1)` |
| `.user-avatar` | 34×34 circle, `bg: var(--dc-gold)` |
| `.user-name` | `color: #fff`, weight 600 |
| `.user-role` | `color: var(--dc-gold)`, capitalize |

---

## Menu Items theo Role

### User (`role: user`)

| Group | Label | Route |
|---|---|---|
| MAIN | Dashboard | `/dashboard` |
| SERVICES | Proxy Orders | `/dashboard/proxy` |
| SERVICES | VPS Instances | `/dashboard/vps` |
| ACCOUNT | Wallet | `/dashboard/wallet` |
| ACCOUNT | Profile | `/dashboard/profile` |

### Reseller (`role: reseller`)

| Group | Label | Route |
|---|---|---|
| OVERVIEW | Dashboard | `/reseller` |
| MANAGEMENT | Sub-Accounts | `/reseller/accounts` |
| MANAGEMENT | Pricing | `/reseller/pricing` |
| DEVELOPER | API Keys | `/reseller/api-keys` |
| DEVELOPER | Webhooks | `/reseller/webhooks` |

### Admin (`role: admin` / `super_admin`)

| Group | Label | Route |
|---|---|---|
| OVERVIEW | Dashboard | `/admin` |
| MANAGEMENT | Users | `/admin/users` |
| MANAGEMENT | Resellers | `/admin/resellers` |
| MANAGEMENT | Proxy Products | `/admin/proxy` |
| MANAGEMENT | Proxy Providers | `/admin/proxy/providers` |
| MANAGEMENT | VPS Plans | `/admin/vps` |
| MANAGEMENT | Audit Logs | `/admin/logs` |
| SERVICES | Proxy Orders | `/dashboard/proxy` |
| SERVICES | VPS Instances | `/dashboard/vps` |
| MY ACCOUNT | Wallet | `/dashboard/wallet` |
| MY ACCOUNT | Profile | `/dashboard/profile` |

---

## Active State Logic

```ts
// Root-only routes (exact match):
const isRootOnly = ['/admin', '/dashboard', '/reseller', '/admin/proxy'].includes(href)

const active = pathname === href ||
  (!isRootOnly && href !== '/' && pathname?.startsWith(href + '/'))
```

> `/admin/proxy` chỉ active khi `pathname === '/admin/proxy'`, không active khi vào `/admin/proxy/providers`.

---

## User Section (Bottom)

- **Avatar:** 2 chữ cái đầu của `fullName`, hoặc ký tự đầu `email`
- **Name:** `user.fullName || user.email`
- **Role badge:** `user.role` (capitalize)
- **Logout button:** icon `LogOut`, hover → đỏ `#ea5455`

---

## Responsive

- Desktop: sidebar sticky, `height: 100vh`
- Mobile (`≤768px`): sidebar ẩn (`display: none`), toggle bằng `mobile-open` class
- Mobile open: `position: fixed`, `left: 0`, `box-shadow: var(--shadow-lg)`
- Backdrop overlay khi mobile open: `.sidebar-overlay.visible`
