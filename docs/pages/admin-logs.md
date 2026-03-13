# Trang: `/admin/logs` — Audit Logs

> Role: **admin** / **super_admin** ONLY

---

## Mục đích

Admin xem toàn bộ audit log của hệ thống: ai làm gì, lúc nào, từ IP nào. Hỗ trợ filter và realtime streaming.

---

## Layout

```
┌──────────────────────────────────────────────────────────────┐
│  Header: "Audit Logs"     [Filter: User|Action|Service]      │
├──────────────────────────────────────────────────────────────┤
│  AuditLog Component (table + realtime stream)                │
└──────────────────────────────────────────────────────────────┘
```

---

## Filter Bar

| Filter | Loại | Ví dụ |
|--------|------|-------|
| User ID | text input (UUID) | Filter theo user cụ thể |
| Action | text input | `login`, `create_order`, `admin_approve_reseller` |
| Service | text input | `user-service`, `proxy-service`, `billing-service` |

Apply → refetch với params.

---

## AuditLog Component

Sử dụng component `<AuditLog>` tái sử dụng được từ `components/ui/AuditLog.tsx`.

### Table Columns

| Column | Mô tả |
|--------|-------|
| Time | `HH:mm:ss DD/MM/YYYY` |
| User | User ID (hoặc email nếu có) |
| Action | Tên action (monospace, màu purple) |
| Service | Badge micro: service name |
| IP | IPv4/IPv6 address |
| Details | Expandable JSON preview |

### Realtime Streaming

- Component subscribe WebSocket / SSE tới log stream
- Log mới tự động append ở đầu bảng với animation
- Toggle button "Live / Paused" để tạm dừng stream

---

## Pagination

- 50 rows/page
- Kết hợp với filters: khi filter thay đổi, reset về page 1

---

## API Calls

```
GET /api/v1/logs?user_id=&action=&service=&page=N&limit=50
```

Realtime:
```
WS  /api/v1/logs/stream   (WebSocket hoặc SSE)
```

---

## File

`frontend/src/app/admin/logs/page.tsx`
