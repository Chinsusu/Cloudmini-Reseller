// Package client provides a concrete implementation of IProxmoxAdapter
// that calls the Proxmox VE REST API using API token authentication.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pvp/proxmox"
)

// Client is a single Proxmox node REST API client.
type Client struct {
	baseURL    string // e.g. https://pve1.example.com:8006
	tokenID    string // e.g. root@pam!mytoken
	tokenSecret string
	httpClient *http.Client
	nodeName   string
}

// New creates a Client for a single Proxmox node.
func New(host string, port int, tokenID, tokenSecret string, verifyCert bool) (*Client, error) {
	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if !verifyCert {
		// In development: skip TLS verification for self-signed certs
		import_crypto_tls(transport)
	}

	return &Client{
		baseURL:     fmt.Sprintf("https://%s:%d/api2/json", host, port),
		tokenID:     tokenID,
		tokenSecret: tokenSecret,
		httpClient:  &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}, nil
}

// ─── Helper: HTTP request with PVE token auth ─────────────────────────────────

func (c *Client) doRequest(ctx context.Context, method, path string, body url.Values) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(body.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("proxmox client: new request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.tokenSecret))
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxmox client: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("proxmox API error %d: %s", resp.StatusCode, string(data))
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("proxmox client: decode: %w", err)
	}
	return envelope.Data, nil
}

// ─── IProxmoxAdapter implementation ──────────────────────────────────────────

func (c *Client) GetNodeInfo(ctx context.Context, nodeName string) (*proxmox.NodeInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/nodes/"+nodeName+"/status", nil)
	if err != nil {
		return nil, fmt.Errorf("GetNodeInfo: %w", err)
	}
	var info proxmox.NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("GetNodeInfo: unmarshal: %w", err)
	}
	info.Name = nodeName
	return &info, nil
}

func (c *Client) ListNodes(ctx context.Context) ([]*proxmox.NodeInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/nodes", nil)
	if err != nil {
		return nil, fmt.Errorf("ListNodes: %w", err)
	}
	var nodes []*proxmox.NodeInfo
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("ListNodes: unmarshal: %w", err)
	}
	return nodes, nil
}

func (c *Client) HealthCheck(ctx context.Context, nodeName string) error {
	_, err := c.GetNodeInfo(ctx, nodeName)
	return err
}

func (c *Client) CreateVM(ctx context.Context, req proxmox.CreateVMRequest) (string, error) {
	params := url.Values{
		"vmid":    {fmt.Sprintf("%d", req.VMID)},
		"name":    {req.Name},
		"cores":   {fmt.Sprintf("%d", req.Cores)},
		"memory":  {fmt.Sprintf("%d", req.MemMB)},
		"ostype":  {req.OSType},
		"scsihw":  {"virtio-scsi-pci"},
		"scsi0":   {fmt.Sprintf("local-lvm:%d,format=raw", req.DiskGB)},
		"net0":    {"virtio,bridge=vmbr0"},
		"agent":   {"1"},
		"onboot":  {"1"},
	}
	if req.CloudInit {
		params.Set("ide2", "local-lvm:cloudinit")
		params.Set("boot", "order=scsi0")
		if req.SSHKeys != "" {
			params.Set("sshkeys", req.SSHKeys)
		}
		if req.Password != "" {
			params.Set("cipassword", req.Password)
		}
		if req.CIDR != "" {
			params.Set("ipconfig0", fmt.Sprintf("ip=%s,gw=%s", req.CIDR, req.Gateway))
		}
	}

	data, err := c.doRequest(ctx, http.MethodPost, "/nodes/"+req.NodeName+"/qemu", params)
	if err != nil {
		return "", fmt.Errorf("CreateVM: %w", err)
	}
	var taskID string
	_ = json.Unmarshal(data, &taskID)
	return taskID, nil
}

func (c *Client) StartVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return c.vmAction(ctx, nodeName, vmid, "start", nil)
}

func (c *Client) StopVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return c.vmAction(ctx, nodeName, vmid, "stop", nil)
}

func (c *Client) ShutdownVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return c.vmAction(ctx, nodeName, vmid, "shutdown", nil)
}

func (c *Client) RebootVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return c.vmAction(ctx, nodeName, vmid, "reboot", nil)
}

func (c *Client) SuspendVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return c.vmAction(ctx, nodeName, vmid, "suspend", nil)
}

func (c *Client) ResumeVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return c.vmAction(ctx, nodeName, vmid, "resume", nil)
}

func (c *Client) DeleteVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	data, err := c.doRequest(ctx, http.MethodDelete,
		fmt.Sprintf("/nodes/%s/qemu/%d", nodeName, vmid), nil)
	if err != nil {
		return "", fmt.Errorf("DeleteVM: %w", err)
	}
	var taskID string
	_ = json.Unmarshal(data, &taskID)
	return taskID, nil
}

func (c *Client) GetVMStatus(ctx context.Context, nodeName string, vmid int) (*proxmox.VMStatus, error) {
	data, err := c.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/nodes/%s/qemu/%d/status/current", nodeName, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("GetVMStatus: %w", err)
	}
	var s proxmox.VMStatus
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("GetVMStatus: unmarshal: %w", err)
	}
	s.VMID = vmid
	return &s, nil
}

