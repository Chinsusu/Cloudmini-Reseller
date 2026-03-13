# Auth Pages: Login / Đăng ký / Quên mật khẩu

> Role: **Public** (không cần đăng nhập)

---

## Design System

### Màu sắc (DC Auth Theme)

| CSS Variable | Giá trị | Dùng cho |
|---|---|---|
| `--dc-gold` | `#E6A817` | CTA button, link |
| `--dc-gold-dark` | `#C98F10` | Hover CTA |
| `--dc-gold-text` | `#1C1C1E` | Text trên button vàng |
| `--dc-dark` | `#1C1C1E` | Panel tối phần dưới card |
| `--dc-dark-2` | `#2A2A2D` | Hover nền tối |
| `--dc-text` | `#1C1C1E` | Text chính |
| `--dc-text-muted` | `#6B7280` | Placeholder, sub-text |
| `--dc-input-border` | `#D1D5DB` | Viền input |
| `--dc-input-focus` | `#E6A817` | Viền input khi focus |

### Layout chung

```
┌──────────────────────────────┐
│  [BG: datacenter-bg.jpg]     │
│  backdrop blur 3px           │
│                              │
│   ┌────────────────────┐     │
│   │  WHITE  (card-top) │     │
│   │  Title / Form      │     │
│   │  [GOLD BUTTON]     │     │
│   ├────────────────────┤     │
│   │  DARK  (card-bot)  │     │
│   │  forgot / register │     │
│   └────────────────────┘     │
└──────────────────────────────┘
```

- Background: `public/datacenter-bg.jpg` + `rgba(0,0,0,.45)` overlay
- Card: `border-radius: 4px`, `max-width: 420px`, không có box-shadow bên ngoài card
- Card top: `background: #fff`, padding `2rem 2.25rem`
- Card bottom: `background: var(--dc-dark)`, padding `1.25rem 2.25rem`

### CSS Classes

| Class | Dùng cho |
|---|---|
| `.dc-auth-page` | Wrapper fullscreen |
| `.dc-card` | Container chính |
| `.dc-card-top` | Phần trắng (form) |
| `.dc-card-bottom` | Phần tối (links) |
| `.dc-title` | Tiêu đề trang |
| `.dc-field` + `.dc-field-icon` | Input có icon trái |
| `.dc-input` | Input styled |
| `.dc-btn-primary` | Button vàng |
| `.dc-btn-outline` | Button viền trắng (dark section) |
| `.dc-link-forgot` | Link vàng |
| `.dc-error` | Error alert |

---

## 1. `/login` — Đăng nhập

**File:** `frontend/src/app/login/page.tsx`

### Fields

| Field | Type | Validation |
|---|---|---|
| Email or login | `email` | required, valid email |
| Password | `password` | min 6 chars |

### Logic

1. `POST /api/v1/auth/login` `{ email, password }`
2. Lưu `access_token` → Cookie `pvp_token` (1 ngày, `sameSite: lax`)
3. Lưu `refresh_token` → Cookie `pvp_refresh` (30 ngày)
4. `GET /api/v1/users/me` → lấy `full_name`
5. Redirect theo role:
   - `admin` / `super_admin` → `/admin`
   - `reseller` → `/reseller`
   - `user` → `/dashboard`

### Links trong card bottom

- **Forgot password?** → `/forgot-password`
- **REGISTER NOW** → `/register`

---

## 2. `/register` — Đăng ký

**File:** `frontend/src/app/register/page.tsx`

### Fields

| Field | Type | Validation |
|---|---|---|
| Full name | `text` | min 2 chars |
| Email | `email` | required, valid email |
| Password | `password` | min 8 chars |
| Confirm password | `password` | phải khớp password |

### Logic

1. `POST /api/v1/auth/register` `{ email, password, full_name }`
2. Response trả về `access_token`, `refresh_token`, `role`, `user_id`
3. Lưu tokens vào cookie (giống Login)
4. Redirect → `/dashboard` (luôn là role `user` khi tự đăng ký)

### Links trong card bottom

- **SIGN IN** → `/login`

---

## 3. `/forgot-password` — Quên mật khẩu

**File:** `frontend/src/app/forgot-password/page.tsx`

### Fields

| Field | Type | Validation |
|---|---|---|
| Email address | `email` | required, valid email |

### Logic

1. `POST /api/v1/auth/forgot-password` `{ email }`
2. Backend gửi email reset link
3. UI chuyển sang trạng thái **Success**: hiện checkmark + message hướng dẫn check email

### States

| State | Hiển thị |
|---|---|
| Default | Form nhập email |
| Loading | Button disabled + spinner |
| Success | ✅ Icon + "Check your email" message |
| Error | `.dc-error` banner |

### Links trong card bottom

- **← Back to login** → `/login`

---

## API Endpoints

```http
POST /api/v1/auth/login
Body: { email, password }
Response: { access_token, refresh_token, role, user_id }

POST /api/v1/auth/register
Body: { email, password, full_name }
Response: { access_token, refresh_token, role, user_id }

POST /api/v1/auth/forgot-password
Body: { email }
Response: { message: "Email sent" }

GET /api/v1/users/me
Header: Authorization: Bearer <token>
Response: { id, email, full_name, role, status }
```

---

## Token Storage

| Token | Cookie | Expires | SameSite |
|---|---|---|---|
| Access token | `pvp_token` | 1 ngày | `lax` |
| Refresh token | `pvp_refresh` | 30 ngày | `lax` |

> ⚠️ **Không dùng `secure: true`** khi chạy trên HTTP (local/LAN). Chỉ bật khi deploy với HTTPS.

Auto-refresh: khi nhận 401, axios interceptor tự gọi `POST /api/v1/auth/refresh` bằng `pvp_refresh` để lấy access token mới. Nếu refresh cũng fail → redirect `/login`.

---

## Zustand Store (`useAuthStore`)

```ts
// lib/store.ts
setUser(user: User, token: string, refresh: string)
clearAuth()

// State
user: { id, email, fullName, role }
isAuthenticated: boolean
```

Persist key: `pvp-auth` (localStorage). Lưu `user` và `isAuthenticated`, không lưu token trực tiếp (token nằm trong cookie).
