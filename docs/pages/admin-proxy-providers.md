# Trang: `/admin/proxy/providers` — Quản lý Proxy Providers

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Admin xem danh sách các provider adapter đã được đăng ký trong hệ thống. Đây là tầng dưới của Proxy Products — mỗi product phải gắn với 1 provider.

---

## Layout

```
┌──────────────────────────────────────────────────┐
│  Header: "Proxy Providers"                       │
│  Subtitle: "N active providers"                  │
├──────────────────────────────────────────────────┤
│  Table: danh sách providers                      │
├──────────────────────────────────────────────────┤
│  Info Card: Hướng dẫn thêm provider              │
└──────────────────────────────────────────────────┘
```

---

## Providers Table

| Column | Mô tả |
|--------|-------|
| Provider | Icon + `display_name` (bold) + UUID sub-text |
| Adapter | Badge với màu riêng theo `adapter_type` |
| Priority | Số nguyên (cao hơn = ưu tiên hơn khi có nhiều provider) |
| Status | Active (green ✓) / Inactive (grey ✗) |
| Created | Date `DD MMM YYYY` |

### Adapter Type → Badge Mapping

| adapter_type | Label | Màu |
|-------------|-------|-----|
| `proxy_cheap` | Proxy-Cheap | `#7367F0` (purple) |
| `sandbox` | Sandbox | `#28C76F` (green) |
| `luminati` | Luminati | `#00CFE8` (teal) |
| `brightdata` | BrightData | `#FF9F43` (orange) |
| (khác) | adapter_type raw | `#6c757d` (grey) |

---

## Empty State

Khi chưa có provider nào:
- Icon Cloud mờ
- Text "No active providers registered"
- Gợi ý: "Insert a row into `proxy.providers` and restart the service"

---

## Info Card

Card ở cuối trang giải thích flow thêm provider:

1. INSERT row vào `proxy.providers` với `adapter_type` đúng
2. Set env vars tương ứng (vd: `PROXY_CHEAP_API_KEY`)
3. Restart proxy-service

> **Lưu ý thiết kế:** Trang này là **read-only**. Admin không tạo/sửa provider từ UI — phải thao tác qua database trực tiếp và envvar. Điều này đảm bảo security — API credentials không đi qua UI.

---

## API Calls

```
GET /api/v1/admin/proxy/providers   → [{ id, name, display_name, adapter_type, priority, is_active, created_at }]
```

> Response: Không bao giờ trả về `config` field (chứa API key/secret) — bị lọc ở backend.

---

## File

`frontend/src/app/admin/proxy/providers/page.tsx`
