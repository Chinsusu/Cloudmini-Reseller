# Trang: `/reseller/webhooks` — Cấu hình Webhooks

> Role: **reseller**

---

## Mục đích

Reseller cấu hình URL endpoints để nhận HTTP notifications khi có sự kiện xảy ra trong hệ thống (order, payment, VPS...).

---

## Layout

```
┌──────────────────────────────────────────┐
│  Header: "Webhooks"          [+ Add]     │
├──────────────────────────────────────────┤
│  Create Webhook Form (collapsible)       │
├──────────────────────────────────────────┤
│  Webhooks List Table                     │
└──────────────────────────────────────────┘
```

---

## Create Webhook Form

| Field | Loại | Validation |
|-------|------|-----------|
| Endpoint URL | text | HTTPS URL hợp lệ |
| Signing Secret | text (optional) | Dùng để verify HMAC-SHA256 signature |
| Events | Multi-select toggles | Ít nhất 1 event |

### Available Events

| Event | Mô tả |
|-------|-------|
| `order.created` | Order mới được tạo |
| `order.status_changed` | Order đổi trạng thái |
| `order.cancelled` | Order bị hủy |
| `payment.completed` | Thanh toán thành công |
| `payment.failed` | Thanh toán thất bại |
| `vps.created` | VPS mới được tạo |
| `vps.status_changed` | VPS đổi trạng thái |
| `reseller.approved` | Tài khoản reseller được approve |
| `reseller.suspended` | Tài khoản reseller bị suspend |

Submit → `POST /api/v1/reseller/webhooks   { url, secret, events }`

---

## Webhooks Table

| Column | Mô tả |
|--------|-------|
| Endpoint | URL đầy đủ |
| Events | Danh sách event tags (hoặc "All events") |
| Status | Badge: active / inactive |
| Created | Date |
| Actions | Delete |

### Delete Webhook

- `useConfirm` → `DELETE /api/v1/reseller/webhooks/{id}`

---

## Webhook Payload Format

Mỗi request Cloudmini gửi đến endpoint bao gồm:

```
POST {endpoint_url}
Headers:
  Content-Type: application/json
  X-Webhook-Event: order.created
  X-Webhook-Signature: sha256=<hmac>
  X-Webhook-Timestamp: <unix_ts>

Body: {
  "event": "order.created",
  "timestamp": "2026-...",
  "data": { ... }
}
```

---

## API Calls

```
GET    /api/v1/reseller/webhooks
POST   /api/v1/reseller/webhooks   { url, secret, events }
DELETE /api/v1/reseller/webhooks/{id}
```

---

## File

`frontend/src/app/reseller/webhooks/page.tsx`
