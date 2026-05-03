package scanner

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/connector"
)

// Scanner orchestrates resource collection across nodes
type Scanner struct {
	execPool map[string]connector.Executor
	timeout  time.Duration
}

type Requirements struct {
	Ports       []int `json:"ports,omitempty"`
	MinMemoryMB int   `json:"min_memory_mb,omitempty"`
	MinDiskGB   int   `json:"min_disk_gb,omitempty"`
	NeedGPU     bool  `json:"need_gpu,omitempty"`
}

type NodeSnapshot struct {
	Node       string          `json:"node"`
	Timestamp  time.Time       `json:"timestamp"`
	OS         string          `json:"os"`
	Arch       string          `json:"arch"`
	CPU        CPUInfo         `json:"cpu"`
	Memory     MemoryInfo      `json:"memory"`
	Disks      []DiskInfo      `json:"disks"`
	GPUs       []GPUInfo       `json:"gpus,omitempty"`
	BarePorts  []PortInfo      `json:"bare_ports"`
	Containers []ContainerInfo `json:"containers,omitempty"`
	Alerts     []Alert         `json:"alerts,omitempty"`
}

type CPUInfo struct {
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usage_percent"`
}

type MemoryInfo struct {
	TotalMB      int     `json:"total_mb"`
	UsedMB       int     `json:"used_mb"`
	AvailableMB  int     `json:"available_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskInfo struct {
	Device       string  `json:"device"`
	TotalMB      int     `json:"total_mb"`
	UsedMB       int     `json:"used_mb"`
	AvailableMB  int     `json:"available_mb"`
	UsagePercent float64 `json:"usage_percent"`
	MountPoint   string  `json:"mount_point"`
}

type GPUInfo struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	MemTotalMB  int    `json:"mem_total_mb"`
	MemUsedMB   int    `json:"mem_used_mb"`
	UtilPercent int    `json:"util_percent"`
	TempC       int    `json:"temp_celsius"`
}

type PortInfo struct {
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
	Process string `json:"process"`
	PID     int    `json:"pid"`
	Bind    string `json:"bind"`
}

type ContainerInfo struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Image  string        `json:"image"`
	Status string        `json:"status"`
	Ports  []PortMapping `json:"ports"`
}

type PortMapping struct {
	HostIP    string `json:"host_ip"`
	HostPort  int    `json:"host_port"`
	Proto     string `json:"proto"`
	ContainerPort int `json:"container_port"`
}

type Alert struct {
	Level   string `json:"level"`   // warning | critical
	Type    string `json:"type"`    // cpu | memory | disk | gpu
	Message string `json:"message"`
	Value   float64 `json:"value"`
}

func New(execPool map[string]connector.Executor, timeoutStr string) *Scanner {
	timeout, _ := time.ParseDuration(timeoutStr)
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Scanner{execPool: execPool, timeout: timeout}
}

func (s *Scanner) ScanNode(ctx context.Context, nodeName, scope string) (*NodeSnapshot, error) {
	exec, ok := s.execPool[nodeName]
	if !ok {
		return nil, fmt.Errorf("node %s not connected", nodeName)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	snap := &NodeSnapshot{
		Node:      nodeName,
		Timestamp: time.Now(),
		OS:        exec.OS(),
		Arch:      exec.Arch(),
	}

	// Always collect OS/arch first

	// Collect based on scope
	scopes := parseScope(scope)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	if scopes["resources"] || scopes["all"] {
		wg.Add(3)
		go func() {
			defer wg.Done()
			cpu, err := s.scanCPU(ctx, exec)
			if err != nil {
				mu.Lock(); firstErr = err; mu.Unlock()
				return
			}
			mu.Lock()
			snap.CPU = cpu
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			mem, err := s.scanMemory(ctx, exec)
			if err != nil {
				mu.Lock(); firstErr = err; mu.Unlock()
				return
			}
			mu.Lock()
			snap.Memory = mem
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			disks, err := s.scanDisk(ctx, exec)
			if err != nil {
				mu.Lock(); firstErr = err; mu.Unlock()
				return
			}
			mu.Lock()
			snap.Disks = disks
			mu.Unlock()
		}()
	}

	if scopes["ports"] || scopes["bare"] || scopes["all"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ports, err := s.scanBarePorts(ctx, exec)
			if err != nil {
				mu.Lock(); firstErr = err; mu.Unlock()
				return
			}
			mu.Lock()
			snap.BarePorts = ports
			mu.Unlock()
		}()
	}

	if scopes["container"] || scopes["all"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			containers, err := s.scanContainers(ctx, exec)
			if err != nil {
				// Docker might not be installed - not a fatal error
				return
			}
			mu.Lock()
			snap.Containers = containers
			mu.Unlock()
		}()
	}

	if scopes["resources"] || scopes["all"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gpus, err := s.scanGPU(ctx, exec)
			if err != nil {
				// GPU might not be available - not fatal
				return
			}
			mu.Lock()
			snap.GPUs = gpus
			mu.Unlock()
		}()
	}

	wg.Wait()

	if firstErr != nil {
		return snap, firstErr
	}

	return snap, nil
}

func (s *Scanner) ScanAll(ctx context.Context) (map[string]*NodeSnapshot, error) {
	results := make(map[string]*NodeSnapshot)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name := range s.execPool {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			snap, err := s.ScanNode(ctx, n, "all")
			if err != nil {
				return
			}
			mu.Lock()
			results[n] = snap
			mu.Unlock()
		}(name)
	}

	wg.Wait()
	return results, nil
}

func parseScope(scope string) map[string]bool {
	scopes := map[string]bool{}
	for _, s := range strings.Split(scope, ",") {
		scopes[strings.TrimSpace(s)] = true
	}
	return scopes
}

// --- Collection implementations ---

func (s *Scanner) scanCPU(ctx context.Context, exec connector.Executor) (CPUInfo, error) {
	var cmd string
	if exec.OS() == "linux" {
		cmd = "nproc"
	} else {
		cmd = "sysctl -n hw.ncpu"
	}
	out, _, err := exec.Run(ctx, cmd)
	if err != nil {
		return CPUInfo{}, err
	}

	cores, _ := strconv.Atoi(strings.TrimSpace(out))
	if cores == 0 {
		cores = 1
	}

	// Get usage
	var usage float64
	if exec.OS() == "linux" {
		out, _, err = exec.Run(ctx, "top -bn1 | grep 'Cpu(s)' | awk '{print $2+$4}' | tr -d '%'")
		if err == nil {
			usage, _ = strconv.ParseFloat(strings.TrimSpace(out), 64)
		}
	}

	return CPUInfo{Cores: cores, UsagePercent: usage}, nil
}

func (s *Scanner) scanMemory(ctx context.Context, exec connector.Executor) (MemoryInfo, error) {
	var info MemoryInfo

	if exec.OS() == "linux" {
		// Use NR==2 to get the second line (memory line) regardless of locale
		out, _, err := exec.Run(ctx, "free -b | awk 'NR==2{print $2,$3,$7}'")
		if err != nil {
			return info, err
		}
		parts := strings.Fields(strings.TrimSpace(out))
		if len(parts) >= 3 {
			total, _ := strconv.ParseInt(parts[0], 10, 64)
			used, _ := strconv.ParseInt(parts[1], 10, 64)
			avail, _ := strconv.ParseInt(parts[2], 10, 64)
			info.TotalMB = int(total / 1024 / 1024)
			info.UsedMB = int(used / 1024 / 1024)
			info.AvailableMB = int(avail / 1024 / 1024)
			if info.TotalMB > 0 {
				info.UsagePercent = float64(info.UsedMB) / float64(info.TotalMB) * 100
			}
		}
	}

	return info, nil
}

func (s *Scanner) scanDisk(ctx context.Context, exec connector.Executor) ([]DiskInfo, error) {
	var disks []DiskInfo

	if exec.OS() == "linux" {
		out, _, err := exec.Run(ctx, "df -B1 --output=source,size,used,avail,target 2>/dev/null | tail -n +2")
		if err != nil {
			return disks, err
		}

		lines := strings.Split(strings.TrimSpace(out), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			total, _ := strconv.ParseInt(fields[1], 10, 64)
			used, _ := strconv.ParseInt(fields[2], 10, 64)
			avail, _ := strconv.ParseInt(fields[3], 10, 64)

			usage := 0.0
			if total > 0 {
				usage = float64(used) / float64(total) * 100
			}

			disks = append(disks, DiskInfo{
				Device:       fields[0],
				TotalMB:      int(total / 1024 / 1024),
				UsedMB:       int(used / 1024 / 1024),
				AvailableMB:  int(avail / 1024 / 1024),
				UsagePercent: usage,
				MountPoint:   fields[4],
			})
		}
	}

	return disks, nil
}

func (s *Scanner) scanGPU(ctx context.Context, exec connector.Executor) ([]GPUInfo, error) {
	out, _, err := exec.Run(ctx, "nvidia-smi --query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu --format=csv,noheader,nounits 2>/dev/null")
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" || line == "no-gpu" {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) < 6 {
			continue
		}

		idx, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		memTotal, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		memUsed, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
		util, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
		temp, _ := strconv.Atoi(strings.TrimSpace(parts[5]))

		gpus = append(gpus, GPUInfo{
			Index:       idx,
			Name:        strings.TrimSpace(parts[1]),
			MemTotalMB:  memTotal,
			MemUsedMB:   memUsed,
			UtilPercent: util,
			TempC:       temp,
		})
	}

	return gpus, nil
}

func (s *Scanner) scanBarePorts(ctx context.Context, exec connector.Executor) ([]PortInfo, error) {
	var ports []PortInfo
	var cmd string

	if exec.OS() == "linux" {
		cmd = "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null"
	} else {
		cmd = "lsof -iTCP -sTCP:LISTEN -nP 2>/dev/null"
	}

	out, _, err := exec.Run(ctx, cmd)
	if err != nil {
		return ports, err
	}

	// Parse ss output format
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Netid") {
			continue
		}

		// Parse: LISTEN 0  128  0.0.0.0:22  0.0.0.0:*  users:(("sshd",pid=1234,fd=3))
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Find the local address field (usually 4th or 5th)
		var localAddr string
		for _, f := range fields {
			if strings.Contains(f, ":") && !strings.HasPrefix(f, "users:") {
				localAddr = f
				break
			}
		}

		if localAddr == "" {
			continue
		}

		// Split address:port
		lastColon := strings.LastIndex(localAddr, ":")
		if lastColon == -1 {
			continue
		}

		bind := localAddr[:lastColon]
		portStr := localAddr[lastColon+1:]
		port, _ := strconv.Atoi(portStr)
		if port == 0 {
			continue
		}

		// Extract process info if available
		process := "unknown"
		pid := 0
		for _, f := range fields {
			if strings.HasPrefix(f, "users:((") || strings.HasPrefix(f, "users:((") {
				// Extract process name from users:(("sshd",pid=1234,fd=3))
				f = strings.TrimPrefix(f, "users:((\"")
				f = strings.TrimPrefix(f, "users:((")
				if idx := strings.Index(f, "\""); idx > 0 {
					process = f[:idx]
				}
				if pidx := strings.Index(f, ",pid="); pidx > 0 {
					pidStr := f[pidx+5:]
					if endIdx := strings.Index(pidStr, ","); endIdx > 0 {
						pidStr = pidStr[:endIdx]
					}
					pid, _ = strconv.Atoi(pidStr)
				}
			}
		}

		ports = append(ports, PortInfo{
			Port:    port,
			Proto:   "tcp",
			Process: process,
			PID:     pid,
			Bind:    bind,
		})
	}

	return ports, nil
}

func (s *Scanner) scanContainers(ctx context.Context, exec connector.Executor) ([]ContainerInfo, error) {
	out, _, err := exec.Run(ctx, "docker ps --format '{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Ports}}\t{{.Status}}' 2>/dev/null")
	if err != nil {
		return nil, err
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" || line == "no-docker" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}

		// Parse port mappings from parts[3]
		var portMaps []PortMapping
		portStr := parts[3]
		if portStr != "" {
			// Format: 0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp
			for _, mapping := range strings.Split(portStr, ", ") {
				mapping = strings.TrimSpace(mapping)
				// Parse: 0.0.0.0:8080->80/tcp
				arrowIdx := strings.Index(mapping, "->")
				if arrowIdx == -1 {
					continue
				}
				hostPart := mapping[:arrowIdx]
				containerPart := mapping[arrowIdx+2:]

				// Host part: 0.0.0.0:8080
				colonIdx := strings.LastIndex(hostPart, ":")
				hostIP := "0.0.0.0"
				hostPort := 0
				if colonIdx != -1 {
					hostIP = hostPart[:colonIdx]
					hostPort, _ = strconv.Atoi(hostPart[colonIdx+1:])
				}

				// Container part: 80/tcp
				containerPort := 0
				proto := "tcp"
				slashIdx := strings.Index(containerPart, "/")
				if slashIdx != -1 {
					containerPort, _ = strconv.Atoi(containerPart[:slashIdx])
					proto = containerPart[slashIdx+1:]
				}

				portMaps = append(portMaps, PortMapping{
					HostIP:        hostIP,
					HostPort:      hostPort,
					Proto:         proto,
					ContainerPort: containerPort,
				})
			}
		}

		status := "running"
		if len(parts) > 4 {
			status = parts[4]
		}

		containers = append(containers, ContainerInfo{
			ID:     parts[0],
			Name:   parts[1],
			Image:  parts[2],
			Status: status,
			Ports:  portMaps,
		})
	}

	return containers, nil
}
