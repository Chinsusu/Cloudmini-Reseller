# Trang: `/admin/users` — Quản lý Người dùng

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Admin xem và quản lý toàn bộ user accounts trong hệ thống: thay đổi role, suspend/activate, sửa profile, và tắt 2FA khẩn cấp.

---

## Layout

```
┌──────────────────────────────────────────────────┐
│  Header: "Users"              [Search input]     │
├──────────────────────────────────────────────────┤
│  Table: danh sách users (paginated 15/page)      │
├──────────────────────────────────────────────────┤
│  Pagination                                      │
└──────────────────────────────────────────────────┘
```

---

## Users Table

| Column | Mô tả |
|--------|-------|
| Email | Email + ID sub-text (mono) |
| Full Name | Họ tên đầy đủ |
| Role | Badge: super_admin (red) / admin (purple) / reseller (blue) / user (grey) |
| Status | Badge: active (green) / suspended (red) / pending (yellow) |
| Created | Date |
| Actions | Edit Profile / Change Role / Suspend / Activate / Disable 2FA / Delete |

---

## Actions

### Edit Profile

- Modal: sửa Full Name + Phone
- `PUT /api/v1/admin/users/{id}/profile   { full_name, phone }`

### Change Role

- Dropdown select: user / reseller / admin / super_admin
- `useConfirm` nếu promote lên admin
- `PUT /api/v1/admin/users/{id}/role   { role }`

### Suspend / Activate

- Toggle giữa `active` và `suspended`
- `PUT /api/v1/admin/users/{id}/status   { status }`

### Disable 2FA (Emergency)

- Chỉ hiện nếu user có 2FA enabled
- `useConfirm({ variant: 'danger', message: 'This will disable 2FA immediately' })`
- `PUT /api/v1/admin/users/{id}/2fa/disable`

### Delete User

- `useConfirm({ variant: 'danger', message: 'This is permanent and cannot be undone' })`
- `DELETE /api/v1/admin/users/{id}`
- Không thể xóa super_admin

---

## Search

- Input realtime search (debounced 300ms)
- Tìm theo: email, full_name
- `GET /api/v1/admin/users?search=...&page=N&limit=15`

---

## API Calls

```
GET    /api/v1/admin/users?page=N&limit=15&search=
PUT    /api/v1/admin/users/{id}/profile   { full_name, phone }
PUT    /api/v1/admin/users/{id}/role      { role }
PUT    /api/v1/admin/users/{id}/status    { status }
PUT    /api/v1/admin/users/{id}/2fa/disable
DELETE /api/v1/admin/users/{id}
```

---

## File

`frontend/src/app/admin/users/page.tsx`
