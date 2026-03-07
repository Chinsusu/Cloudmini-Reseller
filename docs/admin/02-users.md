# Admin Users — `/admin/users`

## Tổng quan

**URL:** `/admin/users`  
**Role yêu cầu:** `admin`, `super_admin`  
**Layout:** `<AppLayout>` với breadcrumb `[Admin > Users]`

Trang quản lý tất cả user accounts: xem danh sách, edit thông tin, đổi role/status, disable 2FA, xóa user.

---

## Layout

```
┌──────────────────────────────────────────────────────────────┐
│  Page Header                                                 │
│  "Users"   [N users total]                                   │
├──────────────────────────────────────────────────────────────┤
│  Users Table                                                 │
│  User | Role | Status | 2FA | Last Login | Joined | Actions │
│  ─────────────────────────────────────────────────────────── │
│  [Avatar+Name+Email] [badge] [badge] [badge] [date] [date]  │
│                              [Edit] [🛡️] [🗑️]               │
├──────────────────────────────────────────────────────────────┤
│  Pagination  (15 per page)                                   │
└──────────────────────────────────────────────────────────────┘
```

---

## Table Columns

| Column | Mô tả |
|--------|-------|
| User | Avatar initials + Full Name + Email |
| Role | badge: user / reseller / admin / super_admin |
| Status | badge: active / suspended / banned |
| 2FA | badge: `2FA On` (green) / `2FA Off` (grey) |
| Last Login | `last_login_at` formatted, `—` nếu chưa login |
| Joined | `created_at` formatted `MMM DD, YYYY` |
| Actions | Edit · Disable 2FA (chỉ khi 2FA On) · Delete |

**Role badge colors:**
| Role | Color |
|------|-------|
| super_admin | primary (purple) |
| admin | info (blue) |
| reseller | warning (orange) |
| user | secondary (grey) |

**Status badge colors:**
| Status | Color |
|--------|-------|
| active | success (green) |
| suspended | warning (yellow) |
| banned | error (red) |

---

## Actions

### Edit User Modal
Click nút **Edit** → modal mở với:
- **User info header** — avatar initials + tên + email
- **Full Name** — text input
- **Phone** — text input
- **Role** — select: `user` / `reseller` / `admin`
- **Status** — select: `active` / `suspended` / `banned`
- **Save Changes** — batch submit tất cả thay đổi song song

### Disable 2FA (row action)
- Nút 🛡️ màu vàng chỉ xuất hiện khi user có `totp_enabled=true`
- Click → `PUT /admin/users/{id}/2fa/disable` → toast + refresh
- Không cần confirm dialog (action có thể hoàn tác bằng user tự bật lại)

### Delete User
- Nút 🗑️ đỏ → `useConfirm` dialog: "Permanently delete ...?"
- Confirm → `DELETE /admin/users/{id}` (soft delete) → toast + refresh

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/admin/users?page=N&limit=15` | Danh sách users paginated |
| `PUT` | `/api/v1/admin/users/{id}/profile` | Cập nhật full_name, phone |
| `PUT` | `/api/v1/admin/users/{id}/role` | Đổi role |
| `PUT` | `/api/v1/admin/users/{id}/status` | Đổi status |
| `PUT` | `/api/v1/admin/users/{id}/2fa/disable` | Force disable 2FA |
| `DELETE` | `/api/v1/admin/users/{id}` | Soft delete user |

**Response format:**
```json
{
  "data": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "full_name": "Nguyen Van A",
      "phone": "+84...",
      "role": "user",
      "status": "active",
      "totp_enabled": false,
      "last_login_at": "2026-03-07T...",
      "created_at": "2026-01-15T08:00:00Z"
    }
  ],
  "meta": { "total": 1234, "page": 1, "limit": 15, "pages": 83 }
}
```

---

## State Management

| State | Triển khai |
|-------|-----------|
| Fetch | `useQuery(['admin-users', page])` |
| Delete | `useMutation → invalidateQueries` |
| Edit | `useMutation × 3` (profile, role, status) chạy song song |
| Disable 2FA | `async/await adminAPI.adminDisable2FA → invalidate` |

---

## States

| State | Hiển thị |
|-------|---------|
| Loading | Loading spinner |
| Empty | "No users found" empty state |
| Error | Toast error |

---

## Navigation
**Sidebar link:** MANAGEMENT → Users  
**Breadcrumb:** Admin > Users
