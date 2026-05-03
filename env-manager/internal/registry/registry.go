package registry

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/scanner"
)

// Registry provides high-level business logic on top of Store
type Registry struct {
	store *Store
}

func NewRegistry(dataDir string) (*Registry, error) {
	store, err := NewStore(dataDir)
	if err != nil {
		return nil, err
	}
	return &Registry{store: store}, nil
}

func (r *Registry) Close() error {
	return r.store.Close()
}

// SyncFromSnapshot updates the registry based on a node scan snapshot
func (r *Registry) SyncFromSnapshot(node string, snap *scanner.NodeSnapshot) error {
	// Register bare metal services
	for _, port := range snap.BarePorts {
		service := Service{
			Name:     port.Process,
			Node:     node,
			Port:     port.Port,
			Proto:    port.Proto,
			Source:   "discovered",
			LastSeen: time.Now(),
			Metadata: fmt.Sprintf(`{"pid":%d,"bind":"%s"}`, port.PID, port.Bind),
		}
		if err := r.store.RegisterService(service); err != nil {
			return err
		}
	}

	// Register container services
	for _, container := range snap.Containers {
		for _, mapping := range container.Ports {
			if mapping.HostPort == 0 {
				continue
			}
			service := Service{
				Name:     fmt.Sprintf("%s/%s", container.Name, container.Image),
				Node:     node,
				Port:     mapping.HostPort,
				Proto:    mapping.Proto,
				Source:   "discovered",
				LastSeen: time.Now(),
				Metadata: fmt.Sprintf(`{"container":"%s","container_port":%d}`, container.ID, mapping.ContainerPort),
			}
			if err := r.store.RegisterService(service); err != nil {
				return err
			}
		}
	}

	return nil
}

// FindAvailablePort searches for an available port in a range
func (r *Registry) FindAvailablePort(node string, preferred, rangeStart, rangeEnd int) (int, error) {
	// Try preferred first
	if preferred > 0 {
		usage, err := r.store.FindPort(preferred, node)
		if err != nil {
			return 0, err
		}
		if !usage.InUse {
			return preferred, nil
		}
	}

	// Search range
	for port := rangeStart; port <= rangeEnd; port++ {
		usage, err := r.store.FindPort(port, node)
		if err != nil {
			return 0, err
		}
		if !usage.InUse {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available port found in range %d-%d on node %s", rangeStart, rangeEnd, node)
}

// ListServices delegates to store
func (r *Registry) ListServices(node string, port int, name string) ([]Service, error) {
	return r.store.ListServices(node, port, name)
}

// FindPort delegates to store
func (r *Registry) FindPort(port int, node string) (*PortUsage, error) {
	return r.store.FindPort(port, node)
}

// ExportToJSON exports all registered services as JSON
func (r *Registry) ExportToJSON() ([]byte, error) {
	services, err := r.store.ListServices("", 0, "")
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(services, "", "  ")
}
