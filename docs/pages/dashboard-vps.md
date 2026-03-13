# Trang: `/dashboard/vps` — Quản lý VPS Instances

> Role: **user**

---

## Mục đích

User xem danh sách VPS instances đang có, thực hiện các hành động (start/stop/reboot/console/terminate), và deploy VPS mới.

---

## Layout

```
┌─────────────────────────────────────────────┐
│  Header: "VPS Instances"     [Deploy VPS]   │
├─────────────────────────────────────────────┤
│  Table: danh sách VPS instances (paginated) │
├─────────────────────────────────────────────┤
│  Pagination                                 │
└─────────────────────────────────────────────┘
```

---

## Table Columns

| Column | Mô tả |
|--------|-------|
| Hostname | Hostname + VMID sub-text |
| Plan | Plan ID |
| IP Address | Public IP (hoặc "—" nếu chưa có) |
| Status | Badge màu theo trạng thái |
| Actions | Start / Stop / Reboot / Console / Terminate |

### Status Colors

| Status | Màu |
|--------|-----|
| running | Green |
| stopped | Grey |
| provisioning | Yellow |
| terminated | Red |
| error | Red |

---

## Deploy VPS Modal

### Bước 1: Chọn Plan

- Grid các plan cards:
  - Plan name (bold)
  - CPU cores · RAM GB · Disk GB
  - Monthly rate + hourly rate breakdown
- Click card → chọn plan (highlighted border)

### Bước 2: Điền thông tin

- **Hostname input:**
  - Lowercase + alphanumeric + dash
  - Regex: `/^[a-z][a-z0-9-]{2,62}[a-z0-9]$/`
  - Validation realtime, hiện error nếu không hợp lệ

### Bước 3: Xác nhận

- Summary: Plan, Hostname, Monthly price
- Submit → disabled nếu validation fail

```
POST /api/v1/vps/orders
Body: { plan_id, hostname, idempotency_key: crypto.randomUUID() }
Response: 202 Accepted
```

Toast: "VPS is being provisioned" (quá trình async, status bắt đầu là `provisioning`)

---

## Actions Per Instance

| Action | Điều kiện | API |
|--------|-----------|-----|
| Start | `status = stopped` | `POST /vps/instances/{id}/start` |
| Stop | `status = running` | `POST /vps/instances/{id}/stop` |
| Reboot | `status = running` | `POST /vps/instances/{id}/reboot` |
| Console | `status ≠ terminated` | `GET /vps/instances/{id}/console` → mở URL mới |
| Terminate | `status ≠ terminated` | `useConfirm` danger → `DELETE /vps/instances/{id}` |

Tất cả actions → Toast success/error sau khi gọi API.

---

## Auto-refresh

`refetchInterval: 10000` (10 giây) — để cập nhật status khi provisioning xong.

---

## API Calls

```
GET  /api/v1/vps/plans
GET  /api/v1/vps/instances?page=N&limit=20
POST /api/v1/vps/orders              { plan_id, hostname, idempotency_key }
POST /api/v1/vps/instances/{id}/start
POST /api/v1/vps/instances/{id}/stop
POST /api/v1/vps/instances/{id}/reboot
GET  /api/v1/vps/instances/{id}/console
DELETE /api/v1/vps/instances/{id}
```

---

## File

`frontend/src/app/dashboard/vps/page.tsx`
