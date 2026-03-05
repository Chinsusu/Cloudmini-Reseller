# Notification Service — Service Design

**Document ID**: PVP-DOC-009  
**Version**: 1.0.0  
**Service**: notification-service  
**Port**: 8086  

---

## 1. Responsibilities

- Consume events từ NATS và trigger notifications
- Email delivery qua SMTP / AWS SES
- Webhook delivery cho resellers (event forwarding)
- In-app notification (stored, fetched on demand)
- Template management với Golang templates
- Retry logic với exponential backoff

---

## 2. Notification Triggers

| Event | Channel | Template |
|---|---|---|
| `user.registered` | Email | `welcome` |
| `user.verified` | Email | `email_verified` |
| `order.proxy.fulfilled` | Email | `proxy_credentials` |
| `order.proxy.failed` | Email | `order_failed` |
| `order.proxy.expiry_reminder` | Email | `proxy_expiring` |
| `vm.ready` | Email | `vps_credentials` |
| `vm.provision.failed` | Email | `vps_failed` |
| `billing.deposit.completed` | Email | `deposit_confirmed` |
| `billing.wallet.low` | Email | `low_balance_alert` |
| `billing.wallet.empty` | Email | `wallet_empty_suspended` |

---

## 3. Webhook (Reseller)

Resellers có thể register webhook endpoints để nhận events liên quan đến users của họ.

```json
// Webhook payload
{
  "event": "order.proxy.fulfilled",
  "timestamp": "2025-01-01T00:00:00Z",
  "data": {
    "order_id": "uuid",
    "user_id": "uuid",
    "status": "active"
  },
  "signature": "HMAC-SHA256 of payload with reseller secret"
}
```

Delivery: max 3 retries, delays: 0s → 30s → 5m. Disable webhook nếu 10 consecutive failures.

---

## 4. API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/notifications` | Danh sách in-app notifications |
| PUT | `/api/v1/notifications/:id/read` | Mark as read |
| PUT | `/api/v1/notifications/read-all` | Mark all as read |
| POST | `/api/v1/reseller/webhooks` | Đăng ký webhook URL |
| GET | `/api/v1/reseller/webhooks` | Danh sách webhooks |
| DELETE | `/api/v1/reseller/webhooks/:id` | Xóa webhook |