func (c *Client) GetVMIPAddress(ctx context.Context, nodeName string, vmid int) (string, error) {
	// Uses QEMU Guest Agent to get IP
	data, err := c.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", nodeName, vmid), nil)
	if err != nil {
		return "", fmt.Errorf("GetVMIPAddress: %w", err)
	}
	var result struct {
		Result []struct {
			Name  string `json:"name"`
			IPAddresses []struct {
				Type    string `json:"ip-address-type"`
				Address string `json:"ip-address"`
			} `json:"ip-addresses"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("GetVMIPAddress: unmarshal: %w", err)
	}
	for _, iface := range result.Result {
		if iface.Name == "lo" {
			continue
		}
		for _, addr := range iface.IPAddresses {
			if addr.Type == "ipv4" && addr.Address != "" {
				return addr.Address, nil
			}
		}
	}
	return "", fmt.Errorf("GetVMIPAddress: no IPv4 found")
}

func (c *Client) GetConsoleToken(ctx context.Context, nodeName string, vmid int) (*proxmox.ConsoleToken, error) {
	data, err := c.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmid),
		url.Values{"websocket": {"1"}})
	if err != nil {
		return nil, fmt.Errorf("GetConsoleToken: %w", err)
	}
	var token proxmox.ConsoleToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("GetConsoleToken: unmarshal: %w", err)
	}
	return &token, nil
}

// WaitForTask polls a Proxmox task until it completes or times out.
func (c *Client) WaitForTask(ctx context.Context, nodeName, taskID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			if t.After(deadline) {
				return fmt.Errorf("WaitForTask: timed out after %s", timeout)
			}
			status, err := c.GetTaskStatus(ctx, nodeName, taskID)
			if err != nil {
				continue // transient error — keep polling
			}
			if !status.IsRunning {
				if status.ExitCode == "OK" || status.ExitCode == "" {
					return nil
				}
				return fmt.Errorf("WaitForTask: task failed with exitcode=%q", status.ExitCode)
			}
		}
	}
}

func (c *Client) GetTaskStatus(ctx context.Context, nodeName, taskID string) (*proxmox.TaskStatus, error) {
	// taskID (UPID) must be URL-encoded
	encoded := url.PathEscape(taskID)
	data, err := c.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/nodes/%s/tasks/%s/status", nodeName, encoded), nil)
	if err != nil {
		return nil, fmt.Errorf("GetTaskStatus: %w", err)
	}
	var t proxmox.TaskStatus
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("GetTaskStatus: unmarshal: %w", err)
	}
	t.IsRunning = t.Status == "running"
	return &t, nil
}

func (c *Client) CreateSnapshot(ctx context.Context, req proxmox.SnapshotRequest) (string, error) {
	params := url.Values{
		"snapname": {req.Name},
		"description": {req.Desc},
	}
	if req.MemState {
		params.Set("vmstate", "1")
	}
	data, err := c.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", req.NodeName, req.VMID), params)
	if err != nil {
		return "", fmt.Errorf("CreateSnapshot: %w", err)
	}
	var taskID string
	_ = json.Unmarshal(data, &taskID)
	return taskID, nil
}

func (c *Client) ListSnapshots(ctx context.Context, nodeName string, vmid int) ([]map[string]any, error) {
	data, err := c.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", nodeName, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("ListSnapshots: %w", err)
	}
	var snaps []map[string]any
	_ = json.Unmarshal(data, &snaps)
	return snaps, nil
}

func (c *Client) DeleteSnapshot(ctx context.Context, nodeName string, vmid int, snapName string) (string, error) {
	data, err := c.doRequest(ctx, http.MethodDelete,
		fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s", nodeName, vmid, snapName), nil)
	if err != nil {
		return "", fmt.Errorf("DeleteSnapshot: %w", err)
	}
	var taskID string
	_ = json.Unmarshal(data, &taskID)
	return taskID, nil
}

func (c *Client) RollbackSnapshot(ctx context.Context, nodeName string, vmid int, snapName string) (string, error) {
	data, err := c.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s/rollback", nodeName, vmid, snapName), nil)
	if err != nil {
		return "", fmt.Errorf("RollbackSnapshot: %w", err)
	}
	var taskID string
	_ = json.Unmarshal(data, &taskID)
	return taskID, nil
}

// vmAction posts a lifecycle command to a QEMU VM.
func (c *Client) vmAction(ctx context.Context, nodeName string, vmid int, action string, params url.Values) (string, error) {
	data, err := c.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/nodes/%s/qemu/%d/status/%s", nodeName, vmid, action), params)
	if err != nil {
		return "", fmt.Errorf("vmAction(%s): %w", action, err)
	}
	var taskID string
	_ = json.Unmarshal(data, &taskID)
	return taskID, nil
}

// import_crypto_tls is a build-time helper placeholder.
// In the real build, replace with:
//   transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
func import_crypto_tls(transport *http.Transport) {
	// Intentionally left for build-tag override.
	// Real implementation: import "crypto/tls" and set InsecureSkipVerify.
	_ = transport
}
