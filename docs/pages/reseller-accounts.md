# Trang: `/reseller/accounts` — Sub-Account Management

> Role: **reseller**

---

## Mục đích

Reseller thêm user vào danh sách sub-account và quản lý credit limit của họ.

---

## Layout

```
┌──────────────────────────────────────────┐
│  Header: "Sub-Accounts"                  │
├──────────────────────────────────────────┤
│  Add Sub-Account Form                    │
├──────────────────────────────────────────┤
│  Table: danh sách sub-accounts           │
└──────────────────────────────────────────┘
```

---

## Add Sub-Account Form

| Field | Loại | Validation |
|-------|------|-----------|
| User ID | text (UUID) | UUID format hợp lệ |
| Credit Limit ($) | number | Min $0, max tùy config reseller |

Submit → `POST /api/v1/reseller/users`

> **Note:** User đó phải đã có tài khoản trong hệ thống. Reseller không tự tạo user mới ở đây — chỉ link user có sẵn.

---

## Sub-Accounts Table

| Column | Mô tả |
|--------|-------|
| User ID | UUID dạng mono font |
| Email | Email của user (nếu có) |
| Credit Limit | `$X.XX` |
| Added | Date added `DD MMM YYYY` |

---

## API Calls

```
GET  /api/v1/reseller/users?page=N&limit=20
POST /api/v1/reseller/users   { user_id, credit_limit }
```

---

## File

`frontend/src/app/reseller/accounts/page.tsx`
