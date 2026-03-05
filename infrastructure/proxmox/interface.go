// Package proxmox provides the IProxmoxAdapter interface and its implementation
// that wraps the Proxmox VE REST API for VM lifecycle operations.
package proxmox

import (
	"context"
	"time"
)

// VMStatus represents the current status of a VM.
type VMStatus struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"` // running|stopped|paused
	CPU       float64 `json:"cpu"`    // 0.0 – 1.0
	MemMB     int64   `json:"mem"`
	DiskGB    float64 `json:"disk"`
	UptimeSec int64   `json:"uptime"`
	IPAddress string  `json:"ip_address,omitempty"`
}

// TaskStatus is the result of polling a Proxmox task.
type TaskStatus struct {
	TaskID    string `json:"upid"`
	Status    string `json:"status"` // running|stopped
	ExitCode  string `json:"exitstatus,omitempty"`
	IsRunning bool
}

// CreateVMRequest contains all parameters needed to create a VM via QEMU API.
type CreateVMRequest struct {
	NodeName  string
	VMID      int
	Name      string
	Cores     int
	MemMB     int
	DiskGB    int
	OSType    string // l26|win10|etc.
	Template  string // local:vztmpl/... or ISO path
	SSHKeys   string // URL-encoded SSH public keys
	Password  string // root password
	CloudInit bool
	CIDR      string // static IP CIDR (optional)
	Gateway   string // gateway IP (optional)
}

// SnapshotRequest parameters for snapshot creation.
type SnapshotRequest struct {
	NodeName string
	VMID     int
	Name     string
	Desc     string
	MemState bool // include RAM state
}

// NodeInfo is a summary of a Proxmox node.
type NodeInfo struct {
	Name      string  `json:"node"`
	Status    string  `json:"status"`
	CPU       float64 `json:"cpu"`   // 0–1
	MemBytes  int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	DiskBytes int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
}

// ConsoleToken contains a noVNC ticket.
type ConsoleToken struct {
	Ticket      string `json:"ticket"`
	Port        string `json:"port"`
	Cert        string `json:"cert"`
	TerminalURL string `json:"terminal_url"`
}

// IProxmoxAdapter defines the contract for all Proxmox node interactions.
type IProxmoxAdapter interface {
	// Node information
	GetNodeInfo(ctx context.Context, nodeName string) (*NodeInfo, error)
	ListNodes(ctx context.Context) ([]*NodeInfo, error)
	HealthCheck(ctx context.Context, nodeName string) error

	// VM lifecycle
	CreateVM(ctx context.Context, req CreateVMRequest) (taskID string, err error)
	StartVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	StopVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	ShutdownVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	RebootVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	SuspendVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	ResumeVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)
	DeleteVM(ctx context.Context, nodeName string, vmid int) (taskID string, err error)

	// Status & monitoring
	GetVMStatus(ctx context.Context, nodeName string, vmid int) (*VMStatus, error)
	GetVMIPAddress(ctx context.Context, nodeName string, vmid int) (string, error)
	GetConsoleToken(ctx context.Context, nodeName string, vmid int) (*ConsoleToken, error)

	// Task polling
	WaitForTask(ctx context.Context, nodeName, taskID string, timeout time.Duration) error
	GetTaskStatus(ctx context.Context, nodeName, taskID string) (*TaskStatus, error)

	// Snapshots
	CreateSnapshot(ctx context.Context, req SnapshotRequest) (taskID string, err error)
	ListSnapshots(ctx context.Context, nodeName string, vmid int) ([]map[string]any, error)
	DeleteSnapshot(ctx context.Context, nodeName string, vmid int, snapName string) (taskID string, err error)
	RollbackSnapshot(ctx context.Context, nodeName string, vmid int, snapName string) (taskID string, err error)
}
