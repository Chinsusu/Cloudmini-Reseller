# Trang: `/reseller/api-keys` — Quản lý API Keys

> Role: **reseller**

---

## Mục đích

Reseller tạo và quản lý API keys để tích hợp programmatic (gọi API Cloudmini từ hệ thống của họ).

---

## Layout

```
┌──────────────────────────────────────────┐
│  Header: "API Keys"         [+ Generate] │
├──────────────────────────────────────────┤
│  [One-time Banner] Hiện plaintext key    │
├──────────────────────────────────────────┤
│  Table: danh sách API keys               │
└──────────────────────────────────────────┘
```

---

## Generate Key Flow

1. Click "Generate Key"
2. Modal mở: nhập **Key Name** (required)
3. Submit → `POST /api/v1/reseller/api-keys   { name, scopes: ['*'] }`
4. Response trả về `plaintext_key` (chỉ xuất hiện 1 lần)
5. **One-time Banner** xuất hiện trên đầu trang:
   - Background vàng / amber
   - Icon cảnh báo
   - Hiện `pvp_live_xxxx...xxxx` dạng monospace
   - Nút **Copy** → copy vào clipboard → Toast "Copied!"
   - Nút **Dismiss** → ẩn banner
   - Banner **không tự ẩn** — user phải tự dismiss
6. Sau khi dismiss: key không bao giờ hiện lại

---

## API Keys Table

| Column | Mô tả |
|--------|-------|
| Name | Tên key |
| Prefix | 8 ký tự đầu (dạng `pvp_live_XXXXXXXX...`) |
| Created | Date |
| Last Used | Date hoặc "Never" |
| Status | Badge: active / revoked |
| Actions | Revoke button |

### Revoke

- `useConfirm({ variant: 'danger', message: 'This will immediately invalidate the key' })`
- Confirm → `DELETE /api/v1/reseller/api-keys/{id}`
- Row cập nhật status → `revoked`

---

## API Calls

```
GET    /api/v1/reseller/api-keys
POST   /api/v1/reseller/api-keys   { name, scopes }
DELETE /api/v1/reseller/api-keys/{id}
```

---

## File

`frontend/src/app/reseller/api-keys/page.tsx`
