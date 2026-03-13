# Tài liệu Thiết kế Frontend — Index

> Mỗi file là tài liệu thiết kế cho 1 trang frontend của Cloudmini.  
> Xem tổng quan system design tại [`../20-FRONTEND-DESIGN.md`](../20-FRONTEND-DESIGN.md)

---

## Public Pages

| File | Routes | Mô tả |
|------|--------|-------|
| [auth-pages.md](./auth-pages.md) | `/login` · `/register` · `/forgot-password` | Đăng nhập, đăng ký, quên mật khẩu |

## Shared Components

| File | Mô tả |
|------|-------|
| [sidebar-navigation.md](./sidebar-navigation.md) | Sidebar nav — dark+gold theme, menu theo role, active state logic |

---

## User Pages

| File | Route | Mô tả |
|------|-------|-------|
| [dashboard.md](./dashboard.md) | `/dashboard` | Tổng quan tài khoản |
| [dashboard-proxy.md](./dashboard-proxy.md) | `/dashboard/proxy` | Proxy Orders + mua proxy |
| [dashboard-vps.md](./dashboard-vps.md) | `/dashboard/vps` | VPS Instances + deploy |
| [dashboard-wallet.md](./dashboard-wallet.md) | `/dashboard/wallet` | Ví & lịch sử giao dịch |
| [dashboard-profile.md](./dashboard-profile.md) | `/dashboard/profile` | Cài đặt tài khoản, 2FA |

---

## Reseller Pages

| File | Route | Mô tả |
|------|-------|-------|
| [reseller.md](./reseller.md) | `/reseller` | Dashboard reseller |
| [reseller-accounts.md](./reseller-accounts.md) | `/reseller/accounts` | Sub-Account management |
| [reseller-pricing.md](./reseller-pricing.md) | `/reseller/pricing` | Đặt giá bán proxy |
| [reseller-api-keys.md](./reseller-api-keys.md) | `/reseller/api-keys` | API Keys |
| [reseller-webhooks.md](./reseller-webhooks.md) | `/reseller/webhooks` | Webhook endpoints |

---

## Admin Pages

> Chỉ accessible với role `admin` / `super_admin`

| File | Route | Mô tả |
|------|-------|-------|
| [admin.md](./admin.md) | `/admin` | Admin dashboard + approve/suspend resellers |
| [admin-users.md](./admin-users.md) | `/admin/users` | Quản lý toàn bộ users |
| [admin-resellers.md](./admin-resellers.md) | `/admin/resellers` | Danh sách + quản lý resellers |
| [admin-proxy.md](./admin-proxy.md) | `/admin/proxy` | Proxy Products (tạo/toggle) |
| [admin-proxy-providers.md](./admin-proxy-providers.md) | `/admin/proxy/providers` | Proxy Providers (read-only) |
| [admin-vps.md](./admin-vps.md) | `/admin/vps` | VPS Plans (tạo/toggle) |
| [admin-logs.md](./admin-logs.md) | `/admin/logs` | Audit Logs + realtime stream |

---

## Cấu trúc Sidebar theo Role

```
admin:    OVERVIEW(Dashboard) → MANAGEMENT(Users, Resellers, Proxy Products, Proxy Providers, VPS Plans, Audit Logs) → SERVICES(Proxy Orders, VPS Instances) → MY ACCOUNT(Wallet, Profile)
reseller: OVERVIEW(Dashboard) → MANAGEMENT(Sub-Accounts, Pricing) → DEVELOPER(API Keys, Webhooks)
user:     MAIN(Dashboard) → SERVICES(Proxy Orders, VPS Instances) → ACCOUNT(Wallet, Profile)
```

---

## Design System

| Thuộc tính | Giá trị |
|-----------|---------|
| Framework | Next.js 14 App Router |
| Font | Public Sans |
| Primary color | `#7367F0` (purple) |
| Layout wrapper | `<AppLayout>` (Sidebar + Topbar) |
| State mgt | Zustand (`useAuthStore`) + React Query |
| Toast | `useToast()` |
| Confirm dialog | `useConfirm()` |
| Pagination | `<Pagination>` component |
