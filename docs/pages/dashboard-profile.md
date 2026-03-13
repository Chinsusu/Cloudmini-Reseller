# Trang: `/dashboard/profile` — Cài đặt Tài khoản

> Role: **user** (và các role khác khi truy cập từ sidebar)

---

## Mục đích

User quản lý thông tin cá nhân, đổi mật khẩu, bật/tắt 2FA, và quản lý phiên đăng nhập.

---

## Layout

3 cards xếp dọc:

```
┌──────────────────────────────────────┐
│  Card 1: Account Information         │
├──────────────────────────────────────┤
│  Card 2: Change Password             │
├──────────────────────────────────────┤
│  Card 3: Two-Factor Authentication   │
└──────────────────────────────────────┘
```

---

## Card 1 — Account Information

- **Avatar circle:** Initials (2 chữ đầu fullName), gradient purple background
- **Info hiển thị:**
  - Full Name (editable)
  - Email (read-only)
  - Phone (editable)
  - Role badge (read-only)
  - Status badge (read-only: active / suspended)
  - Member since: `DD MMM YYYY`
- **Edit mode:** Click "Edit" → text fields thành inputs → "Save" / "Cancel"
- Submit: `PUT /api/v1/users/me   { full_name, phone }`

---

## Card 2 — Change Password

| Field | Validation |
|-------|-----------|
| Current Password | Required |
| New Password | Min 8 chars, phải có chữ hoa + số |
| Confirm Password | Phải khớp với New Password |

Validation client-side realtime, submit disabled nếu fail.

Submit: `PUT /api/v1/users/me/password   { old_password, new_password }`

---

## Card 3 — Two-Factor Authentication (2FA)

### Khi 2FA chưa bật

- Button "Enable 2FA"
- Click → `POST /api/v1/users/me/2fa/setup` → nhận QR code + secret
- Modal hiện QR code để scan bằng Auth app
- Input OTP code → `POST /api/v1/users/me/2fa/enable   { code }`
- Thành công: Badge "2FA Enabled" thay button

### Khi 2FA đã bật

- Badge "2FA Active" (green)
- Button "Disable 2FA" → `useConfirm` → input OTP → `DELETE /api/v1/users/me/2fa   { code }`

---

## API Calls

```
GET    /api/v1/users/me
PUT    /api/v1/users/me           { full_name, phone }
PUT    /api/v1/users/me/password  { old_password, new_password }
POST   /api/v1/users/me/2fa/setup
POST   /api/v1/users/me/2fa/enable  { code }
DELETE /api/v1/users/me/2fa         { code }
```

---

## File

`frontend/src/app/dashboard/profile/page.tsx`
