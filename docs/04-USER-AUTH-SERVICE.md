# User & Auth Service — Service Design

**Document ID**: PVP-DOC-004  
**Version**: 1.0.0  
**Service**: user-service  
**Port**: 8081  

---

## 1. Responsibilities

- User registration và email verification
- Login / logout / token refresh
- Password reset flow
- Profile management
- JWT issuance và refresh token rotation
- API key management

---

## 2. API Endpoints

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/auth/register` | Đăng ký tài khoản |
| POST | `/api/v1/auth/login` | Đăng nhập, nhận JWT |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| POST | `/api/v1/auth/forgot-password` | Gửi email reset |
| POST | `/api/v1/auth/reset-password` | Đặt mật khẩu mới |
| POST | `/api/v1/auth/verify-email` | Xác nhận email |

### User Management

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/users/me` | Lấy thông tin profile |
| PUT | `/api/v1/users/me` | Cập nhật profile |
| PUT | `/api/v1/users/me/password` | Đổi mật khẩu |
| GET | `/api/v1/users/me/api-keys` | Danh sách API keys |
| POST | `/api/v1/users/me/api-keys` | Tạo API key mới |
| DELETE | `/api/v1/users/me/api-keys/:id` | Revoke API key |

### Admin

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/admin/users` | Danh sách users (paginated) |
| GET | `/api/v1/admin/users/:id` | Chi tiết user |
| PUT | `/api/v1/admin/users/:id/status` | Suspend/ban/activate |
| PUT | `/api/v1/admin/users/:id/role` | Thay đổi role |

---

## 3. JWT Design

### Access Token (15 minutes)
```json
{
  "sub": "user-uuid",
  "role": "user",
  "reseller_id": "reseller-uuid-or-null",
  "email": "user@example.com",
  "iat": 1704067200,
  "exp": 1704068100
}
```

### Refresh Token
- Random 256-bit token, stored as SHA256 hash in DB
- TTL: 7 days (user), 30 days (admin)
- Rotation on every use (refresh token rotation)
- Max 5 active sessions per user

---

## 4. Registration Flow

```
POST /auth/register
    │
    ▼
Validate email + password strength
    │
    ▼
Check email uniqueness
    │
    ▼
Hash password (bcrypt cost=12)
    │
    ▼
Create user record (status=pending_verification)
    │
    ▼
Generate verification token → store in Redis (TTL 24h)
    │
    ▼
Publish user.registered event → notification-service sends email
    │
    ▼
Response 201 (không trả JWT, chờ verify email)
```

---

## 5. Password Policy

- Minimum 8 characters
- At least 1 uppercase, 1 lowercase, 1 digit
- bcrypt cost factor: 12
- Brute force protection: lock account after 10 failed attempts (15 minutes)

---

## 6. Events Published

| Event | Trigger | Payload |
|---|---|---|
| `user.registered` | New registration | `{user_id, email}` |
| `user.verified` | Email verified | `{user_id}` |
| `user.login` | Successful login | `{user_id, ip}` |
| `user.password_changed` | Password reset | `{user_id}` |
| `user.suspended` | Admin action | `{user_id, reason}` |
