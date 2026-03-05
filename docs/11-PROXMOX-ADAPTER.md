# Proxmox Adapter — Infrastructure Design

**Document ID**: PVP-DOC-011  
**Version**: 1.0.0  
**Component**: infrastructure/proxmox  

---

## 1. Overview

Proxmox Adapter là lớp abstraction giữa VPS Service và Proxmox VE REST API. Quản lý connection pool tới 10 nodes, xử lý failover, retry, và chuẩn hóa response format.

---

## 2. Proxmox Adapter Interface

```go
type IProxmoxAdapter interface {
    // Node Management
    GetNodeStatus(ctx context.Context, nodeID string) (*NodeStatus, error)
    ListNodes(ctx context.Context) ([]NodeStatus, error)
    
    // VM Operations
    CreateVM(ctx context.Context, req CreateVMRequest) (*VMTask, error)
    StartVM(ctx context.Context, nodeID string, vmid int) (*VMTask, error)
    StopVM(ctx context.Context, nodeID string, vmid int) (*VMTask, error)
    RebootVM(ctx context.Context, nodeID string, vmid int) (*VMTask, error)
    DeleteVM(ctx context.Context, nodeID string, vmid int) (*VMTask, error)
    SuspendVM(ctx context.Context, nodeID string, vmid int) (*VMTask, error)
    ResumeVM(ctx context.Context, nodeID string, vmid int) (*VMTask, error)
    
    // VM Info
    GetVMStatus(ctx context.Context, nodeID string, vmid int) (*VMStatus, error)
    GetVMConfig(ctx context.Context, nodeID string, vmid int) (*VMConfig, error)
    GetVMConsoleURL(ctx context.Context, nodeID string, vmid int) (string, error)
    
    // Task Tracking
    GetTaskStatus(ctx context.Context, nodeID, taskID string) (*TaskStatus, error)
    WaitForTask(ctx context.Context, nodeID, taskID string, timeout time.Duration) error
    
    // Snapshots
    CreateSnapshot(ctx context.Context, nodeID string, vmid int, name string) (*VMTask, error)
    ListSnapshots(ctx context.Context, nodeID string, vmid int) ([]Snapshot, error)
    DeleteSnapshot(ctx context.Context, nodeID string, vmid int, snapName string) (*VMTask, error)
    
    // VMID Management
    NextVMID(ctx context.Context) (int, error)
}
```

---

## 3. Multi-Node Connection Pool

```go
type ProxmoxCluster struct {
    nodes   map[string]*ProxmoxNode   // nodeID → node client
    mu      sync.RWMutex
    checker *HealthChecker
}

type ProxmoxNode struct {
    ID       string
    Client   *proxmox.Client          // HTTP client with auth
    Status   NodeHealth               // online|offline|maintenance
    LastSeen time.Time
}
```

### Node Health Checker
```
Goroutine: every 30 seconds
    │
    ▼
For each node:
    GET /nodes/{node}/status (timeout 5s)
    │
    ├── success → mark online, update LastSeen
    └── error   → mark offline
                  Log WARNING
                  Alert admin (after 3 consecutive failures)
```

---

## 4. CreateVM Request

```go
type CreateVMRequest struct {
    NodeID      string
    VMID        int
    Name        string          // hostname
    Cores       int
    MemoryMB    int
    DiskGB      int
    OSTemplate  string          // template name in Proxmox
    NetworkBridge string        // e.g., "vmbr0"
    IPConfig    IPConfig        // static IP or DHCP
    SSHKeys     []string        // public keys for cloud-init
    Password    string          // root password (hashed)
    StartAfterCreate bool
}

type IPConfig struct {
    IP      string  // e.g., "192.168.1.100/24"
    Gateway string  // e.g., "192.168.1.1"
    DNS     []string
}
```

---

## 5. Task Polling (WaitForTask)

Proxmox mọi VM operation trả về `taskID`. Phải poll cho đến khi complete.

```go
func (a *ProxmoxAdapter) WaitForTask(ctx context.Context, nodeID, taskID string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    interval := 3 * time.Second
    
    for time.Now().Before(deadline) {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(interval):
            status, err := a.GetTaskStatus(ctx, nodeID, taskID)
            if err != nil {
                continue // retry
            }
            
            switch status.Status {
            case "OK":
                return nil
            case "ERROR":
                return fmt.Errorf("proxmox task failed: %s", status.ExitStatus)
            // "running" → continue polling
            }
            
            // Publish log entry with current step
            a.publishProgress(nodeID, taskID, status.Log)
        }
    }
    return ErrTaskTimeout
}
```

---

## 6. VMID Management

Proxmox yêu cầu VMID duy nhất trong cluster (100-999999999).

```go
// Strategy: use database sequence + Proxmox nextid as double check
func (a *ProxmoxAdapter) NextVMID(ctx context.Context) (int, error) {
    // Try Proxmox cluster nextid first
    vmid, err := a.client.GetNextVMID(ctx)
    if err != nil {
        // Fallback to our own sequence starting from 1000
        return a.db.NextVMIDFromSequence(ctx)
    }
    // Ensure >= 1000 (reserve 100-999 for templates)
    if vmid < 1000 {
        vmid = 1000
    }
    return vmid, nil
}
```

---

## 7. Failover Strategy

```
VM Provision request → select Node A
    │
    ▼
Node A offline (detected by health checker)
    │
    ▼
VPS Service: re-run node selection excluding Node A
    │
    ▼
Select Node B → continue provisioning
    │
    ▼
Log: "Node A unavailable, failover to Node B"
```

Node failures chỉ affect new provisioning. Running VMs không bị ảnh hưởng nếu node đang chạy.

---

## 8. Security

- Proxmox API token (không dùng username/password) — scope: `PVEVMAdmin` per node
- Tokens stored encrypted in environment variables
- All API calls qua HTTPS (Proxmox tự-signed cert hoặc Let's Encrypt)
- Network: platform server ↔ Proxmox nodes qua private VLAN, không expose public
