// Package pool provides a multi-node Proxmox pool that implements IProxmoxAdapter
// and routes calls to the appropriate per-node client.
package pool

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pvp/proxmox"
	"github.com/pvp/proxmox/client"
)

// NodeEntry holds a per-node client and its config.
type NodeEntry struct {
	Name   string
	Client *client.Client
}

// Pool is a multi-node Proxmox pool implementing IProxmoxAdapter.
// Calls are routed to the node clients by node name.
type Pool struct {
	mu    sync.RWMutex
	nodes map[string]*client.Client // key: node name
}

// New creates an empty Pool.
func New() *Pool {
	return &Pool{nodes: make(map[string]*client.Client)}
}

// AddNode registers a Proxmox node in the pool.
func (p *Pool) AddNode(nodeName, host string, port int, tokenID, tokenSecret string, verifyCert bool) error {
	c, err := client.New(host, port, tokenID, tokenSecret, verifyCert)
	if err != nil {
		return fmt.Errorf("Pool.AddNode %s: %w", nodeName, err)
	}
	p.mu.Lock()
	p.nodes[nodeName] = c
	p.mu.Unlock()
	return nil
}

func (p *Pool) nodeFor(nodeName string) (*client.Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	c, ok := p.nodes[nodeName]
	if !ok {
		return nil, fmt.Errorf("pool: unknown node %q", nodeName)
	}
	return c, nil
}

// ─── IProxmoxAdapter implementation — delegates to per-node client ────────────

func (p *Pool) ListNodes(ctx context.Context) ([]*proxmox.NodeInfo, error) {
	p.mu.RLock()
	names := make([]string, 0, len(p.nodes))
	for n := range p.nodes {
		names = append(names, n)
	}
	p.mu.RUnlock()

	var infos []*proxmox.NodeInfo
	for _, name := range names {
		c, _ := p.nodeFor(name)
		info, err := c.GetNodeInfo(ctx, name)
		if err != nil {
			continue // skip unreachable nodes
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (p *Pool) GetNodeInfo(ctx context.Context, nodeName string) (*proxmox.NodeInfo, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return nil, err
	}
	return c.GetNodeInfo(ctx, nodeName)
}

func (p *Pool) HealthCheck(ctx context.Context, nodeName string) error {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return err
	}
	return c.HealthCheck(ctx, nodeName)
}

// SelectBestNode returns the node name with the most available RAM, with 20% headroom.
// It implements the "least-loaded" algorithm from the VPS-SERVICE spec.
func (p *Pool) SelectBestNode(ctx context.Context, planRAMMB, planCores, planDiskGB int) (string, error) {
	nodes, err := p.ListNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("SelectBestNode: list nodes: %w", err)
	}

	type candidate struct {
		name      string
		availRAM  int64
	}

	var candidates []candidate
	for _, n := range nodes {
		if n.Status != "online" {
			continue
		}
		availRAM  := n.MaxMem - n.MemBytes
		availDisk := n.MaxDisk - n.DiskBytes

		requiredRAM  := int64(planRAMMB) * 1024 * 1024 * 12 / 10  // 20% headroom
		requiredDisk := int64(planDiskGB) * 1024 * 1024 * 1024 * 12 / 10

		if availRAM >= requiredRAM && availDisk >= requiredDisk {
			candidates = append(candidates, candidate{name: n.Name, availRAM: availRAM})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("SelectBestNode: no node has sufficient resources")
	}

	// Sort by available RAM desc (least loaded = most free resources)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].availRAM > candidates[j].availRAM
	})

	return candidates[0].name, nil
}

// Delegate all per-VM operations to the per-node client.

func (p *Pool) CreateVM(ctx context.Context, req proxmox.CreateVMRequest) (string, error) {
	c, err := p.nodeFor(req.NodeName)
	if err != nil {
		return "", err
	}
	return c.CreateVM(ctx, req)
}

func (p *Pool) StartVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.StartVM(ctx, nodeName, vmid)
}

func (p *Pool) StopVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.StopVM(ctx, nodeName, vmid)
}

func (p *Pool) ShutdownVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.ShutdownVM(ctx, nodeName, vmid)
}

func (p *Pool) RebootVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.RebootVM(ctx, nodeName, vmid)
}

func (p *Pool) SuspendVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.SuspendVM(ctx, nodeName, vmid)
}

func (p *Pool) ResumeVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.ResumeVM(ctx, nodeName, vmid)
}

func (p *Pool) DeleteVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.DeleteVM(ctx, nodeName, vmid)
}

func (p *Pool) GetVMStatus(ctx context.Context, nodeName string, vmid int) (*proxmox.VMStatus, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return nil, err
	}
	return c.GetVMStatus(ctx, nodeName, vmid)
}

func (p *Pool) GetVMIPAddress(ctx context.Context, nodeName string, vmid int) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.GetVMIPAddress(ctx, nodeName, vmid)
}

func (p *Pool) GetConsoleToken(ctx context.Context, nodeName string, vmid int) (*proxmox.ConsoleToken, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return nil, err
	}
	return c.GetConsoleToken(ctx, nodeName, vmid)
}

func (p *Pool) WaitForTask(ctx context.Context, nodeName, taskID string, timeout time.Duration) error {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return err
	}
	return c.WaitForTask(ctx, nodeName, taskID, timeout)
}

func (p *Pool) GetTaskStatus(ctx context.Context, nodeName, taskID string) (*proxmox.TaskStatus, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return nil, err
	}
	return c.GetTaskStatus(ctx, nodeName, taskID)
}

func (p *Pool) CreateSnapshot(ctx context.Context, req proxmox.SnapshotRequest) (string, error) {
	c, err := p.nodeFor(req.NodeName)
	if err != nil {
		return "", err
	}
	return c.CreateSnapshot(ctx, req)
}

func (p *Pool) ListSnapshots(ctx context.Context, nodeName string, vmid int) ([]map[string]any, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return nil, err
	}
	return c.ListSnapshots(ctx, nodeName, vmid)
}

func (p *Pool) DeleteSnapshot(ctx context.Context, nodeName string, vmid int, snapName string) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.DeleteSnapshot(ctx, nodeName, vmid, snapName)
}

func (p *Pool) RollbackSnapshot(ctx context.Context, nodeName string, vmid int, snapName string) (string, error) {
	c, err := p.nodeFor(nodeName)
	if err != nil {
		return "", err
	}
	return c.RollbackSnapshot(ctx, nodeName, vmid, snapName)
}
