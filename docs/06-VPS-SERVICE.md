# VPS Service — Service Design

**Document ID**: PVP-DOC-006  
**Version**: 1.0.0  
**Service**: vps-service  
**Port**: 8083  

---

## 1. Responsibilities

- Quản lý VPS plans và node inventory
- Nhận order tạo VPS, chạy provisioning async
- Tích hợp với Proxmox Adapter để thực thi VM operations
- VM lifecycle: create → run → suspend → resume → terminate
- Resource reservation để tránh overcommit
- Snapshot management
- Hourly usage data cho Billing Service

---

## 2. API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/vps/plans` | Danh sách VPS plans |
| POST | `/api/v1/vps/orders` | Tạo VPS mới (async) |
| GET | `/api/v1/vps/instances` | VPS của user |
| GET | `/api/v1/vps/instances/:id` | Chi tiết VPS |
| GET | `/api/v1/vps/instances/:id/status` | Trạng thái realtime |
| POST | `/api/v1/vps/instances/:id/start` | Khởi động VPS |
| POST | `/api/v1/vps/instances/:id/stop` | Tắt VPS |
| POST | `/api/v1/vps/instances/:id/reboot` | Khởi động lại |
| POST | `/api/v1/vps/instances/:id/suspend` | Tạm dừng (admin/billing) |
| DELETE | `/api/v1/vps/instances/:id` | Xóa VPS |
| GET | `/api/v1/vps/instances/:id/console` | Console URL (Proxmox noVNC) |
| POST | `/api/v1/vps/instances/:id/snapshots` | Tạo snapshot |
| GET | `/api/v1/vps/instances/:id/snapshots` | Danh sách snapshots |
| DELETE | `/api/v1/vps/instances/:id/snapshots/:snap_id` | Xóa snapshot |

---

## 3. VM Provisioning Flow (Async)

### Phase 1 — Synchronous (immediate response)

```
POST /vps/orders
    │
    ▼
1. Validate plan exists and is active
2. Select optimal node (least-loaded algorithm)
3. Reserve resources on node (atomic update)
4. Create billing hold via billing-service
5. Create instance record (status=pending)
6. Enqueue provision job → NATS JetStream
    │
    ▼
Response: 202 Accepted
{
  "instance_id": "uuid",
  "job_id": "uuid",
  "status": "pending",
  "message": "VM is being provisioned. Check /instances/:id/status"
}
```

### Phase 2 — Asynchronous Worker

```
Job Consumer (NATS worker)
    │
    ▼
Update status → provisioning
    │ publish vm.state.changed (provisioning)
    ▼
Call Proxmox API: POST /nodes/{node}/qemu
    │ payload: vmid, cores, ram, disk, os template
    │ publish log entry per step
    ▼
Poll Proxmox task status (every 3s, timeout 120s)
    │
    ▼
Update status → booting
    │ publish vm.state.changed (booting)
    ▼
Wait for VM IP via cloud-init agent (timeout 90s)
    │
    ▼
Inject SSH public key / set root password
    │
    ▼
Verify SSH reachable (3 retries)
    │
    ▼
Update status → running
Store: ip_address, billing_started_at = NOW()
    │
    ▼
Confirm billing hold → convert to actual charge
    │
    ▼
Publish vm.ready event
    → notification-service: send credentials email
    → log-service: update realtime dashboard
```

### Error Handling (Saga Compensation)

```
Bất kỳ bước nào fail:
    │
    ▼
Release resource reservation on node
    │
    ▼
Release billing hold (no charge)
    │
    ▼
Update instance status → failed
    │
    ▼
Publish vm.provision.failed
    → notification-service: alert user
    → log-service: ERROR log entry
```

---

## 4. Node Selection Algorithm

```go
// SelectNode — Least Available Resources algorithm
func SelectNode(nodes []Node, plan Plan) (*Node, error) {
    var candidates []Node
    
    for _, node := range nodes {
        if node.Status != "online" { continue }
        
        availableRAM  := node.TotalRAM - node.ReservedRAM
        availableCPU  := node.TotalCPU - node.ReservedCPU
        availableDisk := node.TotalDisk - node.ReservedDisk
        
        // 20% headroom buffer
        if availableRAM  < plan.RAM  * 1.2 { continue }
        if availableCPU  < plan.CPU  * 1.2 { continue }
        if availableDisk < plan.Disk * 1.2 { continue }
        
        candidates = append(candidates, node)
    }
    
    if len(candidates) == 0 {
        return nil, ErrNoAvailableNode
    }
    
    // Sort by: available_ram DESC (least loaded)
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].AvailableRAM > candidates[j].AvailableRAM
    })
    
    return &candidates[0], nil
}
```

---

## 5. VM State Machine

```
pending ──────────────────────────────► failed
   │                                      ▲
   ▼                                      │
provisioning ── (Proxmox API error) ──────┤
   │                                      │
   ▼                                      │
booting ─────── (timeout/error) ──────────┤
   │
   ▼
running ◄────── resume ────── suspended
   │                               ▲
   ├──── suspend ──────────────────┘
   │    (billing: wallet empty)
   │
   └──── (user delete / admin) ──► terminated
```

---

## 6. Billing Integration

VPS service emit usage events cho billing-service theo lịch:

```
Cron: every hour
    │
    ▼
Query all instances WHERE status = 'running'
    │
    ▼
For each instance:
    hours = (NOW - last_billed_at) in hours
    amount = hours * plan.hourly_rate
    Publish: vps.usage.report {instance_id, hours, amount}
    Update: last_billed_at = NOW
```

**Auto-suspend khi hết tiền:**
- Billing service detect wallet = 0 sau khi charge
- Publish: `billing.wallet.empty {user_id}`
- VPS service consume → suspend tất cả instances của user
- Notify user qua email

---

## 7. Events Published

| Event | Payload |
|---|---|
| `vm.state.changed` | `{instance_id, from, to, timestamp}` |
| `vm.provision.requested` | `{instance_id, node_id, plan, user_id}` |
| `vm.ready` | `{instance_id, ip, credentials_ref}` |
| `vm.provision.failed` | `{instance_id, reason, step}` |
| `vm.suspended` | `{instance_id, reason}` |
| `vm.terminated` | `{instance_id, user_id}` |
| `vps.usage.report` | `{instance_id, hours, amount}` |
