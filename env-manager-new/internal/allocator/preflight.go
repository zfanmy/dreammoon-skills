package allocator

import (
	"context"
	"fmt"
	"strings"

	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/registry"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/scanner"
)

// PreflightCheck performs a deployment readiness check on a node
type PreflightCheck struct {
	scanner  *scanner.Scanner
	registry *registry.Registry
}

type PreflightResult struct {
	OK              bool                     `json:"ok"`
	Node            string                   `json:"node"`
	Warnings        []string                 `json:"warnings"`
	Snapshot        *scanner.NodeSnapshot    `json:"snapshot"`
	PortSuggestions map[int]int              `json:"port_suggestions"` // requested -> suggested
	Blockers        []string                 `json:"blockers"`
}

func NewPreflight(s *scanner.Scanner, r *registry.Registry) *PreflightCheck {
	return &PreflightCheck{scanner: s, registry: r}
}

func (p *PreflightCheck) Check(ctx context.Context, node string, req scanner.Requirements) (*PreflightResult, error) {
	result := &PreflightResult{
		Node:            node,
		Warnings:        []string{},
		PortSuggestions: map[int]int{},
		Blockers:        []string{},
	}

	// 1. Scan node to get current state
	snap, err := p.scanner.ScanNode(ctx, node, "all")
	if err != nil {
		return nil, fmt.Errorf("scan node: %w", err)
	}
	result.Snapshot = snap

	// 2. Check resources
	if req.MinMemoryMB > 0 {
		if snap.Memory.AvailableMB < req.MinMemoryMB {
			result.Blockers = append(result.Blockers,
				fmt.Sprintf("内存不足: 需要 %d MB, 可用 %d MB", req.MinMemoryMB, snap.Memory.AvailableMB))
		} else if snap.Memory.AvailableMB < req.MinMemoryMB*2 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("内存紧张: 可用 %d MB, 需求 %d MB", snap.Memory.AvailableMB, req.MinMemoryMB))
		}
	}

	if req.MinDiskGB > 0 {
		// Check root disk
		for _, disk := range snap.Disks {
			if disk.MountPoint == "/" || disk.MountPoint == "/data" {
				availGB := disk.AvailableMB / 1024
				if availGB < req.MinDiskGB {
					result.Blockers = append(result.Blockers,
						fmt.Sprintf("磁盘不足 (%s): 需要 %d GB, 可用 %d GB", disk.MountPoint, req.MinDiskGB, availGB))
				}
			}
		}
	}

	if req.NeedGPU {
		if len(snap.GPUs) == 0 {
			result.Blockers = append(result.Blockers, "节点无可用 GPU")
		} else {
			// Check if any GPU has reasonable free memory
			gpuAvailable := false
			for _, gpu := range snap.GPUs {
				if gpu.MemTotalMB-gpu.MemUsedMB > 1024 { // At least 1GB free
					gpuAvailable = true
					break
				}
			}
			if !gpuAvailable {
				result.Blockers = append(result.Blockers, "所有 GPU 显存已耗尽")
			}
		}
	}

	// 3. Check ports
	for _, port := range req.Ports {
		inUse := false
		var occupier string

		// Check bare ports
		for _, bp := range snap.BarePorts {
			if bp.Port == port {
				inUse = true
				occupier = bp.Process
				break
			}
		}

		// Check container ports
		if !inUse {
			for _, c := range snap.Containers {
				for _, mp := range c.Ports {
					if mp.HostPort == port {
						inUse = true
						occupier = fmt.Sprintf("容器 %s", c.Name)
						break
					}
				}
			}
		}

		if inUse {
			// Try to find alternative port
			alt, err := p.registry.FindAvailablePort(node, 0, port+1, port+100)
			if err != nil {
				alt, _ = p.registry.FindAvailablePort(node, 0, 8000, 9000)
			}
			if alt > 0 {
				result.PortSuggestions[port] = alt
			}
			result.Blockers = append(result.Blockers,
				fmt.Sprintf("端口 %d 已被占用 (%s)，建议改用 %d", port, occupier, alt))
		}
	}

	// 4. Check alerts from snapshot
	for _, alert := range snap.Alerts {
		if alert.Level == "warning" {
			result.Warnings = append(result.Warnings, alert.Message)
		} else if alert.Level == "critical" {
			result.Blockers = append(result.Blockers, alert.Message)
		}
	}

	// Final verdict
	result.OK = len(result.Blockers) == 0
	return result, nil
}

// PortAllocator allocates ports on nodes
type PortAllocator struct {
	registry *registry.Registry
}

func NewPortAllocator(r *registry.Registry) *PortAllocator {
	return &PortAllocator{registry: r}
}

func (a *PortAllocator) Allocate(ctx context.Context, node string, preferred, rangeStart, rangeEnd int) (int, error) {
	return a.registry.FindAvailablePort(node, preferred, rangeStart, rangeEnd)
}

// IsPortAvailable checks if a specific port is available on a node
func (a *PortAllocator) IsPortAvailable(ctx context.Context, node string, port int) (bool, error) {
	usage, err := a.registry.FindPort(port, node)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return true, nil
		}
		return false, err
	}
	return !usage.InUse, nil
}
